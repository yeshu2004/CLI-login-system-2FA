package model

import "time"

const HELP = "help"
const REGISTER = "register"
const LOGIN = "login"
const WHOAMI = "whoami"
const LOGOUT = "logout"
const EXIT = "exit"

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	MFAEnabled   int
	// TOTPSecret     string
	FailedAttempts int
	// LockedUntil    *time.Time
	CreatedAt   time.Time
	LastLoginAt *time.Time
}