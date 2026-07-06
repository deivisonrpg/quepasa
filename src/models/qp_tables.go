package models

import "strings"

// canonicalTablePrefix is the prefix written literally in every SQL text of
// this codebase (Go queries, migration files, test schemas). Do NOT change it:
// it is the token ApplyTablePrefix looks for.
const canonicalTablePrefix = "quepasa_"

// TablePrefix is the effective prefix of every QuePasa table at runtime.
// To rename all application tables, change only this constant:
// ApplyTablePrefix rewrites every SQL text and RenameLegacyTables renames the
// existing tables of already deployed databases on the next startup.
const TablePrefix = canonicalTablePrefix

// MigrationsTableName is the bookkeeping table that records applied migration ids.
const MigrationsTableName = TablePrefix + "migrations"

// ApplyTablePrefix rewrites the canonical `quepasa_` table prefix of a SQL
// text to the effective TablePrefix. Every query, migration and test schema
// must pass through it before execution.
func ApplyTablePrefix(sql string) string {
	if TablePrefix == canonicalTablePrefix {
		return sql
	}
	return strings.ReplaceAll(sql, canonicalTablePrefix, TablePrefix)
}
