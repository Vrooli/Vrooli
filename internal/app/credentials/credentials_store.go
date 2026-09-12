package credentials

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/credentialauthority"
	"github.com/vrooli/vrooli/internal/securestore"
	kopiaregistry "github.com/vrooli/vrooli/packages/kopiaregistry-go"
)

type credentialStoreEntriesReport struct {
	Basis   string                 `json:"basis"`
	Entries []securestore.EntryRef `json:"entries"`
}

type credentialStoreDeleteEntryReport struct {
	Service string `json:"service"`
	Key     string `json:"key"`
	Deleted bool   `json:"deleted"`
}

const (
	credentialsStoreS3AccessKeyId     = "s3-access-key-id"
	credentialsStoreS3SecretAccessKey = "s3-secret-access-key"
	credentialsStoreS3SessionToken    = "s3-session-token"
)

const (
	credentialsStoreParameterA = 1024
)

const (
	credentialsStoreParameterB = 128
	credentialsStoreParameterC = 64
)

// The `credentials store` surface manages the encrypted credential store — the
// backend for a host with no native one. Native stores are managed by the
// operating system and have nothing here.
//
// Every passphrase arrives on standard input, never in an argument, for the
// same reason credential values do: an argument is visible in /proc, in a
// process listing, in shell history, and in command metrics.

//nolint:gocyclo // store-copy execution handles source discovery, destination policy, and verification outcomes.
func (app *Service) StoreCopy(ctx context.Context, out io.Writer, opts StoreCopyOptions) error {
	sink := strings.TrimSpace(opts.Sink)
	if sink == "" {
		sink = strings.TrimSpace(os.Getenv("VROOLI_CREDENTIAL_COPY_SINK"))
	}
	objectStoreCredentialID := strings.TrimSpace(opts.ObjectStoreCredentialID)
	if objectStoreCredentialID == "" {
		objectStoreCredentialID = strings.TrimSpace(os.Getenv("VROOLI_OBJECT_STORE_CREDENTIAL_IDENTITY"))
	}
	objectStoreRegion, objectStoreEndpoint := strings.TrimSpace(opts.ObjectStoreRegion), strings.TrimSpace(opts.ObjectStoreEndpoint)
	if objectStoreRegion == "" {
		objectStoreRegion = strings.TrimSpace(os.Getenv("VROOLI_OBJECT_STORE_REGION"))
	}
	if objectStoreEndpoint == "" {
		objectStoreEndpoint = strings.TrimSpace(os.Getenv("VROOLI_OBJECT_STORE_ENDPOINT"))
	}
	objectStoreAccessKeyField, objectStoreSecretKeyField, objectStoreSessionField := opts.ObjectStoreAccessKeyField, opts.ObjectStoreSecretKeyField, opts.ObjectStoreSessionField
	if objectStoreAccessKeyField == "" {
		objectStoreAccessKeyField = credentialsStoreS3AccessKeyId
	}
	if objectStoreSecretKeyField == "" {
		objectStoreSecretKeyField = credentialsStoreS3SecretAccessKey
	}
	configured, format := opts.Configured, strings.TrimSpace(opts.Format)
	if format == "" {
		format = string(cliout.FormatHuman)
	}
	if configured {
		config, err := readCredentialCopyConfig()
		if err != nil {
			return err
		}
		if !config.Enabled {
			return fmt.Errorf("encrypted credential-store copy is not configured; run `vrooli credentials store copy configure --sink <directory>`")
		}
		sink = config.Sink
		objectStoreCredentialID = config.ObjectStoreCredentialID
		objectStoreRegion = config.ObjectStoreRegion
		objectStoreEndpoint = config.ObjectStoreEndpoint
		objectStoreAccessKeyField = config.ObjectStoreAccessKeyField
		objectStoreSecretKeyField = config.ObjectStoreSecretKeyField
		objectStoreSessionField = config.ObjectStoreSessionField
	}
	if sink == "" {
		return fmt.Errorf("credentials store copy requires --sink <directory>, --configured, or VROOLI_CREDENTIAL_COPY_SINK")
	}
	if format != string(cliout.FormatHuman) && format != string(cliout.FormatJSON) {
		return fmt.Errorf("credentials store copy format must be text or json")
	}
	status, err := securestore.DescribeStore()
	if err != nil {
		return err
	}
	if !status.Initialized {
		return fmt.Errorf("credential store is not initialized")
	}
	registry := kopiaregistry.New(kopiaregistry.RegistryPath())
	entries, err := registry.Load()
	if err != nil {
		return err
	}
	repositoryPaths := make([]string, 0, len(entries))
	repositorySinks := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Backend == kopiaregistry.BackendFilesystem && strings.TrimSpace(entry.Path) != "" {
			repositoryPaths = append(repositoryPaths, entry.Path)
		}
		if entry.Backend == kopiaregistry.BackendS3 && strings.TrimSpace(entry.Bucket) != "" {
			repositorySink := "s3://" + strings.TrimSpace(entry.Bucket)
			if strings.TrimSpace(entry.Prefix) != "" {
				repositorySink += "/" + strings.Trim(entry.Prefix, "/")
			}
			repositorySinks = append(repositorySinks, repositorySink)
		}
	}
	receiptPath, err := config.VrooliPath(repocontract.HomeKeyState, "credential-store-copy.json")
	if err != nil {
		return err
	}
	var copyStatus securestore.CopyStatus
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(sink)), "s3://") {
		credentials, credentialErr := resolveObjectStoreCredentials(objectStoreCredentialID, objectStoreAccessKeyField, objectStoreSecretKeyField, objectStoreSessionField)
		if credentialErr != nil {
			return credentialErr
		}
		copyStatus, err = securestore.CopyStoreS3(status.Path, sink, receiptPath, securestore.S3CopyOptions{
			Region: objectStoreRegion, Endpoint: objectStoreEndpoint, Credentials: credentials, RepositorySinks: repositorySinks,
		})
	} else {
		home, homeErr := config.VrooliHome()
		if homeErr != nil {
			return homeErr
		}
		copyStatus, err = securestore.CopyStoreWithPolicy(status.Path, sink, receiptPath, securestore.CopyPolicy{
			RepositoryPaths: repositoryPaths, ProtectedRoots: []string{home}, RequireIndependentDevice: true,
		})
	}
	if err != nil {
		return err
	}
	if format == string(cliout.FormatJSON) {
		return cliout.WriteJSONValue(out, copyStatus)
	}
	_, err = fmt.Fprintf(out, "Encrypted credential store copied to %s (generation %s).\n", copyStatus.Path, copyStatus.Generation)
	return err
}

