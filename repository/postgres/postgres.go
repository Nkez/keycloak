package postgres

import "github.com/jmoiron/sqlx"
import _ "github.com/lib/pq"

func NewPostgresDB(cfg string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", cfg)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return db, nil
}
