// Package authcli exposes the safe, resource-owned Grok sign-in observation.
package authcli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	StateSignedIn  = "signed_in"
	StateSignedOut = "signed_out"
	StateUnknown   = "unknown"
)

type Status struct {
	Success     bool   `json:"success"`
	SignInState string `json:"sign_in_state"`
	Reason      string `json:"reason,omitempty"`
}

// statusFor reports signed_out only when neither documented credential source
// is configured. Grok has no local validity command, so configured credentials
// are unknown until Grok itself can validate them without a network request.
func statusFor(home string, deploymentKeyConfigured bool, lstat func(string) (os.FileInfo, error)) Status {
	unknown := Status{Success: true, SignInState: StateUnknown, Reason: "Grok credential validity cannot be checked locally"}
	if strings.TrimSpace(home) == "" || lstat == nil {
		return unknown
	}
	if _, err := os.Stat(home); err != nil {
		return unknown
	}
	if deploymentKeyConfigured {
		return unknown
	}
	_, err := lstat(filepath.Join(home, ".grok", "auth.json"))
	switch {
	case err == nil:
		return unknown
	case errors.Is(err, os.ErrNotExist):
		return Status{Success: true, SignInState: StateSignedOut, Reason: "No local Grok credential source is configured"}
	default:
		return unknown
	}
}

func StatusCommand(args []string) error {
	return runStatusCommand(args, os.Stdout, os.UserHomeDir, os.LookupEnv, os.Lstat)
}

func runStatusCommand(args []string, stdout io.Writer, homeDir func() (string, error), lookupEnv func(string) (string, bool), lstat func(string) (os.FileInfo, error)) error {
	flags := flag.NewFlagSet("auth status", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	jsonOutput := flags.Bool("json", false, "Write machine-readable status")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments")
	}
	if !*jsonOutput {
		return fmt.Errorf("auth status requires --json")
	}
	home, err := homeDir()
	status := Status{Success: true, SignInState: StateUnknown, Reason: "Home directory could not be inspected"}
	if err == nil {
		key, configured := lookupEnv("GROK_DEPLOYMENT_KEY")
		status = statusFor(home, configured && strings.TrimSpace(key) != "", lstat)
	}
	return json.NewEncoder(stdout).Encode(status)
}
