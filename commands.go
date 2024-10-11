package main

import (
	"fmt"
	"github.com/peterh/liner"
	"github.com/pkg/errors"
)

type ProfileOptions struct {
	Profile string `short:"p" long:"profile" description:"Profile to use" required:"true"`
}

type NewCommand struct {
	ProfileOptions
}

func (cmd *NewCommand) Execute(args []string) error {
	fmt.Printf("New command called with profile: %s\n", cmd.Profile)
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
	fmt.Println("edit", cmd.Profile)

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