func credentialCopyConfigPath() (string, error) {
	return config.VrooliPath(repocontract.HomeKeyConfig, "credential-store-copy.json")
}

func readCredentialCopyConfig() (securestore.CopyConfig, error) {
	path, err := credentialCopyConfigPath()
	if err != nil {
		return securestore.CopyConfig{}, err
	}
	return securestore.ReadCopyConfig(path)
}

//nolint:gocyclo // store-copy configuration preserves backend, scope, overwrite, and validation decisions.
func (app *Service) StoreCopyConfigure(ctx context.Context, out io.Writer, opts StoreCopyConfigureOptions) error {
	sink := strings.TrimSpace(opts.Sink)
	interval := opts.Interval
	if interval == 0 {
		interval = securestore.DefaultCopyInterval
	}
	objectStoreCredentialID := strings.TrimSpace(opts.ObjectStoreCredentialID)
	objectStoreRegion := strings.TrimSpace(opts.ObjectStoreRegion)
	objectStoreEndpoint := strings.TrimSpace(opts.ObjectStoreEndpoint)
	objectStoreAccessKeyField := strings.TrimSpace(opts.ObjectStoreAccessKeyField)
	if objectStoreAccessKeyField == "" {
		objectStoreAccessKeyField = credentialsStoreS3AccessKeyId
	}
	objectStoreSecretKeyField := strings.TrimSpace(opts.ObjectStoreSecretKeyField)
	if objectStoreSecretKeyField == "" {
		objectStoreSecretKeyField = credentialsStoreS3SecretAccessKey
	}
	objectStoreSessionField := strings.TrimSpace(opts.ObjectStoreSessionField)
	enabled := opts.Enabled
	format := strings.TrimSpace(opts.Format)
	if format == "" {
		format = string(cliout.FormatHuman)
	}
	if !enabled {
		// Disabling an existing schedule should be a one-flag operation. Keep
		// the last non-secret configuration in place so a later enable can
		// reuse it, while allowing an explicit flag to override any field the
		// operator wants to retain differently.
		existing, existingErr := readCredentialCopyConfig()
		if existingErr != nil {
			return existingErr
		}
		if strings.TrimSpace(sink) == "" {
			sink = existing.Sink
		}
		if interval == securestore.DefaultCopyInterval && existing.Interval > 0 {
			interval = existing.Interval
		}
		if objectStoreCredentialID == "" {
			objectStoreCredentialID = existing.ObjectStoreCredentialID
		}
		if objectStoreRegion == "" {
			objectStoreRegion = existing.ObjectStoreRegion
		}
		if objectStoreEndpoint == "" {
			objectStoreEndpoint = existing.ObjectStoreEndpoint
		}
		if objectStoreAccessKeyField == credentialsStoreS3AccessKeyId && existing.ObjectStoreAccessKeyField != "" {
			objectStoreAccessKeyField = existing.ObjectStoreAccessKeyField
		}
		if objectStoreSecretKeyField == credentialsStoreS3SecretAccessKey && existing.ObjectStoreSecretKeyField != "" {
			objectStoreSecretKeyField = existing.ObjectStoreSecretKeyField
		}
		if objectStoreSessionField == credentialsStoreS3SessionToken && existing.ObjectStoreSessionField != "" {
			objectStoreSessionField = existing.ObjectStoreSessionField
		}
	}
	if strings.TrimSpace(sink) == "" {
		return fmt.Errorf("credentials store copy configure requires --sink <directory|s3://bucket/prefix> when enabling")
	}
	if format != string(cliout.FormatHuman) && format != string(cliout.FormatJSON) {
		return fmt.Errorf("credentials store copy configure format must be text or json")
	}
	config := securestore.CopyConfig{
		Enabled: enabled, Sink: strings.TrimSpace(sink), Interval: interval,
		ObjectStoreCredentialID: strings.TrimSpace(objectStoreCredentialID), ObjectStoreRegion: strings.TrimSpace(objectStoreRegion),
		ObjectStoreEndpoint: strings.TrimSpace(objectStoreEndpoint), ObjectStoreAccessKeyField: strings.TrimSpace(objectStoreAccessKeyField),
		ObjectStoreSecretKeyField: strings.TrimSpace(objectStoreSecretKeyField), ObjectStoreSessionField: strings.TrimSpace(objectStoreSessionField),
	}
	path, err := credentialCopyConfigPath()
	if err != nil {
		return err
	}
	if err := securestore.WriteCopyConfig(path, config); err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve Vrooli executable for credential-store copy schedule: %w", err)
	}
	if err := installCredentialCopySchedule(executable, config.Interval, config.Enabled); err != nil {
		return err
	}
	if format == string(cliout.FormatJSON) {
		return cliout.WriteJSONValue(out, config)
	}
	_, err = fmt.Fprintf(out, "Encrypted credential-store copy configured at %s; refresh interval %s.\n", config.Sink, config.Interval)
	return err
}

