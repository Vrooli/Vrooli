package packaged

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Launcher interface {
	Launch(context.Context, CaptureRequest) (Sample, error)
}

type Service struct {
	store        *Store
	launcher     Launcher
	clock        func() time.Time
	evidenceDir  string
	budgetSource func(context.Context, string) (map[string]float64, error)
}

func NewService(store *Store, launcher Launcher, evidenceDir string) *Service {
	return &Service{store: store, launcher: launcher, clock: time.Now, evidenceDir: evidenceDir}
}

func (s *Service) WithBudgetSource(source func(context.Context, string) (map[string]float64, error)) *Service {
	s.budgetSource = source
	return s
}

func (s *Service) Capture(ctx context.Context, request CaptureRequest) ([]Sample, error) {
	if s == nil || s.store == nil || s.launcher == nil {
		return nil, errors.New("packaged: service is unavailable")
	}
	request.Scenario = strings.TrimSpace(request.Scenario)
	request.Ramp = strings.TrimSpace(request.Ramp)
	request.Strategy = strings.TrimSpace(request.Strategy)
	request.Artifact = strings.TrimSpace(request.Artifact)
	if request.Scenario == "" || request.Ramp != "desktop" || request.Strategy != "electron-sidecar" && request.Strategy != "electron-optimized" || request.Artifact == "" || request.Condition != "cold" && request.Condition != "warm" || request.Runs < 1 || request.Runs > 100 {
		return nil, errors.New("packaged: scenario, desktop ramp, supported Electron strategy, artifact, cold|warm condition, and 1..100 runs are required")
	}
	if request.Strategy == "electron-sidecar" && request.Toggles.Any() {
		return nil, errors.New("packaged: electron-sidecar requires every E3 optimization toggle to be disabled")
	}
	if request.Strategy == "electron-optimized" && !request.Toggles.Any() {
		return nil, errors.New("packaged: electron-optimized requires at least one enabled E3 optimization toggle")
	}
	if request.PlanOnly {
		return nil, nil
	}
	if request.Experiment == "" {
		request.Experiment = fmt.Sprintf("%s-%d", request.Scenario, s.clock().UTC().UnixNano())
	}
	// Every capture performs one discarded warmup and the requested measured
	// samples. Record both in the existing attempt ledger; only measurements
	// enter statistics and reliability acceptance.
	out := make([]Sample, 0, request.Runs)
	for i := 0; i <= request.Runs; i++ {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		launch := request
		if i == 0 {
			launch.Runs = 1
		}
		sample, launchErr := s.launcher.Launch(ctx, launch)
		if launchErr == nil && sample.ToggleAttribution != request.Toggles {
			launchErr = fmt.Errorf("artifact attribution %s does not match requested toggles %s", sample.ToggleAttribution.Identity(), request.Toggles.Identity())
		}
		if launchErr == nil {
			launchErr = sample.Validate()
		}
		attempt := LaunchAttempt{Scenario: request.Scenario, Ramp: request.Ramp, Strategy: request.Strategy, Experiment: request.Experiment, Condition: request.Condition, Toggles: request.Toggles, Attempted: s.clock().UTC(), Succeeded: launchErr == nil, Warmup: i == 0}
		if launchErr != nil {
			attempt.Error = truncate(launchErr.Error(), 240)
		}
		// Request cancellation must stop work, while its outcome remains recordable.
		recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		recordErr := s.store.RecordAttempt(recordCtx, attempt)
		cancel()
		if recordErr != nil {
			return out, fmt.Errorf("packaged: recording attempt: %w", errors.Join(launchErr, recordErr))
		}
		if launchErr != nil {
			return out, fmt.Errorf("packaged: attempt %d/%d (warmup=%t): %w", i, request.Runs, i == 0, launchErr)
		}
		if err := ctx.Err(); err != nil {
			return out, err
		}
		if i == 0 {
			continue
		}
		sample.Scenario, sample.Ramp, sample.Strategy, sample.Condition, sample.ExperimentID = request.Scenario, request.Ramp, request.Strategy, request.Condition, request.Experiment
		if err := s.store.Insert(ctx, sample); err != nil {
			return out, err
		}
		if s.evidenceDir != "" {
			if err := persistEvidence(s.evidenceDir, sample); err != nil {
				return out, err
			}
		}
		out = append(out, sample)
	}
	return out, nil
}

