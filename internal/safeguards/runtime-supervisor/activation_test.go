package runtimesupervisorsafeguard

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/coreactivation"
)

type activationFenceFixture struct {
	calls   *[]string
	failure error
}

func (f activationFenceFixture) Stopped(context.Context) error {
	*f.calls = append(*f.calls, "quiescent")
	return f.failure
}
func (f activationFenceFixture) Close() error { return nil }

func writeActivationRecordFixture(t *testing.T, s *coreactivation.Store, r coreactivation.Record) {
	t.Helper()
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := config.WriteOwnedFileDurable(filepath.Join(s.Directory(), "initialized"), []byte("vrooli-core-activation-store-v1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := config.WriteOwnedFileDurable(filepath.Join(s.Directory(), "current.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, exists, err := s.Current(); err != nil || !exists {
		t.Fatalf("invalid operation fixture: %v", err)
	}
}

func activationFixture(t *testing.T) (ActivationEffects, coreactivation.Record, coreactivation.Layout, *[]string, *platform.CoreServiceState) {
	t.Helper()
	ctx := context.Background()
	home := t.TempDir()
	s, err := coreactivation.NewStoreAt(home, filepath.Join(home, "state", "core-activation"))
	if err != nil {
		t.Fatal(err)
	}
	unit, _ := platform.CoreUnitByID(platform.CoreUnitRuntimeSupervisor)
	l := coreactivation.Layout{Home: home, Executable: filepath.Join(home, "bin", "vrooli"), UnitPath: platform.RuntimeSupervisorUnitPath("linux", home), UnitName: unit.Systemd, LogPath: filepath.Join(home, "logs", "supervisor.log")}
	root := filepath.Join(home, "source")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"prior", "candidate", "worker"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name+"-image"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	id := strings.Repeat("1", 32)
	var d coreactivation.InputDigests
	if err := s.Prepare(ctx, func(p *coreactivation.Preparer) error {
		var err error
		d, err = p.Freeze(ctx, id, l, coreactivation.InputSources{Prior: filepath.Join(root, "prior"), Candidate: filepath.Join(root, "candidate"), Worker: filepath.Join(root, "worker"), PriorUnit: []byte("prior-unit\n"), CandidateUnit: []byte("candidate-unit\n")})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	spec := platform.OperationSpec{Path: filepath.Join(s.Directory(), "inputs", id, "worker"), Dir: home, Env: []string{}, Limits: platform.OperationLimits{MemoryMaxBytes: 128 << 20, TasksMax: 16, CPUWeight: 10, WallTime: time.Minute}}
	intent, err := platform.NewOperationIntent("core-activation", id, spec)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(intent)
	sum := sha256.Sum256(encoded)
	workerUnit := "vrooli-op-core-activation-" + hex.EncodeToString(sum[:12]) + ".service"
	ref := platform.OperationRef{Intent: intent, Unit: workerUnit, Scope: platform.ScopeRef{Kind: platform.ScopeKindCgroup, Path: "/user.slice/fixture/" + platform.OperationsSlice + "/" + workerUnit}, RootPID: 123, RootStartTime: "12345", NativeInvocationID: strings.Repeat("4", 32)}
	boot := "01234567-89ab-cdef-0123-456789abcdef"
	r := coreactivation.Record{SchemaVersion: 1, ID: id, Home: home, SourceRoot: root, Revision: 1, Phase: coreactivation.Publishing, Fence: coreactivation.Claimed, CandidateSHA: d.Candidate, PriorSHA: d.Prior, WorkerSHA: d.Worker, LayoutSHA: d.Layout, CandidateUnitSHA: d.CandidateUnit, PriorUnitSHA: d.PriorUnit, CandidateBuild: strings.Repeat("b", 64), PriorBuild: strings.Repeat("f", 64), CandidateStateContract: strings.Repeat("c", 64), PriorStateContract: strings.Repeat("c", 64), PriorPID: 456, PriorIncarnation: boot + ":456", PriorNativeInvocation: strings.Repeat("5", 32), CreatedUnixNano: time.Now().UnixNano(), DeadlineUnixNano: time.Now().Add(time.Minute).UnixNano(), WorkerIntent: intent, WorkerRef: &ref, DispatchCommitted: true, FailedReleases: []string{}}
	if err := config.WriteOwnedFileDurable(l.Executable, []byte("prior-image"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := config.WriteOwnedFileDurable(l.UnitPath, []byte("prior-unit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r.NativeOverridesDigest, _ = coreactivation.NativeOverridesDigest(ctx, "")
	writeActivationRecordFixture(t, s, r)
	state := &platform.CoreServiceState{Unit: unit.Systemd, LoadState: "loaded", ActiveState: "active", SubState: "running", ControlGroup: "/user.slice/fixture/app.slice/" + unit.Systemd, InvocationID: r.PriorNativeInvocation, FragmentPath: l.UnitPath, KillMode: "control-group", NeedDaemonReload: "no", PID: r.PriorPID}
	calls := &[]string{}
	ops := &activationOps{inspect: func(context.Context) (platform.CoreServiceState, error) { return *state, nil }, command: func(_ context.Context, args ...string) error {
		*calls = append(*calls, strings.Join(args, " "))
		switch args[0] {
		case "restart":
			state.PID = 789
			state.InvocationID = strings.Repeat("6", 32)
		case "stop":
			state.PID = 0
			state.ActiveState = "inactive"
		case "start":
			state.PID = 456
			state.InvocationID = strings.Repeat("7", 32)
			state.ActiveState = "active"
		}
		return nil
	}, validate: func(platform.RenderedArtifact, platform.Scope) platform.Verdict {
		return platform.Verdict{State: platform.VerdictAccepted}
	}, lock: func(context.Context, string) (func(), error) { return func() {}, nil }, image: func(_ context.Context, pid int) (string, error) {
		if pid == 789 {
			return r.CandidateSHA, nil
		}
		return r.PriorSHA, nil
	}, incarnation: func(pid int) (string, error) { return fmt.Sprintf("%s:%d", boot, pid), nil }, worker: func(context.Context, coreactivation.Record) error { return nil }, exclude: func(context.Context, coreactivation.Record) error { return nil }, release: func(context.Context, coreactivation.Record) error { return nil }, pin: func(context.Context, platform.CoreServiceState) (supervisorStopFence, error) {
		*calls = append(*calls, "pin")
		return activationFenceFixture{calls: calls}, nil
	}, lease: func(context.Context, string) (supervisorLease, error) {
		build := r.PriorBuild
		if state.PID == 789 {
			build = r.CandidateBuild
		}
		return supervisorLease{PID: state.PID, Build: build, Incarnation: fmt.Sprintf("%s:%d", boot, state.PID), Started: time.Now().Add(-2 * time.Second), Heartbeat: time.Now(), Deadline: time.Now().Add(30 * time.Second), FleetRenewed: true}, nil
	}}
	return ActivationEffects{Store: s, ops: ops}, r, l, calls, state
}

func TestFrozenActivationPublishesAndRestoresOnlyAfterNativeTreeExclusion(t *testing.T) {
	f, r, l, calls, _ := activationFixture(t)
	ctx := context.Background()
	if err := f.PublishCandidate(ctx, r); err != nil {
		t.Fatal(err)
	}
	r.Phase = coreactivation.Activating
	writeActivationRecordFixture(t, f.Store, r)
	if err := f.ActivateCandidate(ctx, r); err != nil {
		t.Fatal(err)
	}
	r.Phase = coreactivation.Verifying
	writeActivationRecordFixture(t, f.Store, r)
	if err := f.VerifyCandidate(ctx, r); err != nil {
		t.Fatal(err)
	}
	r.Phase, r.Fence, r.FailureClass, r.RestorationRequired = coreactivation.Restoring, coreactivation.Revoked, "verification", true
	writeActivationRecordFixture(t, f.Store, r)
	if err := f.RestorePrior(ctx, r); err != nil {
		t.Fatal(err)
	}
	if err := f.assets().VerifyInstalled(ctx, r, false); err != nil {
		t.Fatal(err)
	}
	want := []string{"daemon-reload", "reset-failed " + l.UnitName, "restart " + l.UnitName, "pin", "stop " + l.UnitName, "quiescent", "daemon-reload", "reset-failed " + l.UnitName, "start " + l.UnitName}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("unsafe native effect order: %v", *calls)
	}
}

func TestFrozenActivationPreservesLoadedOperatorOverride(t *testing.T) {
	f, r, l, _, state := activationFixture(t)
	ctx := context.Background()
	path := filepath.Join(filepath.Dir(l.UnitPath), l.UnitName+".d", "owner.conf")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	const content = "[Service]\nEnvironment=OWNER_POLICY=preserved\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	state.DropInPaths = path
	r.NativeOverridesDigest, _ = coreactivation.NativeOverridesDigest(ctx, path)
	writeActivationRecordFixture(t, f.Store, r)
	if err := f.PublishCandidate(ctx, r); err != nil {
		t.Fatal(err)
	}
	r.Phase = coreactivation.Activating
	writeActivationRecordFixture(t, f.Store, r)
	if err := f.ActivateCandidate(ctx, r); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != content {
		t.Fatal("activation changed operator override")
	}
	if err := os.WriteFile(path, []byte("new operator policy"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := f.ActivateCandidate(ctx, r); err == nil {
		t.Fatal("changed operator override did not fence native effects")
	}
}

func TestFrozenRestorationPreservesBytesWhenQuiescenceOrOwnershipIsUnknown(t *testing.T) {
	for _, mode := range []string{"quiescence", "foreign_unit", "foreign_image", "dropin", "foreign_process"} {
		t.Run(mode, func(t *testing.T) {
			f, r, l, calls, state := activationFixture(t)
			ctx := context.Background()
			if err := f.PublishCandidate(ctx, r); err != nil {
				t.Fatal(err)
			}
			r.Phase, r.Fence, r.FailureClass, r.RestorationRequired = coreactivation.Restoring, coreactivation.Revoked, "activation", true
			writeActivationRecordFixture(t, f.Store, r)
			switch mode {
			case "quiescence":
				f.ops.pin = func(context.Context, platform.CoreServiceState) (supervisorStopFence, error) {
					return activationFenceFixture{calls: calls, failure: errors.New("unknown tree")}, nil
				}
			case "foreign_unit":
				if err := os.WriteFile(l.UnitPath, []byte("operator unit"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "foreign_image":
				if err := os.WriteFile(l.Executable, []byte("operator image"), 0o700); err != nil {
					t.Fatal(err)
				}
			case "dropin":
				state.DropInPaths = "operator.conf"
			case "foreign_process":
				f.ops.image = func(context.Context, int) (string, error) { return strings.Repeat("9", 64), nil }
			}
			image, _ := os.ReadFile(l.Executable)
			unit, _ := os.ReadFile(l.UnitPath)
			if err := f.RestorePrior(ctx, r); err == nil {
				t.Fatal("unqualified restoration succeeded")
			}
			afterImage, _ := os.ReadFile(l.Executable)
			afterUnit, _ := os.ReadFile(l.UnitPath)
			if string(image) != string(afterImage) || string(unit) != string(afterUnit) {
				t.Fatal("unknown ownership changed installed bytes")
			}
			if mode != "quiescence" && len(*calls) != 0 {
				t.Fatalf("foreign ownership reached native effects: %v", *calls)
			}
		})
	}
}

func TestFrozenActivationRechecksFenceAndRejectsUnknownNativeValidation(t *testing.T) {
	for _, mode := range []string{"revoked", "deadline", "validator_unavailable", "revoked_after_reload"} {
		t.Run(mode, func(t *testing.T) {
			f, r, _, calls, _ := activationFixture(t)
			ctx := context.Background()
			if err := f.PublishCandidate(ctx, r); err != nil {
				t.Fatal(err)
			}
			r.Phase = coreactivation.Activating
			if mode == "revoked" {
				r.Phase, r.Fence, r.FailureClass = coreactivation.Restoring, coreactivation.Revoked, "activation"
			}
			if mode == "deadline" {
				r.CreatedUnixNano = time.Now().Add(-time.Minute).UnixNano()
				r.DeadlineUnixNano = time.Now().Add(-time.Second).UnixNano()
			}
			writeActivationRecordFixture(t, f.Store, r)
			if mode == "validator_unavailable" {
				f.ops.validate = func(platform.RenderedArtifact, platform.Scope) platform.Verdict {
					return platform.Verdict{State: platform.VerdictUnavailable}
				}
			}
			if mode == "revoked_after_reload" {
				f.ops.command = func(_ context.Context, args ...string) error {
					*calls = append(*calls, args[0])
					next := r
					next.Phase, next.Fence, next.FailureClass = coreactivation.Restoring, coreactivation.Revoked, "activation"
					writeActivationRecordFixture(t, f.Store, next)
					return nil
				}
			}
			if err := f.ActivateCandidate(ctx, r); err == nil {
				t.Fatal("unqualified activation succeeded")
			}
			want := 0
			if mode == "revoked_after_reload" {
				want = 1
			}
			if len(*calls) != want {
				t.Fatalf("late native effects: %v", *calls)
			}
		})
	}
}
