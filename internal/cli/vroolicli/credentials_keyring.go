package vroolicli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	keyring "github.com/vrooli/vrooli/internal/credentials"
	"github.com/vrooli/vrooli/internal/hostinventory"
)

type credentialKeyringStatus struct {
	State   string   `json:"state"`
	Cause   string   `json:"cause,omitempty"`
	Remedy  []string `json:"remedy,omitempty"`
	Support bool     `json:"supported"`
}

func credentialsKeyring(ctx *CommandContext, args []string, input io.Reader) error {
	if len(args) == 0 || commandtree.WantsHelp(args) {
		fmt.Fprintln(ctx.Stdout, "Usage:\n"+
			"  vrooli credentials keyring status [--format json]\n"+
			"  vrooli credentials keyring inspect [--path <path>] [--format json]\n"+
			"  vrooli credentials keyring repair [--path <path>] [--format json]\n"+
			"  vrooli credentials keyring unlock < passphrase\n\n"+
			"Keyring inspection and repair never print credential values. Unlock reads the passphrase from standard input only.")
		return nil
	}
	switch args[0] {
	case "status":
		return credentialsKeyringStatusCommand(ctx, args[1:])
	case "inspect":
		return credentialsKeyringFile(ctx, args[1:], false)
	case "repair":
		return credentialsKeyringFile(ctx, args[1:], true)
	case "unlock":
		return credentialsKeyringUnlock(ctx, args[1:], input)
	default:
		return fmt.Errorf("unknown credentials keyring command %q", args[0])
	}
}

func keyringFormatAndPath(name string, args []string) (string, string, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path, format := "", "text"
	fs.StringVar(&path, "path", "", "keyring path")
	fs.StringVar(&format, "format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return "", "", err
	}
	if len(fs.Args()) != 0 {
		return "", "", fmt.Errorf("%s accepts no positional arguments", name)
	}
	format = strings.TrimSpace(format)
	if format != "text" && format != "json" {
		return "", "", fmt.Errorf("%s format must be text or json", name)
	}
	return path, format, nil
}

func credentialsKeyringStatusCommand(ctx *CommandContext, args []string) error {
	_, format, err := keyringFormatAndPath("credentials keyring status", args)
	if err != nil {
		return err
	}
	capability := hostinventory.CredentialStoreStatus(context.Background())
	status := credentialKeyringStatus{
		State: capability.State, Cause: capability.Reason, Support: capability.Supported,
		Remedy: credentialKeyringRemedy(capability.State),
	}
	if format == "json" {
		return json.NewEncoder(ctx.Stdout).Encode(status)
	}
	fmt.Fprintf(ctx.Stdout, "Credential keyring: %s\n", status.State)
	if status.Cause != "" {
		fmt.Fprintf(ctx.Stdout, "  Cause:  %s\n", status.Cause)
	}
	for _, remedy := range status.Remedy {
		fmt.Fprintf(ctx.Stdout, "  Remedy: %s\n", remedy)
	}
	return nil
}

func credentialKeyringRemedy(state string) []string {
	switch state {
	case "locked":
		return []string{
			"Pipe the login-keyring passphrase to `vrooli credentials keyring unlock`.",
			"If this is an autologin host, disable autologin and log in interactively, or opt into the high-risk login-keyring unlock safeguard.",
		}
	case "unresponsive":
		return []string{"Keep the session bus available and rerun `vrooli credentials keyring status`; the bounded probe could not determine the store state."}
	case "unavailable":
		return []string{"Start a supported Secret Service session, or run `vrooli credentials doctor` to choose another credential backend."}
	case "empty":
		return []string{"Log in interactively once to create/unlock the login collection, then rerun this status command."}
	case "unsupported":
		return []string{"Use the platform credential backend reported by `vrooli credentials doctor`."}
	default:
		return nil
	}
}

func credentialsKeyringFile(ctx *CommandContext, args []string, repair bool) error {
	path, format, err := keyringFormatAndPath("credentials keyring", args)
	if err != nil {
		return err
	}
	var report keyring.KeyringReport
	if repair {
		report, err = keyring.Repair(path)
	} else {
		report, err = keyring.Inspect(path)
	}
	if err != nil {
		return err
	}
	if format == "json" {
		return json.NewEncoder(ctx.Stdout).Encode(report)
	}
	fmt.Fprintf(ctx.Stdout, "Keyring: %s\n  Loadable: %t\n  Repaired: %d\n", report.Path, report.Loadable, report.Repaired)
	if report.BackupPath != "" {
		fmt.Fprintf(ctx.Stdout, "  Backup:   %s\n", report.BackupPath)
	}
	for _, defect := range report.Defects {
		fmt.Fprintf(ctx.Stdout, "  Defect:   [%s] %s (%d lines; repairable=%t)\n", defect.Section, defect.Field, defect.LineCount, defect.Repairable)
		if defect.Reason != "" {
			fmt.Fprintf(ctx.Stdout, "            %s\n", defect.Reason)
		}
	}
	return nil
}

func credentialsKeyringUnlock(ctx *CommandContext, args []string, input io.Reader) error {
	if len(args) != 0 {
		return fmt.Errorf("credentials keyring unlock accepts no arguments")
	}
	if err := refuseInteractiveStdin(input,
		`printf '%s' "$PASSPHRASE" | vrooli credentials keyring unlock`); err != nil {
		return err
	}
	if input == nil {
		return fmt.Errorf("credentials keyring unlock reads the passphrase from standard input; pipe it in")
	}
	unlockCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := keyring.Unlock(unlockCtx, input); err != nil {
		return err
	}
	_, err := fmt.Fprintln(ctx.Stdout, "Login keyring unlock requested. Credential values were not read or printed.")
	return err
}
