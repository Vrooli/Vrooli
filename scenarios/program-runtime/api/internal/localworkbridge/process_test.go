//go:build linux

package localworkbridge

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/vrooli/pyenv-go"
)

// A finite test binary substitutes for CPython; no host Python, listener,
// credential, owner API or network is used. The production exec/IPC is real.
func TestMain(m *testing.M) {
	if len(os.Args) == 4 && os.Args[1] == "-I" && os.Args[2] == "-S" {
		raw, _ := io.ReadAll(os.Stdin)
		var packet map[string]any
		if json.Unmarshal(raw, &packet) != nil || packet["schema"] != "flywheel.local-work-native-ipc.v1" {
			os.Exit(9)
		}
		if len(os.Environ()) != 0 || strings.Contains(strings.Join(os.Args, " "), "fixture-secret") || strings.Contains(string(raw), "fixture-secret") {
			os.Exit(10)
		}
		mode, _ := os.ReadFile(os.Args[3])
		switch string(mode) {
		case "secret":
			req := os.NewFile(3, "requests")
			reply := os.NewFile(4, "responses")
			fmt.Fprintln(req, `{"purpose":"native_webhook_verify"}`)
			line, _ := bufio.NewReader(reply).ReadString('\n')
			if !strings.Contains(line, `"value_base64":"Zml4dHVyZS1zZWNyZXQ="`) {
				os.Exit(11)
			}
		case "scope":
			fmt.Fprintln(os.NewFile(3, "requests"), `{"purpose":"flywheel_submit"}`)
			time.Sleep(time.Minute)
		case "oversize":
			fmt.Print(strings.Repeat("x", 129))
			os.Exit(0)
		case "stderr":
			fmt.Fprint(os.Stderr, strings.Repeat("fixture-secret", 400))
			os.Exit(0)
		case "malformed":
			fmt.Println(`{"http_status":204,"http_status":204}`)
			os.Exit(0)
		case "extra":
			fmt.Println(`{"http_status":204}`)
			fmt.Println(`{"http_status":204}`)
			os.Exit(0)
		case "sleep":
			os.WriteFile(os.Args[3]+".pid", []byte(fmt.Sprint(os.Getpid())), 0600)
			time.Sleep(time.Minute)
		}
		fmt.Println(`{"http_status":204}`)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

type fixtureProvider struct {
	binding               ApprovedBinding
	resolves, credentials atomic.Int32
	purpose               string
}

func (p *fixtureProvider) Resolve(context.Context) (ApprovedBinding, error) {
	p.resolves.Add(1)
	return p.binding, nil
}
func (p *fixtureProvider) Credential(_ context.Context, purpose string) ([]byte, error) {
	p.credentials.Add(1)
	p.purpose = purpose
	return []byte("fixture-secret"), nil
}
func sum(raw []byte) string { s := sha256.Sum256(raw); return hex.EncodeToString(s[:]) }
func processFixture(t *testing.T, mode string) *fixtureProvider {
	t.Helper()
	dir := t.TempDir()
	venv := filepath.Join(dir, "venv")
	os.MkdirAll(filepath.Join(venv, "bin"), 0700)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(exe, filepath.Join(venv, "bin", "python")); err != nil {
		t.Fatal(err)
	}
	lock := []byte("fixture pinned lock\n")
	lockPath := filepath.Join(dir, "lock")
	os.WriteFile(lockPath, lock, 0600)
	os.WriteFile(filepath.Join(venv, ".vrooli-pyenv-lock-hash"), []byte(sum(lock)), 0600)
	entry := filepath.Join(dir, "ipc_entry.py")
	modules := make(map[string]string)
	for _, name := range moduleNames {
		raw := []byte("fixture module")
		if name == "ipc_entry.py" {
			raw = []byte(mode)
		}
		os.WriteFile(filepath.Join(dir, name), raw, 0600)
		modules[name] = sum(raw)
	}
	os.WriteFile(entry, []byte(mode), 0600)
	config := json.RawMessage(`{"enabled":true,"export":{"enabled":false}}`)
	return &fixtureProvider{binding: ApprovedBinding{Runtime: pyenv.Spec{VenvDir: venv, LockFile: lockPath, BasePython: "3.12"}, LockSHA256: sum(lock), EntryPath: entry, ModuleSHA256: modules, EntrySHA256: sum([]byte(mode)), Config: config, ConfigSHA256: sum(config), NotBefore: time.Now().Add(-time.Minute), ExpiresAt: time.Now().Add(time.Minute)}}
}
func TestProcessDisabledNoIO(t *testing.T) {
	p := &fixtureProvider{}
	r := NewProcessRunner(false, p)
	if _, err := r.Run(context.Background(), Request{Operation: "tick"}); err != ErrDisabled {
		t.Fatal(err)
	}
	if p.resolves.Load() != 0 || p.credentials.Load() != 0 {
		t.Fatal("provider touched")
	}
}
func TestProcessOneShotPrivateIPC(t *testing.T) {
	for _, mode := range []string{"idle", "secret"} {
		t.Run(mode, func(t *testing.T) {
			p := processFixture(t, mode)
			r := NewProcessRunner(true, p)
			op := "tick"
			if mode == "secret" {
				op = "accept_native"
			}
			out, err := r.Run(context.Background(), Request{Operation: op})
			if err != nil || out.HTTPStatus != 204 {
				t.Fatalf("%v %+v", err, out)
			}
			want := int32(0)
			if mode == "secret" {
				want = 1
			}
			if p.credentials.Load() != want {
				t.Fatal("wrong lazy credential calls")
			}
		})
	}
}
func TestProcessRejectsMalformedAndOversizeOutput(t *testing.T) {
	for _, mode := range []string{"malformed", "oversize", "stderr", "extra", "scope"} {
		t.Run(mode, func(t *testing.T) {
			p := processFixture(t, mode)
			_, err := NewProcessRunner(true, p).Run(context.Background(), Request{Operation: "tick"})
			if err != ErrProcess {
				t.Fatalf("%v", err)
			}
			if p.credentials.Load() != 0 {
				t.Fatal("unauthorized credential read")
			}
		})
	}
}
func TestProcessRejectsExpiredAndChangedBindings(t *testing.T) {
	for _, mode := range []string{"expired", "config", "entry", "lock", "relative"} {
		t.Run(mode, func(t *testing.T) {
			p := processFixture(t, "idle")
			switch mode {
			case "expired":
				p.binding.ExpiresAt = time.Now().Add(-time.Second)
			case "config":
				p.binding.ConfigSHA256 = strings.Repeat("0", 64)
			case "entry":
				p.binding.EntrySHA256 = strings.Repeat("0", 64)
			case "lock":
				p.binding.LockSHA256 = strings.Repeat("0", 64)
			case "relative":
				p.binding.Runtime.VenvDir = "relative"
			}
			if _, err := NewProcessRunner(true, p).Run(context.Background(), Request{Operation: "tick"}); err != ErrProcess {
				t.Fatal(err)
			}
		})
	}
}
func TestProcessCancellationReapsChild(t *testing.T) {
	p := processFixture(t, "sleep")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := NewProcessRunner(true, p).Run(ctx, Request{Operation: "tick"}); done <- err }()
	var pid int
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(p.binding.EntryPath + ".pid")
		if err == nil {
			fmt.Sscan(string(raw), &pid)
			break
		}
		time.Sleep(time.Millisecond)
	}
	if pid == 0 {
		cancel()
		<-done
		t.Fatal("child never started")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("canceled child succeeded")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("child not reaped")
	}
	if err := syscall.Kill(pid, 0); err != syscall.ESRCH {
		t.Fatalf("child remains: %v", err)
	}
}
func TestLifecycleDisabledAndLoopbackGate(t *testing.T) {
	mux := http.NewServeMux()
	l, err := Register(mux, Config{})
	if err != nil {
		t.Fatal(err)
	}
	l.Start(context.Background())
	if l.done != nil {
		t.Fatal("disabled goroutine")
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, request("x"))
	if w.Code != 404 {
		t.Fatal(w.Code)
	}
	if err = l.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	mux = http.NewServeMux()
	l, err = Register(mux, Config{Enabled: true, Runner: runnerFunc(func(context.Context, Request) (Result, error) { t.Fatal("remote runner"); return Result{}, nil }), Interval: time.Hour, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	req := request("x")
	req.RemoteAddr = "192.0.2.1:9999"
	req.Header.Set("X-Forwarded-For", "127.0.0.1")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatal(w.Code)
	}
	l.Start(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err = l.Stop(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestModuleClosureRefusesChangesBeforeCredentials(t *testing.T) {
	for _, mode := range []string{"missing", "unknown", "changed", "symlink", "directory", "oversize"} {
		t.Run(mode, func(t *testing.T) {
			p := processFixture(t, "secret")
			path := filepath.Join(filepath.Dir(p.binding.EntryPath), "ipc.py")
			switch mode {
			case "missing":
				delete(p.binding.ModuleSHA256, "ipc.py")
			case "unknown":
				p.binding.ModuleSHA256["unreviewed.py"] = sum(nil)
			case "changed":
				os.WriteFile(path, []byte("changed"), 0600)
			case "symlink":
				os.Remove(path)
				os.Symlink(p.binding.EntryPath, path)
			case "directory":
				os.Remove(path)
				os.Mkdir(path, 0700)
			case "oversize":
				os.WriteFile(path, make([]byte, (1<<20)+1), 0600)
			}
			if _, err := NewProcessRunner(true, p).Run(context.Background(), Request{Operation: "accept_native"}); err != ErrProcess {
				t.Fatal(err)
			}
			if p.credentials.Load() != 0 {
				t.Fatal("unreviewed module received credentials")
			}
		})
	}
}
func TestLifecycleFailureObservableBeforeShutdown(t *testing.T) {
	observed := make(chan error, 1)
	mux := http.NewServeMux()
	l, err := Register(mux, Config{Enabled: true, Interval: time.Second, Timeout: time.Second, Runner: runnerFunc(func(context.Context, Request) (Result, error) { return Result{}, fmt.Errorf("secret diagnostic") }), OnFailure: func(err error) { observed <- err }})
	if err != nil {
		t.Fatal(err)
	}
	l.Start(context.Background())
	defer l.Stop(context.Background())
	select {
	case err := <-observed:
		if err.Error() != "local work bridge lifecycle failed" {
			t.Fatal("unscrubbed failure")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("failure not reported before shutdown")
	}
}
