package sipproxy

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/emiago/sipgo/sip"
)

// OutboundWhatsAppInviteRequest describes a SIP-originated call that should be
// bridged to a WhatsApp section.
type OutboundWhatsAppInviteRequest struct {
	CallID        string
	SessionID     string
	Target        string
	From          string
	RemoteRTPAddr string
	RemoteRTPHost string
	RemoteRTPPort int
	Body          string
}

// OutboundWhatsAppInviteResponse carries the SDP answer returned to the SIP
// caller once the WhatsApp callee accepts and media is ready.
type OutboundWhatsAppInviteResponse struct {
	SDP string
}

// OutboundWhatsAppInviteHandler is implemented by the VoIP layer because only
// it can resolve a WhatsApp section and place the native call.
type OutboundWhatsAppInviteHandler func(context.Context, OutboundWhatsAppInviteRequest) (OutboundWhatsAppInviteResponse, error)

func (scm *SIPCallManagerSipgo) SetOutboundWhatsAppInviteHandler(handler OutboundWhatsAppInviteHandler) {
	scm.handlerMutex.Lock()
	defer scm.handlerMutex.Unlock()
	scm.outboundWhatsAppInviteHandler = handler
}

func (scm *SIPCallManagerSipgo) handleIncomingInvite(req *sip.Request, tx sip.ServerTransaction) {
	if scm == nil {
		return
	}

	handler := scm.getOutboundWhatsAppInviteHandler()
	if handler == nil {
		_ = tx.Respond(sip.NewResponseFromRequest(req, sip.StatusServiceUnavailable, "WhatsApp gateway unavailable", nil))
		return
	}

	dialog, err := scm.dialogUA.ReadInvite(req, tx)
	if err != nil {
		scm.logger.Errorf("failed to create UAS dialog for inbound INVITE: %v", err)
		_ = tx.Respond(sip.NewResponseFromRequest(req, sip.StatusBadRequest, "Invalid dialog", nil))
		return
	}

	callID := requestCallID(req)
	if callID == "" {
		_ = dialog.Respond(sip.StatusBadRequest, "Missing Call-ID", nil)
		return
	}

	sessionID := firstHeaderValue(req, "X-QuePasa-SessionId")
	if sessionID == "" {
		sessionID = firstHeaderValue(req, "X-QuePasa-Session")
	}
	if sessionID == "" {
		_ = dialog.Respond(sip.StatusBadRequest, "Missing X-QuePasa-SessionId", nil)
		return
	}

	target := strings.TrimSpace(req.Recipient.User)
	if value := firstHeaderValue(req, "X-QuePasa-Target-Phone"); value != "" {
		target = value
	}
	target = normalizePhoneTarget(target)
	if target == "" {
		_ = dialog.Respond(sip.StatusBadRequest, "Missing target", nil)
		return
	}

	remoteRTPAddr, ok := parseSDPRTPAddr(string(req.Body()))
	if !ok {
		_ = dialog.Respond(sip.StatusBadRequest, "Missing RTP SDP", nil)
		return
	}
	remoteHost, remotePort, err := splitHostPortInt(remoteRTPAddr)
	if err != nil {
		_ = dialog.Respond(sip.StatusBadRequest, "Invalid RTP SDP", nil)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	callInfo := &CallInfo{
		CallID:              callID,
		FromPhone:           callerUser(req),
		ToPhone:             target,
		State:               CallStateProceeding,
		StartTime:           time.Now(),
		LastUpdate:          time.Now(),
		Context:             ctx,
		CancelFunc:          cancel,
		ServerDialogSession: dialog,
	}
	scm.callsMutex.Lock()
	scm.activeCalls[callID] = callInfo
	scm.callsMutex.Unlock()

	scm.logger.Infof("📞➡️ inbound SIP INVITE for WhatsApp: call_id=%s session=%s target=%s rtp=%s", callID, sessionID, target, remoteRTPAddr)
	_ = dialog.Respond(sip.StatusTrying, "Trying", nil)
	_ = dialog.Respond(sip.StatusRinging, "Ringing", nil)

	response, err := handler(ctx, OutboundWhatsAppInviteRequest{
		CallID:        callID,
		SessionID:     sessionID,
		Target:        target,
		From:          callInfo.FromPhone,
		RemoteRTPAddr: remoteRTPAddr,
		RemoteRTPHost: remoteHost,
		RemoteRTPPort: remotePort,
		Body:          string(req.Body()),
	})
	if err != nil {
		scm.logger.Errorf("inbound SIP INVITE failed for WhatsApp call %s: %v", callID, err)
		scm.updateCallState(callID, CallStateRejected)
		scm.cleanupCall(callID)
		_ = dialog.Respond(sip.StatusTemporarilyUnavailable, "WhatsApp unavailable", nil)
		return
	}

	if strings.TrimSpace(response.SDP) == "" {
		scm.updateCallState(callID, CallStateRejected)
		scm.cleanupCall(callID)
		_ = dialog.Respond(sip.StatusInternalServerError, "Missing SDP answer", nil)
		return
	}

	scm.updateCallState(callID, CallStateAccepted)
	if err := dialog.RespondSDP([]byte(response.SDP)); err != nil {
		scm.logger.Errorf("failed to answer inbound SIP INVITE %s: %v", callID, err)
		scm.cleanupCall(callID)
		return
	}
	scm.logger.Infof("✅ inbound SIP INVITE answered for WhatsApp call %s", callID)
}

func (scm *SIPCallManagerSipgo) handleRemoteAck(req *sip.Request, tx sip.ServerTransaction) {
	callID := requestCallID(req)
	if callID == "" {
		return
	}

	callInfo, exists := scm.callInfo(callID)
	if !exists || callInfo.ServerDialogSession == nil {
		return
	}

	if err := callInfo.ServerDialogSession.ReadAck(req, tx); err != nil {
		scm.logger.Warnf("failed to read ACK for inbound SIP dialog %s: %v", callID, err)
	}
}

func (scm *SIPCallManagerSipgo) getOutboundWhatsAppInviteHandler() OutboundWhatsAppInviteHandler {
	scm.handlerMutex.RLock()
	defer scm.handlerMutex.RUnlock()
	return scm.outboundWhatsAppInviteHandler
}

func firstHeaderValue(req *sip.Request, name string) string {
	if req == nil {
		return ""
	}
	header := req.GetHeader(name)
	if header == nil {
		return ""
	}
	return strings.TrimSpace(header.Value())
}

func callerUser(req *sip.Request) string {
	if req == nil || req.From() == nil {
		return ""
	}
	return strings.TrimSpace(req.From().Address.User)
}

func normalizePhoneTarget(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.Contains(value, "@") {
		value = strings.SplitN(value, "@", 2)[0]
	}
	value = strings.TrimPrefix(value, "sip:")
	value = strings.TrimPrefix(value, "tel:")
	return strings.TrimLeft(value, "+")
}

func splitHostPortInt(addr string) (string, int, error) {
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, err
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port <= 0 {
		return "", 0, fmt.Errorf("invalid port %q", portText)
	}
	return host, port, nil
}
