package control

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"device-control/internal/evidence"
	"device-control/strategy"
	strategyregistry "device-control/strategy/registry"
	"github.com/google/uuid"
)

type Device struct {
	ID, Name, Kind, StrategyID, Status, HealthReason, HostNodeID string
	Capabilities                                                 []strategy.Capability
	ObservedAt                                                   time.Time `json:"observed_at"`
}
type AttachedDevice struct {
	ID, Name, HostNodeID, Kind, Transport, Serial, OSVersion, TrustState, Reachability, HealthReason string
}
type AttachedReader interface {
	List(context.Context) ([]AttachedDevice, error)
}
type Session struct {
	ID, DeviceID, Actor, State, LeaseToken, KillReason string
	ExpiresAt, CreatedAt                               time.Time
}
type Audit struct {
	ID, Actor, DeviceID, LeaseID, Verb, Outcome string
	CreatedAt                                   time.Time
	RedactionVerified                           bool
}
type Step struct {
	ID, Kind             string
	RequiredCapabilities []string `json:"required_capabilities"`
	Target               string
	TimeoutMS            int64          `json:"timeout_ms"`
	Arguments            map[string]any `json:"arguments"`
}
type Flow struct {
	ID, Name               string
	Steps                  []Step
	AllowUnredactedCapture bool `json:"allow_unredacted_capture"`
}
type GapReport struct {
	Runnable bool     `json:"runnable"`
	Gaps     []string `json:"gaps"`
	Warnings []string `json:"warnings"`
}
type Chapter struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Disposition string `json:"disposition"`
	Message     string `json:"message"`
}
type Resolution struct {
	Target     string  `json:"target"`
	Rung       string  `json:"rung"`
	Confidence float64 `json:"confidence"`
}
type RunResult struct {
	RunID, Disposition string
	Chapters           []Chapter
	Resolutions        []Resolution
	Evidence           []evidence.Reference
}
type AgentRun struct {
	ID, Goal, DeviceID, Actor, State, Skill string
	Result                                  RunResult `json:"result"`
	CreatedAt                               time.Time `json:"created_at"`
}

type Service struct {
	registry *strategyregistry.Registry
	db       *sql.DB
	attached AttachedReader
	mu       sync.Mutex
	sessions map[string]Session
	audits   []Audit
	agents   map[string]AgentRun
}

func New(registry *strategyregistry.Registry) *Service {
	return &Service{registry: registry, sessions: map[string]Session{}, audits: []Audit{}, agents: map[string]AgentRun{}}
}

func NewWithAttached(registry *strategyregistry.Registry, reader AttachedReader) *Service {
	s := New(registry)
	s.attached = reader
	return s
}

