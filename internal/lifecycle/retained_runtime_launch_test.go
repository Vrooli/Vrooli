package lifecycle

import (
	"context"
	"hash/crc32"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/process"
)

// This exercises the production lifecycle start/restart and release owner with
// a compiled temporary fixture. It never addresses an installed scenario.
func TestRetainedRuntimeLaunchSurvivesPoisonedSource(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	t.Setenv("VROOLI_HOME", home)
	manifest := lifecycleFixtureManifest("alpha")
	// Other package-test processes have their own allocator map. Spread this
	// fixture across the canonical API band instead of racing for port 15000.
	seed := int(crc32.ChecksumIEEE([]byte(root)) % 4000)
	allocated := 0
	for offset := 0; offset < 4000; offset++ {
		candidate := 16000 + (seed+offset)%4000
		listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(candidate))
		if err != nil {
			continue
		}
		if err = listener.Close(); err != nil {
			t.Fatal(err)
		}
		allocated = candidate
		break
	}
	if allocated == 0 {
		t.Fatal("no available fixture API port")
	}
	port := manifest.Ports["api"]
	port.Port = &allocated
	port.Range = ""
	manifest.Ports["api"] = port
	for i := range manifest.Lifecycle.Health.Checks {
		manifest.Lifecycle.Health.Checks[i].Target = strings.ReplaceAll(manifest.Lifecycle.Health.Checks[i].Target, "${API_PORT}", strconv.Itoa(allocated))
	}
	writeLifecycleFixtureManifest(t, root, manifest)
	source := filepath.Join(root, "scenarios", "alpha")
	declaration := []byte(`{"version":1,"state_contract":"fixture-state-v1","paths":["api/mock-api"],"interpreters":[]}`)
	retainedIntegrationWrite(t, filepath.Join(source, ".vrooli/runtime-release.json"), declaration, 0600)
	state := filepath.Join(source, "data", "canonical-state")
	retainedIntegrationWrite(t, state, []byte("must survive every restart"), 0600)
	r := newLifecycleRunnerForTest(t, root, home, noSupervisorLaunch)
	cleanupRunner(t, r, "alpha", StopOptions{})
	if _, err := r.Start("alpha", StartOptions{}); err != nil {
		t.Fatalf("initial source start: %v", err)
	}
	release, err := r.CaptureRuntimeRelease(context.Background(), "alpha", "")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	if release.SourceRuntimeIdentity == "" || release.BuildIdentity == "" {
		t.Fatalf("missing provenance: %#v", release)
	}
	main := filepath.Join(source, "api/main.go")
	original, err := os.ReadFile(main)
	if err != nil {
		t.Fatal(err)
	}
	poisoned := []byte("this is deliberately invalid Go; any source rebuild must fail\n")
	retainedIntegrationWrite(t, main, poisoned, 0600)

	assertServingFrom := func(want string) int {
		t.Helper()
		records, err := process.ReadScenarioRecords(home, "alpha")
		if err != nil {
			t.Fatal(err)
		}
		if len(records) != 1 {
			t.Fatalf("process records=%#v", records)
		}
		rec := records[0]
		if want == release.RuntimeDir {
			instance := runningInstanceForTest(t, home, "alpha")
			if instance.BuildIdentity != "sha256:"+release.ID {
				t.Fatalf("retained launch identity=%q, want release digest", instance.BuildIdentity)
			}
		}
		if filepath.Clean(rec.WorkingDir) != filepath.Join(want, "api") || !process.IsPIDRunning(rec.PID) {
			t.Fatalf("not serving expected tree %s: %#v", want, rec)
		}
		if got, err := os.ReadFile(state); err != nil || string(got) != "must survive every restart" {
			t.Fatalf("canonical state changed: %q %v", got, err)
		}
		return rec.PID
	}
	if _, err = r.Restart("alpha", StartOptions{RetainedReleaseID: release.ID, ExpectedRetainedReleaseID: "source"}); err != nil {
		t.Fatalf("select retained restart with poisoned source: %v", err)
	}
	assertServingFrom(release.RuntimeDir)
	if _, err = os.Stat(filepath.Join(release.RuntimeDir, "data")); !os.IsNotExist(err) {
		t.Fatalf("data was copied into release: %v", err)
	}

	next := newLifecycleRunnerForTest(t, root, home, noSupervisorLaunch)
	selected, err := next.SelectedRuntimeReleaseID(context.Background(), "alpha")
	if err != nil || selected != release.ID {
		t.Fatalf("durable selection=%q err=%v", selected, err)
	}
	if _, err = next.Restart("alpha", StartOptions{}); err != nil {
		t.Fatalf("ordinary retained restart: %v", err)
	}
	assertServingFrom(release.RuntimeDir)
	if err = next.Stop("alpha", StopOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err = next.Start("alpha", StartOptions{}); err != nil {
		t.Fatalf("ordinary retained start: %v", err)
	}
	retainedPID := assertServingFrom(release.RuntimeDir)
	if _, err = next.Restart("alpha", StartOptions{ForceSetup: true}); err == nil {
		t.Fatal("force unexpectedly rebuilt retained runtime")
	}
	if !process.IsPIDRunning(retainedPID) {
		t.Fatal("force rejection stopped retained process")
	}
	if got, err := os.ReadFile(main); err != nil || string(got) != string(poisoned) {
		t.Fatalf("source poison changed: %q %v", got, err)
	}

	// Returning to source is explicit. The source rebuild is now allowed to run.
	retainedIntegrationWrite(t, main, original, 0600)
	if _, err = next.Restart("alpha", StartOptions{RetainedReleaseID: "source", ExpectedRetainedReleaseID: release.ID}); err != nil {
		t.Fatalf("return to source: %v", err)
	}
	sourcePID := assertServingFrom(source)
	selected, err = next.SelectedRuntimeReleaseID(context.Background(), "alpha")
	if err != nil || selected != "" {
		t.Fatalf("source selection=%q err=%v", selected, err)
	}

	// A replacement can build successfully, stop the old process, then fail to
	// serve. Recovery must use captured bytes rather than rebuilding that candidate.
	retainedIntegrationWrite(t, main, []byte("package main\nimport \"os\"\nfunc main() { os.Exit(27) }\n"), 0600)
	if _, err = next.Restart("alpha", StartOptions{ForceSetup: true}); err == nil {
		t.Fatal("exiting source replacement unexpectedly served")
	}
	if process.IsPIDRunning(sourcePID) {
		t.Fatal("candidate failure did not pass the old-process stop boundary")
	}
	failedRecords, e := process.ReadScenarioRecords(home, "alpha")
	if e != nil {
		t.Fatal(e)
	}
	for _, record := range failedRecords {
		if process.IsPIDRunning(record.PID) {
			t.Fatalf("failed candidate left a serving process: %#v", record)
		}
	}
	if _, err = next.Restart("alpha", StartOptions{RetainedReleaseID: release.ID, ExpectedRetainedReleaseID: "source"}); err != nil {
		t.Fatalf("retained recovery after replacement failure: %v", err)
	}
	assertServingFrom(release.RuntimeDir)
	retainedIntegrationWrite(t, main, original, 0600)
	if _, err = next.Restart("alpha", StartOptions{RetainedReleaseID: "source", ExpectedRetainedReleaseID: release.ID}); err != nil {
		t.Fatalf("restore source after demonstrated recovery: %v", err)
	}
	sourcePID = assertServingFrom(source)

	// A rejected comparison and a corrupted artifact must fail before stop.
	if _, err = next.Restart("alpha", StartOptions{RetainedReleaseID: release.ID, ExpectedRetainedReleaseID: release.ID}); err == nil {
		t.Fatal("stale expected release accepted")
	}
	if !process.IsPIDRunning(sourcePID) {
		t.Fatal("selection conflict stopped source process")
	}
	artifact := filepath.Join(release.RuntimeDir, "api/mock-api")
	if err = os.Chmod(artifact, 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(artifact, []byte("tampered"), 0700); err != nil {
		t.Fatal(err)
	}
	_, err = next.Restart("alpha", StartOptions{RetainedReleaseID: release.ID, ExpectedRetainedReleaseID: "source"})
	if err == nil {
		t.Fatal("tampered retained artifact accepted")
	}
	if !process.IsPIDRunning(sourcePID) {
		t.Fatal("tamper rejection stopped source process")
	}
	if selected, e := next.SelectedRuntimeReleaseID(context.Background(), "alpha"); e != nil || selected != "" {
		t.Fatalf("tamper changed selection: %q %v", selected, e)
	}
	assertServingFrom(source)
}
