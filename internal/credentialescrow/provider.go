// Package credentialescrow owns the control-plane provider for encrypted
// credential-store escrow and recovery. Consumers use the generic
// operatorcapability contract; they do not import this package to render an
// action.
package credentialescrow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/credentialinventory"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/operatorcapability"
	"github.com/vrooli/vrooli/internal/resources/securestore"
	credentialauthority "github.com/vrooli/vrooli/internal/secrets"
	kopiaregistry "github.com/vrooli/vrooli/packages/kopiaregistry-go"
)

const (
	CapabilityID      = "credential-escrow"
	StoreCapabilityID = "credential-store-access"
	ProviderID        = "vrooli.control-plane.credential-escrow"
	Owner             = "vrooli.control-plane"
)

type Provider struct {
	root string
	home string
	now  func() time.Time

	discoverStorage  func(hostinventory.StoragePolicy) ([]hostinventory.StorageCandidate, error)
	describeStore    func() (securestore.StoreStatus, error)
	readCopyConfig   func(string) (securestore.CopyConfig, error)
	readReceipt      func(string) (credentialauthority.RecoveryReceipt, bool, error)
	loadRepositories func() ([]kopiaregistry.Entry, error)
}

func NewProvider(root, home string) *Provider {
	return &Provider{
		root:            strings.TrimSpace(root),
		home:            strings.TrimSpace(home),
		now:             time.Now,
		discoverStorage: hostinventory.DiscoverStorageCandidates,
		describeStore:   securestore.DescribeStore,
		readCopyConfig:  securestore.ReadCopyConfig,
		readReceipt:     credentialauthority.ReadRecoveryReceipt,
		loadRepositories: func() ([]kopiaregistry.Entry, error) {
			return kopiaregistry.New(kopiaregistry.RegistryPath()).Load()
		},
	}
}

// NewProviders returns store access and escrow as separate capabilities. Store
// access may be resolved before an escrow sink exists; separate descriptors
// keep each action's typed input contract valid for its current state.
func NewProviders(root, home string) []operatorcapability.Provider {
	provider := NewProvider(root, home)
	return []operatorcapability.Provider{newStoreProvider(provider), provider}
}

func (p *Provider) Descriptor() operatorcapability.Descriptor {
	return operatorcapability.Descriptor{
		Version:     operatorcapability.ContractVersion,
		ID:          CapabilityID,
		Owner:       Owner,
		Title:       "Protect credentials with verified escrow",
		Description: "Select an approved escrow sink, create a verified encrypted root copy and recovery bundle, and keep their evidence fresh.",
		Risk:        "A destination is never adopted automatically; an unverified or same-device destination is not reported as protection.",
		Inputs: []operatorcapability.InputDescriptor{
			{ID: "sink", Kind: operatorcapability.KindPath, Label: "Escrow destination", Description: "Choose a validated local, removable, or object-store destination. The provider revalidates it immediately before any write.", Required: true, Validation: "approved candidate identity or absolute path"},
			{ID: "recovery_passphrase", Kind: operatorcapability.KindSecret, Label: "Recovery-bundle passphrase", Description: "Enter the passphrase that will protect the recovery bundle. It is used in memory only and is never returned or persisted.", Required: true, Validation: "non-empty; keep separately from the bundle"},
			{ID: "object_store_credential_identity", Kind: operatorcapability.KindPath, Label: "Object-store credential identity", Description: "Reference the existing credential identity holding the object-store access material; enter an identity, never a secret value.", Required: false},
			{ID: "object_store_region", Kind: operatorcapability.KindPath, Label: "Object-store region", Description: "Region used to sign the S3-compatible request.", Required: false},
			{ID: "object_store_endpoint", Kind: operatorcapability.KindPath, Label: "Object-store endpoint", Description: "Optional S3-compatible endpoint URL.", Required: false},
			{ID: "object_store_access_key_field", Kind: operatorcapability.KindPath, Label: "Object-store access-key field", Description: "Credential field containing the access key.", Required: false, Default: "s3-access-key-id"},
			{ID: "object_store_secret_key_field", Kind: operatorcapability.KindPath, Label: "Object-store secret-key field", Description: "Credential field containing the secret key.", Required: false, Default: "s3-secret-access-key"},
			{ID: "object_store_session_field", Kind: operatorcapability.KindPath, Label: "Object-store session-token field", Description: "Optional credential field containing a session token.", Required: false, Default: "s3-session-token"},
			{ID: "confirm", Kind: operatorcapability.KindConfirmation, Label: "Confirm escrow mutations", Description: "Confirm the reviewed root-copy, recovery-bundle, receipt, and schedule mutations.", Required: true},
		},
		Policy: operatorcapability.Policy{
			RequiresConfirmation: true,
			Idempotent:           true,
			Retryable:            true,
			Remediation:          "Select a policy-approved destination, review the preview, and retry a failed verification after correcting the reported condition.",
		},
		Evidence: operatorcapability.EvidenceContract{
			Kinds:          []string{"encrypted-root-copy", "recovery-bundle", "schedule"},
			RequiredFields: []string{"artifact_identity", "source_generation", "checksum", "coverage", "observed_at", "verified"},
			SecretFree:     true,
			Freshness:      "receipt age and source generation must be current",
		},
		Remediation: "Choose an approved sink and provide the recovery passphrase in onboarding; no secret is stored in setup state.",
	}
}

