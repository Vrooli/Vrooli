package launch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/schedule"

	"research-desk/internal/home"
	"research-desk/internal/skillscripts"
	"research-desk/internal/wire"
	"research-desk/internal/writer"
)

// Packets is packet.py (skillscripts.Packets).
type Packets interface {
	Preview(ctx context.Context, spec []byte) (*skillscripts.PacketResult, error)
	Build(ctx context.Context, spec []byte) (*skillscripts.PacketResult, error)
	Check(ctx context.Context, commissionID string) (*skillscripts.PacketResult, error)
}

// Service is the launch domain over one research home.
type Service struct {
	Reader interface {
		home.Snapshotter
		CodexHome(sessionID string) string
	}
	Writer   *writer.Writer
	Packets  Packets
	Limits   func() (*skillscripts.Limits, error)
	Launcher *Launcher
	// Access says how the desk reaches Web Console, recorded in LAUNCH.json;
	// nil records nothing.
	Access func(ctx context.Context) (string, error)
	Clock  schedule.NowSource
	Logger *log.Logger

	mu sync.Mutex
	// running holds commissions with a launch or control in progress here.
	running map[string]bool
}

// Choices are what a preflight or launch request decides that the desk
// cannot: how each blocker is resolved and the operator's
// assertion that an unknown commission was never started.
type Choices struct {
	// Resolutions resolve blockers by commission id; a spec's supersedes and
	// run_alongside count as choices the request may override.
	Resolutions map[string]Resolution
	// AssertNeverStarted is the operator's word that the commission, which
	// the desk classes unknown, was never started.
	AssertNeverStarted bool
}

// Preflight is a packet result with the topic verdicts.
type Preflight struct {
	Packet   *skillscripts.PacketResult
	Verdicts []Verdict
	Blockers []string
	// NeverStarted is the commission's never-started answer when it is unknown; Asserted
	// says the request's assertion was accepted.
	NeverStarted home.NeverStarted
	Asserted     bool
	// OwnerReading is what each topic's owner reads at start. A set over its
	// read ceiling is flagged for trimming; no flag blocks the launch.
	OwnerReading []home.ReadSet
	// Reading is the commission's own read sets (scheduler start and unit),
	// empty for a spec that has no folder yet.
	Reading []home.ReadSet
	// PacketState is the start of an existing packet's own STATE.md, nil for
	// a spec.
	PacketState *StateExcerpt
}

// Launchable reports whether the packet passes and nothing blocks.
func (p Preflight) Launchable() bool { return len(p.Blockers) == 0 }

// specChoices is the part of a spec the desk reads; packet.py validates all of it.
type specChoices struct {
	Topics       []string
	RunAlongside []string
	Supersedes   []string
}

// readSpec parses a spec once: its fields, kept raw so the rest passes
// through unchanged, and the choices the desk reads. A spec that is not a
// JSON object gives nil fields and no choices; packet.py reports it.
func readSpec(spec []byte) (map[string]json.RawMessage, specChoices) {
	var fields map[string]json.RawMessage
	var sc specChoices
	if json.Unmarshal(spec, &fields) != nil || fields == nil {
		return nil, sc
	}
	var supersedes home.IDs
	_ = json.Unmarshal(fields["topics"], &sc.Topics)
	_ = json.Unmarshal(fields["run_alongside"], &sc.RunAlongside)
	_ = json.Unmarshal(fields["supersedes"], &supersedes)
	sc.Supersedes = supersedes
	return fields, sc
}

// resolutions merges a spec's choices with the request's supersedes and
// run_alongside; the request wins.
func resolutions(alongside, supersedes []string, req map[string]Resolution) map[string]Resolution {
	out := map[string]Resolution{}
	for _, id := range alongside {
		out[id] = RunAlongside
	}
	for _, id := range supersedes {
		out[id] = Supersede
	}
	for id, r := range req {
		if r != "" {
			out[id] = r
		}
	}
	return out
}

// PreflightSpec renders a spec in memory and judges its topics.
func (s *Service) PreflightSpec(ctx context.Context, spec []byte, ch Choices) (Preflight, error) {
	result, err := s.Packets.Preview(ctx, spec)
	if err != nil {
		return Preflight{}, home.Cause(home.ErrUnavailable, err)
	}
	snap, err := s.Reader.Snapshot(ctx, home.Options{})
	if err != nil {
		return Preflight{}, err
	}
	_, sc := readSpec(spec)
	return judge(snap, result, sc, ch.Resolutions, ""), nil
}

// PreflightCommission gates a commission folder and judges its topics. A
// prepared folder passes; an unknown one passes only on the operator's
// assertion that it never started, and only when the desk found no session
// or goal for it.
func (s *Service) PreflightCommission(ctx context.Context, id string, ch Choices) (Preflight, error) {
	snap, c, err := s.find(ctx, id)
	if err != nil {
		return Preflight{}, err
	}
	return s.preflightCommission(ctx, snap, c, ch)
}

