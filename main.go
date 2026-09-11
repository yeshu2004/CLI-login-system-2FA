package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github/yeshu2004/cli-login/db"
	"github/yeshu2004/cli-login/model"

	"github.com/chzyer/readline"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

const (
	maxFailedAttempts = 5
	lockDuration      = 15 * time.Minute
	sessionDuration   = 30 * time.Minute
)

type CLI struct {
	Rl *readline.Instance
	Db *db.DB

	CurrentUser      *model.User
	SessionExpiresAt time.Time
}

func newCLI(rl *readline.Instance, db *db.DB) *CLI {
	return &CLI{
		Rl: rl,
		Db: db,
	}
}

func main() {
	db, err := db.ConnectDB()
	if err != nil {
		panic(err)
	}

	fmt.Println("Containerized CLI Login System — type 'help' for commands, 'exit' to quit.")
	rl, err := readline.New("> ")
	if err != nil {
		panic(err)
	}
	cli := newCLI(rl, db)
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil { // io.EOF
			break
		}

		cmd := strings.TrimSpace(line)

		switch cmd {
		case model.HELP:
			displayAllCommands()
		case model.REGISTER:
			cli.registerUser()
		case model.LOGIN:
			cli.loginUser()
		case model.WHOAMI:
				cli.whoami()
		case model.ENABLE2FA:
			cli.enable2FA()
		case model.DISABLE2FA:
			cli.disable2FA()
		case model.LOGOUT:
			cli.logout()
		case model.EXIT:
			fmt.Println("Goodbye!")
			return
		case "":
			continue

		default:
			fmt.Println("unknown command, type 'help'")
		}
	}
}

func displayAllCommands() {
	fmt.Println("register - create a new account")
	fmt.Println("login - login to your account")
	fmt.Println("whoami - show current user")
	fmt.Println("enable-2fa - enable two-factor authentication")
	fmt.Println("disable-2fa - disable two-factor authentication")
	fmt.Println("logout - logout from current session")
	fmt.Println("exit - quit")
}

func (cli *CLI) logout() {
	if cli.CurrentUser == nil {
		fmt.Println("you are not logged in.")
		return
	}

	fmt.Println("Logged out:", cli.CurrentUser.Username)

	cli.CurrentUser = nil
	cli.SessionExpiresAt = time.Time{}
}

