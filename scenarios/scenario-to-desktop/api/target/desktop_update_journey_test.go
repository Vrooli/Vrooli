//go:build linux

package target

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/pe"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

func j10Artifact(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Skip(name + " selects a generated desktop artifact")
	}
	if _, err := os.Stat(value); err != nil {
		t.Fatal(err)
	}
	return value
}

func startJ10LPBS(t *testing.T, next string) {
	t.Helper()
	home := t.TempDir()
	ready := filepath.Join(home, "ready")
	log, err := os.Create(filepath.Join(home, "lpbs.log"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	command := exec.CommandContext(ctx, "go", "test", "-tags=integration", ".", "-run", "^TestDesktopUpdateJourneyServer$", "-count=1", "-v", "-timeout=16m")
	command.Dir = "../../../landing-page-business-suite/api"
	command.Env = append(os.Environ(), "J10_NEXT_APPIMAGE="+next, "J10_SERVER_READY="+ready)
	command.Stdout = log
	command.Stderr = log
	if err := command.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	t.Cleanup(func() {
		request, _ := http.NewRequest(http.MethodPost, "http://127.0.0.1:39565/__journey/stop", nil)
		client := http.Client{Timeout: 5 * time.Second}
		if response, err := client.Do(request); err == nil {
			response.Body.Close()
		}
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			cancel()
		}
		cancel()
		log.Close()
		if dir := os.Getenv("J10_EVIDENCE"); dir != "" {
			data, _ := os.ReadFile(log.Name())
			_ = os.WriteFile(filepath.Join(dir, t.Name()+"-lpbs.log"), data, 0o600)
		}
	})
	deadline := time.NewTimer(3 * time.Minute)
	defer deadline.Stop()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case err := <-done:
			data, _ := os.ReadFile(log.Name())
			t.Fatalf("LPBS journey server exited before ready: %v\n%s", err, data)
		case <-deadline.C:
			t.Fatal("LPBS journey server readiness timeout")
		case <-tick.C:
			if _, err := os.Stat(ready); err == nil {
				return
			}
		}
	}
}

func copyJ10Artifact(t *testing.T, source, destination string) string {
	t.Helper()
	input, err := os.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o700)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.New()
	_, err = io.Copy(io.MultiWriter(output, hash), input)
	closeErr := output.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func startJ10Desktop(t *testing.T, artifact, home, digest string) *ElectronSession {
	t.Helper()
	return startJ10DesktopMode(t, artifact, home, digest, "1")
}

func startJ10DesktopMode(t *testing.T, artifact, home, digest, applyMode string) *ElectronSession {
	t.Helper()
	session, err := StartElectronSession(t.Context(), ElectronSessionOptions{
		ArtifactPath:   artifact,
		Launch:         ElectronLaunchOptions{ContextID: "agent-connect-j10", ScenarioName: "web-console", ArtifactDigest: "sha256:" + digest, TargetID: "local-linux-xvfb", JourneyID: "J10", ProfileID: "update-home", IsolationLeaseID: "j10-update-home", BundledRuntime: true, UserDataDir: filepath.Join(home, "electron")},
		Environment:    map[string]string{"HOME": home, "USERPROFILE": home, "PATH": "/usr/bin:/bin", "VROOLI_ROOT": "", "XDG_CONFIG_HOME": filepath.Join(home, ".config"), "XDG_DATA_HOME": filepath.Join(home, ".local/share"), "XDG_STATE_HOME": filepath.Join(home, ".local/state"), "XDG_CACHE_HOME": filepath.Join(home, ".cache"), "LIBGL_ALWAYS_SOFTWARE": "1", "VROOLI_DESKTOP_UPDATE_APPLY_TEST": applyMode},
		StartupTimeout: 75 * time.Second, Renderer: RendererExpectation{URLPrefix: "http://127.0.0.1:"},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = session.Close(context.Background())
		stopJ10Successor(home)
		if process, ok := session.process.(*execElectronProcess); ok {
			if dir := os.Getenv("J10_EVIDENCE"); dir != "" {
				_ = os.WriteFile(filepath.Join(dir, t.Name()+"-desktop.log"), []byte(process.output.String()), 0o600)
			}
		}
	})
	return session
}

