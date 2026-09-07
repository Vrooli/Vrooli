package credentials

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/nodereach"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/credentialauthority"
	"github.com/vrooli/vrooli/internal/credentialinventory"
	"github.com/vrooli/vrooli/internal/resources"
	grantv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/credentialgrant"
	grantv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/credentialgrant/credentialgrant_v1connect"
)

const (
	credentialsParameterA = 1024
)

const (
	credentialsParameterB = 64
)

type credentialListReport struct {
	Basis            string            `json:"inventory_basis"`
	Managed          bool              `json:"managed_instances_included"`
	CredentialCount  int               `json:"credential_count"`
	DeclarationSites int               `json:"declaration_site_count"`
	Uncovered        []string          `json:"uncovered"`
	RequiredAbsent   []string          `json:"required_absent"`
	Credentials      []credentialEntry `json:"credentials"`
}

type credentialRecoveryExportReport struct {
	Written int      `json:"written"`
	Skipped []string `json:"skipped"`
}

func (app *Service) List(ctx context.Context, root string, out io.Writer, opts ListOptions) error {
	format := strings.TrimSpace(opts.Format)
	if format == "" {
		format = string(cliout.FormatHuman)
	}
	if format != string(cliout.FormatHuman) && format != string(cliout.FormatJSON) {
		return fmt.Errorf("credentials list accepts only --format text|json")
	}
	entries, err := collectCredentialEntries(root)
	if err != nil {
		return err
	}
	recovery := computeRecoveryStatus(entries)
	report := credentialListReport{
		Basis:            "distinct_addresses",
		Managed:          true,
		CredentialCount:  distinctCredentialCount(entries),
		DeclarationSites: len(entries),
		Uncovered:        recovery.Uncovered,
		RequiredAbsent:   recovery.RequiredAbsent,
		Credentials:      entries,
	}
	if format == string(cliout.FormatJSON) {
		return cliout.WriteJSONValue(out, report)
	}
	fmt.Fprintf(out, "Credential addresses (%d; basis=distinct_addresses; managed_instances_included=true)\n", report.CredentialCount)
	fmt.Fprintf(out, "Declaration sites: %d (basis=declaration_sites)\n", report.DeclarationSites)
	fmt.Fprintf(out, "Uncovered: %d (basis=distinct_addresses)\n", len(report.Uncovered))
	fmt.Fprintf(out, "Required but absent: %d (basis=distinct_addresses)\n", len(report.RequiredAbsent))
	writeCredentialTable(out, entries)
	return nil
}

func (app *Service) Delete(ctx context.Context, out io.Writer, opts CredentialSelectorOptions) error {
	identity, err := credentialauthority.ParseIdentity(opts.Identity)
	if err != nil {
		return err
	}
	field := strings.TrimSpace(opts.Field)
	if field == "" {
		field = credentialDefaultField
	}
	if field == "" {
		return errors.New("credential field is required")
	}
	if !opts.Yes {
		return fmt.Errorf("refusing to delete %s:%s without explicit --yes confirmation", identity, field)
	}
	authority, err := credentialAuthority()
	if err != nil {
		return err
	}
	if err := authority.Delete(identity, field); err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "Credential %s:%s deleted; no value was printed.\n", identity, field)
	return err
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

func (app *Service) Provision(ctx context.Context, out, errOut io.Writer, opts CredentialSelectorOptions, input io.Reader) error {
	identity, err := credentialauthority.ParseIdentity(opts.Identity)
	if err != nil {
		return err
	}
	field := strings.TrimSpace(opts.Field)
	if field == "" {
		field = credentialDefaultField
	}
	if field == "" {
		return fmt.Errorf("credential field is required")
	}
	value, err := readCredentialValue(input, errOut)
	if err != nil {
		return err
	}
	authority, err := credentialAuthority()
	if err != nil {
		return err
	}
	wasConfigured := authority.Status(identity, field).Configured
	if err := authority.Put(identity, field, strings.TrimSpace(string(value))); err != nil {
		return err
	}
	if wasConfigured {
		if generation, rotateErr := rotateBridgeCredentialAddress(ctx, string(identity), field); rotateErr != nil {
			_, _ = fmt.Fprintf(errOut, "Credential stored locally, but bridge generation fanout was deferred: %v\n", rotateErr)
		} else {
			_, _ = fmt.Fprintf(out, "Bridge credential generation advanced to %d and delivery was queued.\n", generation)
		}
	}
	// The backend is named rather than assumed: on a headless host the value
	// went into the encrypted store, and telling the operator it went into "the
	// native secure store" would be false on exactly the hosts this path exists
	// for.
	_, err = fmt.Fprintf(out, "Credential %s/%s provisioned in the %s credential store.\n",
		identity, field, authority.Provider())
	return err
}

