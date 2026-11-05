package hyperframes

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func renderFixture(t *testing.T) (Request, string, Installation) {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "index.html"), []byte("<html>test</html>"), 0o600); err != nil {
		t.Fatal(err)
	}
	h, _ := fileHash(filepath.Join(source, "index.html"))
	req := Request{Version: 1, InputDir: source, Composition: ".", Inputs: []Input{{Path: "index.html", SHA256: h}}, TimeoutSeconds: 20, MemoryMiB: 1024}
	for name, p := range map[string]*Executable{"node": &req.Node, "browser": &req.Browser, "ffmpeg": &req.FFmpeg, "ffprobe": &req.FFprobe} {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, []byte(name), 0o700); err != nil {
			t.Fatal(err)
		}
		h, _ := fileHash(path)
		*p = Executable{Path: path, SHA256: h}
	}
	tool := filepath.Join(root, "tool")
	if err := os.Mkdir(tool, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tool, "entry.mjs"), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	tree, _ := TreeHash(tool)
	return req, tool, Installation{Version: Version, Closure: ClosureID(), Tree: tree, Entry: "entry.mjs"}
}

func successfulRunner(t *testing.T, req Request, calls *int) runner {
	t.Helper()
	return func(ctx context.Context, path string, args []string, dir string, env []string, w io.Writer, memory int) error {
		*calls++
		if err := ctx.Err(); err != nil {
			return err
		}
		if len(args) == 1 {
			if path == req.Node.Path {
				_, _ = io.WriteString(w, "v22.23.2\n")
			} else {
				_, _ = io.WriteString(w, "fixture 1\n")
			}
			return nil
		}
		if path == req.FFprobe.Path {
			_, _ = io.WriteString(w, `{"streams":[{"codec_type":"video","width":640,"height":480}],"format":{"duration":"2.0"}}`)
			return nil
		}
		if len(args) > 1 && args[1] == "render" {
			return os.WriteFile(args[len(args)-1], []byte("fixture-encoded-media"), 0o600)
		}
		if !strings.Contains(strings.Join(env, "\n"), "HYPERFRAMES_BROWSER_PATH="+req.Browser.Path) {
			t.Error("missing explicit browser")
		}
		return nil
	}
}