func (s *Service) preflightCommission(ctx context.Context, snap *home.Snapshot, c *home.Commission, ch Choices) (Preflight, error) {
	id := c.ID
	result, err := s.Packets.Check(ctx, id)
	if err != nil {
		return Preflight{}, home.Cause(home.ErrUnavailable, err)
	}
	sc := specChoices{}
	for _, t := range c.Topics {
		sc.Topics = append(sc.Topics, t.Topic)
	}
	if c.Spec != nil {
		sc.RunAlongside, sc.Supersedes = c.Spec.RunAlongside, c.Spec.Supersedes
	}
	p := judge(snap, result, sc, ch.Resolutions, id)
	switch c.State {
	case home.StatePrepared:
		p.PacketState = stateExcerpt(c)
	case home.StateUnknown:
		p.PacketState = stateExcerpt(c)
		p.NeverStarted = c.NeverStarted
		switch {
		case !c.NeverStarted.Assertable:
			p.Blockers = append(p.Blockers, id+" is unknown and cannot be launched on the operator's word: "+c.NeverStarted.Reason)
		case !ch.AssertNeverStarted:
			p.Blockers = append(p.Blockers, id+" is unknown ("+c.StateRule+"); "+c.NeverStarted.Reason+"; launch it only if you know it was never started")
		default:
			p.Asserted = true
		}
	default:
		p.Blockers = append(p.Blockers, id+" is "+string(c.State)+", not prepared; relaunch it as a new commission")
	}
	return p, nil
}

// judge gives a packet result its topic verdicts and blockers; self is the
// commission being judged, which never blocks itself.
func judge(snap *home.Snapshot, result *skillscripts.PacketResult, sc specChoices, req map[string]Resolution, self string) Preflight {
	p := Preflight{Packet: result, Verdicts: Verdicts(snap, sc.Topics, resolutions(sc.RunAlongside, sc.Supersedes, req), self), OwnerReading: snap.OwnerReading(sc.Topics)}
	if c := snap.Commission(self); self != "" && c != nil {
		p.Reading = snap.ReadSets(c)
	}
	p.Blockers = blockers(p)
	return p
}

func blockers(p Preflight) []string {
	var out []string
	if p.Packet != nil && !p.Packet.OK {
		for _, e := range p.Packet.Errors {
			out = append(out, "packet: "+e.Code+": "+e.Message)
		}
		for _, g := range p.Packet.Gates {
			if !g.OK {
				out = append(out, fmt.Sprintf("gate %s: %s over %s", g.Name, g.Value, g.Limit))
			}
		}
	}
	for _, v := range p.Verdicts {
		if v.Blocks {
			out = append(out, v.Topic+": "+v.CommissionID+" "+v.Message)
		}
	}
	// Read ceilings are measured and flagged in the preflight (set.Flags) but never block: a read cost is friction to cut,
	// and a gate that stops the work is a defect (R1). On 2026-11-20 exact-size skill ceilings would have refused every launch
	// after a one-byte skill edit.
	return out
}

// Build writes a new commission folder from a spec. The operator's choices
// for blockers go into the spec first, so the charter names each commission
// it runs alongside and the one it supersedes.
func (s *Service) Build(ctx context.Context, spec []byte, req map[string]Resolution) (Preflight, error) {
	if err := s.Writer.Held(); err != nil {
		return Preflight{}, err
	}
	spec, sc, err := withChoices(spec, req)
	if err != nil {
		return Preflight{}, err
	}
	result, err := s.Packets.Build(ctx, spec)
	if err != nil {
		return Preflight{}, home.Cause(home.ErrUnavailable, err)
	}
	snap, err := s.Reader.Snapshot(ctx, home.Options{})
	if err != nil {
		return Preflight{}, err
	}
	return judge(snap, result, sc, nil, result.CommissionID), nil
}

// withChoices writes the request's choices into a spec: run_alongside
// lists every commission run alongside, supersedes every superseded one, so
// the charter packet.py renders names each. Without choices, or when the
// spec is not a JSON object, the spec passes through verbatim. It returns the
// choices the desk reads from the result.
func withChoices(spec []byte, req map[string]Resolution) ([]byte, specChoices, error) {
	fields, sc := readSpec(spec)
	if len(req) == 0 || fields == nil {
		return spec, sc, nil
	}
	alongside, supersedes := sc.RunAlongside, sc.Supersedes
	chosen := make([]string, 0, len(req))
	for id := range req {
		chosen = append(chosen, id)
	}
	sort.Strings(chosen)
	for _, id := range chosen {
		into, outOf := &supersedes, &alongside
		if req[id] == RunAlongside {
			into, outOf = &alongside, &supersedes
		} else if req[id] != Supersede {
			continue
		}
		*outOf = slices.DeleteFunc(*outOf, func(v string) bool { return v == id })
		if !slices.Contains(*into, id) {
			*into = append(*into, id)
		}
	}
	sc.RunAlongside, sc.Supersedes = alongside, supersedes
	var err error
	if fields["run_alongside"], err = json.Marshal(nonNil(alongside)); err != nil {
		return nil, sc, err
	}
	if fields["supersedes"], err = json.Marshal(nonNil(supersedes)); err != nil {
		return nil, sc, err
	}
	out, err := json.Marshal(fields)
	return out, sc, err
}

