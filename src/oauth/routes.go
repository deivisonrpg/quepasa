package oauth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts public OAuth endpoints.
func RegisterRoutes(r chi.Router) {
	if !IsEnabled() {
		return
	}

	r.Get("/oauth/login", OAuthLoginHandler)
	r.Get("/oauth/callback", OAuthCallbackHandler)
}

// RegisterAuthenticatedResourceRoutes mounts the authenticated resource proxy.
func RegisterAuthenticatedResourceRoutes(r chi.Router) {
	r.Handle("/auth/oauth/resource/*", http.HandlerFunc(AuthenticatedResourceProxyHandler))
}