type storeProvider struct{ parent *Provider }

func newStoreProvider(parent *Provider) operatorcapability.Provider {
	return &storeProvider{parent: parent}
}

func (p *storeProvider) Descriptor() operatorcapability.Descriptor {
	return operatorcapability.Descriptor{
		Version:     operatorcapability.ContractVersion,
		ID:          StoreCapabilityID,
		Owner:       Owner,
		Title:       "Make the credential store available",
		Description: "Initialize or unlock the selected credential backend using an operator-supplied passphrase.",
		Inputs: []operatorcapability.InputDescriptor{
			{ID: "passphrase", Kind: operatorcapability.KindSecret, Label: "Credential-store passphrase", Description: "Enter the credential-store passphrase. It is used in memory only and never appears in process arguments or evidence.", Required: true, Validation: "non-empty"},
			{ID: "confirm", Kind: operatorcapability.KindConfirmation, Label: "Confirm credential-store access", Description: "Confirm the initialization or unlock operation.", Required: true},
		},
		Policy:      operatorcapability.Policy{RequiresConfirmation: true, Idempotent: true, Retryable: true, Remediation: "Correct the passphrase or backend session and retry."},
		Evidence:    operatorcapability.EvidenceContract{Kinds: []string{"credential-store-access"}, RequiredFields: []string{"artifact_identity", "observed_at", "verified"}, SecretFree: true},
		Remediation: "Provide the passphrase through onboarding; it is not saved in setup state.",
	}
}

func (p *storeProvider) Discover(ctx context.Context) (operatorcapability.Status, error) {
	_ = ctx
	status := operatorcapability.Status{Descriptor: p.Descriptor(), State: operatorcapability.StateDiscovered, UpdatedAt: p.parent.now().UTC()}
	description, err := p.parent.describeStore()
	if err != nil {
		status.State = operatorcapability.StateDegraded
		status.Remediation = fmt.Sprintf("credential store diagnosis is unavailable: %v", err)
		return status, nil
	}
	if !description.Initialized || !description.Unlocked {
		status.State = operatorcapability.StateNeedsInput
		status.MissingInputs = []string{"passphrase", "confirm"}
		status.Remediation = status.Descriptor.Remediation
		return status, nil
	}
	status.State = operatorcapability.StateReady
	status.Remediation = "credential store is available to the control plane"
	return status, nil
}

func (p *storeProvider) Preview(ctx context.Context, inputs operatorcapability.InputSet) (operatorcapability.Preview, error) {
	_ = ctx
	if _, ok := inputs.Text("passphrase"); !ok {
		return operatorcapability.Preview{}, fmt.Errorf("credential-store passphrase is required")
	}
	return operatorcapability.Preview{
		CapabilityID: StoreCapabilityID,
		PlanID:       StoreCapabilityID,
		State:        operatorcapability.StateReadyToPreview,
		Mutations:    []operatorcapability.Mutation{{ID: "credential-store-access", Summary: "initialize or unlock the credential store", Reversible: true}},
		Remediation:  "Review and confirm the credential-store access operation.",
	}, nil
}

