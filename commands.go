package main

import (
	"encoding/json"
	"fmt"
	"github.com/keybase/go-keychain"
	"github.com/peterh/liner"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"os"
	"syscall"
)

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

	return newProfile(cmd.Profile, result)
}

type ListCommand struct{}

func (cmd *ListCommand) Execute(args []string) error {
	fmt.Println("List command called, listing all profiles")
	return nil
}

type ShellCommand struct {
	ProfileOptions
}

func (cmd *ShellCommand) Execute(args []string) error {
	item, err := keychain.GetGenericPassword(service, cmd.Profile, "", "")
	if err != nil {
		return errors.Wrap(err, "failed to get profile from keychain")
	}
	if len(item) == 0 {
		return fmt.Errorf("profile %s not found in keychain", cmd.Profile)
	}

	var result map[string]string
	if err := json.Unmarshal(item, &result); err != nil {
		return errors.Wrap(err, "failed to unmarshal profile data")
	}

	result = lo.PickBy(result, func(key string, _ string) bool {
		return lo.ContainsBy(fields, func(f field) bool { return f.name == key })
	})

	for k, v := range result {
		if v != "" {
			if err := os.Setenv(k, v); err != nil {
				return errors.Wrap(err, "failed to set environment variable")
			}
		}
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	fmt.Printf("Executing shell: %s\n", shell)

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