func nonNil(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}

// Relaunch builds a new commission from an earlier one's PACKET.json with
// supersedes and carry_baseline_from set to it.
func (s *Service) Relaunch(ctx context.Context, id string, req map[string]Resolution) (Preflight, error) {
	_, c, err := s.find(ctx, id)
	if err != nil {
		return Preflight{}, err
	}
	if c.Packet == nil {
		return Preflight{}, fmt.Errorf("%w: %s has no PACKET.json (it was not built by packet.py); build a new packet for its topics instead", home.ErrRefused, id)
	}
	var spec map[string]any
	if err := json.Unmarshal(c.Packet.Spec, &spec); err != nil {
		return Preflight{}, fmt.Errorf("%s PACKET.json spec: %w", id, err)
	}
	// A bad start often never wrote BASELINE.json; the new commission then
	// records its own baseline instead of carrying one.
	spec["supersedes"], spec["carry_baseline_from"] = []string{id}, nil
	if _, ok := c.File(home.BaselineFile); ok {
		spec["carry_baseline_from"] = id
	}
	data, err := json.Marshal(spec)
	if err != nil {
		return Preflight{}, err
	}
	return s.Build(ctx, data, req)
}

// Messages are a built packet's charter path and its two messages.
type Messages struct {
	CharterPath, Stance, Kernel string
}

// Messages reads a packet's two messages; it writes nothing.
func (s *Service) Messages(ctx context.Context, id string) (Messages, error) {
	_, c, err := s.find(ctx, id)
	if err != nil {
		return Messages{}, err
	}
	return messages(c)
}

func messages(c *home.Commission) (Messages, error) {
	id := c.ID
	stanceText, err1 := os.ReadFile(filepath.Join(c.Dir, home.StanceFile))
	kernelText, err2 := os.ReadFile(filepath.Join(c.Dir, home.KernelFile))
	if err := errors.Join(err1, err2); err != nil {
		return Messages{}, fmt.Errorf("%w: %s is not a complete packet: %v", home.ErrRefused, id, err)
	}
	return Messages{CharterPath: filepath.Join(c.Dir, home.CharterFile), Stance: string(stanceText), Kernel: strings.TrimSpace(string(kernelText))}, nil
}

func (s *Service) access(ctx context.Context) (string, error) {
	if s.Access == nil {
		return "", nil
	}
	return s.Access(ctx)
}

// supersession is an earlier commission this launch supersedes: stopped
// through its own session first when Stop, then recorded.
type supersession struct {
	ID     string
	Target Target
	Stop   bool
	Open   bool
	// Stale says why a commission is recorded without being stopped: its
	// thread is unknown and it has been silent past StaleAfter.
	Stale string
}

// supersessions checks, before anything is launched, that every superseded
// commission with a supervisor can be stopped through its own thread and
// session; any that cannot refuses the whole launch.
func (s *Service) supersessions(ctx context.Context, snap *home.Snapshot, verdicts []Verdict) ([]supersession, error) {
	var out []supersession
	seen := map[string]bool{}
	for _, v := range verdicts {
		if v.Chosen != Supersede || seen[v.CommissionID] {
			continue
		}
		seen[v.CommissionID] = true
		c := snap.Commission(v.CommissionID)
		if c == nil {
			continue
		}
		t := targetOf(c)
		// A desk launch can have a durable session and a recorded stance_sent,
		// yet no thread_id when Web Console disappeared while waiting at
		// ready_seen. Re-resolve only the unique top-level thread from that
		// launch's exact session; Ready below still proves the terminal shows
		// that thread idle before Stop can type or signal anything.
		if t.Thread == "" && c.Launch != nil && c.Launch.LaunchedBy == "desk" &&
			c.Launch.Failed != nil && c.Launch.Failed.Step == StepReadySeen &&
			hasLaunchStep(c.Launch.Steps, StepStanceSent) && !hasLaunchStep(c.Launch.Steps, StepGoalSent) {
			if thread, err := s.Launcher.Sessions.SessionThread(t.Session); err == nil {
				t.Thread = thread
			}
		}
		sup := supersession{ID: c.ID, Target: t, Open: c.State.Running() || (c.Session != nil && c.Session.WindowOpen)}
		if sup.Stale = staleThread(c, t, snap.Now); sup.Stale != "" || (!sup.Open && c.Session == nil) {
			out = append(out, sup) // nothing the desk can or need stop: recorded only
			continue
		}
		sup.Stop = true
		refuse := func(why string) error {
			return fmt.Errorf("%w: cannot stop %s, which this launch supersedes: %s; nothing was launched", home.ErrRefused, c.ID, why)
		}
		switch {
		case t.Session == "":
			return nil, refuse("no session is known for it")
		case sup.Open:
			if err := s.Launcher.Ready(ctx, t); err != nil {
				return nil, refuse(err.Error())
			}
		default:
			if _, err := s.Launcher.Sessions.SessionProcesses(t.Session); err != nil {
				return nil, refuse("its session's processes cannot be read: " + err.Error())
			}
		}
		out = append(out, sup)
	}
	return out, nil
}

