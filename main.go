package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github/yeshu2004/cli-login/db"
	"github/yeshu2004/cli-login/model"
	"golang.org/x/crypto/bcrypt"
)

const sessionDuration = 30 * time.Minute

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
			if cli.isAuthenticated() {
				cli.whoami()
			}
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
		panic(err)
	}
	fmt.Println("user registered, you can login now.")
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

	b, err := cli.Rl.ReadPassword("Enter the password: ")
	if err != nil {
		fmt.Println("error reading password:", err)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), b); err != nil {
		// passowrd does not matches
		fmt.Println("invalid username or password.")
		return
	}

	now := time.Now()

	cli.CurrentUser = user
	cli.SessionExpiresAt = now.Add(sessionDuration)

	err = cli.Db.UpdateLastLogin(ctx, user.ID, now)
	if err != nil {
		fmt.Println("login successful, but failed to update login time")
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
	if cli.CurrentUser == nil {
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
		fmt.Println("you are not logged in.")
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
