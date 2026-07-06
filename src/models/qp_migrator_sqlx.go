package models

// Internal sqlx migrator adapted from github.com/joncalhoun/migrate (MIT).
// Internalized so the bookkeeping table can be renamed from `migrations` to
// `quepasa_migrations`, allowing QuePasa tables to share a single database
// with the Whatsmeow store tables (`whatsmeow_*`) without name clashes.

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// SqlxMigration is a unique ID plus functions that use a sqlx transaction
// to perform a database migration (and optionally roll it back).
type SqlxMigration struct {
	ID       string
	Migrate  func(tx *sqlx.Tx) error
	Rollback func(tx *sqlx.Tx) error
}

// SqlxQueryMigration creates a SqlxMigration from plain SQL query strings.
func SqlxQueryMigration(id, upQuery, downQuery string) SqlxMigration {
	queryFn := func(query string) func(tx *sqlx.Tx) error {
		if query == "" {
			return nil
		}
		return func(tx *sqlx.Tx) error {
			_, err := tx.Exec(query)
			return err
		}
	}

	return SqlxMigration{
		ID:       id,
		Migrate:  queryFn(upQuery),
		Rollback: queryFn(downQuery),
	}
}

// SqlxMigrator runs a list of migrations, skipping the ones already recorded
// in the bookkeeping table.
type SqlxMigrator struct {
	Migrations []SqlxMigration
	// Printf is used to print out additional information during a migration,
	// such as which step the migration is currently on. If nil it defaults to
	// fmt.Printf.
	Printf func(format string, a ...interface{}) (n int, err error)
}

// Migrate runs the pending migrations using the provided db connection.
func (s *SqlxMigrator) Migrate(sqlDB *sql.DB, dialect string) error {
	db := sqlx.NewDb(sqlDB, dialect)

	s.printf("Creating/checking %s table...\n", MigrationsTableName)
	err := s.createMigrationTable(db)
	if err != nil {
		return err
	}
	for _, m := range s.Migrations {
		var found string
		err := db.Get(&found, db.Rebind("SELECT id FROM "+MigrationsTableName+" WHERE id=?"), m.ID)
		switch err {
		case sql.ErrNoRows:
			s.printf("Running migration: %v\n", m.ID)
			// we need to run the migration so we continue to code below
		case nil:
			s.printf("Skipping migration: %v\n", m.ID)
			continue
		default:
			return fmt.Errorf("looking up migration by id: %w", err)
		}
		err = s.runMigration(db, m)
		if err != nil {
			return err
		}
	}
	return nil
}

// Rollback runs all applied rollbacks using the provided db connection.
func (s *SqlxMigrator) Rollback(sqlDB *sql.DB, dialect string) error {
	db := sqlx.NewDb(sqlDB, dialect)

	s.printf("Creating/checking %s table...\n", MigrationsTableName)
	err := s.createMigrationTable(db)
	if err != nil {
		return err
	}
	for i := len(s.Migrations) - 1; i >= 0; i-- {
		m := s.Migrations[i]
		if m.Rollback == nil {
			s.printf("Rollback not provided: %v\n", m.ID)
			continue
		}
		var found string
		err := db.Get(&found, db.Rebind("SELECT id FROM "+MigrationsTableName+" WHERE id=?"), m.ID)
		switch err {
		case sql.ErrNoRows:
			s.printf("Skipping rollback: %v\n", m.ID)
			continue
		case nil:
			s.printf("Running rollback: %v\n", m.ID)
			// we need to run the rollback so we continue to code below
		default:
			return fmt.Errorf("looking up rollback by id: %w", err)
		}
		err = s.runRollback(db, m)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SqlxMigrator) printf(format string, a ...interface{}) (n int, err error) {
	printf := s.Printf
	if printf == nil {
		printf = fmt.Printf
	}
	return printf(format, a...)
}

func (s *SqlxMigrator) createMigrationTable(db *sqlx.DB) error {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS " + MigrationsTableName + " (id TEXT PRIMARY KEY)")
	if err != nil {
		return fmt.Errorf("creating %s table: %w", MigrationsTableName, err)
	}
	return nil
}

func (s *SqlxMigrator) runMigration(db *sqlx.DB, m SqlxMigration) error {
	errorf := func(err error) error { return fmt.Errorf("running migration: %w", err) }

	tx, err := db.Beginx()
	if err != nil {
		return errorf(err)
	}
	_, err = db.Exec(db.Rebind("INSERT INTO "+MigrationsTableName+" (id) VALUES (?)"), m.ID)
	if err != nil {
		tx.Rollback()
		return errorf(err)
	}
	err = m.Migrate(tx)
	if err != nil {
		tx.Rollback()
		return errorf(err)
	}
	err = tx.Commit()
	if err != nil {
		return errorf(err)
	}
	return nil
}

func (s *SqlxMigrator) runRollback(db *sqlx.DB, m SqlxMigration) error {
	errorf := func(err error) error { return fmt.Errorf("running rollback: %w", err) }

	tx, err := db.Beginx()
	if err != nil {
		return errorf(err)
	}
	_, err = db.Exec(db.Rebind("DELETE FROM "+MigrationsTableName+" WHERE id=?"), m.ID)
	if err != nil {
		tx.Rollback()
		return errorf(err)
	}
	err = m.Rollback(tx)
	if err != nil {
		tx.Rollback()
		return errorf(err)
	}
	err = tx.Commit()
	if err != nil {
		return errorf(err)
	}
	return nil
}