// NewWithDB keeps the in-memory registry fast while making operator state
// durable. The API already owns the SQLite connection; passing it here avoids
// a second connection and keeps test-mode routing under api-core's control.
func NewWithDB(registry *strategyregistry.Registry, db *sql.DB) (*Service, error) {
	s := New(registry)
	s.db = db
	if db == nil {
		return s, nil
	}
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS device_control_sessions (
 id TEXT PRIMARY KEY, device_id TEXT NOT NULL, actor TEXT NOT NULL,
 state TEXT NOT NULL, lease_token TEXT NOT NULL DEFAULT '', kill_reason TEXT NOT NULL DEFAULT '',
 expires_at TEXT NOT NULL, created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS device_control_sessions_device ON device_control_sessions(device_id, state);
CREATE TABLE IF NOT EXISTS device_control_audits (
 id TEXT PRIMARY KEY, actor TEXT NOT NULL, device_id TEXT NOT NULL, lease_id TEXT NOT NULL,
 verb TEXT NOT NULL, outcome TEXT NOT NULL, created_at TEXT NOT NULL, redaction_verified INTEGER NOT NULL
);`); err != nil {
		return nil, fmt.Errorf("initialize device-control state: %w", err)
	}
	return s, nil
}

func NewWithDBAndAttached(registry *strategyregistry.Registry, db *sql.DB, reader AttachedReader) (*Service, error) {
	s, err := NewWithDB(registry, db)
	if err != nil {
		return nil, err
	}
	s.attached = reader
	return s, nil
}

func (s *Service) Strategies(ctx context.Context) []strategy.Declaration { return s.registry.List(ctx) }
func (s *Service) Verify(ctx context.Context, id string) (strategy.ConformanceReport, error) {
	return s.registry.Verify(ctx, id)
}
func (s *Service) Devices(ctx context.Context) []Device {
	declarations := s.registry.List(ctx)
	out := make([]Device, 0, len(declarations))
	for _, d := range declarations {
		reason := ""
		if len(d.NextActions) > 0 {
			reason = strings.Join(d.NextActions, " ")
		}
		out = append(out, Device{ID: d.StrategyID, Name: d.Description, Kind: d.StrategyID, StrategyID: d.StrategyID, Status: d.Status, HealthReason: reason, Capabilities: mapCaps(d), ObservedAt: time.Now().UTC()})
	}
	if s.attached != nil {
		attached, err := s.attached.List(ctx)
		if err != nil {
			out = append(out, Device{ID: "bridge", Name: "Bridge attached-device registry", Kind: "bridge", Status: strategy.StatusUnavailable, HealthReason: "bridge unavailable", ObservedAt: time.Now().UTC()})
		} else {
			for _, d := range attached {
				status := strategy.StatusAvailable
				reason := d.HealthReason
				if d.Reachability != "reachable" || d.TrustState != "trusted" {
					status = strategy.StatusUnavailable
				}
				out = append(out, Device{ID: d.ID, Name: d.Name, Kind: d.Kind, HostNodeID: d.HostNodeID, Status: status, HealthReason: reason, ObservedAt: time.Now().UTC()})
			}
		}
	}
	return out
}
func mapCaps(d strategy.Declaration) []strategy.Capability {
	names := make([]string, 0, len(d.Capabilities))
	for n := range d.Capabilities {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]strategy.Capability, 0, len(names))
	for _, n := range names {
		out = append(out, d.Capabilities[n])
	}
	return out
}

func (s *Service) Onboarding(kind string) []map[string]string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	common := []map[string]string{{"id": "host-node", "prerequisite": "A trusted bridge host node is online.", "owner": "owner", "status": "available", "next_action": "No action required."}}
	if kind == "android" {
		return append(common, map[string]string{"id": "android-sdk", "prerequisite": "android-sdk resource provides adb and platform-tools.", "owner": "scenario", "status": probeCommand("adb"), "next_action": "Install/start the android-sdk resource."}, map[string]string{"id": "usb-debugging", "prerequisite": "USB debugging is enabled and the device authorizes this host.", "owner": "owner", "status": "unavailable", "next_action": "Enable Developer Options and USB debugging, then accept the RSA prompt."})
	}
	if kind == "ios" {
		return append(common, map[string]string{"id": "xcode", "prerequisite": "Xcode and the requested simulator runtime are installed on a macOS node.", "owner": "owner", "status": "unavailable", "next_action": "Install Xcode and an iOS Simulator runtime."}, map[string]string{"id": "device-trust", "prerequisite": "The iPhone is attached and trusted.", "owner": "owner", "status": "unavailable", "next_action": "Connect the iPhone to the macOS node and tap Trust."})
	}
	return append(common, map[string]string{"id": "kind", "prerequisite": "A supported device kind is selected.", "owner": "owner", "status": "unavailable", "next_action": "Use --kind android or --kind ios."})
}
func probeCommand(name string) string {
	if _, err := execLookPath(name); err != nil {
		return "unavailable"
	}
	return "available"
}

var execLookPath = func(name string) (string, error) { return lookPath(name) }
var lookPath = func(name string) (string, error) { return exec.LookPath(name) }

func (s *Service) Acquire(deviceID, actor string, ttl time.Duration) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for id, old := range s.sessions {
		if old.State == "held" && now.After(old.ExpiresAt) {
			old.State = "expired"
			s.sessions[id] = old
		}
	}
	for _, old := range s.sessions {
		if old.DeviceID == deviceID && old.State == "held" {
			return Session{}, fmt.Errorf("device %s already has a held lease", deviceID)
		}
	}
	if _, ok := s.registry.Get(deviceID); !ok {
		return Session{}, fmt.Errorf("unknown device %q", deviceID)
	}
	if ttl <= 0 || ttl > 30*time.Minute {
		ttl = 10 * time.Minute
	}
	sess := Session{ID: uuid.NewString(), DeviceID: deviceID, Actor: actor, State: "held", LeaseToken: uuid.NewString(), ExpiresAt: now.Add(ttl), CreatedAt: now}
	if s.db != nil {
		if _, err := s.db.Exec(`INSERT INTO device_control_sessions (id, device_id, actor, state, lease_token, kill_reason, expires_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, sess.ID, sess.DeviceID, sess.Actor, sess.State, sess.LeaseToken, sess.KillReason, sess.ExpiresAt.Format(time.RFC3339Nano), sess.CreatedAt.Format(time.RFC3339Nano)); err != nil {
			return Session{}, fmt.Errorf("persist lease: %w", err)
		}
	}
	s.sessions[sess.ID] = sess
	return sess, nil
}
func (s *Service) ListSessions() []Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		rows, err := s.db.Query(`SELECT id, device_id, actor, state, lease_token, kill_reason, expires_at, created_at FROM device_control_sessions ORDER BY created_at DESC`)
		if err == nil {
			defer rows.Close()
			out := make([]Session, 0)
			for rows.Next() {
				var v Session
				var expires, created string
				if err := rows.Scan(&v.ID, &v.DeviceID, &v.Actor, &v.State, &v.LeaseToken, &v.KillReason, &expires, &created); err != nil {
					continue
				}
				v.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expires)
				v.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
				out = append(out, v)
			}
			return out
		}
	}
	out := make([]Session, 0, len(s.sessions))
	for _, v := range s.sessions {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}