// staleThread says why an unknown commission may be superseded without a
// stop: no live goal, no Codex thread the desk can name, and no write for
// StaleAfter. Empty when any of that does not hold.
func staleThread(c *home.Commission, t Target, now time.Time) string {
	if c.State != home.StateUnknown || t.Thread != "" || !c.LastWrite.Known || c.LastWrite.AtLeast || now.Sub(c.LastWrite.Value) < StaleAfter {
		return ""
	}
	return "stale: its Codex thread is unknown and nothing in it was written since " + stamp(c.LastWrite.Value) +
		", so it was recorded as superseded without typing into or ending any session"
}

// targetOf is a commission's own conversation: the session and thread
// LAUNCH.json recorded, else the exact match's.
func targetOf(c *home.Commission) Target {
	var t Target
	if c.Launch != nil {
		t = Target{Session: c.Launch.WebConsoleSessionID, Thread: c.Launch.ThreadID}
	}
	if t.Session == "" && c.Session != nil {
		t = Target{Session: c.Session.SessionID, Thread: c.Session.ThreadID}
	} else if t.Thread == "" && c.Session != nil && c.Session.SessionID == t.Session {
		t.Thread = c.Session.ThreadID
	}
	return t
}

// Launch starts an automatic launch of a prepared commission (or an unknown
// one the operator asserts never started) in the background and returns its
// first record. Superseded supervisors are stopped first, in the background
// run; a supersession that cannot be stopped refuses the launch here.
func (s *Service) Launch(ctx context.Context, id string, ch Choices) (*home.LaunchRecord, error) {
	if err := wire.ResearchOperation(ctx, "research.launch", id); err != nil {
		return nil, err
	}
	if err := s.Writer.Held(); err != nil {
		return nil, err
	}
	snap, c, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	p, err := s.preflightCommission(ctx, snap, c, ch)
	if err != nil {
		return nil, err
	}
	if !p.Launchable() {
		return nil, fmt.Errorf("%w: %s", home.ErrRefused, strings.Join(p.Blockers, "; "))
	}
	msgs, err := messages(c)
	if err != nil {
		return nil, err
	}
	settings, err := s.settings(c)
	if err != nil {
		return nil, err
	}
	sups, err := s.supersessions(ctx, snap, p.Verdicts)
	if err != nil {
		return nil, err
	}
	// All affected targets must be covered before any supervisor is stopped
	// or any launch record is written. The packet cannot expand its grant.
	for _, sup := range sups {
		if err := wire.ResearchOperation(ctx, "research.control", sup.ID); err != nil {
			return nil, err
		}
	}
	claimed := []string{}
	for _, cid := range append([]string{id}, ids(sups)...) {
		if !s.claim(cid) {
			s.release(claimed...)
			return nil, fmt.Errorf("%w: a launch or control of %s is already running", home.ErrRefused, cid)
		}
		claimed = append(claimed, cid)
	}
	access, err := s.access(ctx)
	if err != nil {
		s.release(claimed...)
		return nil, home.Cause(home.ErrUnavailable, err)
	}
	record := &home.LaunchRecord{Schema: home.LaunchSchema, CommissionID: id, LaunchedBy: "desk", IdempotencyKey: id + "#1", WebConsoleAccess: access, Steps: []home.LaunchStep{}, Controls: []home.ControlEntry{}, Supersedes: ids(sups)}
	for _, v := range p.Verdicts {
		if v.Chosen == RunAlongside && !slices.Contains(record.RunAlongside, v.CommissionID) {
			record.RunAlongside = append(record.RunAlongside, v.CommissionID)
		}
	}
	if p.Asserted {
		record.NeverStarted = &home.Assertion{At: s.Clock.Now().UTC(), Evidence: p.NeverStarted.Reason}
	}
	if _, err := s.Writer.WriteLaunch(id, record); err != nil {
		s.release(claimed...)
		return nil, err
	}
	snapshot := *record
	go s.run(context.WithoutCancel(ctx), id, msgs.CharterPath, record, Texts{Stance: msgs.Stance, Kernel: msgs.Kernel}, settings, sups, claimed)
	return &snapshot, nil
}

// StepSupersede is the failure step when a superseded supervisor could not
// be stopped; nothing was launched.
const StepSupersede = "supersede"

func ids(sups []supersession) []string {
	var out []string
	for _, s := range sups {
		out = append(out, s.ID)
	}
	return out
}

