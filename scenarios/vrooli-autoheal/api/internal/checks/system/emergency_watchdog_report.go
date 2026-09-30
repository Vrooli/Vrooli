package system

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/internal/hostpressure"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

const (
	// EmergencyWatchdogReportCheckID is the sink's reader.
	EmergencyWatchdogReportCheckID = "system-emergency-watchdog-report"
	// emergencyWatchdogTimerInterval is the timer cadence rendered by
	// platform-go's watchdog definition; a report older than three intervals
	// is stale, and a stale report is undetermined, never healthy.
	emergencyWatchdogTimerInterval = 5 * time.Minute
	emergencyWatchdogStaleAfter    = 3 * emergencyWatchdogTimerInterval
	emergencyWatchdogReportRelPath = ".vrooli/state/emergency-watchdog/last-report.json"
	// attributedFindings are the findings the watchdog attaches attribution to.
	findingForkRate    = "fork-rate"
	findingCPUPressure = "cpu-pressure"
)

// EmergencyWatchdogReportCheck reads the emergency watchdog's last report
// (the state file the timer writes on every run) and turns each finding into
// a result entry that the incident service fingerprints separately. The
// watchdog senses; this check is how its findings reach autoheal, which
// decides.
type EmergencyWatchdogReportCheck struct {
	path      string
	now       func() time.Time
	authority *StormAuthority

	mu        sync.Mutex
	target    *StormTarget
	contained map[string]StormOutcome
}

type EmergencyWatchdogReportOption func(*EmergencyWatchdogReportCheck)

// WithReportPath pins the state file; production derives it from the home.
func WithReportPath(path string) EmergencyWatchdogReportOption {
	return func(c *EmergencyWatchdogReportCheck) { c.path = path }
}

// WithReportClock replaces the staleness clock.
func WithReportClock(now func() time.Time) EmergencyWatchdogReportOption {
	return func(c *EmergencyWatchdogReportCheck) { c.now = now }
}

// WithStormAuthority replaces the storm authority (tests inject fakes).
func WithStormAuthority(authority *StormAuthority) EmergencyWatchdogReportOption {
	return func(c *EmergencyWatchdogReportCheck) { c.authority = authority }
}

// WithContainStormMode sets the operator's containStorm mode on the
// production authority: automatic (default) or propose_only.
func WithContainStormMode(mode string) EmergencyWatchdogReportOption {
	return func(c *EmergencyWatchdogReportCheck) {
		if c.authority != nil {
			c.authority.Mode = normalizeContainStormMode(mode)
		}
	}
}

