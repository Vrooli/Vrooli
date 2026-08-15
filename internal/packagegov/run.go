package packagegov

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/shell"
)

// CommandOptions controls how a governed package lifecycle command is run.
// The zero value preserves the historical defaults: the current process
// environment and stdin. Lifecycle-owned callers should provide an explicit
// environment and a non-interactive stdin instead.
type CommandOptions struct {
	Context context.Context
	Env     []string
	Stdin   io.Reader
}

func RunCommands(workdir string, commands []CommandSpec, stdout, stderr io.Writer) error {
	return RunCommandsWithOptions(workdir, commands, stdout, stderr, CommandOptions{
		Env:   os.Environ(),
		Stdin: os.Stdin,
	})
}

// RunCommandsWithOptions runs governed package lifecycle commands with the
// supplied process boundary. Keeping this separate from RunCommands lets
// interactive operator commands retain their existing behavior while
// lifecycle-owned setup can be deterministic and prompt-free.
func RunCommandsWithOptions(workdir string, commands []CommandSpec, stdout, stderr io.Writer, options CommandOptions) error {
	env := options.Env
	if env == nil {
		env = os.Environ()
	}
	stdin := options.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}
	for _, command := range commands {
		if len(command.Run) == 0 {
			continue
		}
		if _, err := fmt.Fprintf(stdout, "packagegov: %s\n", command.Name); err != nil {
			return err
		}
		spec := shell.Spec{
			Context: options.Context,
			Name:    command.Run[0],
			Args:    append([]string(nil), command.Run[1:]...),
			Dir:     workdir,
			Env:     env,
			Stdin:   stdin,
			Stdout:  stdout,
			Stderr:  stderr,
		}
		if err := shell.Run(spec); err != nil {
			return err
		}
	}
	return nil
}

func MatchDependents(dependents []Dependent, target string) []Dependent {
	target = strings.TrimSpace(target)
	if target == "" || target == "all" {
		return append([]Dependent(nil), dependents...)
	}
	filtered := make([]Dependent, 0, len(dependents))
	for _, dep := range dependents {
		if dep.ConsumerName == target {
			filtered = append(filtered, dep)
		}
	}
	return filtered
}