func (p *storeProvider) Apply(ctx context.Context, inputs operatorcapability.InputSet) (operatorcapability.Result, error) {
	_ = ctx
	passphrase, ok := inputs.Text("passphrase")
	if !ok || strings.TrimSpace(passphrase) == "" {
		return operatorcapability.Result{CapabilityID: StoreCapabilityID, State: operatorcapability.StateNeedsInput, Outcome: "missing_passphrase", Retryable: true, ErrorCode: "missing_passphrase", Remediation: "provide the credential-store passphrase"}, nil
	}
	description, err := p.parent.describeStore()
	if err != nil {
		return operatorcapability.Result{CapabilityID: StoreCapabilityID, State: operatorcapability.StateRetryableFailure, Outcome: "diagnosis_failed", Retryable: true, ErrorCode: "diagnosis_failed", Remediation: err.Error()}, nil
	}
	if description.Initialized {
		_, err = securestore.UnlockStore(passphrase)
	} else {
		_, err = securestore.InitializeStore(passphrase)
	}
	if err != nil {
		return operatorcapability.Result{CapabilityID: StoreCapabilityID, State: operatorcapability.StateRetryableFailure, Outcome: "store_access_failed", Retryable: true, ErrorCode: "store_access_failed", Remediation: "credential-store access failed; correct the passphrase or backend session and retry"}, nil
	}
	return operatorcapability.Result{
		CapabilityID: StoreCapabilityID, State: operatorcapability.StateReady, Outcome: "credential_store_ready", Retryable: true,
		Evidence: []operatorcapability.EvidenceReference{{Kind: "credential-store-access", ArtifactIdentity: "encrypted-credential-store", ObservedAt: time.Now().UTC(), Verified: true}},
	}, nil
}

func (p *Provider) Discover(ctx context.Context) (operatorcapability.Status, error) {
	_ = ctx
	if p == nil {
		return operatorcapability.Status{}, fmt.Errorf("credential escrow provider is nil")
	}
	descriptor := p.Descriptor()
	protectedRoots := []string{p.home}
	store, storeErr := p.describeStore()
	if storeErr == nil && strings.TrimSpace(store.Path) != "" {
		protectedRoots = append(protectedRoots, filepath.Dir(store.Path))
	}
	repositoryRoots, repositoriesErr := p.repositoryRoots()
	if repositoriesErr != nil {
		return operatorcapability.Status{}, repositoriesErr
	}
	candidates, err := p.discoverStorage(hostinventory.StoragePolicy{
		ProtectedRoots:            protectedRoots,
		RepositoryRoots:           repositoryRoots,
		RequirePhysicalSeparation: true,
	})
	if err != nil {
		return operatorcapability.Status{}, err
	}
	capabilityCandidates := make([]operatorcapability.Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		metadata := map[string]string{}
		if candidate.Filesystem != "" {
			metadata["filesystem"] = candidate.Filesystem
		}
		capabilityCandidates = append(capabilityCandidates, operatorcapability.Candidate{
			ID: candidate.ID, Kind: candidate.Kind, Label: candidate.Location, Location: candidate.Location,
			StableIdentity: candidate.StableIdentity, DeviceIdentity: candidate.DeviceIdentity,
			Writable: candidate.Writable, PhysicallyIndependent: candidate.PhysicalIndependence,
			Status: candidate.Status, Risk: candidate.Risk, Remediation: candidate.Remediation, Metadata: metadata,
		})
	}
	sort.Slice(capabilityCandidates, func(i, j int) bool { return capabilityCandidates[i].Location < capabilityCandidates[j].Location })
	descriptor.Inputs[0].Candidates = append([]operatorcapability.Candidate(nil), capabilityCandidates...)

	status := operatorcapability.Status{Descriptor: descriptor, Candidates: capabilityCandidates, State: operatorcapability.StateDiscovered, UpdatedAt: p.now().UTC()}
	if storeErr != nil {
		status.State = operatorcapability.StateDegraded
		status.Remediation = fmt.Sprintf("credential store diagnosis is unavailable: %v", storeErr)
		return status, nil
	}
	if !store.Initialized {
		status.State = operatorcapability.StateDegraded
		status.Remediation = "initialize or unlock the encrypted credential store before exporting escrow artifacts"
		return status, nil
	}
	configPath := filepath.Join(p.home, "config", "credential-store-copy.json")
	copyConfig, configErr := p.readCopyConfig(configPath)
	if configErr != nil {
		status.State = operatorcapability.StateDegraded
		status.Remediation = fmt.Sprintf("read escrow configuration: %v", configErr)
		return status, nil
	}
	missing := []string{}
	if !copyConfig.Enabled || strings.TrimSpace(copyConfig.Sink) == "" {
		missing = append(missing, "sink")
	}
	receipt, found, receiptErr := p.readReceipt(filepath.Join(p.home, "state"))
	if receiptErr != nil {
		status.State = operatorcapability.StateDegraded
		status.Remediation = fmt.Sprintf("read recovery evidence: %v", receiptErr)
		return status, nil
	}
	if !found || len(receipt.Entries) == 0 {
		missing = append(missing, "recovery_passphrase")
	} else {
		inventory, inventoryErr := credentialinventory.Collect(p.root)
		if inventoryErr != nil {
			status.State = operatorcapability.StateDegraded
			status.Remediation = "credential coverage could not be revalidated; correct the authority or manifest discovery condition"
			return status, nil
		}
		if len(inventory.RequiredAbsent) > 0 {
			status.State = operatorcapability.StateDegraded
			status.Remediation = fmt.Sprintf("%d required credential value(s) are absent; provision them before claiming complete recovery coverage", len(inventory.RequiredAbsent))
			return status, nil
		}
		if !coversAll(receipt.Entries, inventory.Entries) {
			missing = append(missing, "recovery_passphrase")
		}
	}
	status.MissingInputs = missing
	if len(missing) > 0 {
		status.State = operatorcapability.StateNeedsInput
		status.Remediation = descriptor.Remediation
		return status, nil
	}
	ready, evidence, remediation := p.verifyEvidence(store, receipt, copyConfig)
	status.Evidence = evidence
	if !ready {
		status.State = operatorcapability.StateDegraded
		status.Remediation = remediation
		return status, nil
	}
	status.State = operatorcapability.StateReady
	status.Remediation = "verified escrow evidence exists; re-preview before a refresh or explicit retry"
	return status, nil
}

