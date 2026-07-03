package models

import (
	"database/sql"
	"strings"

	"github.com/jmoiron/sqlx"
)

type QpDataUserContextsSql struct {
	db *sqlx.DB
}

func (source QpDataUserContextsSql) Find(username string, contextID string) (*QpUserContextAccess, error) {
	username = strings.TrimSpace(username)
	contextID = strings.TrimSpace(contextID)
	if username == "" || contextID == "" {
		return nil, sql.ErrNoRows
	}

	result := &QpUserContextAccess{}
	err := source.db.Get(result, `
		SELECT username, contextid, label, enabled, created_at, updated_at
		FROM user_contexts
		WHERE username = ? AND contextid = ?
	`, username, contextID)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (source QpDataUserContextsSql) ListAll() ([]*QpUserContextAccess, error) {
	result := []*QpUserContextAccess{}
	err := source.db.Select(&result, `
		SELECT username, contextid, label, enabled, created_at, updated_at
		FROM user_contexts
		ORDER BY username ASC, contextid ASC
	`)
	return result, err
}

func (source QpDataUserContextsSql) ListForUser(username string, enabledOnly bool) ([]*QpUserContextAccess, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return []*QpUserContextAccess{}, nil
	}

	result := []*QpUserContextAccess{}
	query := `
		SELECT username, contextid, label, enabled, created_at, updated_at
		FROM user_contexts
		WHERE username = ?
	`
	args := []any{username}
	if enabledOnly {
		query += " AND enabled = ?"
		args = append(args, true)
	}
	query += " ORDER BY label ASC, contextid ASC"

	err := source.db.Select(&result, query, args...)
	return result, err
}

func (source QpDataUserContextsSql) Upsert(access *QpUserContextAccess) error {
	if access == nil {
		return nil
	}

	access.Username = strings.TrimSpace(access.Username)
	access.ContextID = strings.TrimSpace(access.ContextID)
	access.Label = strings.TrimSpace(access.Label)
	if access.Username == "" || access.ContextID == "" {
		return sql.ErrNoRows
	}

	_, err := source.db.NamedExec(`
		INSERT INTO user_contexts (username, contextid, label, enabled, updated_at)
		VALUES (:username, :contextid, :label, :enabled, CURRENT_TIMESTAMP)
		ON CONFLICT(username, contextid) DO UPDATE SET
			label = excluded.label,
			enabled = excluded.enabled,
			updated_at = CURRENT_TIMESTAMP
	`, access)
	return err
}

func (source QpDataUserContextsSql) Delete(username string, contextID string) (bool, error) {
	result, err := source.db.Exec(`
		DELETE FROM user_contexts
		WHERE username = ? AND contextid = ?
	`, strings.TrimSpace(username), strings.TrimSpace(contextID))
	if err != nil {
		return false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}
