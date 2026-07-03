package models

import "time"

// QpUserContextAccess links an authenticated user to a tenant context.
// The context remains optional for QuePasa sessions in general, but apps that
// need tenant sharing can require this explicit access table.
type QpUserContextAccess struct {
	Username  string    `db:"username" json:"username"`
	ContextID string    `db:"contextid" json:"contextid"`
	Label     string    `db:"label" json:"label,omitempty"`
	Enabled   bool      `db:"enabled" json:"enabled"`
	CreatedAt time.Time `db:"created_at" json:"createdAt,omitempty"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt,omitempty"`
}
