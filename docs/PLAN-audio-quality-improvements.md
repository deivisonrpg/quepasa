# PLAN: Audio Quality Improvements (Quepasa ↔ Asterisk)

**Created:** 2026-07-04
**Status:** Proposed
**Priority:** Medium
**Target Module:** `/src/voip/`, `/src/sipproxy/`

## Context

Current wire format on the Asterisk-facing SIP leg is **L16/16000** (raw, uncompressed
16-bit PCM at 16kHz) — see `ISSUE-g711-mulaw-inverted-companding.md`. Asterisk transcodes
L16 to whatever the far-end trunk actually needs. This is lossless and simple, but:

- Heavy bandwidth (256kbps vs ~24-32kbps for a compressed codec) — fine on LAN/localhost,
  a real cost over a WAN link to a remote Asterisk server.
- WhatsApp's own call audio arrives already encoded (MLow by default, or Opus when the
  WhatsApp client/server negotiates `use_mlow_codec_v1=false` — see `src/voip/calls/codec.go`).
  Today it's always decoded to 16kHz PCM before reaching the SIP leg, regardless of source
  codec.

This doc lists options to improve quality/efficiency on that path, roughly ordered by
confidence (verified against code) and effort.

## Option 1 — Negotiate G.711 (PCMU/PCMA) directly on the SIP leg

**Confidence: high (already flagged as remaining work).** The G.711 μ-law/A-law codecs in
`voip_codec.go` were fixed and validated 2026-06-28 (`ISSUE-g711-mulaw-inverted-companding.md`)
but have **no production caller** — nothing currently sends PCMU/PCMA to Asterisk, L16 is
used and Asterisk transcodes.

Work needed:
- SDP offer (`sipproxy_call_manager_extensions.go`) add `a=rtpmap:0 PCMU/8000` /
  `a=rtpmap:8 PCMA/8000` alongside/instead of L16.
- RTP packetization: 8kHz clock (not 16kHz), 160-byte/20ms frames (not 640), payload type
  0/8 (not 118).
- Resampling: internal pipeline is 16kHz mono — G.711 is 8kHz, so downsample before encode,
  upsample after decode. Lossy step, but standard and well understood.
- Removes one hop of transcoding at Asterisk (fewer moving parts) if the upstream trunk
  itself uses G.711 anyway — otherwise no real win over letting Asterisk transcode L16.

Low risk (codecs already tested), moderate effort (SDP + RTP + resampler wiring only,
no new external dependency).

## Option 2 — Opus on the SIP leg

**Confidence: medium (scoped in prior conversation, not yet started).** Real bandwidth win
over a WAN link, matches what the WhatsApp Official Calling API already negotiates directly
with Asterisk (`gateway-whatsapp-official` endpoint already has `opus` in its codec list).

Blockers:
- **No Opus encoder exists yet.** `pion/opus` (already a dependency) is decode-only, used
  today just for playing back Ogg/Opus prompt files (`source.go`). Pure-Go Opus encoders are
  rare/immature; most viable options wrap libopus via cgo — adds a C toolchain dependency,
  complicates cross-compilation and the Docker build.
  - Opus supports 16kHz natively (SILK mode), so no resampling needed if kept at the
    pipeline's native rate — simpler than the G.711 8kHz path.
- SDP offer: add `a=rtpmap:X opus/48000/2` (RTP always signals 48000/2 for Opus per RFC
  6716 even when the actual encoded rate is lower).
- Asterisk pjsip endpoint needs `opus` in its codec list (same change already made for
  the official gateway).
- Only worth it once a real WAN-hop deployment exists — no benefit for local/LAN testing
  (see prior discussion 2026-07-04).

Higher effort than Option 1 because of the encoder gap; evaluate a cgo-free pure Go
implementation first before accepting the cgo dependency.

## Option 3 — Opus/MLow passthrough (skip decode+re-encode)

**Confidence: medium, narrower payoff.** For calls where WhatsApp happens to negotiate Opus
(not the default — MLow is), skip decoding to PCM and re-packetize the already-decrypted
Opus frames straight into a new RTP session toward Asterisk.

Constraints:
- Only applies to the Opus-negotiated subset of calls; MLow calls (the majority/default)
  have no passthrough option — Asterisk has no MLow support at all, decode is mandatory
  for those.
- Not literally free: WhatsApp's media is E2E-encrypted with their own session keys, so
  Quepasa still decrypts every packet; passthrough just skips the codec math (decode to
  PCM + re-encode to Opus), not the packet handling.
- Adds a second maintained code path alongside the universal PCM pipeline — the PCM
  pipeline also feeds other consumers for free (e.g. `TranscriptBackgroundService`/STT
  needs PCM regardless), so passthrough calls would need their own tap point for those
  features if kept.

Do this only if Option 2 ships and CPU/latency on the Opus-call subset is measured to be
worth the added complexity — not a blanket win.

## Other ideas worth investigating later (unverified — not scoped in code yet)

- **Jitter buffer tuning** on the RTP receive path — no investigation done here on current
  buffer depth/adaptivity.
- **Packet loss concealment (PLC)** — check whether any is applied today on the Asterisk-facing
  leg when WhatsApp-side packets drop.
- **Echo cancellation / comfort noise / VAD** — not investigated; only relevant if bridging
  introduces audible echo or dead air.

These need their own code investigation before scoping — listed here only as a reminder,
not evaluated.

## Recommendation

Start with **Option 1** (G.711 direct negotiation) — codecs are already fixed and tested,
work is contained to SDP/RTP/resampler wiring, no new dependency. Revisit **Option 2** (Opus)
only when/if a real WAN-hop deployment makes the bandwidth difference matter, and only after
confirming a cgo-free encoder path (or accepting the cgo dependency deliberately).
