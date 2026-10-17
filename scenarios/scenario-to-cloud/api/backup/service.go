package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"

	"github.com/vrooli/vrooli/packages/recoverypoint"
)

// Repository is the persistence the service needs; persistence.Repository
// satisfies it.
type Repository interface {
	CreateRecoveryPoint(ctx context.Context, rp *domain.RecoveryPoint) (*domain.RecoveryPoint, error)
	GetRecoveryPoint(ctx context.Context, id string) (*domain.RecoveryPoint, error)
	ListRecoveryPoints(ctx context.Context, deploymentID string) ([]domain.RecoveryPoint, error)
	SetRecoveryPointProtection(ctx context.Context, id string, protected bool, protectedBy []string) (*domain.RecoveryPoint, error)
	DeleteRecoveryPoint(ctx context.Context, id string) error
	CreateRestoreReceipt(ctx context.Context, receipt *domain.RestoreReceipt) (*domain.RestoreReceipt, error)
	ListRestoreReceipts(ctx context.Context, recoveryPointID string) ([]domain.RestoreReceipt, error)
}

// Budgets are the frozen numeric bounds a restore is measured against
// (certification/budgets.json qualification.*).
type Budgets struct {
	FreshHostRestoreSeconds int64 `json:"fresh_host_restore_seconds_max"`
	RecoveryPointSeconds    int64 `json:"backup_recovery_point_seconds_max"`
}

