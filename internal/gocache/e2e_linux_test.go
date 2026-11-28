//go:build linux

package gocache

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// buildTarget is a real cmd/go working against one store through the real
// helper, with the protocol activated in an isolated go env file.
type buildTarget struct {
	t       *testing.T
	goBin   string
	helper  string
	cache   string
	envFile string
	module  string
}

func newBuildTarget(t *testing.T) *buildTarget {
	t.Helper()
	if testing.Short() {
		t.Skip("compiles real Go packages through the GOCACHEPROG helper")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain not on PATH")
	}
	_, file, _, _ := runtime.Caller(0)
	repo := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	work := t.TempDir()
	target := &buildTarget{t: t, goBin: goBin, helper: filepath.Join(work, ProgramName), cache: filepath.Join(work, "cache"), envFile: filepath.Join(work, "go-env"), module: filepath.Join(work, "module")}
	build := exec.Command(goBin, "build", "-o", target.helper, "./cmd/vrooli-gocacheprog")
	build.Dir = repo
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build helper: %v\n%s", err, output)
	}
	if err := os.WriteFile(target.envFile, []byte("GOCACHEPROG="+target.helper+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Activation happened before any build in this test started.
	activated := time.Now().Add(-time.Hour)
	if err := os.Chtimes(target.envFile, activated, activated); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOENV", target.envFile)
	files := map[string]string{
		"go.mod":              "module example.com/proving\n\ngo 1.24\n",
		"wire/wire.go":        "package wire\n\nimport (\n\t\"encoding/json\"\n\t\"net/http\"\n)\n\nfunc Status(v any) (int, []byte) { b, _ := json.Marshal(v); return http.StatusOK, b }\n",
		"render/render.go":    "package render\n\nimport (\n\t\"strings\"\n\t\"text/template\"\n\n\t\"example.com/proving/wire\"\n)\n\nfunc Page(v any) string {\n\tvar b strings.Builder\n\t_, body := wire.Status(v)\n\ttemplate.Must(template.New(\"p\").Parse(\"{{.}}\")).Execute(&b, string(body))\n\treturn b.String()\n}\n",
		"secure/secure.go":    "package secure\n\nimport (\n\t\"crypto/tls\"\n\n\t\"example.com/proving/render\"\n)\n\nfunc Config() (*tls.Config, string) { return &tls.Config{MinVersion: tls.VersionTLS13}, render.Page(1) }\n",
		"cmd/app/main.go":     "package main\n\nimport (\n\t\"fmt\"\n\n\t\"example.com/proving/secure\"\n)\n\nfunc main() { _, page := secure.Config(); fmt.Println(page) }\n",
		"cmd/sleeper/main.go": "package main\n\nimport \"time\"\n\nfunc main() { time.Sleep(20 * time.Second) }\n",
	}
	for name, content := range files {
		path := filepath.Join(target.module, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return target
}

func (b *buildTarget) env(extra ...string) []string {
	env := make([]string, 0, len(os.Environ())+8)
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		switch key {
		case "GOCACHE", "GOENV", "GOCACHEPROG", "GOFLAGS", "GOTOOLCHAIN", "GOPROXY":
			continue
		}
		env = append(env, entry)
	}
	env = append(env, "GOCACHE="+b.cache, "GOENV="+b.envFile, "GOTOOLCHAIN=local", "GOFLAGS=-buildvcs=false", "GOPROXY=off")
	return append(env, extra...)
}

func (b *buildTarget) command(ctx context.Context, extra []string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, b.goBin, args...)
	cmd.Dir = b.module
	cmd.Env = b.env(extra...)
	return cmd
}

func (b *buildTarget) run(args ...string) (string, error) {
	output, err := b.command(context.Background(), nil, args...).CombinedOutput()
	return string(output), err
}

func (b *buildTarget) store() Store { return Store{Root: b.cache} }

// backdate makes every entry look long idle, the condition under which a
// time-based cleaner deleted outputs from running builds.
func (b *buildTarget) backdate(age time.Duration) {
	b.t.Helper()
	at := time.Now().Add(-age)
	_ = filepath.WalkDir(b.cache, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if _, _, ok := parseEntryName(entry.Name()); ok {
			_ = os.Chtimes(path, at, at)
		}
		return nil
	})
}

