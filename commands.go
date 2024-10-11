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

var fields = []string{
	"AWS_ACCESS_KEY_ID",
	"AWS_SECRET_ACCESS_KEY",
	"RESTIC_REPOSITORY",
	"RESTIC_PASSWORD",
}

func readField(name string) (string, error) {
	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)
	res, err := line.Prompt(name + "=")
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
		f, err := readField(field)
		if err != nil {
			return errors.Wrap(err, "read field")
		}
		if f != "" {
			result[field] = f
		}
	}

	out, err := json.Marshal(result)
	if err != nil {
		return errors.Wrap(err, "marshal")
	}

	fmt.Println(string(out))

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
