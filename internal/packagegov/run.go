package packagegov

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/shell"
)

func RunCommands(workdir string, commands []CommandSpec, stdout, stderr io.Writer) error {
	for _, command := range commands {
		if len(command.Run) == 0 {
			continue
		}
		if _, err := fmt.Fprintf(stdout, "packagegov: %s\n", command.Name); err != nil {
			return err
		}
		spec := shell.Spec{
			Name:   command.Run[0],
			Args:   append([]string(nil), command.Run[1:]...),
			Dir:    workdir,
			Env:    os.Environ(),
			Stdout: stdout,
			Stderr: stderr,
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