func TestRenderPublishesVerifiedBundleAndReusesOnlyExactReceipt(t *testing.T) {
	req, tool, installation := renderFixture(t)
	out := filepath.Join(t.TempDir(), "render")
	calls := 0
	run := successfulRunner(t, req, &calls)
	r, err := execute(context.Background(), req, out, tool, installation, run)
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != "completed" || r.Reused || calls != 8 || r.Profile != fixedRenderProfile() {
		t.Fatalf("unexpected result %+v calls=%d", r, calls)
	}
	if _, err = os.Stat(filepath.Join(out, "inputs")); !os.IsNotExist(err) {
		t.Fatal("temporary inputs retained in completed bundle")
	}
	r, err = execute(context.Background(), req, out, tool, installation, run)
	if err != nil || !r.Reused || calls != 8 {
		t.Fatalf("cache failed %+v %v calls %d", r, err, calls)
	}
	req.MemoryMiB = 2048
	if _, err = execute(context.Background(), req, out, tool, installation, run); err == nil {
		t.Fatal("changed render profile reused")
	}
	req.MemoryMiB = 1024
	if err = os.WriteFile(filepath.Join(out, "render.mp4"), []byte("corrupted"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = execute(context.Background(), req, out, tool, installation, run); err == nil {
		t.Fatal("corrupt output reused")
	}
}

func TestRenderResourceProfileChangesIdentityAndRefusesReceiptTampering(t *testing.T) {
	req, tool, installation := renderFixture(t)
	profile := fixedRenderProfile()
	key := renderKey(req, installation)
	legacy := profile
	legacy.ID = "strict-software-one-worker-png-no-cache-v1"
	legacy.TasksMax = 256
	if key == renderKeyWithProfile(req, installation, legacy) {
		t.Fatal("historical task profile reused")
	}
	for _, changed := range []RenderProfile{
		func() RenderProfile { p := profile; p.TasksMax = 256; return p }(),
		func() RenderProfile { p := profile; p.CaptureWorkers = 2; return p }(),
		func() RenderProfile { p := profile; p.TaskPressureSampleMS = 1000; return p }(),
	} {
		if key == renderKeyWithProfile(req, installation, changed) {
			t.Fatal("resource policy omitted from reuse key")
		}
	}
	out := filepath.Join(t.TempDir(), "render")
	calls := 0
	r, err := execute(context.Background(), req, out, tool, installation, successfulRunner(t, req, &calls))
	if err != nil {
		t.Fatal(err)
	}
	r.Profile.TasksMax = 256 // Keep matching key: receipt itself is inconsistent.
	data, _ := json.Marshal(r)
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readReusable(context.Background(), out, key); err == nil {
		t.Fatal("mismatched receipt profile accepted")
	}
}

func TestRenderFailureNeverPublishesFinal(t *testing.T) {
	req, tool, installation := renderFixture(t)
	out := filepath.Join(t.TempDir(), "render")
	calls := 0
	ok := successfulRunner(t, req, &calls)
	run := func(ctx context.Context, path string, args []string, dir string, env []string, w io.Writer, m int) error {
		if len(args) > 1 && args[1] == "render" {
			_ = os.WriteFile(args[len(args)-1], []byte("partial"), 0o600)
			return errors.New("encoder interrupted")
		}
		return ok(ctx, path, args, dir, env, w, m)
	}
	if _, err := execute(context.Background(), req, out, tool, installation, run); err == nil {
		t.Fatal("expected encoder failure")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("partial final bundle published")
	}
}

func TestRenderRejectsUndeclaredAndSymlinkInputs(t *testing.T) {
	req, _, _ := renderFixture(t)
	extra := filepath.Join(req.InputDir, "extra")
	_ = os.WriteFile(extra, []byte("extra"), 0o600)
	if err := verifyInputs(req); err == nil {
		t.Fatal("undeclared input accepted")
	}
	_ = os.Remove(extra)
	_ = os.Symlink(req.Node.Path, extra)
	if err := verifyInputs(req); err == nil {
		t.Fatal("symlink input accepted")
	}
}

func TestRenderDetectsMidRenderSourceAndToolMutation(t *testing.T) {
	for _, target := range []string{"source", "tool", "executable"} {
		t.Run(target, func(t *testing.T) {
			req, tool, installation := renderFixture(t)
			calls := 0
			ok := successfulRunner(t, req, &calls)
			run := func(ctx context.Context, path string, args []string, dir string, env []string, w io.Writer, m int) error {
				err := ok(ctx, path, args, dir, env, w, m)
				if path == req.FFprobe.Path && len(args) > 1 {
					file := filepath.Join(req.InputDir, "index.html")
					if target == "tool" {
						file = filepath.Join(tool, "entry.mjs")
					}
					if target == "executable" {
						file = req.Browser.Path
					}
					_ = os.WriteFile(file, []byte("changed"), 0o700)
				}
				return err
			}
			out := filepath.Join(t.TempDir(), "render")
			if _, err := execute(context.Background(), req, out, tool, installation, run); err == nil {
				t.Fatal("changed inputs published")
			}
			if _, err := os.Stat(out); !os.IsNotExist(err) {
				t.Fatal("final exists")
			}
		})
	}
}

func TestRenderRequestAndProbeValidation(t *testing.T) {
	req, _, _ := renderFixture(t)
	if err := validateRequest(req); err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{"v20.9.1", "v22.11.9", "bad"} {
		if supportedNode(v) {
			t.Fatalf("unsupported node %s", v)
		}
	}
	for _, v := range []string{"v22.12.0", "v24.1.0"} {
		if !supportedNode(v) {
			t.Fatalf("supported node refused %s", v)
		}
	}
	for _, p := range []string{"../escape", "/absolute", "a/../b", "bad\\path"} {
		if safeRelative(p, false) {
			t.Fatalf("unsafe path accepted %s", p)
		}
	}
	if err := validateProbe([]byte(`{"streams":[],"format":{"duration":"2"}}`)); err == nil {
		t.Fatal("missing video accepted")
	}
	if err := validateProbe([]byte(`{"streams":[{"codec_type":"video","width":640,"height":480}],"format":{"duration":"nan"}}`)); err == nil {
		t.Fatal("nonfinite duration accepted")
	}
}

func TestManagedInstallationRequiresReceiptAndStableBytes(t *testing.T) {
	_, dir, _ := renderFixture(t)
	if _, err := VerifyInstallation(dir); err == nil {
		t.Fatal("unwitnessed tool accepted")
	}
	entry := "node_modules/hyperframes/bin/hyperframes.mjs"
	_ = os.MkdirAll(filepath.Dir(filepath.Join(dir, entry)), 0o700)
	_ = os.WriteFile(filepath.Join(dir, entry), []byte("entry"), 0o600)
	tree, _ := TreeHash(dir)
	b, _ := json.Marshal(Installation{Version: Version, Closure: ClosureID(), Tree: tree, Entry: entry, OS: runtime.GOOS, Arch: runtime.GOARCH})
	_ = os.WriteFile(filepath.Join(dir, "installation.json"), b, 0o600)
	if _, err := VerifyInstallation(dir); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, entry), []byte("mutated"), 0o600)
	if _, err := VerifyInstallation(dir); err == nil {
		t.Fatal("modified tool accepted")
	}
}

