// Package modelpolicydrift owns the scheduled, reporting side of model-policy
// drift detection. The host safeguard remains read-only; this package is the
// control-plane loop that persists the last observation and files typed bugs.
package modelpolicydrift

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/cli-core/agentcatalog"
	"github.com/vrooli/cli-core/upstreamcheck"
)

var runners = []string{"codex", "claude-code", "opencode", "grok"}

type Snapshot struct {
	LastRun    time.Time         `json:"last_run,omitempty"`
	Status     string            `json:"status"`
	Measured   int               `json:"measured"`
	Total      int               `json:"total"`
	Findings   []Finding         `json:"findings,omitempty"`
	LastErrors map[string]string `json:"last_errors,omitempty"`
	// Reported retains published receiver capture IDs (older snapshots may contain timestamps).
	Reported      map[string]string `json:"reported,omitempty"`
	IntervalHours int               `json:"interval_hours"`
}

type Finding struct {
	Runner      string `json:"runner"`
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Role        string `json:"role,omitempty"`
	Model       string `json:"model,omitempty"`
	Message     string `json:"message"`
	Fingerprint string `json:"fingerprint"`
}

type Report struct {
	Finding Finding
	At      time.Time
}

type Reporter interface {
	Report(context.Context, Report) (string, error)
}

type CheckFunc func(context.Context, string, string) ([]agentcatalog.PolicyValidationFinding, agentcatalog.LiveModelCatalog, error)

type Scheduler struct {
	root     string
	path     string
	interval time.Duration
	reporter Reporter
	check    CheckFunc
	now      func() time.Time
	runMu    sync.Mutex
	mu       sync.RWMutex
	snapshot Snapshot
	stop     chan struct{}
	done     chan struct{}
}

func New(root, statePath string, interval time.Duration, reporter Reporter) *Scheduler {
	if interval <= 0 || interval > 14*24*time.Hour {
		interval = 24 * time.Hour
	}
	if strings.TrimSpace(statePath) == "" {
		statePath = filepath.Join(root, ".vrooli", "agent-manager", "model-policy-drift.json")
	}
	s := &Scheduler{root: root, path: statePath, interval: interval, reporter: reporter, check: checkRunner, now: time.Now, stop: make(chan struct{}), done: make(chan struct{})}
	s.snapshot = Snapshot{Status: "not_measured", Total: len(runners), Reported: map[string]string{}, LastErrors: map[string]string{}, IntervalHours: int(interval / time.Hour)}
	s.load()
	return s
}

func (s *Scheduler) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := s.snapshot
	out.Findings = append([]Finding(nil), s.snapshot.Findings...)
	out.LastErrors = cloneMap(s.snapshot.LastErrors)
	out.Reported = cloneMap(s.snapshot.Reported)
	return out
}

