// Package manifestdispatch contains the small adapter shared by root command
// boundaries that are still backed by legacy application methods.
package manifestdispatch

import (
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

// LegacyArgs converts a parsed manifest RunContext back into the argv shape
// accepted by an existing application method. The manifest remains the
// parser/help source while the domain method keeps its established behavior.
func LegacyArgs(ctx cliapp.RunContext) []string {
	args := append([]string(nil), ctx.Args()...)
	for _, flag := range ctx.Schema().Flags {
		if !ctx.FlagProvided(flag.Name) {
			continue
		}
		if flag.Bool {
			if ctx.BoolFlag(flag.Name) {
				args = append(args, "--"+flag.Name)
			}
			continue
		}
		for _, value := range ctx.FlagValues(flag.Name) {
			args = append(args, "--"+flag.Name, value)
		}
	}
	return args
}

// WithJSON adds the command-local JSON pseudo-flag when root global parsing
// selected JSON before the family command.
func WithJSON(args []string, enabled bool) []string {
	if !enabled || hasFlag(args, "--json") {
		return args
	}
	return append(args, "--json")
}

func hasFlag(args []string, wanted string) bool {
	for _, arg := range args {
		if arg == wanted || strings.HasPrefix(arg, wanted+"=") {
			return true
		}
	}
	return false
}

// WantsHelp reports whether the family should use its established help
// renderer. This keeps existing multi-command family help byte-identical while
// leaf command parsing and dispatch come from cli-core.
func WantsHelp(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}
