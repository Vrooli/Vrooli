package main

import (
	"fmt"
	"io"
)

func showMainHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "                          ___")
	_, _ = fmt.Fprintln(w, " _   _ _ __ ___   ___    / (_)")
	_, _ = fmt.Fprintln(w, "| | | | '__/ _ \\ / _ \\  / /| |")
	_, _ = fmt.Fprintln(w, "| |_| | | | (_) | (_) |/ / | |")
	_, _ = fmt.Fprintln(w, " \\___/|_|  \\___/ \\___//_/  |_|")
	_, _ = fmt.Fprintln(w, "                                   ")
	_, _ = fmt.Fprintln(w, "Vrooli CLI - AI Platform Management Tool")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "USAGE:")
	_, _ = fmt.Fprintln(w, "    vrooli <command> [options]")
	_, _ = fmt.Fprintln(w)
	renderCommandGroups(w, groupedTopLevelCommands())
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "OPTIONS:")
	_, _ = fmt.Fprintln(w, "    --help, -h          Show help for a command")
	_, _ = fmt.Fprintln(w, "    --version, -v       Show version information")
	_, _ = fmt.Fprintln(w, "    --json              Forward JSON output mode to compatible commands")
	_, _ = fmt.Fprintln(w, "    --verbose           Forward verbose output mode to compatible commands")
	_, _ = fmt.Fprintln(w, "    --no-color          Disable ANSI color output")
	_, _ = fmt.Fprintln(w, "    --no-stale-check    Skip the Go source freshness check")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "For more help on a specific command:")
	_, _ = fmt.Fprintln(w, "    vrooli <command> --help")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Documentation: docs/")
}

func showCleanupHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "vrooli cleanup - Clean up orphaned processes and stale locks")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  vrooli cleanup orphans    Kill orphaned Vrooli processes")
	_, _ = fmt.Fprintln(w, "  vrooli cleanup locks      Clean stale port lock files")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Options:")
	_, _ = fmt.Fprintln(w, "  --help, -h    Show this help message")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Examples:")
	_, _ = fmt.Fprintln(w, "  vrooli cleanup orphans    # Kill orphaned processes (interactive)")
	_, _ = fmt.Fprintln(w, "  vrooli cleanup locks      # Remove stale lock files")
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
