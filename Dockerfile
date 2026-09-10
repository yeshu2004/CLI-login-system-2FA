FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o cli-login .

FROM alpine:3.22

WORKDIR /app

COPY --from=builder /app/cli-login .
COPY --from=builder /app/db/users.sql ./db/users.sql

RUN mkdir -p /app/data

VOLUME ["/app/data"]

CMD ["./cli-login"]