func (s *Scheduler) Start(ctx context.Context) {
	go func() {
		defer close(s.done)
		s.run(ctx)
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.run(ctx)
			case <-s.stop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *Scheduler) Stop() {
	select {
	case <-s.stop:
		return
	default:
		close(s.stop)
	}
	select {
	case <-s.done:
	case <-time.After(2 * time.Second):
	}
}

func (s *Scheduler) RunOnce(ctx context.Context) Snapshot {
	s.run(ctx)
	return s.Snapshot()
}

func (s *Scheduler) run(ctx context.Context) {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	now := s.now().UTC()
	findings := make([]Finding, 0)
	errorsByRunner := map[string]string{}
	measured := 0
	reports := make([]Finding, 0)
	for _, runner := range runners {
		path := filepath.Join(s.root, "resources", runner, "model-policy.json")
		observed, _, err := s.check(ctx, runner, path)
		if err != nil {
			errorsByRunner[runner] = err.Error()
			observed = append(observed, agentcatalog.PolicyValidationFinding{Type: "policy_check_unavailable", Severity: "warning", Message: err.Error()})
		} else {
			measured++
		}
		source, _ := os.ReadFile(path)
		revision := fmt.Sprintf("%x", sha256.Sum256(source))[:12]
		unnamed := make([]string, 0)
		for _, item := range observed {
			if item.Type == "unnamed_live_model" {
				unnamed = append(unnamed, item.Model)
			}
		}
		sort.Strings(unnamed)
		if len(unnamed) > 0 {
			digest := fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(unnamed, "\n"))))[:12]
			names := unnamed
			if len(names) > 10 {
				names = names[:10]
			}
			reports = append(reports, Finding{Runner: runner, Type: "unreviewed_live_catalog", Severity: "warning", Message: fmt.Sprintf("%d unnamed live models require owner review (first 10): %s", len(unnamed), strings.Join(names, ", ")), Fingerprint: fmt.Sprintf("model-policy-drift/v2/%s/catalog/%s/%s", runner, revision, digest)})
		}
		for _, item := range observed {
			fingerprint := fmt.Sprintf("model-policy-drift/v2/%s/%s/%s/%s/%s", runner, revision, item.Type, item.Role, item.Model)
			finding := Finding{Runner: runner, Type: item.Type, Severity: item.Severity, Role: item.Role, Model: item.Model, Message: item.Message, Fingerprint: fingerprint}
			findings = append(findings, finding)
			if item.Type != "unnamed_live_model" {
				reports = append(reports, finding)
			}
		}
	}
	status := "healthy"
	if len(errorsByRunner) > 0 {
		status = "warning"
	}
	if measured == 0 {
		status = "not_measured"
	} else {
		for _, item := range findings {
			if item.Severity == "error" {
				status = "critical"
				break
			}
			status = "warning"
		}
	}

	s.mu.Lock()
	previous := s.snapshot
	reported := map[string]string{}
	for _, item := range reports {
		if at, ok := previous.Reported[item.Fingerprint]; ok {
			reported[item.Fingerprint] = at
		}
	}
	s.snapshot = Snapshot{LastRun: now, Status: status, Measured: measured, Total: len(runners), Findings: findings, LastErrors: errorsByRunner, Reported: reported, IntervalHours: int(s.interval / time.Hour)}
	s.persistLocked()
	s.mu.Unlock()

	for _, item := range reports {
		if _, ok := reported[item.Fingerprint]; ok {
			continue
		}
		var err error
		var receipt string
		if s.reporter == nil {
			err = fmt.Errorf("model-policy reporter unavailable")
		} else {
			receipt, err = s.reporter.Report(ctx, Report{Finding: item, At: now})
			if err == nil && strings.TrimSpace(receipt) == "" {
				err = fmt.Errorf("model-policy receiver returned no published capture ID")
			}
		}
		s.mu.Lock()
		if err != nil {
			s.snapshot.LastErrors["report:"+item.Fingerprint] = err.Error()
			if s.snapshot.Status == "healthy" {
				s.snapshot.Status = "warning"
			}
		} else {
			s.snapshot.Reported[item.Fingerprint] = receipt
		}
		s.persistLocked()
		s.mu.Unlock()
	}

}

func (s *Scheduler) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var value Snapshot
	if json.Unmarshal(data, &value) != nil {
		return
	}
	if value.Total == 0 {
		value.Total = len(runners)
	}
	if value.Reported == nil {
		value.Reported = map[string]string{}
	}
	if value.LastErrors == nil {
		value.LastErrors = map[string]string{}
	}
	if value.IntervalHours == 0 {
		value.IntervalHours = int(s.interval / time.Hour)
	}
	s.snapshot = value
}

func (s *Scheduler) persistLocked() {
	err := func() error {
		data, err := json.MarshalIndent(s.snapshot, "", "  ")
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
			return err
		}
		tmp := s.path + ".tmp"
		if err := os.WriteFile(tmp, data, 0o600); err != nil {
			return err
		}
		return os.Rename(tmp, s.path)
	}()
	if err != nil {
		s.snapshot.LastErrors["persistence"] = err.Error()
	}
}

func cloneMap(input map[string]string) map[string]string {
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

// checkRunner keeps catalog and CLI-version maintenance in the same durable
// owner loop. Per-resource source/pin/latest semantics stay in upstream-check.
func checkRunner(ctx context.Context, runner, path string) ([]agentcatalog.PolicyValidationFinding, agentcatalog.LiveModelCatalog, error) {
	findings, live, err := agentcatalog.ValidateCatalogAgainstLive(ctx, runner, path)
	findings = append(findings, checkUpstream(ctx, runner, upstreamcheck.DefaultAggregateRunner)...)
	return findings, live, err
}

func checkUpstream(ctx context.Context, runner string, run upstreamcheck.RunnerFunc) []agentcatalog.PolicyValidationFinding {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	aggregate := upstreamcheck.RunAggregate(ctx, run, []upstreamcheck.AggregateEntry{{Name: runner, CheckCmd: []string{"resource-" + runner, "upstream-check", "--json"}}})
	findings := make([]agentcatalog.PolicyValidationFinding, 0)
	for _, report := range aggregate.Resources {
		switch report.Status {
		case upstreamcheck.StatusBehind:
			findings = append(findings, agentcatalog.PolicyValidationFinding{Type: "upstream_cli_behind", Severity: "warning", Model: report.Latest, Message: fmt.Sprintf("%s selected CLI %s is behind latest %s (provisioning pin %s); qualify resource-%s update and retain rollback before activation", runner, report.Installed, report.Latest, report.Pinned, runner)})
		case upstreamcheck.StatusUnknown:
			findings = append(findings, agentcatalog.PolicyValidationFinding{Type: "upstream_cli_unknown", Severity: "warning", Message: "selected CLI/upstream version observation unavailable: " + report.Note})
		}
	}
	return findings
}
