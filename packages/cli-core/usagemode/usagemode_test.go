package usagemode

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

func tp(t time.Time) *time.Time { return &t }

func mode(id string, scope Scope, start time.Time, expires *time.Time, runners ...string) Mode {
	m := Mode{ID: id, Scope: scope, StartedAt: start, ExpiresAt: expires, Until: Until{Kind: UntilTime}, Runners: map[string]RunnerSettings{}}
	for _, r := range runners {
		m.Runners[r] = RunnerSettings{SessionModel: "low-" + id}
	}
	return m
}

func useHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("VROOLI_HOME", home)
	// Harness defaults come from the test, never the developer's own config.
	t.Setenv("CODEX_HOME", t.TempDir())
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	return home
}

func TestPathUsesVrooliHome(t *testing.T) {
	home := useHome(t)
	if got, want := Path(), filepath.Join(home, "state", "vrooli", "usage-mode", "mode.json"); got != want {
		t.Fatalf("Path = %q, want %q", got, want)
	}
}

func TestReadFailsOpen(t *testing.T) {
	useHome(t)
	if got := Read(t0); got != nil {
		t.Fatalf("missing file: %v", got)
	}
	for name, body := range map[string]string{"corrupt": "{not json", "empty": "", "wrong schema": `{"schema_version":9,"modes":[{"id":"x","until":{"kind":"off"}}]}`} {
		if err := os.MkdirAll(filepath.Dir(Path()), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(Path(), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if got := Read(t0); got != nil {
			t.Fatalf("%s: %v", name, got)
		}
	}
}

func TestExpiryBoundaryAndOff(t *testing.T) {
	useHome(t)
	exp := t0.Add(time.Hour)
	off := mode("off", Scope{}, t0.Add(-time.Hour), nil, "codex")
	off.Until.Kind = UntilOff
	noExpiry := mode("noexp", Scope{}, t0.Add(-time.Hour), nil, "codex") // time kind without expiry is invalid
	future := mode("future", Scope{}, t0.Add(time.Minute), tp(exp), "codex")
	if err := Write(File{Modes: []Mode{mode("a", Scope{}, t0.Add(-time.Hour), tp(exp), "codex"), off, noExpiry, future}}); err != nil {
		t.Fatal(err)
	}
	ids := func(now time.Time) string {
		var out []string
		for _, m := range Read(now) {
			out = append(out, m.ID)
		}
		return strings.Join(out, ",")
	}
	if got := ids(exp.Add(-time.Nanosecond)); got != "a,off,future" {
		t.Fatalf("just before expiry: %s", got)
	}
	if got := ids(exp); got != "off" {
		t.Fatalf("at expiry the mode is over: %s", got)
	}
	if got := ids(t0.Add(-2 * time.Hour)); got != "" {
		t.Fatalf("before start: %s", got)
	}
}

func TestResolvePrecedence(t *testing.T) {
	exp := tp(t0.Add(time.Hour))
	early, late := t0.Add(-2*time.Hour), t0.Add(-time.Hour)
	modes := []Mode{
		mode("global", Scope{}, late, exp, "codex"),
		mode("runner-old", Scope{Runner: "codex"}, early, exp, "codex"),
		mode("runner-new", Scope{Runner: "codex"}, late, exp, "codex"),
		mode("role", Scope{Role: "code.default"}, early, exp, "codex"),
		mode("profile", Scope{Profile: "p1"}, early, exp, "codex"),
		mode("other-runner", Scope{Runner: "claude"}, late, exp, "claude"),
	}
	cases := []struct {
		q    Query
		want string
	}{
		{Query{Runner: "codex"}, "runner-new"},
		{Query{Runner: "codex", Role: "code.default"}, "role"},
		{Query{Runner: "codex", Role: "code.default", Profile: "p1"}, "profile"},
		{Query{Runner: "codex", Profile: "p2"}, "runner-new"},
		{Query{Runner: "claude-code"}, "other-runner"},
		{Query{Runner: "grok"}, ""},
	}
	for _, c := range cases {
		c.q.Now = t0
		got, ok := resolveFrom(modes, c.q)
		if (c.want == "") == ok || got.ID != c.want {
			t.Errorf("%+v: got %q ok=%v, want %q", c.q, got.ID, ok, c.want)
		}
	}
}

func TestExemptRoles(t *testing.T) {
	m := mode("m", Scope{}, t0.Add(-time.Hour), tp(t0.Add(time.Hour)), "codex")
	m.ExemptRoles = []string{"code.special"}
	for _, role := range []string{"code.fallback", "code.review", "judgment.supervision", "code.supervision", "code.special"} {
		if _, ok := resolveFrom([]Mode{m}, Query{Runner: "codex", Role: role, Now: t0}); ok {
			t.Errorf("%s should keep its model", role)
		}
	}
	if _, ok := resolveFrom([]Mode{m}, Query{Runner: "codex", Role: "code.default", Now: t0}); !ok {
		t.Error("ordinary role should be lowered")
	}
	// A mode scoped to an exempt role lists it explicitly.
	m.Scope.Role = "code.review"
	if _, ok := resolveFrom([]Mode{m}, Query{Runner: "codex", Role: "code.review", Now: t0}); !ok {
		t.Error("role-scoped mode should apply to its own role")
	}
}

func TestWriteAtomicPermissionsAndUnwritable(t *testing.T) {
	useHome(t)
	if err := Write(File{Modes: []Mode{mode("a", Scope{}, t0, tp(t0.Add(time.Hour)), "codex")}}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(Path())
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode file: %v %v", info, err)
	}
	entries, _ := os.ReadDir(filepath.Dir(Path()))
	if len(entries) != 1 {
		t.Fatalf("temp files left behind: %v", entries)
	}
	// Unwritable: the parent is a regular file.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(filepath.Join(blocker, "x", "mode.json"), File{}); err == nil {
		t.Fatal("write under a file should fail")
	}
	if got := ReadFile(filepath.Join(blocker, "x", "mode.json"), t0); got != nil {
		t.Fatalf("read under a file: %v", got)
	}
}

func TestConcurrentWritesLeaveAValidFile(t *testing.T) {
	useHome(t)
	var wg sync.WaitGroup
	stop := make(chan struct{})
	bad := make(chan string, 1)
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
			}
			if raw, err := os.ReadFile(Path()); err == nil {
				if !strings.HasSuffix(strings.TrimSpace(string(raw)), "}") {
					select {
					case bad <- string(raw):
					default:
					}
				}
			}
		}
	}()
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				id := string(rune('a' + i))
				if err := Write(File{Modes: []Mode{mode(id, Scope{}, t0, tp(t0.Add(time.Hour)), "codex")}}); err != nil {
					t.Error(err)
				}
			}
		}(i)
	}
	wg.Wait()
	close(stop)
	select {
	case raw := <-bad:
		t.Fatalf("reader saw a torn file: %q", raw)
	default:
	}
	if got := Read(t0); len(got) != 1 {
		t.Fatalf("final file: %v", got)
	}
}