func (p *Provider) verifyEvidence(store securestore.StoreStatus, receipt credentialauthority.RecoveryReceipt, copyConfig securestore.CopyConfig) (bool, []operatorcapability.EvidenceReference, string) {
	copyStatus, copyFound := readCopyStatus(filepath.Join(p.home, "state", "credential-store-copy.json"))
	if !copyFound || copyStatus.Verification != "readback" || copyStatus.Checksum == "" || copyStatus.VerifiedAt.IsZero() {
		return false, nil, "encrypted root-copy evidence is absent or was not verified by read-back"
	}
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(copyStatus.Path)), "s3://") {
		if _, err := os.Stat(copyStatus.Path); err != nil {
			return false, nil, "encrypted root-copy artifact is missing from its receipt location"
		}
	}
	if receipt.Checksum == "" || receipt.VerifiedAt.IsZero() || receipt.Verification != "decrypt-readback" {
		return false, nil, "recovery-bundle evidence is absent or was not decrypted and read back"
	}
	if store.Path != "" && receipt.SourceGeneration != "" {
		generation, err := securestore.StoreGeneration(store.Path)
		if err != nil || generation != receipt.SourceGeneration {
			return false, nil, "recovery evidence is stale for the current credential-store generation"
		}
	}
	inventory, err := credentialinventory.Collect(p.root)
	if err != nil {
		return false, nil, "credential coverage could not be revalidated"
	}
	if len(inventory.RequiredAbsent) > 0 || !coversAll(receipt.Entries, inventory.Entries) {
		return false, nil, "recovery evidence does not cover every configured credential"
	}
	if copyConfig.Enabled && copyStatus.ScheduleState != "ready" {
		return false, []operatorcapability.EvidenceReference{{Kind: "encrypted-root-copy", ArtifactIdentity: artifactIdentity(copyStatus.Path), SourceGeneration: copyStatus.Generation, Checksum: copyStatus.Checksum, ObservedAt: copyStatus.VerifiedAt, Verified: true}, {Kind: "recovery-bundle", ArtifactIdentity: receipt.ArtifactIdentity, SourceGeneration: receipt.SourceGeneration, Checksum: receipt.Checksum, Coverage: recoveryCoverage(receipt.Entries), ObservedAt: receipt.VerifiedAt, Verified: true}, {Kind: "schedule", ArtifactIdentity: "native-schedule", ObservedAt: receipt.VerifiedAt, Verified: false, Remediation: copyStatus.Remediation}}, "artifacts are verified but the native refresh schedule is degraded"
	}
	return true, []operatorcapability.EvidenceReference{{Kind: "encrypted-root-copy", ArtifactIdentity: artifactIdentity(copyStatus.Path), SourceGeneration: copyStatus.Generation, Checksum: copyStatus.Checksum, ObservedAt: copyStatus.VerifiedAt, Verified: true}, {Kind: "recovery-bundle", ArtifactIdentity: receipt.ArtifactIdentity, SourceGeneration: receipt.SourceGeneration, Checksum: receipt.Checksum, Coverage: recoveryCoverage(receipt.Entries), ObservedAt: receipt.VerifiedAt, Verified: true}, {Kind: "schedule", ArtifactIdentity: "native-schedule", ObservedAt: receipt.VerifiedAt, Verified: true}}, ""
}

