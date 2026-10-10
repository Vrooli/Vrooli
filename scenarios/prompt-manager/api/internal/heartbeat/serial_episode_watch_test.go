package heartbeat

import (
	"context"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/effortauthority"
	"prompt-manager/internal/store"
	"strings"
	"testing"
	"time"
)

func serialPMFixture(t *testing.T) (*finiteFixture, *effortauthority.Authority, effortauthority.Policy, effortauthority.Binding, effortauthority.Intent, *store.FiniteLeaderState) {
	t.Helper()
	ctx := context.Background()
	f, a, _, p, _, _ := finiteAuthorityFixture(t, true)
	p.ID += "-serial"
	p.MaxStarts = 3
	p.TotalTurns = 3 * int64(p.MaxTurns)
	p.TotalToolCalls = 3 * int64(p.MaxToolCalls)
	p.TotalRunSeconds = 3 * p.MaxRunSeconds
	p.AllowChildren = true
	p.Effects = []string{"run.create", "run.child"}
	p.Profiles["worker"] = effortauthority.Digest("complete worker profile")
	p.SerialEdges = []effortauthority.SerialEdge{{FromMember: "lead", FromProfile: "qualified-profile", ToMember: "lead", ToProfile: "worker"}, {FromMember: "lead", FromProfile: "worker", ToMember: "lead", ToProfile: "qualified-profile"}}
	if err := a.Approve(ctx, "disposable-human-fixture", p); err != nil {
		t.Fatal(err)
	}
	b := effortauthority.Binding{PolicyID: p.ID, PolicyDigest: effortauthority.Digest(p), Epoch: p.Epoch, Owner: p.Owner, Client: p.Client, Deadline: p.Deadline}
	f.runtime.Efforts.Bindings["committee/lead"] = b
	var state *store.FiniteLeaderState
	if err := f.teams.WithFiniteLeader(ctx, "committee", "lead", func(_ *store.HeartbeatConfig, s *store.FiniteLeaderState, save func() error) error {
		s.ID = uuid.NewString()
		s.TaskID = "retained-parent-task"
		s.RunID = uuid.NewString()
		s.TaskStarted = true
		s.DispatchStarted = true
		s.Status = "running"
		state = s
		return save()
	}); err != nil {
		t.Fatal(err)
	}
	intent := effortauthority.Intent{Audience: effortauthority.Audience, Method: "POST", Endpoint: "/api/v1/runs", PolicyID: p.ID, Epoch: p.Epoch, Client: p.Client, Repository: p.Repository, Effort: p.Effort, Revision: p.Revision, ContentDigest: p.ContentDigest, Team: p.Team, Member: "lead", Profile: "qualified-profile", ProfileDigest: p.Profiles["qualified-profile"], Effect: "run.create", IdempotencyKey: "finite-leader-" + state.ID, InputDigest: effortauthority.Digest("original exact root input"), Turns: p.MaxTurns, ToolCalls: p.MaxToolCalls, RunSeconds: p.MaxRunSeconds}
	if _, _, err := a.Reserve(ctx, b, intent, nil); err != nil {
		t.Fatal(err)
	}
	if err := a.Accept(ctx, b, intent.IdempotencyKey, state.RunID); err != nil {
		t.Fatal(err)
	}
	f.agent.getRuns[state.RunID] = &Run{ID: state.RunID, TaskID: state.TaskID, Status: "running"}
	return f, a, p, b, intent, state
}
func advanceSerialPM(t *testing.T, f *finiteFixture, a *effortauthority.Authority, p effortauthority.Policy, b effortauthority.Binding, source effortauthority.Intent, run string, profile string) (effortauthority.Intent, string) {
	t.Helper()
	ctx := context.Background()
	next := source
	next.IdempotencyKey = "serial-" + run
	next.InputDigest = effortauthority.Digest("exact output and new task " + run)
	next.Effect = "run.child"
	next.ParentRunID = run
	next.Profile = profile
	next.ProfileDigest = p.Profiles[profile]
	if _, _, err := a.PrepareSerialHandoff(ctx, b, source.IdempotencyKey, next); err != nil {
		t.Fatal(err)
	}
	if err := a.Settle(ctx, b, source.IdempotencyKey, run); err != nil {
		t.Fatal(err)
	}
	f.agent.getRuns[run].Status = "complete"
	if _, _, err := a.ReserveSerialEpisode(ctx, b, next.IdempotencyKey); err != nil {
		t.Fatal(err)
	}
	newRun := uuid.NewString()
	if err := a.Accept(ctx, b, next.IdempotencyKey, newRun); err != nil {
		t.Fatal(err)
	}
	task := "retained-parent-task"
	f.agent.getRuns[newRun] = &Run{ID: newRun, TaskID: task, ParentRunID: run, Status: "running"}
	return next, newRun
}
func TestSerialPMFollowsExactAlternatingEpisodesWithoutRootRelaunch(t *testing.T) {
	ctx := context.Background()
	f, a, p, b, root, original := serialPMFixture(t)
	cfgBefore, _ := f.teams.GetHeartbeatConfig(ctx, "committee", "lead")
	cfgDigest := effortauthority.Digest(cfgBefore)
	state, err := f.runtime.Tick(ctx, "committee", "lead")
	if err != nil || len(state.SerialEpisodes) != 1 {
		t.Fatal(err)
	}
	worker, workerRun := advanceSerialPM(t, f, a, p, b, root, original.RunID, "worker")
	state, err = f.runtime.Tick(ctx, "committee", "lead")
	if err != nil || len(state.SerialEpisodes) != 2 || state.SerialCurrentRunID != workerRun {
		t.Fatal("worker projection", err)
	}
	parent, parentRun := advanceSerialPM(t, f, a, p, b, worker, workerRun, "qualified-profile")
	restarted := *f.runtime
	state, err = restarted.ReconcileKnownRun(ctx, "committee", "lead")
	if err != nil || len(state.SerialEpisodes) != 3 || state.SerialCurrentRunID != parentRun {
		t.Fatal("fresh parent projection", err)
	}
	if state.ID != original.ID || state.RunID != original.RunID || state.TaskID != original.TaskID || state.ProfileKey != original.ProfileKey || effortauthority.Digest(state.Binding) != effortauthority.Digest(original.Binding) {
		t.Fatal("original immutable leader association replaced")
	}
	if err = a.Settle(ctx, b, parent.IdempotencyKey, parentRun); err != nil {
		t.Fatal(err)
	}
	f.agent.getRuns[parentRun].Status = "complete"
	if _, err = restarted.Tick(ctx, "committee", "lead"); err != nil {
		t.Fatal("maxStarts terminal read", err)
	}
	if _, err = restarted.Dispatch(ctx, "committee", "lead"); err != nil {
		t.Fatal(err)
	}
	cfgAfter, _ := f.teams.GetHeartbeatConfig(ctx, "committee", "lead")
	if effortauthority.Digest(cfgAfter) != cfgDigest || f.queue.enqueues != 0 || len(f.agent.createTaskCalls) != 0 || len(f.agent.createRunCalls) != 0 {
		t.Fatal("watcher rewrote config or dispatched")
	}
	rec, err := a.Store.Get(ctx, p.ID)
	if err != nil || len(rec.Reservations) != 3 {
		t.Fatal("watcher changed reservations", err)
	}
}
func TestSerialPMMissingUncertainOrWrongOwnerHoldsWithoutProjectionSave(t *testing.T) {
	for _, name := range []string{"missing-handoff", "missing-run", "wrong-task", "wrong-parent", "unknown-status", "false-terminal", "source-unaccepted", "wrong-source-key", "wrong-edge", "wrong-next-input", "duplicate-run", "revoked", "expired", "no-reader"} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			f, a, p, b, root, initial := serialPMFixture(t)
			if name != "missing-handoff" {
				_, _ = advanceSerialPM(t, f, a, p, b, root, initial.RunID, "worker")
			} else {
				if err := a.Settle(ctx, b, root.IdempotencyKey, initial.RunID); err != nil {
					t.Fatal(err)
				}
				f.agent.getRuns[initial.RunID].Status = "complete"
			}
			key := "serial-" + initial.RunID
			switch name {
			case "missing-run":
				delete(f.agent.getRuns, initial.RunID)
			case "wrong-task":
				f.agent.getRuns[initial.RunID].TaskID = "foreign"
			case "wrong-parent":
				f.agent.getRuns[initial.RunID].ParentRunID = "foreign"
			case "unknown-status":
				f.agent.getRuns[initial.RunID].Status = "unknown"
			case "false-terminal":
				f.agent.getRuns[initial.RunID].Status = "running"
			case "revoked":
				if err := a.Revoke(ctx, "disposable-human-fixture", p.ID); err != nil {
					t.Fatal(err)
				}
			case "expired":
				a.Now = func() time.Time { return p.Deadline }
			case "no-reader":
				f.runtime.Efforts.Authority = &serialPMNoReader{Engine: a}
			default:
				if err := a.Store.Update(ctx, p.ID, func(r *effortauthority.Record) error {
					switch name {
					case "source-unaccepted":
						v := r.Reservations[root.IdempotencyKey]
						v.NativeBound = false
						r.Reservations[root.IdempotencyKey] = v
					case "wrong-source-key":
						h := r.SerialHandoffs[key]
						h.SourceKey = "foreign"
						r.SerialHandoffs[key] = h
					case "wrong-edge":
						h := r.SerialHandoffs[key]
						h.Next.Profile = "qualified-profile"
						h.Next.ProfileDigest = p.Profiles[h.Next.Profile]
						r.SerialHandoffs[key] = h
						v := r.Reservations[key]
						v.Intent = h.Next
						v.IntentDigest = effortauthority.Digest(v.Intent)
						r.Reservations[key] = v
					case "wrong-next-input":
						h := r.SerialHandoffs[key]
						h.Next.InputDigest = effortauthority.Digest("foreign")
						r.SerialHandoffs[key] = h
					case "duplicate-run":
						v := r.Reservations[key]
						v.RunID = initial.RunID
						r.Reservations[key] = v
					}
					return nil
				}); err != nil {
					t.Fatal(err)
				}
			}
			before, _ := f.teams.ReadFiniteLeader(ctx, "committee", "lead")
			beforeDigest := effortauthority.Digest(before)
			cfg, _ := f.teams.GetHeartbeatConfig(ctx, "committee", "lead")
			cfgDigest := effortauthority.Digest(cfg)
			if _, err := f.runtime.Tick(ctx, "committee", "lead"); err == nil {
				t.Fatal("uncertain serial chain admitted")
			}
			after, _ := f.teams.ReadFiniteLeader(ctx, "committee", "lead")
			cfg, _ = f.teams.GetHeartbeatConfig(ctx, "committee", "lead")
			if effortauthority.Digest(after) != beforeDigest || effortauthority.Digest(cfg) != cfgDigest || f.queue.enqueues != 0 || len(f.agent.createRunCalls) != 0 || len(f.agent.createTaskCalls) != 0 {
				t.Fatal("held read caused projection/config/dispatch effects")
			}
		})
	}
}

