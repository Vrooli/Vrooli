package vroolicli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/resources/securestore"
	credentialauthority "github.com/vrooli/vrooli/internal/secrets"
)

// runCredentialsCommand owns all local credential writes. Values are accepted
// exclusively through stdin so they do not enter argv, command metrics, shell
// history, or status output.
func (app *App) runCredentialsCommand(ctx *CommandContext, args []string) error {
	if len(args) == 0 || commandtree.WantsHelp(args) {
		fmt.Fprintln(ctx.Stdout, "Usage:\n  vrooli credentials doctor [--format json]\n  vrooli credentials list [--format json]\n  vrooli credentials provision --identity <namespace/name> --field <field> < value\n  vrooli credentials status --identity <namespace/name> --field <field> [--format json]\n  vrooli credentials delete --identity <namespace/name> --field <field> --yes\n  vrooli credentials keyring <inspect|repair> [--path <file>] [--format json]\n  vrooli credentials store <status|init|unlock|lock|rewrap>\n  vrooli credentials recovery export --entry <identity>:<field> --output <bundle> < passphrase\n  vrooli credentials recovery restore --input <bundle> < passphrase\n\nCredential values, store passphrases, and recovery passphrases are read only from standard input and never printed.\n`credentials store` manages the encrypted backend used on a host with no native credential store.\n`credentials keyring` diagnoses and repairs a GNOME keyring file the desktop daemon refuses to load.")
		return nil
	}
	switch args[0] {
	case "doctor":
		return credentialsDoctor(ctx, args[1:])
	case "keyring":
		return credentialsKeyring(ctx, args[1:])
	case "list":
		return credentialsList(ctx, args[1:])
	case "provision":
		return provisionCredential(ctx, args[1:], os.Stdin)
	case "status":
		return credentialStatus(ctx, args[1:])
	case "delete":
		return deleteCredential(ctx, args[1:])
	case "store":
		return credentialsStore(ctx, args[1:], os.Stdin)
	case "recovery":
		return recoveryCredentials(ctx, args[1:], os.Stdin)
	default:
		return fmt.Errorf("unknown credentials command %q", args[0])
	}
}

// credentialAuthority is the one construction path every credential subcommand
// uses. It is not named "native": on a host with no native store the authority
// it returns is the encrypted file store, and every command below works there
// unchanged.
func credentialAuthority() (*credentialauthority.Authority, error) {
	return credentialauthority.DefaultAuthority()
}

// credentialSelectorFlags is the one parser for the --identity/--field pair
// every credential subcommand accepts, plus the optional --format the read-only
// ones add. Three near-identical copies of this had already drifted apart in
// how they reported a bad field.
func credentialSelectorFlags(name string, args []string, withFormat bool) (credentialauthority.Identity, string, string, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	identityRaw, field, format := "", "", "text"
	fs.StringVar(&identityRaw, "identity", "", "logical credential identity")
	fs.StringVar(&field, "field", "value", "credential field")
	if withFormat {
		fs.StringVar(&format, "format", "text", "output format: text or json")
	}
	if err := fs.Parse(args); err != nil {
		return "", "", "", err
	}
	if len(fs.Args()) != 0 {
		return "", "", "", fmt.Errorf("%s accepts no positional arguments", name)
	}
	identity, err := credentialauthority.ParseIdentity(identityRaw)
	if err != nil {
		return "", "", "", err
	}
	field = strings.TrimSpace(field)
	if field == "" {
		return "", "", "", fmt.Errorf("credential field is required")
	}
	format = strings.TrimSpace(format)
	if format != "text" && format != "json" {
		return "", "", "", fmt.Errorf("%s format must be text or json", name)
	}
	return identity, field, format, nil
}

func provisionCredential(ctx *CommandContext, args []string, input io.Reader) error {
	identity, field, _, err := credentialSelectorFlags("credentials provision", args, false)
	if err != nil {
		return err
	}
	value, err := io.ReadAll(io.LimitReader(input, 64*1024))
	if err != nil {
		return fmt.Errorf("read credential input: %w", err)
	}
	authority, err := credentialAuthority()
	if err != nil {
		return err
	}
	if err := authority.Put(identity, field, strings.TrimSpace(string(value))); err != nil {
		return err
	}
	// The backend is named rather than assumed: on a headless host the value
	// went into the encrypted store, and telling the operator it went into "the
	// native secure store" would be false on exactly the hosts this path exists
	// for.
	_, err = fmt.Fprintf(ctx.Stdout, "Credential %s/%s provisioned in the %s credential store.\n",
		identity, field, authority.Provider())
	return err
}

