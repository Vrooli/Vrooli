package resources

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

func updateTestDigest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func updateTestClone(t *testing.T, u ManagedServiceUpdate) ManagedServiceUpdate {
	t.Helper()
	data, err := json.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	var clone ManagedServiceUpdate
	if err := json.Unmarshal(data, &clone); err != nil {
		t.Fatal(err)
	}
	return clone
}

func updateTestInitial(t *testing.T) ManagedServiceUpdate {
	t.Helper()
	m, err := manifestpkg.Load(filepath.Join(projectRoot(t), "resources", "cloudflared", "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	m.ManagedService.UpdatePolicy.ReadinessMetrics = nil // Generic process fixtures do not claim Cloudflare edge connectivity.
	m.ManagedService.Artifact.Version = "2.0.0"
	m.ManagedService.Artifact.SHA256ByPlatform = nil
	m.ManagedService.Artifact.SHA256 = updateTestDigest("new-image")
	target, err := newManagedUpdateContract(m, ManagedServiceArtifact{Resource: m.Name, Path: filepath.Join(t.TempDir(), "new-image"), Version: "2.0.0", SHA256: updateTestDigest("new-image")})
	if err != nil {
		t.Fatal(err)
	}
	m.ManagedService.Artifact.Version = "1.0.0"
	m.ManagedService.Artifact.SHA256 = updateTestDigest("old-image")
	rollback, err := newManagedUpdateContract(m, ManagedServiceArtifact{Resource: m.Name, Path: filepath.Join(t.TempDir(), "retained-image"), Version: "1.0.0", SHA256: updateTestDigest("old-image")})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 5, 21, 0, 0, 0, time.UTC)
	original := ManagedServiceState{InstanceID: "original", OwnershipTokenHash: updateTestDigest("original-token"), PID: 100, Incarnation: "boot:100", ArtifactPath: rollback.Artifact.Path, ArtifactSHA256: rollback.Artifact.SHA256, ArtifactVersion: rollback.Artifact.Version, StartedAt: now.Add(-time.Hour)}
	u := ManagedServiceUpdate{SchemaVersion: 1, ID: "update-fixture", RequestID: "request-fixture", Resource: m.Name, Revision: 1, Phase: UpdatePrepared, CreatedAt: now, UpdatedAt: now, SourceRevision: updateTestDigest("source-revision"), Target: target, Rollback: rollback, Original: original}
	if err := u.Validate(); err != nil {
		t.Fatal(err)
	}
	return u
}

func updateTestIntent(role string, c ManagedUpdateContract) *ManagedUpdateLaunchIntent {
	native, err := platform.NewServiceIntent("resource-update", role, platform.OwnedServiceSpec{Path: c.Artifact.Path, Dir: filepath.Dir(c.Artifact.Path), Env: []string{}, Limits: platform.ServiceLimits{MemoryMaxBytes: 256 << 20, TasksMax: 32, CPUWeight: 100}})
	if err != nil {
		panic(err)
	}
	root := filepath.Dir(c.Artifact.Path)
	r := ManagedUpdateRuntime{Root: root, Home: root, StateDir: filepath.Join(root, "state"), Launcher: c.Artifact.Path, LauncherSHA256: c.Artifact.SHA256, Port: 32042, Limits: platform.ServiceLimits{MemoryMaxBytes: 256 << 20, TasksMax: 32, CPUWeight: 100}}
	return &ManagedUpdateLaunchIntent{InstanceID: role, OwnershipTokenHash: updateTestDigest(role + "-token"), ContractDigest: c.Digest, NativeIntent: native, Runtime: r}
}

func updateTestObserveStarted(p *ManagedUpdateProcess, c ManagedUpdateContract, now time.Time) {
	pid := 200
	if p.Intent.InstanceID == "primary" {
		pid = 300
	}
	if strings.HasPrefix(p.Intent.InstanceID, "restored") {
		pid = 400
	}
	p.State = &ManagedServiceState{InstanceID: p.Intent.InstanceID, OwnershipTokenHash: p.Intent.OwnershipTokenHash, PID: pid, Incarnation: "boot:" + p.Intent.InstanceID, ArtifactPath: c.Artifact.Path, ArtifactSHA256: c.Artifact.SHA256, ArtifactVersion: c.Artifact.Version, StartedAt: now}
	unit, err := p.Intent.NativeIntent.UnitName()
	if err != nil {
		panic(err)
	}
	p.NativeRef = &platform.ServiceRef{Intent: platform.OperationIntent(p.Intent.NativeIntent), Unit: unit, Scope: platform.ScopeRef{Kind: platform.ScopeKindCgroup, Path: "/user.slice/" + platform.ServicesSlice + "/" + unit}, RootPID: pid, RootStartTime: p.Intent.InstanceID, NativeInvocationID: updateTestDigest(p.Intent.InstanceID)[:32]}
}

func updateTestProof(s ManagedServiceState, now time.Time) *ManagedUpdateProof {
	return &ManagedUpdateProof{PID: s.PID, Incarnation: s.Incarnation, InstanceID: s.InstanceID, ArtifactSHA256: s.ArtifactSHA256, ObservedAt: now, Ready: true, OwnedListener: true}
}

type updateTraceStep struct {
	event   ManagedUpdateEvent
	phase   ManagedUpdatePhase
	observe func(*ManagedServiceUpdate, time.Time)
}

func updateSuccessTrace() []updateTraceStep {
	return []updateTraceStep{
		{UpdateLaunchCandidate, UpdateCandidateStarting, func(u *ManagedServiceUpdate, _ time.Time) {
			u.Candidate.Intent = updateTestIntent("candidate", u.Target)
		}},
		{UpdateCandidateStarted, UpdateCandidateVerifying, func(u *ManagedServiceUpdate, at time.Time) {
			updateTestObserveStarted(&u.Candidate, u.Target, at)
		}},
		{UpdateCandidateVerified, UpdateCandidateReady, func(u *ManagedServiceUpdate, at time.Time) {
			u.Candidate.Proof = updateTestProof(*u.Candidate.State, at)
		}},
		{UpdateStopPrimary, UpdatePrimaryStopping, func(u *ManagedServiceUpdate, at time.Time) {
			u.Candidate.Proof = updateTestProof(*u.Candidate.State, at)
			u.Primary.Intent = updateTestIntent("primary", u.Target)
		}},
		{UpdatePrimaryStopped, UpdatePrimaryStarting, func(u *ManagedServiceUpdate, _ time.Time) { u.PrimaryStopVerified = true }},
		{UpdatePrimaryStarted, UpdatePrimaryVerifying, func(u *ManagedServiceUpdate, at time.Time) {
			updateTestObserveStarted(&u.Primary, u.Target, at)
		}},
		{UpdatePrimaryVerified, UpdateCommitting, func(u *ManagedServiceUpdate, at time.Time) { u.Primary.Proof = updateTestProof(*u.Primary.State, at) }},
		{UpdateFinishSuccess, UpdateSucceeded, func(u *ManagedServiceUpdate, at time.Time) {
			u.Primary.Proof = updateTestProof(*u.Primary.State, at)
			u.CandidateQuiescent = true
		}},
	}
}

func updateRestorationTrace() []updateTraceStep {
	return append(updateSuccessTrace()[:6], []updateTraceStep{
		{UpdateFailure, UpdateRestoring, func(u *ManagedServiceUpdate, _ time.Time) { u.Error = "new primary readiness failed" }},
		{UpdateLaunchRestore, UpdateRestoreStarting, func(u *ManagedServiceUpdate, _ time.Time) {
			u.PrimaryQuiescent = true
			u.Restored.Intent = updateTestIntent("restored", u.Rollback)
		}},
		{UpdateRestoreStarted, UpdateRestoreVerifying, func(u *ManagedServiceUpdate, at time.Time) {
			updateTestObserveStarted(&u.Restored, u.Rollback, at)
		}},
		{UpdateRestoreVerified, UpdateRestoreCommitting, func(u *ManagedServiceUpdate, at time.Time) { u.Restored.Proof = updateTestProof(*u.Restored.State, at) }},
		{UpdateFinishRestored, UpdateFailedRestored, func(u *ManagedServiceUpdate, at time.Time) {
			u.Restored.Proof = updateTestProof(*u.Restored.State, at)
			u.CandidateQuiescent = true
		}},
	}...)
}

func updateTestQuiescence(p ManagedUpdateProcess, at time.Time) *ManagedUpdateQuiescence {
	q := &ManagedUpdateQuiescence{InstanceID: p.Intent.InstanceID, OwnershipTokenHash: p.Intent.OwnershipTokenHash, ObservedAt: at, NativeAbsent: true, ProcessesAbsent: true, ListenerAbsent: true}
	if p.State != nil {
		q.PID, q.Incarnation = p.State.PID, p.State.Incarnation
	}
	return q
}

func updateRetryRestorationTrace(acknowledged bool) []updateTraceStep {
	count := 8 // Intent committed, dispatch acknowledgement lost.
	if acknowledged {
		count = 9 // Process receipt committed, process then exited.
	}
	trace := append([]updateTraceStep{}, updateRestorationTrace()[:count]...)
	trace = append(trace,
		updateTraceStep{UpdateRecoveryFailed, UpdateRecoveryNeeded, nil},
		updateTraceStep{UpdateRetryRecovery, UpdateRestoring, nil},
		updateTraceStep{UpdateRetireRestore, UpdateRestoring, func(u *ManagedServiceUpdate, at time.Time) {
			u.Restored.Quiescence = updateTestQuiescence(u.Restored, at)
		}},
		updateTraceStep{UpdateLaunchRestore, UpdateRestoreStarting, func(u *ManagedServiceUpdate, _ time.Time) {
			u.Restored.Intent = updateTestIntent("restored-second", u.Rollback)
		}},
		updateTraceStep{UpdateRestoreStarted, UpdateRestoreVerifying, func(u *ManagedServiceUpdate, at time.Time) {
			updateTestObserveStarted(&u.Restored, u.Rollback, at)
		}},
	)
	return append(trace, updateRestorationTrace()[9:]...)
}

func updateRunTrace(t *testing.T, u ManagedServiceUpdate, trace []updateTraceStep) []ManagedServiceUpdate {
	t.Helper()
	states := []ManagedServiceUpdate{updateTestClone(t, u)}
	for _, step := range trace {
		at := u.UpdatedAt.Add(time.Second)
		if step.observe != nil {
			step.observe(&u, at)
		}
		next, err := transitionManagedUpdate(u, step.event, at)
		if err != nil {
			t.Fatalf("%s from %s: %v", step.event, u.Phase, err)
		}
		if next.Phase != step.phase {
			t.Fatalf("%s reached %s, want %s", step.event, next.Phase, step.phase)
		}
		u = updateTestClone(t, next)
		states = append(states, u)
	}
	return states
}

func TestManagedUpdateTracesPreserveServiceAtEveryTerminalOutcome(t *testing.T) {
	for name, trace := range map[string][]updateTraceStep{
		"successful promotion":                       updateSuccessTrace(),
		"failed update restored":                     updateRestorationTrace(),
		"rollback process failed then restored":      updateRetryRestorationTrace(true),
		"rollback dispatch reply lost then restored": updateRetryRestorationTrace(false),
		"failed candidate preserves original": {
			{UpdateFailure, UpdatePreserving, func(u *ManagedServiceUpdate, _ time.Time) { u.Error = "candidate failed before dispatch" }},
			{UpdateOriginalVerified, UpdatePreserveCommitting, func(u *ManagedServiceUpdate, at time.Time) { u.OriginalProof = updateTestProof(u.Original, at) }},
			{UpdateFinishPreserved, UpdateFailedPreserved, func(u *ManagedServiceUpdate, at time.Time) {
				u.CandidateQuiescent = true
				u.OriginalProof = updateTestProof(u.Original, at)
			}},
		},
		"abort preserves original": {
			{UpdateBeginAbort, UpdatePreserving, nil},
			{UpdateOriginalVerified, UpdatePreserveCommitting, func(u *ManagedServiceUpdate, at time.Time) { u.OriginalProof = updateTestProof(u.Original, at) }},
			{UpdateFinishPreserved, UpdateAborted, func(u *ManagedServiceUpdate, at time.Time) {
				u.CandidateQuiescent = true
				u.OriginalProof = updateTestProof(u.Original, at)
			}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			states := updateRunTrace(t, updateTestInitial(t), trace)
			last := states[len(states)-1]
			if !last.Phase.Terminal() {
				t.Fatal("trace did not finish")
			}
			if err := last.Validate(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestManagedUpdateCannotStopOriginalOnStaleOrUnownedCandidateProof(t *testing.T) {
	states := updateRunTrace(t, updateTestInitial(t), updateSuccessTrace()[:3])
	ready := states[len(states)-1]
	for name, change := range map[string]func(*ManagedServiceUpdate, time.Time){
		"no proof": func(u *ManagedServiceUpdate, _ time.Time) { u.Candidate.Proof = nil },
		"stale proof": func(u *ManagedServiceUpdate, at time.Time) {
			u.Candidate.Proof.ObservedAt = at.Add(-managedUpdateProofMaxAge - time.Nanosecond)
		},
		"future proof": func(u *ManagedServiceUpdate, at time.Time) { u.Candidate.Proof.ObservedAt = at.Add(time.Nanosecond) },
		"before launch": func(u *ManagedServiceUpdate, _ time.Time) {
			u.Candidate.Proof.ObservedAt = u.Candidate.State.StartedAt.Add(-time.Nanosecond)
		},
		"other instance": func(u *ManagedServiceUpdate, _ time.Time) { u.Candidate.Proof.InstanceID = "unrelated" },
		"other artifact": func(u *ManagedServiceUpdate, _ time.Time) {
			u.Candidate.Proof.ArtifactSHA256 = updateTestDigest("unrelated")
		},
		"unowned listener":       func(u *ManagedServiceUpdate, _ time.Time) { u.Candidate.Proof.OwnedListener = false },
		"not ready":              func(u *ManagedServiceUpdate, _ time.Time) { u.Candidate.Proof.Ready = false },
		"missing primary intent": func(u *ManagedServiceUpdate, _ time.Time) { u.Primary.Intent = nil },
	} {
		t.Run(name, func(t *testing.T) {
			u := updateTestClone(t, ready)
			at := u.UpdatedAt.Add(time.Second)
			u.Candidate.Proof = updateTestProof(*u.Candidate.State, at)
			u.Primary.Intent = updateTestIntent("primary", u.Target)
			change(&u, at)
			if _, err := transitionManagedUpdate(u, UpdateStopPrimary, at); err == nil {
				t.Fatal("unsafe original stop admitted")
			}
		})
	}
}

func TestManagedUpdateRoleRequiresMatchingNativeIntentProcessAndScope(t *testing.T) {
	states := updateRunTrace(t, updateTestInitial(t), updateSuccessTrace()[:2])
	started := states[len(states)-1]
	for name, change := range map[string]func(*ManagedUpdateProcess){
		"missing native receipt": func(p *ManagedUpdateProcess) { p.NativeRef = nil },
		"bounded job intent":     func(p *ManagedUpdateProcess) { p.Intent.NativeIntent.Kind = "" },
		"another owner":          func(p *ManagedUpdateProcess) { p.Intent.NativeIntent.Owner = "other-owner" },
		"another attempt":        func(p *ManagedUpdateProcess) { p.Intent.NativeIntent.AttemptID = "other-attempt" },
		"another invocation":     func(p *ManagedUpdateProcess) { p.NativeRef.Intent.InvocationID = updateTestDigest("other")[:32] },
		"another root PID":       func(p *ManagedUpdateProcess) { p.NativeRef.RootPID++ },
		"reused PID":             func(p *ManagedUpdateProcess) { p.NativeRef.RootStartTime = "another-incarnation" },
		"foreign namespace": func(p *ManagedUpdateProcess) {
			p.NativeRef.Scope.Path = "/user.slice/vrooli-agents.slice/" + p.NativeRef.Unit
		},
		"partial inode": func(p *ManagedUpdateProcess) { p.NativeRef.ScopeDevice = 1 },
	} {
		t.Run(name, func(t *testing.T) {
			u := updateTestClone(t, started)
			change(&u.Candidate)
			if err := u.Validate(); err == nil {
				t.Fatal("role receipt accepted native identity unrelated to its persisted launch")
			}
		})
	}
}

func TestManagedUpdateRecoveryFailureKeepsCandidateAndResumesSameIdentity(t *testing.T) {
	trace := updateRestorationTrace()[:10]
	states := updateRunTrace(t, updateTestInitial(t), trace)
	u := states[len(states)-1]
	// Even after rollback readiness, a cleanup failure is unresolved recovery,
	// not update success and not permission to forget the working candidate.
	at := u.UpdatedAt.Add(time.Second)
	u, err := transitionManagedUpdate(u, UpdateRecoveryFailed, at)
	if err != nil {
		t.Fatal(err)
	}
	if u.Phase.Terminal() || u.CandidateQuiescent || u.RecoveryFrom != UpdateRestoring {
		t.Fatalf("recovery failure discarded route: %+v", u)
	}
	originalIntent := *u.Restored.Intent
	u, err = transitionManagedUpdate(updateTestClone(t, u), UpdateRetryRecovery, at.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if *u.Restored.Intent != originalIntent || u.Phase != UpdateRestoring {
		t.Fatal("recovery replay changed native identity")
	}
}

func TestManagedUpdateRetirementRequiresFreshCompleteExactExclusion(t *testing.T) {
	trace := updateRetryRestorationTrace(true)
	states := updateRunTrace(t, updateTestInitial(t), trace[:11])
	restoring := states[len(states)-1]
	if restoring.Phase != UpdateRestoring || restoring.Restored.State == nil {
		t.Fatal("fixture did not retain failed restoration")
	}
	for name, change := range map[string]func(*ManagedUpdateQuiescence){
		"wrong identity":         func(q *ManagedUpdateQuiescence) { q.InstanceID = "foreign" },
		"wrong token":            func(q *ManagedUpdateQuiescence) { q.OwnershipTokenHash = updateTestDigest("foreign") },
		"wrong incarnation":      func(q *ManagedUpdateQuiescence) { q.Incarnation = "reused-pid" },
		"wrong PID":              func(q *ManagedUpdateQuiescence) { q.PID++ },
		"unit still exists":      func(q *ManagedUpdateQuiescence) { q.NativeAbsent = false },
		"descendants unexcluded": func(q *ManagedUpdateQuiescence) { q.ProcessesAbsent = false },
		"listener unexcluded":    func(q *ManagedUpdateQuiescence) { q.ListenerAbsent = false },
		"future evidence":        func(q *ManagedUpdateQuiescence) { q.ObservedAt = q.ObservedAt.Add(time.Second) },
		"stale evidence":         func(q *ManagedUpdateQuiescence) { q.ObservedAt = q.ObservedAt.Add(-time.Minute) },
	} {
		t.Run(name, func(t *testing.T) {
			u := updateTestClone(t, restoring)
			at := u.UpdatedAt.Add(time.Second)
			u.Restored.Quiescence = updateTestQuiescence(u.Restored, at)
			change(u.Restored.Quiescence)
			if _, err := transitionManagedUpdate(u, UpdateRetireRestore, at); err == nil {
				t.Fatal("unproven restoration exclusion permitted a replacement identity")
			}
		})
	}
	states = updateRunTrace(t, restoring, trace[11:12])
	u := states[len(states)-1]
	if u.Restored.Intent != nil || len(u.RetiredRestores) != 1 || u.RetiredRestores[0].Process.Intent.InstanceID != restoring.Restored.Intent.InstanceID || u.CandidateQuiescent {
		t.Fatal("retirement forgot identity history or discarded the serving candidate")
	}
	u.Restored.Intent = restoring.Restored.Intent
	if _, err := transitionManagedUpdate(u, UpdateLaunchRestore, u.UpdatedAt.Add(time.Second)); err == nil {
		t.Fatal("retired ownership identity was reusable")
	}
}

func TestManagedUpdateRestorationAttemptsRemainBoundedAndRecoverable(t *testing.T) {
	states := updateRunTrace(t, updateTestInitial(t), updateRestorationTrace()[:7])
	u := states[len(states)-1]
	for attempt := 0; attempt < managedUpdateRestoreAttempts; attempt++ {
		trace := []updateTraceStep{
			{UpdateLaunchRestore, UpdateRestoreStarting, func(u *ManagedServiceUpdate, _ time.Time) {
				u.PrimaryQuiescent = true
				u.Restored.Intent = updateTestIntent("restore-attempt-"+string(rune('a'+attempt)), u.Rollback)
			}},
			{UpdateRecoveryFailed, UpdateRecoveryNeeded, nil},
			{UpdateRetryRecovery, UpdateRestoring, nil},
		}
		states = updateRunTrace(t, u, trace)
		u = states[len(states)-1]
		at := u.UpdatedAt.Add(time.Second)
		u.Restored.Quiescence = updateTestQuiescence(u.Restored, at)
		next, err := transitionManagedUpdate(u, UpdateRetireRestore, at)
		if attempt == managedUpdateRestoreAttempts-1 {
			if err == nil || u.CandidateQuiescent || u.Phase.Terminal() || len(u.RetiredRestores) != managedUpdateRestoreAttempts-1 {
				t.Fatal("exhausted recovery lost history or admitted unlimited replacements")
			}
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		u = updateTestClone(t, next)
	}
}

func TestManagedUpdateTerminalReceiptRequiresServingEvidence(t *testing.T) {
	for name, phase := range map[string]ManagedUpdatePhase{"success": UpdateSucceeded, "preserved": UpdateFailedPreserved, "restored": UpdateFailedRestored, "abort": UpdateAborted} {
		t.Run(name, func(t *testing.T) {
			u := updateTestInitial(t)
			u.Phase = phase
			u.Error = "failure"
			u.AbortRequested = phase == UpdateAborted
			u.CandidateQuiescent = true
			if err := u.Validate(); err == nil {
				t.Fatal("terminal receipt accepted without a verified serving process")
			}
		})
	}
	u := updateTestInitial(t)
	u.Error = "candidate failed"
	u.Phase = UpdatePreserving
	u.CandidateQuiescent = true
	if _, err := transitionManagedUpdate(u, UpdateFinishPreserved, u.UpdatedAt.Add(time.Second)); err == nil {
		t.Fatal("preserved claim accepted without original readiness proof")
	}
}

func TestManagedUpdateLaunchContractsFreezeSourceAndDoNotCaptureEnvironment(t *testing.T) {
	t.Setenv("VROOLI_UPDATE_TEST_SECRET", "must-never-be-in-receipt")
	u := updateTestInitial(t)
	original := u.Target.Manifest
	frozen, err := newManagedUpdateContract(original, u.Target.Artifact)
	if err != nil {
		t.Fatal(err)
	}
	original.ManagedService.Arguments[0] = "mutated-source"
	if frozen.Digest != frozen.digest() || frozen.Manifest.ManagedService.Arguments[0] == "mutated-source" {
		t.Fatal("source mutation changed retained contract")
	}
	data, err := json.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "must-never-be-in-receipt") || strings.Contains(string(data), "process_environment") {
		t.Fatal("receipt captured caller environment")
	}
	frozen.Manifest.ManagedService.Arguments[0] = "tampered-checkpoint"
	if err := frozen.validate(); err == nil {
		t.Fatal("tampered launch contract accepted")
	}
}

func TestManagedUpdateRejectsIllegalPhaseEventPairs(t *testing.T) {
	legal := map[ManagedUpdatePhase][]ManagedUpdateEvent{
		UpdatePrepared:           {UpdateLaunchCandidate, UpdateFailure, UpdateBeginAbort},
		UpdateCandidateStarting:  {UpdateCandidateStarted, UpdateFailure, UpdateBeginAbort},
		UpdateCandidateVerifying: {UpdateCandidateVerified, UpdateFailure, UpdateBeginAbort},
		UpdateCandidateReady:     {UpdateStopPrimary, UpdateFailure, UpdateBeginAbort},
		UpdatePrimaryStopping:    {UpdatePrimaryStopped, UpdateFailure, UpdateBeginAbort},
		UpdatePrimaryStarting:    {UpdatePrimaryStarted, UpdateFailure, UpdateBeginAbort},
		UpdatePrimaryVerifying:   {UpdatePrimaryVerified, UpdateFailure, UpdateBeginAbort},
		UpdateCommitting:         {UpdateFinishSuccess, UpdateFailure, UpdateBeginAbort},
		UpdatePreserving:         {UpdateOriginalVerified, UpdateRestoreRequired, UpdateRecoveryFailed},
		UpdatePreserveCommitting: {UpdateFinishPreserved, UpdateRestoreRequired, UpdateRecoveryFailed},
		UpdateRestoring:          {UpdateLaunchRestore, UpdateRecoveryFailed, UpdateRetireRestore},
		UpdateRestoreStarting:    {UpdateRestoreStarted, UpdateRecoveryFailed},
		UpdateRestoreVerifying:   {UpdateRestoreVerified, UpdateRecoveryFailed},
		UpdateRestoreCommitting:  {UpdateFinishRestored, UpdateRecoveryFailed},
		UpdateRecoveryNeeded:     {UpdateRetryRecovery},
	}
	events := []ManagedUpdateEvent{UpdateLaunchCandidate, UpdateCandidateStarted, UpdateCandidateVerified, UpdateStopPrimary, UpdatePrimaryStopped, UpdatePrimaryStarted, UpdatePrimaryVerified, UpdateFinishSuccess, UpdateFailure, UpdateOriginalVerified, UpdateFinishPreserved, UpdateRestoreRequired, UpdateLaunchRestore, UpdateRestoreStarted, UpdateRestoreVerified, UpdateFinishRestored, UpdateRecoveryFailed, UpdateRetryRecovery, UpdateRetireRestore, UpdateBeginAbort, "unknown"}
	states := updateRunTrace(t, updateTestInitial(t), updateSuccessTrace())
	states = append(states, updateRunTrace(t, updateTestInitial(t), updateRestorationTrace())...)
	preserved := updateTestInitial(t)
	preserved.Error = "candidate failed"
	preserved.Phase = UpdatePreserving
	states = append(states, preserved)
	recovery, err := transitionManagedUpdate(preserved, UpdateRecoveryFailed, preserved.UpdatedAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	states = append(states, recovery)
	for _, state := range states {
		for _, event := range events {
			allowed := false
			for _, candidate := range legal[state.Phase] {
				if candidate == event {
					allowed = true
				}
			}
			if allowed {
				continue
			}
			t.Run(string(state.Phase)+"/"+string(event), func(t *testing.T) {
				u := updateTestClone(t, state)
				u.Error = "injected failure"
				if _, err := transitionManagedUpdate(u, event, u.UpdatedAt.Add(time.Second)); err == nil {
					t.Fatal("illegal transition admitted")
				}
			})
		}
	}
}