func readCredentialValue(input io.Reader, prompt io.Writer) (string, error) {
	if file, ok := input.(*os.File); ok {
		if info, err := file.Stat(); err == nil && info.Mode()&os.ModeCharDevice != 0 {
			value, readErr := readInteractiveCredentialValue(file, prompt)
			if readErr != nil {
				return "", readErr
			}
			if strings.TrimSpace(value) == "" {
				return "", fmt.Errorf("credential value is required")
			}
			return value, nil
		}
	}
	if err := refuseInteractiveStdin(input,
		`provide the value on standard input to vrooli credentials provision`); err != nil {
		return "", err
	}
	value, err := io.ReadAll(io.LimitReader(input, credentialsParameterB*credentialsParameterA))
	if err != nil {
		return "", fmt.Errorf("read credential input: %w", err)
	}
	valueString := strings.TrimSpace(string(value))
	if valueString == "" {
		return "", fmt.Errorf("credential value is required")
	}
	return valueString, nil
}

func rotateBridgeCredentialAddress(ctx context.Context, logicalID, field string) (int64, error) {
	bridgeClient := nodereach.New(nodereach.Config{Token: os.Getenv("VROOLI_BRIDGE_API_TOKEN")})
	baseURL, err := bridgeClient.ResolveURL(ctx)
	if err != nil {
		return 0, fmt.Errorf("resolve vrooli-bridge endpoint: %w", err)
	}
	token := strings.TrimSpace(os.Getenv("VROOLI_BRIDGE_API_TOKEN"))
	transport := bearerRoundTripper{base: http.DefaultTransport, token: token}
	client := grantv1connect.NewCredentialGrantServiceClient(&http.Client{Transport: transport}, baseURL)
	response, err := client.RotateAddress(ctx, connect.NewRequest(&grantv1.RotateAddressRequest{LogicalId: logicalID, Field: field}))
	if err != nil {
		return 0, err
	}
	return response.Msg.GetGeneration(), nil
}

type bearerRoundTripper struct {
	base  http.RoundTripper
	token string
}

func (t bearerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.token == "" {
		return t.base.RoundTrip(req)
	}
	copyReq := req.Clone(req.Context())
	copyReq.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(copyReq)
}

func (app *Service) Status(ctx context.Context, out io.Writer, opts CredentialSelectorOptions) error {
	identity, err := credentialauthority.ParseIdentity(opts.Identity)
	if err != nil {
		return err
	}
	field := strings.TrimSpace(opts.Field)
	if field == "" {
		field = credentialDefaultField
	}
	format := strings.TrimSpace(opts.Format)
	if format == "" {
		format = string(cliout.FormatHuman)
	}
	if format != string(cliout.FormatHuman) && format != string(cliout.FormatJSON) {
		return fmt.Errorf("credentials status format must be text or json")
	}
	authority, err := credentialAuthority()
	if err != nil {
		return err
	}
	status := authority.Status(identity, field)
	if format == string(cliout.FormatJSON) {
		return cliout.WriteJSONValue(out, status)
	}
	// The provider state travels with the answer so `configured: false` can
	// never be misread as "the operator never set this" while the store is down.
	_, err = fmt.Fprintf(out, "Credential %s/%s: %s (provider %s, %s)\n",
		identity, field,
		map[bool]string{true: "configured", false: "unconfigured"}[status.Configured],
		status.Provider, status.ProviderState)
	if err != nil {
		return err
	}
	if status.ProviderDetail != "" {
		_, err = fmt.Fprintf(out, "  %s\n  Run `vrooli credentials doctor` for the host diagnosis.\n", status.ProviderDetail)
	}
	return err
}

const credentialDefaultField = "value"

// collectCredentialEntries projects the authoritative inventory walk into the
// richer CLI row shape. Source discovery lives in credentialinventory so list,
// doctor, and recovery export cannot drift apart.
func collectCredentialEntries(root string) ([]credentialEntry, error) {
	result, err := credentialinventory.CollectWithOptions(root, credentialinventory.CollectionOptions{
		LiveVaultUnsealKeyEntries:  liveVaultUnsealKeyEntries,
		LiveKopiaRepositoryEntries: liveKopiaRepositoryEntries,
	})
	if err != nil {
		return nil, err
	}
	entries := make([]credentialEntry, 0, len(result.Rows))
	for _, row := range result.Rows {
		entries = append(entries, credentialEntry{
			Resource: row.Resource, Env: row.Env, LogicalID: row.LogicalID, Field: row.Field,
			Label: row.Label, Description: row.Description, Required: row.Required,
			Provisioning: row.Provisioning, DerivedFrom: row.DerivedFrom,
			Configured: row.Configured, State: row.State, Remediation: row.Remediation,
		})
	}
	return entries, nil
}

// liveVaultUnsealKeyEntries is replaceable so credential tests can isolate
// dynamic managed-instance discovery without depending on host state.
var liveVaultUnsealKeyEntries = resources.LiveVaultUnsealKeyEntries