func NewEmergencyWatchdogReportCheck(opts ...EmergencyWatchdogReportOption) *EmergencyWatchdogReportCheck {
	c := &EmergencyWatchdogReportCheck{now: time.Now, authority: NewProductionStormAuthority(ContainStormAutomatic), contained: map[string]StormOutcome{}}
	if home, err := os.UserHomeDir(); err == nil {
		c.path = filepath.Join(home, filepath.FromSlash(emergencyWatchdogReportRelPath))
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *EmergencyWatchdogReportCheck) ID() string    { return EmergencyWatchdogReportCheckID }
func (c *EmergencyWatchdogReportCheck) Title() string { return "Emergency Watchdog Report" }
func (c *EmergencyWatchdogReportCheck) Description() string {
	return "Reads the emergency watchdog's last report and opens one incident per sustained finding, naming the attributed parent"
}

func (c *EmergencyWatchdogReportCheck) Importance() string {
	return "The watchdog senses host pressure but cannot act; a finding that never reaches autoheal is a finding nobody answers"
}
func (c *EmergencyWatchdogReportCheck) Category() checks.Category  { return checks.CategorySystem }
func (c *EmergencyWatchdogReportCheck) IntervalSeconds() int       { return 60 }
func (c *EmergencyWatchdogReportCheck) Platforms() []platform.Type { return nil }

// watchdogReport is the subset of the watchdog's output this check reads.
type watchdogReport struct {
	CapturedAt  time.Time                        `json:"captured_at"`
	Findings    []string                         `json:"findings"`
	Evidence    map[string][]string              `json:"evidence"`
	Attribution *hostpressure.AttributionReading `json:"attribution,omitempty"`
}

func (c *EmergencyWatchdogReportCheck) Run(context.Context) checks.Result {
	result := checks.Result{CheckID: c.ID(), Details: map[string]interface{}{"reportPath": c.path}}
	undetermined := func(reason string) checks.Result {
		result.Status = checks.StatusWarning
		result.Details["reportState"] = "undetermined"
		result.Details["reportReason"] = reason
		result.Message = "Emergency watchdog report is undetermined: " + reason
		return result
	}
	if strings.TrimSpace(c.path) == "" {
		return undetermined("no home directory to locate the report")
	}
	data, err := os.ReadFile(c.path)
	if err != nil {
		return undetermined(fmt.Sprintf("read %s: %v", c.path, err))
	}
	var report watchdogReport
	if err := json.Unmarshal(data, &report); err != nil {
		return undetermined(fmt.Sprintf("parse %s: %v", c.path, err))
	}
	if report.CapturedAt.IsZero() {
		return undetermined("report carries no captured_at")
	}
	age := c.now().Sub(report.CapturedAt)
	result.Details["reportAge"] = age.String()
	result.Details["capturedAt"] = report.CapturedAt.UTC().Format(time.RFC3339)
	if age > emergencyWatchdogStaleAfter {
		return undetermined(fmt.Sprintf("report is %s old, older than %s (three timer intervals)", age.Round(time.Second), emergencyWatchdogStaleAfter))
	}
	result.Details["reportState"] = "read"
	findings := aggregateFindings(report)
	c.setTarget(stormTargetFromFindings(findings))
	c.annotateContainment(findings, result.Details)
	result.Details["findings"] = findings
	if report.Attribution != nil {
		result.Details["attribution"] = attributionMap(report.Attribution)
	}
	if len(findings) == 0 {
		result.Status = checks.StatusOK
		result.Message = "Emergency watchdog reported no findings"
		return result
	}
	result.Status = checks.StatusCritical
	result.Message = fmt.Sprintf("Emergency watchdog reported %d finding(s): %s", len(findings), strings.Join(report.Findings, "; "))
	return result
}

// attributionDetail attaches the parent ranking to the findings the
// watchdog attributes (fork rate and CPU pressure); other findings carry none.
func attributionDetail(finding string, attribution *hostpressure.AttributionReading) map[string]any {
	if attribution == nil || attribution.State != hostpressure.Read {
		return nil
	}
	if finding != findingForkRate && finding != findingCPUPressure {
		return nil
	}
	out := attributionMap(attribution)
	if top, ok := attribution.TopParent(); ok {
		out["top_parent"] = parentMap(top)
	}
	return out
}

func attributionMap(attribution *hostpressure.AttributionReading) map[string]any {
	byChildren := make([]map[string]any, 0, len(attribution.ByChildren))
	for _, parent := range attribution.ByChildren {
		byChildren = append(byChildren, parentMap(parent))
	}
	byDelta := make([]map[string]any, 0, len(attribution.ByDelta))
	for _, parent := range attribution.ByDelta {
		byDelta = append(byDelta, parentMap(parent))
	}
	return map[string]any{"state": string(attribution.State), "reason": attribution.Reason, "by_children": byChildren, "by_delta": byDelta}
}

func parentMap(parent hostpressure.Parent) map[string]any {
	return map[string]any{"pid": parent.PID, "name": parent.Name, "children": parent.Children, "delta": parent.Delta, "scope": parent.Scope}
}

// stormTargetFromFindings picks the attributed parent of the first fork-rate
// or cpu-pressure finding whose scope is an agent session. The watchdog only
// reports a finding once its authored sustain has elapsed, so a listed
// finding is a sustained one; attribution without a finding is not a target.
func stormTargetFromFindings(findings []map[string]any) *StormTarget {
	for _, finding := range findings {
		name, _ := finding["name"].(string)
		if name != findingForkRate && name != findingCPUPressure {
			continue
		}
		attribution, _ := finding["attribution"].(map[string]any)
		top, _ := attribution["top_parent"].(map[string]any)
		scopePath, _ := top["scope"].(string)
		scopeName, ok := AgentScopeName(scopePath)
		if !ok {
			continue
		}
		target := &StormTarget{Finding: name, ScopePath: scopePath, ScopeName: scopeName}
		target.Name, _ = top["name"].(string)
		switch pid := top["pid"].(type) {
		case int64:
			target.PID = pid
		case int:
			target.PID = int64(pid)
		case float64:
			target.PID = int64(pid)
		}
		switch children := top["children"].(type) {
		case int:
			target.Children = children
		case float64:
			target.Children = int(children)
		}
		return target
	}
	return nil
}

func (c *EmergencyWatchdogReportCheck) setTarget(target *StormTarget) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.target = target
}