// LoadBudgets reads certification/budgets.json.
func LoadBudgets(path string) (Budgets, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // scenario-owned budgets file
	if err != nil {
		return Budgets{}, err
	}
	var doc struct {
		Qualification Budgets `json:"qualification"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return Budgets{}, err
	}
	if doc.Qualification.FreshHostRestoreSeconds <= 0 || doc.Qualification.RecoveryPointSeconds <= 0 {
		return Budgets{}, fmt.Errorf("budgets %s declare no restore/recovery-point bounds", path)
	}
	return doc.Qualification, nil
}

var identifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// Service owns recovery points for deployments.
type Service struct {
	Repo Repository
	// Root is the directory recovery points are written beneath:
	// <Root>/<deployment-id>/<recovery-point-id>/.
	Root      string
	Keys      recoverypoint.KeyResolver
	Sealer    recoverypoint.Sealer
	Providers recoverypoint.Registry
	Boundary  recoverypoint.Boundary
	// Registrar registers bindings with the backup owner; nil skips
	// registration and records provider_ref empty.
	Registrar Registrar
	// Owner is the owner id bindings are registered under (the scenario id).
	Owner   string
	Budgets Budgets
	Now     func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

// RecoveryPointDir is the canonical directory of one recovery point.
func (s *Service) RecoveryPointDir(deploymentID, recoveryPointID string) string {
	return filepath.Join(s.Root, deploymentID, recoveryPointID)
}

func (s *Service) removeRecoveryPointDir(deploymentID, recoveryPointID string) {
	if strings.TrimSpace(s.Root) == "" || !identifierPattern.MatchString(deploymentID) || !identifierPattern.MatchString(recoveryPointID) {
		return
	}
	_ = os.RemoveAll(s.RecoveryPointDir(deploymentID, recoveryPointID))
}

func (s *Service) boundary() recoverypoint.Boundary {
	if s.Boundary != nil {
		return s.Boundary
	}
	return recoverypoint.ApplicationHooks{}
}

// CaptureRequest describes one recovery point.
type CaptureRequest struct {
	DeploymentID string `json:"deployment_id"`
	// OperationID and Step derive the recovery point id (<operation>-<step>)
	// so one (operation, step) maps to exactly one point; RecoveryPointID
	// overrides it.
	OperationID     string               `json:"operation_id,omitempty"`
	Step            string               `json:"step,omitempty"`
	RecoveryPointID string               `json:"recovery_point_id,omitempty"`
	Bindings        []domain.DataBinding `json:"bindings"`
	// References the point is bound to (never values).
	ReleaseDigest         string   `json:"release_digest"`
	SchemaVersion         string   `json:"schema_version"`
	ConfigurationDigest   string   `json:"configuration_digest"`
	CredentialVersionRefs []string `json:"credential_version_refs"`
	// RecoveryKeyRef is the credential reference (logical_id:field) of the
	// recovery key, resolvable outside the target failure domain.
	RecoveryKeyRef  string `json:"recovery_key_ref"`
	RetentionPolicy string `json:"retention_policy,omitempty"`
	// MigrationPosture is recorded as given, or selected from Predecessor.
	MigrationPosture string            `json:"migration_posture,omitempty"`
	Predecessor      *PredecessorState `json:"predecessor,omitempty"`
	ProtectedBy      []string          `json:"protected_by,omitempty"`
}

func (r CaptureRequest) id() string {
	if r.RecoveryPointID != "" {
		return r.RecoveryPointID
	}
	if r.OperationID != "" && r.Step != "" {
		return r.OperationID + "-" + r.Step
	}
	return ""
}

// Capture registers the bindings with the backup owner, enters each
// binding's consistency boundary, captures through its provider, seals under
// the referenced key and records the recovery point bound to the release,
// schema, configuration and credential-version references. An existing
// point with the same id is returned unchanged.
func (s *Service) Capture(ctx context.Context, req CaptureRequest) (*domain.RecoveryPoint, error) {
	id := req.id()
	if !identifierPattern.MatchString(req.DeploymentID) {
		return nil, apierrors.New(apierrors.CodeInvalidRequest, "deployment id is required")
	}
	if !identifierPattern.MatchString(id) {
		return nil, apierrors.New(apierrors.CodeInvalidRequest, "recovery point id (or operation id and step) is required")
	}
	if strings.TrimSpace(req.RecoveryKeyRef) == "" {
		return nil, apierrors.New(apierrors.CodeInvalidRequest, "recovery_key_ref is required; recovery points are always encrypted")
	}
	if err := ValidateBindings(req.Bindings); err != nil {
		return nil, err
	}
	if s.Keys == nil || s.Providers == nil {
		return nil, apierrors.New(apierrors.CodeBackupProviderUnavailable, "recovery service has no key resolver or provider registry")
	}
	if existing, err := s.Repo.GetRecoveryPoint(ctx, id); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}
	posture := req.MigrationPosture
	if posture == "" && req.Predecessor != nil {
		posture = SelectPosture(*req.Predecessor)
	}
	if posture == "" {
		return nil, apierrors.New(apierrors.CodeInvalidRequest, "migration_posture or predecessor state is required to select the storage posture")
	}
	bindings := append([]domain.DataBinding(nil), req.Bindings...)
	provider := ""
	providerRef := ""
	if s.Registrar != nil {
		provider = DBMTool
		refs := make([]string, 0, len(bindings))
		for i := range bindings {
			ref, err := s.Registrar.Register(ctx, s.Owner, bindings[i])
			if err != nil {
				return nil, err
			}
			bindings[i].ProviderRef = ref
			refs = append(refs, bindings[i].ID+"="+ref)
		}
		providerRef = strings.Join(refs, ",")
	}
	dir := s.RecoveryPointDir(req.DeploymentID, id)
	manifest, err := recoverypoint.Capture(ctx, recoverypoint.CaptureRequest{
		DeploymentID: req.DeploymentID, RecoveryPointID: id, Dir: dir,
		Bindings: EngineBindings(bindings),
		Refs:     recoverypoint.Refs{SchemaVersion: req.SchemaVersion, ConfigurationDigest: req.ConfigurationDigest, CredentialVersionRefs: append([]string{}, req.CredentialVersionRefs...)},
		KeyRef:   req.RecoveryKeyRef, Keys: s.Keys, Sealer: s.Sealer, Providers: s.Providers, Boundary: s.boundary(),
		Provider: provider, ProviderRef: providerRef, RetentionPolicy: req.RetentionPolicy, MigrationPosture: posture, Now: s.now,
	})
	if err != nil {
		return nil, FromEngine(err, recoverypoint.CodeCaptureFailed)
	}
	rp := FromManifest(manifest, dir)
	rp.Bindings = bindings
	rp.OperationID = req.OperationID
	rp.ReleaseDigest = req.ReleaseDigest
	rp.ProtectedBy = append([]string{}, req.ProtectedBy...)
	rp.Protected = len(rp.ProtectedBy) > 0
	stored, err := s.Repo.CreateRecoveryPoint(ctx, rp)
	if err != nil {
		return nil, err
	}
	return stored, nil
}

// FromManifest projects the engine manifest onto the domain record.
func FromManifest(m recoverypoint.Manifest, location string) *domain.RecoveryPoint {
	rp := &domain.RecoveryPoint{
		ID: m.ID, SchemaVersionRecord: domain.RecoveryPointSchemaVersion, DeploymentID: m.DeploymentID,
		BindingIDs: m.BindingIDs(), Bindings: DomainBindings(m.Bindings),
		SchemaVersion: m.Refs.SchemaVersion, ConfigurationDigest: m.Refs.ConfigurationDigest,
		CredentialVersionRefs: append([]string{}, m.Refs.CredentialVersionRefs...),
		ConsistencyToken:      m.ConsistencyToken, Consistency: make([]domain.ConsistencyRecord, 0, len(m.Consistency)),
		CapturedAt: m.CapturedAt, Provider: m.Provider, ProviderRef: m.ProviderRef,
		Checksums: map[string]domain.BindingChecksum{}, Encrypted: m.Encrypted, RecoveryKeyRef: m.KeyRef,
		RetentionPolicy: m.RetentionPolicy, Protected: m.Protected, MigrationPosture: m.MigrationPosture,
		Location: location, ManifestDigest: m.Digest,
	}
	for _, c := range m.Consistency {
		rp.Consistency = append(rp.Consistency, domain.ConsistencyRecord{Binding: c.Binding, Mode: c.Mode, WriteQuiescence: c.WriteQuiescence, Token: c.Token})
	}
	for id, inv := range m.Checksums {
		rp.Checksums[id] = domain.BindingChecksum{Count: inv.Count, Checksum: inv.Checksum, Comparable: inv.Comparable}
	}
	return rp
}

// Invariant is an application-level check run after a restore against the
// restored bindings (for example "the acknowledged-write ledger equals the
// restored rows").
type Invariant func(ctx context.Context, into map[string]string) domain.InvariantResult

// RestoreRequest restores one recovery point into clean targets.
type RestoreRequest struct {
	DeploymentID    string `json:"deployment_id"`
	RecoveryPointID string `json:"recovery_point_id"`
	// TargetRef names the replacement host/target the restore lands on.
	TargetRef string `json:"target_ref"`
	// Into maps binding id to its target locator on the replacement host.
	Into     map[string]string `json:"into"`
	Bindings []string          `json:"bindings,omitempty"`
	// Expect is the inventory the caller expects per binding (the fixture
	// oracle); compared against the recovery point before the restore and
	// against the restored bindings after it.
	Expect     map[string]domain.BindingChecksum `json:"expect,omitempty"`
	Invariants []Invariant                       `json:"-"`
}

// Restore restores and records a receipt for every attempt: refusals
// (corrupt archive, missing key, unclean target) are receipts with outcome
// refused, failures with outcome failed, and success only when every binding
// restored, matched and every invariant passed. RTO is the restore duration;
// recovery-point age is measured at the moment the restore started.
func (s *Service) Restore(ctx context.Context, req RestoreRequest) (*domain.RestoreReceipt, error) {
	rp, err := s.Repo.GetRecoveryPoint(ctx, req.RecoveryPointID)
	if err != nil {
		return nil, err
	}
	if rp == nil || (req.DeploymentID != "" && rp.DeploymentID != req.DeploymentID) {
		return nil, apierrors.New(apierrors.CodeRecoveryPointNotFound, "recovery point not found").WithDetail("recovery_point_id", req.RecoveryPointID)
	}
	if len(req.Into) == 0 {
		return nil, apierrors.New(apierrors.CodeInvalidRequest, "restore targets (into) are required")
	}
	if s.Keys == nil || s.Providers == nil {
		return nil, apierrors.New(apierrors.CodeBackupProviderUnavailable, "recovery service has no key resolver or provider registry")
	}
	started := s.now()
	receipt := &domain.RestoreReceipt{
		ID: uuid.NewString(), RecoveryPointID: rp.ID, DeploymentID: rp.DeploymentID, TargetRef: req.TargetRef,
		StartedAt: started, RecoveryPointAge: started.Sub(rp.CapturedAt), InvariantResults: []domain.InvariantResult{},
		RTOBudgetSeconds: s.Budgets.FreshHostRestoreSeconds, RPOBudgetSeconds: s.Budgets.RecoveryPointSeconds,
	}
	report, restoreErr := recoverypoint.Restore(ctx, recoverypoint.RestoreRequest{
		Dir: rp.Location, Into: req.Into, Bindings: req.Bindings, Keys: s.Keys, Sealer: s.Sealer, Providers: s.Providers, Now: s.now,
	})
	for _, b := range report.Bindings {
		receipt.InvariantResults = append(receipt.InvariantResults,
			domain.InvariantResult{Binding: b.ID, Check: "restored_count", Expected: fmt.Sprint(b.Captured.Count), Observed: fmt.Sprint(b.Restored.Count), Passed: b.Written && b.Matched})
		if want, ok := req.Expect[b.ID]; ok && b.Written {
			receipt.InvariantResults = append(receipt.InvariantResults,
				domain.InvariantResult{Binding: b.ID, Check: "expected_count", Expected: fmt.Sprint(want.Count), Observed: fmt.Sprint(b.Restored.Count), Passed: want.Count == b.Restored.Count})
			if want.Checksum != "" && b.Restored.Comparable {
				receipt.InvariantResults = append(receipt.InvariantResults,
					domain.InvariantResult{Binding: b.ID, Check: "expected_checksum", Expected: want.Checksum, Observed: b.Restored.Checksum, Passed: want.Checksum == b.Restored.Checksum})
			}
		}
	}
	var typed *apierrors.Error
	if restoreErr != nil {
		typed = FromEngine(restoreErr, recoverypoint.CodeRestoreFailed)
		receipt.Outcome = domain.RestoreOutcomeFailed
		switch typed.Code {
		case apierrors.CodeRecoveryPointCorrupt, apierrors.CodeRecoveryKeyUnavailable, apierrors.CodeRestoreTargetNotClean, apierrors.CodeBackupProviderUnavailable:
			receipt.Outcome = domain.RestoreOutcomeRefused
		}
		receipt.ErrorCode, receipt.ErrorMessage = typed.Code, typed.Message
	} else {
		receipt.Outcome = domain.RestoreOutcomeSucceeded
		for _, check := range req.Invariants {
			result := check(ctx, req.Into)
			receipt.InvariantResults = append(receipt.InvariantResults, result)
		}
	}
	for _, result := range receipt.InvariantResults {
		if !result.Passed && receipt.Outcome == domain.RestoreOutcomeSucceeded {
			receipt.Outcome = domain.RestoreOutcomeFailed
			receipt.ErrorCode = apierrors.CodeRestoreFailed
			receipt.ErrorMessage = fmt.Sprintf("invariant %s/%s failed: expected %s, observed %s", result.Binding, result.Check, result.Expected, result.Observed)
			typed = apierrors.New(apierrors.CodeRestoreFailed, receipt.ErrorMessage).WithDetail("binding", result.Binding).WithDetail("check", result.Check)
		}
	}
	receipt.CompletedAt = s.now()
	receipt.MeasuredRTO = receipt.CompletedAt.Sub(receipt.StartedAt)
	receipt.WithinBudgets = receipt.Outcome == domain.RestoreOutcomeSucceeded &&
		(s.Budgets.FreshHostRestoreSeconds <= 0 || receipt.MeasuredRTO <= time.Duration(s.Budgets.FreshHostRestoreSeconds)*time.Second) &&
		(s.Budgets.RecoveryPointSeconds <= 0 || receipt.RecoveryPointAge <= time.Duration(s.Budgets.RecoveryPointSeconds)*time.Second)
	if _, err := s.Repo.CreateRestoreReceipt(ctx, receipt); err != nil {
		return receipt, err
	}
	if typed != nil {
		return receipt, typed.WithDetail("restore_receipt_id", receipt.ID)
	}
	return receipt, nil
}

// VerifyRequest checks a recovery point without restoring it.
type VerifyRequest struct {
	DeploymentID    string
	RecoveryPointID string
	// OpenArtifacts also resolves the key and decrypts every artifact.
	OpenArtifacts bool
	Expect        map[string]domain.BindingChecksum
}

// VerifyReport is the verification outcome.
type VerifyReport struct {
	RecoveryPoint    *domain.RecoveryPoint    `json:"recovery_point"`
	VerifiedAt       time.Time                `json:"verified_at"`
	RecoveryPointAge time.Duration            `json:"recovery_point_age_ns"`
	ArtifactsIntact  bool                     `json:"artifacts_intact"`
	KeyResolved      bool                     `json:"key_resolved"`
	ArtifactsOpened  bool                     `json:"artifacts_opened"`
	Invariants       []domain.InvariantResult `json:"invariants"`
	Outcome          string                   `json:"outcome"`
}

// Verify checks the manifest digest, every sealed artifact and the expected
// inventory; with OpenArtifacts it also proves the key reference resolves.
func (s *Service) Verify(ctx context.Context, req VerifyRequest) (*VerifyReport, error) {
	rp, err := s.Repo.GetRecoveryPoint(ctx, req.RecoveryPointID)
	if err != nil {
		return nil, err
	}
	if rp == nil || (req.DeploymentID != "" && rp.DeploymentID != req.DeploymentID) {
		return nil, apierrors.New(apierrors.CodeRecoveryPointNotFound, "recovery point not found").WithDetail("recovery_point_id", req.RecoveryPointID)
	}
	verifyReq := recoverypoint.VerifyRequest{Dir: rp.Location, Sealer: s.Sealer, Now: s.now}
	if req.OpenArtifacts {
		verifyReq.Keys = s.Keys
	}
	engine, verifyErr := recoverypoint.Verify(ctx, verifyReq)
	report := &VerifyReport{RecoveryPoint: rp, VerifiedAt: engine.VerifiedAt, RecoveryPointAge: engine.RecoveryPointAge, ArtifactsIntact: engine.ArtifactsIntact, KeyResolved: engine.KeyResolved, ArtifactsOpened: engine.ArtifactsOpened, Invariants: []domain.InvariantResult{}, Outcome: domain.RestoreOutcomeFailed}
	if verifyErr != nil {
		return report, FromEngine(verifyErr, recoverypoint.CodeVerifyFailed)
	}
	passed := true
	for _, binding := range sortedKeys(req.Expect) {
		want := req.Expect[binding]
		got, ok := rp.Checksums[binding]
		if !ok {
			report.Invariants = append(report.Invariants, domain.InvariantResult{Binding: binding, Check: "present", Expected: "captured", Observed: "missing"})
			passed = false
			continue
		}
		count := domain.InvariantResult{Binding: binding, Check: "count", Expected: fmt.Sprint(want.Count), Observed: fmt.Sprint(got.Count), Passed: want.Count == got.Count}
		report.Invariants = append(report.Invariants, count)
		passed = passed && count.Passed
		if want.Checksum != "" && got.Comparable {
			sum := domain.InvariantResult{Binding: binding, Check: "checksum", Expected: want.Checksum, Observed: got.Checksum, Passed: want.Checksum == got.Checksum}
			report.Invariants = append(report.Invariants, sum)
			passed = passed && sum.Passed
		}
	}
	if !passed {
		return report, apierrors.New(apierrors.CodeRestoreFailed, "recovery point does not satisfy the expected inventory").WithDetail("recovery_point_id", rp.ID).WithDetail("invariants", report.Invariants)
	}
	report.Outcome = domain.RestoreOutcomeSucceeded
	return report, nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