func readCopyStatus(path string) (securestore.CopyStatus, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return securestore.CopyStatus{}, false
	}
	var status securestore.CopyStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return securestore.CopyStatus{}, false
	}
	return status, true
}

func (p *Provider) Preview(ctx context.Context, inputs operatorcapability.InputSet) (operatorcapability.Preview, error) {
	_ = ctx
	sink, ok := inputs.Text("sink")
	if !ok || strings.TrimSpace(sink) == "" {
		return operatorcapability.Preview{}, fmt.Errorf("escrow destination is required")
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(sink)), "s3://") {
		return operatorcapability.Preview{
			CapabilityID: CapabilityID,
			PlanID:       operatorcapability.StableIdempotencyKey(CapabilityID, map[string]json.RawMessage{"sink": json.RawMessage(fmt.Sprintf("%q", sink))}),
			State:        operatorcapability.StateReadyToPreview,
			Candidates:   []operatorcapability.Candidate{{ID: sink, Kind: "s3-compatible", Label: sink, Location: sink, Status: "pending-validation", Risk: "object-store authority and repository containment are revalidated during apply"}},
			Mutations:    []operatorcapability.Mutation{{ID: "encrypted-root-copy", Summary: "write and read back the encrypted credential-store root object", Reversible: true}, {ID: "recovery-bundle", Summary: "write and verify the encrypted recovery bundle object", Reversible: true}, {ID: "evidence-and-schedule", Summary: "record metadata-only evidence and install the native refresh schedule", Reversible: true}},
			Remediation:  "Review the object-store reference, endpoint, and exact mutations before applying.",
			ExpiresAt:    p.now().UTC().Add(10 * time.Minute),
		}, nil
	}
	status, err := p.Discover(context.Background())
	if err != nil {
		return operatorcapability.Preview{}, err
	}
	selected := findCandidate(status.Candidates, sink)
	if selected == nil {
		return operatorcapability.Preview{}, fmt.Errorf("escrow destination %q is not a discovered candidate", sink)
	}
	if selected.Status != "ready" {
		return operatorcapability.Preview{}, fmt.Errorf("escrow destination is not ready: %s", selected.Remediation)
	}
	return operatorcapability.Preview{
		CapabilityID: CapabilityID,
		PlanID:       operatorcapability.StableIdempotencyKey(CapabilityID, map[string]json.RawMessage{"sink": json.RawMessage(fmt.Sprintf("%q", sink))}),
		State:        operatorcapability.StateReadyToPreview,
		Candidates:   []operatorcapability.Candidate{*selected},
		Mutations: []operatorcapability.Mutation{
			{ID: "encrypted-root-copy", Summary: "write and read back the encrypted credential-store root copy", Reversible: true},
			{ID: "recovery-bundle", Summary: "write and verify an encrypted recovery bundle", Reversible: true},
			{ID: "evidence-and-schedule", Summary: "record metadata-only evidence and install the native refresh schedule", Reversible: true},
		},
		Remediation: "Review the selected sink and confirm the exact mutations before applying.",
		ExpiresAt:   p.now().UTC().Add(10 * time.Minute),
	}, nil
}

