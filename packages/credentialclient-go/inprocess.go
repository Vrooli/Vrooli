package credentialclient

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/internal/credentialinventory"
	controlcredentials "github.com/vrooli/vrooli/internal/credentials"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/securestore"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const (
	credentialBundleDirMode  = 0o700
	credentialBundleFileMode = 0o600
	credentialTokenBytes     = 32
	credentialHTTPTimeout    = 15 * time.Second
	credentialResponseLimit  = 1 << 20
	defaultHydrationLeaseTTL = 5 * time.Minute
	recoveryFreshnessWindow  = 30 * 24 * time.Hour
)

type InProcessOptions struct {
	Authority       *credentialauthority.Authority
	Root            string
	StateDir        string
	Descriptors     func() ([]CredentialRef, error)
	DescriptorScope *Scope
}

type inProcessClient struct {
	authority   *credentialauthority.Authority
	root        string
	stateDir    string
	descriptors func() ([]CredentialRef, error)
	scope       *Scope
	leaseMu     sync.Mutex
	leases      map[string]*hydrationLease
}

type hydrationLease struct {
	env     string
	target  map[string]string
	expires time.Time
}

func NewInProcess(options InProcessOptions) (Client, error) {
	if options.Authority == nil {
		return nil, fmt.Errorf("in-process credential authority is required")
	}
	return &inProcessClient{authority: options.Authority, root: options.Root, stateDir: options.StateDir, descriptors: options.Descriptors, scope: options.DescriptorScope, leases: make(map[string]*hydrationLease)}, nil
}

func (c *inProcessClient) Provision(_ context.Context, request ProvisionRequest) (ProvisionResponse, error) {
	identity, err := credentialauthority.ParseIdentity(request.Identity)
	if err != nil {
		return ProvisionResponse{}, err
	}
	// Treat operator delivery as a candidate transaction even for the
	// in-process transport. The active value is changed only after the
	// candidate has been durably accepted, and a failed activation cannot
	// erase the previous active value.
	candidate, err := c.authority.PutCandidate(identity, request.Field, request.Value)
	if err != nil {
		return ProvisionResponse{}, err
	}
	if err := c.authority.ActivateCandidate(candidate); err != nil {
		return ProvisionResponse{}, err
	}
	return ProvisionResponse{Identity: string(identity), Field: request.Field, Provider: c.authority.Provider(), Status: "provisioned"}, nil
}

func (c *inProcessClient) Hydrate(_ context.Context, request HydrationRequest) (HydrationResponse, error) {
	identity, err := credentialauthority.ParseIdentity(request.Identity)
	if err != nil {
		return HydrationResponse{}, err
	}
	if strings.TrimSpace(request.Env) == "" || request.Target == nil {
		return HydrationResponse{}, fmt.Errorf("runtime injection requires an environment name and target")
	}
	if err := c.authority.Inject(identity, request.Field, request.Env, request.Target); err != nil {
		return HydrationResponse{}, err
	}
	ttl := request.LeaseTTL
	if ttl <= 0 {
		ttl = defaultHydrationLeaseTTL
	}
	leaseID, err := newHydrationLeaseID()
	if err != nil {
		delete(request.Target, request.Env)
		return HydrationResponse{}, err
	}
	now := time.Now().UTC()
	expires := now.Add(ttl)
	c.leaseMu.Lock()
	c.reapExpiredLocked(now)
	c.leases[leaseID] = &hydrationLease{env: request.Env, target: request.Target, expires: expires}
	c.leaseMu.Unlock()
	return HydrationResponse{Identity: string(identity), Field: request.Field, ExposureMode: ExposureRuntimeInjection, Injected: true, LeaseID: leaseID, ExpiresAt: expires}, nil
}

func (c *inProcessClient) RevokeHydration(_ context.Context, request HydrationRevocationRequest) (HydrationRevocationResponse, error) {
	leaseID := strings.TrimSpace(request.LeaseID)
	if leaseID == "" {
		return HydrationRevocationResponse{}, fmt.Errorf("hydration lease id is required")
	}
	now := time.Now().UTC()
	c.leaseMu.Lock()
	c.reapExpiredLocked(now)
	lease, ok := c.leases[leaseID]
	if ok {
		delete(lease.target, lease.env)
		delete(c.leases, leaseID)
	}
	c.leaseMu.Unlock()
	if !ok {
		return HydrationRevocationResponse{}, fmt.Errorf("hydration lease %q is unknown or already revoked", leaseID)
	}
	exposure := "stopped_process"
	if request.ProcessRunning {
		exposure = "already_running_process"
	}
	return HydrationRevocationResponse{LeaseID: leaseID, Status: "revoked", FutureDeliveryStopped: true, ProcessExposure: exposure}, nil
}

func (c *inProcessClient) reapExpiredLocked(now time.Time) {
	for leaseID, lease := range c.leases {
		if !lease.expires.After(now) {
			delete(lease.target, lease.env)
			delete(c.leases, leaseID)
		}
	}
}

