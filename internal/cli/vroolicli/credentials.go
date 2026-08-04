package vroolicli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/credentialspec"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/resources/catalog"
	resourceenv "github.com/vrooli/vrooli/internal/resources/env"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
	credentialauthority "github.com/vrooli/vrooli/internal/secrets"
)

// runCredentialsCommand owns all local credential writes. Values are accepted
// exclusively through stdin so they do not enter argv, command metrics, shell
// history, or status output.
func (app *App) runCredentialsCommand(ctx *CommandContext, args []string) error {
	if len(args) == 0 || commandtree.WantsHelp(args) {
		fmt.Fprintln(ctx.Stdout, "Usage:\n  vrooli credentials doctor [--check-writes] [--format json]\n  vrooli credentials provision --identity <namespace/name> --field <field> < value\n  vrooli credentials status --identity <namespace/name> --field <field> [--format json]\n  vrooli credentials store <status|init|unlock|lock|rewrap|change-passphrase>\n  vrooli credentials recovery export --entry <identity>:<field> --output <bundle> < passphrase\n  vrooli credentials recovery verify --input <bundle> < passphrase\n  vrooli credentials recovery restore --input <bundle> < passphrase\n\nCredential values, store passphrases, and recovery passphrases are read only from standard input and never printed.\n`credentials store` manages the encrypted backend used on a host with no native credential store.")
		return nil
	}
	switch args[0] {
	case "doctor":
		return credentialsDoctor(ctx, args[1:])
	case "provision":
		return provisionCredential(ctx, args[1:], os.Stdin)
	case "status":
		return credentialStatus(ctx, args[1:])
	case "store":
		return credentialsStore(ctx, args[1:], os.Stdin)
	case "recovery":
		return recoveryCredentials(ctx, args[1:], os.Stdin)
	default:
		return fmt.Errorf("unknown credentials command %q", args[0])
	}
}