func (p *Provider) Apply(ctx context.Context, inputs operatorcapability.InputSet) (operatorcapability.Result, error) {
	_ = ctx
	sink, ok := inputs.Text("sink")
	if !ok || strings.TrimSpace(sink) == "" {
		return escrowFailure("missing_sink", "select a validated escrow destination")
	}
	passphrase, ok := inputs.Text("recovery_passphrase")
	if !ok || strings.TrimSpace(passphrase) == "" {
		return escrowFailure("missing_recovery_passphrase", "provide the recovery-bundle passphrase through onboarding")
	}
	isS3 := strings.HasPrefix(strings.ToLower(strings.TrimSpace(sink)), "s3://")
	var selected *operatorcapability.Candidate
	var s3Options securestore.S3CopyOptions
	var err error
	if isS3 {
		s3Options, err = p.objectStoreOptions(inputs)
		if err != nil {
			return escrowFailure("object_store_authority_invalid", "provide a credential identity and region; secret values remain in the credential authority")
		}
	} else {
		status, discoverErr := p.Discover(context.Background())
		if discoverErr != nil {
			return escrowFailure("discovery_failed", discoverErr.Error())
		}
		selected = findCandidate(status.Candidates, sink)
		if selected == nil || selected.Status != "ready" {
			remediation := "re-discover storage candidates and choose a ready destination"
			if selected != nil && selected.Remediation != "" {
				remediation = selected.Remediation
			}
			return escrowFailure("unsafe_sink", remediation)
		}
	}
	store, err := p.describeStore()
	if err != nil || !store.Initialized {
		return escrowFailure("credential_store_unavailable", "initialize or unlock the credential store before applying escrow")
	}
	entries, err := credentialinventory.Collect(p.root)
	if err != nil {
		return escrowFailure("credential_inventory_unavailable", "credential coverage could not be established")
	}
	if len(entries.RequiredAbsent) > 0 {
		return operatorcapability.Result{CapabilityID: CapabilityID, State: operatorcapability.StateDegraded, Outcome: "required_credentials_absent", Retryable: true, ErrorCode: "required_credentials_absent", Remediation: "provision every required credential before claiming complete recovery coverage"}, nil
	}
	if len(entries.Entries) == 0 {
		return escrowFailure("no_configured_credentials", "configure at least one declared credential before exporting recovery")
	}
	repositoryRoots, err := p.repositoryRoots()
	if err != nil {
		return escrowFailure("repository_inventory_unavailable", "backup repository roots could not be validated")
	}
	copyReceiptPath := filepath.Join(p.home, "state", "credential-store-copy.json")
	var copyStatus securestore.CopyStatus
	if isS3 {
		copyStatus, err = securestore.CopyStoreS3(store.Path, sink, copyReceiptPath, s3Options)
	} else {
		rootSink := filepath.Join(selected.Location, "vrooli-credential-escrow", "root-copy")
		copyStatus, err = securestore.CopyStoreWithPolicy(store.Path, rootSink, copyReceiptPath, securestore.CopyPolicy{RepositoryPaths: repositoryRoots, ProtectedRoots: []string{p.home}, RequireIndependentDevice: true})
	}
	if err != nil {
		return escrowFailure("root_copy_failed", "the encrypted root copy was not verified; correct the sink and retry")
	}

	bundlePath := strings.TrimRight(sink, "/") + "/recovery/credentials.bundle.json"
	if !isS3 {
		bundlePath = filepath.Join(selected.Location, "vrooli-credential-escrow", "recovery", "credentials.bundle.json")
	}
	var bundle []byte
	if isS3 {
		authority, authorityErr := credentialauthority.DefaultAuthority()
		if authorityErr != nil {
			return escrowFailure("recovery_authority_unavailable", "credential authority is unavailable")
		}
		bundle, err = authority.ExportRecovery(entries.Entries, passphrase)
		if err == nil {
			_, _, err = securestore.UploadS3Artifact(sink, "recovery/credentials.bundle.json", bundle, s3Options)
			bundlePath = strings.TrimRight(sink, "/") + "/recovery/credentials.bundle.json"
		}
	} else {
		bundlePath = filepath.Join(selected.Location, "vrooli-credential-escrow", "recovery", "credentials.bundle.json")
		bundle, err = ensureRecoveryBundle(bundlePath, entries.Entries, passphrase)
	}
	if err != nil {
		return escrowFailure("recovery_bundle_failed", "the recovery bundle was not readable with the supplied passphrase; preserve any existing bundle and retry")
	}
	manifest, err := credentialauthority.InspectRecovery(bundle, passphrase)
	if err != nil {
		return escrowFailure("recovery_verification_failed", "recovery bundle read-back verification failed")
	}
	bundleChecksum := sha256.Sum256(bundle)
	scheduleConfigPath := filepath.Join(p.home, "config", "credential-store-copy.json")
	copyConfig := securestore.CopyConfig{Enabled: true, Sink: sink, Interval: securestore.DefaultCopyInterval}
	if isS3 {
		copyConfig.Sink = sink
		copyConfig.ObjectStoreCredentialID = inputOrDefault(inputs, "object_store_credential_identity", "")
		copyConfig.ObjectStoreRegion = inputOrDefault(inputs, "object_store_region", "")
		copyConfig.ObjectStoreEndpoint = inputOrDefault(inputs, "object_store_endpoint", "")
		copyConfig.ObjectStoreAccessKeyField = inputOrDefault(inputs, "object_store_access_key_field", "s3-access-key-id")
		copyConfig.ObjectStoreSecretKeyField = inputOrDefault(inputs, "object_store_secret_key_field", "s3-secret-access-key")
		copyConfig.ObjectStoreSessionField = inputOrDefault(inputs, "object_store_session_field", "s3-session-token")
	}
	if err := securestore.WriteCopyConfig(scheduleConfigPath, copyConfig); err != nil {
		return escrowFailure("copy_configuration_failed", "verified artifacts exist but the refresh configuration was not committed")
	}
	executable, err := os.Executable()
	if err != nil {
		return escrowFailure("schedule_executable_failed", "the control-plane executable could not be resolved for scheduling")
	}
	schedule, scheduleErr := securestore.InstallCopySchedule(executable, copyConfig.Interval, true)
	copyStatus.ScheduleState = schedule.State
	if schedule.Remediation != "" {
		copyStatus.Remediation = schedule.Remediation
	}
	if err := securestore.WriteCopyReceipt(copyReceiptPath, copyStatus); err != nil {
		return escrowFailure("copy_evidence_failed", "the root copy exists but its metadata receipt could not be updated")
	}
	now := p.now().UTC()
	recoveryReceipt := credentialauthority.RecoveryReceipt{
		ArtifactIdentity: artifactIdentity(bundlePath),
		SourceGeneration: copyStatus.Generation,
		Checksum:         hex.EncodeToString(bundleChecksum[:]),
		VerifiedAt:       now,
		Verification:     "decrypt-readback",
		SinkIdentity:     copyStatus.SinkIdentity,
		ScheduleState:    schedule.State,
		Remediation:      schedule.Remediation,
	}
	if err := credentialauthority.WriteRecoveryReceiptWithMetadata(filepath.Join(p.home, "state"), bundlePath, manifest.Entries, recoveryReceipt, now); err != nil {
		return escrowFailure("recovery_evidence_failed", "the recovery bundle exists but its metadata receipt could not be written")
	}
	state := operatorcapability.StateReady
	outcome := "escrow_ready"
	remediation := "root copy and recovery bundle verified"
	if scheduleErr != nil || !schedule.Supported || schedule.State != "ready" {
		state = operatorcapability.StateDegraded
		outcome = "escrow_ready_schedule_degraded"
		remediation = schedule.Remediation
		if remediation == "" {
			remediation = "artifacts are verified; refresh manually until the native schedule is available"
		}
	}
	return operatorcapability.Result{
		CapabilityID: CapabilityID, State: state, Outcome: outcome, Retryable: true, Remediation: remediation,
		Mutations:   []operatorcapability.Mutation{{ID: "encrypted-root-copy", Summary: "verified encrypted credential-store root copy", Reversible: true}, {ID: "recovery-bundle", Summary: fmt.Sprintf("verified recovery bundle covering %d entries", len(manifest.Entries)), Reversible: true}, {ID: "schedule", Summary: "recorded native schedule state", Reversible: true}},
		Evidence:    []operatorcapability.EvidenceReference{{Kind: "encrypted-root-copy", ArtifactIdentity: artifactIdentity(copyStatus.Path), SourceGeneration: copyStatus.Generation, Checksum: copyStatus.Checksum, ObservedAt: copyStatus.VerifiedAt, Verified: copyStatus.Verification == "readback", Remediation: copyStatus.Remediation}, {Kind: "recovery-bundle", ArtifactIdentity: artifactIdentity(bundlePath), SourceGeneration: copyStatus.Generation, Checksum: hex.EncodeToString(bundleChecksum[:]), Coverage: recoveryCoverage(manifest.Entries), ObservedAt: now, Verified: true, Remediation: schedule.Remediation}, {Kind: "schedule", ArtifactIdentity: schedule.Provider, ObservedAt: schedule.UpdatedAt, Verified: schedule.State == "ready", Remediation: schedule.Remediation}},
		CompletedAt: now,
	}, nil
}

