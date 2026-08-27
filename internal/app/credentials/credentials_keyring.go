package credentials

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cliout"
	keyring "github.com/vrooli/vrooli/internal/credentials"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/logx"
)

const (
	credentialsKeyringParameterA = 1024
)

const (
	credentialsKeyringParameterB = 64
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
			"  vrooli credentials keyring repair [--path <path>] [--retire-backup <exact-path>] [--offer-retire-older-than <duration>] [--format json]\n"+
			"  vrooli credentials keyring unlock\n\n"+
			"`repair` walks the whole credential-store ladder: it identifies this host's backend,\n"+
			"repairs the on-disk keyring, restarts a wedged credential daemon, and reports the lock\n"+
			"state. Rungs that do not apply to this platform say so; a rung with no automated remedy\n"+
			"names what only a person can do. `inspect` is the read-only file-level subset.\n\n"+
			"Keyring inspection and repair never print credential values. Unlock prompts securely in the terminal.")
		return nil
	}
	handlers := map[string]func([]string) error{
		"status":  func(args []string) error { return credentialsKeyringStatusCommand(ctx, args) },
		"inspect": func(args []string) error { return credentialsKeyringFile(ctx, args, false) },
		"repair":  func(args []string) error { return credentialsKeyringRepair(ctx, args) },
		"unlock":  func(args []string) error { return credentialsKeyringUnlock(ctx, args, input) },
	}
	handler, ok := handlers[args[0]]
	if !ok {
		return fmt.Errorf("unknown credentials keyring command %q", args[0])
	}
	return handler(args[1:])
}

