// Package durablebackup adapts the public data-backup-manager evidence

// surface into the provider-neutral capability catalog. It reports state only;
// backup, restore, and drill execution remain owned by data-backup-manager.
package durablebackup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/operatorcapability"
)

const (
	providerVerified            = "verified"
	defaultDrillFreshnessWindow = 7 * 24 * time.Hour
)

const (
	providerParameterA = 4
	providerParameterB = 50
)

const (
	CapabilityID = "durable-backup-evidence"
	Owner        = "data-backup-manager"
)

// productionRequestTimeout bounds each evidence request when the caller is
// a control-plane CLI with no request-scoped deadline. A degraded DBM must
// produce a degraded capability status, not hang onboarding indefinitely.
var productionRequestTimeout = tuning.ServiceHealthTimeout()

// DrillEvidence is the metadata needed to prove that a recovery drill
// reached a verified restore and produced a checksum. It intentionally omits
// source locators and all artifact contents.
type DrillEvidence struct {
	ID         string
	SnapshotID string
	RestoreID  string
	Status     string
	Checksum   string
	ObservedAt time.Time
}

// SourceCoverage keeps the owner-produced identity of each protected source
// alongside its readiness facts. Counts alone are not sufficient evidence:
// two different source sets can have the same count.
type SourceCoverage struct {
	ID             string
	Owner          string
	Name           string
	Planned        bool
	BackedUp       bool
	Verified       bool
	LastSuccessAt  time.Time
	LastVerifiedAt time.Time
}

// Evidence is the normalized public evidence returned by the DBM API.
type Evidence struct {
	Registered  int
	Recommended int
	Sensitive   int
	Planned     int
	BackedUp    int
	Verified    int
	Sources     []SourceCoverage
	Drill       DrillEvidence
}

// Fetcher is a narrow seam around the public DBM coverage/drill APIs.
type Fetcher func(context.Context) (Evidence, error)

type Provider struct {
	now                  func() time.Time
	fetch                Fetcher
	resolvePort          func() string
	drillFreshnessWindow time.Duration
}

func NewProvider() *Provider {
	provider := &Provider{now: time.Now, resolvePort: cliutil.DetectPortFromVrooli("data-backup-manager", "API_PORT"), drillFreshnessWindow: defaultDrillFreshnessWindow}
	provider.fetch = func(ctx context.Context) (Evidence, error) { return fetchProduction(ctx, provider.resolvePort) }
	return provider
}

// NewProviderWithFetcher is used by contract tests and by embedding callers
// that already have a typed DBM API client.
func NewProviderWithFetcher(fetch Fetcher) *Provider {
	return &Provider{now: time.Now, fetch: fetch, resolvePort: func() string { return "" }, drillFreshnessWindow: defaultDrillFreshnessWindow}
}

// NewProviderWithFetcherAndFreshness is the deterministic seam for callers
// that have a plan-specific drill policy. Production discovery uses the
// conservative default until DBM exposes the selected plan's policy in its
// public evidence contract.
func NewProviderWithFetcherAndFreshness(fetch Fetcher, maxAge time.Duration) *Provider {
	provider := NewProviderWithFetcher(fetch)
	if maxAge > 0 {
		provider.drillFreshnessWindow = maxAge
	}
	return provider
}