func resolveObjectStoreCredentials(identityName, accessField, secretField, sessionField string) (securestore.ObjectStoreCredentials, error) {
	identity, err := credentialauthority.ParseIdentity(identityName)
	if err != nil {
		return securestore.ObjectStoreCredentials{}, fmt.Errorf("object-store credential identity: %w", err)
	}
	if accessField == "" {
		accessField = credentialsStoreS3AccessKeyId
	}
	if secretField == "" {
		secretField = credentialsStoreS3SecretAccessKey
	}
	authority, err := credentialauthority.DefaultAuthority()
	if err != nil {
		return securestore.ObjectStoreCredentials{}, fmt.Errorf("object-store credential authority: %w", err)
	}
	accessKey, err := authority.Resolve(identity, accessField)
	if err != nil {
		return securestore.ObjectStoreCredentials{}, fmt.Errorf("resolve object-store access credential %s:%s: %w", identity, accessField, err)
	}
	secretKey, err := authority.Resolve(identity, secretField)
	if err != nil {
		return securestore.ObjectStoreCredentials{}, fmt.Errorf("resolve object-store secret credential %s:%s: %w", identity, secretField, err)
	}
	var sessionToken string
	if strings.TrimSpace(sessionField) != "" {
		sessionToken, err = authority.Resolve(identity, sessionField)
		if err != nil && !errors.Is(err, credentialauthority.ErrUnconfigured) {
			return securestore.ObjectStoreCredentials{}, fmt.Errorf("resolve object-store session credential %s:%s: %w", identity, sessionField, err)
		}
	}
	return securestore.ObjectStoreCredentials{AccessKey: accessKey, SecretKey: secretKey, SessionToken: sessionToken}, nil
}

