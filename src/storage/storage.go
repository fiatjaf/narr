package storage

import (
	"database/sql"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type Storage struct {
	db *sql.DB
}

func New(path string) (*Storage, *sql.DB, error) {
	if pos := strings.IndexRune(path, '?'); pos == -1 {
		params := "_journal=WAL&_sync=NORMAL&_busy_timeout=5000&cache=shared"
		log.Printf("opening db with params: %s", params)
		path = path + "?" + params
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, nil, err
	}

	if err = migrate(db); err != nil {
		return nil, nil, err
	}

	return &Storage{db: db}, db, nil
}