// refuseInteractiveStdin turns a silent hang into an instruction.
//
// Every credential command reads its secret from standard input, which is the
// right channel — a value in argv is visible in /proc, in a process listing,
// and in shell history. But a bare read from a terminal produces no prompt and
// no output, so an operator who forgets the pipe sees a command that has
// apparently frozen and has no way to tell that from real work. That has now
// cost real time more than once.
//
// Only an actual terminal is refused. A pipe, a file, and the bytes.Reader a
// test supplies all read normally, so this changes no scripted behaviour.
func refuseInteractiveStdin(input io.Reader, usage string) error {
	file, ok := input.(*os.File)
	if !ok {
		return nil
	}
	info, err := file.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return nil
	}
	return fmt.Errorf(
		"this command reads its secret from standard input and will not prompt; pipe it in:\n  %s", usage)
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
	if err := refuseInteractiveStdin(input,
		`printf '%s' "$VALUE" | vrooli credentials provision --identity <id> --field <field>`); err != nil {
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

type credentialEntries []string

// collectCredentialEntries is the manifest-backed inventory shared by doctor,
// list, and recovery export. The authority itself deliberately cannot
// enumerate identities; declarations are the source of the inventory.
//
// It walks scenarios as well as resources. That is not a convenience: this
// function is what `recovery export --all` selects from, so a declaration this
// inventory misses is a credential no bundle ever captures. Before scenarios
// could declare, tunnel-manager's Cloudflare token was exactly that.
func collectCredentialEntries(root string) ([]credentialEntry, error) {
	if strings.TrimSpace(root) == "" {
		return nil, nil
	}
	entries := []credentialEntry{}

	// Live managed instances are the second inventory source. Declarations are
	// the inventory for anything an author wrote down, but a managed instance's
	// ID is generated at runtime, so no manifest can name its unseal key — and
	// material no inventory names is material `recovery export --all` silently
	// omits. For Vault that omission is unrecoverable: without the unseal key
	// the instance stays sealed forever.
	vaultEntries := liveVaultUnsealKeyEntries()
	kopiaEntries := liveKopiaRepositoryEntries()

	// One verdict per store, read once. Availability was previously inferred
	// from whichever reads each resource happened to make, so a resource whose
	// single credential read cleanly reported "configured" in the same table
	// where 26 siblings reported "provider_unavailable" — two answers about one
	// store, which is precisely the confusion the failure taxonomy exists to
	// prevent. Authority.Availability memoizes, so this costs one probe.
	providerState := credentialauthority.ProviderAvailable
	if authority, authErr := credentialAuthority(); authErr != nil {
		providerState = credentialauthority.ProviderStateFor(authErr)
	} else if availErr := authority.Availability(); availErr != nil {
		providerState = credentialauthority.ProviderStateFor(availErr)
	}

	for _, vault := range vaultEntries {
		declaration := credentialspec.Declaration{Descriptors: []credentialspec.Descriptor{{
			LogicalID: vault.LogicalID,
			Field:     vault.Field,
			Label:     "Vault unseal key (instance " + vault.InstanceID + ")",
			Required:  true,
		}}}
		gaps, gapErr := resourceenv.ResolveScenarioCredentialGaps("vault", declaration)
		if gapErr != nil {
			continue
		}
		entries = append(entries, credentialEntriesFor("vault", declaration, gaps, providerState)...)
	}
	for _, kopia := range kopiaEntries {
		declaration := credentialspec.Declaration{Descriptors: []credentialspec.Descriptor{{
			LogicalID: kopia.LogicalID,
			Field:     kopia.Field,
			Label:     "Kopia repository passphrase (repository " + kopia.Repository + ")",
			Required:  true,
		}}}
		gaps, gapErr := resourceenv.ResolveScenarioCredentialGaps("kopia", declaration)
		if gapErr != nil {
			continue
		}
		entries = append(entries, credentialEntriesFor("kopia", declaration, gaps, providerState)...)
	}

	names, err := catalog.New(root).ManifestNames()
	if err != nil {
		return nil, fmt.Errorf("discover resource manifests: %w", err)
	}
	sort.Strings(names)
	for _, name := range names {
		resourceManifest, err := manifestpkg.Load(manifestpkg.DefaultPath(root, name))
		if err != nil {
			continue
		}
		if len(resourceManifest.Credentials.All()) == 0 {
			continue
		}
		gaps, err := resourceenv.ResolveCredentialGaps(resourceManifest)
		if err != nil {
			continue
		}
		entries = append(entries, credentialEntriesFor(resourceManifest.Name, resourceManifest.Credentials, gaps, providerState)...)
	}

	scenarios, err := scenario.Discover(root, scenario.SandboxEnvFromEnv())
	if err != nil {
		// A scenario tree that cannot be walked must not blank out the
		// resource inventory an operator is trying to read.
		return entries, nil
	}
	sort.Slice(scenarios, func(i, j int) bool { return scenarios[i].Slug < scenarios[j].Slug })
	for _, found := range scenarios {
		if len(found.Manifest.Credentials.All()) == 0 {
			continue
		}
		gaps, err := resourceenv.ResolveScenarioCredentialGaps(found.Slug, found.Manifest.Credentials)
		if err != nil {
			continue
		}
		entries = append(entries, credentialEntriesFor(found.Slug, found.Manifest.Credentials, gaps, providerState)...)
	}
	return entries, nil
}

// liveVaultUnsealKeyEntries is the runtime-instance inventory source. It is a
// variable because it reads this host's broker state directly, which a test
// must be able to replace — otherwise a credential test's outcome depends on
// whether the machine running it happens to have a managed Vault.
var liveVaultUnsealKeyEntries = resources.LiveVaultUnsealKeyEntries

// liveKopiaRepositoryEntries is replaceable so inventory tests can assert one
// row per repository without mutating host state.
var liveKopiaRepositoryEntries = resources.LiveKopiaRepositoryEntries

// credentialEntriesFor turns one declaration plus its gap report into inventory
// rows. Gaps are keyed by identity and field rather than by env, because a
// descriptor resolved directly by Vrooli-authored code has no env to key on and
// would otherwise all collide on the empty string.
func credentialEntriesFor(owner string, declaration credentialspec.Declaration, gaps resourceenv.CredentialResolution, providerState credentialauthority.ProviderState) []credentialEntry {
	gapByKey := make(map[string]resourceenv.MissingCredential, len(gaps.Missing))
	for _, gap := range gaps.Missing {
		gapByKey[gap.LogicalID+":"+gap.Field] = gap
	}
	out := make([]credentialEntry, 0, len(declaration.Descriptors))
	for _, descriptor := range declaration.All() {
		field := descriptor.ResolvedField()
		identity := strings.TrimSpace(descriptor.LogicalID)
		entry := credentialEntry{
			Resource: owner, Env: strings.TrimSpace(descriptor.Env),
			LogicalID: identity, Field: field,
			Label: strings.TrimSpace(descriptor.Label), Required: descriptor.Required,
			Configured: true, State: "configured",
		}
		if gap, missing := gapByKey[identity+":"+field]; missing {
			entry.Configured = false
			entry.State = string(gap.Reason)
			entry.Remediation = gap.Remediation
		} else if providerState != credentialauthority.ProviderAvailable {
			// The store is down, so nothing can be claimed configured — even a
			// value this resource's own read happened to return. Configured is
			// only meaningful when the provider answered, and a row asserting
			// otherwise beside 26 rows reporting an outage is worse than no row.
			entry.Configured = false
			entry.State = string(unavailableGapReason(providerState))
			entry.Remediation = unavailableRemediation(providerState)
		}
		out = append(out, entry)
	}
	return out
}

// unavailableGapReason and unavailableRemediation mirror what the resolver
// reports for the same condition, so an operator reading the inventory and an
// operator reading a scenario start see one vocabulary rather than two.
func unavailableGapReason(state credentialauthority.ProviderState) resourceenv.CredentialGapReason {
	if state == credentialauthority.ProviderAbsent {
		return resourceenv.GapProviderAbsent
	}
	return resourceenv.GapProviderUnavailable
}

func unavailableRemediation(state credentialauthority.ProviderState) string {
	if state == credentialauthority.ProviderAbsent {
		return "this host has no credential backend; run `vrooli credentials doctor` to see what to install"
	}
	return "the credential store is unreachable; run `vrooli credentials doctor` for the host diagnosis"
}

func (entries *credentialEntries) String() string { return strings.Join(*entries, ",") }
func (entries *credentialEntries) Set(value string) error {
	*entries = append(*entries, value)
	return nil
}

func recoveryCredentials(ctx *CommandContext, args []string, input io.Reader) error {
	if len(args) == 0 || commandtree.WantsHelp(args) {
		fmt.Fprintln(ctx.Stdout, "Usage:\n  vrooli credentials recovery export --entry <identity>:<field> --output <bundle> [--format json] < passphrase\n  vrooli credentials recovery export --all --output <bundle> [--format json] < passphrase\n  vrooli credentials recovery verify --input <bundle> [--format json] < passphrase\n  vrooli credentials recovery restore --input <bundle> < passphrase\n\nverify proves a bundle opens and lists what it would restore, without writing anything or printing a value.\n--all captures every configured credential declared by a resource or scenario manifest, plus the unseal key of every live managed Vault instance. A Vault unseal key is irreplaceable: without it the instance stays sealed and its contents are gone. Root tokens are deliberately excluded — Vault regenerates one from the unseal key, so a bundle carrying both would widen the blast radius for nothing.")
		return nil
	}
	switch args[0] {
	case "export":
		return exportCredentialRecovery(ctx, args[1:], input)
	case "verify":
		return verifyCredentialRecovery(ctx, args[1:], input)
	case "restore":
		return restoreCredentialRecovery(ctx, args[1:], input)
	default:
		return fmt.Errorf("unknown credentials recovery command %q", args[0])
	}
}

// recoveryStateDir resolves where the export receipt lives, through the repo
// contract rather than an assembled path.
func recoveryStateDir() (string, error) {
	return config.VrooliPath(repocontract.HomeKeyState)
}

func recoveryPassphrase(input io.Reader) (string, error) {
	if err := refuseInteractiveStdin(input,
		`printf '%s' "$PASSPHRASE" | vrooli credentials recovery <export|verify|restore> ...`); err != nil {
		return "", err
	}
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
	all := false
	format := "text"
	fs.Var(&entries, "entry", "credential entry in identity:field form; repeat for each entry")
	fs.StringVar(&output, "output", "", "new encrypted recovery bundle path")
	fs.BoolVar(&all, "all", false, "include every configured credential declared by a resource manifest")
	fs.StringVar(&format, "format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 || strings.TrimSpace(output) == "" {
		return fmt.Errorf("recovery export requires --output")
	}
	if all && len(entries) > 0 {
		return fmt.Errorf("recovery export accepts either --all or --entry, not both")
	}
	if !all && len(entries) == 0 {
		return fmt.Errorf("recovery export requires at least one --entry or --all")
	}
	format = strings.TrimSpace(format)
	if format != "text" && format != "json" {
		return fmt.Errorf("credentials recovery export format must be text or json")
	}
	selected := make([]credentialauthority.RecoveryEntry, 0, len(entries))
	skipped := []string{}
	if all {
		declared, err := collectCredentialEntries(ctx.Root)
		if err != nil {
			return err
		}
		// The inventory lists one row per declaration, and several resources
		// deliberately share a credential — three declare vrooli/openrouter:api-key.
		// Deduplicating by store key keeps the bundle, its counts, and the
		// skipped list describing credentials rather than declarations; without
		// it a bundle of eight secrets reports ten and names one of them three
		// times, which reads like a defect an operator then has to rule out.
		seen := map[string]bool{}
		for _, entry := range declared {
			identity := strings.TrimSpace(entry.LogicalID)
			label := identity + ":" + entry.Field
			if seen[label] {
				continue
			}
			seen[label] = true
			if !entry.Configured {
				skipped = append(skipped, label)
				continue
			}
			parsed, err := credentialauthority.ParseIdentity(identity)
			if err != nil {
				return err
			}
			selected = append(selected, credentialauthority.RecoveryEntry{Identity: parsed, Field: entry.Field})
		}
		if len(selected) == 0 {
			return fmt.Errorf("recovery export --all found no configured credentials; skipped %s", strings.Join(skipped, ", "))
		}
	}
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
	// Record the export so `doctor` can report a host that has never made a
	// bundle. A failure here does not fail the export: the bundle on disk is
	// what matters, and refusing to acknowledge a good backup because a note
	// could not be written would be the wrong trade.
	if stateDir, dirErr := recoveryStateDir(); dirErr == nil {
		_ = credentialauthority.WriteRecoveryReceipt(stateDir, output, selected, time.Now())
	}

	if format == "json" {
		return json.NewEncoder(ctx.Stdout).Encode(struct {
			Written int      `json:"written"`
			Skipped []string `json:"skipped"`
		}{Written: len(selected), Skipped: skipped})
	}
	if _, err = fmt.Fprintf(ctx.Stdout, "Encrypted recovery bundle created for %d credential entries.\n", len(selected)); err != nil {
		return err
	}
	if all {
		fmt.Fprintf(ctx.Stdout, "Skipped %d unconfigured entries", len(skipped))
		if len(skipped) > 0 {
			fmt.Fprintf(ctx.Stdout, ": %s", strings.Join(skipped, ", "))
		}
		fmt.Fprintln(ctx.Stdout, ".")
	}
	return nil
}

// verifyCredentialRecovery proves a bundle opens and reports what it holds.
//
// It exists because "I ran the export command and it did not error" is not
// evidence that a bundle can be restored — a mistyped passphrase produces a
// perfectly valid file that nothing will ever open, and the operator finds out
// only when the original is gone. Verification needs no store and writes
// nothing, so it is safe to run anywhere, including on the machine that will
// hold the backup rather than the one that made it.
func verifyCredentialRecovery(ctx *CommandContext, args []string, input io.Reader) error {
	fs := flag.NewFlagSet("credentials recovery verify", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path, format := "", "text"
	fs.StringVar(&path, "input", "", "encrypted recovery bundle path")
	fs.StringVar(&format, "format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 || strings.TrimSpace(path) == "" {
		return fmt.Errorf("recovery verify requires --input")
	}
	format = strings.TrimSpace(format)
	if format != "text" && format != "json" {
		return fmt.Errorf("credentials recovery verify format must be text or json")
	}
	passphrase, err := recoveryPassphrase(input)
	if err != nil {
		return err
	}
	bundle, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("read recovery bundle: %w", err)
	}
	manifest, err := credentialauthority.InspectRecovery(bundle, passphrase)
	if err != nil {
		return err
	}
	if format == "json" {
		return json.NewEncoder(ctx.Stdout).Encode(manifest)
	}
	if _, err := fmt.Fprintf(ctx.Stdout,
		"Recovery bundle opens. It holds %d credential(s) and would restore:\n", len(manifest.Entries)); err != nil {
		return err
	}
	for _, entry := range manifest.Entries {
		if _, err := fmt.Fprintf(ctx.Stdout, "  %s:%s\n", entry.Identity, entry.Field); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(ctx.Stdout, "\nNo value was printed and nothing was written. Keep this bundle and its passphrase apart, and off this machine.")
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