// credentialsStoreCopyScheduled is the timer/service entrypoint. It performs
// one configured refresh and exits, so an OS scheduler can invoke it without
// ever placing a passphrase or credential value in a process argument.
func (app *Service) StoreCopyScheduled(ctx context.Context, out io.Writer, format string) error {
	if format == "" {
		format = string(cliout.FormatHuman)
	}
	var err error
	config, err := readCredentialCopyConfig()
	if err != nil {
		return err
	}
	if !config.Enabled {
		return fmt.Errorf("encrypted credential-store copy is not enabled")
	}
	return app.StoreCopy(ctx, out, StoreCopyOptions{Sink: config.Sink, Format: format, ObjectStoreCredentialID: config.ObjectStoreCredentialID, ObjectStoreRegion: config.ObjectStoreRegion, ObjectStoreEndpoint: config.ObjectStoreEndpoint, ObjectStoreAccessKeyField: config.ObjectStoreAccessKeyField, ObjectStoreSecretKeyField: config.ObjectStoreSecretKeyField, ObjectStoreSessionField: config.ObjectStoreSessionField})
}

func (app *Service) StoreReselect(ctx context.Context, root string, out io.Writer, format string) error {
	if format == "" {
		format = string(cliout.FormatHuman)
	}
	var err error
	entries, err := credentialMigrationEntries(root)
	if err != nil {
		return err
	}
	receipt, err := securestore.ReselectBackend(entries)
	if format == string(cliout.FormatJSON) {
		encodeErr := cliout.WriteJSONValue(out, receipt)
		if err != nil {
			return err
		}
		return encodeErr
	}
	if err != nil {
		return err
	}
	if receipt.From == receipt.To {
		_, err = fmt.Fprintf(out, "Credential backend %s is already selected; no migration was needed.\n", receipt.To)
		return err
	}
	_, err = fmt.Fprintf(out, "Credential backend reselected from %s to %s; verified %d credential(s).\n",
		receipt.From, receipt.To, len(receipt.Verified))
	return err
}

func (app *Service) StoreRetire(ctx context.Context, out io.Writer, backend string) error {
	backend = strings.TrimSpace(backend)
	if backend == "" {
		return fmt.Errorf("credentials store retire requires --backend encrypted-file")
	}
	if err := securestore.RetireEmptyBackend(backend); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "Retired the empty %s credential backend.\n", backend)
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
func storePassphrase(input io.Reader, prompt io.Writer) (string, error) {
	if input == nil {
		return "", nil
	}
	if file, ok := input.(*os.File); ok {
		if info, err := file.Stat(); err == nil && info.Mode()&os.ModeCharDevice != 0 {
			return readInteractivePassphrase(file, prompt)
		}
	}
	if err := refuseInteractiveStdin(input,
		`provide the passphrase on standard input to vrooli credentials store <init|unlock|rewrap>`); err != nil {
		return "", err
	}
	value, err := io.ReadAll(io.LimitReader(input, credentialsStoreParameterC*credentialsStoreParameterA))
	if err != nil {
		return "", fmt.Errorf("read credential store passphrase: %w", err)
	}
	return strings.TrimSpace(string(value)), nil
}