func TestToolTreeRejectsEscapingSymlink(t *testing.T) {
	dir := t.TempDir()
	if err := os.Symlink("/bin/sh", filepath.Join(dir, "escape")); err != nil {
		t.Fatal(err)
	}
	if _, err := TreeHash(dir); err == nil {
		t.Fatal("escaping link accepted")
	}
}

func TestTemporaryBudgetRejectsOversizeWithoutReadingMedia(t *testing.T) {
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "large"))
	if err != nil {
		t.Fatal(err)
	}
	if err = f.Truncate(4<<30 + 1); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if err = checkDiskBudget(dir); err == nil {
		t.Fatal("oversized temporary output accepted")
	}
}

func TestCancellationStopsAndJoinsOwnedScope(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	exited := make(chan struct{})
	stopped := false
	go func() { time.Sleep(time.Millisecond); cancel() }()
	err := waitOwned(ctx, func() error { <-exited; return errors.New("stopped") }, func(context.Context) error { stopped = true; close(exited); return nil }, func() ([]int, error) { return nil, nil })
	if !errors.Is(err, context.Canceled) || !stopped {
		t.Fatalf("owned tree was not cancelled and joined: %v stopped=%v", err, stopped)
	}
}

func TestSuccessfulRootStillDrainsDescendants(t *testing.T) {
	stopped := false
	err := waitOwned(context.Background(), func() error { return nil }, func(context.Context) error { stopped = true; return nil }, func() ([]int, error) {
		if !stopped {
			return []int{42}, nil
		}
		return nil, nil
	})
	if err != nil || !stopped {
		t.Fatalf("residual descendants ignored %v", err)
	}
}

func TestUnknownCleanupCannotBecomeSuccess(t *testing.T) {
	err := waitOwned(context.Background(), func() error { return nil }, func(context.Context) error { t.Fatal("unverified identity must not be stopped"); return nil }, func() ([]int, error) { return nil, errors.New("scope identity unavailable") })
	if err == nil {
		t.Fatal("unknown cleanup reported success")
	}
}

func TestRejectedRequestReturnsFailureReceipt(t *testing.T) {
	r, err := Execute(context.Background(), filepath.Join(t.TempDir(), "missing.json"), filepath.Join(t.TempDir(), "out"))
	if err == nil || r.Status != "failed" || r.Error == "" {
		t.Fatalf("rejected request has success-shaped receipt %+v %v", r, err)
	}
}

func TestCancelledIdentityAndSnapshotOperationsStopBeforeIO(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	missing := filepath.Join(t.TempDir(), "missing")
	for name, run := range map[string]func() error{
		"file hash":             func() error { _, e := fileHashContext(ctx, missing); return e },
		"tree hash":             func() error { _, e := TreeHashContext(ctx, missing); return e },
		"installation":          func() error { _, e := VerifyInstallationContext(ctx, missing); return e },
		"selected installation": func() error { _, _, e := SelectedInstallationContext(ctx); return e },
		"snapshot copy": func() error {
			return copyInputContext(ctx, missing, filepath.Join(t.TempDir(), "out"), strings.Repeat("0", 64))
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := run(); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelled operation reached filesystem or ignored cancellation: %v", err)
			}
		})
	}
}

func TestInvocationDeadlineIncludesWorkBeforeRenderer(t *testing.T) {
	req, tool, installation := renderFixture(t)
	req.TimeoutSeconds = 1
	ctx := context.WithValue(context.Background(), invocationStartKey{}, time.Now().Add(-2*time.Second))
	called := false
	_, err := execute(ctx, req, filepath.Join(t.TempDir(), "out"), tool, installation, func(context.Context, string, []string, string, []string, io.Writer, int) error {
		called = true
		return nil
	})
	if !errors.Is(err, context.DeadlineExceeded) || called {
		t.Fatalf("expired invocation reached renderer: %v called=%t", err, called)
	}
}

type cancellingReader struct {
	cancel context.CancelFunc
	reads  int
}

func (r *cancellingReader) Read(p []byte) (int, error) {
	r.reads++
	r.cancel()
	return copy(p, []byte("read before cancellation")), nil
}

func TestIdentityReaderObservesCancellationDuringRead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := &cancellingReader{cancel: cancel}
	_, err := io.Copy(io.Discard, contextReader{ctx: ctx, reader: r})
	if !errors.Is(err, context.Canceled) || r.reads != 1 {
		t.Fatalf("read continued after cancellation: %v reads=%d", err, r.reads)
	}
}
