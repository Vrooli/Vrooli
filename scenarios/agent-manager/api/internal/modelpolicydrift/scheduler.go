// Package modelpolicydrift owns the scheduled, reporting side of model-policy
// drift detection. The host safeguard remains read-only; this package is the
// control-plane loop that persists the last observation and files typed bugs.
package modelpolicydrift

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/cli-core/agentcatalog"
)

var runners = []string{"codex", "claude-code", "opencode", "grok"}

type Snapshot struct {
	LastRun       time.Time         `json:"last_run,omitempty"`
	Status        string            `json:"status"`
	Measured      int               `json:"measured"`
	Total         int               `json:"total"`
	Findings      []Finding         `json:"findings,omitempty"`
	LastErrors    map[string]string `json:"last_errors,omitempty"`
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
	Report(context.Context, Report) error
}

type CheckFunc func(context.Context, string, string) ([]agentcatalog.PolicyValidationFinding, agentcatalog.LiveModelCatalog, error)

type Scheduler struct {
	root     string
	path     string
	interval time.Duration
	reporter Reporter
	check    CheckFunc
	now      func() time.Time
	mu       sync.RWMutex
	snapshot Snapshot
	stop     chan struct{}
	done     chan struct{}
}

func New(root, statePath string, interval time.Duration, reporter Reporter) *Scheduler {
	if interval <= 0 || interval > 14*24*time.Hour {
		interval = 7 * 24 * time.Hour
	}
	if strings.TrimSpace(statePath) == "" {
		statePath = filepath.Join(root, ".vrooli", "agent-manager", "model-policy-drift.json")
	}
	s := &Scheduler{root: root, path: statePath, interval: interval, reporter: reporter, check: agentcatalog.ValidateCatalogAgainstLive, now: time.Now, stop: make(chan struct{}), done: make(chan struct{})}
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
	now := s.now().UTC()
	findings := make([]Finding, 0)
	errorsByRunner := map[string]string{}
	measured := 0
	for _, runner := range runners {
		path := filepath.Join(s.root, "resources", runner, "model-policy.json")
		observed, _, err := s.check(ctx, runner, path)
		if err != nil {
			errorsByRunner[runner] = err.Error()
			continue
		}
		measured++
		for _, item := range observed {
			fingerprint := fmt.Sprintf("model-policy-drift/%s/%s/%s/%s", runner, item.Type, item.Role, item.Model)
			findings = append(findings, Finding{Runner: runner, Type: item.Type, Severity: item.Severity, Role: item.Role, Model: item.Model, Message: item.Message, Fingerprint: fingerprint})
		}
	}
	status := "healthy"
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
	reported := cloneMap(previous.Reported)
	s.snapshot = Snapshot{LastRun: now, Status: status, Measured: measured, Total: len(runners), Findings: findings, LastErrors: errorsByRunner, Reported: reported, IntervalHours: int(s.interval / time.Hour)}
	s.persistLocked()
	s.mu.Unlock()

	if s.reporter != nil {
		for _, item := range findings {
			// A large provider catalog can legitimately contain hundreds of
			// candidates. Keep those visible in health, but file scenario-qa
			// reports only for policy integrity/staleness findings so startup
			// cannot flood the inbox with one report per unadopted model.
			if item.Type == "unnamed_live_model" {
				continue
			}
			if _, ok := reported[item.Fingerprint]; ok {
				continue
			}
			if err := s.reporter.Report(ctx, Report{Finding: item, At: now}); err == nil {
				s.mu.Lock()
				s.snapshot.Reported[item.Fingerprint] = now.Format(time.RFC3339)
				s.persistLocked()
				s.mu.Unlock()
			}
		}
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
	data, err := json.MarshalIndent(s.snapshot, "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return
	}
	_ = os.Rename(tmp, s.path)
}

func cloneMap(input map[string]string) map[string]string {
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
