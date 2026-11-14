package lifecycle

import (
	"context"
	"errors"
	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

type componentExitHandle struct {
	alive          bool
	signals, reads int
	err            error
	closed         bool
}

func TestEmergencyCannotBypassManagedExitOrAdmittedWork(t *testing.T) {
	for _, name := range []string{"managed-component", "managed-reported-drained", "admitted-work"} {
		t.Run(name, func(t *testing.T) {
			r, s, events := restartFenceTestRunner(t, "refuse")
			if name == "managed-component" || name == "managed-reported-drained" {
				s.components = []*commonv1.LifecycleManagedComponent{{Name: "desktop-helper", Generation: "g"}}
				s.prepareDrained = name == "managed-reported-drained"
			} else {
				s.admitting = 1
			}
			fence, err := r.prepareProviderFence(context.Background(), scenario.Scenario{Slug: "alpha", Manifest: scenario.ServiceManifest{Ports: map[string]scenario.Port{"api": {EnvVar: "API_PORT"}}}}, true, "operator emergency", "op", nil, nil)
			if err == nil || fence != nil {
				t.Fatalf("emergency returned active fence without safety proof: %v", err)
			}
			got := snapshotRestartFenceEvents(s, events)
			wantWait := "drain"
			if name != "admitted-work" {
				wantWait = "quiesce"
			}
			if len(got) != 2 || got[0] != "prepare" || got[1] != wantWait {
				t.Fatalf("emergency bypassed owner wait: %v, want [prepare %s]", got, wantWait)
			}
		})
	}
}

func TestManagedComponentCompletionReusesOnlyExactDrainedRetainedFence(t *testing.T) {
	for _, emergency := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary", true: "emergency"}[emergency], func(t *testing.T) {
			r, s, events := restartFenceTestRunner(t, "refuse")
			s.components = []*commonv1.LifecycleManagedComponent{{Name: "desktop-helper", Generation: "g"}}
			s.closed = true
			s.opID = "retained-operation"
			s.statusDrained = true
			s.prepareDrained = true
			fence, err := r.prepareProviderFence(context.Background(), scenario.Scenario{Slug: "alpha", Manifest: scenario.ServiceManifest{Ports: map[string]scenario.Port{"api": {EnvVar: "API_PORT"}}}}, emergency, "operator recovery", "retained-operation", nil, nil)
			if err != nil || fence == nil || !fence.active {
				t.Fatalf("exact completed fence did not recover: fence=%v err=%v", fence, err)
			}
			got := snapshotRestartFenceEvents(s, events)
			if len(got) != 1 || got[0] != "prepare" {
				t.Fatalf("completed exact generation was re-signaled: %v", got)
			}
		})
	}
}

func TestVerifiedComponentExitCannotBecomeLaterNumericTeardownTarget(t *testing.T) {
	ctx := context.Background()
	root, home := t.TempDir(), t.TempDir()
	writeLifecycleFixture(t, root, "alpha")
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatal(err)
	}
	instance, err := store.CreateLease(ctx, scenarioruntime.Instance{Scenario: "alpha", Variant: "live", Status: scenarioruntime.StatusStarting, HostBootID: "boot"}, scenarioruntime.DefaultHeartbeatTTL)
	if err != nil {
		t.Fatal(err)
	}
	record := process.Record{PID: 424242, PGID: 424242, Scenario: "alpha", Step: "start-desktop-helper", ProcessID: "vrooli.develop.alpha.start-desktop-helper", Phase: "develop", Status: "running", Generation: "g", Incarnation: "boot:123", RuntimeInstanceID: instance.InstanceID, StartedAt: time.Now().UTC()}
	_, err = store.AddProcessRef(ctx, scenarioruntime.ProcessRef{RefID: process.LaunchRefID(record.Generation), InstanceID: instance.InstanceID, PID: &record.PID, PGID: &record.PGID, Step: record.Step, ProcessID: record.ProcessID, Status: "running", StartedAt: record.StartedAt, HostBootID: "boot"})
	if err != nil {
		t.Fatal(err)
	}
	_ = store.Close()
	if err = process.WriteScenarioRecord(home, "alpha", record.Step, record); err != nil {
		t.Fatal(err)
	}
	signals := 0
	r := newLifecycleRunnerForTest(t, root, home, func(d *lifecycleDeps) {
		d.hostSession = func(context.Context, string) (hostsession.Snapshot, error) {
			return hostsession.Snapshot{BootID: "boot"}, nil
		}
		d.isPIDRunning = func(int) bool { return true }
		d.signalProcessGroup = func(int, bool) error { signals++; return nil }
		d.signalPID = func(int, bool) error { signals++; return nil }
		d.readProcessEnv = func(int) (map[string]string, error) {
			return map[string]string{runtimeRegistryInstanceEnv: instance.InstanceID, "VROOLI_SCENARIO": "alpha", "VROOLI_VARIANT": "live", "VROOLI_COMPONENT_GENERATION": "replacement-generation"}, nil
		}
	})
	if err = r.retireComponentGeneration(ctx, scenario.Scenario{Slug: "alpha"}, record); err != nil {
		t.Fatal(err)
	}
	inspection, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatal(err)
	}
	refs, err := inspection.ListProcessRefs(ctx, instance.InstanceID)
	_ = inspection.Close()
	if err != nil || len(refs) != 1 || refs[0].Status != "exited" || refs[0].RefID != process.LaunchRefID(record.Generation) {
		t.Fatalf("exact verified runtime ref was not retired: refs=%+v err=%v", refs, err)
	}
	if err = r.cleanupScenarioRuntimeWithRegistryContext(ctx, "alpha", "live", "", false, true, true); err != nil {
		t.Fatal(err)
	}
	if signals != 0 {
		t.Fatalf("verified exit caused %d subsequent numeric signals", signals)
	}
}

