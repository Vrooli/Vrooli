package vroolicli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/resources/securestore"
)

// The `credentials store` surface manages the encrypted credential store — the
// backend for a host with no native one. Native stores are managed by the
// operating system and have nothing here.
//
// Every passphrase arrives on standard input, never in an argument, for the
// same reason credential values do: an argument is visible in /proc, in a
// process listing, in shell history, and in command metrics.

func credentialsStore(ctx *CommandContext, args []string, input io.Reader) error {
	if len(args) == 0 || commandtree.WantsHelp(args) {
		fmt.Fprintln(ctx.Stdout, "Usage:\n"+
			"  vrooli credentials store status [--format json]\n"+
			"  vrooli credentials store init [--format json] < passphrase\n"+
			"  vrooli credentials store unlock < passphrase\n"+
			"  vrooli credentials store lock\n"+
			"  vrooli credentials store change-passphrase < current-passphrase\\nnew-passphrase\n"+
			"  vrooli credentials store rewrap < passphrase\n"+
			"  vrooli credentials store reselect [--format json]\n"+
			"  vrooli credentials store retire --backend encrypted-file\n\n"+
			"The encrypted store is the credential backend on a host with no native one.\n"+
			"A host whose TPM is reachable needs no passphrase and no unlock; supply one on\n"+
			"standard input for any host that has no host-bound wrap.")
		return nil
	}
	switch args[0] {
	case "status":
		return credentialsStoreStatus(ctx, args[1:])
	case "init":
		return credentialsStoreInit(ctx, args[1:], input)
	case "unlock":
		return credentialsStoreUnlock(ctx, args[1:], input)
	case "lock":
		return credentialsStoreLock(ctx, args[1:])
	case "change-passphrase":
		return credentialsStoreChangePassphrase(ctx, args[1:], input)
	case "rewrap":
		return credentialsStoreRewrap(ctx, args[1:], input)
	case "reselect":
		return credentialsStoreReselect(ctx, args[1:])
	case "retire":
		return credentialsStoreRetire(ctx, args[1:])
	default:
		return fmt.Errorf("unknown credentials store command %q", args[0])
	}
}

func credentialsStoreReselect(ctx *CommandContext, args []string) error {
	format, err := storeFormatFlag("credentials store reselect", args)
	if err != nil {
		return err
	}
	entries, err := credentialMigrationEntries(ctx.Root)
	if err != nil {
		return err
	}
	receipt, err := securestore.ReselectBackend(entries)
	if format == "json" {
		encodeErr := json.NewEncoder(ctx.Stdout).Encode(receipt)
		if err != nil {
			return err
		}
		return encodeErr
	}
	if err != nil {
		return err
	}
	if receipt.From == receipt.To {
		_, err = fmt.Fprintf(ctx.Stdout, "Credential backend %s is already selected; no migration was needed.\n", receipt.To)
		return err
	}
	_, err = fmt.Fprintf(ctx.Stdout, "Credential backend reselected from %s to %s; verified %d credential(s).\n",
		receipt.From, receipt.To, len(receipt.Verified))
	return err
}