func (p *Provider) Descriptor() operatorcapability.Descriptor {
	return operatorcapability.Descriptor{
		Version: operatorcapability.ContractVersion, ID: CapabilityID, Owner: Owner,
		Scope: "durable backup evidence", Purpose: "show whether the backup owner has current coverage and recovery evidence",
		Sensitivity: operatorcapability.SensitivityOperator, Disposition: operatorcapability.DispositionProtected,
		Provenance:  operatorcapability.PermissionProvenance{Requester: "onboarding operator", Scope: "data-backup-manager evidence for this host", GrantSource: "owner-reported typed evidence", RevocationLimit: "onboarding cannot revoke or mutate backup policy; use data-backup-manager"},
		Lifecycle:   operatorcapability.Lifecycle{Preview: true, Apply: true, Verify: true, Recovery: "start data-backup-manager and complete its owner-controlled recovery drill"},
		Title:       "Durable backup and recovery evidence",
		Description: "Read the backup coverage and verified recovery-drill evidence owned by data-backup-manager.",
		Risk:        "This is evidence only. It never starts a backup, restore, or drill from onboarding.",
		Policy:      operatorcapability.Policy{Idempotent: true, Retryable: true, Remediation: "Start data-backup-manager and resolve its reported destination, coverage, or drill remediation."},
		Evidence: operatorcapability.EvidenceContract{
			Kinds:          []string{"durable-backup-coverage", "recovery-drill"},
			RequiredFields: []string{"artifact_identity", "coverage", "checksum", "observed_at", providerVerified},
			SecretFree:     true,
			Freshness:      "coverage and the latest verified drill must be current in data-backup-manager",
		},
		Remediation: "Start data-backup-manager, make its approved destination writable and separate, then complete a backup and recovery drill.",
	}
}

func (p *Provider) Discover(ctx context.Context) (operatorcapability.Status, error) {
	if p == nil || p.fetch == nil {
		return operatorcapability.Status{}, fmt.Errorf("durable backup evidence provider is not configured")
	}
	evidence, err := p.fetch(ctx)
	if err != nil {
		return operatorcapability.Status{
			Descriptor:  p.Descriptor(),
			State:       operatorcapability.StateDegraded,
			Remediation: fmt.Sprintf("data-backup-manager evidence is unavailable: %v", err),
			UpdatedAt:   p.now().UTC(),
		}, nil
	}

	now := p.now().UTC()
	status := operatorcapability.Status{Descriptor: p.Descriptor(), State: operatorcapability.StateReady, UpdatedAt: now}
	drillFresh := p.drillIsFresh(evidence.Drill, now)
	status.Evidence = p.evidenceReferences(evidence, now, drillFresh)
	remediation := make([]string, 0, providerParameterA)
	if evidence.Registered == 0 {
		remediation = append(remediation, "register the intended durable sources")
	}
	if evidence.Recommended > 0 {
		remediation = append(remediation, fmt.Sprintf("review %d unregistered non-sensitive durable source(s)", evidence.Recommended))
	}
	if evidence.Planned < evidence.Registered {
		remediation = append(remediation, "bind every registered durable source to an enabled backup plan")
	}
	if !p.sourceCoverageComplete(evidence, now) {
		remediation = append(remediation, fmt.Sprintf("complete a current successful backup and verified restore for every planned source; source evidence must be newer than %s", p.drillFreshnessWindow))
	}
	if evidence.Sensitive > 0 {
		remediation = append(remediation, fmt.Sprintf("explicitly review %d sensitive source suggestion(s)", evidence.Sensitive))
	}
	if evidence.Drill.Status != providerVerified || strings.TrimSpace(evidence.Drill.ID) == "" || strings.TrimSpace(evidence.Drill.SnapshotID) == "" || strings.TrimSpace(evidence.Drill.Checksum) == "" {
		remediation = append(remediation, "run a recovery drill; a snapshot alone is not recovery evidence")
	} else if !drillFresh {
		remediation = append(remediation, fmt.Sprintf("latest verified recovery drill is older than %s; run a new drill", p.drillFreshnessWindow))
	}
	if len(remediation) > 0 {
		status.State = operatorcapability.StateDegraded
		status.Remediation = strings.Join(remediation, "; ")
	}
	return status, nil
}