func (s *Service) run(ctx context.Context, id, charter string, record *home.LaunchRecord, texts Texts, settings Settings, sups []supersession, claimed []string) {
	defer s.release(claimed...)
	for _, sup := range sups {
		if err := wire.ResearchOperation(ctx, "research.control", sup.ID); err != nil {
			s.fail(id, record, StepSupersede, err.Error())
			return
		}
		outcome := "no window found; no session is known for it, so nothing could still be running under the desk's control"
		if sup.Stale != "" {
			outcome = sup.Stale
		}
		if sup.Stop {
			stopped, err := s.Launcher.Stop(ctx, sup.Target, sup.Open)
			if err != nil {
				s.fail(id, record, StepSupersede, "stop "+sup.ID+": "+err.Error()+"; nothing was launched")
				return
			}
			outcome = stopped.Outcome()
		}
		if err := s.markSuperseded(ctx, sup, id, outcome); err != nil {
			s.fail(id, record, StepSupersede, "record "+sup.ID+" as superseded: "+err.Error()+"; nothing was launched")
			return
		}
	}
	if err := wire.ResearchOperation(ctx, "research.launch", id); err != nil {
		s.fail(id, record, StepCreated, err.Error())
		return
	}
	spec := SessionSpec{IdempotencyKey: record.IdempotencyKey, Label: "research · " + id}
	err := s.Launcher.Run(ctx, spec, texts, settings, func(step, sessionID, threadID string) error {
		record.WebConsoleSessionID = sessionID
		if threadID != "" {
			record.ThreadID = threadID
		}
		record.Steps = append(record.Steps, home.LaunchStep{Step: step, At: s.Clock.Now().UTC()})
		_, err := s.Writer.WriteLaunch(id, record)
		return err
	})
	if err == nil {
		s.Logger.Printf("launch: %s verified (session %s, charter %s)", id, record.WebConsoleSessionID, charter)
		return
	}
	step, reason := failure(err)
	s.fail(id, record, step, reason)
}

func (s *Service) fail(id string, record *home.LaunchRecord, step, reason string) {
	record.Failed = &home.LaunchFailure{Step: step, Reason: reason}
	if _, werr := s.Writer.WriteLaunch(id, record); werr != nil {
		s.Logger.Printf("launch: %s failed at %s (%s) and the failure could not be recorded: %v", id, step, reason, werr)
		return
	}
	s.Logger.Printf("launch: %s failed at %s: %s", id, step, reason)
}

// markSuperseded records in the earlier commission's LAUNCH.json that it was
// superseded by the new one, at the operator's choice; this ends it.
func (s *Service) markSuperseded(ctx context.Context, sup supersession, by, outcome string) error {
	_, c, err := s.find(ctx, sup.ID)
	if err != nil {
		return err
	}
	record := launchRecord(c)
	fillTarget(record, sup.Target)
	now := s.Clock.Now().UTC()
	record.SupersededBy = by
	record.Controls = append(record.Controls, home.ControlEntry{Action: ActSupersede, At: now, Outcome: outcome})
	if record.Discarded == nil {
		record.Discarded = &home.Discard{At: now, Reason: "superseded by " + by + " (the operator's choice at launch)"}
	}
	_, err = s.Writer.WriteLaunch(sup.ID, record)
	return err
}

// settings are the packet's supervisor model and effort, else the defaults,
// and the session idle window from limits.json.
func (s *Service) settings(c *home.Commission) (Settings, error) {
	limits, err := s.Limits()
	if err != nil {
		return Settings{}, err
	}
	idle, err := idleArchiveWindow(limits.Session.IdleArchiveAfter)
	if err != nil {
		return Settings{}, err
	}
	out := Settings{Model: limits.Defaults.SupervisorModel, Effort: limits.Defaults.SupervisorEffort, IdleArchiveAfter: idle}
	if c.Spec != nil && c.Spec.SupervisorModel != "" {
		out.Model = c.Spec.SupervisorModel
	}
	if c.Spec != nil && c.Spec.SupervisorEffort != "" {
		out.Effort = c.Spec.SupervisorEffort
	}
	return out, nil
}

