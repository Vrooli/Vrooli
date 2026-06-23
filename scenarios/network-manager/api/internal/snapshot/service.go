package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"network-manager/internal/clock"
)

type Service struct {
	repo   Repository
	runner ProbeRunner
	clock  clock.Clock
}

type Config struct {
	Repo   Repository
	Runner ProbeRunner
	Clock  clock.Clock
}

func NewService(cfg Config) *Service {
	s := &Service{repo: cfg.Repo, runner: cfg.Runner, clock: cfg.Clock}
	if s.clock == nil {
		s.clock = clock.System{}
	}
	if s.runner == nil {
		s.runner = RealProbeRunner{}
	}
	return s
}

func (s *Service) Run(ctx context.Context, profile string, dryRun bool) (Snapshot, error) {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		profile = "home"
	}
	results, err := s.runner.Run(ctx, profile)
	if err != nil {
		return Snapshot{}, fmt.Errorf("run probes: %w", err)
	}
	snap := Snapshot{
		Status:    "complete",
		Profile:   profile,
		Summary:   summarize(results),
		Metrics:   metricsFromResults(results),
		Findings:  findingsFromResults(results),
		CreatedAt: s.clock.Now().UTC(),
	}
	if dryRun {
		snap.ID = "snapshot-dry-run"
		snap.Status = "dry_run"
		return snap, nil
	}
	count, err := s.repo.Count(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	if count == 0 {
		snap.Status = "baseline"
		snap.Findings = append([]string{"Baseline anchor: first persisted read-only snapshot for future optimization comparisons."}, snap.Findings...)
	}
	return s.repo.Create(ctx, snap)
}

func (s *Service) List(ctx context.Context) ([]Snapshot, error) {
	return s.repo.List(ctx)
}

func (s *Service) Get(ctx context.Context, id string) (Snapshot, error) {
	return s.repo.Get(ctx, strings.TrimSpace(id))
}

func (s *Service) Export(ctx context.Context, id, format string) (string, string, string, error) {
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = "markdown"
	}
	snap, err := s.Get(ctx, id)
	if err != nil {
		return "", "", "", err
	}
	switch format {
	case "markdown", "md":
		return snap.ID, "markdown", markdownReport(snap), nil
	case "json":
		body, err := json.MarshalIndent(snap, "", "  ")
		if err != nil {
			return "", "", "", fmt.Errorf("marshal snapshot report: %w", err)
		}
		return snap.ID, "json", string(body), nil
	default:
		return "", "", "", fmt.Errorf("unsupported export format %q", format)
	}
}

func metricsFromResults(results []ProbeResult) []Metric {
	out := make([]Metric, 0, len(results))
	for _, r := range results {
		out = append(out, Metric{Name: r.Name, Value: r.Value, Unit: r.Unit, Status: r.Status})
	}
	return out
}

func findingsFromResults(results []ProbeResult) []string {
	var out []string
	for _, r := range results {
		if strings.TrimSpace(r.Finding) != "" {
			out = append(out, r.Finding)
		}
	}
	if len(out) == 0 {
		out = append(out, "All supported read-only probes completed.")
	}
	return out
}

func summarize(results []ProbeResult) string {
	var healthy, degraded, unavailable, failed int
	for _, r := range results {
		switch r.Status {
		case "healthy":
			healthy++
		case "degraded":
			degraded++
		case "unavailable", "unsupported":
			unavailable++
		default:
			failed++
		}
	}
	return fmt.Sprintf("%d healthy, %d degraded, %d unavailable, %d failed probe results.", healthy, degraded, unavailable, failed)
}

func markdownReport(s Snapshot) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Network Snapshot %s\n\n", s.ID)
	fmt.Fprintf(&b, "- Status: %s\n- Profile: %s\n- Created: %s\n- Summary: %s\n\n", s.Status, s.Profile, s.CreatedAt.UTC().Format(time.RFC3339), s.Summary)
	b.WriteString("## Metrics\n\n| Metric | Value | Unit | Status |\n|---|---:|---|---|\n")
	for _, m := range s.Metrics {
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", m.Name, m.Value, m.Unit, m.Status)
	}
	b.WriteString("\n## Findings\n\n")
	for _, f := range s.Findings {
		fmt.Fprintf(&b, "- %s\n", f)
	}
	return b.String()
}