// liveKopiaRepositoryEntries is replaceable so inventory tests can assert one
// row per repository without mutating host state.
var liveKopiaRepositoryEntries = resources.LiveKopiaRepositoryEntries

// recoveryStateDir resolves where the export receipt lives, through the repo
// contract rather than an assembled path.
func recoveryStateDir() (string, error) {
	return config.VrooliPath(repocontract.HomeKeyState)
}

func recoveryPassphrase(input io.Reader, prompt io.Writer) (string, error) {
	if file, ok := input.(*os.File); ok {
		if info, err := file.Stat(); err == nil && info.Mode()&os.ModeCharDevice != 0 {
			value, readErr := readInteractivePassphrase(file, prompt)
			if readErr != nil {
				return "", readErr
			}
			if value == "" {
				return "", fmt.Errorf("recovery passphrase is required")
			}
			return value, nil
		}
	}
	if err := refuseInteractiveStdin(input,
		`provide the recovery passphrase on standard input to vrooli credentials recovery`); err != nil {
		return "", err
	}
	value, err := io.ReadAll(io.LimitReader(input, credentialsParameterB*credentialsParameterA))
	if err != nil {
		return "", fmt.Errorf("read recovery passphrase: %w", err)
	}
	passphrase := strings.TrimSpace(string(value))
	if passphrase == "" {
		return "", fmt.Errorf("recovery passphrase is required")
	}
	return passphrase, nil
}

//nolint:gocyclo // recovery export branches by provider, encryption, and artifact verification state.
func (app *Service) ExportRecovery(ctx context.Context, root string, out, errOut io.Writer, opts RecoveryExportOptions, input io.Reader) error {
	entries := opts.Entries
	output := strings.TrimSpace(opts.Output)
	all := opts.All
	format := strings.TrimSpace(opts.Format)
	if output == "" {
		return fmt.Errorf("recovery export requires --output")
	}
	if all && len(entries) > 0 {
		return fmt.Errorf("recovery export accepts either --all or --entry, not both")
	}
	if !all && len(entries) == 0 {
		return fmt.Errorf("recovery export requires at least one --entry or --all")
	}
	if format == "" {
		format = string(cliout.FormatHuman)
	}
	if format != string(cliout.FormatHuman) && format != string(cliout.FormatJSON) {
		return fmt.Errorf("credentials recovery export format must be text or json")
	}
	selected := make([]credentialauthority.RecoveryEntry, 0, len(entries))
	skipped := []string{}
	if all {
		declared, err := collectCredentialEntries(root)
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
	passphrase, err := recoveryPassphrase(input, errOut)
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
	file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, tuning.PermSecret)
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

	if format == string(cliout.FormatJSON) {
		return cliout.WriteJSONValue(out, credentialRecoveryExportReport{Written: len(selected), Skipped: skipped})
	}
	if _, err = fmt.Fprintf(out, "Encrypted recovery bundle created for %d credential entries.\n", len(selected)); err != nil {
		return err
	}
	if all {
		fmt.Fprintf(out, "Skipped %d unconfigured entries", len(skipped))
		if len(skipped) > 0 {
			fmt.Fprintf(out, ": %s", strings.Join(skipped, ", "))
		}
		fmt.Fprintln(out, ".")
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
func (app *Service) VerifyRecovery(ctx context.Context, out, errOut io.Writer, opts RecoveryBundleOptions, input io.Reader) error {
	path := strings.TrimSpace(opts.Input)
	format := strings.TrimSpace(opts.Format)
	if path == "" {
		return fmt.Errorf("recovery verify requires --input")
	}
	if format == "" {
		format = string(cliout.FormatHuman)
	}
	format = strings.TrimSpace(format)
	if format != string(cliout.FormatHuman) && format != string(cliout.FormatJSON) {
		return fmt.Errorf("credentials recovery verify format must be text or json")
	}
	passphrase, err := recoveryPassphrase(input, errOut)
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
	if format == string(cliout.FormatJSON) {
		return cliout.WriteJSONValue(out, manifest)
	}
	if _, err := fmt.Fprintf(out,
		"Recovery bundle opens. It holds %d credential(s) and would restore:\n", len(manifest.Entries)); err != nil {
		return err
	}
	for _, entry := range manifest.Entries {
		if _, err := fmt.Fprintf(out, "  %s:%s\n", entry.Identity, entry.Field); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(out, "\nNo value was printed and nothing was written. Keep this bundle and its passphrase apart, and off this machine.")
	return err
}

func (app *Service) RestoreRecovery(ctx context.Context, out, errOut io.Writer, opts RecoveryBundleOptions, input io.Reader) error {
	path := strings.TrimSpace(opts.Input)
	if path == "" {
		return fmt.Errorf("recovery restore requires --input")
	}
	passphrase, err := recoveryPassphrase(input, errOut)
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
	_, err = fmt.Fprintf(out, "Encrypted recovery bundle restored to the %s credential store.\n", authority.Provider())
	return err
}