func TestVerifiedExitPublishesTombstoneBeforeRegistryFailure(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	writeLifecycleFixture(t, root, "alpha")
	record := process.Record{PID: 424242, PGID: 424242, Scenario: "alpha", Step: "start-desktop-helper", Status: "running", Generation: "g", Incarnation: "boot:123", RuntimeInstanceID: "runtime", StartedAt: time.Now().UTC()}
	if err := process.WriteScenarioRecord(home, "alpha", record.Step, record); err != nil {
		t.Fatal(err)
	}
	called := false
	r := newLifecycleRunnerForTest(t, root, home, func(d *lifecycleDeps) {
		d.runtimeRegistry = func(context.Context, string) (scenarioRuntimeStore, error) {
			called = true
			records, err := process.ReadScenarioRecords(home, "alpha")
			if err != nil || len(records) != 1 || records[0].Status != "exited_verified" {
				t.Fatalf("registry retirement preceded atomic filesystem suppression: records=%v err=%v", records, err)
			}
			return nil, errors.New("registry unavailable")
		}
	})
	if err := r.retireComponentGeneration(context.Background(), scenario.Scenario{Slug: "alpha"}, record); err == nil || !called {
		t.Fatalf("registry failure became completed retirement: %v", err)
	}
	records, err := process.ReadScenarioRecords(home, "alpha")
	if err != nil || len(records) != 1 || records[0].Status != "exited_verified" {
		t.Fatalf("failed retirement lost safe suppression: records=%v err=%v", records, err)
	}
}

func (h *componentExitHandle) Alive() (bool, error) { h.reads++; return h.alive, h.err }
func (h *componentExitHandle) Signal(force bool) error {
	h.signals++
	if !h.alive {
		return errors.New("not alive")
	}
	return nil
}
func (h *componentExitHandle) Close() error { h.closed = true; return nil }

func componentExitFixture() (scenario.Scenario, process.Record, []*commonv1.LifecycleManagedComponent) {
	item := scenario.Scenario{Slug: "device-control", Manifest: scenario.ServiceManifest{Components: map[string]scenario.Component{"desktop-helper": {Role: "sidecar"}}}}
	record := process.Record{PID: 42, Scenario: item.Slug, Step: "start-desktop-helper", Phase: "develop", ProcessID: "vrooli.develop.device-control.start-desktop-helper", Status: "running", Generation: "generation", Incarnation: "boot:123"}
	return item, record, []*commonv1.LifecycleManagedComponent{{Name: "desktop-helper", Generation: "generation"}}
}

