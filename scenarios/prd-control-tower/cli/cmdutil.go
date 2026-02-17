package main

import (
	"flag"

	"github.com/vrooli/cli-core/cliutil"
)

// parseArgs parses flags interspersed with positional arguments and returns the positionals.
func parseArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return nil, err
	}
	return fs.Args(), nil
}
