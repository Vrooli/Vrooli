package execution

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// fixDiscoveryFreshness is how long a per-scenario discovery marker suppresses
// re-triggering an on-demand readiness review. Discovery is a background hint,
// not a hot path, so a generous window avoids hammering git-control-tower when
// many feature items target the same scenario in quick succession.
const fixDiscoveryFreshness = 6 * time.Hour

// fixDiscoveryReviewTimeout bounds an async discovery review so a wedged
// git-control-tower run cannot leak goroutines indefinitely.
const fixDiscoveryReviewTimeout = 10 * time.Minute

// DiscoveryFinding is one issue surfaced by an on-demand readiness review,
// destined to be filed as a fix backlog item.
type DiscoveryFinding struct {
	Dimension string
	Status    string // "red" or "yellow"
	Details   string
}

// RemediationFiler files fix backlog items discovered for a scenario. The
// implementation lives in the backlog package (which owns item creation);
// execution holds only this interface so it never imports backlog (which would
// be a cycle — backlog imports execution to queue runs).
type RemediationFiler interface {
	// FileRemediation persists one fix item per finding for the scenario and
	// returns how many were newly created (idempotent: existing items are
	// skipped, not duplicated).
	FileRemediation(ctx context.Context, scenario string, findings []DiscoveryFinding) (int, error)
}

// SetRemediationFiler wires the filer post-construction (bootstrap), mirroring
// SetActivityLaneReader. Tests can leave it unset; discovery then no-ops.
func (s *Service) SetRemediationFiler(f RemediationFiler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.remediationFiler = f
}

// maybeTriggerDiscovery launches async on-demand readiness discovery for any
// target scenario that has NO open remediation item and no fresh discovery
// marker. It never blocks the queue: the review runs in a goroutine and any
// fix items it files are caught by the gate on the next queue attempt.
//
// Pre-conditions (governance flag, reviewClient/filer presence) are checked by
// the caller via the gate; this method re-checks filer/reviewClient defensively.
func (s *Service) maybeTriggerDiscovery(featureScenarios []string, openItems []openRemediationItem) {
	s.mu.Lock()
	filer := s.remediationFiler
	s.mu.Unlock()
	if filer == nil || s.reviewClient == nil {
		return
	}

	withOpen := make(map[string]struct{})
	for _, ri := range openItems {
		for _, sc := range ri.scenarios {
			withOpen[sc] = struct{}{}
		}
	}

	for _, scenario := range featureScenarios {
		if _, ok := withOpen[scenario]; ok {
			continue // already has remediation work; the gate handles it
		}
		if scenario == s.selfScenarioName {
			continue // never self-discover
		}
		if s.discoveryMarkerFresh(scenario) {
			continue
		}
		// Stamp the marker before launching so concurrent queue calls dedupe.
		if err := s.writeDiscoveryMarker(scenario); err != nil {
			slog.Warn("fix-discovery: failed to write marker", "scenario", scenario, "err", err)
			continue
		}
		go s.runFixDiscovery(scenario)
	}
}

// runFixDiscovery runs a standalone readiness review for one scenario and files
// fix items for each red/yellow dimension. It owns its own bounded context so
// it survives the queue request returning.
func (s *Service) runFixDiscovery(scenario string) {
	ctx, cancel := context.WithTimeout(context.Background(), fixDiscoveryReviewTimeout)
	defer cancel()

	req := ReviewRequest{
		ScenarioName:  scenario,
		ExpectedPaths: []string{"scenarios/" + scenario + "/**"},
	}
	if s.reviewThresholdsProvider != nil {
		if th, thErr := s.reviewThresholdsProvider.LoadReviewThresholds(); thErr == nil {
			req.Thresholds = th
		}
	}

	jobID, err := s.reviewClient.TriggerReview(ctx, req)
	if err != nil {
		slog.Warn("fix-discovery: trigger review failed", "scenario", scenario, "err", err)
		return
	}

	pollInterval := s.finalizationCfg.ReviewPollInterval
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}
	var result *ReviewResult
	for {
		r, done, pollErr := s.reviewClient.PollReview(ctx, jobID)
		if pollErr != nil {
			slog.Warn("fix-discovery: poll review failed", "scenario", scenario, "err", pollErr)
			return
		}
		if done {
			result = r
			break
		}
		select {
		case <-ctx.Done():
			slog.Warn("fix-discovery: review timed out", "scenario", scenario)
			return
		case <-time.After(pollInterval):
		}
	}

	findings := discoveryFindingsFromReview(result)
	if len(findings) == 0 {
		return
	}
	created, err := s.remediationFiler.FileRemediation(ctx, scenario, findings)
	if err != nil {
		slog.Warn("fix-discovery: file remediation failed", "scenario", scenario, "err", err)
		return
	}
	slog.Info("fix-discovery: filed remediation items", "scenario", scenario, "created", created, "findings", len(findings))
}

// discoveryFindingsFromReview maps the red/yellow dimensions of a readiness
// review into discovery findings. Green/skipped dimensions are not actionable.
func discoveryFindingsFromReview(result *ReviewResult) []DiscoveryFinding {
	if result == nil {
		return nil
	}
	var findings []DiscoveryFinding
	for _, dim := range result.Dimensions {
		switch strings.ToLower(strings.TrimSpace(dim.Status)) {
		case "red", "yellow":
			findings = append(findings, DiscoveryFinding{
				Dimension: dim.Name,
				Status:    strings.ToLower(strings.TrimSpace(dim.Status)),
				Details:   dim.Details,
			})
		}
	}
	return findings
}

// discoveryMarkerDir is the minimal per-scenario freshness store. It is NOT a
// general scenario-health table — only a timestamp marker to throttle reviews.
func (s *Service) discoveryMarkerDir() string {
	return filepath.Join(s.dataRoot, ".fix-discovery")
}

func (s *Service) discoveryMarkerPath(scenario string) string {
	// Scenario names from ScenariosFromGlobs are single path segments, but
	// strip any separators defensively so a marker can never escape the dir.
	safe := strings.NewReplacer("/", "_", `\`, "_", "..", "_").Replace(scenario)
	return filepath.Join(s.discoveryMarkerDir(), safe+".json")
}

type discoveryMarker struct {
	RanAt string `json:"ran_at"`
}

// discoveryMarkerFresh reports whether a discovery review ran for the scenario
// within fixDiscoveryFreshness. A missing/unparesable/old marker is stale.
func (s *Service) discoveryMarkerFresh(scenario string) bool {
	data, err := os.ReadFile(s.discoveryMarkerPath(scenario))
	if err != nil {
		return false
	}
	var m discoveryMarker
	if json.Unmarshal(data, &m) != nil {
		return false
	}
	ranAt, err := time.Parse(time.RFC3339, m.RanAt)
	if err != nil {
		return false
	}
	return time.Since(ranAt) < fixDiscoveryFreshness
}

func (s *Service) writeDiscoveryMarker(scenario string) error {
	if err := os.MkdirAll(s.discoveryMarkerDir(), 0o750); err != nil {
		return err
	}
	data, err := json.Marshal(discoveryMarker{RanAt: nowRFC3339()})
	if err != nil {
		return err
	}
	return os.WriteFile(s.discoveryMarkerPath(scenario), data, 0o600)
}
