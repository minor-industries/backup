package main

import (
	"fmt"
	"github.com/keybase/go-keychain"
	keychain2 "github.com/minor-industries/backup/keychain"
	"github.com/peterh/liner"
	"github.com/pkg/errors"
	"os"
	"syscall"
)

// TODO: only use minor-industries/backup/keychain, not keybase/go-keychain

type ProfileOptions struct {
	Profile string `short:"p" long:"profile" description:"Profile to use" required:"true"`
}

type field struct {
	name     string
	required bool
	secret   bool
}

var fields = []field{
	{"AWS_ACCESS_KEY_ID", false, false},
	{"AWS_SECRET_ACCESS_KEY", false, true},
	{"RESTIC_REPOSITORY", true, false},
	{"RESTIC_PASSWORD", true, true},
}

func readField(f field) (string, error) {
	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)

	var res string
	var err error
	if f.secret {
		res, err = line.PasswordPrompt("(secret) " + f.name + "=")
	} else {
		res, err = line.Prompt(f.name + "=")
	}

	if err != nil {
		return "", errors.Wrap(err, "get line")
	}

	return res, nil
}

type NewCommand struct {
	ProfileOptions
}

func (cmd *NewCommand) Execute(args []string) error {
	fmt.Println("new profile")

	result := map[string]string{}

	for _, field := range fields {
	retry:
		value, err := readField(field)
		if err != nil {
			return errors.Wrap(err, "read field")
		}

		if value == "" {
			if field.required {
				fmt.Printf("%s is required\n", field.name)
				goto retry
			}
		}

		if value != "" {
			result[field.name] = value
		}
	}

	return keychain2.NewProfile(cmd.Profile, &keychain2.Profile{
		AwsAccessKeyID:     result["AWS_ACCESS_KEY_ID"],
		AwsSecretAccessKey: result["AWS_SECRET_ACCESS_KEY"],
		ResticRepository:   result["RESTIC_REPOSITORY"],
		ResticPassword:     result["RESTIC_PASSWORD"],
	})
}

type ListCommand struct{}

func (cmd *ListCommand) Execute(args []string) error {
	fmt.Println("List command called, listing all profiles")

	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetService(keychain2.KeychainServiceName)
	query.SetMatchLimit(keychain.MatchLimitAll)
	query.SetReturnAttributes(true)

	results, err := keychain.QueryItem(query)
	if err != nil {
		return errors.Wrap(err, "failed to query keychain")
	}

	if len(results) == 0 {
		fmt.Println("No profiles found.")
		return nil
	}

	// Print the account names (profile names) found
	for _, result := range results {
		fmt.Println(result.Account)
	}

	return nil
}

type ShellCommand struct {
	ProfileOptions
}

func (cmd *ShellCommand) Execute(args []string) error {
	result, err := keychain2.LoadProfile(cmd.Profile)
	if err != nil {
		return errors.Wrap(err, "load profile")
	}

	env := []string{
		"AWS_ACCESS_KEY_ID", result.AwsAccessKeyID,
		"AWS_SECRET_ACCESS_KEY", result.AwsSecretAccessKey,
		"RESTIC_REPOSITORY", result.ResticRepository,
		"RESTIC_PASSWORD", result.ResticPassword,
	}
	for i := 0; i < len(env); i += 2 {
		k, v := env[i], env[i+1]
		err := os.Setenv(k, v)
		if err != nil {
			return errors.Wrap(err, "set env")
		}
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	// [[ -n "$CUSTOM_PS1" ]] && PS1="$CUSTOM_PS1"
	normalPS1 := "%n@%m %1~ %# "
	customPS1 := fmt.Sprintf("[%s] %s", cmd.Profile, normalPS1)
	if err := os.Setenv("CUSTOM_PS1", customPS1); err != nil {
		return errors.Wrap(err, "set env")
	}

	return syscall.Exec(shell, []string{shell}, os.Environ())
}

type EditCommand struct {
	ProfileOptions
}

func (cmd *EditCommand) Execute(args []string) error {
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
