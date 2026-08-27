// Package durablebackup adapts the public data-backup-manager evidence

// surface into the provider-neutral capability catalog. It reports state only;
// backup, restore, and drill execution remain owned by data-backup-manager.
package durablebackup

import (
	"context"
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
	providerVerified = "verified"
)

const (
	providerParameterA = 4
	providerParameterB = 50
)

const (
	CapabilityID = "durable-backup-evidence"
	Owner        = "data-backup-manager"
	// productionRequestTimeout bounds each evidence request when the caller is
	// a control-plane CLI with no request-scoped deadline. A degraded DBM must
	// produce a degraded capability status, not hang onboarding indefinitely.
	productionRequestTimeout = tuning.ServiceHealthTimeout
)

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

// Evidence is the normalized public evidence returned by the DBM API.
type Evidence struct {
	Registered  int
	Recommended int
	Sensitive   int
	Planned     int
	BackedUp    int
	Verified    int
	Drill       DrillEvidence
}

// Fetcher is a narrow seam around the public DBM coverage/drill APIs.
type Fetcher func(context.Context) (Evidence, error)

type Provider struct {
	now         func() time.Time
	fetch       Fetcher
	resolvePort func() string
}

func NewProvider() *Provider {
	provider := &Provider{now: time.Now, resolvePort: cliutil.DetectPortFromVrooli("data-backup-manager", "API_PORT")}
	provider.fetch = func(ctx context.Context) (Evidence, error) { return fetchProduction(ctx, provider.resolvePort) }
	return provider
}

// NewProviderWithFetcher is used by contract tests and by embedding callers
// that already have a typed DBM API client.
func NewProviderWithFetcher(fetch Fetcher) *Provider {
	return &Provider{now: time.Now, fetch: fetch, resolvePort: func() string { return "" }}
}

func (p *Provider) Descriptor() operatorcapability.Descriptor {
	return operatorcapability.Descriptor{
		Version:     operatorcapability.ContractVersion,
		ID:          CapabilityID,
		Owner:       Owner,
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

	status := operatorcapability.Status{Descriptor: p.Descriptor(), State: operatorcapability.StateReady, UpdatedAt: p.now().UTC()}
	status.Evidence = evidenceReferences(evidence, p.now())
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
	if evidence.BackedUp < evidence.Planned || evidence.Verified < evidence.Registered {
		remediation = append(remediation, "complete a successful backup and verified restore for every planned source")
	}
	if evidence.Sensitive > 0 {
		remediation = append(remediation, fmt.Sprintf("explicitly review %d sensitive source suggestion(s)", evidence.Sensitive))
	}
	if evidence.Drill.Status != providerVerified || strings.TrimSpace(evidence.Drill.Checksum) == "" {
		remediation = append(remediation, "run a recovery drill; a snapshot alone is not recovery evidence")
	}
	if len(remediation) > 0 {
		status.State = operatorcapability.StateDegraded
		status.Remediation = strings.Join(remediation, "; ")
	}
	return status, nil
}

func evidenceReferences(e Evidence, now time.Time) []operatorcapability.EvidenceReference {
	coverageVerified := e.Registered > 0 && e.Planned == e.Registered && e.BackedUp == e.Registered && e.Verified == e.Registered && e.Recommended == 0
	refs := []operatorcapability.EvidenceReference{{
		Kind:             "durable-backup-coverage",
		ArtifactIdentity: "data-backup-manager/coverage",
		Coverage:         []string{fmt.Sprintf("registered:%d", e.Registered), fmt.Sprintf("planned:%d", e.Planned), fmt.Sprintf("backed_up:%d", e.BackedUp), fmt.Sprintf("verified:%d", e.Verified)},
		ObservedAt:       now.UTC(),
		Verified:         coverageVerified,
	}}
	if e.Drill.ID != "" {
		refs = append(refs, operatorcapability.EvidenceReference{
			Kind:             "recovery-drill",
			ArtifactIdentity: "data-backup-manager/drill/" + e.Drill.ID,
			SourceGeneration: e.Drill.SnapshotID,
			Checksum:         e.Drill.Checksum,
			ObservedAt:       e.Drill.ObservedAt,
			Verified:         e.Drill.Status == providerVerified && e.Drill.Checksum != "",
			Remediation:      e.Drill.Status,
		})
	}
	return refs
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
