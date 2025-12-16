package main

import (
	"flag"
	"strings"
)

// parseArgs allows flag parsing even when the first argument is positional.
// Users can run `cmd <name> --flag value` or `cmd --flag value <name>`.
func parseArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		positional := args[0]
		if err := fs.Parse(args[1:]); err != nil {
			return nil, err
		}
		return append([]string{positional}, fs.Args()...), nil
	}
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	return fs.Args(), nil
}
