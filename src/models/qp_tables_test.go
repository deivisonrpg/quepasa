package models

import (
	"strings"
	"testing"
)

func TestApplyTablePrefixCanonicalIsIdentity(t *testing.T) {
	if TablePrefix != canonicalTablePrefix {
		t.Skip("TablePrefix customized, identity does not apply")
	}
	query := "SELECT * FROM quepasa_users WHERE username = ?"
	if got := ApplyTablePrefix(query); got != query {
		t.Errorf("expected identity, got: %s", got)
	}
}

func TestApplyTablePrefixRewritesCanonicalToken(t *testing.T) {
	query := "SELECT * FROM quepasa_users JOIN quepasa_servers"
	got := strings.ReplaceAll(query, canonicalTablePrefix, TablePrefix)
	if ApplyTablePrefix(query) != got {
		t.Errorf("ApplyTablePrefix mismatch: %s", ApplyTablePrefix(query))
	}
	if !strings.Contains(ApplyTablePrefix(query), TablePrefix+"users") {
		t.Errorf("expected %susers in: %s", TablePrefix, ApplyTablePrefix(query))
	}
}

// with a remote whatsmeow store (postgres/mysql) the application tables must
// keep living in the local sqlite database: the application migrations are
// sqlite-only
func TestGetDatabaseParametersNonSqliteFallsBackToLocalSqlite(t *testing.T) {
	t.Setenv("DBDRIVER", "postgres")
	t.Setenv("DBDATABASE", "remote_store")

	parameters := getDatabaseParameters()
	if parameters.Driver != "sqlite3" {
		t.Errorf("expected sqlite3 fallback, got driver: %s", parameters.Driver)
	}
	if parameters.DataBase != "quepasa" {
		t.Errorf("expected local quepasa database, got: %s", parameters.DataBase)
	}
}

func TestGetDatabaseParametersSqliteDefaultsToQuepasa(t *testing.T) {
	t.Setenv("DBDRIVER", "sqlite3")
	t.Setenv("DBDATABASE", "")

	parameters := getDatabaseParameters()
	if parameters.Driver != "sqlite3" || parameters.DataBase != "quepasa" {
		t.Errorf("expected sqlite3/quepasa, got: %s/%s", parameters.Driver, parameters.DataBase)
	}
}

func TestGetDatabaseParametersSqliteCustomDatabase(t *testing.T) {
	t.Setenv("DBDRIVER", "sqlite3")
	t.Setenv("DBDATABASE", "whatsmeow")

	parameters := getDatabaseParameters()
	if parameters.DataBase != "whatsmeow" {
		t.Errorf("expected custom database name, got: %s", parameters.DataBase)
	}
}
