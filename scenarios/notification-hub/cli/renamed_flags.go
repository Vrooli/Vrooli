package main

import "strings"

// Flags that were renamed because one word meant three things: "deadline" was
// when an ask expires, how long `wait` waits, and how long `ask --and-wait`
// waits. Each now has its own name, and the old names are gone: they fail like
// any unknown flag. The only thing kept is a pointer to the new name in that
// error, so a caller written against the old name learns the fix in one read.
var renamedFlags = map[string]map[string]string{
	"ask":  {"deadline": "expires", "wait-for": "timeout"},
	"wait": {"deadline": "timeout"},
}

// renamedFlagError is the unknown-option error of cli-core with the new flag
// name added to its first line.
type renamedFlagError struct {
	cause error
	hint  string
}

func (e *renamedFlagError) Error() string {
	message := e.cause.Error()
	first, rest, hasRest := strings.Cut(message, "\n")
	first += " (" + e.hint + ")"
	if !hasRest {
		return first
	}
	return first + "\n" + rest
}

func (e *renamedFlagError) Unwrap() error { return e.cause }

// explainRenamedFlag adds the new name to an unknown-option error for a renamed
// flag of `conversations ask` or `conversations wait`. Any other error is
// returned unchanged.
func explainRenamedFlag(args []string, err error) error {
	if err == nil {
		return nil
	}
	renames := renamedFlagsOf(args)
	if renames == nil {
		return err
	}
	for old, renamed := range renames {
		if strings.HasPrefix(err.Error(), "unknown option: --"+old) {
			return &renamedFlagError{cause: err, hint: "--" + old + " was renamed to --" + renamed}
		}
	}
	return err
}

// renamedFlagsOf returns the renamed flags of the `conversations` subcommand
// that args invoke, or nil.
func renamedFlagsOf(args []string) map[string]string {
	for i, arg := range args {
		if arg == "--" {
			return nil
		}
		if arg == "conversations" && i+1 < len(args) {
			return renamedFlags[args[i+1]]
		}
	}
	return nil
}