func credentialsStoreRetire(ctx *CommandContext, args []string) error {
	fs := flag.NewFlagSet("credentials store retire", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	backend := ""
	fs.StringVar(&backend, "backend", "", "backend to retire")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 || strings.TrimSpace(backend) == "" {
		return fmt.Errorf("credentials store retire requires --backend encrypted-file")
	}
	if err := securestore.RetireEmptyBackend(backend); err != nil {
		return err
	}
	_, err := fmt.Fprintf(ctx.Stdout, "Retired the empty %s credential backend.\n", backend)
	return err
}

func credentialMigrationEntries(root string) ([]securestore.MigrationEntry, error) {
	entries, err := collectCredentialEntries(root)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	migrated := make([]securestore.MigrationEntry, 0, len(entries))
	for _, entry := range entries {
		key := entry.LogicalID + ":" + entry.Field
		name := "vrooli.credentials.v1/" + key
		if entry.LogicalID == "" || entry.Field == "" || seen[name] {
			continue
		}
		seen[name] = true
		migrated = append(migrated, securestore.MigrationEntry{Service: "vrooli.credentials.v1", Key: key})
	}
	return migrated, nil
}

// storePassphrase reads a passphrase from standard input. Unlike a credential
// value it is optional: a host whose host-bound wrap works needs none, and
// demanding one there would reintroduce the human-at-boot requirement the
// host-bound wrap exists to remove.
func storePassphrase(input io.Reader) (string, error) {
	if input == nil {
		return "", nil
	}
	if err := refuseInteractiveStdin(input,
		`printf '%s' "$PASSPHRASE" | vrooli credentials store <init|unlock|rewrap>`); err != nil {
		return "", err
	}
	value, err := io.ReadAll(io.LimitReader(input, 64*1024))
	if err != nil {
		return "", fmt.Errorf("read credential store passphrase: %w", err)
	}
	return strings.TrimSpace(string(value)), nil
}

func storeFormatFlag(name string, args []string) (string, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	format := "text"
	fs.StringVar(&format, "format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if len(fs.Args()) != 0 {
		return "", fmt.Errorf("%s accepts no positional arguments", name)
	}
	format = strings.TrimSpace(format)
	if format != "text" && format != "json" {
		return "", fmt.Errorf("%s format must be text or json", name)
	}
	return format, nil
}

func credentialsStoreStatus(ctx *CommandContext, args []string) error {
	format, err := storeFormatFlag("credentials store status", args)
	if err != nil {
		return err
	}
	status, err := securestore.DescribeStore()
	if err != nil {
		return err
	}
	if format == "json" {
		return json.NewEncoder(ctx.Stdout).Encode(status)
	}
	writeStoreStatus(ctx, status)
	return nil
}

func writeStoreStatus(ctx *CommandContext, status securestore.StoreStatus) {
	fmt.Fprintf(ctx.Stdout, "Encrypted credential store\n")
	fmt.Fprintf(ctx.Stdout, "  Path:        %s\n", status.Path)
	if !status.Initialized {
		fmt.Fprintf(ctx.Stdout, "  State:       not initialized\n")
		if status.HostBoundBlocked != "" {
			fmt.Fprintf(ctx.Stdout, "\nRun `vrooli credentials store init` and pipe a passphrase in on standard input.\n")
			fmt.Fprintf(ctx.Stdout, "The unattended host-bound wrap will not open on this host as it stands:\n  %s\n", status.HostBoundBlocked)
			return
		}
		fmt.Fprintf(ctx.Stdout, "\nRun `vrooli credentials store init` to create it. A host with a reachable TPM\nneeds no passphrase; otherwise pipe one in on standard input.\n")
		return
	}
	fmt.Fprintf(ctx.Stdout, "  State:       initialized, %d credential(s)\n", status.Entries)
	fmt.Fprintf(ctx.Stdout, "  Authority:   %s\n",
		map[bool]string{true: "yes — this host's credentials live here", false: "no — a native store is the authority on this host"}[status.Active])
	fmt.Fprintf(ctx.Stdout, "  Unlocked:    %t\n", status.Unlocked)
	if status.ActiveWrap != "" {
		fmt.Fprintf(ctx.Stdout, "  Opened by:   %s (%s)\n", status.ActiveWrap, status.ActiveKeyStore)
	}
	switch {
	case status.ActiveWrap == "host-bound":
		fmt.Fprintf(ctx.Stdout, "  Unlock kept: not needed — the host-bound wrap opens this store with no human action\n")
	case status.ActiveWrap == "native-wrap":
		fmt.Fprintf(ctx.Stdout, "  Unlock kept: not needed — the native platform wrap opens this store with no human action\n")
	case status.UnlockCache != "":
		fmt.Fprintf(ctx.Stdout, "  Unlock kept: %s (session tmpfs; gone at logout)\n", status.UnlockCache)
	default:
		fmt.Fprintf(ctx.Stdout, "  Unlock kept: nowhere — this host has no session-scoped memory, so an unlock lasts one command\n")
	}
	fmt.Fprintf(ctx.Stdout, "  Key wraps:\n")
	for _, wrap := range status.Wraps {
		fmt.Fprintf(ctx.Stdout, "    %-12s %s%s\n", wrap.Provider, wrap.KeyStore, keyStoreCaveat(wrap.KeyStore))
	}
	// A store with no host-bound wrap, on a host that could have one, is the
	// case worth naming: the operator is typing a passphrase after every boot
	// and may not know that is avoidable.
	if status.HostBoundBlocked != "" && !hasWrap(status, "host-bound") {
		fmt.Fprintf(ctx.Stdout, "\nThis store has no unattended wrap, so it needs a passphrase after every reboot.\n  %s\n", status.HostBoundBlocked)
		fmt.Fprintf(ctx.Stdout, "Once `vrooli setup` has granted it, `vrooli credentials store rewrap` adds the\nhost-bound wrap. It keeps the same data key, so no stored value is re-encrypted.\n")
	}
}

func hasWrap(status securestore.StoreStatus, provider string) bool {
	for _, wrap := range status.Wraps {
		if wrap.Provider == provider {
			return true
		}
	}
	return false
}

// keyStoreCaveat states the difference between the wraps rather than letting an
// operator assume one uniform level of protection. On hardware with no TPM,
// systemd-creds protects the wrap with a key on the same disk, so possession of
// the disk — the Pi's SD card — is enough.
func keyStoreCaveat(keyStore string) string {
	switch keyStore {
	case "tpm2":
		return " — bound to this host's TPM; disk theft alone does not open it"
	case "host-key":
		return " — bound to a root-owned key on this same disk; possession of the disk opens it"
	case "operator-passphrase":
		return " — opens only with the passphrase you supply"
	case "unencrypted-keyring":
		return " — values are readable with a text editor; file mode is the only protection"
	case "encrypted-keyring":
		return " — the keyring file is not readable as a plaintext GKeyFile"
	case "keychain":
		return " — protected by the macOS login Keychain"
	case "dpapi":
		return " — protected by the Windows user-bound DPAPI"
	default:
		return ""
	}
}

func credentialsStoreInit(ctx *CommandContext, args []string, input io.Reader) error {
	format, err := storeFormatFlag("credentials store init", args)
	if err != nil {
		return err
	}
	passphrase, err := storePassphrase(input)
	if err != nil {
		return err
	}
	status, err := securestore.InitializeStore(passphrase)
	if err != nil {
		return err
	}
	if format == "json" {
		return json.NewEncoder(ctx.Stdout).Encode(status)
	}
	fmt.Fprintf(ctx.Stdout, "Encrypted credential store created at %s.\n\n", status.Path)
	writeStoreStatus(ctx, status)
	fmt.Fprintf(ctx.Stdout, "\nProvision a credential with `vrooli credentials provision --identity <id> --field <field>`.\n")
	return nil
}

func credentialsStoreUnlock(ctx *CommandContext, args []string, input io.Reader) error {
	if len(args) != 0 {
		return fmt.Errorf("credentials store unlock accepts no arguments")
	}
	passphrase, err := storePassphrase(input)
	if err != nil {
		return err
	}
	if passphrase == "" {
		return fmt.Errorf("credentials store unlock reads the passphrase from standard input; pipe it in")
	}
	status, err := securestore.UnlockStore(passphrase)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(ctx.Stdout,
		"Credential store unlocked with the %s wrap. Later commands in this login session will not prompt; run `vrooli credentials store lock` to end that.\n",
		status.ActiveWrap)
	return err
}

func credentialsStoreLock(ctx *CommandContext, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("credentials store lock accepts no arguments")
	}
	if err := securestore.LockStore(); err != nil {
		return err
	}
	_, err := fmt.Fprintln(ctx.Stdout, "Credential store locked. The next command that needs a value will ask for the passphrase again.")
	return err
}

