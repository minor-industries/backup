package keychain

import (
	"encoding/json"
	"fmt"
	"github.com/keybase/go-keychain"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

const (
	KeychainServiceName = "restic-backup-profile"
)

type Profile struct {
	AwsAccessKeyID     string `json:"AWS_ACCESS_KEY_ID,omitempty"`
	AwsSecretAccessKey string `json:"AWS_SECRET_ACCESS_KEY,omitempty"`
	ResticRepository   string `json:"RESTIC_REPOSITORY,omitempty"`
	ResticPassword     string `json:"RESTIC_PASSWORD,omitempty"`
}

func NewProfile(profileName string, profile *Profile) error {
	out, err := json.Marshal(profile)
	if err != nil {
		return errors.Wrap(err, "marshal")
	}

	fmt.Println("new profile", profile, string(out))

	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(KeychainServiceName)
	item.SetAccount(profileName)
	item.SetData(out)
	err = keychain.AddItem(item)
	return errors.Wrap(err, "add item")
}

func LoadProfile(profileName string) (*Profile, error) {
	item, err := keychain.GetGenericPassword(KeychainServiceName, profileName, "", "")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get profile from keychain")
	}
	if len(item) == 0 {
		return nil, fmt.Errorf("profile %s not found in keychain", profileName)
	}

	var result Profile
	if err := json.Unmarshal(item, &result); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal profile data")
	}

	return &result, nil
}

func DeleteProfile(profileName string) error {
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetService(KeychainServiceName)
	query.SetAccount(profileName)
	query.SetMatchLimit(keychain.MatchLimitOne)
	query.SetReturnData(false) // We just need to delete, no data retrieval needed

	err := keychain.DeleteItem(query)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to delete profile %s", profileName))
	}
	return nil
}

func ListProfiles() ([]string, error) {
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetService(KeychainServiceName)
	query.SetMatchLimit(keychain.MatchLimitAll)
	query.SetReturnAttributes(true)

	results, err := keychain.QueryItem(query)
	if err != nil {
		return nil, err
	}

	return lo.Map(results, func(item keychain.QueryResult, _ int) string {
		return item.Account
	}), nil
}
