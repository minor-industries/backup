package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/keybase/go-keychain"
	"os"
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

func main() {
	var parser = flags.NewParser(nil, flags.Default)

	must := func(_ *flags.Command, err error) {
		if err != nil {
			panic(err)
		}
	}

	must(parser.AddCommand("new", "Create a new profile", "Creates a new profile", &NewCommand{}))
	must(parser.AddCommand("list", "List profiles", "Lists all profiles", &ListCommand{}))
	must(parser.AddCommand("shell", "Open shell for profile", "Opens a shell with the selected profile", &ShellCommand{}))
	must(parser.AddCommand("edit", "Edit a profile", "Edits the selected profile", &EditCommand{}))

	if _, err := parser.Parse(); err != nil {
		os.Exit(1)
	}
}