// optionalStorePassphrase reads a passphrase when one was piped in, and reports
// none when standard input is a terminal or the null device.
//
// It is the right shape for rewrap and the wrong shape for init and unlock. A
// store that is already open — the normal case, because setup or onboarding has
// just unlocked it — needs no passphrase at all to gain a wrap, so demanding
// one would refuse the convergence on exactly the hosts that are ready for it.
// init and unlock genuinely cannot proceed without the secret, so they keep the
// strict guard that tells an operator how to supply it.
func optionalStorePassphrase(input io.Reader) (string, error) {
	if input == nil {
		return "", nil
	}
	if file, ok := input.(*os.File); ok {
		info, err := file.Stat()
		if err != nil || info.Mode()&os.ModeCharDevice != 0 {
			return "", nil
		}
	}
	value, err := io.ReadAll(io.LimitReader(input, credentialsStoreParameterC*credentialsStoreParameterA))
	if err != nil {
		return "", fmt.Errorf("read credential store passphrase: %w", err)
	}
	return strings.TrimSpace(string(value)), nil
}

func storeFormat(format string) (string, error) {
	format = strings.TrimSpace(format)
	if format == "" {
		format = string(cliout.FormatHuman)
	}
	if format != string(cliout.FormatHuman) && format != string(cliout.FormatJSON) {
		return "", fmt.Errorf("store format must be text or json")
	}
	return format, nil
}

func (app *Service) StoreStatus(ctx context.Context, out io.Writer, format string) error {
	format, err := storeFormat(format)
	if err != nil {
		return err
	}
	status, err := securestore.DescribeStore()
	if err != nil {
		return err
	}
	if format == string(cliout.FormatJSON) {
		return cliout.WriteJSONValue(out, status)
	}
	writeStoreStatus(out, status)
	return nil
}

func (app *Service) StoreEntries(ctx context.Context, out io.Writer, format string) error {
	format, err := storeFormat(format)
	if err != nil {
		return err
	}
	entries, err := securestore.ListEntryRefs()
	if err != nil {
		return err
	}
	if format == string(cliout.FormatJSON) {
		return cliout.WriteJSONValue(out, credentialStoreEntriesReport{Basis: "sealed_store_metadata; values_not_read", Entries: entries})
	}
	fmt.Fprintf(out, "Encrypted credential store entries (%d; basis=sealed_store_metadata; values_not_read)\n", len(entries))
	for _, entry := range entries {
		fmt.Fprintf(out, "  %s | %s\n", entry.Service, entry.Key)
	}
	return nil
}

func (app *Service) StoreDeleteEntry(ctx context.Context, out io.Writer, service, key, format string, yes bool) error {
	service, key = strings.TrimSpace(service), strings.TrimSpace(key)
	format, err := storeFormat(format)
	if err != nil {
		return err
	}
	if service == "" || key == "" {
		return fmt.Errorf("credentials store entries delete requires --service and --key")
	}
	if !yes {
		return fmt.Errorf("refusing to delete store entry without explicit --yes confirmation")
	}
	deleted, err := securestore.DeleteEntryRef(service, key)
	if err != nil {
		return err
	}
	result := credentialStoreDeleteEntryReport{Service: service, Key: key, Deleted: deleted}
	if format == string(cliout.FormatJSON) {
		return cliout.WriteJSONValue(out, result)
	}
	if deleted {
		fmt.Fprintf(out, "Deleted encrypted credential-store entry %s | %s; no value was printed.\n", result.Service, result.Key)
	} else {
		fmt.Fprintf(out, "Encrypted credential-store entry %s | %s was already absent; no value was printed.\n", result.Service, result.Key)
	}
	return nil
}