func credentialStatus(ctx *CommandContext, args []string) error {
	identity, field, format, err := credentialSelectorFlags("credentials status", args, true)
	if err != nil {
		return err
	}
	authority, err := credentialAuthority()
	if err != nil {
		return err
	}
	status := authority.Status(identity, field)
	if format == "json" {
		return json.NewEncoder(ctx.Stdout).Encode(status)
	}
	// The provider state travels with the answer so `configured: false` can
	// never be misread as "the operator never set this" while the store is down.
	_, err = fmt.Fprintf(ctx.Stdout, "Credential %s/%s: %s (provider %s, %s)\n",
		identity, field,
		map[bool]string{true: "configured", false: "unconfigured"}[status.Configured],
		status.Provider, status.ProviderState)
	if err != nil {
		return err
	}
	if status.ProviderDetail != "" {
		_, err = fmt.Fprintf(ctx.Stdout, "  %s\n  Run `vrooli credentials doctor` for the host diagnosis.\n", status.ProviderDetail)
	}
	return err
}

// deleteCredential is the deprovision half of the operator surface. Without it
// a leaked or rotated key could be written but never revoked through the
// documented interface, and the desktop runtime was the only caller of
// Authority.Delete.
func deleteCredential(ctx *CommandContext, args []string) error {
	fs := flag.NewFlagSet("credentials delete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	identityRaw, field := "", ""
	confirmed := false
	fs.StringVar(&identityRaw, "identity", "", "logical credential identity")
	fs.StringVar(&field, "field", "value", "credential field")
	fs.BoolVar(&confirmed, "yes", false, "confirm the removal; required because a deleted credential is unrecoverable without a recovery bundle")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("credentials delete accepts no positional arguments")
	}
	identity, err := credentialauthority.ParseIdentity(identityRaw)
	if err != nil {
		return err
	}
	field = strings.TrimSpace(field)
	if field == "" {
		return fmt.Errorf("credential field is required")
	}
	if !confirmed {
		return fmt.Errorf(
			"refusing to delete %s/%s without --yes: removal is unrecoverable unless the value is in a recovery bundle",
			identity, field)
	}

	authority, err := credentialAuthority()
	if err != nil {
		return err
	}
	// Read the state first so the operator learns whether anything was actually
	// removed, and so a provider outage cannot be reported as a clean delete.
	status := authority.Status(identity, field)
	if status.ProviderState != credentialauthority.ProviderAvailable {
		return fmt.Errorf("cannot delete %s/%s: %s; run `vrooli credentials doctor` for the host diagnosis",
			identity, field, status.ProviderDetail)
	}
	if err := authority.Delete(identity, field); err != nil {
		return err
	}
	if !status.Configured {
		_, err = fmt.Fprintf(ctx.Stdout, "Credential %s/%s was not configured; nothing to remove.\n", identity, field)
		return err
	}
	_, err = fmt.Fprintf(ctx.Stdout,
		"Credential %s/%s removed from the %s credential store. Resources that declare it now report unhealthy until it is provisioned again.\n",
		identity, field, status.Provider)
	return err
}

type credentialEntries []string

func (entries *credentialEntries) String() string { return strings.Join(*entries, ",") }
func (entries *credentialEntries) Set(value string) error {
	*entries = append(*entries, value)
	return nil
}

func recoveryCredentials(ctx *CommandContext, args []string, input io.Reader) error {
	if len(args) == 0 || commandtree.WantsHelp(args) {
		fmt.Fprintln(ctx.Stdout, "Usage:\n  vrooli credentials recovery export --entry <identity>:<field> --output <bundle> < passphrase\n  vrooli credentials recovery restore --input <bundle> < passphrase")
		return nil
	}
	switch args[0] {
	case "export":
		return exportCredentialRecovery(ctx, args[1:], input)
	case "restore":
		return restoreCredentialRecovery(ctx, args[1:], input)
	default:
		return fmt.Errorf("unknown credentials recovery command %q", args[0])
	}
}

func recoveryPassphrase(input io.Reader) (string, error) {
	value, err := io.ReadAll(io.LimitReader(input, 64*1024))
	if err != nil {
		return "", fmt.Errorf("read recovery passphrase: %w", err)
	}
	passphrase := strings.TrimSpace(string(value))
	if passphrase == "" {
		return "", fmt.Errorf("recovery passphrase is required")
	}
	return passphrase, nil
}

