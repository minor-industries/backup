package keychain

import (
	"encoding/json"
	"fmt"
	"github.com/keybase/go-keychain"
	"github.com/pkg/errors"
)

const (
	KeychainServiceName = "restic-backup-profile"
)

func NewProfile(profile string, result map[string]string) error {
	out, err := json.Marshal(result)
	if err != nil {
		return errors.Wrap(err, "marshal")
	}

	fmt.Println("new profile", profile, string(out))

	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(KeychainServiceName)
	item.SetAccount(profile)
	item.SetData(out)
	err = keychain.AddItem(item)
	return errors.Wrap(err, "add item")
}

func LoadProfile(profile string) (map[string]string, error) {
	item, err := keychain.GetGenericPassword(KeychainServiceName, profile, "", "")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get profile from keychain")
	}
	if len(item) == 0 {
		return nil, fmt.Errorf("profile %s not found in keychain", profile)
	}

	var result map[string]string
	if err := json.Unmarshal(item, &result); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal profile data")
	}

	return result, nil
}