// annotateContainment marks findings whose scope this authority froze, drops
// scopes an operator thawed out of band, and surfaces the thaw command in
// the finding's reason so the incident names it.
func (c *EmergencyWatchdogReportCheck) annotateContainment(findings []map[string]any, details map[string]interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.contained) == 0 {
		return
	}
	// A scope thawed out of band, or gone entirely (the session exited or
	// the operator stopped it), is no longer contained; only a scope that
	// still reads frozen stays on the list.
	for scope, outcome := range c.contained {
		if c.authority != nil && c.authority.Frozen != nil {
			frozen, err := c.authority.Frozen(outcome.Scope)
			if err != nil || !frozen {
				delete(c.contained, scope)
			}
		}
	}
	containment := make([]map[string]any, 0, len(c.contained))
	for _, outcome := range c.contained {
		containment = append(containment, map[string]any{
			"scope": outcome.Target.ScopeName, "scope_path": outcome.Target.ScopePath, "pid": outcome.Target.PID,
			"working_dir": outcome.Target.WorkingDir, "frozen_at": outcome.FrozenAt.UTC().Format(time.RFC3339), "thaw_command": outcome.ThawCommand, "epoch": outcome.EpochID,
		})
	}
	if len(containment) > 0 {
		details["containment"] = containment
	}
	for _, finding := range findings {
		attribution, _ := finding["attribution"].(map[string]any)
		top, _ := attribution["top_parent"].(map[string]any)
		scopePath, _ := top["scope"].(string)
		name, ok := AgentScopeName(scopePath)
		if !ok {
			continue
		}
		if outcome, frozen := c.contained[name]; frozen {
			finding["contained"] = true
			finding["thaw_command"] = outcome.ThawCommand
			if reason, _ := finding["reason"].(string); reason != "" {
				finding["reason"] = reason + "; contained: " + name + " is frozen since " + outcome.FrozenAt.UTC().Format(time.RFC3339) + ", thaw with `" + outcome.ThawCommand + "`"
			}
		}
	}
}

// RecoveryActions offers contain-storm only when the last result named an
// agent session scope for a sustained finding and that scope is not frozen
// yet. In propose_only mode the action is Dangerous, so the auto-heal pass
// never selects it and the operator must run it.
func (c *EmergencyWatchdogReportCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	if c.authority == nil || lastResult == nil {
		return nil
	}
	target := stormTargetFromFindings(findingsFromDetails(lastResult.Details))
	c.mu.Lock()
	_, frozen := c.contained[targetScope(target)]
	c.mu.Unlock()
	available := target != nil && !frozen && lastResult.Status == checks.StatusCritical
	return []checks.RecoveryAction{{
		ID:          ContainStormActionID,
		Name:        "Contain storm",
		Description: "Freeze the attributed agent session scope (never a supervisor or a resource); reversible with `vrooli agent thaw <scope>`",
		Dangerous:   c.authority.ProposeOnly(),
		Available:   available,
	}}
}

