package lifecycle

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/lifecycle/retainedruntime"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
)

// Given a qualified native API plus a real pinned Node process, when a newer
// retained UI exits after replacement begins, then the lifecycle owner can
// explicitly restore both prior artifacts without moving canonical state.
// The explicit gate admits the host Node executable, not an installed scenario.
func TestRetainedMixedRuntimeRecoversAfterNodeReplacementFailure(t *testing.T) {
	if os.Getenv("VROOLI_RETAINED_NODE_LIVE") != "1" {
		t.Skip("set VROOLI_RETAINED_NODE_LIVE=1 to qualify with a real host Node interpreter")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	node, err = filepath.EvalSymlinks(node)
	if err != nil {
		t.Fatal(err)
	}
	if err = validateRetainedNativeExecutable(node); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(node)+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("NODE_OPTIONS", "")
	t.Setenv("NODE_PATH", "")
	root, home := t.TempDir(), t.TempDir()
	t.Setenv("VROOLI_HOME", home)
	const name = "retained-mixed-fixture"
	manifest := lifecycleFixtureManifest(name)
	apiPort := retainedMixedPort(t, root, 16000, 4000)
	uiPort := retainedMixedPort(t, root, 22000, 3000)
	manifest.Ports["api"] = scenario.Port{EnvVar: "API_PORT", Port: &apiPort}
	manifest.Ports["ui"] = scenario.Port{EnvVar: "UI_PORT", Port: &uiPort}
	manifest.Components["ui"] = scenario.Component{
		Role: "ui", Build: scenario.ComponentBuild{Kind: "node_bundle", Dir: "ui", Output: "ui/server.cjs"},
		Run: scenario.ComponentRun{Argv: []string{"node", "server.cjs"}, CWD: "ui", Port: "ui"},
	}
	manifest.Lifecycle.Health.Checks = []scenario.HealthCheck{
		{Name: "api", Type: "http", Target: fmt.Sprintf("http://127.0.0.1:%d/health", apiPort), Critical: true, Timeout: 1000},
		{Name: "ui", Type: "http", Target: fmt.Sprintf("http://127.0.0.1:%d/health", uiPort), Critical: true, Timeout: 1000},
	}
	writeLifecycleFixtureManifest(t, root, manifest)
	source := filepath.Join(root, "scenarios", name)
	apiDir := filepath.Join(source, "api")
	// Build a temporary test artifact only. Neither a scenario install nor its
	// build/setup hooks are invoked. Every process launch below uses the owner.
	cmd := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-buildvcs=false", "-o", "mock-api", ".")
	cmd.Dir = apiDir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=", "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile mixed fixture: %v: %s", err, out)
	}
	retainedIntegrationWrite(t, filepath.Join(source, ".vrooli/runtime-release.json"), []byte(`{"version":1,"state_contract":"mixed-fixture-state-v1","paths":["api/mock-api","ui/server.cjs","ui/package.json"],"interpreters":["node"]}`), 0600)
	retainedIntegrationWrite(t, filepath.Join(source, "ui/package.json"), []byte(`{"name":"retained-mixed-fixture-ui","private":true,"dependencies":{}}`), 0600)
	const goodUI = `const http=require('node:http'); const fs=require('node:fs'); const path=require('node:path');
 const server=http.createServer((req,res)=>{const state=fs.readFileSync(path.join(process.env.SCENARIO_DATA_DIR,'sentinel'),'utf8');
 res.setHeader('content-type','application/json'); res.end(JSON.stringify({status:'healthy',readiness:true,service:'mixed-fixture-ui',build_identity:process.env.VROOLI_BUILD_IDENTITY,version:'ui-v1',state,scenarioPath:process.env.SCENARIO_PATH,cwd:process.cwd(),interpreter:process.execPath}));});
 server.listen(Number(process.env.UI_PORT),'127.0.0.1');`
	uiPath := filepath.Join(source, "ui/server.cjs")
	retainedIntegrationWrite(t, uiPath, []byte(goodUI), 0600)
	sentinel := filepath.Join(source, "data/sentinel")
	retainedIntegrationWrite(t, sentinel, []byte("keep canonical state"), 0600)
	r := newLifecycleRunnerForTest(t, root, home, noSupervisorLaunch)
	cleanupRunner(t, r, name, StopOptions{})
	raw, err := os.ReadFile(filepath.Join(source, ".vrooli/service.json"))
	if err != nil {
		t.Fatal(err)
	}
	capture := func(label string) retainedruntime.Release {
		t.Helper()
		release, err := r.runtimeReleaseStore().Capture(context.Background(), retainedruntime.CaptureSpec{Scenario: name, Variant: "live", SourceDir: source, BuildIdentity: label, SourceRuntimeIdentity: "isolated-fixture-not-production", ServiceManifest: raw, ForbiddenPaths: []string{"data", "logs"}})
		if err != nil {
			t.Fatal(err)
		}
		return release
	}
	old := capture("fixture-original")
	// Capturing fixture bytes is not an invented serving receipt. Only this real
	// owner start and the subsequent HTTP/process checks qualify the fixture.
	if _, err = r.Start(name, StartOptions{RetainedReleaseID: old.ID, ExpectedRetainedReleaseID: "source"}); err != nil {
		t.Fatalf("initial mixed retained start: %v", err)
	}
	assertServing := func(want retainedruntime.Release) []int {
		t.Helper()
		records, err := process.ReadScenarioRecords(home, name)
		if err != nil {
			t.Fatal(err)
		}
		if len(records) != 2 {
			t.Fatalf("expected native and Node records: %+v", records)
		}
		var pids []int
		for _, record := range records {
			if !process.IsPIDRunning(record.PID) || !strings.HasPrefix(filepath.Clean(record.WorkingDir), want.RuntimeDir+string(filepath.Separator)) {
				t.Fatalf("not serving retained component: %+v", record)
			}
			pids = append(pids, record.PID)
		}
		instance := runningInstanceForTest(t, home, name)
		if instance.BuildIdentity != "sha256:"+want.ID {
			t.Fatalf("wrong release identity: %+v", instance)
		}
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/health", uiPort))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var body map[string]any
		if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK || body["version"] != "ui-v1" || body["state"] != "keep canonical state" || body["scenarioPath"] != source || body["cwd"] != filepath.Join(want.RuntimeDir, "ui") || body["build_identity"] != "sha256:"+want.ID {
			t.Fatalf("mixed runtime identity/state mismatch: %#v", body)
		}
		if len(want.Interpreters) != 1 || body["interpreter"] != want.Interpreters[0].Path {
			t.Fatalf("Node interpreter not pinned: %#v", body)
		}
		if b, err := os.ReadFile(sentinel); err != nil || string(b) != "keep canonical state" {
			t.Fatalf("canonical state changed: %q %v", b, err)
		}
		if _, err = os.Stat(filepath.Join(want.RuntimeDir, "data")); !os.IsNotExist(err) {
			t.Fatalf("state leaked into runtime: %v", err)
		}
		return pids
	}
	oldPIDs := assertServing(old)
	retainedIntegrationWrite(t, uiPath, []byte("process.exit(27);\n"), 0600)
	broken := capture("fixture-replacement-exits")
	if _, err = r.Restart(name, StartOptions{RetainedReleaseID: broken.ID, ExpectedRetainedReleaseID: old.ID}); err == nil {
		t.Fatal("exiting Node replacement unexpectedly succeeded")
	}
	for _, pid := range oldPIDs {
		if process.IsPIDRunning(pid) {
			t.Fatalf("failed replacement did not pass stop boundary for prior PID %d", pid)
		}
	}
	// A fresh controller must discover durable selected intent and restore the
	// previous exact native/Node closure, not quietly rebuild mutable source.
	next := newLifecycleRunnerForTest(t, root, home, noSupervisorLaunch)
	if selected, err := next.SelectedRuntimeReleaseID(context.Background(), name); err != nil || selected != broken.ID {
		t.Fatalf("failed replacement selection lost: %q %v", selected, err)
	}
	if _, err = next.Restart(name, StartOptions{RetainedReleaseID: old.ID, ExpectedRetainedReleaseID: broken.ID}); err != nil {
		t.Fatalf("restore mixed prior runtime: %v", err)
	}
	restored := assertServing(old)
	if b, err := os.ReadFile(uiPath); err != nil || string(b) != "process.exit(27);\n" {
		t.Fatalf("recovery rewrote source: %q %v", b, err)
	}
	if _, err = next.Restart(name, StartOptions{RetainedReleaseID: broken.ID, ExpectedRetainedReleaseID: broken.ID}); err == nil {
		t.Fatal("stale selection accepted")
	}
	for _, pid := range restored {
		if !process.IsPIDRunning(pid) {
			t.Fatalf("stale selection stopped healthy fixture PID %d", pid)
		}
	}
	t.Logf("Qualified isolated native+Node recovery: retained release %s; interpreter %s", old.ID, old.Interpreters[0].SHA256)
}

func retainedMixedPort(t *testing.T, seed string, base, count int) int {
	t.Helper()
	offset := int(crc32.ChecksumIEEE([]byte(seed))) % count
	for n := 0; n < count; n++ {
		port := base + (offset+n)%count
		listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
		if err != nil {
			continue
		}
		if err = listener.Close(); err != nil {
			t.Fatal(err)
		}
		return port
	}
	t.Fatal("no free isolated fixture port")
	return 0
}
