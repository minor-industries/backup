package main

import (
	"encoding/json"
	"fmt"
	"github.com/peterh/liner"
	"github.com/pkg/errors"
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

	return newProfile(result)
}

func newProfile(result map[string]string) error {
	out, err := json.Marshal(result)
	if err != nil {
		return errors.Wrap(err, "marshal")
	}

	fmt.Println("new profile", string(out))

	return nil
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
	fmt.Printf("Shell command called with profile: %s\n", cmd.Profile)
	return nil
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
