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
			"`repair` walks the whole credential-store ladder: it identifies this host's backend,\n"+
			"repairs the on-disk keyring, restarts a wedged credential daemon, and reports the lock\n"+
			"state. Rungs that do not apply to this platform say so; a rung with no automated remedy\n"+
			"names what only a person can do. `inspect` is the read-only file-level subset.\n\n"+
			"Keyring inspection and repair never print credential values. Unlock reads the passphrase from standard input only.")
		return nil
	}
	switch args[0] {
	case "status":
		return credentialsKeyringStatusCommand(ctx, args[1:])
	case "inspect":
		return credentialsKeyringFile(ctx, args[1:], false)
	case "repair":
		return credentialsKeyringRepair(ctx, args[1:])
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

// credentialKeyringRemedy names the next command for each observed state.
//
// Every entry must be an action. The previous "unresponsive" remedy told the
// operator to rerun the diagnostic they had just run, which is a loop, not a
// remedy — and on the host that motivated this change it kept a wedged daemon
// in place for four days while three separate tools reported the fault
// correctly and none of them offered a way out.
func credentialKeyringRemedy(state string) []string {
	switch state {
	case "locked":
		return []string{
			"Pipe the login-keyring passphrase to `vrooli credentials keyring unlock`.",
			"If this is an autologin host, disable autologin and log in interactively, or opt into the high-risk login-keyring unlock safeguard.",
		}
	case "unresponsive":
		return []string{"Run `vrooli credentials keyring repair`. It restarts the credential daemon when this host has a restartable one, re-probes the store to prove the restart helped, and tells you what only a person can do when it did not."}
	case "unavailable":
		return []string{
			"Run `vrooli credentials keyring repair` to confirm whether a credential daemon is reachable at all on this host.",
			"If it reports none, run `vrooli credentials doctor` to choose another credential backend.",
		}
	case "empty":
		return []string{"Log in interactively once to create/unlock the login collection, then rerun `vrooli credentials keyring repair`."}
	case "unsupported":
		return []string{"Use the platform credential backend reported by `vrooli credentials doctor`."}
	case "ready":
		return nil
	default:
		return []string{"Run `vrooli credentials keyring repair` for a full ladder walk of this host's credential store."}
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
	fmt.Fprintf(ctx.Stdout, "Keyring: %s\n  Format:   %s\n", report.Path, keyringFormatLabel(report.Format))
	// Never print a verdict the inspection did not reach. An encrypted keyring
	// is opaque here, and printing "Loadable: true" for one is how a wedged
	// host reads as healthy.
	if report.Assessed {
		fmt.Fprintf(ctx.Stdout, "  Loadable: %t\n", report.Loadable)
	} else {
		fmt.Fprintf(ctx.Stdout, "  Loadable: unknown (file contents are opaque to inspection; run `vrooli credentials keyring repair` to check the live store)\n")
	}
	fmt.Fprintf(ctx.Stdout, "  Repaired: %d\n", report.Repaired)
	if report.StaleDaemonCheck != "" {
		fmt.Fprintf(ctx.Stdout, "  Daemon:   %s\n", keyringDaemonLabel(report))
	}
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

func keyringFormatLabel(format string) string {
	if strings.TrimSpace(format) == "" {
		return "unrecognized"
	}
	return format
}

func keyringDaemonLabel(report keyring.KeyringReport) string {
	switch {
	case report.StaleDaemonCheck != "checked":
		return "not checked on this host"
	case report.StaleDaemon:
		return report.StaleDaemonDetail
	default:
		return "running daemon is at least as new as the file"
	}
}

// credentialsKeyringRepair walks the whole credential-store ladder rather than
// only rewriting a file, and exits non-zero when the store is still broken.
//
// The exit code matters: this command is what an operator and an autoheal check
// both run, and a repair that leaves the store unreachable must not report
// success. The previous file-only repair exited zero on a host whose credential
// daemon had been wedged for four days.
func credentialsKeyringRepair(ctx *CommandContext, args []string) error {
	path, format, err := keyringFormatAndPath("credentials keyring repair", args)
	if err != nil {
		return err
	}
	repairCtx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	report, err := keyring.RepairStore(repairCtx, path)
	if err != nil {
		return err
	}
	if format == "json" {
		if err := json.NewEncoder(ctx.Stdout).Encode(report); err != nil {
			return err
		}
		return keyringRepairExit(report)
	}

	fmt.Fprintf(ctx.Stdout, "Credential store repair: %s\n", repairHeadline(report))
	fmt.Fprintf(ctx.Stdout, "  Host:  %s (adapter %s)\n", report.Platform, report.Adapter)
	fmt.Fprintf(ctx.Stdout, "  State: %s -> %s\n\n", report.StateBefore, report.StateAfter)
	for _, rung := range report.Rungs {
		fmt.Fprintf(ctx.Stdout, "  [%-14s] %s\n", rung.Status, rung.Name)
		if rung.Detail != "" {
			fmt.Fprintf(ctx.Stdout, "                   %s\n", rung.Detail)
		}
		if rung.Action != "" {
			fmt.Fprintf(ctx.Stdout, "                   ran: %s\n", rung.Action)
		}
	}
	if len(report.Remedy) > 0 {
		fmt.Fprintln(ctx.Stdout, "\n  Next:")
		for _, remedy := range report.Remedy {
			fmt.Fprintf(ctx.Stdout, "    - %s\n", remedy)
		}
	}
	return keyringRepairExit(report)
}

func repairHeadline(report keyring.RepairReport) string {
	if report.Resolved {
		return "resolved"
	}
	return "unresolved"
}

// keyringRepairExit fails the command when the store is still broken.
//
// `credentials doctor` reports a condition and exits zero because it is purely
// a diagnostic. `repair` is not: it is what an operator and an autoheal check
// run to make the store work, so it must exit non-zero when it did not. The
// message stays one line because the rungs above already explained why.
func keyringRepairExit(report keyring.RepairReport) error {
	if report.Resolved {
		return nil
	}
	return fmt.Errorf("credential store is still %s; see the rungs and remedy above", report.StateAfter)
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
