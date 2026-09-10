package db

import (
	"context"
	"database/sql"
	"fmt"
	"github/yeshu2004/cli-login/model"
	"log"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	Conn *sql.DB
}

func ConnectDB() (*DB, error) {
	// 1. Open database connection (Creates the file if it doesn't exist)
	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Fatalf("Failed to create data dir: %v", err)
	}

	// Open database file in the persistent directory
	db, err := sql.Open("sqlite", "./data/app.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// 2. Performance Tuning (Crucial for SQLite concurrency)
	// WAL mode allows concurrent reads while writing.
	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;`); err != nil {
		db.Close()
		log.Fatalf("Failed to set PRAGMAs: %v", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Connected to SQL Database!")

	query, err := os.ReadFile("./db/users.sql")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to read user.sql file: %w", err)
	}
	_, err = db.ExecContext(context.Background(), string(query))
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to execute user.sql: %w", err)
	}

	return &DB{Conn: db}, nil

}

func (db *DB) RegisterUser(ctx context.Context, username string, passwordHash string) error {
	query := `INSERT INTO users (user_name, password_hash) VALUES (?, ?)`

	_, err := db.Conn.ExecContext(ctx, query, username, passwordHash)
	if err != nil {
		return fmt.Errorf("failed to register user: %w", err)
	}

	return nil
}

func (d *DB) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	var totpSecret sql.NullString
	var lockedUntil sql.NullTime

	query := `SELECT user_id, user_name, password_hash, mfa_enabled, failed_attempts, created_at, last_login_at, totp_secret, locked_until FROM users WHERE user_name = ?`

	err := d.Conn.QueryRowContext(ctx, query, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.MFAEnabled, &user.FailedAttempts, &user.CreatedAt, &user.LastLoginAt, &totpSecret, &lockedUntil)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}

		return nil, err
	}

	if totpSecret.Valid {
		user.TOTPSecret = totpSecret.String
	}

	if lockedUntil.Valid {
		user.LockedUntil = &lockedUntil.Time
	}

	return &user, nil
}

func (db *DB) UpdateLastLogin(ctx context.Context, userId int64, time time.Time) error {
	query := `UPDATE users SET last_login_at = ? WHERE user_id = ?`

	row, err := db.Conn.ExecContext(ctx, query, time, userId)

	if err != nil {
		return err
	}

	_, err = row.RowsAffected()
	if err != nil {
		return err
	}
	return nil
}

func (d *DB) EnableMFA(ctx context.Context, userID int64, secret string) error {
	row, err := d.Conn.ExecContext(ctx, `UPDATE users SET mfa_enabled = 1, totp_secret = ? WHERE user_id = ?`, secret, userID)
	if err != nil {
		return err
	}

	_, err = row.RowsAffected()
	if err != nil {
		return err
	}

	return nil
}

func (d *DB) DisableMFA(ctx context.Context, userID int64) error {
	row, err := d.Conn.ExecContext(ctx, `UPDATE users SET mfa_enabled = 0, totp_secret = NULL WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}

	_, err = row.RowsAffected()
	if err != nil {
		return err
	}

	return nil
}

func (d *DB) IncrementFailedAttempts(ctx context.Context, userID int64) error {
	row, err := d.Conn.ExecContext(ctx, `UPDATE users SET failed_attempts = failed_attempts + 1 WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}

	_, err = row.RowsAffected()
	if err != nil {
		return err
	}
	return nil
}

func (d *DB) ResetFailedAttempts(ctx context.Context, userID int64) error {
	row, err := d.Conn.ExecContext(ctx, `UPDATE users SET failed_attempts = 0, locked_until = NULL WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}

	_, err = row.RowsAffected()
	if err != nil {
		return err
	}
	return nil
}

func (d *DB) LockUser(ctx context.Context, userID int64, lockedUntil time.Time) error {
	row, err := d.Conn.ExecContext(ctx, `UPDATE users SET locked_until = ? WHERE user_id = ?`, lockedUntil, userID)
	if err != nil {
		return err
	}

	_, err = row.RowsAffected()
	if err != nil {
		return err
	}
	return nil
}

func (d *DB) ClearLock(ctx context.Context, userID int64) error {
	row, err := d.Conn.ExecContext(ctx, `UPDATE users SET locked_until = NULL, failed_attempts = 0 WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}

	_, err = row.RowsAffected()
	if err != nil {
		return err
	}
	return nil
}
