package flagutil

import (
	"flag"
	"strings"
)

type boolFlag interface {
	IsBoolFlag() bool
}

// ParseInterspersed parses args while allowing flags to appear after positional args.
// The standard flag package stops parsing at the first positional token.
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
			positionals = append(positionals, args[i+1:]...)
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