func exportCredentialRecovery(ctx *CommandContext, args []string, input io.Reader) error {
	fs := flag.NewFlagSet("credentials recovery export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var entries credentialEntries
	output := ""
	fs.Var(&entries, "entry", "credential entry in identity:field form; repeat for each entry")
	fs.StringVar(&output, "output", "", "new encrypted recovery bundle path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 || len(entries) == 0 || strings.TrimSpace(output) == "" {
		return fmt.Errorf("recovery export requires at least one --entry and --output")
	}
	selected := make([]credentialauthority.RecoveryEntry, 0, len(entries))
	for _, raw := range entries {
		identityRaw, field, ok := strings.Cut(strings.TrimSpace(raw), ":")
		if !ok || strings.TrimSpace(field) == "" {
			return fmt.Errorf("recovery entry must use identity:field form")
		}
		identity, err := credentialauthority.ParseIdentity(identityRaw)
		if err != nil {
			return err
		}
		selected = append(selected, credentialauthority.RecoveryEntry{Identity: identity, Field: field})
	}
	passphrase, err := recoveryPassphrase(input)
	if err != nil {
		return err
	}
	authority, err := credentialAuthority()
	if err != nil {
		return err
	}
	bundle, err := authority.ExportRecovery(selected, passphrase)
	if err != nil {
		return err
	}
	output = filepath.Clean(output)
	file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create recovery bundle: %w", err)
	}
	if _, err := file.Write(bundle); err != nil {
		_ = file.Close()
		_ = os.Remove(output)
		return fmt.Errorf("write recovery bundle: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(output)
		return fmt.Errorf("close recovery bundle: %w", err)
	}
	_, err = fmt.Fprintf(ctx.Stdout, "Encrypted recovery bundle created for %d credential entries.\n", len(selected))
	return err
}

func restoreCredentialRecovery(ctx *CommandContext, args []string, input io.Reader) error {
	fs := flag.NewFlagSet("credentials recovery restore", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := ""
	fs.StringVar(&path, "input", "", "encrypted recovery bundle path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 || strings.TrimSpace(path) == "" {
		return fmt.Errorf("recovery restore requires --input")
	}
	passphrase, err := recoveryPassphrase(input)
	if err != nil {
		return err
	}
	bundle, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("read recovery bundle: %w", err)
	}
	authority, err := credentialAuthority()
	if err != nil {
		return err
	}
	if err := authority.RestoreRecovery(bundle, passphrase); err != nil {
		return err
	}
	_, err = fmt.Fprintf(ctx.Stdout, "Encrypted recovery bundle restored to the %s credential store.\n", authority.Provider())
	return err
}

// credentialsKeyring is the operator's way out of a keyring file the desktop
// daemon will not load.
//
// It exists because that state has no other exit. GNOME Keyring rejects the
// whole file over one malformed entry, so every Secret Service API — including
// the one the rest of this command group uses — reports the keyring as simply
// absent. Diagnosing it by hand means knowing an undocumented text format, and
// the operator most likely to hit it is locked out of the desktop that would
// let them look.
func credentialsKeyring(ctx *CommandContext, args []string) error {
	if len(args) == 0 || commandtree.WantsHelp(args) {
		fmt.Fprintln(ctx.Stdout, "Usage:\n  vrooli credentials keyring inspect [--path <file>] [--format json]\n  vrooli credentials keyring repair [--path <file>] [--format json]\n\nInspect reports which entries stop GNOME Keyring from loading a keyring file.\nRepair rewrites the Vrooli-owned entries among them, after taking a backup, and\ndeclines entries written by other applications. Neither prints secret material.")
		return nil
	}

	action := args[0]
	if action != "inspect" && action != "repair" {
		return fmt.Errorf("unknown credentials keyring command %q; use inspect or repair", action)
	}

	fs := flag.NewFlagSet("credentials keyring "+action, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path, format := "", "text"
	fs.StringVar(&path, "path", "", "keyring file to examine; defaults to every keyring in the user's keyring directory")
	fs.StringVar(&format, "format", "text", "output format: text or json")
	if err := fs.Parse(args[1:]); err != nil {
		return fmt.Errorf("credentials keyring %s: %w", action, err)
	}
	if format != "text" && format != "json" {
		return fmt.Errorf("credentials keyring %s: --format must be text or json", action)
	}

	targets, err := keyringTargets(path)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		if format == "json" {
			return json.NewEncoder(ctx.Stdout).Encode([]securestore.KeyringReport{})
		}
		_, err := fmt.Fprintln(ctx.Stdout, "No keyring files found. This host stores credentials somewhere other than a GNOME keyring; run `vrooli credentials doctor` for the backend it does use.")
		return err
	}

	reports := make([]securestore.KeyringReport, 0, len(targets))
	var failed error
	for _, target := range targets {
		var report securestore.KeyringReport
		var reportErr error
		if action == "repair" {
			report, reportErr = securestore.RepairKeyringFile(target)
		} else {
			report, reportErr = securestore.InspectKeyringFile(target)
		}
		if reportErr != nil {
			// One unreadable keyring must not hide the diagnosis of the others,
			// which is the whole reason an operator is running this.
			failed = reportErr
			report.Path = target
		}
		reports = append(reports, report)
	}

	// Sweeping is part of repair, not of inspection: a read-only diagnosis must
	// stay read-only, and an operator running inspect is often deciding whether
	// to trust this command with a write at all.
	var swept []string
	if action == "repair" && strings.TrimSpace(path) == "" {
		dir, dirErr := securestore.DefaultKeyringDir()
		if dirErr != nil {
			return dirErr
		}
		swept, err = securestore.SweepAbandonedTemporaries(dir, time.Now())
		if err != nil && failed == nil {
			failed = err
		}
	}

	if format == "json" {
		payload := struct {
			Reports []securestore.KeyringReport `json:"reports"`
			Swept   []string                    `json:"sweptTemporaries,omitempty"`
		}{Reports: reports, Swept: swept}
		if err := json.NewEncoder(ctx.Stdout).Encode(payload); err != nil {
			return err
		}
		return failed
	}
	if err := writeKeyringReports(ctx, action, reports); err != nil {
		return err
	}
	if len(swept) > 0 {
		// Each of these held a full copy of every secret in the keyring, so the
		// count is worth stating rather than folding into a silent cleanup.
		if _, err := fmt.Fprintf(ctx.Stdout, "Removed %d abandoned keyring temporar%s, each a full copy of the keyring's secrets.\n",
			len(swept), map[bool]string{true: "y", false: "ies"}[len(swept) == 1]); err != nil {
			return err
		}
	}
	return failed
}

// keyringTargets resolves which files to examine. An explicit --path wins;
// otherwise every keyring in the user's keyring directory is examined, because
// an operator who has been locked out does not know which file is at fault.
func keyringTargets(path string) ([]string, error) {
	if strings.TrimSpace(path) != "" {
		return []string{path}, nil
	}
	dir, err := securestore.DefaultKeyringDir()
	if err != nil {
		return nil, err
	}
	matches, err := filepath.Glob(filepath.Join(dir, "*.keyring"))
	if err != nil {
		return nil, fmt.Errorf("list keyring files: %w", err)
	}
	return matches, nil
}

func writeKeyringReports(ctx *CommandContext, action string, reports []securestore.KeyringReport) error {
	for _, report := range reports {
		state := "loadable"
		if !report.Loadable {
			state = "NOT loadable by GNOME Keyring"
		}
		if _, err := fmt.Fprintf(ctx.Stdout, "%s: %s\n", report.Path, state); err != nil {
			return err
		}
		if report.BackupPath != "" {
			if _, err := fmt.Fprintf(ctx.Stdout, "  backup: %s\n", report.BackupPath); err != nil {
				return err
			}
		}
		for _, defect := range report.Defects {
			label := defect.Label
			if label == "" {
				label = "(no label)"
			}
			disposition := "REPAIRABLE — run `vrooli credentials keyring repair`"
			switch {
			case action == "repair" && defect.Repairable:
				disposition = "repaired"
			case !defect.Repairable:
				disposition = "left alone — " + defect.Reason
			}
			if _, err := fmt.Fprintf(ctx.Stdout, "  [%s] %s: field %q spans %d lines; %s\n",
				defect.Section, label, defect.Field, defect.LineCount, disposition); err != nil {
				return err
			}
		}
		if action == "repair" && report.Repaired > 0 {
			if _, err := fmt.Fprintf(ctx.Stdout, "  repaired %d entr%s — log out and back in so the keyring daemon reloads the file\n",
				report.Repaired, map[bool]string{true: "y", false: "ies"}[report.Repaired == 1]); err != nil {
				return err
			}
		}
	}
	return nil
}
