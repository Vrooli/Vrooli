package eligibility

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/discovery"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"

	"test-genie/internal/orchestrator/workspace"
)

// Storage-health isolation finding codes the routing decision keys off of.
// Both belong to storage-manager's L2 (isolation-safe) rung: their presence
// means test-DB isolation cannot be statically proven, so the routed e2e path
// is not eligible and the playbooks phase must refuse destructive flows
// fail-closed.
const (
	// CodeRoutedSeamsUnwired is emitted (ERROR) when a Go scenario with a
	// relational store has not wired one or more of the four routed test-DB
	// seams (database.Open→*RoutedDB, EnsureSchemas, TestModeMiddleware,
	// devrouting.Register).
	CodeRoutedSeamsUnwired = "ROUTED_SEAMS_UNWIRED"
	// CodeStorageIsolationUnverified is emitted (WARNING) when the API surface
	// is not Go, so isolation cannot be statically verified — the non-Go
	// fail-safe. Advisory in the storage phase but still disqualifying for the
	// routed path (isolation unproven ⟹ no destructive E2E).
	CodeStorageIsolationUnverified = "STORAGE_ISOLATION_UNVERIFIED"
)

// storageHealthProviderScenario is the scenario whose ScenarioValidationService
// owns the storage-isolation verdict.
const storageHealthProviderScenario = "storage-manager"

// defaultStorageCheckTimeout bounds the storage-manager validation RPC. storage
// validation is a fast static analysis (no execution), so a tight bound keeps
// the playbooks eligibility check responsive.
const defaultStorageCheckTimeout = 30 * time.Second

// IsolationFinding is one storage-manager L2 finding that disqualified a scenario
// from the routed path. It carries enough to render a loud, instructive refusal.
type IsolationFinding struct {
	Code        string
	Severity    string
	Message     string
	Location    string
	Remediation string
}

// Eligibility is the outcome of a routing-eligibility check, now sourced from
// storage-manager's L2 (isolation-safe) verdict.
type Eligibility struct {
	// Routed is true when storage-manager statically proved test-DB isolation
	// (no ROUTED_SEAMS_UNWIRED, no STORAGE_ISOLATION_UNVERIFIED). The playbooks
	// phase installs a test pool on the live process (no restart). False means
	// isolation is unproven and destructive playbooks must be refused.
	Routed bool

	// BlockingFindings are the storage-manager L2 isolation findings that
	// disqualified the scenario. Empty when Routed is true.
	BlockingFindings []IsolationFinding

	// Unverified is true when the disqualification was specifically
	// STORAGE_ISOLATION_UNVERIFIED (a non-Go API whose isolation cannot be
	// statically verified) rather than unwired Go seams. Drives a distinct
	// remediation message.
	Unverified bool
}

// StorageValidationClient is the subset of scenario-validation the checker
// needs. Defined here so tests can inject a stub without HTTP.
type StorageValidationClient interface {
	ValidateScenario(context.Context, *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error)
}

// ResolveStorageHealthURL resolves storage-manager's base URL. Tests override it.
var ResolveStorageHealthURL = func(ctx context.Context) (string, error) {
	return discovery.ResolveScenarioURLDefault(ctx, storageHealthProviderScenario)
}

// NewStorageValidationClient builds the storage-manager validation client. Tests
// override it to return a stub.
var NewStorageValidationClient = func(timeout time.Duration, baseURL string) StorageValidationClient {
	return scenariovalidationconnect.NewScenarioValidationServiceClient(&http.Client{Timeout: timeout}, baseURL)
}

// Checker fetches and caches per-scenario eligibility for the lifetime of a
// test-genie run.
type Checker struct {
	timeout time.Duration

	mu    sync.Mutex
	cache map[string]Eligibility
}

// NewChecker returns a Checker that queries storage-manager for the isolation
// verdict.
func NewChecker() *Checker {
	return &Checker{
		timeout: defaultStorageCheckTimeout,
		cache:   map[string]Eligibility{},
	}
}

// Check returns the eligibility of `scenario` for the routed path by querying
// storage-manager's ScenarioValidationService and inspecting its L2 isolation
// findings. A storage-manager failure (unreachable, RPC error) is returned as an
// error — the caller treats "isolation cannot be verified" as "not eligible".
func (c *Checker) Check(ctx context.Context, scenario string, mapping workspace.Mapping) (Eligibility, error) {
	c.mu.Lock()
	if cached, ok := c.cache[scenario]; ok {
		c.mu.Unlock()
		return cached, nil
	}
	c.mu.Unlock()

	baseURL, err := ResolveStorageHealthURL(ctx)
	if err != nil {
		return Eligibility{}, fmt.Errorf("resolve %s URL: %w", storageHealthProviderScenario, err)
	}
	if strings.TrimSpace(baseURL) == "" {
		return Eligibility{}, fmt.Errorf("%s base URL is empty", storageHealthProviderScenario)
	}

	resp, err := NewStorageValidationClient(c.timeout, baseURL).ValidateScenario(ctx, connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: scenario,
		Path:     strings.TrimSpace(mapping.PhysicalScenarioDir),
	}))
	if err != nil {
		return Eligibility{}, fmt.Errorf("%s validation RPC failed: %w", storageHealthProviderScenario, err)
	}
	if resp == nil || resp.Msg == nil {
		return Eligibility{}, fmt.Errorf("%s returned an empty validation response", storageHealthProviderScenario)
	}

	elig := decideFromAssessment(resp.Msg.GetAssessment())

	c.mu.Lock()
	c.cache[scenario] = elig
	c.mu.Unlock()
	return elig, nil
}

// Invalidate drops the cached eligibility for a single scenario so the next
// Check re-fetches. Used by the playbooks-phase claim defer so a successive run
// in the same test-genie process picks up code fixes made between runs.
func (c *Checker) Invalidate(scenario string) {
	c.mu.Lock()
	delete(c.cache, scenario)
	c.mu.Unlock()
}

// decideFromAssessment projects a storage-manager MaturityAssessment onto a
// routing-eligibility decision: routed-eligible IFF the L2 isolation rung is
// clean (no ROUTED_SEAMS_UNWIRED and no STORAGE_ISOLATION_UNVERIFIED).
func decideFromAssessment(a *commonv1.MaturityAssessment) Eligibility {
	elig := Eligibility{Routed: true}
	for _, f := range a.GetFindings() {
		if f == nil {
			continue
		}
		code := strings.TrimSpace(f.GetCode())
		switch code {
		case CodeRoutedSeamsUnwired, CodeStorageIsolationUnverified:
			elig.Routed = false
			if code == CodeStorageIsolationUnverified {
				elig.Unverified = true
			}
			elig.BlockingFindings = append(elig.BlockingFindings, IsolationFinding{
				Code:        code,
				Severity:    f.GetSeverity(),
				Message:     firstNonEmpty(f.GetMessage(), f.GetTitle()),
				Location:    f.GetLocation(),
				Remediation: f.GetRemediation(),
			})
		}
	}
	return elig
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