func j10BrowserState(t *testing.T, endpoint, mode string) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 40*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "python3", "desktop_update_journey.py", endpoint, mode).CombinedOutput()
	if err != nil {
		t.Fatalf("desktop %s state: %v\n%s", mode, err, out)
	}
	var result map[string]any
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("desktop %s JSON: %v\n%s", mode, err, out)
	}
	return result
}

func waitJ10Receipt(t *testing.T, home string) map[string]any {
	t.Helper()
	deadline := time.NewTimer(2 * time.Minute)
	defer deadline.Stop()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-deadline.C:
			state, _ := os.ReadFile(filepath.Join(home, "electron", "update-state.json"))
			t.Fatalf("update receipt timeout; last state %s", state)
		case <-tick.C:
			state, _ := os.ReadFile(filepath.Join(home, "electron", "update-state.json"))
			var status struct{ State, Reason string }
			_ = json.Unmarshal(state, &status)
			if status.State == "failed" {
				t.Fatalf("real updater failed: %s", status.Reason)
			}
			data, err := os.ReadFile(filepath.Join(home, "electron", "update-receipt.json"))
			if err == nil {
				var receipt map[string]any
				if json.Unmarshal(data, &receipt) == nil {
					if receipt["outcome"] == "applied" && status.State != "healthy" {
						continue // Receipt persistence precedes the final durable transition.
					}
					t.Logf("real update receipt: %s", data)
					if dir := os.Getenv("J10_EVIDENCE"); dir != "" {
						_ = os.WriteFile(filepath.Join(dir, t.Name()+"-receipt.json"), data, 0o600)
					}
					return receipt
				}
			}
		}
	}
}

