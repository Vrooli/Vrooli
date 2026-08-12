package protogen

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/vrooli/packages/proto/genmanifest"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestProductionPipelineIsShellFree(t *testing.T) {
	production, err := os.ReadFile("generator.go")
	if err != nil {
		t.Fatal(err)
	}
	makefile, err := os.ReadFile(filepath.Join("..", "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range [][]byte{production, makefile} {
		text := string(source)
		for _, forbidden := range []string{"find ", "touch ", "rm -rf", "basename", "flock", "git diff"} {
			if strings.Contains(text, forbidden) {
				t.Errorf("production pipeline contains forbidden shell operation %q", forbidden)
			}
		}
	}
}

func TestGeneratorLockSerializesInProcessCalls(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "generator.lock")
	first, err := New(Config{RepoRoot: t.TempDir(), ProtoRoot: t.TempDir(), LockPath: lockPath, LockPoll: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	second, err := New(Config{RepoRoot: t.TempDir(), ProtoRoot: t.TempDir(), LockPath: lockPath, LockPoll: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	release, err := first.acquireLock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	acquired := make(chan struct{})
	go func() {
		releaseSecond, lockErr := second.acquireLock(context.Background())
		if lockErr != nil {
			return
		}
		close(acquired)
		releaseSecond()
	}()
	select {
	case <-acquired:
		t.Fatal("second generator acquired the in-process lock early")
	case <-time.After(10 * time.Millisecond):
	}
	release()
	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("second generator did not acquire the lock after release")
	}
}

func TestPublishFileLeavesNoOpModificationTimeUntouched(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.binpb")
	source := filepath.Join(dir, "stage.binpb")
	contents := []byte("same bytes")
	if err := os.WriteFile(target, contents, 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	if err := os.WriteFile(source, contents, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := publishFile(source, target); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if !after.ModTime().Equal(info.ModTime()) {
		t.Fatalf("no-op publish changed mtime from %v to %v", info.ModTime(), after.ModTime())
	}
}

func TestScopedPublishLeavesUntouchedScenarioOutputsUnchanged(t *testing.T) {
	root := t.TempDir()
	protoRoot := filepath.Join(root, "packages", "proto")
	targetGen := filepath.Join(protoRoot, "gen")
	stageGen := filepath.Join(root, "stage", "gen")
	for _, path := range []string{
		filepath.Join(targetGen, "go", "alpha", "binding.go"),
		filepath.Join(targetGen, "go", "beta", "binding.go"),
		filepath.Join(targetGen, "typescript", "buf", "shared.ts"),
		filepath.Join(targetGen, "manifests", "alpha.lock.json"),
		filepath.Join(targetGen, "descriptor", "image.binpb"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	untouched := filepath.Join(targetGen, "go", "beta", "binding.go")
	untouchedInfo, err := os.Stat(untouched)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		filepath.Join(stageGen, "go", "alpha", "binding.go"),
		filepath.Join(stageGen, "typescript", "buf", "shared.ts"),
		filepath.Join(stageGen, "manifests", "alpha.lock.json"),
		filepath.Join(stageGen, "descriptor", "image.binpb"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("new"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	generator := &Generator{cfg: Config{ProtoRoot: protoRoot, Logger: io.Discard}}
	if err := generator.publish(stageGen, []string{"alpha"}, false); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filepath.Join(targetGen, "go", "alpha", "binding.go")); err != nil || string(got) != "new" {
		t.Fatalf("selected output = %q, err=%v; want new", got, err)
	}
	if got, err := os.ReadFile(filepath.Join(targetGen, "typescript", "buf", "shared.ts")); err != nil || string(got) != "new" {
		t.Fatalf("shared output = %q, err=%v; want new", got, err)
	}
	afterInfo, err := os.Stat(untouched)
	if err != nil {
		t.Fatal(err)
	}
	if !afterInfo.ModTime().Equal(untouchedInfo.ModTime()) {
		t.Fatalf("untouched scenario mtime changed from %v to %v", untouchedInfo.ModTime(), afterInfo.ModTime())
	}
	if got, err := os.ReadFile(untouched); err != nil || string(got) != "old" {
		t.Fatalf("untouched scenario output = %q, err=%v; want old", got, err)
	}
}

func TestCompareTreesClassifiesCorruptedOutputWithRepositoryPath(t *testing.T) {
	dir := t.TempDir()
	staged := filepath.Join(dir, "staged")
	committed := filepath.Join(dir, "committed")
	for _, root := range []string{staged, committed} {
		if err := os.MkdirAll(filepath.Join(root, "go"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(staged, "go", "demo.go"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(committed, "go", "demo.go"), []byte("corrupted"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := compareTrees(staged, committed)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0] != "content-differs:gen/go/demo.go" {
		t.Fatalf("findings = %v, want content-differs:gen/go/demo.go", findings)
	}
}

func TestResolveScopeIncludesDependents(t *testing.T) {
	root := t.TempDir()
	protoRoot := filepath.Join(root, "packages", "proto")
	common := filepath.Join(protoRoot, "schemas", "common", "v1")
	dependent := filepath.Join(protoRoot, "schemas", "dependent", "v1")
	independent := filepath.Join(protoRoot, "schemas", "independent", "v1")
	for _, dir := range []string{common, dependent, independent} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeProto := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeProto(filepath.Join(common, "types.proto"), "syntax = \"proto3\";\npackage common.v1;\nmessage Shared {}\n")
	writeProto(filepath.Join(dependent, "service.proto"), "syntax = \"proto3\";\npackage dependent.v1;\nimport \"common/v1/types.proto\";\nmessage Service { common.v1.Shared shared = 1; }\n")
	writeProto(filepath.Join(independent, "service.proto"), "syntax = \"proto3\";\npackage independent.v1;\nmessage Service {}\n")

	generator, err := New(Config{RepoRoot: root, ProtoRoot: protoRoot, Scenarios: []string{"common"}})
	if err != nil {
		t.Fatal(err)
	}
	scope, err := generator.resolveScope([]string{"common", "dependent", "independent"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"common", "dependent"}
	if strings.Join(scope, ",") != strings.Join(want, ",") {
		t.Fatalf("scope = %v, want %v", scope, want)
	}
}

func TestCommonFanOutMatchesExpectedFleetClosure(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	protoRoot := filepath.Join(root, "packages", "proto")
	all, err := genmanifest.ScenarioNames(protoRoot)
	if err != nil {
		t.Fatal(err)
	}
	generator, err := New(Config{RepoRoot: root, ProtoRoot: protoRoot, Scenarios: []string{"common"}})
	if err != nil {
		t.Fatal(err)
	}
	scope, err := generator.resolveScope(all)
	if err != nil {
		t.Fatal(err)
	}
	if len(scope) != 27 {
		t.Fatalf("common fan-out = %d scenarios (%v), want 27", len(scope), scope)
	}
}

func TestScopedBufOutputMatchesCommittedSlices(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	protoRoot := filepath.Join(root, "packages", "proto")
	for _, scenario := range []string{"backdrop-studio", "measures", "browser-automation-studio"} {
		t.Run(scenario, func(t *testing.T) {
			stage := t.TempDir()
			cmd := exec.Command("buf", "generate", "--output", stage, "--path", filepath.ToSlash(filepath.Join("schemas", scenario)))
			cmd.Dir = protoRoot
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("scoped buf generate: %v\n%s", err, output)
			}
			if err := markPythonPackages(filepath.Join(stage, "gen")); err != nil {
				t.Fatal(err)
			}
			for _, rel := range genmanifest.ScenarioOutputDirs(scenario) {
				rel = strings.TrimPrefix(rel, "gen/")
				staged, err := treeDigests(filepath.Join(stage, "gen", filepath.FromSlash(rel)))
				if err != nil {
					t.Fatal(err)
				}
				committed, err := treeDigests(filepath.Join(protoRoot, "gen", filepath.FromSlash(rel)))
				if err != nil {
					t.Fatal(err)
				}
				if !equalDigests(staged, committed) {
					t.Fatalf("scoped output differs for %s", rel)
				}
			}
		})
	}
}

func TestConcurrentGeneratorsKeepPublishedArtifactsReadable(t *testing.T) {
	root := t.TempDir()
	protoRoot := filepath.Join(root, "packages", "proto")
	schemaRoot := filepath.Join(protoRoot, "schemas", "demo", "v1")
	if err := os.MkdirAll(schemaRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(schemaRoot, "demo.proto"), []byte("syntax = \"proto3\";\npackage demo.v1;\nmessage Demo {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(protoRoot, "buf.gen.yaml"), []byte("version: v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tool := range []string{"buf", "protoc-gen-go", "protoc-gen-connect-go", "protoc-gen-es", "protoc"} {
		path := filepath.Join(root, "internal", "tools", tool, "tool.json")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(`{"version":"test"}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(protoRoot, "gen", "typescript"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(protoRoot, "gen", "typescript", "package.json"), []byte(`{"name":"demo"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	var sequence atomic.Uint64
	fakeTool := func(_ context.Context, _ string, name string, args ...string) error {
		if name != "buf" {
			return fmt.Errorf("unexpected tool %q", name)
		}
		if len(args) > 0 && args[0] == "generate" {
			output := argumentAfter(args, "--output")
			generation := sequence.Add(1)
			for _, rel := range []string{
				filepath.Join("gen", "go", "demo", "demo.go"),
				filepath.Join("gen", "typescript", "demo", "demo.ts"),
				filepath.Join("gen", "typescript", "js", "demo", "demo.js"),
				filepath.Join("gen", "python", "demo", "demo.py"),
			} {
				path := filepath.Join(output, rel)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					return err
				}
				if err := os.WriteFile(path, []byte(fmt.Sprintf("generation-%d", generation)), 0o644); err != nil {
					return err
				}
			}
			return nil
		}
		if len(args) > 0 && args[0] == "build" {
			output := argumentAfter(args, "-o")
			generation := sequence.Load()
			set := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{Name: proto.String(fmt.Sprintf("demo/v1/%d.proto", generation)), Syntax: proto.String("proto3")}}}
			raw, err := proto.Marshal(set)
			if err != nil {
				return err
			}
			return os.WriteFile(output, raw, 0o644)
		}
		return nil
	}
	config := Config{RepoRoot: root, ProtoRoot: protoRoot, LockPath: filepath.Join(root, "generator.lock"), StageParent: filepath.Join(root, "packages"), Logger: io.Discard, RunTool: fakeTool}
	first, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Generate(context.Background()); err != nil {
		t.Fatal(err)
	}
	baseline := artifactCounts(t, filepath.Join(protoRoot, "gen"))

	stop := make(chan struct{})
	readerErr := make(chan error, 1)
	var reader sync.WaitGroup
	reader.Add(1)
	go func() {
		defer reader.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			counts := artifactCounts(t, filepath.Join(protoRoot, "gen"))
			for _, rootName := range []string{"go", "typescript", "python"} {
				if counts[rootName] < baseline[rootName] {
					readerErr <- fmt.Errorf("reader observed reduced %s file count: got %d want at least %d", rootName, counts[rootName], baseline[rootName])
					return
				}
			}
			raw, err := os.ReadFile(filepath.Join(protoRoot, "gen", "descriptor", "image.binpb"))
			if err != nil {
				readerErr <- fmt.Errorf("reader could not read descriptor: %w", err)
				return
			}
			set := &descriptorpb.FileDescriptorSet{}
			if err := proto.Unmarshal(raw, set); err != nil {
				readerErr <- fmt.Errorf("reader observed invalid descriptor: %w", err)
				return
			}
		}
	}()

	var writers sync.WaitGroup
	writerErr := make(chan error, 4)
	for i := 0; i < 4; i++ {
		writers.Add(1)
		go func() {
			defer writers.Done()
			generator, newErr := New(config)
			if newErr != nil {
				writerErr <- newErr
				return
			}
			writerErr <- generator.Generate(context.Background())
		}()
	}
	writers.Wait()
	close(stop)
	reader.Wait()
	select {
	case err := <-readerErr:
		t.Fatal(err)
	default:
	}
	for i := 0; i < 4; i++ {
		if err := <-writerErr; err != nil {
			t.Fatal(err)
		}
	}
}

func argumentAfter(args []string, flag string) string {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag {
			return args[i+1]
		}
	}
	return ""
}

func artifactCounts(t *testing.T, root string) map[string]int {
	t.Helper()
	counts := make(map[string]int)
	for _, name := range []string{"go", "typescript", "python"} {
		count := 0
		_ = filepath.WalkDir(filepath.Join(root, name), func(_ string, entry os.DirEntry, err error) error {
			if err == nil && !entry.IsDir() {
				count++
			}
			return nil
		})
		counts[name] = count
	}
	return counts
}