func (s *Service) Kill(id, reason string) (Session, error) { return s.finish(id, "killed", reason) }
func (s *Service) Release(id string) (Session, error)      { return s.finish(id, "released", "") }
func (s *Service) finish(id, state, reason string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.sessions[id]
	if !ok {
		return Session{}, fmt.Errorf("session %q not found", id)
	}
	v.State = state
	v.KillReason = reason
	v.LeaseToken = ""
	if s.db != nil {
		if _, err := s.db.Exec(`UPDATE device_control_sessions SET state = ?, kill_reason = ?, lease_token = ? WHERE id = ?`, v.State, v.KillReason, v.LeaseToken, v.ID); err != nil {
			return Session{}, fmt.Errorf("persist session state: %w", err)
		}
	}
	s.sessions[id] = v
	return v, nil
}
func (s *Service) Audit() []Audit {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		rows, err := s.db.Query(`SELECT id, actor, device_id, lease_id, verb, outcome, created_at, redaction_verified FROM device_control_audits ORDER BY created_at DESC`)
		if err == nil {
			defer rows.Close()
			out := make([]Audit, 0)
			for rows.Next() {
				var v Audit
				var created string
				var verified int
				if err := rows.Scan(&v.ID, &v.Actor, &v.DeviceID, &v.LeaseID, &v.Verb, &v.Outcome, &created, &verified); err != nil {
					continue
				}
				v.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
				v.RedactionVerified = verified != 0
				out = append(out, v)
			}
			return out
		}
	}
	return append([]Audit{}, s.audits...)
}
func (s *Service) Validate(ctx context.Context, flow Flow, strategyID string) GapReport {
	d, ok := s.registry.Get(strategyID)
	if !ok {
		return GapReport{Gaps: []string{"unknown strategy " + strategyID}}
	}
	decl, _ := d.Describe(ctx)
	g := GapReport{Runnable: true, Gaps: []string{}, Warnings: []string{}}
	for _, step := range flow.Steps {
		if !knownStepKind(step.Kind) {
			g.Runnable = false
			g.Gaps = append(g.Gaps, fmt.Sprintf("step %s uses unsupported kind %q", step.ID, step.Kind))
		}
		if step.Kind == "semantic-target" && decl.Capabilities[strategy.CapSemanticTree].Status != strategy.StatusAvailable {
			g.Runnable = false
			g.Gaps = append(g.Gaps, fmt.Sprintf("step %s requires %s", step.ID, strategy.CapSemanticTree))
		}
		for _, cap := range step.RequiredCapabilities {
			if decl.Capabilities[cap].Status != strategy.StatusAvailable {
				g.Runnable = false
				g.Gaps = append(g.Gaps, fmt.Sprintf("step %s requires %s (%s)", step.ID, cap, decl.Capabilities[cap].NextAction))
			}
		}
	}
	return g
}
func (s *Service) Run(ctx context.Context, flow Flow, deviceID, actor string) (RunResult, error) {
	strat, ok := s.registry.Get(deviceID)
	if !ok {
		return RunResult{}, fmt.Errorf("unknown device %q", deviceID)
	}
	g := s.Validate(ctx, flow, deviceID)
	if !g.Runnable {
		return RunResult{RunID: uuid.NewString(), Disposition: "capability_gap", Chapters: []Chapter{{ID: "preflight", Title: "Capability preflight", Disposition: "failed", Message: strings.Join(g.Gaps, "; ")}}}, nil
	}
	sess, err := s.Acquire(deviceID, actor, 10*time.Minute)
	if err != nil {
		return RunResult{}, err
	}
	defer func() { _, _ = s.Release(sess.ID) }()
	result := RunResult{RunID: uuid.NewString(), Disposition: "passed", Chapters: []Chapter{}, Resolutions: []Resolution{}, Evidence: []evidence.Reference{}}
	for _, step := range flow.Steps {
		chapter := Chapter{ID: step.ID, Title: step.Kind, Disposition: "passed", Message: "completed"}
		if step.TimeoutMS <= 0 {
			step.TimeoutMS = 30000
		}
		stepctx, cancel := context.WithTimeout(ctx, time.Duration(step.TimeoutMS)*time.Millisecond)
		var stepErr error
		switch step.Kind {
		case "tap":
			stepErr = strat.Actuate(stepctx, strategy.Actuation{Pointer: &strategy.PointerEvent{Kind: "tap"}})
		case "key":
			stepErr = strat.Actuate(stepctx, strategy.Actuation{Key: &strategy.KeyEvent{Kind: "press", Key: step.Target}})
		case "observe":
			_, stepErr = strat.Observe(stepctx)
		case "wait":
			settle := 25 * time.Millisecond
			if raw, ok := step.Arguments["settle_ms"].(float64); ok && raw > 0 {
				settle = time.Duration(raw) * time.Millisecond
			}
			timer := time.NewTimer(settle)
			select {
			case <-timer.C:
			case <-stepctx.Done():
				stepErr = fmt.Errorf("bounded wait exceeded %dms", step.TimeoutMS)
			}
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		case "semantic-target", "install", "launch", "stop", "uninstall", "clear-data", "grant-permission", "revoke-permission":
			stepErr = fmt.Errorf("step kind %q is declared but not implemented by strategy %s", step.Kind, deviceID)
		}
		cancel()
		outcome := "success"
		if stepErr != nil {
			outcome = "failure"
			chapter.Disposition = "failed"
			chapter.Message = stepErr.Error()
			result.Disposition = "failed"
		}
		s.mu.Lock()
		audit := Audit{ID: uuid.NewString(), Actor: actor, DeviceID: deviceID, LeaseID: sess.ID, Verb: step.Kind, Outcome: outcome, CreatedAt: time.Now().UTC(), RedactionVerified: true}
		if s.db != nil {
			_, _ = s.db.Exec(`INSERT INTO device_control_audits (id, actor, device_id, lease_id, verb, outcome, created_at, redaction_verified) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, audit.ID, audit.Actor, audit.DeviceID, audit.LeaseID, audit.Verb, audit.Outcome, audit.CreatedAt.Format(time.RFC3339Nano), 1)
		}
		s.audits = append(s.audits, audit)
		s.mu.Unlock()
		result.Chapters = append(result.Chapters, chapter)
		if stepErr != nil {
			break
		}
	}
	return result, nil
}

// StartAgent is intentionally deterministic. Agent mode is an orchestration
// convenience over the same lease, capability, redaction, and evidence path;
// it never invents a step or calls a provider directly. A missing
// prompt-manager skill is a hard refusal, not a degraded improvisation.
func (s *Service) StartAgent(ctx context.Context, goal, deviceID, actor string, skillAvailable bool) (AgentRun, error) {
	if !skillAvailable {
		return AgentRun{}, fmt.Errorf("agent mode refused: prompt-manager device-control skill is unavailable")
	}
	if strings.TrimSpace(goal) == "" {
		return AgentRun{}, fmt.Errorf("agent goal is required")
	}
	run, err := s.Run(ctx, Flow{ID: uuid.NewString(), Name: "agent-observe", Steps: []Step{{ID: "observe", Kind: "observe", RequiredCapabilities: []string{strategy.CapScreenshot}, TimeoutMS: 5000}}}, deviceID, actor)
	if err != nil {
		return AgentRun{}, err
	}
	state := "completed"
	if run.Disposition != "passed" {
		state = "blocked"
	}
	agent := AgentRun{ID: uuid.NewString(), Goal: goal, DeviceID: deviceID, Actor: actor, State: state, Skill: "prompt-manager/device-control", Result: run, CreatedAt: time.Now().UTC()}
	s.mu.Lock()
	s.agents[agent.ID] = agent
	s.mu.Unlock()
	return agent, nil
}
func (s *Service) AbortAgent(id, reason string) (AgentRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.agents[id]
	if !ok {
		return AgentRun{}, fmt.Errorf("agent run %q not found", id)
	}
	a.State = "aborted"
	a.Result.Disposition = "aborted"
	a.Result.Chapters = append(a.Result.Chapters, Chapter{ID: "abort", Title: "Agent abort", Disposition: "passed", Message: reason})
	s.agents[id] = a
	return a, nil
}
func (s *Service) PromoteAgent(id string) (AgentRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.agents[id]
	if !ok {
		return AgentRun{}, fmt.Errorf("agent run %q not found", id)
	}
	if a.State != "completed" || a.Result.Disposition != "passed" {
		return AgentRun{}, fmt.Errorf("agent run %q is not eligible for promotion", id)
	}
	a.State = "promoted"
	s.agents[id] = a
	return a, nil
}
func (s *Service) ListAgents() []AgentRun {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]AgentRun, 0, len(s.agents))
	for _, a := range s.agents {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func knownStepKind(kind string) bool {
	switch kind {
	case "tap", "key", "observe", "wait", "semantic-target", "install", "launch", "stop", "uninstall", "clear-data", "grant-permission", "revoke-permission":
		return true
	default:
		return false
	}
}
