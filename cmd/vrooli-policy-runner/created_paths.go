package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	agentharness "github.com/vrooli/agentharness"
)

// runCreatedPaths is the one switch and the one report for the rule that lets
// an agent delete files its own session created.
//
//	created-paths status          whether the rule is on, and the last day's allowances
//	created-paths on|off [--by X] turn it on or off; off returns every deletion to the guard's other rules
//	created-paths audit [--since 24h]  list each automatic allowance
func runCreatedPaths(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: vrooli-policy-runner created-paths status|on|off|audit")
	}
	dir, err := agentharness.DefaultDataDir()
	if err != nil {
		return err
	}
	since := 24 * time.Hour
	by := os.Getenv("USER")
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--since":
			if i+1 >= len(args) {
				return errors.New("--since needs a duration such as 24h")
			}
			since, err = time.ParseDuration(args[i+1])
			if err != nil {
				return err
			}
			i++
		case "--by":
			if i+1 >= len(args) {
				return errors.New("--by needs a name")
			}
			by = args[i+1]
			i++
		default:
			return fmt.Errorf("unknown option %q", args[i])
		}
	}
	switch args[0] {
	case "on", "off":
		settings := agentharness.CreationSettings{Enabled: args[0] == "on", Changed: time.Now().UTC(), By: by}
		if err := agentharness.SaveCreationSettings(dir, settings); err != nil {
			return err
		}
		return reportCreatedPaths(stdout, dir, since)
	case "status":
		return reportCreatedPaths(stdout, dir, since)
	case "audit":
		for _, entry := range agentharness.ReadCreationAudit(dir, time.Now().Add(-since)) {
			fmt.Fprintf(stdout, "%s %s (%s, %s)\n", entry.At.Format(time.RFC3339), entry.Path, entry.Source, entry.Certainty)
		}
		return nil
	}
	return fmt.Errorf("unknown created-paths command %q", args[0])
}

func reportCreatedPaths(stdout io.Writer, dir string, since time.Duration) error {
	settings := agentharness.LoadCreationSettings(dir)
	state := "on"
	if !settings.Enabled {
		state = "off"
	}
	fmt.Fprintf(stdout, "Deleting files a session created itself: %s", state)
	if !settings.Changed.IsZero() {
		fmt.Fprintf(stdout, " (changed %s by %s)", settings.Changed.Format(time.RFC3339), strings.TrimSpace(settings.By))
	}
	fmt.Fprintln(stdout)
	entries := agentharness.ReadCreationAudit(dir, time.Now().Add(-since))
	fmt.Fprintf(stdout, "Allowed automatically in the last %s: %d\n", since, len(entries))
	return nil
}