func TestHelperServesRealBuildsInterchangeablyWithTheNativeCache(t *testing.T) {
	target := newBuildTarget(t)
	if output, err := target.run("build", "./..."); err != nil {
		t.Fatalf("cold build through helper: %v\n%s", err, output)
	}
	// A warm build through the helper compiles nothing.
	if output, err := target.run("build", "-v", "./..."); err != nil || strings.TrimSpace(output) != "" {
		t.Fatalf("warm build recompiled %q: %v", output, err)
	}
	// cmd/go's native cache reads the helper's entries as its own.
	if output, err := target.run("build", "-v", "./..."); err != nil || strings.TrimSpace(output) != "" {
		t.Fatalf("second warm build: %q %v", output, err)
	}
	native := exec.Command(target.goBin, "build", "-v", "./...")
	native.Dir = target.module
	native.Env = append(target.env(), "GOENV=off")
	if output, err := native.CombinedOutput(); err != nil || strings.TrimSpace(string(output)) != "" {
		t.Fatalf("native cache could not reuse helper entries: %q %v", output, err)
	}
	if output, err := target.run("run", "./cmd/app"); err != nil || !strings.Contains(output, "1") {
		t.Fatalf("built program: %q %v", output, err)
	}
	journals, _ := filepath.Glob(filepath.Join(target.cache, StateDirName, sessionsDirName, "*"+pinJournalSuffix))
	if len(journals) != 0 {
		t.Fatalf("finished go commands left session journals: %v", journals)
	}
}

// The outcome under test: reclaim runs while real builds run, every entry
// looks idle, and no build fails.
func TestReclaimDuringRealBuildsNeverBreaksThem(t *testing.T) {
	target := newBuildTarget(t)
	if output, err := target.run("build", "./..."); err != nil {
		t.Fatalf("warm-up build: %v\n%s", err, output)
	}
	policy := Policy{BudgetBytes: 0, MinIdle: DefaultMinIdle}
	var (
		freed, pinnedSkips, applies int
		blocked                     []string
	)
	for round := range 4 {
		target.backdate(3 * time.Hour)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		var wg sync.WaitGroup
		failures := make(chan string, 2)
		builds := [][]string{{"build", "-o", filepath.Join(t.TempDir(), "app"), "./cmd/app"}, {"vet", "./..."}}
		for _, args := range builds {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if output, err := target.command(ctx, nil, args...).CombinedOutput(); err != nil {
					failures <- fmt.Sprintf("round %d: go %s: %v\n%s", round, strings.Join(args, " "), err, output)
				}
			}()
		}
		done := make(chan struct{})
		go func() { wg.Wait(); close(done) }()
	reclaim:
		for {
			select {
			case <-done:
				break reclaim
			default:
			}
			plan, err := MakePlan(PlanOptions{Store: target.store(), Program: target.helper, Policy: policy, Fresh: true})
			if err != nil {
				t.Fatalf("plan: %v", err)
			}
			if plan.BlockedReason != "" {
				blocked = append(blocked, plan.BlockedReason)
				continue
			}
			if plan.Token == "" {
				continue
			}
			receipt, err := Apply(ApplyOptions{Store: target.store(), Program: target.helper, Policy: policy, Token: plan.Token, MaxBytes: 4 << 20})
			if err != nil {
				t.Fatalf("apply: %v", err)
			}
			applies++
			freed += int(receipt.FreedBytes)
			pinnedSkips += receipt.SkippedPinned
		}
		cancel()
		close(failures)
		for failure := range failures {
			t.Error(failure)
		}
	}
	if output, err := target.run("run", "./cmd/app"); err != nil {
		t.Fatalf("build after reclaim rounds: %v\n%s", err, output)
	}
	t.Logf("applies=%d freed=%d pinned_skips=%d blocked_plans=%d", applies, freed, pinnedSkips, len(blocked))
	if freed == 0 || applies == 0 {
		t.Fatalf("reclaim never removed anything while builds ran (applies=%d, blocked=%v)", applies, blocked)
	}
	if pinnedSkips == 0 {
		t.Fatal("reclaim never met an output a live build held; the test did not exercise concurrency")
	}
}

func TestReclaimWaitsForAGoCommandOutsideTheProtocol(t *testing.T) {
	target := newBuildTarget(t)
	if output, err := target.run("build", "./..."); err != nil {
		t.Fatalf("warm-up build: %v\n%s", err, output)
	}
	target.backdate(3 * time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// GOENV=off: this go command uses the store natively, with no session.
	native := target.command(ctx, []string{"GOENV=off"}, "run", "./cmd/sleeper")
	if err := native.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { cancel(); _ = native.Wait() }()
	policy := Policy{BudgetBytes: 0, MinIdle: DefaultMinIdle}
	deadline := time.Now().Add(15 * time.Second)
	for {
		plan, err := MakePlan(PlanOptions{Store: target.store(), Program: target.helper, Policy: policy, Fresh: true})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(plan.BlockedReason, fmt.Sprintf("pid %d", native.Process.Pid)) && strings.Contains(plan.BlockedReason, "GOCACHEPROG unset") {
			if plan.Token != "" {
				t.Fatal("a blocked plan carried a token")
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("audit never reported the native go command (last plan %+v)", plan)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
