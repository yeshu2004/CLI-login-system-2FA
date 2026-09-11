package main

import (
	"github/yeshu2004/cli-login/db"
	"github/yeshu2004/cli-login/cli"
)

func main() {
	db, err := db.ConnectDB()
	if err != nil {
		panic(err)
	}

	cli.Run(db);
}