func credentialsStoreRewrap(ctx *CommandContext, args []string, input io.Reader) error {
	if len(args) != 0 {
		return fmt.Errorf("credentials store rewrap accepts no arguments")
	}
	passphrase, err := storePassphrase(input)
	if err != nil {
		return err
	}
	wrap, err := securestore.RewrapStore(passphrase)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(ctx.Stdout,
		"Added a %s wrap protected by %s%s.\nNo stored value was re-encrypted: only the wrap changed.\n",
		wrap.Provider, wrap.KeyStore, keyStoreCaveat(wrap.KeyStore))
	return err
}

func credentialsStoreChangePassphrase(ctx *CommandContext, args []string, input io.Reader) error {
	if len(args) != 0 {
		return fmt.Errorf("credentials store change-passphrase accepts no arguments")
	}
	if err := refuseInteractiveStdin(input,
		`printf '%s\n%s\n' "$CURRENT" "$NEW" | vrooli credentials store change-passphrase`); err != nil {
		return err
	}
	contents, err := io.ReadAll(io.LimitReader(input, 128*1024))
	if err != nil {
		return fmt.Errorf("read credential store passphrases: %w", err)
	}
	current, next, found := strings.Cut(string(contents), "\n")
	if !found || strings.TrimSpace(current) == "" || strings.TrimSpace(next) == "" {
		return fmt.Errorf("credentials store change-passphrase reads current and new passphrases from stdin on separate lines")
	}
	if err := securestore.ChangePassphraseStore(current, next); err != nil {
		return err
	}
	_, err = fmt.Fprintln(ctx.Stdout, "Credential store passphrase changed. No stored value was re-encrypted.")
	return err
}
