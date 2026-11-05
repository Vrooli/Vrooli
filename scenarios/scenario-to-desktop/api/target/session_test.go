package target

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type fakeElectronProcess struct {
	pid      int
	server   *http.Server
	listener net.Listener
	done     chan error
	once     sync.Once
}

func TestVerifyOfflineBundledAPIRequiresHealthBeforeTargetBinding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("health probe path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("starting"))
	}))
	defer server.Close()

	err := verifyOfflineBundledAPIWithin(server.URL, 15000, filepath.Join(t.TempDir(), "relay-errors"), 50*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "private_port=15000") || !strings.Contains(err.Error(), "health_status=503") || !strings.Contains(err.Error(), "starting") {
		t.Fatalf("unhealthy bundled API diagnostic = %v", err)
	}
}

func TestElectronSessionWaitForExitUsesProcessCompletionSignal(t *testing.T) {
	session := &ElectronSession{waitDone: make(chan struct{})}
	close(session.waitDone)
	if err := session.WaitForExit(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func (p *fakeElectronProcess) PID() int { return p.pid }
func (p *fakeElectronProcess) Signal(os.Signal) error {
	p.once.Do(func() {
		p.done <- p.server.Shutdown(context.Background())
	})
	return nil
}

func (p *fakeElectronProcess) Kill() error {
	p.once.Do(func() {
		p.done <- p.server.Close()
	})
	return nil
}
func (p *fakeElectronProcess) Wait() error { return <-p.done }

func fakeElectronStarter(pages string) ProcessStarter {
	return func(_ context.Context, _ string, args []string, _ []string) (ElectronProcess, error) {
		var port int
		for _, arg := range args {
			if strings.HasPrefix(arg, "--remote-debugging-port=") {
				parsed, err := strconv.Atoi(strings.TrimPrefix(arg, "--remote-debugging-port="))
				if err != nil {
					return nil, err
				}
				port = parsed
			}
		}
		listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			return nil, err
		}
		var process *fakeElectronProcess
		upgrader := websocket.Upgrader{}
		server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/json/list" {
				w.Header().Set("content-type", "application/json")
				_, _ = w.Write([]byte(pages))
				return
			}
			if r.URL.Path == "/json/version" {
				w.Header().Set("content-type", "application/json")
				_, _ = fmt.Fprintf(w, `{"webSocketDebuggerUrl":"ws://127.0.0.1:%d/devtools/browser/test"}`, port)
				return
			}
			if r.URL.Path == "/devtools/browser/test" {
				connection, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					return
				}
				defer connection.Close()
				var command struct {
					ID     int    `json:"id"`
					Method string `json:"method"`
				}
				if connection.ReadJSON(&command) == nil && command.Method == "Browser.close" {
					_ = connection.WriteJSON(map[string]any{"id": command.ID, "result": map[string]any{}})
					go func() { _ = process.Signal(os.Interrupt) }()
				}
				return
			}
			http.NotFound(w, r)
		})}
		process = &fakeElectronProcess{pid: 4242, server: server, listener: listener, done: make(chan error, 1)}
		go func() { _ = server.Serve(listener) }()
		return process, nil
	}
}

func sessionLaunchOptions() ElectronLaunchOptions {
	return ElectronLaunchOptions{
		ContextID:        "ctx-1",
		ScenarioName:     "example-app",
		ArtifactDigest:   "sha256:artifact",
		TargetID:         "local-linux",
		JourneyID:        "login",
		ProfileID:        "normal",
		IsolationLeaseID: "lease-1",
	}
}

