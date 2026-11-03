package target

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// This journey runs the real generated AppImage through the target-owned
// lifecycle. No scenario API, renderer, runtime or resource is substituted.
func startJ9Bundle(t *testing.T) (*ElectronSession, string) {
	t.Helper()
	artifact := os.Getenv("J9_APPIMAGE")
	if artifact == "" {
		t.Skip("J9_APPIMAGE selects the real pipeline artifact")
	}
	if os.Getenv("DISPLAY") == "" {
		t.Fatal("run with the managed xvfb-run display")
	}
	file, err := os.Open(artifact)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
	home := t.TempDir()
	if binary := os.Getenv("J9_CODEX_BINARY"); binary != "" {
		bin := filepath.Join(home, ".local", "bin")
		if err := os.MkdirAll(bin, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(binary, filepath.Join(bin, "codex")); err != nil {
			t.Fatal(err)
		}
	}

	for _, dir := range []string{"/usr/bin", "/bin"} {
		if _, err := os.Stat(filepath.Join(dir, "vrooli")); !os.IsNotExist(err) {
			t.Fatalf("PATH contains vrooli: %s", dir)
		}
	}
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	t.Cleanup(cancel)
	session, err := StartElectronSession(ctx, ElectronSessionOptions{
		ArtifactPath:   artifact,
		Launch:         ElectronLaunchOptions{ContextID: "agent-connect-j9", ScenarioName: "web-console", ArtifactDigest: "sha256:" + hex.EncodeToString(digest.Sum(nil)), TargetID: "local-linux-xvfb", JourneyID: "J9", ProfileID: "clean-home", IsolationLeaseID: "j9-clean-home", BundledRuntime: true, UserDataDir: filepath.Join(home, "electron")},
		Environment:    map[string]string{"HOME": home, "USERPROFILE": home, "PATH": filepath.Join(home, ".local", "bin") + ":/usr/bin:/bin", "VROOLI_ROOT": "", "XDG_CONFIG_HOME": filepath.Join(home, ".config"), "XDG_DATA_HOME": filepath.Join(home, ".local/share"), "XDG_STATE_HOME": filepath.Join(home, ".local/state"), "XDG_CACHE_HOME": filepath.Join(home, ".cache"), "LIBGL_ALWAYS_SOFTWARE": "1"},
		StartupTimeout: 75 * time.Second, Renderer: RendererExpectation{URLPrefix: "http://127.0.0.1:"},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	t.Logf("real AppImage sha256:%s", hex.EncodeToString(digest.Sum(nil)))
	return session, home
}

func TestJ9_BundleStartsWithNoVrooliOnPath(t *testing.T) {
	session, _ := startJ9Bundle(t)
	ctx := t.Context()
	target := session.Target()
	if target.GetRendererId() == "" {
		t.Fatal("bundle has no renderer")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.GetRendererUrl(), nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || !strings.Contains(string(data), `id="root"`) {
		t.Fatalf("bundle renderer is not serving Web Console: status %d", resp.StatusCode)
	}
	t.Logf("selected renderer %s", target.GetRendererUrl())
}

// runtimePort uses the real bundle's authenticated discovery endpoint.
func runtimePort(t *testing.T, home, scenario string) int {
	t.Helper()
	var dataDir string
	err := filepath.WalkDir(home, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Name() == "ipc_port" && filepath.Base(filepath.Dir(path)) == "runtime" {
			dataDir = filepath.Dir(filepath.Dir(path))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if dataDir == "" {
		t.Fatal("bundle did not create IPC port metadata")
	}
	raw, err := os.ReadFile(filepath.Join(dataDir, "runtime", "ipc_port"))
	if err != nil {
		t.Fatal(err)
	}
	ipc, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	token, err := os.ReadFile(filepath.Join(dataDir, "runtime", "auth_token"))
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/scenario-port?scenario=%s&port_name=API_PORT", ipc, scenario), nil)
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(token)))
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var result struct{ Port int }
	if response.StatusCode != 200 {
		t.Fatalf("runtime discovery: HTTP %d", response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Port == 0 {
		t.Fatal("runtime discovered no port")
	}
	return result.Port
}

func TestJ9_RequiredDependencyBundledAndAnswers(t *testing.T) {
	session, home := startJ9Bundle(t)
	_ = session
	port := runtimePort(t, home, "audio-tools")
	response, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var health map[string]any
	if err := json.NewDecoder(response.Body).Decode(&health); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 || health["status"] != "healthy" {
		t.Fatalf("embedded audio health: %d %+v", response.StatusCode, health)
	}
	// No host registry exists in this home; the dependency must be supplied by
	// this bundle, which has a distinct API process and port.
	if web := runtimePort(t, home, "web-console"); web == port {
		t.Fatal("dependency resolved to root API")
	}
	t.Logf("embedded Audio Tools answers /health on %d: %+v", port, health)
}

func TestJ9_NoDeadActionInUI(t *testing.T) {
	session, _ := startJ9Bundle(t)
	runJ9Browser(t, session, "actions")
}

func TestJ9_FindsAgentAndReadsSignIn(t *testing.T) {
	session, _ := startJ9Bundle(t)
	runJ9Browser(t, session, "agent")
}

func runJ9Browser(t *testing.T, session *ElectronSession, check string) {
	t.Helper()
	evidence := os.Getenv("J9_EVIDENCE")
	if evidence == "" {
		evidence = t.TempDir()
	}
	cmd := exec.CommandContext(t.Context(), "python3", "desktop_journey.py", session.Target().GetCdpEndpoint(), session.Target().GetRendererUrl(), check, evidence)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("real Electron UI %s: %v\n%s", check, err, output)
	}
	t.Log(string(output))
}

func TestJ9_NoVrooliCallerLintPasses(t *testing.T) {
	// The full-stack fallback is permitted only behind its desktop boundary;
	// native discovery must fail closed instead of trying the CLI on a miss.
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	census := exec.CommandContext(t.Context(), "go", "test", "./internal/cli/rootcli/invokers", "-run", "^TestDesktopConsumersUseCanonicalInvoker$", "-count=1")
	census.Dir = root
	if output, err := census.CombinedOutput(); err != nil {
		t.Fatalf("caller census: %v\n%s", err, output)
	}
	command := exec.CommandContext(t.Context(), "go", "test", "./cliutil", "-run", "^TestDesktopPortCallsRuntimeAndNeverCLIFallback$", "-count=1")
	command.Dir = filepath.Join(root, "packages", "cli-core")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("desktop invocation boundary: %v\n%s", err, output)
	}
	t.Log("canonical caller census and desktop no-fallback regression pass")
}

// The unrelated HTTP origin is an attacker fixture, not a substituted product
// service. Both the app and its identity owner are the real packaged runtime.
func TestJ9_UnrelatedOriginCannotGetRuntimeToken(t *testing.T) {
	session, _ := startJ9Bundle(t)
	attacker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, "<!doctype html><title>Unrelated origin</title><p>Unrelated local service</p>")
	}))
	defer attacker.Close()
	evidence := os.Getenv("J9_EVIDENCE")
	if evidence == "" {
		evidence = t.TempDir()
	}
	cmd := exec.CommandContext(t.Context(), "python3", "desktop_origin_journey.py", session.Target().GetCdpEndpoint(), session.Target().GetRendererUrl(), attacker.URL+"/", evidence)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("real renderer authority: %v\n%s", err, output)
	}
	t.Log(string(output))
}