func persistEvidence(root string, sample Sample) error {
	data, err := json.MarshalIndent(sample, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Join(root, safeSegment(sample.ExperimentID), sample.Condition)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("packaged: create evidence directory: %w", err)
	}
	path := filepath.Join(dir, safeSegment(sample.SampleID)+".json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("packaged: write evidence: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("packaged: publish evidence: %w", err)
	}
	return nil
}

func safeSegment(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	clean := strings.Trim(b.String(), ".")
	if clean == "" || clean == ".." {
		return "sample"
	}
	return clean
}

func (s *Service) Report(ctx context.Context, scenario, ramp, strategy, condition, experiment string) (Report, error) {
	if s == nil || s.store == nil {
		return Report{}, errors.New("packaged: service is unavailable")
	}
	samples, err := s.store.List(ctx, scenario, ramp, strategy, condition, experiment)
	if err != nil {
		return Report{}, err
	}
	attempts, err := s.store.ListAttempts(ctx, scenario, ramp, strategy, condition, experiment)
	if err != nil {
		return Report{}, err
	}
	attemptsByCell := make(map[string][]LaunchAttempt)
	for _, attempt := range attempts {
		key := attemptGroupKey(attempt.Scenario, attempt.Ramp, attempt.Strategy, attempt.Condition, attempt.Experiment, attempt.Toggles)
		attemptsByCell[key] = append(attemptsByCell[key], attempt)
	}
	groups := map[identity][]Sample{}
	for _, sample := range samples {
		key := identity{sample.Scenario, sample.Ramp, sample.Strategy, sample.Platform, sample.Condition, sample.HostFingerprint, sample.ArtifactDigest, sample.Display, sample.ProfilerMode, sample.PowerState, sample.ExperimentID, sample.ToggleAttribution}
		groups[key] = append(groups[key], sample)
	}
	keys := make([]identity, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].scenario != keys[j].scenario {
			return keys[i].scenario < keys[j].scenario
		}
		if keys[i].condition != keys[j].condition {
			return keys[i].condition < keys[j].condition
		}
		return keys[i].experiment < keys[j].experiment
	})
	report := Report{SchemaVersion: "packaged-comparison.v1", Cells: make([]ReportCell, 0, len(keys))}
	seenCells := make(map[string]bool)
	for _, key := range keys {
		values := groups[key]
		comparable := key.scenario != "" && key.ramp != "" && key.strategy != "" && key.platform != "" && key.host != "" && key.digest != "" && key.display != "" && key.profiler != "" && key.power != ""
		journeyStatus, journeyReceipts := summarizeJourneyEvidence(values)
		cellKey := attemptGroupKey(key.scenario, key.ramp, key.strategy, key.condition, key.experiment, key.toggles)
		cellAttempts := attemptsByCell[cellKey]
		seenCells[cellKey] = true
		group := ReportCell{Scenario: key.scenario, Ramp: key.ramp, Strategy: key.strategy, Platform: key.platform, Condition: key.condition, ExperimentID: key.experiment, Comparable: comparable, Samples: len(values), Metrics: summarize(values), ArtifactDigest: key.digest, Environment: EnvironmentMetadata{HostFingerprint: key.host, Display: key.display, ProfilerMode: key.profiler, PowerState: key.power}, JourneyParity: journeyStatus, JourneyReceipts: journeyReceipts, Reliability: summarizeReliability(cellAttempts, len(values), false), WarmupReliability: summarizeReliability(cellAttempts, 0, true)}
		group.ToggleAttribution = values[0].ToggleAttribution
		group.AttributionStatus = "unavailable: packaged-launch producer has not recorded optimization toggle values"
		if key.strategy == "electron-sidecar" {
			group.AttributionStatus = "known: electron-sidecar baseline disables E3 optimization toggles"
		} else if key.strategy == "electron-optimized" && group.ToggleAttribution.Any() {
			group.AttributionStatus = "known: packaged-launch producer recorded independent E3 optimization toggles"
		}
		if strings.HasPrefix(journeyStatus, "unavailable") || strings.HasPrefix(journeyStatus, "incomplete") {
			report.MissingData = append(report.MissingData, "journey parity evidence "+journeyStatus+" for "+key.experiment)
		}
		if len(cellAttempts) == 0 {
			report.MissingData = append(report.MissingData, "launch attempt reliability unavailable for "+key.experiment)
		}
		for _, sample := range values[1:] {
			if sample.ToggleAttribution != group.ToggleAttribution {
				report.MissingData = append(report.MissingData, "mixed optimization toggles within group "+key.experiment)
				group.Comparable = false
			}
		}
		if key.host == "" || key.digest == "" || key.display == "" || key.profiler == "" || key.power == "" {
			report.MissingData = append(report.MissingData, "incomplete environment or artifact provenance for "+key.experiment)
		}
		if s.budgetSource != nil {
			budget, budgetErr := s.budgetSource(ctx, key.scenario)
			if budgetErr != nil {
				return Report{}, fmt.Errorf("packaged: read advisory budget for %s: %w", key.scenario, budgetErr)
			}
			group.AdvisoryBudgets = budget
		}
		report.Cells = append(report.Cells, group)
	}
	for cellKey, cellAttempts := range attemptsByCell {
		if seenCells[cellKey] || len(cellAttempts) == 0 {
			continue
		}
		first := cellAttempts[0]
		group := ReportCell{Scenario: first.Scenario, Ramp: first.Ramp, Strategy: first.Strategy, Condition: first.Condition, ExperimentID: first.Experiment, Comparable: false, Metrics: map[string]MetricSummary{}, ToggleAttribution: first.Toggles, AttributionStatus: "known: captured from requested matrix cell", JourneyParity: "unavailable: no valid launch samples", Reliability: summarizeReliability(cellAttempts, 0, false), WarmupReliability: summarizeReliability(cellAttempts, 0, true)}
		report.Cells = append(report.Cells, group)
		report.MissingData = append(report.MissingData, "no valid launch samples for "+first.Scenario+" "+first.Strategy+" "+first.Condition)
	}
	sort.Slice(report.Cells, func(i, j int) bool {
		if report.Cells[i].Scenario != report.Cells[j].Scenario {
			return report.Cells[i].Scenario < report.Cells[j].Scenario
		}
		if report.Cells[i].Condition != report.Cells[j].Condition {
			return report.Cells[i].Condition < report.Cells[j].Condition
		}
		return report.Cells[i].Strategy < report.Cells[j].Strategy
	})
	return report, nil
}

