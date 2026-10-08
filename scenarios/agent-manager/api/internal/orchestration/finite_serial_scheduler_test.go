// These disposable scheduler fixtures qualify actual owner transactions/admission
// with a closed dispatcher. They never establish real native/model execution.
package orchestration

import (
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/spawn"
	"agent-manager/internal/repository"
	"context"
	"encoding/json"
	"errors"
	"github.com/vrooli/api-core/effortauthority"
	isolation "github.com/vrooli/vrooli/packages/nativeisolation"
	"testing"
	"time"
)

type serialSchedulerTerminal struct {
	err   error
	reads int
}

func (f *serialSchedulerTerminal) Terminal(context.Context, string) error { f.reads++; return f.err }

type serialSchedulerFixture struct {
	o                            *Orchestrator
	authority                    *effortauthority.Authority
	policy                       effortauthority.Policy
	task                         *domain.Task
	parentProfile, workerProfile *domain.AgentProfile
	root                         *domain.Run
	now                          *time.Time
	terminal                     *serialSchedulerTerminal
}

func newSerialSchedulerFixture(t *testing.T, maxStarts int) *serialSchedulerFixture {
	return newSerialSchedulerFixturePrepared(t, maxStarts, nil)
}
func newSerialSchedulerFixturePrepared(t *testing.T, maxStarts int, prepare func(*Orchestrator, *domain.AgentProfile) *domain.AgentProfile) *serialSchedulerFixture {
	t.Helper()
	ctx := context.Background()
	o, a, p, task, profile, key, now := nativeEffortFixture(t)
	if prepare != nil {
		profile = prepare(o, profile)
	}
	profile.Effort = domain.EffortMedium
	if err := o.profiles.Update(ctx, profile); err != nil {
		t.Fatal(err)
	}
	profile, err := o.profiles.GetByKey(ctx, profile.ProfileKey)
	if err != nil {
		t.Fatal(err)
	}
	worker, err := o.CreateProfile(ctx, &domain.AgentProfile{Name: "Existing bounded worker fixture", ProfileKey: "serial-worker-fixture", RoleRef: profile.RoleRef, Effort: profile.Effort, MaxTurns: profile.MaxTurns, Timeout: profile.Timeout, NetworkAccess: profile.NetworkAccess, SandboxConfig: profile.SandboxConfig, DeclaredScopes: append([]string(nil), profile.DeclaredScopes...)})
	if err != nil {
		t.Fatal(err)
	}
	worker, err = o.profiles.GetByKey(ctx, worker.ProfileKey)
	if err != nil {
		t.Fatal(err)
	}
	p.ID += "-serial"
	p.MaxConcurrent = 1
	p.MaxStarts = maxStarts
	p.AllowRecovery = false
	p.Effects = []string{"run.create", "run.child"}
	p.Profiles = map[string]string{profile.ProfileKey: EffortProfileDigest(profile), worker.ProfileKey: EffortProfileDigest(worker)}
	p.SerialEdges = []effortauthority.SerialEdge{{FromMember: "lead", FromProfile: profile.ProfileKey, ToMember: "lead", ToProfile: worker.ProfileKey}, {FromMember: "lead", FromProfile: worker.ProfileKey, ToMember: "lead", ToProfile: profile.ProfileKey}}
	if err = a.Approve(ctx, "disposable-human-fixture", p); err != nil {
		t.Fatal(err)
	}
	manifest := fixtureNativeManifest(p, profile)
	manifest.ProfileBindings = map[string]string{profile.ProfileKey: p.Profiles[profile.ProfileKey], worker.ProfileKey: p.Profiles[worker.ProfileKey]}
	runtime, err := isolation.NewRuntime(manifest)
	if err != nil {
		t.Fatal(err)
	}
	o.finiteNativeFactory = fixtureNativeAdmission{runtime}
	terminal := &serialSchedulerTerminal{}
	o.finiteNativeTerminal = terminal
	req := effortRequest(t, o, p, task, profile, key, "serial-original-root")
	if _, err = o.CreateRun(ctx, req); !errors.Is(err, spawn.ErrDispatcherClosed) {
		t.Fatal("initial synthetic admission", err)
	}
	root, err := o.runs.GetByIdempotencyKey(ctx, req.IdempotencyKey)
	if err != nil || root == nil {
		t.Fatal(err)
	}
	f := &serialSchedulerFixture{o: o, authority: a, policy: p, task: task, parentProfile: profile, workerProfile: worker, root: root, now: now, terminal: terminal}
	f.complete(t, root, worker.ProfileKey)
	return f
}
func (f *serialSchedulerFixture) complete(t *testing.T, run *domain.Run, nextProfile string) {
	t.Helper()
	run.Status = domain.RunStatusComplete
	command, _ := json.Marshal(map[string]any{"finite_serial": map[string]any{"version": 1, "next_profile": nextProfile, "title": "Bounded next action", "instruction": "Perform the exact accepted task step inside the original scope.", "summary": "Previous bounded step retained."}})
	run.Result = &domain.RunResult{Success: true, ExitCode: 0, FinalOutput: string(command), Selection: domain.FinalOutputSelection{Status: domain.FinalOutputSelectionSelected, SelectedCandidateID: "owner-selected-fixture-output", Rule: "fixture-only", AlgorithmVersion: "fixture-only/1"}, Candidates: []domain.FinalOutputCandidate{{ID: "owner-selected-fixture-output", Content: string(command), Terminal: true, EvidenceTier: 1}}}
	cfg := run.ResolvedConfig
	if cfg == nil || cfg.Admission == nil {
		t.Fatal("actual admission missing")
	}
	// Same existing exact-parent receipt contract; both profiles have identical
	// resolved runner/model/effort. Synthetic receipt is explicitly fixture-only.
	cfg.Admission.Receipt = &domain.QualificationReceipt{Route: cfg.RoleRef, EffectiveRunner: string(cfg.RunnerType), EffectiveModel: cfg.Model, EffectiveEffort: string(cfg.Effort), RuntimeVersion: "disposable-codec-fixture/1", AcceptedOutput: true}
	if err := f.o.runs.Update(context.Background(), run); err != nil {
		t.Fatal(err)
	}
}
func (f *serialSchedulerFixture) successor(t *testing.T, source *domain.Run) *domain.Run {
	t.Helper()
	err := f.o.advanceFiniteSerialEpisode(context.Background(), source)
	if !errors.Is(err, spawn.ErrDispatcherClosed) {
		t.Fatal("expected closed synthetic dispatcher after exact admission", err)
	}
	next, err := f.o.runs.GetByIdempotencyKey(context.Background(), "serial-"+source.ID.String())
	if err != nil || next == nil {
		t.Fatal("missing actual accepted successor", err)
	}
	return next
}
func TestFiniteSerialSchedulerActualAlternatingAcceptedEpisodesAndRestartReplay(t *testing.T) {
	ctx := context.Background()
	f := newSerialSchedulerFixture(t, 3)
	taskBefore, err := f.o.tasks.Get(ctx, f.task.ID)
	if err != nil {
		t.Fatal(err)
	}
	taskDigest := effortauthority.Digest(taskBefore)
	worker := f.successor(t, f.root)
	if worker.ID == f.root.ID || worker.TaskID != f.root.TaskID || worker.ParentRunID == nil || *worker.ParentRunID != f.root.ID || worker.ResolvedConfig.Admission.EffortIntent.Profile != f.workerProfile.ProfileKey {
		t.Fatal("worker lineage/task/profile differs")
	}
	rec, err := f.authority.Store.Get(ctx, f.policy.ID)
	if err != nil || len(rec.Reservations) != 2 || !rec.Reservations[f.root.IdempotencyKey].Terminal {
		t.Fatal("source terminal/budget", err)
	}
	beforeReplay := effortauthority.Digest(rec)
	restarted := *f.o
	if err = restarted.advanceFiniteSerialEpisode(ctx, f.root); err != nil {
		t.Fatal("same accepted successor replay", err)
	}
	replay, err := f.o.runs.GetByIdempotencyKey(ctx, worker.IdempotencyKey)
	if err != nil || replay.ID != worker.ID {
		t.Fatal("replay changed run", err)
	}
	rec, err = f.authority.Store.Get(ctx, f.policy.ID)
	if err != nil || effortauthority.Digest(rec) != beforeReplay {
		t.Fatal("replay charged/changed ledger", err)
	}
	f.complete(t, worker, f.parentProfile.ProfileKey)
	parent := f.successor(t, worker)
	if parent.ID == worker.ID || parent.ID == f.root.ID || parent.TaskID != f.root.TaskID || parent.ParentRunID == nil || *parent.ParentRunID != worker.ID || parent.ResolvedConfig.Admission.EffortIntent.Profile != f.parentProfile.ProfileKey {
		t.Fatal("fresh parent lineage/task/profile differs")
	}
	rec, err = f.authority.Store.Get(ctx, f.policy.ID)
	if err != nil || len(rec.Reservations) != 3 || len(rec.SerialHandoffs) != 2 {
		t.Fatal("three exact accepted episodes", err)
	}
	var turns, tools, seconds int64
	active := 0
	for _, slot := range rec.Reservations {
		turns += int64(slot.Intent.Turns)
		tools += int64(slot.Intent.ToolCalls)
		seconds += slot.Intent.RunSeconds
		if !slot.Terminal {
			active++
		}
	}
	if active != 1 || turns != 3*int64(f.policy.MaxTurns) || tools != 3*int64(f.policy.MaxToolCalls) || seconds != 3*f.policy.MaxRunSeconds {
		t.Fatal("conservation or concurrency widened")
	}
	for _, run := range []*domain.Run{worker, parent} {
		if run.OwnerExpiresAt == nil || !run.OwnerExpiresAt.Equal(f.policy.Deadline) || run.ResolvedConfig.Admission.Effort == nil || !run.ResolvedConfig.Admission.Effort.Deadline.Equal(f.policy.Deadline) {
			t.Fatal("original deadline renewed")
		}
		if len(run.ResolvedConfig.Admission.EffortTaskProjection) == 0 {
			t.Fatal("immutable owner task projection absent")
		}
	}
	taskAfter, err := f.o.tasks.Get(ctx, f.task.ID)
	if err != nil || effortauthority.Digest(taskAfter) != taskDigest {
		t.Fatal("shared task rewritten", err)
	}
}
func TestFiniteSerialSchedulerRefusalsHaveZeroLedgerRunTaskAndProfileEffects(t *testing.T) {
	for _, name := range []string{"unknown-terminal", "absent-terminal-owner", "invalid-result", "unselected-result", "duplicate-selected", "mismatched-selected", "unknown-field", "duplicate-key", "case-alias", "wrong-profile", "changed-task", "changed-profile", "absent-receipt", "missing-source-expiry", "changed-source-expiry", "corrupt-root-task-projection", "no-handoff", "expired", "revoked"} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			f := newSerialSchedulerFixture(t, 3)
			source := f.root
			switch name {
			case "unknown-terminal":
				f.terminal.err = isolation.ErrUnknown
			case "absent-terminal-owner":
				f.o.finiteNativeTerminal = nil
			case "invalid-result":
				source.Result.Success = false
			case "unselected-result":
				source.Result.Selection.SelectedCandidateID = "missing"
			case "duplicate-selected":
				source.Result.Candidates = append(source.Result.Candidates, source.Result.Candidates[0])
			case "mismatched-selected":
				source.Result.Candidates[0].Content = "different"
			case "unknown-field":
				source.Result.FinalOutput = `{"finite_serial":{"version":1,"next_profile":"serial-worker-fixture","title":"Step","instruction":"Do scoped work","summary":"Done","owner":"invented"}}`
			case "duplicate-key":
				source.Result.FinalOutput = `{"finite_serial":{"version":1,"next_profile":"wrong","next_profile":"serial-worker-fixture","title":"Step","instruction":"Do scoped work","summary":"Done"}}`
			case "case-alias":
				source.Result.FinalOutput = `{"finite_serial":{"Version":1,"next_profile":"serial-worker-fixture","title":"Step","instruction":"Do scoped work","summary":"Done"}}`
			case "wrong-profile":
				source.Result.FinalOutput = `{"finite_serial":{"version":1,"next_profile":"not-approved","title":"Step","instruction":"Do scoped work","summary":"Done"}}`
			case "changed-task":
				f.task.Description += " changed owner task"
				if err := f.o.tasks.Update(ctx, f.task); err != nil {
					t.Fatal(err)
				}
			case "changed-profile":
				f.workerProfile.Description = "changed complete profile"
				if err := f.o.profiles.Update(ctx, f.workerProfile); err != nil {
					t.Fatal(err)
				}
			case "absent-receipt":
				source.ResolvedConfig.Admission.Receipt = nil
				source.ResolvedConfig.Admission.RuntimeVersion = ""
				source.ResolvedConfig.Admission.PassedControlArgs = nil
			case "missing-source-expiry":
				source.OwnerExpiresAt = nil
			case "changed-source-expiry":
				changedExpiry := f.policy.Deadline.Add(-time.Hour)
				source.OwnerExpiresAt = &changedExpiry
			case "corrupt-root-task-projection":
				projection := serialTaskRoot{TaskID: f.task.ID.String(), RootTaskDigest: effortauthority.Digest("foreign task snapshot")}
				source.ResolvedConfig.Admission.EffortTaskProjection, _ = json.Marshal(projection)
			case "no-handoff":
				source.Result.FinalOutput = "Final useful work, no next handoff."
			case "expired":
				*f.now = f.policy.Deadline
			case "revoked":
				if err := f.authority.Revoke(ctx, "fixture-human", f.policy.ID); err != nil {
					t.Fatal(err)
				}
			}
			if name == "unknown-field" || name == "duplicate-key" || name == "case-alias" || name == "wrong-profile" || name == "no-handoff" {
				source.Result.Candidates[0].Content = source.Result.FinalOutput
			}
			if err := f.o.runs.Update(ctx, source); err != nil {
				t.Fatal(err)
			}
			rec, err := f.authority.Store.Get(ctx, f.policy.ID)
			if err != nil {
				t.Fatal(err)
			}
			before := effortauthority.Digest(rec)
			task, err := f.o.tasks.Get(ctx, f.task.ID)
			if err != nil {
				t.Fatal(err)
			}
			taskBefore := effortauthority.Digest(task)
			profile, err := f.o.profiles.GetByKey(ctx, f.workerProfile.ProfileKey)
			if err != nil {
				t.Fatal(err)
			}
			profileBefore := EffortProfileDigest(profile)
			runBefore, err := f.o.runs.Get(ctx, source.ID)
			if err != nil {
				t.Fatal(err)
			}
			runDigest := effortauthority.Digest(runBefore)
			if err := f.o.advanceFiniteSerialEpisode(ctx, source); err == nil {
				t.Fatal("refused serial source accepted")
			}
			after, err := f.authority.Store.Get(ctx, f.policy.ID)
			if err != nil || effortauthority.Digest(after) != before {
				t.Fatal("refusal mutated ledger", err)
			}
			next, err := f.o.runs.GetByIdempotencyKey(ctx, "serial-"+source.ID.String())
			if err != nil || next != nil {
				t.Fatal("refusal created successor", err)
			}
			task, err = f.o.tasks.Get(ctx, f.task.ID)
			if err != nil || effortauthority.Digest(task) != taskBefore {
				t.Fatal("refusal changed task", err)
			}
			profile, err = f.o.profiles.GetByKey(ctx, f.workerProfile.ProfileKey)
			if err != nil || EffortProfileDigest(profile) != profileBefore {
				t.Fatal("refusal changed profile", err)
			}
			runAfter, err := f.o.runs.Get(ctx, source.ID)
			if err != nil || effortauthority.Digest(runAfter) != runDigest {
				t.Fatal("refusal changed source run", err)
			}
		})
	}
}
func TestFiniteSerialSchedulerExhaustedStartsNeverCreatesExtraEpisode(t *testing.T) {
	ctx := context.Background()
	f := newSerialSchedulerFixture(t, 1)
	err := f.o.advanceFiniteSerialEpisode(ctx, f.root)
	if err == nil {
		t.Fatal("exhausted starts admitted successor")
	}
	rec, e := f.authority.Store.Get(ctx, f.policy.ID)
	if e != nil || len(rec.Reservations) != 1 {
		t.Fatal("exhausted starts consumed extra reservation", e)
	}
	next, e := f.o.runs.GetByIdempotencyKey(ctx, "serial-"+f.root.ID.String())
	if e != nil || next != nil {
		t.Fatal("exhaustion persisted extra run", e)
	}
	// The real terminal owner's settlement and exact prepared metadata may remain;
	// they are not a new start or a refund. Never conflate these with zero writes.
	if rec.Reservations[f.root.IdempotencyKey].Intent.RunSeconds != f.policy.MaxRunSeconds {
		t.Fatal("terminal source budget refunded")
	}
}