// Record returns a commission's LAUNCH.json (nil when absent) and whether an
// attempt or control is running here.
func (s *Service) Record(ctx context.Context, id string) (*home.LaunchRecord, bool, error) {
	_, c, err := s.find(ctx, id)
	if err != nil {
		return nil, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return c.Launch, s.running[id], nil
}

// MarkInterrupted records every desk launch left unfinished by an earlier
// instance as failed: interrupted. Such an attempt is never resumed.
func (s *Service) MarkInterrupted(ctx context.Context) error {
	if err := s.Writer.Held(); err != nil {
		return err
	}
	snap, err := s.Reader.Snapshot(ctx, home.Options{})
	if err != nil {
		return err
	}
	for _, c := range snap.Commissions {
		r := c.Launch
		if r == nil || r.LaunchedBy != "desk" || r.Failed != nil || r.Discarded != nil || s.isRunning(c.ID) {
			continue
		}
		next := nextStep(r.Steps)
		if next == "" {
			continue
		}
		r.Failed = &home.LaunchFailure{Step: next, Reason: "interrupted"}
		if _, err := s.Writer.WriteLaunch(c.ID, r); err != nil {
			return err
		}
		s.Logger.Printf("launch: %s was interrupted at %s by a desk restart; relaunch it as a new commission", c.ID, next)
	}
	return nil
}

var stepOrder = []string{StepCreated, StepComposerReady, StepStanceSent, StepReadySeen, StepGoalSent, StepGoalVerified}

// nextStep is the step after the last recorded one, or "" when all passed.
func nextStep(steps []home.LaunchStep) string {
	if len(steps) == 0 {
		return StepCreated
	}
	last := steps[len(steps)-1].Step
	for i, name := range stepOrder {
		if name == last && i+1 < len(stepOrder) {
			return stepOrder[i+1]
		}
	}
	return ""
}

// Control actions.
const (
	ActPause        = "pause"
	ActResume       = "resume"
	ActStop         = "stop"
	ActDiscard      = "discard"
	ActResendStance = "resend-stance"
	ActRetryGoal    = "retry-goal"
	// ActSupersede is recorded, never requested: a launch superseded it.
	ActSupersede = "supersede"
)

// ControlOperation returns the delegated operation required by a control.
// Stopping is separate because it ends a session; callers need a grant that
// permits only a closeout stop, not the broader launch/control surface.
func ControlOperation(action string) string {
	if action == ActStop {
		return "research.closeout"
	}
	return "research.control"
}

// ControlResult is the record after a control and the goal status read back.
type ControlResult struct {
	Record     *home.LaunchRecord
	GoalStatus string
	Message    string
}

// controlTime bounds one control. A control outlives a cut request: half a
// stop is worse than a late answer, and LAUNCH.json records the outcome.
const controlTime = 2 * time.Minute

// Control carries out one operator control on a commission's session.
func (s *Service) Control(ctx context.Context, id, action, reason string) (ControlResult, error) {
	if err := wire.ResearchOperation(ctx, ControlOperation(action), id); err != nil {
		return ControlResult{}, err
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), controlTime)
	defer cancel()
	if err := s.Writer.Held(); err != nil {
		return ControlResult{}, err
	}
	if action == ActDiscard && strings.TrimSpace(reason) == "" {
		return ControlResult{}, fmt.Errorf("%w: discard needs the operator's reason, in their words", home.ErrRefused)
	}
	_, c, err := s.find(ctx, id)
	if err != nil {
		return ControlResult{}, err
	}
	if !s.claim(id) {
		return ControlResult{}, fmt.Errorf("%w: a launch or control of %s is already running", home.ErrRefused, id)
	}
	defer s.release(id)

	record := launchRecord(c)
	if record.Discarded != nil {
		return ControlResult{}, fmt.Errorf("%w: %s was discarded at %s", home.ErrRefused, id, record.Discarded.At.Format(time.RFC3339))
	}
	// The commission's own conversation. Nothing else is ever typed into.
	t := targetOf(c)
	open := c.State.Running() || (action == ActRetryGoal && c.Session != nil && c.Session.WindowOpen)
	out := ControlResult{Record: record}
	switch {
	case action == ActDiscard && !open:
	case t.Session == "":
		return out, fmt.Errorf("%w: no session is known for %s (no LAUNCH.json and no exact match)", home.ErrRefused, id)
	case !open && action != ActStop && action != ActRetryGoal:
		return out, fmt.Errorf("%w: %s is %s; its Codex window is not open", home.ErrRefused, id, c.State)
	}
	if action == ActStop {
		status, statusErr := s.Launcher.GoalStatus(ctx, t)
		if statusErr != nil {
			return out, fmt.Errorf("%w: stop requires a readable, paused manager goal after closeout: %v", home.ErrRefused, statusErr)
		}
		if status != "paused" {
			return out, fmt.Errorf("%w: stop is only allowed after the manager goal is paused at closeout; goal status is %q", home.ErrRefused, status)
		}
	}

	// Codex applies /goal pause|resume to a goal in any state, and resuming a
	// completed goal starts it again (measured, launch-proof.txt). Resume
	// also starts a goal halted by a turn error: the desk's recovery pass
	// does it automatically, and this is the steward's or operator's way
	// once that pass escalated.
	if want, ok := map[string]string{ActPause: "active", ActResume: "paused"}[action]; ok {
		status, serr := s.Launcher.GoalStatus(ctx, t)
		if serr != nil {
			return out, fmt.Errorf("%w: %s needs the goal's status, which is unknown: %v", home.ErrRefused, action, serr)
		}
		halted := action == ActResume && status == "blocked" && c.State == home.StateHalted
		if status != want && !halted {
			return out, fmt.Errorf("%w: %s only applies to a goal that is %s, or a halted one; this goal is %q", home.ErrRefused, action, want, status)
		}
	}
	var outcome string // what a stop found and left
	switch action {
	case ActPause:
		out.GoalStatus, err = s.Launcher.Pause(ctx, t)
	case ActResume:
		out.GoalStatus, err = s.Launcher.Resume(ctx, t)
	case ActResendStance:
		msgs, merr := s.Messages(ctx, id)
		if err = merr; err == nil {
			err = s.Launcher.ResendStance(ctx, t, msgs.Stance)
		}
	case ActRetryGoal:
		out.GoalStatus, err = s.retryFailedGoal(ctx, c, record, t)
		if err == nil {
			out.Message = "launch goal recovered without creating another session"
		}
	case ActStop:
		var stopped Stopped
		stopped, err = s.Launcher.Stop(ctx, t, open)
		outcome = stopped.Outcome()
	case ActDiscard:
		if open {
			var stopped Stopped
			stopped, err = s.Launcher.Stop(ctx, t, true)
			outcome = stopped.Outcome()
		}
	default:
		return out, fmt.Errorf("%w: unknown control %q", home.ErrRefused, action)
	}
	if errors.Is(err, ErrNotReady) {
		return out, fmt.Errorf("%w: %s %s: %v", home.ErrRefused, action, id, err)
	}
	if err != nil {
		return out, home.Cause(home.ErrUnavailable, fmt.Errorf("%s %s: %w", action, id, err))
	}
	statusNote := ""
	if out.GoalStatus == "" && t.Thread != "" {
		var gerr error
		if out.GoalStatus, gerr = s.Launcher.GoalStatus(ctx, t); gerr != nil {
			statusNote = gerr.Error()
		}
	}
	now := s.Clock.Now().UTC()
	fillTarget(record, t)
	record.Controls = append(record.Controls, home.ControlEntry{Action: action, At: now, Outcome: outcome})
	if action == ActDiscard {
		record.Discarded = &home.Discard{At: now, Reason: strings.TrimSpace(reason)}
	}
	if _, err := s.Writer.WriteLaunch(id, record); err != nil {
		return out, fmt.Errorf("%s was carried out but LAUNCH.json was not written: %w", action, err)
	}
	switch {
	case out.GoalStatus != "":
		out.Message = action + " done; goal " + out.GoalStatus
	case statusNote != "":
		out.Message = action + " done; goal status unknown: " + statusNote
	default:
		out.Message = action + " done; goal status unknown"
	}
	if action == ActRetryGoal {
		out.Message = fmt.Sprintf("retry-goal done; goal %s; no new session created", out.GoalStatus)
	}
	return out, nil
}

