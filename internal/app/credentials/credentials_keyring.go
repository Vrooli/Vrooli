package credentials

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/cliout"
	keyring "github.com/vrooli/vrooli/internal/credentials"
	"github.com/vrooli/vrooli/internal/hostinventory"
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

func normalizeKeyringOptions(opts KeyringOptions) (string, string, error) {
	path, format := strings.TrimSpace(opts.Path), strings.TrimSpace(opts.Format)
	if format == "" {
		format = string(cliout.FormatHuman)
	}
	if format != string(cliout.FormatHuman) && format != string(cliout.FormatJSON) {
		return "", "", fmt.Errorf("keyring format must be text or json")
	}
	return path, format, nil
}

func (app *Service) KeyringStatus(operationCtx context.Context, out io.Writer, opts KeyringOptions) error {
	_, format, err := normalizeKeyringOptions(opts)
	if err != nil {
		return err
	}
	capability := hostinventory.CredentialStoreStatus(operationCtx)
	verdictReport, inspectErr := keyring.Inspect("")
	verdict := keyring.KeyringVerdict{State: keyring.KeyringAbsent, Reason: capability.Reason}
	if inspectErr == nil {
		verdict = keyring.DeriveKeyringVerdict(verdictReport, capability)
	}
	status := credentialKeyringStatus{
		State: string(verdict.State), Cause: verdict.Reason, Support: capability.Supported,
		Remedy: credentialKeyringRemedy(string(verdict.State)),
	}
	if format == string(cliout.FormatJSON) {
		return cliout.WriteJSONValue(out, status)
	}
	fmt.Fprintf(out, "Credential keyring: %s\n", status.State)
	if status.Cause != "" {
		fmt.Fprintf(out, "  Cause:  %s\n", status.Cause)
	}
	for _, remedy := range status.Remedy {
		fmt.Fprintf(out, "  Remedy: %s\n", remedy)
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

func (app *Service) KeyringFile(operationCtx context.Context, out io.Writer, opts KeyringOptions, repair bool) error {
	path, format, err := normalizeKeyringOptions(opts)
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
	capability := hostinventory.CredentialStoreStatus(operationCtx)
	verdict := keyring.DeriveKeyringVerdict(report, capability)
	report.Verdict = string(verdict.State)
	report.VerdictReason = verdict.Reason
	if format == string(cliout.FormatJSON) {
		return cliout.WriteJSONValue(out, report)
	}
	fmt.Fprintf(out, "Keyring: %s\n  Format:   %s\n", report.Path, keyringFormatLabel(report.Format))
	// Never print a verdict the inspection did not reach. An encrypted keyring
	// is opaque here, and printing "Loadable: true" for one is how a wedged
	// host reads as healthy.
	if report.Assessed {
		fmt.Fprintf(out, "  Loadable: %t\n", report.Loadable)
	} else {
		fmt.Fprintf(out, "  Loadable: unknown (file contents are opaque to inspection; run `vrooli credentials keyring repair` to check the live store)\n")
	}
	if report.Verdict != "" {
		fmt.Fprintf(out, "  Verdict:   %s\n", report.Verdict)
		if report.VerdictReason != "" {
			fmt.Fprintf(out, "  Reason:    %s\n", report.VerdictReason)
		}
	}
	fmt.Fprintf(out, "  Repaired: %d\n", report.Repaired)
	if report.StaleDaemonCheck != "" {
		fmt.Fprintf(out, "  Daemon:   %s\n", keyringDaemonLabel(report))
	}
	if report.BackupPath != "" {
		fmt.Fprintf(out, "  Backup:   %s\n", report.BackupPath)
	}
	for _, backup := range report.Backups {
		fmt.Fprintf(out, "  Backup file: %s (age=%s)\n", backup.Path, formatKeyringAge(backup.AgeSeconds))
	}
	for _, defect := range report.Defects {
		fmt.Fprintf(out, "  Defect:   [%s] %s (%d lines; repairable=%t)\n", defect.Section, defect.Field, defect.LineCount, defect.Repairable)
		if defect.Reason != "" {
			fmt.Fprintf(out, "            %s\n", defect.Reason)
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

func (app *Service) KeyringRepair(operationCtx context.Context, out io.Writer, opts KeyringRepairOptions) error {
	path, format, err := normalizeKeyringOptions(KeyringOptions{Path: opts.Path, Format: opts.Format})
	if err != nil {
		return err
	}
	retireBackup := strings.TrimSpace(opts.RetireBackup)
	if format != string(cliout.FormatHuman) && format != string(cliout.FormatJSON) {
		return fmt.Errorf("credentials keyring repair format must be text or json")
	}
	retirementThreshold := opts.OfferRetireOlderThan
	if retirementThreshold < 0 {
		return fmt.Errorf("offer-retire-older-than must be a positive duration")
	}
	repairCtx, cancel := context.WithTimeout(operationCtx, tuning.CredentialRepairTimeout())
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
	if format == string(cliout.FormatJSON) {
		if err := cliout.WriteJSONValue(out, report); err != nil {
			return err
		}
		return keyringRepairExit(report)
	}

	fmt.Fprintf(out, "Credential store repair: %s\n", repairHeadline(report))
	fmt.Fprintf(out, "  Host:  %s (adapter %s)\n", report.Platform, report.Adapter)
	fmt.Fprintf(out, "  State: %s -> %s\n\n", report.StateBefore, report.StateAfter)
	for _, rung := range report.Rungs {
		fmt.Fprintf(out, "  [%-14s] %s\n", rung.Status, rung.Name)
		if rung.Detail != "" {
			fmt.Fprintf(out, "                   %s\n", rung.Detail)
		}
		if rung.Action != "" {
			fmt.Fprintf(out, "                   ran: %s\n", rung.Action)
		}
	}
	if len(report.Remedy) > 0 {
		fmt.Fprintln(out, "\n  Next:")
		for _, remedy := range report.Remedy {
			fmt.Fprintf(out, "    - %s\n", remedy)
		}
	}
	if len(report.RetirementOffers) > 0 {
		fmt.Fprintln(out, "\n  Retirement offers:")
		for _, offer := range report.RetirementOffers {
			fmt.Fprintf(out, "    - %s\n", offer)
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

func (app *Service) KeyringUnlock(operationCtx context.Context, out, errOut io.Writer, input io.Reader) error {
	passphrase, err := keyringPassphrase(input, errOut)
	if err != nil {
		return err
	}
	unlockCtx, cancel := context.WithTimeout(operationCtx, tuning.ReloadFallbackGracePeriod())
	defer cancel()
	if err := keyring.Unlock(unlockCtx, strings.NewReader(passphrase)); err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, "Login keyring unlock requested. Credential values were not read or printed.")
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
