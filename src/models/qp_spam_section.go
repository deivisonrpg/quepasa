package models

import "time"

// QpSpamSection stores the prioritized set of WhatsApp sections allowed for /spam.
// Lower priorities run first. Equal priorities form a random dispatch pool.
type QpSpamSection struct {
	Token     string    `db:"token" json:"token"`
	Priority  int       `db:"priority" json:"priority"`
	Enabled   bool      `db:"enabled" json:"enabled"`
	Label     string    `db:"label" json:"label,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"createdAt,omitempty"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt,omitempty"`
}

func (section *QpSpamSection) EffectivePriority() int {
	if section == nil || section.Priority <= 0 {
		return 10
	}
	return section.Priority
}