func (p *Provider) evidenceReferences(e Evidence, now time.Time, drillFresh bool) []operatorcapability.EvidenceReference {
	coverageVerified := p.sourceCoverageComplete(e, now)
	coverage := []string{
		fmt.Sprintf("registered:%d", e.Registered),
		fmt.Sprintf("planned:%d", e.Planned),
		fmt.Sprintf("backed_up:%d", e.BackedUp),
		fmt.Sprintf("verified:%d", e.Verified),
	}
	sources := append([]SourceCoverage(nil), e.Sources...)
	sort.SliceStable(sources, func(i, j int) bool {
		return sources[i].ID < sources[j].ID
	})
	for _, source := range sources {
		coverage = append(coverage,
			"source:"+source.ID,
			"source:"+source.ID+":planned:"+fmt.Sprint(source.Planned),
			"source:"+source.ID+":backed_up_at:"+source.LastSuccessAt.UTC().Format(time.RFC3339),
			"source:"+source.ID+":verified_at:"+source.LastVerifiedAt.UTC().Format(time.RFC3339),
		)
	}
	coverageChecksum := sha256.Sum256([]byte(strings.Join(coverage, "\n")))
	refs := []operatorcapability.EvidenceReference{{
		Kind:             "durable-backup-coverage",
		ArtifactIdentity: "data-backup-manager/coverage",
		SourceGeneration: "coverage-" + hex.EncodeToString(coverageChecksum[:]),
		Coverage:         coverage,
		Checksum:         hex.EncodeToString(coverageChecksum[:]),
		ObservedAt:       now.UTC(),
		Verified:         coverageVerified,
	}}
	if e.Drill.ID != "" {
		refs = append(refs, operatorcapability.EvidenceReference{
			Kind:             "recovery-drill",
			ArtifactIdentity: "data-backup-manager/drill/" + e.Drill.ID,
			SourceGeneration: e.Drill.SnapshotID,
			Checksum:         e.Drill.Checksum,
			Coverage:         []string{"drill:" + e.Drill.ID, "snapshot:" + e.Drill.SnapshotID, "restore:" + e.Drill.RestoreID},
			ObservedAt:       e.Drill.ObservedAt,
			Verified:         e.Drill.Status == providerVerified && e.Drill.ID != "" && e.Drill.SnapshotID != "" && e.Drill.Checksum != "" && drillFresh,
			Remediation:      e.Drill.Status,
		})
	}
	return refs
}

// Verify is the owner boundary for recovery evidence. The adapter refreshes
// the owner-produced report, then binds the resulting redacted receipts to
// the exact request context before returning them to the generic registry.
func (p *Provider) Verify(ctx context.Context, request operatorcapability.VerificationRequest) ([]operatorcapability.EvidenceReference, error) {
	status, err := p.Discover(ctx)
	if err != nil {
		return nil, err
	}
	if status.State != operatorcapability.StateReady {
		return nil, fmt.Errorf("durable backup evidence is not ready: %s", status.Remediation)
	}
	now := p.now().UTC()
	receipts := append([]operatorcapability.EvidenceReference(nil), status.Evidence...)
	for index := range receipts {
		receipt := &receipts[index]
		receipt.SchemaVersion = operatorcapability.EvidenceSchemaVersion
		receipt.CapabilityID = request.CapabilityID
		receipt.TargetID = request.TargetID
		receipt.Environment = request.Environment
		receipt.AccountIdentity = request.AccountIdentity
		receipt.Operation = request.Operation
		receipt.Status = "exercised"
		receipt.EffectClass = string(operatorcapability.EffectReadOnly)
		receipt.ExpiresAt = receipt.ObservedAt.Add(p.drillFreshnessWindow)
		if request.CredentialRef != nil {
			ref := *request.CredentialRef
			receipt.CredentialRef = &ref
		}
		if !receipt.FreshAt(now) {
			return nil, fmt.Errorf("durable backup evidence receipt %q is not fresh", receipt.Kind)
		}
	}
	return receipts, nil
}

