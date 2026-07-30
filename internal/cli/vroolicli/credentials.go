package vroolicli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/resources/securestore"
	credentialauthority "github.com/vrooli/vrooli/internal/secrets"
)

// runCredentialsCommand owns all local credential writes. Values are accepted
// exclusively through stdin so they do not enter argv, command metrics, shell
// history, or status output.
func (app *App) runCredentialsCommand(ctx *CommandContext, args []string) error {
	if len(args) == 0 || commandtree.WantsHelp(args) {
		fmt.Fprintln(ctx.Stdout, "Usage:\n  vrooli credentials provision --identity <namespace/name> --field <field> < value\n  vrooli credentials status --identity <namespace/name> --field <field> [--format json]\n  vrooli credentials recovery export --entry <identity>:<field> --output <bundle> < passphrase\n  vrooli credentials recovery restore --input <bundle> < passphrase\n\nCredential values and recovery passphrases are read only from standard input and never printed.")
		return nil
	}
	switch args[0] {
	case "provision":
		return provisionCredential(ctx, args[1:], os.Stdin)
	case "status":
		return credentialStatus(ctx, args[1:])
	case "recovery":
		return recoveryCredentials(ctx, args[1:], os.Stdin)
	default:
		return fmt.Errorf("unknown credentials command %q", args[0])
	}
}

func nativeCredentialAuthority() (*credentialauthority.Authority, error) {
	return credentialauthority.NewAuthority(securestore.Default())
}

func credentialFlags(name string, args []string) (credentialauthority.Identity, string, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	identityRaw, field := "", ""
	fs.StringVar(&identityRaw, "identity", "", "logical credential identity")
	fs.StringVar(&field, "field", "value", "credential field")
	if err := fs.Parse(args); err != nil {
		return "", "", err
	}
	if len(fs.Args()) != 0 {
		return "", "", fmt.Errorf("%s accepts no positional arguments", name)
	}
	identity, err := credentialauthority.ParseIdentity(identityRaw)
	if err != nil {
		return "", "", err
	}
	field = strings.TrimSpace(field)
	if field == "" {
		return "", "", fmt.Errorf("credential field is required")
	}
	return identity, field, nil
}

func provisionCredential(ctx *CommandContext, args []string, input io.Reader) error {
	identity, field, err := credentialFlags("credentials provision", args)
	if err != nil {
		return err
	}
	value, err := io.ReadAll(io.LimitReader(input, 64*1024))
	if err != nil {
		return fmt.Errorf("read credential input: %w", err)
	}
	authority, err := nativeCredentialAuthority()
	if err != nil {
		return err
	}
	if err := authority.Put(identity, field, strings.TrimSpace(string(value))); err != nil {
		return err
	}
	_, err = fmt.Fprintf(ctx.Stdout, "Credential %s/%s provisioned in the native secure store.\n", identity, field)
	return err
}

func credentialStatus(ctx *CommandContext, args []string) error {
	fs := flag.NewFlagSet("credentials status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	identityRaw, field := "", ""
	format := "text"
	fs.StringVar(&identityRaw, "identity", "", "logical credential identity")
	fs.StringVar(&field, "field", "value", "credential field")
	fs.StringVar(&format, "format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("credentials status accepts no positional arguments")
	}
	identity, err := credentialauthority.ParseIdentity(identityRaw)
	if err != nil {
		return err
	}
	field = strings.TrimSpace(field)
	if field == "" {
		return fmt.Errorf("credential field is required")
	}
	authority, err := nativeCredentialAuthority()
	if err != nil {
		return err
	}
	status := authority.Status(identity, field)
	if format != "text" && format != "json" {
		return fmt.Errorf("credential status format must be text or json")
	}
	if format == "json" {
		return json.NewEncoder(ctx.Stdout).Encode(struct {
			Identity   credentialauthority.Identity `json:"identity"`
			Field      string                       `json:"field"`
			Configured bool                         `json:"configured"`
			Provider   string                       `json:"provider"`
		}{Identity: status.Identity, Field: status.Field, Configured: status.Configured, Provider: status.Provider})
	}
	_, err = fmt.Fprintf(ctx.Stdout, "Credential %s/%s: %s (%s)\n", identity, field, map[bool]string{true: "configured", false: "unconfigured"}[status.Configured], status.Provider)
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
	authority, err := nativeCredentialAuthority()
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
	authority, err := nativeCredentialAuthority()
	if err != nil {
		return err
	}
	if err := authority.RestoreRecovery(bundle, passphrase); err != nil {
		return err
	}
	_, err = fmt.Fprintln(ctx.Stdout, "Encrypted recovery bundle restored to the native secure store.")
	return err
}