func targetScope(target *StormTarget) string {
	if target == nil {
		return ""
	}
	return target.ScopeName
}

func findingsFromDetails(details map[string]interface{}) []map[string]any {
	if details == nil {
		return nil
	}
	switch raw := details["findings"].(type) {
	case []map[string]any:
		return raw
	case []any:
		out := make([]map[string]any, 0, len(raw))
		for _, item := range raw {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	}
	return nil
}

// ExecuteAction runs contain-storm against the target the last Run named.
func (c *EmergencyWatchdogReportCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	started := time.Now()
	result := checks.ActionResult{ActionID: actionID, CheckID: c.ID(), Timestamp: started}
	if actionID != ContainStormActionID {
		result.Error = fmt.Sprintf("unknown action %q", actionID)
		result.Message = "Unknown recovery action"
		return result
	}
	c.mu.Lock()
	target := c.target
	c.mu.Unlock()
	if target == nil {
		result.Error = "no sustained finding names an agent session scope"
		result.Message = "contain-storm has no target"
		return result
	}
	outcome, err := c.authority.Contain(ctx, *target)
	result.Duration = time.Since(started)
	if err != nil {
		result.Error = err.Error()
		result.Message = "contain-storm refused"
		return result
	}
	c.mu.Lock()
	c.contained[outcome.Target.ScopeName] = outcome
	c.mu.Unlock()
	result.Success = true
	result.Message = fmt.Sprintf("froze %s (%s pid %d) in epoch %s; thaw with `%s`", outcome.Target.ScopeName, outcome.Target.Name, outcome.Target.PID, outcome.EpochID, outcome.ThawCommand)
	result.Output = outcome.Decision.Reason
	return result
}

// ContainedScopes lists the scopes this authority froze and has not seen thawed.
func (c *EmergencyWatchdogReportCheck) ContainedScopes() []StormOutcome {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]StormOutcome, 0, len(c.contained))
	for _, outcome := range c.contained {
		out = append(out, outcome)
	}
	return out
}

// aggregateFindings turns the watchdog's finding lines into one finding per
// family. The watchdog names per-workload findings "family:<detail>"
// (unmanaged-workload:<pid>, crash-loop:<unit>); one incident per family,
// carrying the count and the first reasons, is the actionable shape. One
// incident per process (190 of them on a whole_host posture) is noise that
// buries the storm finding it exists to surface.
func aggregateFindings(report watchdogReport) []map[string]any {
	type family struct {
		name    string
		reasons []string
		count   int
		first   string
	}
	order := []string{}
	families := map[string]*family{}
	for _, line := range report.Findings {
		name, reason, _ := strings.Cut(line, ": ")
		name = strings.TrimSpace(name)
		base, _, _ := strings.Cut(name, ":")
		base = strings.TrimSpace(base)
		f, ok := families[base]
		if !ok {
			f = &family{name: base, first: name}
			families[base] = f
			order = append(order, base)
		}
		f.count++
		if len(f.reasons) < maxAggregatedReasons {
			detail := strings.TrimSpace(reason)
			if name != base {
				detail = strings.TrimSpace(strings.TrimPrefix(name, base+":")) + ": " + detail
			}
			f.reasons = append(f.reasons, detail)
		}
	}
	findings := make([]map[string]any, 0, len(order))
	for _, base := range order {
		f := families[base]
		finding := map[string]any{"name": f.name, "reason": strings.Join(f.reasons, "; ")}
		if f.count > 1 {
			finding["count"] = f.count
			finding["reason"] = fmt.Sprintf("%d %s findings; first: %s", f.count, f.name, strings.Join(f.reasons, "; "))
		}
		if evidence := report.Evidence[f.first]; len(evidence) > 0 {
			finding["evidence"] = evidence
		}
		if attribution := attributionDetail(f.name, report.Attribution); attribution != nil {
			finding["attribution"] = attribution
		}
		findings = append(findings, finding)
	}
	return findings
}

const maxAggregatedReasons = 5
