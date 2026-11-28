package generation

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func snapshotAttemptFixture(t *testing.T) (*DefaultService, string, string) {
	t.Helper()
	root := writeGeneratorStampFixture(t)
	if err := os.WriteFile(filepath.Join(root, "build-tools", "dist", "template-generator.js"), []byte("// fake generator input\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	events, bin := t.TempDir(), t.TempDir()
	writeNodeFixture(t, bin, `set -eu
name=${2##*/}
snapshot=${1%/build-tools/dist/template-generator.js}
touch "$snapshot/owned-$name"
printf '%s' "$snapshot" > "$SNAPSHOT_TEST_EVENTS/$name.path.tmp"
mv "$SNAPSHOT_TEST_EVENTS/$name.path.tmp" "$SNAPSHOT_TEST_EVENTS/$name.path"
attempt=0
while [ ! -f "$SNAPSHOT_TEST_EVENTS/release-$name" ]; do
  attempt=$((attempt+1))
  [ "$attempt" -lt 500 ] || exit 42
  sleep 0.01
done
test -f "$snapshot/owned-$name" || exit 43
cat "$2"
printf 'consumed own snapshot %s\n' "$snapshot"
case "$name" in failed) exit 7;; interrupted) exit 130;; esac`)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SNAPSHOT_TEST_EVENTS", events)
	builds := newMockBuildStore()
	return &DefaultService{templateDir: root, builds: builds}, events, t.TempDir()
}

func waitSnapshotPath(t *testing.T, events, name string) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(filepath.Join(events, name+".path")); err == nil && len(data) > 0 {
			return string(data)
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("fake generator did not reach barrier: %s", name)
	return ""
}

func startSnapshotAttempt(t *testing.T, service *DefaultService, events, parent, name string) <-chan struct{} {
	t.Helper()
	configPath := filepath.Join(parent, name)
	config, err := json.Marshal(map[string]string{"attempt": name})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, config, 0o600); err != nil {
		t.Fatal(err)
	}
	service.builds.Create(name)
	done := make(chan struct{})
	go func() {
		defer close(done)
		service.runTemplateGenerator(name, &DesktopConfig{OutputPath: filepath.Join(parent, "output-"+name)}, configPath)
	}()
	t.Cleanup(func() {
		_ = os.WriteFile(filepath.Join(events, "release-"+name), nil, 0o600)
		select {
		case <-done:
		case <-time.After(6 * time.Second):
			t.Errorf("fake generator did not terminate: %s", name)
		}
	})
	return done
}

func finishSnapshotAttempt(t *testing.T, events, name string, done <-chan struct{}) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(events, "release-"+name), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("fake generator did not complete: %s", name)
	}
}

func TestConcurrentTemplateSnapshotsArePrivateAndRetired(t *testing.T) {
	service, events, parent := snapshotAttemptFixture(t)
	first := startSnapshotAttempt(t, service, events, parent, "first")
	firstPath := waitSnapshotPath(t, events, "first")
	second := startSnapshotAttempt(t, service, events, parent, "second")
	secondPath := waitSnapshotPath(t, events, "second")
	if firstPath == secondPath {
		t.Fatal("concurrent generators share a mutable snapshot")
	}
	for _, name := range []string{"first", "second"} {
		path := firstPath
		if name == "second" {
			path = secondPath
		}
		if _, err := os.Stat(filepath.Join(path, "owned-"+name)); err != nil {
			t.Fatalf("other attempt removed active input: %v", err)
		}
	}
	finishSnapshotAttempt(t, events, "second", second)
	if _, err := os.Stat(filepath.Join(firstPath, "owned-first")); err != nil {
		t.Fatalf("other attempt cleanup removed live input: %v", err)
	}
	finishSnapshotAttempt(t, events, "first", first)
	for _, name := range []string{"first", "second"} {
		status, _ := service.builds.Get(name)
		if status.Status != "ready" || !strings.Contains(strings.Join(status.BuildLog, "\n"), "consumed own snapshot") || status.Artifacts["template_snapshot_hash"] == "" {
			t.Fatalf("generation failed or lost digest: %#v", status)
		}
		if !strings.Contains(strings.Join(status.BuildLog, "\n"), `"attempt":"`+name+`"`) {
			t.Fatalf("generator consumed the wrong config: %#v", status.BuildLog)
		}
	}
	for _, path := range []string{firstPath, secondPath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("completed temporary snapshot retained: %s %v", path, err)
		}
	}
}

func TestTemplateSnapshotRetiredAfterGeneratorFailureOrInterruptExit(t *testing.T) {
	for _, name := range []string{"failed", "interrupted"} {
		t.Run(name, func(t *testing.T) {
			service, events, parent := snapshotAttemptFixture(t)
			done := startSnapshotAttempt(t, service, events, parent, name)
			path := waitSnapshotPath(t, events, name)
			finishSnapshotAttempt(t, events, name, done)
			status, _ := service.builds.Get(name)
			if status.Status != "failed" || status.CompletedAt == nil || status.Artifacts["template_snapshot_hash"] == "" {
				t.Fatalf("failed generator reported success: %#v", status)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("failed attempt retained snapshot: %v", err)
			}
		})
	}
}

// Observe every state published to the consumer, not only the final stored state.
type snapshotStateObserver struct {
	BuildStore
	states []string
}

func (s *snapshotStateObserver) Update(id string, update func(*BuildStatus)) {
	s.BuildStore.Update(id, func(status *BuildStatus) {
		update(status)
		s.states = append(s.states, status.Status)
	})
}

func TestTemplateSnapshotCleanupFailureNeverPublishesReady(t *testing.T) {
	for _, name := range []string{"first", "failed"} {
		t.Run(name, func(t *testing.T) {
			service, events, parent := snapshotAttemptFixture(t)
			observer := &snapshotStateObserver{BuildStore: service.builds}
			service.builds = observer
			service.removeTemplateSnapshot = func(path string) error {
				if err := os.RemoveAll(path); err != nil {
					return err
				}
				return errors.New("injected snapshot cleanup failure")
			}
			done := startSnapshotAttempt(t, service, events, parent, name)
			waitSnapshotPath(t, events, name)
			finishSnapshotAttempt(t, events, name, done)
			status, _ := service.builds.Get(name)
			for _, state := range observer.states {
				if state == "ready" {
					t.Fatalf("published transient ready before failed cleanup: %v", observer.states)
				}
			}
			if status.Status != "failed" || status.CompletedAt == nil || status.Artifacts["output_path"] != "" {
				t.Fatalf("cleanup failure advertised usable output: %#v", status)
			}
			log := strings.Join(status.ErrorLog, "\n")
			if !strings.Contains(log, "injected snapshot cleanup failure") {
				t.Fatalf("lost cleanup diagnostic: %s", log)
			}
			if name == "failed" && !strings.Contains(log, "exit status 7") {
				t.Fatalf("lost original generator error: %s", log)
			}
		})
	}
}
