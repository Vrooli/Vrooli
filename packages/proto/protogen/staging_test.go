package protogen

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

// stageHelperEnv makes this test binary act as a staging owner process: it
// creates one owned staging tree, reports its path and waits to be killed.
const stageHelperEnv = "PROTOGEN_STAGE_HELPER_PARENT"

func TestStageHelperProcess(t *testing.T) {
	parent := os.Getenv(stageHelperEnv)
	if parent == "" {
		t.Skip("helper process only")
	}
	staged, err := createStage(parent, generationStagePrefix)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(2)
	}
	fmt.Println(staged.Dir)
	time.Sleep(time.Minute)
	os.Exit(0)
}

// startStageOwner runs a separate process that owns a staging tree under parent.
func startStageOwner(t *testing.T, parent string) (*exec.Cmd, string) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestStageHelperProcess$")
	cmd.Env = append(os.Environ(), stageHelperEnv+"="+parent)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatalf("read helper stage path: %v", err)
	}
	path := strings.TrimSpace(line)
	if !strings.HasPrefix(filepath.Base(path), generationStagePrefix) {
		t.Fatalf("helper did not create a stage: %q", line)
	}
	return cmd, path
}

// TestSweepRemovesTreeOfKilledRunAndKeepsLiveRun pins the interrupted path: a
// run killed without unwinding leaves its tree, and the next sweep removes it
// because its owner lock died with it, while a concurrent live run's tree is
// untouched.
func TestSweepRemovesTreeOfKilledRunAndKeepsLiveRun(t *testing.T) {
	parent := t.TempDir()
	killed, killedStage := startStageOwner(t, parent)
	_, liveStage := startStageOwner(t, parent)

	if err := killed.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = killed.Wait()
	if _, err := os.Stat(killedStage); err != nil {
		t.Fatalf("a killed run cannot unwind, so its tree should remain until swept: %v", err)
	}

	removed, err := sweepStages(parent, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(removed, []string{killedStage}) {
		t.Fatalf("removed = %v, want only the killed run's tree %s", removed, killedStage)
	}
	if _, err := os.Stat(liveStage); err != nil {
		t.Fatalf("sweep deleted a live run's staging tree: %v", err)
	}
}

func TestSweepClassifiesOwnedOrphanedAndLocklessTrees(t *testing.T) {
	parent := t.TempDir()
	live, err := createStage(parent, verificationStagePrefix)
	if err != nil {
		t.Fatal(err)
	}
	defer live.Remove()
	// An owner that released its lock without removing the tree is
	// indistinguishable from a dead one.
	abandoned, err := createStage(parent, generationStagePrefix)
	if err != nil {
		t.Fatal(err)
	}
	abandoned.release()
	legacyOld := writeStageTree(t, parent, ".proto-gen-verify-legacy", unownedStageGrace+time.Minute)
	legacyFresh := writeStageTree(t, parent, ".proto-descriptor-fresh", time.Second)
	writeStageTree(t, parent, "api-core", unownedStageGrace+time.Hour)

	removed, err := sweepStages(parent, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(removed)
	want := []string{abandoned.Dir, legacyOld}
	sort.Strings(want)
	if !reflect.DeepEqual(removed, want) {
		t.Fatalf("removed = %v, want %v", removed, want)
	}
	got := entryNames(t, parent)
	wantNames := []string{filepath.Base(live.Dir), filepath.Base(legacyFresh), "api-core"}
	sort.Strings(wantNames)
	if !reflect.DeepEqual(got, wantNames) {
		t.Fatalf("surviving entries = %v, want %v", got, wantNames)
	}
}

func TestGenerateRemovesItsStagingTreeOnEveryExit(t *testing.T) {
	for _, tc := range []struct {
		name    string
		toolErr error
	}{
		{name: "success", toolErr: nil},
		{name: "real failure", toolErr: errors.New("plugin exploded")},
		{name: "cancelled", toolErr: context.Canceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			generator, stageParent := newStagingGenerator(t, tc.toolErr)
			err := generator.Generate(context.Background())
			if tc.toolErr != nil && err == nil {
				t.Fatal("expected generation to fail")
			}
			if got := stageNames(t, stageParent); len(got) != 0 {
				t.Fatalf("run left staging trees behind: %v", got)
			}
		})
	}
}

func TestVerifyRemovesItsStagingTreeOnMismatch(t *testing.T) {
	generator, stageParent := newStagingGenerator(t, nil)
	err := generator.Verify(context.Background())
	if err == nil || !strings.Contains(err.Error(), "generated artifacts differ") {
		t.Fatalf("Verify error = %v, want a mismatch against the empty committed tree", err)
	}
	if got := stageNames(t, stageParent); len(got) != 0 {
		t.Fatalf("failed verification left staging trees behind: %v", got)
	}
}

// TestGenerateRemovesStagingTreeWhenInterruptedMidTool cancels the context
// while buf is running, which is what the CLI's signal handler does on SIGINT
// or SIGTERM.
func TestGenerateRemovesStagingTreeWhenInterruptedMidTool(t *testing.T) {
	generator, stageParent := newStagingGenerator(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	running := make(chan struct{})
	generator.cfg.RunTool = func(ctx context.Context, _ string, _ string, args ...string) error {
		if len(args) > 0 && args[0] == "generate" {
			close(running)
			<-ctx.Done()
			return ctx.Err()
		}
		return nil
	}
	done := make(chan error, 1)
	go func() { done <- generator.Generate(ctx) }()
	<-running
	if got := stageNames(t, stageParent); len(got) != 1 {
		t.Fatalf("staging trees while running = %v, want exactly the live one", got)
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("Generate error = %v, want cancellation", err)
	}
	if got := stageNames(t, stageParent); len(got) != 0 {
		t.Fatalf("interrupted run left staging trees behind: %v", got)
	}
}

func TestGenerateStagesBesideArtifactStoreOutsideRepository(t *testing.T) {
	root := t.TempDir()
	protoRoot := filepath.Join(root, "packages", "proto")
	writeGeneratorFixture(t, root, protoRoot)
	artifactRoot := filepath.Join(t.TempDir(), "proto-artifacts")
	var outputs []string
	generator, err := New(Config{
		RepoRoot: root, ProtoRoot: protoRoot, ArtifactRoot: artifactRoot,
		LockPath: filepath.Join(t.TempDir(), "generator.lock"), Logger: io.Discard,
		RunTool: func(_ context.Context, _ string, _ string, args ...string) error {
			if output := argumentAfter(args, "--output"); output != "" {
				outputs = append(outputs, output)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := generator.Generate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(outputs) != 1 || filepath.Dir(outputs[0]) != StagingRoot(artifactRoot) {
		t.Fatalf("buf output = %v, want one stage directly under %s", outputs, StagingRoot(artifactRoot))
	}
	if got := entryNames(t, filepath.Join(root, "packages")); !reflect.DeepEqual(got, []string{"proto"}) {
		t.Fatalf("repository packages/ gained entries during generation: %v", got)
	}
}

func TestResolveStageParentFallsBackWhenRuntimeRootIsUnusable(t *testing.T) {
	publish := t.TempDir()
	runtime := filepath.Join(t.TempDir(), "staging")
	if got := resolveStageParent(runtime, publish, "fallback"); got != runtime {
		t.Fatalf("same-filesystem runtime root = %q, want %q", got, runtime)
	}
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := resolveStageParent(filepath.Join(blocker, "staging"), publish, "fallback"); got != "fallback" {
		t.Fatalf("uncreatable runtime root = %q, want fallback", got)
	}
}

func TestMaterializeCompatibilityViewLeavesOnlyTheView(t *testing.T) {
	store, snapshot := publishedDemoSnapshot(t)
	defer snapshot.Close()
	parent := filepath.Join(writableTempDir(t), "packages", "proto")
	target := filepath.Join(parent, "gen")
	for range 2 { // the second run exercises replacing an existing view
		if err := store.MaterializeCompatibilityView(context.Background(), snapshot, target); err != nil {
			t.Fatal(err)
		}
		if got := entryNames(t, parent); !reflect.DeepEqual(got, []string{"gen"}) {
			t.Fatalf("compatibility publication left entries beside the view: %v", got)
		}
		if got := stageNames(t, StagingRoot(store.Root)); len(got) != 0 {
			t.Fatalf("compatibility publication left staging trees: %v", got)
		}
	}
	if _, err := os.Stat(filepath.Join(target, stageOwnerLockName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("owner lock leaked into the published view: %v", err)
	}
}

func TestCleanStagesKeepsLiveTrees(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	live, err := createStage(first, generationStagePrefix)
	if err != nil {
		t.Fatal(err)
	}
	defer live.Remove()
	writeStageTree(t, first, ".proto-gen-verify-old", unownedStageGrace+time.Minute)
	writeStageTree(t, second, ".proto-compatibility-old", unownedStageGrace+time.Minute)

	if err := CleanStages(first, second, filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Fatal(err)
	}
	if got := stageNames(t, first); !reflect.DeepEqual(got, []string{filepath.Base(live.Dir)}) {
		t.Fatalf("first parent = %v, want only the live tree", got)
	}
	if got := stageNames(t, second); len(got) != 0 {
		t.Fatalf("second parent = %v, want empty", got)
	}
}

func TestDefaultStagingAndBackupClassificationKeepsPublishedAndAuthoredInputs(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{
		"packages/.proto-gen-verify-123/gen/go/types.pb.go",
		"packages/.proto-gen-stage-123/gen/go/types.pb.go",
		"packages/.proto-descriptor-123/image.binpb",
		"packages/proto/.proto-compatibility-123/gen/go/types.pb.go",
		"packages/proto/gen.previous",
		"packages/proto/gen.previous/go/types.pb.go",
	} {
		if !IsDefaultStagingOrBackupPath(root, filepath.Join(root, filepath.FromSlash(name))) {
			t.Errorf("owner work path was not classified: %s", name)
		}
	}
	for _, name := range []string{
		"packages/proto/gen/go/types.pb.go",
		"packages/.authored/types.go",
		"packages/authored/.proto-gen-verify-123/source.go",
		"internal/.proto-gen-verify-123/source.go",
		"packages/.proto-gen-verify/source.go",
		"packages/proto/gen.previous-authored/source.go",
		"packages/other/gen.previous/source.go",
		"packages/.proto-compatibility-123/source.go",
	} {
		if IsDefaultStagingOrBackupPath(root, filepath.Join(root, filepath.FromSlash(name))) {
			t.Errorf("published or authored path was hidden: %s", name)
		}
	}
}

// writeStageTree creates a lockless staging-shaped tree (as written before
// owner locks existed) and backdates it to age. Directory modification time is
// what the lockless grace keys on, so it is set last.
func writeStageTree(t *testing.T, parent, name string, age time.Duration) string {
	t.Helper()
	path := filepath.Join(parent, name)
	if err := os.MkdirAll(filepath.Join(path, "gen", "go"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "gen", "go", "demo.go"), []byte("package demo"), 0o644); err != nil {
		t.Fatal(err)
	}
	stamp := time.Now().Add(-age)
	if err := os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	return path
}

func entryNames(t *testing.T, parent string) []string {
	t.Helper()
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names
}

func stageNames(t *testing.T, parent string) []string {
	t.Helper()
	entries, err := os.ReadDir(parent)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		if hasStagePrefix(entry.Name()) {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names
}

// writeGeneratorFixture writes the smallest repository a fake-tool Generate or
// Verify can run against: one schema owner plus the toolchain inputs manifests
// fingerprint.
func writeGeneratorFixture(t *testing.T, root, protoRoot string) {
	t.Helper()
	files := map[string]string{
		filepath.Join(protoRoot, "schemas", "demo", "v1", "demo.proto"): "syntax = \"proto3\";\npackage demo.v1;\nmessage Demo {}\n",
		filepath.Join(protoRoot, "buf.gen.yaml"):                        "version: v2\n",
		filepath.Join(protoRoot, "protogen", "generator.go"):            "package protogen\n",
	}
	for _, tool := range []string{"buf", "protoc-gen-go", "protoc-gen-connect-go", "protoc-gen-es", "protoc"} {
		files[filepath.Join(root, "internal", "tools", tool, "tool.json")] = `{"version":"test"}`
	}
	for path, body := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// newStagingGenerator builds the minimal fixture Generate and Verify need,
// with a tool runner whose failure mode the caller chooses. Staging goes to a
// dedicated parent so assertions see only this run's trees.
func newStagingGenerator(t *testing.T, toolErr error) (*Generator, string) {
	t.Helper()
	root := t.TempDir()
	protoRoot := filepath.Join(root, "packages", "proto")
	writeGeneratorFixture(t, root, protoRoot)
	stageParent := filepath.Join(t.TempDir(), "staging")
	fakeTool := func(_ context.Context, _ string, _ string, args ...string) error {
		if len(args) > 0 && args[0] == "generate" {
			// Emit into the staging tree first so the assertion is about
			// cleanup, not about an empty directory.
			path := filepath.Join(argumentAfter(args, "--output"), "gen", "go", "demo", "demo.go")
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte("package demo"), 0o644); err != nil {
				return err
			}
		}
		return toolErr
	}
	generator, err := New(Config{
		RepoRoot:    root,
		ProtoRoot:   protoRoot,
		StageParent: stageParent,
		LockPath:    filepath.Join(root, "generator.lock"),
		Logger:      io.Discard,
		RunTool:     fakeTool,
	})
	if err != nil {
		t.Fatal(err)
	}
	return generator, stageParent
}

func publishedDemoSnapshot(t *testing.T) (*Store, *Snapshot) {
	t.Helper()
	store, err := NewArtifactStore(filepath.Join(writableTempDir(t), "proto"))
	if err != nil {
		t.Fatal(err)
	}
	candidate := testCandidate(t, writableTempDir(t), "selected", "selected")
	candidate.ValidationReceipt = &ArtifactValidationReceipt{Status: "passed", Checks: []string{"test"}}
	if _, err := store.Publish(context.Background(), candidate); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.Resolve(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return store, snapshot
}
