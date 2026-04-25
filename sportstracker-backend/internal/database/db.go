package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

// Connect opens a connection to PostgreSQL using env vars.
func Connect() (*sql.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		host := os.Getenv("DB_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("DB_PORT")
		if port == "" {
			port = "5432"
		}
		user := os.Getenv("DB_USER")
		if user == "" {
			user = "postgres"
		}
		password := os.Getenv("DB_PASSWORD")
		dbname := os.Getenv("DB_NAME")
		if dbname == "" {
			dbname = "sportstracker"
		}
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, password, dbname)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("could not ping database: %w", err)
	}

	return db, nil
}

// RunMigrations creates tables if they don't exist yet.
func RunMigrations(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS series (
			id          SERIAL PRIMARY KEY,
			title       VARCHAR(200) NOT NULL,
			sport       VARCHAR(100) NOT NULL,
			platform    VARCHAR(100),
			status      VARCHAR(50) NOT NULL DEFAULT 'pending',
			episodes    INTEGER DEFAULT 0,
			year        INTEGER,
			description TEXT,
			image_url   VARCHAR(500),
			created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS ratings (
			id         SERIAL PRIMARY KEY,
			series_id  INTEGER NOT NULL REFERENCES series(id) ON DELETE CASCADE,
			score      INTEGER NOT NULL CHECK (score >= 1 AND score <= 10),
			comment    TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ratings_series_id ON ratings(series_id)`,
		`CREATE INDEX IF NOT EXISTS idx_series_title ON series(title)`,
		`CREATE INDEX IF NOT EXISTS idx_series_sport ON series(sport)`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("migration failed: %w\nQuery: %s", err, q)
		}
	}
	return nil
}