func attemptGroupKey(scenario, ramp, strategy, condition, experiment string, toggles OptimizationToggles) string {
	return strings.Join([]string{scenario, ramp, strategy, condition, experiment, toggles.Identity()}, "\x00")
}

func summarizeReliability(attempts []LaunchAttempt, successfulSamples int, warmup bool) ReliabilityEvidence {
	if len(attempts) == 0 {
		return ReliabilityEvidence{Attempts: successfulSamples, Successful: successfulSamples, Status: "unavailable: launch attempt outcomes were not retained"}
	}
	kind := "measurement"
	if warmup {
		kind = "warmup"
	}
	out := ReliabilityEvidence{}
	for _, attempt := range attempts {
		if attempt.Warmup != warmup {
			continue
		}
		out.Attempts++
		if attempt.Succeeded {
			out.Successful++
		} else {
			out.Failed++
			if len(out.Failures) < 20 {
				out.Failures = append(out.Failures, truncate(attempt.Error, 240))
			}
		}
	}
	if out.Attempts == 0 {
		out.Status = "unavailable: no " + kind + " operations were attempted"
		return out
	}
	if out.Failed == 0 {
		out.Status = fmt.Sprintf("recorded: %d of %d %s operations succeeded", out.Successful, out.Attempts, kind)
	} else {
		out.Status = fmt.Sprintf("failed: %d of %d %s operations failed", out.Failed, out.Attempts, kind)
	}
	return out
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

func summarizeJourneyEvidence(samples []Sample) (string, []JourneyEvidence) {
	if len(samples) == 0 {
		return "unavailable: no launch samples", nil
	}
	receipts := make([]JourneyEvidence, 0, len(samples))
	failed, incomplete := 0, 0
	for _, sample := range samples {
		journey := sample.JourneyEvidence
		if journey == nil || strings.TrimSpace(journey.JourneyID) == "" || strings.TrimSpace(journey.CaptureID) == "" || strings.TrimSpace(journey.Disposition) == "" {
			incomplete++
			continue
		}
		receipts = append(receipts, *journey)
		if journey.Disposition != "pass" {
			failed++
		}
	}
	if len(receipts) == 0 {
		return "unavailable: no valid journey replay receipts", nil
	}
	if incomplete > 0 {
		return fmt.Sprintf("incomplete: %d of %d launches lack journey evidence", incomplete, len(samples)), receipts
	}
	if failed > 0 {
		return fmt.Sprintf("failed: %d of %d journey replays did not pass", failed, len(samples)), receipts
	}
	return fmt.Sprintf("passed: %d of %d journey replays", len(receipts), len(samples)), receipts
}

// RenderMarkdown produces a human-readable view without converting missing
// journey or reliability evidence into a pass.
func RenderMarkdown(report Report) string {
	var b strings.Builder
	b.WriteString("# Packaged experiment comparison\n\n")
	b.WriteString("Schema: `" + report.SchemaVersion + "`\n\n")
	b.WriteString("| Scenario | Strategy | Attribution status | Toggles (readiness / scheme / V8 / fuses) | Platform | Condition | Samples | app_interactive p50 | Journey parity | Reliability | Artifact SHA-256 | Environment (host / display / profiler / power) | Comparable |\n")
	b.WriteString("|---|---|---|---|---|---|---:|---:|---|---|---|---|:---:|\n")
	for _, g := range report.Cells {
		p50 := "unavailable"
		if value := g.Metrics["app_interactive_ms"].P50; value != nil {
			p50 = fmt.Sprintf("%.3f ms", *value)
		}
		t := g.ToggleAttribution
		fmt.Fprintf(&b, "| %s | %s | %s | %t / %t / %t / %t | %s | %s | %d | %s | %s | %s (%d/%d known measurement operations) | `%s` | %s / %s / %s / %s | %t |\n", g.Scenario, g.Strategy, g.AttributionStatus, t.SharedRuntimeReadiness, t.SecureCustomSchemeUI, t.V8SnapshotCodeCache, t.ElectronFuses, g.Platform, g.Condition, g.Samples, p50, g.JourneyParity, g.Reliability.Status, g.Reliability.Successful, g.Reliability.Attempts, g.ArtifactDigest, g.Environment.HostFingerprint, g.Environment.Display, g.Environment.ProfilerMode, g.Environment.PowerState, g.Comparable)
	}
	if len(report.MissingData) > 0 {
		b.WriteString("\n## Missing data\n\n")
		for _, item := range report.MissingData {
			b.WriteString("- " + item + "\n")
		}
	}
	return b.String()
}

type identity struct {
	scenario, ramp, strategy, platform, condition, host, digest, display, profiler, power, experiment string
	toggles                                                                                           OptimizationToggles
}

func summarize(samples []Sample) map[string]MetricSummary {
	values := map[string][]float64{}
	missing := map[string]int{}
	for _, sample := range samples {
		for _, name := range []string{"first_window_ms", "backend_ready_ms", "first_contentful_paint_ms", "app_interactive_ms", "tree_pss_mb", "steady_pss_mb", "idle_cpu_pct", "artifact_bytes", "installed_bytes", "update_bytes"} {
			if name == "app_interactive_ms" || name == "first_contentful_paint_ms" || name == "backend_ready_ms" || name == "first_window_ms" {
				markName := map[string]string{"app_interactive_ms": "app_interactive", "first_contentful_paint_ms": "first_contentful_paint", "backend_ready_ms": "backend_ready", "first_window_ms": "first_window"}[name]
				mark, ok := sample.Mark(markName)
				if !ok || mark.Availability.State != "available" {
					missing[name]++
					continue
				}
				values[name] = append(values[name], float64(mark.ElapsedNs)/1e6)
				continue
			}
			metric, ok := sample.Metric(name)
			if !ok || metric.Availability.State != "available" {
				missing[name]++
				continue
			}
			values[name] = append(values[name], metric.Value)
		}
	}
	out := map[string]MetricSummary{}
	for _, name := range []string{"first_window_ms", "backend_ready_ms", "first_contentful_paint_ms", "app_interactive_ms", "tree_pss_mb", "steady_pss_mb", "idle_cpu_pct", "artifact_bytes", "installed_bytes", "update_bytes"} {
		v := values[name]
		sort.Float64s(v)
		summary := MetricSummary{Available: len(v), Unavailable: missing[name]}
		if len(v) > 0 {
			min, maxv := v[0], v[len(v)-1]
			p50 := nearest(v, .50)
			p95 := nearest(v, .95)
			spread := maxv - min
			summary.Min = &min
			summary.Max = &maxv
			summary.P50 = &p50
			summary.P95 = &p95
			summary.Spread = &spread
		}
		out[name] = summary
	}
	return out
}

func nearest(values []float64, percentile float64) float64 {
	rank := int(math.Ceil(percentile * float64(len(values))))
	if rank < 1 {
		rank = 1
	}
	return values[rank-1]
}

type HTTPLauncher struct {
	BaseURL string
	Client  *http.Client
	Port    func() string
}

func (l *HTTPLauncher) Launch(ctx context.Context, request CaptureRequest) (Sample, error) {
	base := strings.TrimRight(strings.TrimSpace(l.BaseURL), "/")
	if base == "" && l.Port != nil {
		if port := strings.TrimSpace(l.Port()); port != "" {
			base = "http://localhost:" + port
		}
	}
	if base == "" {
		return Sample{}, errors.New("packaged: scenario-to-desktop API URL is not configured")
	}
	body, err := json.Marshal(map[string]any{"scenario": request.Scenario, "artifact_path": request.Artifact, "condition": request.Condition, "strategy": request.Strategy, "experiment_id": request.Experiment, "journey": request.Journey, "optimization_toggles": request.Toggles})
	if err != nil {
		return Sample{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/measured-launch", bytes.NewReader(body))
	if err != nil {
		return Sample{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := l.Client
	if client == nil {
		client = &http.Client{Timeout: 6 * time.Minute}
	}
	response, err := client.Do(req)
	if err != nil {
		return Sample{}, fmt.Errorf("call scenario-to-desktop: %w", err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return Sample{}, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Sample{}, fmt.Errorf("scenario-to-desktop returned %s: %s", response.Status, strings.TrimSpace(string(data)))
	}
	var sample Sample
	if err := json.Unmarshal(data, &sample); err != nil {
		return Sample{}, fmt.Errorf("decode packaged-launch sample: %w", err)
	}
	if err := sample.Validate(); err != nil {
		return Sample{}, err
	}
	return sample, nil
}