func (p *Provider) sourceCoverageComplete(e Evidence, now time.Time) bool {
	if e.Registered == 0 || e.Recommended != 0 {
		return false
	}
	if len(e.Sources) == 0 {
		// Aggregate counts intentionally cannot establish source identity or
		// freshness. Current DBM reports one row per registered source; an
		// omitted row is an incomplete owner contract, not legacy evidence.
		return false
	}
	seen := make(map[string]struct{}, len(e.Sources))
	for _, source := range e.Sources {
		if strings.TrimSpace(source.ID) == "" {
			return false
		}
		if _, exists := seen[source.ID]; exists {
			return false
		}
		seen[source.ID] = struct{}{}
		if !source.Planned || !source.BackedUp || !source.Verified || !p.sourceTimestampFresh(source.LastSuccessAt, now) || !p.sourceTimestampFresh(source.LastVerifiedAt, now) {
			return false
		}
	}
	return len(e.Sources) == e.Registered
}

func (p *Provider) sourceTimestampFresh(observedAt, now time.Time) bool {
	if p == nil || p.drillFreshnessWindow <= 0 || observedAt.IsZero() || observedAt.After(now) {
		return false
	}
	return now.Sub(observedAt) <= p.drillFreshnessWindow
}

func (p *Provider) drillIsFresh(drill DrillEvidence, now time.Time) bool {
	if p == nil || p.drillFreshnessWindow <= 0 || drill.Status != providerVerified || strings.TrimSpace(drill.Checksum) == "" || drill.ObservedAt.IsZero() {
		return false
	}
	return !drill.ObservedAt.After(now) && now.Sub(drill.ObservedAt) <= p.drillFreshnessWindow
}

func (p *Provider) Preview(context.Context, operatorcapability.InputSet) (operatorcapability.Preview, error) {
	return operatorcapability.Preview{CapabilityID: CapabilityID, PlanID: CapabilityID, State: operatorcapability.StateReadyToPreview, Remediation: "This capability is read-only evidence owned by data-backup-manager."}, nil
}

func (p *Provider) Apply(ctx context.Context, _ operatorcapability.InputSet) (operatorcapability.Result, error) {
	status, err := p.Discover(ctx)
	if err != nil {
		return operatorcapability.Result{CapabilityID: CapabilityID, State: operatorcapability.StateDegraded, Outcome: "evidence_refresh_failed", Retryable: true, ErrorCode: "evidence_refresh_failed", Remediation: err.Error()}, nil
	}
	return operatorcapability.Result{CapabilityID: CapabilityID, State: status.State, Outcome: "durable_backup_evidence_refreshed", Retryable: true, Remediation: status.Remediation, Evidence: status.Evidence, CompletedAt: p.now().UTC()}, nil
}

