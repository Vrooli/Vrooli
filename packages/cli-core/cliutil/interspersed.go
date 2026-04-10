package cliutil

import (
	"flag"
	"strings"
)

type boolFlag interface {
	IsBoolFlag() bool
}

// ParseInterspersed parses args allowing flags to appear after positional args.
// Drop-in replacement for fs.Parse(args) that handles interspersed ordering.
//
// Go's flag.FlagSet.Parse stops at the first non-flag argument, which means
// "task status my-id --status pending" silently drops "--status pending".
// This function reorders args so flags come first, then calls fs.Parse.
func ParseInterspersed(fs *flag.FlagSet, args []string) error {
	reordered := reorderForFlagSet(fs, args)
	return fs.Parse(reordered)
}

func reorderForFlagSet(fs *flag.FlagSet, args []string) []string {
	if len(args) == 0 {
		return args
	}

	flags := make([]string, 0, len(args))
	positionals := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--" {
			positionals = append(positionals, args[i:]...)
			break
		}

		if !isFlagToken(arg) {
			positionals = append(positionals, arg)
			continue
		}

		name, hasInlineValue := flagName(arg)
		if hasInlineValue {
			flags = append(flags, arg)
			continue
		}

		flags = append(flags, arg)

		f := fs.Lookup(name)
		if f == nil || !isBoolFlagValue(f.Value) {
			if i+1 < len(args) && args[i+1] != "--" {
				i++
				flags = append(flags, args[i])
			}
		}
	}

	return append(flags, positionals...)
}

func isFlagToken(arg string) bool {
	if arg == "-" {
		return false
	}
	return strings.HasPrefix(arg, "-")
}

func flagName(arg string) (name string, hasInlineValue bool) {
	trimmed := strings.TrimLeft(arg, "-")
	if trimmed == "" {
		return "", false
	}
	if idx := strings.Index(trimmed, "="); idx >= 0 {
		return trimmed[:idx], true
	}
	return trimmed, false
}

func isBoolFlagValue(v flag.Value) bool {
	if bf, ok := v.(boolFlag); ok {
		return bf.IsBoolFlag()
	}
	return false
}
