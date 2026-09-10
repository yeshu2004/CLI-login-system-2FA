# Containerized CLI Login System

A containerized interactive CLI authentication system built with Go and SQLite.

The system supports secure password authentication, optional TOTP-based two-factor authentication, account lockout, session management, and persistent database storage.

## Features

* User registration
* Secure password hashing using bcrypt
* Username/password authentication
* Google Authenticator-compatible TOTP 2FA
* Enable/disable 2FA
* Failed login attempt tracking
* Account lockout after multiple failed attempts
* Persistent login metadata
* In-memory session management
* Configurable session timeout
* Interactive CLI using readline
* Command history and terminal interaction
* SQLite database persistence
* Docker and Docker Compose support

## Tech Stack

* Go
* SQLite
* `database/sql`
* bcrypt
* TOTP
* readline
* Docker
* Docker Compose

## Project Structure

```text
.
├── db/
│   └── ...
├── data/
├── main.go
├── Dockerfile
├── docker-compose.yml
├── .dockerignore
├── go.mod
├── go.sum
└── README.md
```

## Requirements

For local development:

* Go 1.25+
* SQLite

For containerized execution:

* Docker
* Docker Compose

## Running Locally

Install dependencies:

```bash
go mod download
```

Run the application:

```bash
go run .
```

The CLI will start with:

```text
Containerized CLI Login System — type 'help' for commands, 'exit' to quit.

>
```

## Running With Docker

Build the Docker image:

```bash
docker compose build
```

Run the CLI with an interactive terminal:

```bash
docker compose run --rm cli-login
```

This starts the CLI in an interactive terminal, allowing you to enter commands

To stop and remove the containers:

```bash
docker compose down
```

### Persistent SQLite Storage

The SQLite database is stored under:

```text
/app/data
```

inside the container.

Docker Compose mounts this directory to a named Docker volume:

```yaml
volumes:
  - sqlite-data:/app/data
```

Therefore, the database persists when the container is stopped or recreated.

To remove the database as well as the container:

```bash
docker compose down -v
```

> Warning: `docker compose down -v` deletes the persistent database volume.

## CLI Commands

```text
help
register
login
whoami
enable-2fa
disable-2fa
logout
exit
```

## Authentication

Passwords are never stored as plaintext.

During registration, passwords are hashed using bcrypt before being stored in SQLite.

During login, the supplied password is compared against the stored bcrypt hash.

## Two-Factor Authentication

The application supports TOTP-based two-factor authentication compatible with applications such as Google Authenticator.

To enable 2FA:

```text
> enable-2fa
```

The application generates a TOTP secret and OTP URL.

The user adds the account to their authenticator application and confirms setup using the generated six-digit code.

Once enabled, login requires:

1. Username
2. Password
3. Six-digit TOTP code

To disable 2FA:

```text
> disable-2fa
```

The current TOTP code is required before 2FA is disabled.

## Account Lockout

Failed login attempts are tracked in SQLite.

After five failed password attempts, the account is temporarily locked for 15 minutes.

A successful password authentication resets the failed attempt counter.

The lock state is persisted in the database.

## Sessions

After successful authentication, the application creates an in-memory session for the authenticated user.

The session contains:

* Current user
* Session expiration time

The default session timeout is 30 minutes.

The session is destroyed when the user logs out or the session expires.

## Database

The application uses SQLite with a `users` table containing authentication and account metadata.

Important fields include:

* `user_id`
* `user_name`
* `password_hash`
* `mfa_enabled`
* `totp_secret`
* `failed_attempts`
* `locked_until`
* `created_at`
* `last_login_at`

## Security Considerations

* Passwords are stored using bcrypt hashes.
* TOTP secrets are stored in the database and should be protected appropriately in a production environment.
* Failed login attempts are tracked persistently.
* Accounts are temporarily locked after repeated failed authentication.
* Unknown usernames return the same generic authentication error as incorrect passwords to avoid username enumeration.
* Active sessions are kept in memory rather than storing session credentials in SQLite.

## Development

Run tests with:

```bash
go test ./...
```

Build locally:

```bash
go build -o cli-login .
```

Run the binary:

```bash
./cli-login
```

## Docker Persistence Test

To verify persistence:

1. Start the application:

```bash
docker compose up --build
```

2. Register a user.

3. Exit the application.

4. Stop the container:

```bash
docker compose down
```

5. Start it again:

```bash
docker compose up
```

6. Login using the previously registered account.

The account should still exist because SQLite is stored in the Docker named volume.