func fetchProduction(ctx context.Context, resolvePort func() string) (Evidence, error) {
	port := strings.TrimSpace(resolvePort())
	if port == "" {
		return Evidence{}, fmt.Errorf("data-backup-manager API port is unavailable")
	}
	baseURL := "http://127.0.0.1:" + port
	httpClient := &http.Client{Timeout: productionRequestTimeout}
	var coverageResponse struct {
		Report *struct {
			Summary *struct {
				RegisteredCount  int `json:"registeredCount"`
				RecommendedCount int `json:"recommendedCount"`
				SensitiveCount   int `json:"sensitiveCount"`
				PlannedCount     int `json:"plannedCount"`
				BackedUpCount    int `json:"backedUpCount"`
				VerifiedCount    int `json:"verifiedCount"`
			} `json:"summary"`
			RegisteredTargets []struct {
				ID             string     `json:"id"`
				Owner          string     `json:"owner"`
				Name           string     `json:"name"`
				Planned        bool       `json:"planned"`
				LastSuccessAt  *time.Time `json:"lastSuccessAt"`
				LastVerifiedAt *time.Time `json:"lastVerifiedAt"`
			} `json:"registeredTargets"`
		} `json:"report"`
	}
	err := postJSON(ctx, httpClient, baseURL+"/vrooli.data_backup_manager.v1.coverage.CoverageService/GetCoverageReport", map[string]any{}, &coverageResponse)
	if err != nil {
		return Evidence{}, fmt.Errorf("read data-backup-manager coverage: %w", err)
	}
	if coverageResponse.Report == nil || coverageResponse.Report.Summary == nil {
		return Evidence{}, fmt.Errorf("data-backup-manager returned no coverage summary")
	}
	summary := coverageResponse.Report.Summary
	evidence := Evidence{
		Registered: summary.RegisteredCount, Recommended: summary.RecommendedCount, Sensitive: summary.SensitiveCount,
		Planned: summary.PlannedCount, BackedUp: summary.BackedUpCount, Verified: summary.VerifiedCount,
	}
	for _, source := range coverageResponse.Report.RegisteredTargets {
		row := SourceCoverage{ID: source.ID, Owner: source.Owner, Name: source.Name, Planned: source.Planned}
		if source.LastSuccessAt != nil {
			row.LastSuccessAt = source.LastSuccessAt.UTC()
			row.BackedUp = true
		}
		if source.LastVerifiedAt != nil {
			row.LastVerifiedAt = source.LastVerifiedAt.UTC()
			row.Verified = true
		}
		evidence.Sources = append(evidence.Sources, row)
	}
	var drillResponse struct {
		Drills []struct {
			ID          string     `json:"id"`
			SnapshotID  string     `json:"snapshotId"`
			RestoreID   string     `json:"restoreId"`
			Status      string     `json:"status"`
			RequestedAt *time.Time `json:"requestedAt"`
			StartedAt   *time.Time `json:"startedAt"`
			FinishedAt  *time.Time `json:"finishedAt"`
		} `json:"drills"`
	}
	err = postJSON(ctx, httpClient, baseURL+"/vrooli.data_backup_manager.v1.drills.RecoveryDrillsService/ListDrills", map[string]any{"pageSize": providerParameterB}, &drillResponse)
	if err != nil {
		return Evidence{}, fmt.Errorf("read data-backup-manager recovery drills: %w", err)
	}
	sort.SliceStable(drillResponse.Drills, func(i, j int) bool {
		return drillTime(drillResponse.Drills[i].FinishedAt, drillResponse.Drills[i].StartedAt, drillResponse.Drills[i].RequestedAt).After(drillTime(drillResponse.Drills[j].FinishedAt, drillResponse.Drills[j].StartedAt, drillResponse.Drills[j].RequestedAt))
	})
	if len(drillResponse.Drills) == 0 {
		return evidence, nil
	}
	latest := drillResponse.Drills[0]
	evidence.Drill = DrillEvidence{ID: latest.ID, SnapshotID: latest.SnapshotID, RestoreID: latest.RestoreID, Status: normalizeStatus(latest.Status), ObservedAt: drillTime(latest.FinishedAt, latest.StartedAt, latest.RequestedAt)}
	if evidence.Drill.Status == providerVerified && latest.RestoreID != "" {
		var restoreResponse struct {
			Restore *struct {
				Checksum   string     `json:"checksum"`
				FinishedAt *time.Time `json:"finishedAt"`
			} `json:"restore"`
		}
		err := postJSON(ctx, httpClient, baseURL+"/vrooli.data_backup_manager.v1.restores.RestoresService/GetRestore", map[string]any{"id": latest.RestoreID}, &restoreResponse)
		if err != nil {
			return Evidence{}, fmt.Errorf("read verified restore for drill %q: %w", latest.ID, err)
		}
		if restoreResponse.Restore != nil {
			evidence.Drill.Checksum = restoreResponse.Restore.Checksum
			if restoreResponse.Restore.FinishedAt != nil {
				evidence.Drill.ObservedAt = restoreResponse.Restore.FinishedAt.UTC()
			}
		}
	}
	return evidence, nil
}

func drillTime(finished, started, requested *time.Time) time.Time {
	for _, candidate := range []*time.Time{finished, started, requested} {
		if candidate != nil {
			return candidate.UTC()
		}
	}
	return time.Time{}
}

func normalizeStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	status = strings.TrimPrefix(status, "drill_status_")
	return status
}

func postJSON(ctx context.Context, client *http.Client, endpoint string, requestBody, responseBody any) error {
	body, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("HTTP %s", response.Status)
	}
	return json.NewDecoder(response.Body).Decode(responseBody)
}
