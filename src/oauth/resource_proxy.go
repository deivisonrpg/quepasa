package oauth

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

var resourceProxyClient = &http.Client{Timeout: 20 * time.Second}

// AuthenticatedResourceProxyHandler forwards authenticated requests to the
// configured OAuth resource server using the provider access token captured
// during the OAuth login flow.
func AuthenticatedResourceProxyHandler(w http.ResponseWriter, r *http.Request) {
	targetURL, err := buildResourceProxyURL(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	sessionToken := SessionTokenFromRequest(r)
	accessToken, ok := AccessTokenForSession(sessionToken)
	if !ok {
		http.Error(w, "oauth access token not available; login again", http.StatusUnauthorized)
		return
	}

	outReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	copyResourceProxyRequestHeaders(outReq.Header, r.Header)
	outReq.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := resourceProxyClient.Do(outReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	copyResourceProxyResponseHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func buildResourceProxyURL(r *http.Request) (string, error) {
	cfg := GetOAuthConfig()
	if cfg == nil || strings.TrimSpace(cfg.ResourceURL) == "" {
		return "", fmt.Errorf("oauth resource base url not configured")
	}

	resourcePath := strings.TrimSpace(chi.URLParam(r, "*"))
	resourcePath = strings.TrimLeft(resourcePath, "/")
	if resourcePath == "" {
		return "", fmt.Errorf("missing resource path")
	}
	if strings.Contains(resourcePath, "://") || strings.HasPrefix(resourcePath, "//") {
		return "", fmt.Errorf("invalid resource path")
	}

	baseURL, err := url.Parse(cfg.ResourceURL)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return "", fmt.Errorf("invalid oauth resource base url")
	}

	target := *baseURL
	target.Path = strings.TrimRight(baseURL.Path, "/") + "/" + resourcePath
	target.RawQuery = r.URL.RawQuery
	return target.String(), nil
}

func SessionTokenFromRequest(r *http.Request) string {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(authHeader) > 7 && strings.EqualFold(authHeader[:7], "Bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}

	if cookie, err := r.Cookie("jwt"); err == nil {
		return strings.TrimSpace(cookie.Value)
	}
	return ""
}

func copyResourceProxyRequestHeaders(dst http.Header, src http.Header) {
	for _, key := range []string{"Accept", "Content-Type"} {
		if value := src.Values(key); len(value) > 0 {
			dst[key] = append([]string(nil), value...)
		}
	}
}

func copyResourceProxyResponseHeaders(dst http.Header, src http.Header) {
	for key, values := range src {
		if isHopByHopHeader(key) {
			continue
		}
		dst[key] = append([]string(nil), values...)
	}
}

func isHopByHopHeader(key string) bool {
	switch strings.ToLower(key) {
	case "connection", "keep-alive", "proxy-authenticate", "proxy-authorization", "te", "trailer", "transfer-encoding", "upgrade":
		return true
	default:
		return false
	}
}