// A row-write refusal after durable reservation is uncertainty, never permission
// to reopen the predecessor or spend a second start under a new key.
type serialSchedulerRefuseCreate struct {
	repository.RunRepository
	calls int
}

func (r *serialSchedulerRefuseCreate) Create(context.Context, *domain.Run) error {
	r.calls++
	return errors.New("disposable row-write refusal before persistence")
}
func TestFiniteSerialSchedulerReservedButMissingRowCrashCutHoldsExactKey(t *testing.T) {
	ctx := context.Background()
	f := newSerialSchedulerFixture(t, 3)
	originalRepo := f.o.runs
	refusing := &serialSchedulerRefuseCreate{RunRepository: originalRepo}
	f.o.runs = refusing
	if err := f.o.advanceFiniteSerialEpisode(ctx, f.root); err == nil {
		t.Fatal("missing row uncertainty accepted")
	}
	if refusing.calls != 1 {
		t.Fatal("expected one original row-write attempt", refusing.calls)
	}
	rec, err := f.authority.Store.Get(ctx, f.policy.ID)
	if err != nil || len(rec.Reservations) != 2 || len(rec.SerialHandoffs) != 1 {
		t.Fatal("exact durable crash-cut state missing", err)
	}
	key := "serial-" + f.root.ID.String()
	pending, ok := rec.Reservations[key]
	if !ok || !pending.NativeBound || pending.RunID != "" || pending.Terminal || !rec.Reservations[f.root.IdempotencyKey].Terminal {
		t.Fatal("pending reservation/source terminal changed")
	}
	before := effortauthority.Digest(rec)
	f.o.runs = originalRepo
	restarted := *f.o
	if err := restarted.advanceFiniteSerialEpisode(ctx, f.root); err == nil {
		t.Fatal("uncertain pending reservation reopened")
	}
	after, err := f.authority.Store.Get(ctx, f.policy.ID)
	if err != nil || effortauthority.Digest(after) != before {
		t.Fatal("uncertain retry consumed/changed ledger", err)
	}
	next, err := originalRepo.GetByIdempotencyKey(ctx, key)
	if err != nil || next != nil {
		t.Fatal("uncertain retry persisted a run", err)
	}
	if refusing.calls != 1 {
		t.Fatal("uncertain retry redispatched row write")
	}
	if len(after.Reservations) != 2 || len(after.SerialHandoffs) != 1 {
		t.Fatal("alternate key/start invented")
	}
}
func TestFiniteSerialSchedulerAbsentOwnerRefusesWithoutPanic(t *testing.T) {
	o := &Orchestrator{}
	if err := o.advanceFiniteSerialEpisode(context.Background(), nil); err != nil {
		t.Fatal("ordinary absent run changed", err)
	}
	if err := o.advanceFiniteSerialEpisode(context.Background(), &domain.Run{}); err != nil {
		t.Fatal("ordinary nonfinite run changed", err)
	}
	run := &domain.Run{ResolvedConfig: &domain.RunConfig{Admission: &domain.RunAdmission{Effort: &effortauthority.Binding{}}}}
	if err := o.advanceFiniteSerialEpisode(context.Background(), run); !errors.Is(err, effortauthority.ErrRefused) {
		t.Fatal("absent owner accepted", err)
	}
}
