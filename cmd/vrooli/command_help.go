package main

import (
	"fmt"
	"io"

	"github.com/vrooli/vrooli/internal/cli/topcli"
)

func showMainHelp(w io.Writer) {
	topcli.RenderMainHelp(w, topcli.CommandSpecs())
}

func showCleanupHelp(w io.Writer) {
	topcli.RenderCleanupHelp(w)
}

func printUnknownCommand(w io.Writer, command string, suggestions []string) {
	_, _ = fmt.Fprintf(w, "Unknown command: %s\n", command)
	if len(suggestions) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Did you mean one of these?")
		for _, suggestion := range suggestions {
			_, _ = fmt.Fprintf(w, "  %s\n", suggestion)
		}
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Run 'vrooli --help' for usage information")
}

func printUnknownScenarioCommand(w io.Writer, command string, suggestions []string) {
	_, _ = fmt.Fprintf(w, "Unknown scenario command: %s\n", command)
	if len(suggestions) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Did you mean one of these?")
		for _, suggestion := range suggestions {
			_, _ = fmt.Fprintf(w, "  %s\n", suggestion)
		}
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Run 'vrooli scenario --help' for usage information")
}
