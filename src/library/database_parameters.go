package library

import (
	"fmt"
	"log"
	"os"
)

type DatabaseParameters struct {
	Driver   string `json:"driver,omitempty"`
	Host     string `json:"host,omitempty"`
	DataBase string `json:"database,omitempty"`
	Port     string `json:"port,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	SSL      string `json:"ssl,omitempty"`
}

func (config *DatabaseParameters) GetConnectionString() (connection string) {
	if config.Driver == "mysql" {
		connection = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
			config.User, config.Password, config.Host, config.Port, config.DataBase)
	} else if config.Driver == "postgres" {
		connection = fmt.Sprintf("host=%s dbname=%s port=%s user=%s password=%s sslmode=%s",
			config.Host, config.DataBase, config.Port, config.User, config.Password, config.SSL)
	} else if config.Driver == "sqlite3" {
		connection = GetSQLite(config.DataBase)
	} else {
		log.Fatal("database driver not supported")
	}
	return
}

// GetSQLiteFilename resolves the sqlite file used for a database base name:
// legacy `<name>.db` when it exists, `<name>.sqlite` otherwise.
func GetSQLiteFilename(database string) string {
	if _, err := os.Stat(database + ".db"); err == nil {
		return database + ".db"
	}
	return database + ".sqlite"
}

func GetSQLite(database string) string {
	// WAL + busy timeout: the application pool and the whatsmeow store pool
	// share this single database file, so concurrent writers must not fail
	// immediately with "database is locked"
	const options = "?_foreign_keys=true&_busy_timeout=10000&_journal_mode=WAL"
	return "file:" + GetSQLiteFilename(database) + options
}
