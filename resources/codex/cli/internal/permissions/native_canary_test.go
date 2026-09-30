package permissions

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// This opt-in test runs native commands, never a model. Its isolated config
// does not touch the user's config or credentials. The explicit executable
// prevents a different PATH installation from masquerading as the target.
func TestNativeExecutionSandboxCanary(t *testing.T) {
	binary := os.Getenv("VROOLI_CODEX_NATIVE_BINARY")
	if binary == "" {
		t.Skip("set VROOLI_CODEX_NATIVE_BINARY to qualify an installed native runtime")
	}
	if !filepath.IsAbs(binary) {
		t.Fatal("canary executable must be absolute")
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Fatal(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	scratch, err := os.MkdirTemp(home, ".vrooli-permission-canary-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(scratch)
	workspace := filepath.Join(scratch, "workspace")
	if err = os.Mkdir(workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	forbidden := filepath.Join(scratch, "outside.txt")
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); fmt.Fprint(w, "canary") }))
	defer server.Close()
	e := testExecution()
	e.Network.Domains = map[string]string{"127.0.0.1": "allow", "localhost": "deny"}
	configHome := filepath.Join(scratch, "config")
	if err = os.Mkdir(configHome, 0o700); err != nil {
		t.Fatal(err)
	}
	a := &Adapter{SettingsPath: filepath.Join(configHome, "config.toml"), Scope: ScopeUser}
	if err = a.Save(Policy{Execution: e, ExecutionActive: true}); err != nil {
		t.Fatal(err)
	}
	blocked := strings.Replace(server.URL, "127.0.0.1", "localhost", 1)
	script := `import pathlib,sys,urllib.request,socket,errno
from urllib.parse import urlparse
print("proxy endpoints:",[(key,urlparse(value).hostname,urlparse(value).port) for key,value in urllib.request.getproxies().items() if key in ("http","https","all")],flush=True)
pathlib.Path("allowed.txt").write_text("allowed")
try:
 pathlib.Path(sys.argv[1]).write_text("forbidden")
except OSError as error:
 assert error.errno in (errno.EACCES,errno.EPERM,errno.EROFS),error
else:
 raise AssertionError("outside workspace write succeeded")
assert urllib.request.urlopen(sys.argv[2],timeout=5).read()==b"canary"
try:
 urllib.request.urlopen(sys.argv[3],timeout=5)
except Exception:
 pass
else:
 raise AssertionError("denied network destination succeeded")
from urllib.parse import urlparse
address=urlparse(sys.argv[2])
try:
 socket.create_connection((address.hostname,address.port),timeout=2)
except OSError:
 pass
else:
 raise AssertionError("direct socket bypassed network proxy")
print("PASS workspace write, outside write denied, loopback allowed, denied hostname blocked, direct proxy bypass blocked")
`
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "sandbox", "-C", workspace, "-P", ExecutionProfile, python, "-c", script, forbidden, server.URL, blocked)
	cmd.Env = canaryEnvironment(filepath.Dir(a.SettingsPath))
	output, err := cmd.CombinedOutput()
	t.Logf("native executable=%s\n%s", binary, output)
	if err != nil {
		t.Fatalf("native sandbox canary: %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("denied request reached local fixture: request count %d", requests.Load())
	}
	if _, err = os.Stat(forbidden); !os.IsNotExist(err) {
		t.Fatal("forbidden artifact exists")
	}
	disabled := false
	e.Filesystem.Workspace = "read"
	e.Network.Enabled = &disabled
	e.Network.Domains = nil
	if err = a.Save(Policy{Execution: e, ExecutionActive: true}); err != nil {
		t.Fatal(err)
	}
	script = `import pathlib,sys,urllib.request,errno
try:
 pathlib.Path("read-only-denied.txt").write_text("forbidden")
except OSError as error:
 assert error.errno in (errno.EACCES,errno.EPERM,errno.EROFS),error
else:
 raise AssertionError("read-only workspace write succeeded")
try:
 urllib.request.urlopen(sys.argv[1],timeout=3)
except Exception:
 pass
else:
 raise AssertionError("network-disabled request succeeded")
print("PASS read-only workspace, disabled network")
`
	cmd = exec.CommandContext(ctx, binary, "sandbox", "-C", workspace, "-P", ExecutionProfile, python, "-c", script, server.URL)
	cmd.Env = canaryEnvironment(configHome)
	output, err = cmd.CombinedOutput()
	t.Logf("%s", output)
	if err != nil {
		t.Fatalf("read-only/network-off canary: %v", err)
	}
	if requests.Load() != 1 {
		t.Fatal("network-off request reached fixture")
	}
}

func canaryEnvironment(configHome string) []string {
	env := []string{}
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		lower := strings.ToLower(key)
		if key == "CODEX_HOME" || lower == "http_proxy" || lower == "https_proxy" || lower == "all_proxy" || lower == "no_proxy" {
			continue
		}
		env = append(env, entry)
	}
	return append(env, "CODEX_HOME="+configHome)
}

func TestNativeOfflineExecutionSandboxCanary(t *testing.T) {
	binary := os.Getenv("VROOLI_CODEX_NATIVE_BINARY")
	if binary == "" {
		t.Skip("set VROOLI_CODEX_NATIVE_BINARY to qualify an installed native runtime")
	}
	if !filepath.IsAbs(binary) {
		t.Fatal("native executable must be absolute")
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Fatal(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	scratch, err := os.MkdirTemp(home, ".vrooli-offline-canary-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(scratch)
	workspace := filepath.Join(scratch, "workspace")
	configHome := filepath.Join(scratch, "config")
	for _, path := range []string{workspace, configHome} {
		if err = os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	a := &Adapter{SettingsPath: filepath.Join(configHome, "config.toml"), Scope: ScopeUser}
	e := testExecution()
	disabled := false
	e.Network.Enabled = &disabled
	e.Network.Domains = nil
	if err = a.Save(Policy{Execution: e, ExecutionActive: true}); err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); fmt.Fprint(w, "unexpected") }))
	defer server.Close()
	script := `import pathlib,sys,urllib.request,errno
pathlib.Path("allowed.txt").write_text("allowed")
try:
 pathlib.Path(sys.argv[1]).write_text("forbidden")
except OSError as error:
 assert error.errno in (errno.EACCES,errno.EPERM,errno.EROFS),error
else:
 raise AssertionError("outside write succeeded")
try:
 urllib.request.urlopen(sys.argv[2],timeout=2)
except Exception:
 pass
else:
 raise AssertionError("disabled network request succeeded")
print("PASS offline workspace write, outside write denied, network disabled")
`
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "sandbox", "-C", workspace, "-P", ExecutionProfile, python, "-c", script, filepath.Join(scratch, "outside.txt"), server.URL)
	cmd.Env = canaryEnvironment(configHome)
	output, err := cmd.CombinedOutput()
	t.Logf("native executable=%s\n%s", binary, output)
	if err != nil {
		t.Fatalf("offline native canary: %v", err)
	}
	if requests.Load() != 0 {
		t.Fatal("network-off fixture received a request")
	}
}
