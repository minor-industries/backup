//go:build !darwin

package keychain

import (
	"github.com/pkg/errors"
)

var errNotImplemented = errors.New("keychain only available on MacOS")

func NewProfile(profileName string, profile *Profile) error {
	return errNotImplemented
}

func LoadProfile(profileName string) (*Profile, error) {
	return nil, errNotImplemented
}

func DeleteProfile(profileName string) error {
	return errNotImplemented
}

func ListProfiles() ([]string, error) {
	return nil, errNotImplemented
}

var ErrorItemNotFound = errors.New("item not found")
