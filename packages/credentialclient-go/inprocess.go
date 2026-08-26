package credentialclient

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/credentialinventory"
	controlcredentials "github.com/vrooli/vrooli/internal/credentials"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/resources/securestore"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

type InProcessOptions struct {
	Authority   *credentialauthority.Authority
	Root        string
	StateDir    string
	Descriptors func() ([]CredentialRef, error)
}

type inProcessClient struct {
	authority   *credentialauthority.Authority
	root        string
	stateDir    string
	descriptors func() ([]CredentialRef, error)
}

func NewInProcess(options InProcessOptions) (Client, error) {
	if options.Authority == nil {
		return nil, fmt.Errorf("in-process credential authority is required")
	}
	return &inProcessClient{authority: options.Authority, root: options.Root, stateDir: options.StateDir, descriptors: options.Descriptors}, nil
}

func (c *inProcessClient) Provision(_ context.Context, request ProvisionRequest) (ProvisionResponse, error) {
	identity, err := credentialauthority.ParseIdentity(request.Identity)
	if err != nil {
		return ProvisionResponse{}, err
	}
	if err := c.authority.Put(identity, request.Field, request.Value); err != nil {
		return ProvisionResponse{}, err
	}
	return ProvisionResponse{Identity: string(identity), Field: request.Field, Provider: c.authority.Provider(), Status: "provisioned"}, nil
}

func (c *inProcessClient) Status(_ context.Context, identity, field string) (CredentialStatus, error) {
	parsed, err := credentialauthority.ParseIdentity(identity)
	if err != nil {
		return CredentialStatus{}, err
	}
	status := c.authority.Status(parsed, field)
	return CredentialStatus{Identity: string(status.Identity), Field: status.Field, Configured: status.Configured, Provider: status.Provider, ProviderState: string(status.ProviderState), ProviderDetail: status.ProviderDetail}, nil
}

func (c *inProcessClient) Resolve(_ context.Context, identity, field string) (string, error) {
	parsed, err := credentialauthority.ParseIdentity(identity)
	if err != nil {
		return "", err
	}
	return c.authority.Resolve(parsed, field)
}

func (c *inProcessClient) Delete(_ context.Context, identity, field string) error {
	parsed, err := credentialauthority.ParseIdentity(identity)
	if err != nil {
		return err
	}
	return c.authority.Delete(parsed, field)
}

// List returns the whole declared population for the configured root.
//
// Manifest descriptors are read first because they carry what a recovery entry
// cannot: the owner label, the operator-facing label, and the required flag.
// The control-plane inventory then contributes the live managed instances that
// no manifest declares — the release-authority key, device-control entries,
// Vault unseal keys, Kopia repository passphrases — as metadata-only refs, so
// the population stays whole without inventing an owner for them.
func (c *inProcessClient) List(_ context.Context) ([]CredentialRef, error) {
	if strings.TrimSpace(c.root) == "" {
		if c.descriptors == nil {
			return []CredentialRef{}, nil
		}
		return c.descriptors()
	}
	refs, err := DescriptorsForScope(c.root, Scope{IncludeProject: true})
	if err != nil {
		return nil, err
	}
	collected, err := credentialinventory.Collect(c.root)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		seen[ref.LogicalID+":"+ref.Field] = struct{}{}
	}
	for _, entry := range collected.Declared {
		key := string(entry.Identity) + ":" + entry.Field
		if _, found := seen[key]; found {
			continue
		}
		seen[key] = struct{}{}
		refs = append(refs, CredentialRef{LogicalID: string(entry.Identity), Field: entry.Field, Required: true})
	}
	return refs, nil
}

