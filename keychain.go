package main

import (
	"encoding/json"
	"fmt"
	"github.com/keybase/go-keychain"
	"github.com/pkg/errors"
)

const (
	service = "restic-backup-profile"
)

func newProfile(profile string, result map[string]string) error {
	out, err := json.Marshal(result)
	if err != nil {
		return errors.Wrap(err, "marshal")
	}

	fmt.Println("new profile", profile, string(out))

	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(service)
	item.SetAccount(profile)
	item.SetData(out)
	err = keychain.AddItem(item)
	return errors.Wrap(err, "add item")
}

func query(profile string) {
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetService(service)
	query.SetAccount(profile)
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