func TestReadLatency(t *testing.T) {
	useHome(t)
	if err := Write(File{Modes: []Mode{mode("a", Scope{}, t0, tp(t0.Add(time.Hour)), "codex")}}); err != nil {
		t.Fatal(err)
	}
	Read(t0)
	const n = 200
	start := time.Now()
	for i := 0; i < n; i++ {
		Read(t0)
	}
	if avg := time.Since(start) / n; avg > time.Millisecond {
		t.Fatalf("typical read %v exceeds 1ms", avg)
	}
}

func TestUpdateSerialisesConcurrentWriters(t *testing.T) {
	useHome(t)
	const writers = 24
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := "um-w" + string(rune('a'+i))
			if err := Update(func(f *File) error {
				time.Sleep(2 * time.Millisecond) // widen the read-modify-write window
				f.Modes = append(f.Modes, mode(id, Scope{}, t0.Add(-time.Hour), tp(t0.Add(time.Hour)), "codex"))
				return nil
			}); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	if got := len(Read(t0)); got != writers {
		t.Fatalf("lost updates: %d of %d modes survived", got, writers)
	}
}

func TestUpdateKeepsFieldsThisVersionDoesNotKnow(t *testing.T) {
	useHome(t)
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	future := `{"schema_version":1,"future_top":{"a":1},"modes":[{"id":"um-x","kind":"learn","started_at":"2026-10-30T10:00:00Z",` +
		`"until":{"kind":"off"},"scope":{},"runners":{},"future_mode":[1,2],` +
		`"learn":{"split_percent":30,"runners":{"codex":{"session_model":"m","future_setting":true}},"future_learn":"keep"}}]}`
	if err := os.WriteFile(path, []byte(future), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Update(func(f *File) error {
		f.Modes = append(f.Modes, mode("um-y", Scope{}, t0.Add(-time.Hour), tp(t0.Add(time.Hour)), "codex"))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	for _, want := range []string{`"future_top"`, `"future_mode"`, `"future_learn"`, `"future_setting"`, `"split_percent": 30`, `"um-y"`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("rewrite dropped %s:\n%s", want, raw)
		}
	}
}

func TestUpdateErrorWritesNothingAndForeignSchemaIsRefused(t *testing.T) {
	useHome(t)
	if err := Update(func(f *File) error { f.Modes = append(f.Modes, Mode{ID: "um-1"}); return os.ErrInvalid }); err == nil {
		t.Fatal("callback error lost")
	}
	if _, err := os.Stat(Path()); err == nil {
		t.Fatal("a failed update wrote the file")
	}
	if err := os.MkdirAll(filepath.Dir(Path()), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Path(), []byte(`{"schema_version":2,"modes":[{"id":"newer"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Update(func(*File) error { return nil }); err == nil {
		t.Fatal("overwrote a file of a newer schema")
	}
	if raw, _ := os.ReadFile(Path()); !strings.Contains(string(raw), "newer") {
		t.Fatal("newer-schema data was destroyed")
	}
}

func TestReadFileIgnoresFifoAndHugeFiles(t *testing.T) {
	dir := t.TempDir()
	fifo := filepath.Join(dir, "fifo.json")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Skip("no fifo support: ", err)
	}
	done := make(chan []Mode, 1)
	go func() { done <- ReadFile(fifo, t0) }()
	select {
	case modes := <-done:
		if modes != nil {
			t.Fatalf("fifo produced modes: %v", modes)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("reading a FIFO blocked the launch")
	}
	huge := filepath.Join(dir, "huge.json")
	valid := `{"schema_version":1,"modes":[{"id":"um-1","until":{"kind":"off"},"started_at":"2026-10-30T10:00:00Z"}],"pad":"` +
		strings.Repeat("x", MaxFileBytes) + `"}`
	if err := os.WriteFile(huge, []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	if modes := ReadFile(huge, t0); modes != nil {
		t.Fatal("a file over the size cap was parsed")
	}
}