func TestStartElectronSessionSelectsRendererAndCleansOwnedProfile(t *testing.T) {
	artifact := filepath.Join(t.TempDir(), "example-app")
	if err := os.WriteFile(artifact, []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	session, err := StartElectronSession(context.Background(), ElectronSessionOptions{
		ArtifactPath: artifact,
		Launch:       sessionLaunchOptions(),
		Renderer:     RendererExpectation{URLPrefix: "http://127.0.0.1:4400"},
		StartProcess: fakeElectronStarter(`[{"id":"main","type":"page","url":"http://127.0.0.1:4400","webSocketDebuggerUrl":"ws://main"}]`),
	})
	if err != nil {
		t.Fatal(err)
	}
	profile := session.profileDir
	if session.Target().GetRendererId() != "main" || session.PID() != 4242 || !session.Target().GetLoopbackOnly() {
		t.Fatalf("target = %#v", session.Target())
	}
	if err := session.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(profile); !os.IsNotExist(err) {
		t.Fatalf("owned profile still exists: %s, err=%v", profile, err)
	}
	if err := session.Close(context.Background()); err != nil {
		t.Fatalf("second Close = %v", err)
	}
}

func TestBundledElectronSessionRemovesInheritedEndpointOverrides(t *testing.T) {
	artifact := filepath.Join(t.TempDir(), "example-app")
	if err := os.WriteFile(artifact, []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_VALIDATION_API_URL", "http://127.0.0.1:1")
	t.Setenv("VROOLI_VALIDATION_RENDERER_URL", "http://127.0.0.1:2")
	var launchedEnv []string
	_, err := StartElectronSession(context.Background(), ElectronSessionOptions{
		ArtifactPath: artifact,
		Launch: func() ElectronLaunchOptions {
			options := sessionLaunchOptions()
			options.BundledRuntime = true
			return options
		}(),
		Environment: map[string]string{"VROOLI_VALIDATION_API_URL": "http://127.0.0.1:3", "DISPLAY": ":99"},
		StartProcess: func(_ context.Context, _ string, _ []string, env []string) (ElectronProcess, error) {
			launchedEnv = append([]string(nil), env...)
			return nil, fmt.Errorf("captured launch environment")
		},
	})
	if err == nil || !strings.Contains(err.Error(), "captured launch environment") {
		t.Fatalf("launch error = %v", err)
	}
	for _, entry := range launchedEnv {
		if strings.HasPrefix(entry, "VROOLI_VALIDATION_API_URL=") || strings.HasPrefix(entry, "VROOLI_VALIDATION_RENDERER_URL=") {
			t.Fatalf("bundled child inherited validation endpoint override: %q", entry)
		}
	}
	if !strings.Contains(strings.Join(launchedEnv, "\n"), "DISPLAY=:99") {
		t.Fatal("bundled child environment lost the managed display")
	}
}

func TestStartElectronSessionWaitsForRendererReadiness(t *testing.T) {
	artifact := filepath.Join(t.TempDir(), "example-app")
	if err := os.WriteFile(artifact, []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	requests := 0
	session, err := StartElectronSession(context.Background(), ElectronSessionOptions{
		ArtifactPath:   artifact,
		Launch:         sessionLaunchOptions(),
		StartupTimeout: 500 * time.Millisecond,
		Renderer:       RendererExpectation{ID: "main"},
		StartProcess: func(_ context.Context, _ string, args []string, _ []string) (ElectronProcess, error) {
			var port int
			for _, arg := range args {
				if strings.HasPrefix(arg, "--remote-debugging-port=") {
					parsed, err := strconv.Atoi(strings.TrimPrefix(arg, "--remote-debugging-port="))
					if err != nil {
						return nil, err
					}
					port = parsed
				}
			}
			listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
			if err != nil {
				return nil, err
			}
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/json/list" {
					http.NotFound(w, r)
					return
				}
				mu.Lock()
				requests++
				ready := requests > 1
				mu.Unlock()
				if ready {
					_, _ = w.Write([]byte(`[{"id":"main","type":"page","url":"http://127.0.0.1:4400","webSocketDebuggerUrl":"ws://main"}]`))
					return
				}
				_, _ = w.Write([]byte(`[]`))
			})}
			process := &fakeElectronProcess{pid: 4242, server: server, listener: listener, done: make(chan error, 1)}
			go func() { _ = server.Serve(listener) }()
			return process, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = session.Close(context.Background()) }()
	if session.Target().GetRendererId() != "main" {
		t.Fatalf("renderer = %q, want main", session.Target().GetRendererId())
	}
}

func TestStartElectronSessionWithHelloDesktopAppImage(t *testing.T) {
	artifact := strings.TrimSpace(os.Getenv("VROOLI_ELECTRON_INTEGRATION_APP"))
	if artifact == "" {
		t.Skip("set VROOLI_ELECTRON_INTEGRATION_APP to run the real AppImage launch check")
	}
	if os.Getenv("DISPLAY") == "" {
		t.Skip("real Electron launch requires a managed display; run under xvfb-run")
	}

	session, err := StartElectronSession(context.Background(), ElectronSessionOptions{
		ArtifactPath:   artifact,
		Launch:         sessionLaunchOptions(),
		StartupTimeout: 15 * time.Second,
		Renderer:       RendererExpectation{URLPrefix: "http://127.0.0.1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = session.Close(context.Background()) }()
	if session.Target().GetRendererId() == "" || session.Target().GetRendererUrl() == "" {
		t.Fatalf("real Electron target is incomplete: %#v", session.Target())
	}
}

func TestStartElectronSessionRefusesAmbiguousRendererAndCleansUp(t *testing.T) {
	artifact := filepath.Join(t.TempDir(), "example-app")
	if err := os.WriteFile(artifact, []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := StartElectronSession(context.Background(), ElectronSessionOptions{
		ArtifactPath:   artifact,
		Launch:         sessionLaunchOptions(),
		StartupTimeout: 200 * time.Millisecond,
		StartProcess: fakeElectronStarter(`[
            {"id":"one","type":"page","url":"http://one","webSocketDebuggerUrl":"ws://one"},
            {"id":"two","type":"page","url":"http://two","webSocketDebuggerUrl":"ws://two"}
        ]`),
	})
	if err == nil || !strings.Contains(err.Error(), "identity is required") {
		t.Fatalf("error = %v, want missing identity refusal", err)
	}
}

func TestStartElectronSessionRefusesUnidentifiedRendererAndCleansUp(t *testing.T) {
	artifact := filepath.Join(t.TempDir(), "example-app")
	if err := os.WriteFile(artifact, []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := StartElectronSession(context.Background(), ElectronSessionOptions{
		ArtifactPath:   artifact,
		Launch:         sessionLaunchOptions(),
		StartupTimeout: 200 * time.Millisecond,
		StartProcess: fakeElectronStarter(`[{
            "id":"splash",
            "type":"page",
            "title":"Example - Loading",
            "url":"file:///splash.html",
            "webSocketDebuggerUrl":"ws://splash"
        }]`),
	})
	if err == nil || !strings.Contains(err.Error(), "identity is required") {
		t.Fatalf("error = %v, want missing identity refusal", err)
	}
}

func TestStartElectronSessionRefusesRendererCrashAndCleansUp(t *testing.T) {
	artifact := filepath.Join(t.TempDir(), "example-app")
	if err := os.WriteFile(artifact, []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := StartElectronSession(context.Background(), ElectronSessionOptions{
		ArtifactPath:   artifact,
		Launch:         sessionLaunchOptions(),
		StartupTimeout: 200 * time.Millisecond,
		StartProcess: func(ctx context.Context, path string, args []string, env []string) (ElectronProcess, error) {
			process, err := fakeElectronStarter(`[{"id":"main","type":"page","url":"http://127.0.0.1:4400","webSocketDebuggerUrl":"ws://main"}]`)(ctx, path, args, env)
			if err != nil {
				return nil, err
			}
			if err := process.Kill(); err != nil {
				return nil, err
			}
			return process, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "exited before CDP became ready") {
		t.Fatalf("error = %v, want renderer crash refusal", err)
	}
}

func TestStartElectronSessionRejectsMissingArtifact(t *testing.T) {
	_, err := StartElectronSession(context.Background(), ElectronSessionOptions{ArtifactPath: "/does/not/exist", Launch: sessionLaunchOptions()})
	if err == nil || !strings.Contains(err.Error(), "stat Electron artifact") {
		t.Fatalf("error = %v, want artifact failure", err)
	}
}

func TestIsExpectedElectronTermination(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: true},
		{name: "process done", err: os.ErrProcessDone, want: true},
		{name: "interrupt", err: fmt.Errorf("signal: interrupt"), want: true},
		{name: "killed", err: fmt.Errorf("signal: killed"), want: true},
		{name: "unexpected", err: fmt.Errorf("exit status 1"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isExpectedElectronTermination(tt.err); got != tt.want {
				t.Fatalf("isExpectedElectronTermination(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestElectronRestartPreservesProfileAndTransfersCleanup(t *testing.T) {
	artifact := filepath.Join(t.TempDir(), "example-app")
	if err := os.WriteFile(artifact, []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	session, err := StartElectronSession(context.Background(), ElectronSessionOptions{
		ArtifactPath: artifact, Launch: sessionLaunchOptions(), Renderer: RendererExpectation{URLPrefix: "http://127.0.0.1:4400"},
		StartProcess: fakeElectronStarter(`[{"id":"main","type":"page","url":"http://127.0.0.1:4400","webSocketDebuggerUrl":"ws://main"}]`),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close(context.Background())
	if _, err := session.Restart(context.Background()); err == nil {
		t.Fatal("running process restart accepted")
	}
	profile := session.profileDir
	marker := filepath.Join(profile, "window-state.json")
	if err := os.WriteFile(marker, []byte(`{"width":1200,"height":800}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := session.RequestExit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := session.WaitForExit(context.Background()); err != nil {
		t.Fatal(err)
	}
	starter := session.options.StartProcess
	session.options.StartProcess = func(context.Context, string, []string, []string) (ElectronProcess, error) {
		return nil, fmt.Errorf("injected restart failure")
	}
	if _, err := session.Restart(context.Background()); err == nil {
		t.Fatal("failed launch accepted")
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("failed restart removed profile", err)
	}
	session.options.StartProcess = starter
	next, err := session.Restart(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer next.Close(context.Background())
	if next.profileDir != profile || !next.ownedProfile {
		t.Fatal("profile ownership not retained")
	}
	if err := session.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("old close removed profile", err)
	}
	if _, err := session.Restart(context.Background()); err == nil {
		t.Fatal("superseded session restarted")
	}
	if err := next.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(profile); !os.IsNotExist(err) {
		t.Fatal("profile was not removed", err)
	}
}

func TestPinnedOfflineAppImageBrowserCloseRestartRetainsPrivateSession(t *testing.T) {
	if os.Getenv("VROOLI_TARGET_OFFLINE_LIFECYCLE") != "1" {
		t.Skip("explicit pinned AppImage lifecycle check required")
	}
	artifact := os.Getenv("VROOLI_TARGET_OFFLINE_LIFECYCLE_ARTIFACT")
	if artifact == "" {
		t.Fatal("pinned AppImage path is required")
	}
	contents, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(contents)
	if got := hex.EncodeToString(digest[:]); got != "137c187f0b36c178a51244314292ffd38b64ad67d60c5011044f1061cd4ed338" {
		t.Fatalf("pinned AppImage digest = %s", got)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	launch := sessionLaunchOptions()
	launch.ScenarioName = "device-studio"
	launch.ArtifactDigest = "sha256:" + hex.EncodeToString(digest[:])
	launch.ContextID = "offline-close-lifecycle-test"
	launch.TargetID = "local-linux-amd64"
	launch.JourneyID = "desktop-offline-png-roundtrip"
	launch.ProfileID = "offline"
	launch.IsolationLeaseID = "pinned-appimage-browser-close-test"
	first, err := StartElectronSession(ctx, ElectronSessionOptions{
		ArtifactPath: artifact, Launch: func() ElectronLaunchOptions {
			launch.BundledRuntime = true
			launch.OfflineNetwork = true
			return launch
		}(), Environment: map[string]string{"LIBGL_ALWAYS_SOFTWARE": "1", "MESA_LOADER_DRIVER_OVERRIDE": "llvmpipe"},
		OfflineNetwork: true, StartupTimeout: 90 * time.Second, Renderer: RendererExpectation{URLPrefix: "http://127.0.0.1:"},
	})
	if err != nil {
		t.Fatal(err)
	}
	root := first.networkRoot
	if err := first.RequestExit(ctx); err != nil {
		_ = first.Close(context.Background())
		t.Fatal(err)
	}
	if err := first.WaitForExit(ctx); err != nil {
		_ = first.Close(context.Background())
		t.Fatal(err)
	}
	second, err := first.Restart(ctx)
	if err != nil {
		_ = first.Close(context.Background())
		t.Fatal(err)
	}
	if second.networkRoot != root || second.profileDir != first.profileDir {
		_ = second.Close(context.Background())
		t.Fatal("restart did not retain the same private root and profile")
	}
	if err := second.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("private root remains after owner cleanup: %s (%v)", root, err)
	}
}

// Explicit integration opt-in: the target adapter owns the app/profile; the
// operator selects the desktop matching the local observation helper.
func TestStartPortalCompanionContextOnSelectedDesktop(t *testing.T) {
	if os.Getenv("VROOLI_PORTAL_CONTEXT_INTEGRATION") != "1" {
		t.Skip("explicit selected-desktop integration opt-in required")
	}
	artifact := os.Getenv("VROOLI_ELECTRON_INTEGRATION_APP")
	source := os.Getenv("VROOLI_DESKTOP_ACTIVATION_CONFIG")
	if artifact == "" || source == "" || os.Getenv("DISPLAY") == "" {
		t.Fatal("artifact, activation source and selected DISPLAY required")
	}
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	bytes, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(bytes)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	title := fmt.Sprintf("Portal-context-fixture-%d", os.Getpid())
	fixture := exec.CommandContext(ctx, "xmessage", "-title", title, "Portal context validation fixture")
	if err = fixture.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = fixture.Process.Kill(); _ = fixture.Wait() }()
	opts := sessionLaunchOptions()
	opts.ScenarioName = "portal"
	opts.ArtifactDigest = hex.EncodeToString(digest[:])
	opts.ContextID = title
	opts.TargetID = title
	opts.JourneyID = "portal-context"
	opts.ProfileID = "selected-desktop"
	opts.IsolationLeaseID = title
	app, err := StartElectronSession(ctx, ElectronSessionOptions{ArtifactPath: artifact, Launch: opts, StartupTimeout: 20 * time.Second, Renderer: RendererExpectation{URLPrefix: "http://127.0.0.1:24965"}, Environment: map[string]string{"DISPLAY": os.Getenv("DISPLAY"), "XAUTHORITY": os.Getenv("XAUTHORITY"), "VROOLI_DESKTOP_ACTIVATION_CONFIG": source, "VROOLI_VALIDATION_API_URL": "http://127.0.0.1:17476", "VROOLI_VALIDATION_RENDERER_URL": "http://127.0.0.1:24965"}})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := app.Close(context.Background()); err != nil {
			t.Errorf("managed cleanup: %v", err)
		}
	}()
	check := exec.CommandContext(ctx, "node", filepath.Join(root, "scenarios/portal/docs/internal/plans/artifacts/portal-everywhere-20260905/live-native-context.cjs"), app.Endpoint(), title, strconv.Itoa(fixture.Process.Pid))
	check.Dir = root
	output, err := check.CombinedOutput()
	t.Log(string(output))
	if err != nil {
		t.Fatal(err)
	}
}