// retryFailedGoal recovers a desk launch that stopped while waiting for ready
// or failed at the goal boundary. At ready_seen, the exact session may not yet
// have a thread in LAUNCH.json, so it resolves only a unique top-level thread
// from that session's Codex transcripts. It then reads exact goal rows first;
// matching work is reconciled without typing and uncertainty fails closed.
func (s *Service) retryFailedGoal(ctx context.Context, c *home.Commission, record *home.LaunchRecord, t Target) (string, error) {
	windowOpen := c.Session != nil && c.Session.WindowOpen
	readyFailure := record.Failed != nil && record.Failed.Step == StepReadySeen
	goalFailure := record.Failed != nil && (record.Failed.Step == StepGoalSent || record.Failed.Step == StepGoalVerified)
	if record.LaunchedBy != "desk" || (!readyFailure && !goalFailure) ||
		record.Discarded != nil || record.SupersededBy != "" {
		return "", fmt.Errorf("%w: retry-goal requires this commission's desk launch failed at ready_seen, goal_sent or goal_verified", home.ErrRefused)
	}
	if t.Session == "" || record.WebConsoleSessionID != t.Session {
		return "", fmt.Errorf("%w: retry-goal requires the launch record's exact session", home.ErrRefused)
	}
	if record.ThreadID != "" && t.Thread != record.ThreadID {
		return "", fmt.Errorf("%w: retry-goal target thread differs from the launch record", home.ErrRefused)
	}
	if readyFailure {
		thread, err := s.Launcher.Sessions.SessionThread(t.Session)
		if err != nil {
			return "", home.Cause(home.ErrUnavailable, fmt.Errorf("resolve the unique top-level thread in the launch's exact session: %w", err))
		}
		if t.Thread != "" && t.Thread != thread {
			return "", fmt.Errorf("%w: resolved top-level thread differs from the commission's target", home.ErrRefused)
		}
		t.Thread = thread
	}
	if t.Thread == "" {
		return "", fmt.Errorf("%w: retry-goal requires a known thread in the launch's exact session", home.ErrRefused)
	}
	if readyFailure {
		if !hasLaunchStep(record.Steps, StepStanceSent) || hasLaunchStep(record.Steps, StepGoalSent) {
			return "", fmt.Errorf("%w: ready_seen recovery requires a recorded stance_sent and no goal_sent step", home.ErrRefused)
		}
	}
	if record.ThreadID == "" {
		record.ThreadID = t.Thread
	}
	msgs, err := s.Messages(ctx, c.ID)
	if err != nil {
		return "", err
	}
	goals, err := s.Launcher.Sessions.SessionGoals(ctx, t.Session)
	if err != nil {
		return "", home.Cause(home.ErrUnavailable, fmt.Errorf("read the exact session's goals before retrying: %w", err))
	}
	var existing *home.Goal
	for i := range goals {
		if goals[i].ThreadID == t.Thread && (goals[i].SessionID == "" || goals[i].SessionID == t.Session) {
			existing = &goals[i]
			break
		}
	}
	if existing != nil {
		if strings.TrimSpace(existing.Objective) != strings.TrimSpace(msgs.Kernel) {
			return "", fmt.Errorf("%w: target thread already has a different goal; no text was sent", home.ErrRefused)
		}
		if existing.Status != "active" && existing.Status != "complete" {
			return existing.Status, fmt.Errorf("%w: matching goal exists with status %q; retry-goal will not resume or replace it", home.ErrRefused, existing.Status)
		}
		record.Steps = ensureGoalSteps(record.Steps, s.Clock.Now().UTC())
		record.Failed = nil
		record.Controls = append(record.Controls, home.ControlEntry{Action: ActRetryGoal, At: s.Clock.Now().UTC(), Outcome: "matching goal already existed; no duplicate sent"})
		if _, err := s.Writer.WriteLaunch(c.ID, record); err != nil {
			return "", err
		}
		return existing.Status, nil
	}
	if !c.State.Running() && !windowOpen {
		return "", fmt.Errorf("%w: no matching goal exists and the exact session is not open; no text was sent", home.ErrRefused)
	}
	thread, sent, err := s.Launcher.RetryGoal(ctx, t, msgs.Kernel)
	if err != nil {
		return "", home.Cause(home.ErrUnavailable, err)
	}
	if thread != t.Thread {
		return "", fmt.Errorf("%w: retry-goal appeared in a different thread %q; launch record unchanged", home.ErrRefused, thread)
	}
	record.Steps = ensureGoalSteps(record.Steps, s.Clock.Now().UTC())
	record.Failed = nil
	outcome := "matching goal appeared during recovery; no duplicate sent"
	if sent {
		outcome = "goal sent once after exact-thread absence was rechecked"
	}
	record.Controls = append(record.Controls, home.ControlEntry{Action: ActRetryGoal, At: s.Clock.Now().UTC(), Outcome: outcome})
	if _, err := s.Writer.WriteLaunch(c.ID, record); err != nil {
		return "", fmt.Errorf("goal ran but launch recovery was not recorded: %w", err)
	}
	return "active", nil
}