func (cli *CLI) registerUser() {
	cli.Rl.SetPrompt("Enter the username: ")
	inp, err := cli.Rl.Readline()
	if err != nil {
		return
	}
	defer cli.Rl.SetPrompt("> ")

	userName := strings.TrimSpace(inp)

	b, err := cli.Rl.ReadPassword("Enter the password: ")
	if err != nil {
		fmt.Println("error reading password:", err)
		return
	}
	// now hash the password
	password, err := bcrypt.GenerateFromPassword([]byte(b), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("error in hasing the password: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cli.Db.RegisterUser(ctx, userName, string(password)); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("user registered, you can login now.")
}

func (cli *CLI) enable2FA() {
	if !cli.isAuthenticated() {
		fmt.Println("you are not logged in.")
		return
	}

	if cli.CurrentUser.MFAEnabled == 1 {
		fmt.Println("2FA is already enabled.")
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "CLI Login System",
		AccountName: cli.CurrentUser.Username,
		SecretSize:  20,
	})
	if err != nil {
		fmt.Println("failed to generate 2FA secret:", err)
		return
	}

	fmt.Println()
	fmt.Println("2FA Setup")
	fmt.Println("---------")
	fmt.Println("Secret:", key.Secret())
	fmt.Println("OTP URL:", key.URL())
	fmt.Println()
	fmt.Println("Add this account to Google Authenticator.")
	fmt.Println()

	cli.Rl.SetPrompt("Enter the 6-digit code: ")
	defer cli.Rl.SetPrompt("> ")

	code, err := cli.Rl.Readline()
	if err != nil {
		return
	}

	code = strings.TrimSpace(code)
	valid := totp.Validate(code, key.Secret())
	if !valid {
		fmt.Println("invalid authentication code.")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cli.Db.EnableMFA(ctx, cli.CurrentUser.ID, key.Secret()); err != nil {
		fmt.Println("failed to enable 2FA:", err)
		return
	}

	cli.CurrentUser.MFAEnabled = 1
	cli.CurrentUser.TOTPSecret = key.Secret()

	fmt.Println("2FA enabled successfully.")
}

func (cli *CLI) disable2FA() {
	if !cli.isAuthenticated() {
		fmt.Println("you are not logged in.")
		return
	}

	if cli.CurrentUser.MFAEnabled == 0 {
		fmt.Println("2FA is already disabled.")
		return
	}

	cli.Rl.SetPrompt("Enter the 6-digit authentication code: ")
	defer cli.Rl.SetPrompt("> ")

	code, err := cli.Rl.Readline()
	if err != nil {
		return
	}

	code = strings.TrimSpace(code)

	if !totp.Validate(code, cli.CurrentUser.TOTPSecret) {
		fmt.Println("invalid authentication code.")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = cli.Db.DisableMFA(ctx, cli.CurrentUser.ID)
	if err != nil {
		fmt.Println("failed to disable 2FA:", err)
		return
	}

	cli.CurrentUser.MFAEnabled = 0
	cli.CurrentUser.TOTPSecret = ""

	fmt.Println("2FA disabled successfully.")
}

func (cli *CLI) loginUser() {
	cli.Rl.SetPrompt("Enter the username: ")
	inp, err := cli.Rl.Readline()
	if err != nil {
		return
	}
	defer cli.Rl.SetPrompt("> ")

	userName := strings.TrimSpace(inp)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	user, err := cli.Db.GetUserByUsername(ctx, userName)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("invalid username or password.")
			return // re enters the user name again...
		}

		fmt.Println("error finding user:", err)
		return
	}

	// check if the account still have the lock
	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		fmt.Println("account is locked.")
		fmt.Println("try again after:", user.LockedUntil.Format("02 Jan 2006 15:04:05"))
		return
	}

	// if lock period has expired, clear the lock
	if user.LockedUntil != nil && time.Now().After(*user.LockedUntil) {
		err = cli.Db.ClearLock(ctx, user.ID)
		if err != nil {
			fmt.Println("error clearing account lock:", err)
			return
		}

		user.LockedUntil = nil
		user.FailedAttempts = 0
	}

	b, err := cli.Rl.ReadPassword("Enter the password: ")
	if err != nil {
		fmt.Println("error reading password:", err)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), b); err != nil {
		// passowrd does not matches, incr the faild attempt
		err = cli.Db.IncrementFailedAttempts(ctx, user.ID)
		if err != nil {
			fmt.Println("error updating failed attempts:", err)
			return
		}

		user.FailedAttempts++
		fmt.Println("invalid username or password.")

		// lock after 5 failed attempts
		if user.FailedAttempts >= maxFailedAttempts {
			lockedUntil := time.Now().Add(lockDuration)

			err = cli.Db.LockUser(ctx, user.ID, lockedUntil)
			if err != nil {
				fmt.Println("error locking account:", err)
				return
			}

			fmt.Println("account locked for 15 minutes.")
		}
		return
	}

	// if password correct reset failed attempts
	err = cli.Db.ResetFailedAttempts(ctx, user.ID)
	if err != nil {
		fmt.Println("error resetting failed attempts:", err)
		return
	}
	user.FailedAttempts = 0

	if user.MFAEnabled == 1 {
		cli.Rl.SetPrompt("Enter the 6-digit authentication code: ")
		defer cli.Rl.SetPrompt("> ")

		code, err := cli.Rl.Readline()
		if err != nil {
			return
		}

		code = strings.TrimSpace(code)

		if !totp.Validate(code, user.TOTPSecret) {
			fmt.Println("invalid authentication code.")
			return
		}
	}

	now := time.Now()

	cli.CurrentUser = user
	cli.SessionExpiresAt = now.Add(sessionDuration)

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = cli.Db.UpdateLastLogin(ctx, user.ID, now)
	if err != nil {
		fmt.Printf("login successful, but failed to update login time: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println("Login successful!")
	fmt.Println()
	fmt.Println("User Details")
	fmt.Println("------------")
	fmt.Println("Username:", user.Username)
	fmt.Println("Registered:", user.CreatedAt.Format("02 Jan 2006 15:04:05"))
	fmt.Println("MFA:", formatMFAStatus(user.MFAEnabled))
	fmt.Println("Last login:", formatLastLogin(user.LastLoginAt))
	fmt.Println("Session expires:", cli.SessionExpiresAt.Format("02 Jan 2006 15:04:05"))

	fmt.Println()
}

func (cli *CLI) whoami() {
	if !cli.isAuthenticated(){
		fmt.Println("you are not logged in.")
		return
	}

	if time.Now().After(cli.SessionExpiresAt) {
		cli.CurrentUser = nil
		fmt.Println("session expired. please login again.")
		return
	}

	fmt.Println("Username:", cli.CurrentUser.Username)
	fmt.Println("MFA:", formatMFAStatus(cli.CurrentUser.MFAEnabled))
	fmt.Println("Session expires:", cli.SessionExpiresAt.Format("02 Jan 2006 15:04:05"))
}

func (cli *CLI) isAuthenticated() bool {
	if cli.CurrentUser == nil {
		return false
	}

	if time.Now().After(cli.SessionExpiresAt) {
		cli.CurrentUser = nil
		cli.SessionExpiresAt = time.Time{}

		fmt.Println("session expired. please login again.")
		return false
	}

	return true
}

func formatLastLogin(t *time.Time) string {
	if t == nil {
		return "never"
	}

	return t.Format("02 Jan 2006 15:04:05")
}

func formatMFAStatus(val int) string {
	if val == 0 {
		return "disabled"
	}

	return "enabled"
}