func escrowFailure(code, remediation string) (operatorcapability.Result, error) {
	return operatorcapability.Result{CapabilityID: CapabilityID, State: operatorcapability.StateRetryableFailure, Outcome: code, Retryable: true, ErrorCode: code, Remediation: remediation}, nil
}

func ensureRecoveryBundle(path string, entries []credentialauthority.RecoveryEntry, passphrase string) ([]byte, error) {
	if existing, err := os.ReadFile(path); err == nil {
		manifest, inspectErr := credentialauthority.InspectRecovery(existing, passphrase)
		if inspectErr != nil || !coversAll(manifest.Entries, entries) {
			return nil, fmt.Errorf("existing recovery bundle cannot be adopted")
		}
		return existing, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	bundle, err := credentialauthority.DefaultAuthority()
	if err != nil {
		return nil, err
	}
	data, err := bundle.ExportRecovery(entries, passphrase)
	if err != nil {
		return nil, err
	}
	if err := atomicSecretFile(path, data); err != nil {
		return nil, err
	}
	return data, nil
}

func atomicSecretFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".recovery-bundle-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer func() { _ = temp.Close(); _ = os.Remove(tempPath) }()
	if err := temp.Chmod(0o600); err != nil {
		return err
	}
	if _, err := temp.Write(data); err != nil {
		return err
	}
	if err := temp.Sync(); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func coversAll(have, want []credentialauthority.RecoveryEntry) bool {
	seen := make(map[string]struct{}, len(have))
	for _, entry := range have {
		seen[string(entry.Identity)+":"+entry.Field] = struct{}{}
	}
	for _, entry := range want {
		if _, ok := seen[string(entry.Identity)+":"+entry.Field]; !ok {
			return false
		}
	}
	return true
}

func recoveryCoverage(entries []credentialauthority.RecoveryEntry) []string {
	coverage := make([]string, 0, len(entries))
	for _, entry := range entries {
		coverage = append(coverage, string(entry.Identity)+":"+entry.Field)
	}
	sort.Strings(coverage)
	return coverage
}

func artifactIdentity(path string) string {
	hash := sha256.Sum256([]byte(filepath.Clean(path)))
	return hex.EncodeToString(hash[:])
}

func (p *Provider) repositoryRoots() ([]string, error) {
	entries, err := p.loadRepositories()
	if err != nil {
		return nil, err
	}
	roots := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Backend == kopiaregistry.BackendFilesystem && strings.TrimSpace(entry.Path) != "" {
			roots = append(roots, entry.Path)
		}
	}
	sort.Strings(roots)
	return roots, nil
}