func ensureGoalSteps(steps []home.LaunchStep, at time.Time) []home.LaunchStep {
	has := map[string]bool{}
	for _, step := range steps {
		has[step.Step] = true
	}
	if !has[StepReadySeen] {
		steps = append(steps, home.LaunchStep{Step: StepReadySeen, At: at})
	}
	for _, name := range []string{StepGoalSent, StepGoalVerified} {
		if !has[name] {
			steps = append(steps, home.LaunchStep{Step: name, At: at})
		}
	}
	return steps
}

func hasLaunchStep(steps []home.LaunchStep, name string) bool {
	for _, step := range steps {
		if step.Step == name {
			return true
		}
	}
	return false
}

// launchRecord is a commission's LAUNCH.json, or the record a hand-launched
// commission gets at its first control or supersession.
func launchRecord(c *home.Commission) *home.LaunchRecord {
	if c.Launch != nil {
		return c.Launch
	}
	return &home.LaunchRecord{Schema: home.LaunchSchema, CommissionID: c.ID, LaunchedBy: "hand", Steps: []home.LaunchStep{}, Controls: []home.ControlEntry{}}
}

// fillTarget records the session and thread a control or supersession acted
// on when LAUNCH.json does not have them yet; a thread is recorded only for
// the session already on record.
func fillTarget(record *home.LaunchRecord, t Target) {
	if record.WebConsoleSessionID == "" {
		record.WebConsoleSessionID = t.Session
	}
	if record.ThreadID == "" && record.WebConsoleSessionID == t.Session {
		record.ThreadID = t.Thread
	}
}

func (s *Service) find(ctx context.Context, id string) (*home.Snapshot, *home.Commission, error) {
	snap, err := s.Reader.Snapshot(ctx, home.Options{})
	if err != nil {
		return nil, nil, err
	}
	c, err := snap.Find(id)
	return snap, c, err
}

func (s *Service) claim(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running == nil {
		s.running = map[string]bool{}
	}
	if s.running[id] {
		return false
	}
	s.running[id] = true
	return true
}

func (s *Service) release(ids ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		delete(s.running, id)
	}
}

func (s *Service) isRunning(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running[id]
}