func TestManagedComponentStopRequiresExactCurrentLaunchAndWait(t *testing.T) {
	for _, name := range []string{"success", "running-timeout", "missing", "corrupt", "stale-generation", "record-changed-during-pin", "pin-unknown", "wait-unknown", "forged-exit-tombstone"} {
		t.Run(name, func(t *testing.T) {
			item, record, requests := componentExitFixture()
			h := &componentExitHandle{alive: true}
			reads := 0
			ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
			defer cancel()
			read := func() ([]process.Record, error) {
				reads++
				switch name {
				case "missing":
					return nil, nil
				case "corrupt":
					return nil, errors.New("corrupt record")
				case "stale-generation":
					record.Generation = "other"
				case "forged-exit-tombstone":
					record.Status = "exited_verified"
				case "record-changed-during-pin":
					if reads > 1 {
						record.Incarnation = "boot:999"
					}
				}
				return []process.Record{record}, nil
			}
			pin := func(process.Record) (process.Handle, error) {
				if name == "pin-unknown" {
					return nil, errors.New("unsupported identity")
				}
				if name == "success" {
					h.alive = false
				}
				if name == "wait-unknown" {
					h.err = errors.New("unknown")
				}
				return &componentSignalHandle{componentExitHandle: h}, nil
			}
			err := stopComponentGenerations(ctx, item, requests, read, pin)
			if name == "success" {
				if err != nil || h.signals != 1 || h.reads == 0 {
					t.Fatalf("completion without signal/wait: err=%v handle=%+v", err, h)
				}
			} else if err == nil {
				t.Fatal("unknown or running process reported exited")
			}
			if name != "success" && name != "running-timeout" && name != "wait-unknown" && h.signals != 0 {
				t.Fatal("signaled mismatched/unknown process")
			}
		})
	}
}

// The test seam permits a handle to become exited on the signal, while
// production pinning separately proves it alive before returning the handle.
type componentSignalHandle struct{ *componentExitHandle }

func (h *componentSignalHandle) Signal(bool) error { h.signals++; return nil }

type canceledComponentHandle struct {
	*componentExitHandle
	cancel context.CancelFunc
}

func (h *canceledComponentHandle) Signal(bool) error {
	h.signals++
	h.alive = false
	h.cancel()
	return nil
}

func TestManagedComponentCancellationAfterSignalCannotManufactureRetryProof(t *testing.T) {
	item, record, requests := componentExitFixture()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h := &canceledComponentHandle{componentExitHandle: &componentExitHandle{alive: true}, cancel: cancel}
	read := func() ([]process.Record, error) { return []process.Record{record}, nil }
	err := stopComponentGenerations(ctx, item, requests, read, func(process.Record) (process.Handle, error) { return h, nil })
	if !errors.Is(err, context.Canceled) || h.signals != 1 || !h.closed {
		t.Fatalf("cancellation became completion: err=%v handle=%+v", err, h)
	}
	// The command never completed its authoritative wait. A later missing PID
	// must fail pinning; its absence cannot manufacture the lost completion.
	err = stopComponentGenerations(context.Background(), item, requests, read, func(process.Record) (process.Handle, error) { return nil, errors.New("recorded process absent") })
	if err == nil {
		t.Fatal("retry manufactured exit proof from absence")
	}
}

func TestComponentQuiescenceKeepsFenceAndGeneration(t *testing.T) {
	p := &commonv1.LifecycleStanding{Provider: "device-control", ProviderInstanceId: "provider", OperationId: "op", FenceToken: "token", Revision: 3, AdmissionClosed: true, InventoryComplete: true, ManagedComponents: []*commonv1.LifecycleManagedComponent{{Name: "desktop-helper", Generation: "generation"}}}
	for _, name := range []string{"ready", "changed-fence", "changed-generation", "not-quiesced", "incomplete", "admitted-work", "invalid-admitting-count"} {
		t.Run(name, func(t *testing.T) {
			q := &commonv1.LifecycleStanding{Provider: p.Provider, ProviderInstanceId: p.ProviderInstanceId, OperationId: p.OperationId, FenceToken: p.FenceToken, Revision: p.Revision, AdmissionClosed: true, InventoryComplete: true, ComponentStopReady: true, ManagedComponents: []*commonv1.LifecycleManagedComponent{{Name: "desktop-helper", Generation: "generation"}}}
			switch name {
			case "changed-fence":
				q.Revision++
			case "changed-generation":
				q.ManagedComponents[0].Generation = "other"
			case "not-quiesced":
				q.ComponentStopReady = false
			case "incomplete":
				q.InventoryComplete = false
			case "admitted-work":
				q.Admitting = 1
			case "invalid-admitting-count":
				q.Admitting = -1
			}
			err := validateComponentStopStanding("device-control", p, q)
			if (err == nil) != (name == "ready") {
				t.Fatalf("standing acceptance: %v", err)
			}
		})
	}
}
