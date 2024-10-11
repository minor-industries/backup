package main

import (
	"fmt"
	"github.com/keybase/go-keychain"
	"github.com/peterh/liner"
	"github.com/pkg/errors"
)

const (
	account = "restic"
	service = "restic-backup"
)

func query() {
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetService(service)
	query.SetAccount(account)
	//query.SetAccessGroup(accessGroup)
	query.SetMatchLimit(keychain.MatchLimitOne)
	query.SetReturnData(true)
	results, err := keychain.QueryItem(query)
	if err != nil {
		panic(err)
	} else if len(results) != 1 {
		fmt.Println("found", len(results), "results")
	} else {
		password := string(results[0].Data)
		fmt.Println(password)
	}
}

func run() error {
	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)
	res, err := line.Prompt("AWS_SOMETHING_SOMETHING=")
	if err != nil {
		return errors.Wrap(err, "get line")
	}

	fmt.Println(res)
	return nil
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