func newHydrationLeaseID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("create hydration lease: %w", err)
	}
	return hex.EncodeToString(raw[:]), nil
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
	return c.authority.Require(parsed, field)
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
	scope := Scope{IncludeProject: true}
	if c.scope != nil {
		scope = *c.scope
	}
	refs, err := DescriptorsForScope(c.root, scope)
	if err != nil {
		return nil, err
	}
	if c.scope != nil {
		return refs, nil
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
	if c.scope != nil {
		response.ManagedInstancesIncluded = false
		response.InventoryBasis = "scoped_distinct_addresses"
		response.RequiredAbsent = nil
		for _, descriptor := range refs {
			if descriptor.Required && !credentialConfigured(c.authority, descriptor) {
				response.RequiredAbsent = append(response.RequiredAbsent, descriptor.LogicalID+":"+descriptor.Field)
			}
		}
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
		Recovery:                 RecoveryStatus{Status: "unassessed", Uncovered: []string{}, RequiredAbsent: append([]string(nil), inventory.RequiredAbsent...), Basis: inventory.InventoryBasis, ManagedInstancesIncluded: inventory.ManagedInstancesIncluded},
	}
	response.Provider = ProviderDiagnosis{Platform: diagnosis.Platform, Adapter: diagnosis.Adapter, Backend: diagnosis.Backend, Condition: diagnosis.Condition, Available: diagnosis.Available, Writable: diagnosis.Writable, Explanation: diagnosis.Explanation, Fix: diagnosis.Fix}
	if c.stateDir != "" {
		if receipt, found, receiptErr := credentialauthority.ReadRecoveryReceipt(c.stateDir); receiptErr == nil && found {
			response.Recovery.ReceiptExists = true
			response.Recovery.ExportedAt = receipt.ExportedAt.Format("2006-01-02T15:04:05Z07:00")
			response.Recovery.EntryCount = len(receipt.Entries)
			response.Recovery.AgeSeconds = maxInt64(0, int64(time.Since(receipt.ExportedAt).Seconds()))
			for _, descriptor := range descriptors {
				if !credentialConfigured(c.authority, descriptor) {
					continue
				}
				identity, parseErr := credentialauthority.ParseIdentity(descriptor.LogicalID)
				if parseErr != nil || !receipt.Covers(identity, descriptor.Field) {
					response.Recovery.Uncovered = append(response.Recovery.Uncovered, descriptor.LogicalID+":"+descriptor.Field)
				}
			}
			switch {
			case len(response.Recovery.Uncovered) > 0 || len(response.Recovery.RequiredAbsent) > 0:
				response.Recovery.Status = "incomplete"
				response.Recovery.FreshnessReason = "the latest recovery receipt does not cover every current required or configured credential"
			case response.Recovery.AgeSeconds > int64(recoveryFreshnessWindow.Seconds()):
				response.Recovery.Status = "stale"
				response.Recovery.FreshnessReason = "the recovery receipt is older than the supported freshness window"
			default:
				response.Recovery.Status = "protected"
				response.Recovery.FreshnessReason = "the latest recovery receipt covers the current configured credential inventory"
			}
		} else {
			response.Recovery.Status = "incomplete"
			response.Recovery.FreshnessReason = "no verified recovery receipt exists"
			for _, descriptor := range descriptors {
				if credentialConfigured(c.authority, descriptor) {
					response.Recovery.Uncovered = append(response.Recovery.Uncovered, descriptor.LogicalID+":"+descriptor.Field)
				}
			}
		}
	}
	return response, nil
}

func maxInt64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
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
	if _, err := credentialauthority.InspectRecovery(bundle, request.Passphrase); err != nil {
		return RecoveryExportResponse{}, fmt.Errorf("verify recovery bundle before recording receipt: %w", err)
	}
	if strings.TrimSpace(request.OutputPath) == "" {
		return RecoveryExportResponse{}, fmt.Errorf("recovery output path is required")
	}
	if err := os.MkdirAll(filepath.Dir(request.OutputPath), credentialBundleDirMode); err != nil { //nolint:mnd // credential bundle directory mode is a security contract
		return RecoveryExportResponse{}, err
	}
	if err := os.WriteFile(request.OutputPath, bundle, credentialBundleFileMode); err != nil {
		return RecoveryExportResponse{}, err
	}
	if c.stateDir != "" {
		now := time.Now().UTC()
		if err := credentialauthority.WriteRecoveryReceiptWithMetadata(c.stateDir, request.OutputPath, entries, credentialauthority.RecoveryReceipt{VerifiedAt: now, Verification: "decrypt-readback", ScheduleState: "manual"}, now); err != nil {
			return RecoveryExportResponse{}, fmt.Errorf("record verified recovery receipt: %w", err)
		}
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