func assertJ10UpdateTrace(t *testing.T, home string, required []string) {
	t.Helper()
	specData, err := os.ReadFile("../../templates/vanilla/storage/update-lifecycle.spec.json")
	if err != nil {
		t.Fatal(err)
	}
	var specification struct {
		Initial          string
		TransitionMatrix map[string]map[string]*string
	}
	if err := json.Unmarshal(specData, &specification); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(home, "electron", "update-state.json"))
	if err != nil {
		t.Fatal(err)
	}
	var state struct {
		State       string
		Transitions []struct{ State, ObservedAt string }
	}
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
	}
	index := 0
	previous := specification.Initial
	for _, transition := range state.Transitions {
		if transition.ObservedAt == "" {
			t.Fatal("persisted transition has no observation time")
		}
		expected := specification.TransitionMatrix[previous][transition.State]
		if expected == nil || *expected != transition.State {
			t.Fatalf("production trace violates lifecycle contract: %s -> %s", previous, transition.State)
		}
		previous = transition.State
		if index < len(required) && transition.State == required[index] {
			index++
		}
	}
	if index != len(required) || state.State != required[len(required)-1] {
		t.Fatalf("production update trace does not contain %v: %s", required, data)
	}
	t.Logf("persisted production update trace: %s", data)
	if dir := os.Getenv("J10_EVIDENCE"); dir != "" {
		if err := os.WriteFile(filepath.Join(dir, t.Name()+"-transitions.json"), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func runJ10Applied(t *testing.T) {
	base, next := j10Artifact(t, "J10_BASE_APPIMAGE"), j10Artifact(t, "J10_NEXT_APPIMAGE")
	startJ10LPBS(t, next)
	home := t.TempDir()
	installed := filepath.Join(home, "Aquila.AppImage")
	digest := copyJ10Artifact(t, base, installed)
	session := startJ10Desktop(t, installed, home, digest)
	before := j10BrowserState(t, session.endpoint, "before")
	if before["info"].(map[string]any)["version"] != "0.0.1" {
		t.Fatalf("base version: %+v", before)
	}
	receipt := waitJ10Receipt(t, home)
	if receipt["outcome"] != "applied" || receipt["health"] != "passed" || receipt["toVersion"] != "0.0.2" {
		t.Fatalf("update did not apply: %+v", receipt)
	}
	assertJ10UpdateTrace(t, home, []string{"offered", "verified", "apply_pending", "applying", "installing", "relaunched", "healthy"})
	after := j10BrowserState(t, session.endpoint, "after")
	if j10Digest(t, installed) != j10Digest(t, next) {
		t.Fatal("installed AppImage bytes differ from the LPBS successor")
	}
	if after["info"].(map[string]any)["version"] != "0.0.2" {
		t.Fatalf("successor version: %+v", after)
	}
	if before["deviceId"] == "" || before["deviceId"] != after["deviceId"] || before["note"] != after["note"] {
		t.Fatalf("desktop data changed: before=%+v after=%+v", before, after)
	}
	response, err := http.Get("http://127.0.0.1:39565/__journey/evidence")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var events map[string]int
	if err := json.NewDecoder(response.Body).Decode(&events); err != nil {
		t.Fatal(err)
	}
	if events["update_check"] < 1 || events["update_download"] < 1 {
		t.Fatalf("LPBS did not serve real update: %+v", events)
	}
	t.Logf("LPBS delivery events=%v; preserved device=%s and saved note", events, before["deviceId"])
}

func TestJ10_AppImageUpdatesThroughLPBSChannel(t *testing.T) { runJ10Applied(t) }
func TestJ10_DataPreservedAcrossUpdate(t *testing.T)         { runJ10Applied(t) }

// Exercise Later in the native update dialog and ordinary Electron quit without
// the Restart Now test seam. The following launch must consume a real intent.
func TestJ10_LaterThenQuitRetainsUpdateReceipt(t *testing.T) {
	base, next := j10Artifact(t, "J10_BASE_APPIMAGE"), j10Artifact(t, "J10_NEXT_APPIMAGE")
	startJ10LPBS(t, next)
	home := t.TempDir()
	installed := filepath.Join(home, "Aquila.AppImage")
	digest := copyJ10Artifact(t, base, installed)
	session := startJ10DesktopMode(t, installed, home, digest, "")
	before := j10BrowserState(t, session.endpoint, "before")
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	output, err := exec.CommandContext(ctx, "python3", "desktop_update_later_journey.py").CombinedOutput()
	if err != nil {
		t.Fatalf("native Later dialog: %v\n%s", err, output)
	}
	t.Logf("native update dialog: %s", output)
	if _, err := os.Stat(filepath.Join(home, "electron", "update-intent.json")); !os.IsNotExist(err) {
		t.Fatalf("Later should not begin installation before ordinary quit: %v", err)
	}
	j10BrowserState(t, session.endpoint, "after") // Browser.close requests normal app quit.
	deadline := time.Now().Add(time.Minute)
	for j10Digest(t, installed) != j10Digest(t, next) {
		if time.Now().After(deadline) {
			t.Fatal("ordinary quit did not replace the AppImage")
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err := session.Close(context.Background()); err != nil && !errors.Is(err, os.ErrProcessDone) {
		t.Fatal(err)
	}
	marker, err := os.ReadFile(filepath.Join(home, "electron", "update-intent.json"))
	if err != nil {
		t.Fatal("ordinary quit omitted durable recovery intent: ", err)
	}
	t.Logf("ordinary-quit intent: %s", marker)
	successor := startJ10DesktopMode(t, installed, home, j10Digest(t, next), "")
	receipt := waitJ10Receipt(t, home)
	if receipt["outcome"] != "applied" || receipt["health"] != "passed" || receipt["toVersion"] != "0.0.2" {
		t.Fatalf("ordinary quit lost receipt: %+v", receipt)
	}
	assertJ10UpdateTrace(t, home, []string{"offered", "verified", "apply_pending", "applying", "installing", "relaunched", "healthy"})
	after := j10BrowserState(t, successor.endpoint, "after")
	if before["deviceId"] == "" || before["deviceId"] != after["deviceId"] || before["note"] != after["note"] {
		t.Fatalf("ordinary quit lost profile: before=%v after=%v", before, after)
	}
}

func TestJ10_InterruptedUpdateClassified(t *testing.T) {
	base, next := j10Artifact(t, "J10_BASE_APPIMAGE"), j10Artifact(t, "J10_NEXT_APPIMAGE")
	home := t.TempDir()
	installed := filepath.Join(home, "Aquila.AppImage")
	digest := copyJ10Artifact(t, base, installed)
	session := startJ10Desktop(t, installed, home, digest)
	before := j10BrowserState(t, session.endpoint, "before")
	// Seed an existing profile through a normal application shutdown before
	// offering the update. A forced crash may lose newly buffered DOMStorage;
	// this journey proves recovery preserves already saved user data.
	j10BrowserState(t, session.endpoint, "after")
	if err := session.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	startJ10LPBS(t, next)
	session = startJ10Desktop(t, installed, home, digest)
	restored := j10BrowserState(t, session.endpoint, "observe")
	if before["deviceId"] != restored["deviceId"] || before["note"] != restored["note"] {
		t.Fatalf("profile did not persist before update: before=%v restored=%v", before, restored)
	}
	// Interrupt an actual persisted apply intent, never manufacture a marker.
	deadline := time.NewTimer(2 * time.Minute)
	defer deadline.Stop()
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	interrupted := false
	for !interrupted {
		select {
		case <-deadline.C:
			t.Fatal("no real update intent to interrupt")
		case <-tick.C:
			if _, err := os.Stat(filepath.Join(home, "electron", "update-intent.json")); err == nil {
				// Freeze the launcher before enumerating descendants: scanning /proc
				// must not give the installer time to replace the predecessor.
				if err := session.process.Signal(syscall.SIGSTOP); err != nil {
					t.Fatal(err)
				}
				if err := session.process.Kill(); err != nil {
					t.Fatal(err)
				}
				interrupted = true
			}
		}
	}
	_ = session.Close(context.Background())
	// Stop new offers while observing recovery of the interrupted installation.
	request, _ := http.NewRequest(http.MethodPost, "http://127.0.0.1:39565/__journey/stop", nil)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	relaunched := startJ10Desktop(t, installed, home, digest)
	receipt := waitJ10Receipt(t, home)
	if receipt["outcome"] != "interrupted" {
		t.Fatalf("interrupted application was not classified: %+v", receipt)
	}
	assertJ10UpdateTrace(t, home, []string{"offered", "verified", "recovering"})
	after := j10BrowserState(t, relaunched.endpoint, "after")
	if before["deviceId"] != after["deviceId"] || before["note"] != after["note"] {
		t.Fatalf("interruption lost user data: before=%v after=%v", before, after)
	}
}

func TestJ10_WindowsBuildVerified(t *testing.T) {
	root := j10Artifact(t, "J10_WINDOWS_DIR")
	resultPath := j10Artifact(t, "J10_WINDOWS_RESULT")
	resultData, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Stages map[string]struct {
			Details struct {
				Build struct {
					PlatformResults map[string]struct{ Status, Artifact string } `json:"platform_results"`
				}
			}
		}
	}
	if err := json.Unmarshal(resultData, &result); err != nil {
		t.Fatal(err)
	}
	windows := result.Stages["build"].Details.Build.PlatformResults["windows-amd64"]
	if windows.Status != "PLATFORM_BUILD_STATUS_READY" {
		t.Fatalf("Windows build owner reported %s", windows.Status)
	}
	ownerRoot := root
	if retentionPath := os.Getenv("J10_WINDOWS_RETENTION"); retentionPath != "" {
		// The build cache is disposable. A retention receipt binds relocated
		// bytes (including the unchanged owner result) to their original build.
		data, err := os.ReadFile(retentionPath)
		if err != nil {
			t.Fatal(err)
		}
		var retention struct {
			SourceRoot string            `json:"source_root"`
			Files      map[string]string `json:"files"`
		}
		if err := json.Unmarshal(data, &retention); err != nil {
			t.Fatal(err)
		}
		if retention.SourceRoot == "" || len(retention.Files) == 0 {
			t.Fatal("retention receipt lacks original build identity or file digests")
		}
		ownerRoot = retention.SourceRoot
		for name, expected := range retention.Files {
			if !filepath.IsLocal(name) || expected == "" {
				t.Fatalf("invalid retained artifact: %q", name)
			}
			if actual := j10Digest(t, filepath.Join(root, name)); actual != expected {
				t.Fatalf("retained artifact %s changed: %s != %s", name, actual, expected)
			}
		}
		resultName, err := filepath.Rel(root, resultPath)
		if err != nil || retention.Files[resultName] == "" {
			t.Fatal("owner build result is not bound by the retention receipt")
		}
		artifactName, err := filepath.Rel(ownerRoot, windows.Artifact)
		if err != nil || retention.Files[artifactName] == "" {
			t.Fatal("owner Windows artifact is not bound by the retention receipt")
		}
	}
	if filepath.Dir(windows.Artifact) != filepath.Join(ownerRoot, "dist-electron") {
		t.Fatal("Windows artifact does not belong to the selected build")
	}

	installers, err := filepath.Glob(filepath.Join(root, "dist-electron", "*.exe"))
	if err != nil || len(installers) == 0 {
		t.Fatalf("real NSIS installer missing: %v", err)
	}
	for _, path := range installers {
		binary, err := pe.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		binary.Close()
		t.Logf("verified actual Windows installer %s", path)
	}
	paths, err := filepath.Glob(filepath.Join(root, "dist-electron", "win-unpacked", "*.exe"))
	if err != nil || len(paths) == 0 {
		t.Fatal("packaged Windows application is missing")
	}
	paths = append(paths, filepath.Join(root, "bundle", "runtime", "win-x64", "runtime.exe"))
	for _, path := range paths {
		binary, err := pe.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		if binary.Machine != pe.IMAGE_FILE_MACHINE_AMD64 {
			t.Fatalf("%s machine=%x", path, binary.Machine)
		}
		binary.Close()
	}
	data, err := os.ReadFile(filepath.Join(root, "bundle", "bundle.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Services []struct {
			Type     string
			Binaries map[string]struct{ Path string }
		}
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, service := range manifest.Services {
		if service.Type != "api-binary" {
			continue
		}
		count++
		binary, err := pe.Open(filepath.Join(root, "bundle", service.Binaries["win-x64"].Path))
		if err != nil {
			t.Fatal(err)
		}
		if binary.Machine != pe.IMAGE_FILE_MACHINE_AMD64 {
			t.Fatal("Windows service is not AMD64 PE")
		}
		binary.Close()
	}
	if count < 2 {
		t.Fatal("root and embedded peer Windows APIs missing")
	}
	t.Log(fmt.Sprintf("Windows application and bundled runtime are AMD64 PE; installer count %d", len(installers)))
}

// electron-updater starts a detached successor. Its inherited private HOME is
// the fixture ownership boundary even if the original process has exited.
func stopJ10Successor(home string) {
	entries, _ := os.ReadDir("/proc")
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid == os.Getpid() {
			continue
		}
		environment, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "environ"))
		if err != nil {
			continue
		}
		owned := false
		for _, item := range bytes.Split(environment, []byte{0}) {
			if string(item) == "HOME="+home {
				owned = true
				break
			}
		}
		if owned {
			_ = syscall.Kill(pid, syscall.SIGTERM)
		}
	}
}

func j10Digest(t *testing.T, path string) string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(digest.Sum(nil))
}
