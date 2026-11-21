package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	agentharness "github.com/vrooli/agentharness"
)

func runGuardMode(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: vrooli-policy-runner guard-mode list|audit [--snapshot-dir <path>]")
	}
	command := args[0]
	if command != "list" && command != "audit" {
		return fmt.Errorf("guard-mode %s is read-only here; use the operator-only `vrooli guard-mode` command to change a mode", command)
	}
	fs := flag.NewFlagSet("guard-mode "+command, flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("snapshot-dir", "", "policy snapshot and guard mode directory")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected guard-mode arguments: %s", strings.Join(fs.Args(), " "))
	}
	dataDir := strings.TrimSpace(*dir)
	if dataDir == "" {
		var err error
		dataDir, err = agentharness.DefaultDataDir()
		if err != nil {
			return err
		}
	}
	store := agentharness.NewGuardModeStore(dataDir)
	switch command {
	case "list":
		active, err := store.Active(time.Now().UTC())
		if err != nil {
			return err
		}
		return writeJSON(stdout, map[string]any{"active": active})
	case "audit":
		entries, err := store.Audit()
		if err != nil {
			return err
		}
		return writeJSON(stdout, map[string]any{"entries": entries})
	default:
		return errors.New("unreachable guard-mode command")
	}
}