type serialPMNoReader struct{ effortauthority.Engine }

func TestSerialPMPromptUsesOwnerFinalCodecContractWithoutLegacyCLI(t *testing.T) {
	ctx := context.Background()
	f, a, p, _, _, _ := serialPMFixture(t)
	cfg, err := f.teams.GetHeartbeatConfig(ctx, "committee", "lead")
	if err != nil {
		t.Fatal(err)
	}
	qualified, err := f.runtime.prepareFiniteCaller(ctx, "committee", "lead", cfg)
	if err != nil {
		t.Fatal(err)
	}
	guidance, err := finitePromptGuidance(qualified)
	if err != nil || !strings.Contains(guidance, `"finite_serial"`) || !strings.Contains(guidance, `"next_profile"`) || !strings.Contains(guidance, `"worker"`) || strings.Contains(guidance, "agent-manager run park") || strings.Contains(guidance, "agent-manager run identity") {
		t.Fatal("wrong serial doctrine", err)
	}
	legacy, err := finitePromptGuidance(ctx)
	if err != nil || legacy != finiteLeaderGuidance {
		t.Fatal("legacy doctrine changed", err)
	}
	if err = a.Revoke(ctx, "disposable-human-fixture", p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = finitePromptGuidance(qualified); err == nil {
		t.Fatal("revoked caller fell back to legacy doctrine")
	}
}

func TestSerialPMExplicitResetCannotReplaceOriginalChain(t *testing.T) {
	ctx := context.Background()
	f, a, p, b, root, original := serialPMFixture(t)
	_, _ = advanceSerialPM(t, f, a, p, b, root, original.RunID, "worker")
	before, _ := f.teams.ReadFiniteLeader(ctx, "committee", "lead")
	digest := effortauthority.Digest(before)
	if err := f.runtime.Restart(ctx, "committee", "lead", p.Revision, "synthetic-evidence"); err == nil {
		t.Fatal("serial restart reset original identity")
	}
	if err := f.runtime.Reopen(ctx, "committee", "lead", p.Revision, "synthetic-evidence"); err == nil {
		t.Fatal("serial reopen reset original identity")
	}
	after, _ := f.teams.ReadFiniteLeader(ctx, "committee", "lead")
	if effortauthority.Digest(after) != digest || f.queue.enqueues != 0 || len(f.agent.createRunCalls) != 0 {
		t.Fatal("reset refusal had effects")
	}
}