func writeStoreStatus(out io.Writer, status securestore.StoreStatus) {
	fmt.Fprintf(out, "Encrypted credential store\n")
	fmt.Fprintf(out, "  Path:        %s\n", status.Path)
	if !status.Initialized {
		fmt.Fprintf(out, "  State:       not initialized\n")
		if status.HostBoundBlocked != "" {
			fmt.Fprintf(out, "\nRun `vrooli credentials store init` and enter the passphrase at its secure prompt.\n")
			fmt.Fprintf(out, "The unattended host-bound wrap will not open on this host as it stands:\n  %s\n", status.HostBoundBlocked)
			return
		}
		fmt.Fprintf(out, "\nRun `vrooli credentials store init` to create it. A host with a reachable TPM\nneeds no passphrase; otherwise enter one at its secure prompt.\n")
		return
	}
	fmt.Fprintf(out, "  State:       initialized, %d credential(s)\n", status.Entries)
	fmt.Fprintf(out, "  Authority:   %s\n",
		map[bool]string{true: "yes — this host's credentials live here", false: "no — a native store is the authority on this host"}[status.Active])
	fmt.Fprintf(out, "  Unlocked:    %t\n", status.Unlocked)
	if status.ActiveWrap != "" {
		fmt.Fprintf(out, "  Opened by:   %s (%s)\n", status.ActiveWrap, status.ActiveKeyStore)
	}
	switch {
	case status.ActiveWrap == "host-bound":
		fmt.Fprintf(out, "  Unlock kept: not needed — the host-bound wrap opens this store with no human action\n")
	case status.ActiveWrap == "native-wrap":
		fmt.Fprintf(out, "  Unlock kept: not needed — the native platform wrap opens this store with no human action\n")
	case status.UnlockCache != "":
		fmt.Fprintf(out, "  Unlock kept: %s (session tmpfs; gone at logout)\n", status.UnlockCache)
	default:
		fmt.Fprintf(out, "  Unlock kept: nowhere — this host has no session-scoped memory, so an unlock lasts one command\n")
	}
	fmt.Fprintf(out, "  Key wraps:\n")
	for _, wrap := range status.Wraps {
		fmt.Fprintf(out, "    %-12s %s%s\n", wrap.Provider, wrap.KeyStore, keyStoreCaveat(wrap.KeyStore))
	}
	// Whether a reboot needs a human is the fact an operator most needs from
	// this command, and it is not readable from the wrap list: a wrap that has
	// stopped opening is still listed. So the verified answer is stated
	// outright.
	if status.Unattended.Enabled {
		fmt.Fprintf(out, "  Unattended:  yes — the %s wrap (%s) opens this store after a reboot with no passphrase\n",
			status.Unattended.Provider, status.Unattended.KeyStore)
		return
	}
	fmt.Fprintf(out, "  Unattended:  no — this store needs a passphrase after every reboot\n")
	if status.Unattended.Blocked != "" {
		fmt.Fprintf(out, "\nWhy:\n  %s\n", status.Unattended.Blocked)
	}
	fmt.Fprintf(out, "\nRun `vrooli setup`; it grants what the host needs and adds the wrap in the same\nrun. It keeps the same data key, so no stored value is re-encrypted.\n")
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

func (app *Service) StoreInit(ctx context.Context, out, errOut io.Writer, format string, input io.Reader) error {
	format, err := storeFormat(format)
	if err != nil {
		return err
	}
	passphrase, err := storePassphrase(input, errOut)
	if err != nil {
		return err
	}
	status, err := securestore.InitializeStore(passphrase)
	if err != nil {
		return err
	}
	if format == string(cliout.FormatJSON) {
		return cliout.WriteJSONValue(out, status)
	}
	fmt.Fprintf(out, "Encrypted credential store created at %s.\n\n", status.Path)
	writeStoreStatus(out, status)
	fmt.Fprintf(out, "\nProvision a credential with `vrooli credentials provision --identity <id> --field <field>`.\n")
	return nil
}

// convergeUnattended is what makes "supply the passphrase once" true. Any
// command that leaves the store open ends by making sure this host can open it
// again without one, and says which. A blocked host is reported, never failed:
// the store is usable either way, and the only difference is whether a reboot
// needs a human.
func convergeUnattended(out io.Writer, passphrase string) {
	status, err := securestore.EnsureUnattendedWrap(passphrase)
	if err != nil {
		fmt.Fprintf(out, "Unattended access could not be evaluated: %v\n", err)
		return
	}
	if status.Added || status.Repaired || !status.Enabled {
		writeUnattendedStatus(out, status)
	}
}

func (app *Service) StoreUnlock(ctx context.Context, out, errOut io.Writer, input io.Reader) error {
	passphrase, err := storePassphrase(input, errOut)
	if err != nil {
		return err
	}
	if passphrase == "" {
		return fmt.Errorf("credentials store unlock requires a passphrase; run it in a terminal to enter one securely")
	}
	status, err := securestore.UnlockStore(passphrase)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out,
		"Credential store unlocked with the %s wrap. Later commands in this login session will not prompt; run `vrooli credentials store lock` to end that.\n",
		status.ActiveWrap); err != nil {
		return err
	}
	convergeUnattended(out, passphrase)
	return nil
}