// Inventory returns the control-plane inventory when a repository root is
// available. The fallback uses the injected descriptor source so older
// callers and test doubles remain metadata-safe and functional.
func (c *inProcessClient) Inventory(ctx context.Context) (InventoryResponse, error) {
	refs, err := c.List(ctx)
	if err != nil {
		return InventoryResponse{}, err
	}
	response := InventoryResponse{
		Credentials:              append([]CredentialRef(nil), refs...),
		CredentialCount:          distinctCredentialCount(refs),
		DeclarationSiteCount:     len(refs),
		InventoryBasis:           "distinct_addresses",
		ManagedInstancesIncluded: false,
		Uncovered:                []string{},
		RequiredAbsent:           []string{},
	}
	if strings.TrimSpace(c.root) == "" {
		return response, nil
	}
	// List has already merged the managed instances into the population; the
	// inventory result is read again here only for the counting basis and the
	// required-absent classification, which are Collect's own answers.
	collected, collectErr := credentialinventory.Collect(c.root)
	if collectErr != nil {
		return response, collectErr
	}
	response.DeclarationSiteCount = collected.DeclarationSiteCount
	response.ManagedInstancesIncluded = collected.ManagedInstancesIncluded
	response.InventoryBasis = collected.Basis
	response.RequiredAbsent = append(response.RequiredAbsent, collected.RequiredAbsent...)
	return response, nil
}

func (c *inProcessClient) Doctor(ctx context.Context) (DoctorResponse, error) {
	diagnosis := securestore.Diagnose()
	inventory, err := c.Inventory(ctx)
	if err != nil {
		return DoctorResponse{}, err
	}
	descriptors := inventory.Credentials
	response := DoctorResponse{
		Credentials:              descriptors,
		CredentialCount:          inventory.CredentialCount,
		DeclarationSiteCount:     inventory.DeclarationSiteCount,
		InventoryBasis:           inventory.InventoryBasis,
		ManagedInstancesIncluded: inventory.ManagedInstancesIncluded,
		Recovery:                 RecoveryStatus{Uncovered: []string{}, RequiredAbsent: append([]string(nil), inventory.RequiredAbsent...), Basis: inventory.InventoryBasis, ManagedInstancesIncluded: inventory.ManagedInstancesIncluded},
	}
	response.Provider = ProviderDiagnosis{Platform: diagnosis.Platform, Adapter: diagnosis.Adapter, Backend: diagnosis.Backend, Condition: diagnosis.Condition, Available: diagnosis.Available, Writable: diagnosis.Writable, Explanation: diagnosis.Explanation, Fix: diagnosis.Fix}
	if c.stateDir != "" {
		if receipt, found, receiptErr := credentialauthority.ReadRecoveryReceipt(c.stateDir); receiptErr == nil && found {
			response.Recovery.ReceiptExists = true
			response.Recovery.ExportedAt = receipt.ExportedAt.Format("2006-01-02T15:04:05Z07:00")
			response.Recovery.EntryCount = len(receipt.Entries)
			for _, descriptor := range descriptors {
				if !credentialConfigured(c.authority, descriptor) {
					continue
				}
				identity, parseErr := credentialauthority.ParseIdentity(descriptor.LogicalID)
				if parseErr != nil || !receipt.Covers(identity, descriptor.Field) {
					response.Recovery.Uncovered = append(response.Recovery.Uncovered, descriptor.LogicalID+":"+descriptor.Field)
				}
			}
		} else {
			for _, descriptor := range descriptors {
				if credentialConfigured(c.authority, descriptor) {
					response.Recovery.Uncovered = append(response.Recovery.Uncovered, descriptor.LogicalID+":"+descriptor.Field)
				}
			}
		}
	}
	return response, nil
}

func credentialConfigured(authority *credentialauthority.Authority, descriptor CredentialRef) bool {
	identity, err := credentialauthority.ParseIdentity(descriptor.LogicalID)
	return err == nil && authority.Status(identity, descriptor.Field).Configured
}

func distinctCredentialCount(descriptors []CredentialRef) int {
	seen := make(map[string]struct{}, len(descriptors))
	for _, descriptor := range descriptors {
		seen[descriptor.LogicalID+":"+descriptor.Field] = struct{}{}
	}
	return len(seen)
}

func (c *inProcessClient) KeyringInspect(_ context.Context, path string) (KeyringReport, error) {
	report, err := c.authority.KeyringInspect(path)
	capability := hostinventory.CredentialStoreStatus(context.Background())
	verdict := controlcredentials.DeriveKeyringVerdict(report, capability)
	return keyringReport(report, verdict), err
}