func keyringFormatAndPath(name string, args []string) (string, string, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path, format := "", credentialsText
	fs.StringVar(&path, "path", "", "keyring path")
	fs.StringVar(&format, "format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return "", "", err
	}
	if len(fs.Args()) != 0 {
		return "", "", fmt.Errorf("%s accepts no positional arguments", name)
	}
	format = strings.TrimSpace(format)
	if format != credentialsText && format != string(logx.FormatJSON) {
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
	verdictReport, inspectErr := keyring.Inspect("")
	verdict := keyring.KeyringVerdict{State: keyring.KeyringAbsent, Reason: capability.Reason}
	if inspectErr == nil {
		verdict = keyring.DeriveKeyringVerdict(verdictReport, capability)
	}
	status := credentialKeyringStatus{
		State: string(verdict.State), Cause: verdict.Reason, Support: capability.Supported,
		Remedy: credentialKeyringRemedy(string(verdict.State)),
	}
	if format == string(logx.FormatJSON) {
		return cliout.WriteJSONValue(ctx.Stdout, status)
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
			"Run `vrooli credentials keyring unlock` and enter the passphrase at its secure prompt.",
			"If this is an autologin host, disable autologin and log in interactively, or opt into the high-risk login-keyring unlock safeguard.",
		}
	case "unresponsive":
		return []string{"Run `vrooli credentials keyring repair`. It restarts the credential daemon when this host has a restartable one, re-probes the store to prove the restart helped, and tells you what only a person can do when it did not."}
	case "unavailable", "absent":
		return []string{
			"Run `vrooli credentials keyring repair` to confirm whether a credential daemon is reachable at all on this host.",
			"If it reports none, run `vrooli credentials doctor` to choose another credential backend.",
		}
	case "empty":
		return []string{"Log in interactively once to create/unlock the login collection, then rerun `vrooli credentials keyring repair`."}
	case "unsupported":
		return []string{"Use the platform credential backend reported by `vrooli credentials doctor`."}
	case "ready", "unlocked":
		return nil
	case "file_rejected":
		return []string{"Run `vrooli credentials keyring repair` to inspect and repair Vrooli-owned malformed entries."}
	case "daemon_stale":
		return []string{"Log out and back in so the keyring daemon reloads the current keyring file."}
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
	capability := hostinventory.CredentialStoreStatus(context.Background())
	verdict := keyring.DeriveKeyringVerdict(report, capability)
	report.Verdict = string(verdict.State)
	report.VerdictReason = verdict.Reason
	if format == string(logx.FormatJSON) {
		return cliout.WriteJSONValue(ctx.Stdout, report)
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
	if report.Verdict != "" {
		fmt.Fprintf(ctx.Stdout, "  Verdict:   %s\n", report.Verdict)
		if report.VerdictReason != "" {
			fmt.Fprintf(ctx.Stdout, "  Reason:    %s\n", report.VerdictReason)
		}
	}
	fmt.Fprintf(ctx.Stdout, "  Repaired: %d\n", report.Repaired)
	if report.StaleDaemonCheck != "" {
		fmt.Fprintf(ctx.Stdout, "  Daemon:   %s\n", keyringDaemonLabel(report))
	}
	if report.BackupPath != "" {
		fmt.Fprintf(ctx.Stdout, "  Backup:   %s\n", report.BackupPath)
	}
	for _, backup := range report.Backups {
		fmt.Fprintf(ctx.Stdout, "  Backup file: %s (age=%s)\n", backup.Path, formatKeyringAge(backup.AgeSeconds))
	}
	for _, defect := range report.Defects {
		fmt.Fprintf(ctx.Stdout, "  Defect:   [%s] %s (%d lines; repairable=%t)\n", defect.Section, defect.Field, defect.LineCount, defect.Repairable)
		if defect.Reason != "" {
			fmt.Fprintf(ctx.Stdout, "            %s\n", defect.Reason)
		}
	}
	return nil
}

func formatKeyringAge(seconds int64) string {
	if seconds < 0 {
		seconds = 0
	}
	return (time.Duration(seconds) * time.Second).String()
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
	fs := flag.NewFlagSet("credentials keyring repair", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path, format, retireBackup, offerOlderThan := "", credentialsText, "", ""
	fs.StringVar(&path, "path", "", "keyring path")
	fs.StringVar(&format, "format", "text", "output format: text or json")
	fs.StringVar(&retireBackup, "retire-backup", "", "explicit keyring backup path to retire after the repair")
	fs.StringVar(&offerOlderThan, "offer-retire-older-than", "", "offer explicit Vrooli retire commands for backups older than this duration (for example 720h)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("credentials keyring repair accepts no positional arguments")
	}
	format = strings.TrimSpace(format)
	if format != credentialsText && format != string(logx.FormatJSON) {
		return fmt.Errorf("credentials keyring repair format must be text or json")
	}
	var retirementThreshold time.Duration
	if strings.TrimSpace(offerOlderThan) != "" {
		parsedThreshold, parseErr := time.ParseDuration(strings.TrimSpace(offerOlderThan))
		if parseErr != nil || parsedThreshold <= 0 {
			return fmt.Errorf("offer-retire-older-than must be a positive duration such as 720h")
		}
		retirementThreshold = parsedThreshold
	}
	repairCtx, cancel := context.WithTimeout(context.Background(), tuning.CredentialRepairTimeout())
	defer cancel()
	report, err := keyring.RepairStore(repairCtx, path)
	if err != nil {
		return err
	}
	if strings.TrimSpace(retireBackup) != "" {
		if err := keyring.RetireBackup(retireBackup); err != nil {
			return err
		}
		report.Rungs = append(report.Rungs, keyring.Rung{Name: "retire_explicit_backup", Status: "repaired", Detail: "retired the explicitly named keyring backup"})
	}
	report.RetirementOffers = keyringRetirementOffers(report, retirementThreshold)
	if format == string(logx.FormatJSON) {
		if err := cliout.WriteJSONValue(ctx.Stdout, report); err != nil {
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
	if len(report.RetirementOffers) > 0 {
		fmt.Fprintln(ctx.Stdout, "\n  Retirement offers:")
		for _, offer := range report.RetirementOffers {
			fmt.Fprintf(ctx.Stdout, "    - %s\n", offer)
		}
	}
	return keyringRepairExit(report)
}

func keyringRetirementOffers(report keyring.RepairReport, threshold time.Duration) []string {
	if threshold <= 0 || report.File == nil {
		return nil
	}
	offers := make([]string, 0)
	for _, backup := range report.File.Backups {
		if time.Duration(backup.AgeSeconds)*time.Second < threshold {
			continue
		}
		offers = append(offers, fmt.Sprintf("%s is %s old; run `vrooli credentials keyring repair --retire-backup %s` to retire this exact file", backup.Path, formatKeyringAge(backup.AgeSeconds), backup.Path))
	}
	return offers
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
	passphrase, err := keyringPassphrase(input, ctx.Stderr)
	if err != nil {
		return err
	}
	unlockCtx, cancel := context.WithTimeout(context.Background(), tuning.ReloadFallbackGracePeriod())
	defer cancel()
	if err := keyring.Unlock(unlockCtx, strings.NewReader(passphrase)); err != nil {
		return err
	}
	_, err = fmt.Fprintln(ctx.Stdout, "Login keyring unlock requested. Credential values were not read or printed.")
	return err
}

func keyringPassphrase(input io.Reader, prompt io.Writer) (string, error) {
	if input == nil {
		return "", fmt.Errorf("login keyring passphrase is required")
	}
	if file, ok := input.(*os.File); ok {
		if info, err := file.Stat(); err == nil && info.Mode()&os.ModeCharDevice != 0 {
			value, readErr := readInteractivePassphrase(file, prompt)
			if readErr != nil {
				return "", readErr
			}
			if strings.TrimSpace(value) == "" {
				return "", fmt.Errorf("login keyring passphrase is required")
			}
			return value, nil
		}
	}
	value, err := io.ReadAll(io.LimitReader(input, credentialsKeyringParameterB*credentialsKeyringParameterA))
	if err != nil {
		return "", fmt.Errorf("read login keyring passphrase: %w", err)
	}
	passphrase := strings.TrimSpace(string(value))
	if passphrase == "" {
		return "", fmt.Errorf("login keyring passphrase is required")
	}
	return passphrase, nil
}
