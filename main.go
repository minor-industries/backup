package main

import (
	"github.com/jessevdk/go-flags"
	"os"
)

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