func (c *inProcessClient) KeyringRepair(_ context.Context, path string) (KeyringReport, error) {
	report, err := c.authority.KeyringRepair(path)
	capability := hostinventory.CredentialStoreStatus(context.Background())
	verdict := controlcredentials.DeriveKeyringVerdict(report, capability)
	return keyringReport(report, verdict), err
}

func keyringReport(report securestore.KeyringReport, verdict controlcredentials.KeyringVerdict) KeyringReport {
	backups := make([]KeyringBackup, 0, len(report.Backups))
	for _, backup := range report.Backups {
		backups = append(backups, KeyringBackup{Path: backup.Path, ModifiedAt: backup.ModifiedAt, AgeSeconds: backup.AgeSeconds})
	}
	return KeyringReport{Path: report.Path, Format: report.Format, Assessed: report.Assessed, Loadable: report.Loadable, Repaired: report.Repaired, Verdict: string(verdict.State), VerdictReason: verdict.Reason, Backups: backups}
}

func (c *inProcessClient) RecoveryExport(_ context.Context, request RecoveryExportRequest) (RecoveryExportResponse, error) {
	if len(request.Entries) == 0 {
		return RecoveryExportResponse{}, fmt.Errorf("recovery export requires at least one credential")
	}
	entries := make([]credentialauthority.RecoveryEntry, 0, len(request.Entries))
	for _, ref := range request.Entries {
		identity, err := credentialauthority.ParseIdentity(ref.LogicalID)
		if err != nil {
			return RecoveryExportResponse{}, err
		}
		entries = append(entries, credentialauthority.RecoveryEntry{Identity: identity, Field: ref.Field})
	}
	bundle, err := c.authority.ExportRecovery(entries, request.Passphrase)
	if err != nil {
		return RecoveryExportResponse{}, err
	}
	if strings.TrimSpace(request.OutputPath) == "" {
		return RecoveryExportResponse{}, fmt.Errorf("recovery output path is required")
	}
	if err := os.MkdirAll(filepath.Dir(request.OutputPath), 0o700); err != nil {
		return RecoveryExportResponse{}, err
	}
	if err := os.WriteFile(request.OutputPath, bundle, 0o600); err != nil {
		return RecoveryExportResponse{}, err
	}
	if c.stateDir != "" {
		_ = credentialauthority.WriteRecoveryReceipt(c.stateDir, request.OutputPath, entries, time.Now())
	}
	return RecoveryExportResponse{Path: request.OutputPath, EntryCount: len(entries)}, nil
}

func (c *inProcessClient) RecoveryVerify(_ context.Context, request RecoveryVerifyRequest) (RecoveryVerifyResponse, error) {
	bundle, err := os.ReadFile(request.InputPath)
	if err != nil {
		return RecoveryVerifyResponse{}, err
	}
	manifest, err := credentialauthority.InspectRecovery(bundle, request.Passphrase)
	if err != nil {
		return RecoveryVerifyResponse{}, err
	}
	entries := make([]string, 0, len(manifest.Entries))
	for _, entry := range manifest.Entries {
		entries = append(entries, string(entry.Identity)+":"+entry.Field)
	}
	return RecoveryVerifyResponse{Version: manifest.Version, Entries: entries}, nil
}

func (c *inProcessClient) RecoveryRestore(_ context.Context, request RecoveryRestoreRequest) error {
	bundle, err := os.ReadFile(request.InputPath)
	if err != nil {
		return err
	}
	return c.authority.RestoreRecovery(bundle, request.Passphrase)
}

func (c *inProcessClient) StoreStatus(context.Context) (StoreStatus, error) {
	status, err := securestore.DescribeStore()
	if err != nil {
		return StoreStatus{}, err
	}
	return StoreStatus{Path: status.Path, Initialized: status.Initialized, Unlocked: status.Unlocked, Entries: status.Entries, Active: status.Active}, nil
}

var _ Client = (*inProcessClient)(nil)
