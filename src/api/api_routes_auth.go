package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func registerCanonicalPublicAuthRoutes(r chi.Router) {
	r.Get("/auth/config", LoginConfigController)
	r.Post("/auth/login", CanonicalLoginPostController)
}

func registerCanonicalProtectedAuthRoutes(r chi.Router) {
	r.Get("/auth/session", AuthenticatedSessionController)
	r.Get("/auth/account", AuthenticatedAccountController)
	r.Get("/auth/masterkey/status", AuthenticatedMasterKeyStatusController)
	r.Get("/auth/contexts", AuthenticatedContextAccessListController)
	r.Post("/auth/contexts", AuthenticatedContextAccessUpsertController)
	r.Patch("/auth/contexts", AuthenticatedContextAccessUpsertController)
	r.Delete("/auth/contexts", AuthenticatedContextAccessDeleteController)
	r.Get("/auth/contexts/sessions", AuthenticatedContextSessionsController)
	r.Handle("/auth/oauth/resource/*", http.HandlerFunc(AuthenticatedOAuthResourceProxyController))
}