func findCandidate(candidates []operatorcapability.Candidate, value string) *operatorcapability.Candidate {
	value = strings.TrimSpace(value)
	for _, candidate := range candidates {
		if candidate.ID == value || candidate.Location == value {
			copy := candidate
			return &copy
		}
	}
	return nil
}

func (p *Provider) objectStoreOptions(inputs operatorcapability.InputSet) (securestore.S3CopyOptions, error) {
	identityRaw, ok := inputs.Text("object_store_credential_identity")
	if !ok || strings.TrimSpace(identityRaw) == "" {
		return securestore.S3CopyOptions{}, fmt.Errorf("object-store credential identity is required")
	}
	identity, err := credentialauthority.ParseIdentity(identityRaw)
	if err != nil {
		return securestore.S3CopyOptions{}, err
	}
	region, ok := inputs.Text("object_store_region")
	if !ok || strings.TrimSpace(region) == "" {
		return securestore.S3CopyOptions{}, fmt.Errorf("object-store region is required")
	}
	accessField := inputOrDefault(inputs, "object_store_access_key_field", "s3-access-key-id")
	secretField := inputOrDefault(inputs, "object_store_secret_key_field", "s3-secret-access-key")
	sessionField := inputOrDefault(inputs, "object_store_session_field", "s3-session-token")
	authority, err := credentialauthority.DefaultAuthority()
	if err != nil {
		return securestore.S3CopyOptions{}, err
	}
	accessKey, err := authority.Resolve(identity, accessField)
	if err != nil {
		return securestore.S3CopyOptions{}, err
	}
	secretKey, err := authority.Resolve(identity, secretField)
	if err != nil {
		return securestore.S3CopyOptions{}, err
	}
	sessionToken, sessionErr := authority.Resolve(identity, sessionField)
	if sessionErr != nil && !errors.Is(sessionErr, credentialauthority.ErrUnconfigured) {
		return securestore.S3CopyOptions{}, sessionErr
	}
	repositories, err := p.loadRepositories()
	if err != nil {
		return securestore.S3CopyOptions{}, err
	}
	repositorySinks := make([]string, 0, len(repositories))
	for _, repository := range repositories {
		if repository.Backend != kopiaregistry.BackendS3 || strings.TrimSpace(repository.Bucket) == "" {
			continue
		}
		sink := "s3://" + strings.TrimSpace(repository.Bucket)
		if strings.TrimSpace(repository.Prefix) != "" {
			sink += "/" + strings.Trim(repository.Prefix, "/")
		}
		repositorySinks = append(repositorySinks, sink)
	}
	return securestore.S3CopyOptions{Region: region, Endpoint: inputOrDefault(inputs, "object_store_endpoint", ""), Credentials: securestore.ObjectStoreCredentials{AccessKey: accessKey, SecretKey: secretKey, SessionToken: sessionToken}, RepositorySinks: repositorySinks}, nil
}

func inputOrDefault(inputs operatorcapability.InputSet, id, fallback string) string {
	if value, ok := inputs.Text(id); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}