func (app *Service) StoreLock(ctx context.Context, out io.Writer) error {
	if err := securestore.LockStore(); err != nil {
		return err
	}
	_, err := fmt.Fprintln(out, "Credential store locked. The next command that needs a value will ask for the passphrase again.")
	return err
}

func (app *Service) StoreRewrap(ctx context.Context, out, errOut io.Writer, format string, input io.Reader) error {
	format, err := storeFormat(format)
	if err != nil {
		return err
	}
	passphrase, err := optionalStorePassphrase(input)
	if err != nil {
		return err
	}
	status, err := securestore.EnsureUnattendedWrap(passphrase)
	if err != nil {
		return err
	}
	// The status is printed before the command decides its exit code, so a
	// caller that reads the JSON gets the reason a host is still attended
	// rather than only a non-zero exit. Setup depends on that: it reports the
	// blocked reason to the operator instead of failing the whole run over a
	// host that simply has no TPM.
	if format == string(cliout.FormatJSON) {
		if encodeErr := cliout.WriteJSONValue(out, status); encodeErr != nil {
			return encodeErr
		}
	} else {
		writeUnattendedStatus(out, status)
	}
	if !status.Enabled {
		return fmt.Errorf("no unattended key wrap can protect this store: %s", status.Blocked)
	}
	return nil
}

// writeUnattendedStatus is the one rendering of the unattended answer, shared by
// `store status`, `store rewrap`, and the lines setup prints.
func writeUnattendedStatus(out io.Writer, status securestore.UnattendedStatus) {
	if !status.Enabled {
		fmt.Fprintf(out, "This store still needs a passphrase after every reboot.\n  %s\n", status.Blocked)
		return
	}
	switch {
	case status.Added:
		fmt.Fprintf(out, "Added a %s wrap protected by %s%s.\nNo stored value was re-encrypted: only the wrap changed.\n",
			status.Provider, status.KeyStore, keyStoreCaveat(status.KeyStore))
	case status.Repaired:
		fmt.Fprintf(out, "Replaced the %s wrap, which had stopped opening, with a working one protected by %s%s.\nNo stored value was re-encrypted: only the wrap changed.\n",
			status.Provider, status.KeyStore, keyStoreCaveat(status.KeyStore))
	default:
		fmt.Fprintf(out, "This store already opens with no passphrase, through its %s wrap protected by %s%s.\n",
			status.Provider, status.KeyStore, keyStoreCaveat(status.KeyStore))
	}
}

func (app *Service) StoreChangePassphrase(ctx context.Context, out, errOut io.Writer, input io.Reader) error {
	if file, ok := input.(*os.File); ok {
		if info, err := file.Stat(); err == nil && info.Mode()&os.ModeCharDevice != 0 {
			current, err := readInteractivePassphraseWithLabel(file, errOut, "Current credential store passphrase: ")
			if err != nil {
				return err
			}
			next, err := readInteractivePassphraseWithLabel(file, errOut, "New credential store passphrase: ")
			if err != nil {
				return err
			}
			if strings.TrimSpace(current) == "" || strings.TrimSpace(next) == "" {
				return fmt.Errorf("both credential store passphrases are required")
			}
			if err := securestore.ChangePassphraseStore(current, next); err != nil {
				return err
			}
			_, err = fmt.Fprintln(out, "Credential store passphrase changed. No stored value was re-encrypted.")
			return err
		}
	}
	if err := refuseInteractiveStdin(input,
		`provide the current and new passphrases on standard input to vrooli credentials store change-passphrase`); err != nil {
		return err
	}
	contents, err := io.ReadAll(io.LimitReader(input, credentialsStoreParameterB*credentialsStoreParameterA))
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
	_, err = fmt.Fprintln(out, "Credential store passphrase changed. No stored value was re-encrypted.")
	return err
}
