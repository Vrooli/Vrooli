package recovery

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fakeLive is a recovery.LiveControl over an in-memory instance table. A
// restart moves the instance to the working tree only when the engagement no
// longer routes it to the frozen copy, as the lifecycle resolver does.
type fakeLive struct {
	svc         Service
	instances   map[string]string // key -> working dir
	failRestart bool
	failStop    bool
	restarts    []string
	stops       []string
}

func (f *fakeLive) Instances(string) ([]ServingInstance, error) {
	out := []ServingInstance{}
	for key, dir := range f.instances {
		out = append(out, ServingInstance{Key: key, WorkingDir: dir})
	}
	return out, nil
}

func (f *fakeLive) Restart(key string) error {
	f.restarts = append(f.restarts, key)
	if f.failRestart {
		return errors.New("build ui: exit status 1")
	}
	if _, err := f.svc.Store.ReadManifest("demo", "abc"); err == nil {
		return nil // still engaged: the restart resolves the frozen copy again
	}
	f.instances[key] = f.svc.scenarioDir("demo")
	return nil
}

func (f *fakeLive) Stop(key string) error {
	f.stops = append(f.stops, key)
	if f.failStop {
		return errors.New("stop failed")
	}
	delete(f.instances, key)
	return nil
}

// engagedLive opens a shadow engagement whose live instance runs from the
// engagement's serving tree.
func engagedLive(t *testing.T) (Service, *fakeLive) {
	t.Helper()
	svc, _ := newTestService(t, time.Now())
	if _, err := svc.WriteEngagement(WriteRequest{Scenario: "demo", Slug: "abc", Mode: "shadow"}); err != nil {
		t.Fatal(err)
	}
	serving := filepath.Join(svc.Store.ServingBasePath("demo", "abc"), "g-1", "scenarios", "demo", "ui")
	if err := os.MkdirAll(serving, 0o755); err != nil {
		t.Fatal(err)
	}
	live := &fakeLive{svc: svc, instances: map[string]string{
		"demo":        serving,
		"demo@shadow": svc.scenarioDir("demo"),
	}}
	svc.Live = live
	live.svc = svc
	return svc, live
}

// O16: clean (gc) must not delete the tree live runs from.
func TestCleanRefusesWhileLiveRunsFromServingTree(t *testing.T) {
	svc, live := engagedLive(t)
	_, err := svc.Clean(CleanRequest{Scenario: "demo", Slug: "abc"})
	if !errors.Is(err, ErrServingFromEngagement) {
		t.Fatalf("clean with live on the serving tree: %v", err)
	}
	if _, err := os.Stat(svc.Store.ManifestPath("demo", "abc")); err != nil {
		t.Fatalf("refused clean removed the engagement: %v", err)
	}
	if len(live.restarts)+len(live.stops) != 0 {
		t.Fatalf("refused clean touched live: restarts=%v stops=%v", live.restarts, live.stops)
	}
}

// abandon: restart live from the (restored) working tree, then remove the
// engagement, so live never runs from a deleted tree.
func TestCleanRestartLiveMovesLiveOffTheEngagement(t *testing.T) {
	svc, live := engagedLive(t)
	out, err := svc.Clean(CleanRequest{Scenario: "demo", Slug: "abc", RestartLive: true})
	if err != nil {
		t.Fatalf("clean --restart-live: %v", err)
	}
	if len(out.Restarted) != 1 || out.Restarted[0] != "demo" || live.instances["demo"] != svc.scenarioDir("demo") {
		t.Fatalf("live not restarted onto the working tree: %+v %v", out, live.instances)
	}
	if _, err := os.Stat(svc.Store.EngagementDir("demo", "abc")); !os.IsNotExist(err) {
		t.Fatalf("engagement not removed (err=%v)", err)
	}
}

// A failed rebuild keeps the old instance serving, so the engagement (and
// the tree it serves from) must stay.
func TestCleanRestartLiveFailureKeepsEngagement(t *testing.T) {
	svc, live := engagedLive(t)
	live.failRestart = true
	if _, err := svc.Clean(CleanRequest{Scenario: "demo", Slug: "abc", RestartLive: true}); err == nil {
		t.Fatal("clean must fail when live cannot be restarted")
	}
	if _, err := svc.Store.ReadManifest("demo", "abc"); err != nil {
		t.Fatalf("engagement manifest not reinstated: %v", err)
	}
	if len(live.stops) != 0 {
		t.Fatalf("restart-live must never stop live: %v", live.stops)
	}
}

// --force: restart live, and stop it if the restart fails; then remove.
func TestCleanForceStopsLiveWhenRestartFails(t *testing.T) {
	svc, live := engagedLive(t)
	live.failRestart = true
	out, err := svc.Clean(CleanRequest{Scenario: "demo", Slug: "abc", Force: true})
	if err != nil {
		t.Fatalf("clean --force: %v", err)
	}
	if len(out.Stopped) != 1 || out.Stopped[0] != "demo" {
		t.Fatalf("live not stopped after failed restart: %+v", out)
	}
	if _, err := os.Stat(svc.Store.EngagementDir("demo", "abc")); !os.IsNotExist(err) {
		t.Fatalf("engagement not removed (err=%v)", err)
	}
}

// When no instance runs from the engagement (live already on the working
// tree after promote), clean proceeds without touching instances.
func TestCleanWithoutServingInstancesNeedsNoFlags(t *testing.T) {
	svc, live := engagedLive(t)
	live.instances["demo"] = svc.scenarioDir("demo")
	if _, err := svc.Clean(CleanRequest{Scenario: "demo", Slug: "abc"}); err != nil {
		t.Fatalf("clean: %v", err)
	}
	if len(live.restarts)+len(live.stops) != 0 {
		t.Fatalf("clean touched instances: %v %v", live.restarts, live.stops)
	}
}
