package protogen

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
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
	if err := generator.publish(stageGen, []string{"alpha"}, false, true); err != nil {
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

// TestCommonFanOutMatchesExpectedFleetClosure pins the fan-out RULE against the
// live repository: selecting a schema must select exactly those scenarios whose
// input closure contains it, no more and no less.
//
// It deliberately asserts a relational invariant rather than a scenario count.
// A count pinned to the fleet's size at authoring time fails every time an
// unrelated scenario starts or stops importing common, which teaches readers to
// re-baseline the number instead of investigating — and it would still pass if
// the fan-out selected the wrong scenarios, so long as it selected the right
// number of them.
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

	// Re-derive the expected set from each scenario's input closure. This shares
	// InputClosure with resolveScope, so it does not police the closure itself —
	// what it pins is the inversion on top: that scope is exactly the scenarios
	// whose closure contains the selected schema.
	commonPrefix := filepath.Join("schemas", "common") + string(filepath.Separator)
	want := make(map[string]bool)
	for _, scenario := range all {
		files, _, closureErr := genmanifest.InputClosure(genmanifest.Options{RepoRoot: root, ProtoRoot: protoRoot}, scenario)
		if closureErr != nil {
			t.Fatalf("resolve imports for %s: %v", scenario, closureErr)
		}
		for _, file := range files {
			if strings.HasPrefix(filepath.FromSlash(file), commonPrefix) {
				want[scenario] = true
				break
			}
		}
	}
	// The selected schema is always in its own scope, even if nothing imports it.
	want["common"] = true

	got := make(map[string]bool, len(scope))
	for _, scenario := range scope {
		got[scenario] = true
	}

	for scenario := range want {
		if !got[scenario] {
			t.Errorf("scenario %q imports a common schema but was not selected", scenario)
		}
	}
	for scenario := range got {
		if !want[scenario] {
			t.Errorf("scenario %q was selected but imports no common schema", scenario)
		}
	}

	// Guard the degenerate pass: an InputClosure that silently returned nothing
	// would make both loops above vacuous and the fan-out meaningless.
	if len(scope) < 2 {
		t.Fatalf("common fan-out = %v; a shared schema must reach more than itself", scope)
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
			if argumentAfter(args, "-o") == "" {
				return nil
			}
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

func TestGeneratePublishesSelectedArtifactAndRejectsSourceDrift(t *testing.T) {
	root := t.TempDir()
	protoRoot := filepath.Join(root, "packages", "proto")
	schemaPath := filepath.Join(protoRoot, "schemas", "demo", "v1", "demo.proto")
	if err := os.MkdirAll(filepath.Dir(schemaPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(schemaPath, []byte("syntax = \"proto3\";\npackage demo.v1;\nmessage Demo {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"buf.yaml", "buf.gen.yaml", "buf.lock"} {
		if err := os.WriteFile(filepath.Join(protoRoot, file), []byte("version: v2\n"), 0o644); err != nil {
			t.Fatal(err)
		}
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
	var drift atomic.Bool
	fakeTool := func(_ context.Context, _ string, name string, args ...string) error {
		if len(args) == 0 {
			return nil
		}
		switch args[0] {
		case "build":
			if output := argumentAfter(args, "-o"); output != "" {
				return os.WriteFile(output, []byte("descriptor"), 0o644)
			}
			return nil
		case "generate":
			output := argumentAfter(args, "--output")
			path := filepath.Join(output, "gen", "go", "demo", "demo.go")
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte("package demo\n"), 0o644); err != nil {
				return err
			}
			if drift.Load() {
				return os.WriteFile(schemaPath, []byte("syntax = \"proto3\";\npackage demo.v1;\nmessage Demo { string changed = 1; }\n"), 0o644)
			}
			return nil
		default:
			_ = name
			return nil
		}
	}
	config := Config{
		RepoRoot:         root,
		ProtoRoot:        protoRoot,
		LockPath:         filepath.Join(root, "generator.lock"),
		StageParent:      filepath.Join(root, "packages"),
		ArtifactRoot:     filepath.Join(t.TempDir(), "proto-artifacts"),
		PublishArtifacts: true,
		RunTool:          fakeTool,
		Logger:           io.Discard,
	}
	generator, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := generator.Generate(context.Background()); err != nil {
		t.Fatal(err)
	}
	store, err := NewArtifactStore(config.ArtifactRoot)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := store.Resolve(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	oldID := selected.ArtifactID
	selected.Close()
	drift.Store(true)
	if err := generator.Generate(context.Background()); err == nil || !strings.Contains(err.Error(), "source changed during generation") {
		t.Fatalf("drifted generation error = %v, want source-drift rejection", err)
	}
	selected, err = store.Resolve(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer selected.Close()
	if selected.ArtifactID != oldID {
		t.Fatalf("selected artifact after source drift = %q, want %q", selected.ArtifactID, oldID)
	}
}

// Given an older selected artifact and unrelated edits in the compatibility
// tree, a scoped generation must publish its own outputs without restoring the
// rest of that snapshot. All tooling, schemas and artifact storage are isolated
// under t.TempDir; this regression never invokes a real generator subprocess.
func TestScopedArtifactPublishPreservesUnrelatedWorktreeOutputs(t *testing.T) {
	for _, tc := range []struct {
		artifacts, descriptorBuilt bool
	}{{true, true}, {true, false}, {false, true}, {false, false}} {
		t.Run(fmt.Sprintf("artifacts=%t/descriptor_built=%t", tc.artifacts, tc.descriptorBuilt), func(t *testing.T) {
			root := t.TempDir()
			protoRoot := filepath.Join(root, "packages", "proto")
			genRoot := filepath.Join(protoRoot, "gen")
			write := func(path, contents string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			for _, scenario := range []string{"alpha", "beta"} {
				write(filepath.Join(protoRoot, "schemas", scenario, "v1", "binding.proto"),
					fmt.Sprintf("syntax = \"proto3\";\npackage %s.v1;\nmessage Binding {}\n", scenario))
			}
			write(filepath.Join(protoRoot, "schemas", "consumer", "v1", "binding.proto"),
				"syntax = \"proto3\";\npackage consumer.v1;\nimport \"alpha/v1/binding.proto\";\nmessage Binding { alpha.v1.Binding alpha = 1; }\n")
			for _, file := range []string{"buf.yaml", "buf.gen.yaml", "buf.lock"} {
				write(filepath.Join(protoRoot, file), "version: v2\n")
			}
			for _, tool := range []string{"buf", "protoc-gen-go", "protoc-gen-connect-go", "protoc-gen-es", "protoc"} {
				write(filepath.Join(root, "internal", "tools", tool, "tool.json"), `{"version":"test"}`)
			}
			generation := 0
			fakeTool := func(_ context.Context, _ string, name string, args ...string) error {
				if name != "buf" || len(args) == 0 {
					return fmt.Errorf("unexpected tool invocation: %s %v", name, args)
				}
				switch args[0] {
				case "generate":
					generation++
					scenarios := []string{"alpha", "beta", "consumer"}
					if generation == 2 {
						var paths []string
						for i, arg := range args {
							if arg == "--path" && i+1 < len(args) {
								paths = append(paths, args[i+1])
							}
						}
						if !reflect.DeepEqual(paths, []string{"schemas/alpha", "schemas/consumer"}) {
							return fmt.Errorf("scoped tool paths = %v; want alpha and its reverse dependent consumer", paths)
						}
						scenarios = []string{"alpha", "consumer"}
						// An unrelated edit made after staging must also survive.
						write(filepath.Join(genRoot, "typescript", "beta", "binding.ts"), "concurrent beta edit")
					}
					staged := filepath.Join(argumentAfter(args, "--output"), "gen")
					for _, scenario := range scenarios {
						for _, rel := range []string{"go/" + scenario + "/binding.go", "python/" + scenario + "/binding.py", "typescript/" + scenario + "/binding.ts"} {
							write(filepath.Join(staged, rel), fmt.Sprintf("%s generation %d", scenario, generation))
						}
					}
					if generation == 1 {
						write(filepath.Join(staged, "go", "alpha", "retired.go"), "retired selected output")
					}
					write(filepath.Join(staged, "typescript", "buf", "shared.ts"), fmt.Sprintf("shared generation %d", generation))
					return nil
				case "build":
					output := argumentAfter(args, "-o")
					if output == "" { // Full-fleet schema-owner validation.
						return nil
					}
					if generation == 2 && !tc.descriptorBuilt {
						write(output, "partial failed descriptor")
						return fmt.Errorf("unrelated schema prevents the whole-tree descriptor build")
					}
					write(output, fmt.Sprintf("descriptor generation %d", generation))
					return nil
				default:
					return fmt.Errorf("unexpected buf command: %v", args)
				}
			}
			config := Config{RepoRoot: root, ProtoRoot: protoRoot,
				LockPath: filepath.Join(root, "generator.lock"), StageParent: filepath.Join(root, "packages"),
				ArtifactRoot: filepath.Join(root, "artifacts"), PublishArtifacts: tc.artifacts, RunTool: fakeTool}
			generator, err := New(config)
			if err != nil {
				t.Fatal(err)
			}
			if err := generator.Generate(context.Background()); err != nil {
				t.Fatal(err)
			}
			preserved := map[string]string{
				"go/beta/binding.go":         "unrelated beta edit",
				"go/beta/effort.pb.go":       "new unrelated type",
				"typescript/beta/binding.ts": "concurrent beta edit",
				"typescript/beta/effort.ts":  "new unrelated TS type",
				"manifests/beta.lock.json":   "unrelated beta manifest",
				"typescript/buf/other.ts":    "unmaterialized shared type",
				"typescript/package.json":    "{\"name\":\"fixture-types\"}",
			}
			for rel, contents := range preserved {
				write(filepath.Join(genRoot, rel), contents)
			}
			write(filepath.Join(genRoot, "typescript", "beta", "binding.ts"), "beta edit before staging")
			untouched := filepath.Join(genRoot, "go", "beta", "binding.go")
			before, err := os.Stat(untouched)
			if err != nil {
				t.Fatal(err)
			}
			deleted := filepath.Join(genRoot, "python", "beta", "binding.py")
			if err := os.Remove(deleted); err != nil {
				t.Fatal(err)
			}
			write(filepath.Join(genRoot, "descriptor", "image.binpb"), "newer unrelated descriptor")
			config.Scenarios = []string{"alpha"}
			generator, err = New(config)
			if err != nil {
				t.Fatal(err)
			}
			if err := generator.Generate(context.Background()); err != nil {
				t.Fatal(err)
			}
			for rel, want := range preserved {
				if got, err := os.ReadFile(filepath.Join(genRoot, rel)); err != nil || string(got) != want {
					t.Errorf("unrelated output %s = %q, err=%v; want %q", rel, got, err, want)
				}
			}
			if _, err := os.Stat(deleted); !os.IsNotExist(err) {
				t.Errorf("unrelated deleted output was restored: %v", err)
			}
			if after, err := os.Stat(untouched); err != nil || !os.SameFile(before, after) || !after.ModTime().Equal(before.ModTime()) {
				t.Errorf("unrelated output was replaced or touched: %v", err)
			}
			if _, err := os.Stat(filepath.Join(genRoot, "go", "alpha", "retired.go")); !os.IsNotExist(err) {
				t.Errorf("retired selected output was not pruned: %v", err)
			}
			for _, scenario := range []string{"alpha", "consumer"} {
				for _, rel := range []string{"go/" + scenario + "/binding.go", "python/" + scenario + "/binding.py", "typescript/" + scenario + "/binding.ts"} {
					if got, err := os.ReadFile(filepath.Join(genRoot, rel)); err != nil || string(got) != scenario+" generation 2" {
						t.Errorf("selected output %s = %q, err=%v; want %s generation 2", rel, got, err, scenario)
					}
				}
			}
			if got, err := os.ReadFile(filepath.Join(genRoot, "typescript", "buf", "shared.ts")); err != nil || string(got) != "shared generation 2" {
				t.Errorf("selected shared import = %q, err=%v; want shared generation 2", got, err)
			}
			wantDescriptor := "newer unrelated descriptor"
			if tc.descriptorBuilt {
				wantDescriptor = "descriptor generation 2"
			}
			if got, err := os.ReadFile(filepath.Join(genRoot, "descriptor", "image.binpb")); err != nil || string(got) != wantDescriptor {
				t.Errorf("descriptor = %q, err=%v; want %q", got, err, wantDescriptor)
			}
			if tc.artifacts {
				store, err := NewArtifactStore(config.ArtifactRoot)
				if err != nil {
					t.Fatal(err)
				}
				selected, err := store.Resolve(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				defer selected.Close()
				wantDescriptor = "descriptor generation 1"
				if tc.descriptorBuilt {
					wantDescriptor = "descriptor generation 2"
				}
				for rel, want := range map[string]string{
					"go/alpha/binding.go":    "alpha generation 2",
					"go/consumer/binding.go": "consumer generation 2",
					"go/beta/binding.go":     "beta generation 1",
					"descriptor/image.binpb": wantDescriptor,
				} {
					if got, err := os.ReadFile(filepath.Join(selected.GenRoot(), rel)); err != nil || string(got) != want {
						t.Errorf("immutable artifact output %s = %q, err=%v; want %q", rel, got, err, want)
					}
				}
				if _, err := os.Stat(filepath.Join(selected.GenRoot(), "go", "alpha", "retired.go")); !os.IsNotExist(err) {
					t.Errorf("retired selected output remained in the artifact: %v", err)
				}
			}
		})
	}
}

func TestBufCommandArgsUseExtendedBoundedTimeout(t *testing.T) {
	args := bufCommandArgs("generate", "--output", "/tmp/stage")
	want := []string{"generate", "--timeout", "15m0s", "--output", "/tmp/stage"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("buf command args = %#v, want %#v", args, want)
	}
}

func TestGenerateArgsScopesChangedSets(t *testing.T) {
	got := generateArgs([]string{"leaf", "dependent"}, "/tmp/stage", true)
	want := []string{"generate", "--timeout", "15m0s", "--output", "/tmp/stage", "--path", "schemas/leaf", "--path", "schemas/dependent"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("scoped generate args = %#v, want %#v", got, want)
	}
}

func TestGenerateArgsFullRunHasNoPaths(t *testing.T) {
	got := generateArgs([]string{"leaf"}, "/tmp/stage", false)
	want := []string{"generate", "--timeout", "15m0s", "--output", "/tmp/stage"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("full generate args = %#v, want %#v", got, want)
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

func TestStageRetainableKeepsFailuresAndDropsCancellations(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{name: "success", err: nil, want: false},
		{name: "real failure", err: fmt.Errorf("buf generate: plugin exploded"), want: true},
		{name: "cancelled", err: fmt.Errorf("buf generate: %w", context.Canceled), want: false},
		{name: "timed out", err: fmt.Errorf("buf generate: %w", context.DeadlineExceeded), want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := stageRetainable(tc.err); got != tc.want {
				t.Fatalf("stageRetainable(%v) = %t, want %t", tc.err, got, tc.want)
			}
		})
	}
}

// writeStageTree creates a staging tree with a file in it and backdates both to
// age. Directory modification time is what reaping keys on, so it is set last.
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

func stageTreeNames(t *testing.T, parent string) []string {
	t.Helper()
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names
}

func newStageParentGenerator(t *testing.T) (*Generator, string) {
	t.Helper()
	root := t.TempDir()
	stageParent := filepath.Join(root, "packages")
	protoRoot := filepath.Join(stageParent, "proto")
	if err := os.MkdirAll(protoRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	generator, err := New(Config{RepoRoot: root, ProtoRoot: protoRoot, StageParent: stageParent, Logger: io.Discard})
	if err != nil {
		t.Fatal(err)
	}
	return generator, stageParent
}

func TestReapStaleStagesRemovesOnlyExpiredStagingTrees(t *testing.T) {
	generator, stageParent := newStageParentGenerator(t)

	writeStageTree(t, stageParent, ".proto-gen-stage-expired", stageRetention+time.Hour)
	writeStageTree(t, stageParent, ".proto-gen-verify-expired", stageRetention+time.Hour)
	writeStageTree(t, stageParent, ".proto-descriptor-expired", stageRetention+time.Hour)
	writeStageTree(t, stageParent, ".proto-gen-stage-fresh", time.Minute)
	// An unrelated sibling proves reaping is keyed on the staging prefixes and
	// never treats the packages directory as disposable.
	writeStageTree(t, stageParent, "api-core", stageRetention+time.Hour)

	generator.reapStaleStages(time.Now())

	got := stageTreeNames(t, stageParent)
	want := []string{".proto-gen-stage-fresh", "api-core", "proto"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("surviving entries = %v, want %v", got, want)
	}
}

func TestCleanStagesRemovesEveryStagingTreeRegardlessOfAge(t *testing.T) {
	_, stageParent := newStageParentGenerator(t)

	writeStageTree(t, stageParent, ".proto-gen-stage-fresh", time.Minute)
	writeStageTree(t, stageParent, ".proto-gen-verify-fresh", time.Minute)
	writeStageTree(t, stageParent, "api-core", time.Minute)

	if err := CleanStages(stageParent); err != nil {
		t.Fatal(err)
	}

	got := stageTreeNames(t, stageParent)
	want := []string{"api-core", "proto"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("surviving entries = %v, want %v", got, want)
	}
}

func TestCleanStagesTargetsTheDirectoryTheGeneratorWritesTo(t *testing.T) {
	root := t.TempDir()
	protoRoot := filepath.Join(root, "packages", "proto")
	if err := os.MkdirAll(protoRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	generator, err := New(Config{RepoRoot: root, ProtoRoot: protoRoot, Logger: io.Discard})
	if err != nil {
		t.Fatal(err)
	}
	if got := DefaultStageParent(protoRoot); got != generator.cfg.StageParent {
		t.Fatalf("DefaultStageParent = %q, but generator stages into %q", got, generator.cfg.StageParent)
	}
}

// newStagingGenerator builds the minimal fixture Generate needs, with a tool
// runner whose failure mode the caller chooses.
func newStagingGenerator(t *testing.T, toolErr error) (*Generator, string) {
	t.Helper()
	root := t.TempDir()
	stageParent := filepath.Join(root, "packages")
	protoRoot := filepath.Join(stageParent, "proto")
	schemaRoot := filepath.Join(protoRoot, "schemas", "demo", "v1")
	if err := os.MkdirAll(schemaRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(schemaRoot, "demo.proto"), []byte("syntax = \"proto3\";\npackage demo.v1;\nmessage Demo {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fakeTool := func(_ context.Context, _ string, _ string, args ...string) error {
		if len(args) > 0 && args[0] == "generate" {
			// Emit into the staging tree first so the retained-vs-removed
			// assertion is about cleanup, not about an empty directory.
			output := argumentAfter(args, "--output")
			path := filepath.Join(output, "gen", "go", "demo", "demo.go")
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte("package demo"), 0o644); err != nil {
				return err
			}
			return toolErr
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

func TestGenerateRemovesStagingTreeWhenCancelled(t *testing.T) {
	generator, stageParent := newStagingGenerator(t, context.Canceled)

	if err := generator.Generate(context.Background()); err == nil {
		t.Fatal("expected generation to fail")
	}

	if got := stageTreeNames(t, stageParent); !reflect.DeepEqual(got, []string{"proto"}) {
		t.Fatalf("cancelled run left staging trees behind: %v", got)
	}
}

func TestGenerateRetainsStagingTreeOnRealFailure(t *testing.T) {
	generator, stageParent := newStagingGenerator(t, fmt.Errorf("plugin exploded"))

	if err := generator.Generate(context.Background()); err == nil {
		t.Fatal("expected generation to fail")
	}

	var retained int
	for _, name := range stageTreeNames(t, stageParent) {
		if hasStagePrefix(name) {
			retained++
		}
	}
	if retained != 1 {
		t.Fatalf("retained staging trees = %d, want 1 (a real failure stays inspectable)", retained)
	}
}

func TestGenerateReapsExpiredStagingTreesFromEarlierRuns(t *testing.T) {
	generator, stageParent := newStagingGenerator(t, fmt.Errorf("plugin exploded"))
	writeStageTree(t, stageParent, ".proto-gen-stage-ancient", stageRetention+time.Hour)

	if err := generator.Generate(context.Background()); err == nil {
		t.Fatal("expected generation to fail")
	}

	for _, name := range stageTreeNames(t, stageParent) {
		if name == ".proto-gen-stage-ancient" {
			t.Fatal("expired staging tree from an earlier run survived a later run")
		}
	}
}

// TestPublishDirectoryMergeKeepsUnstagedTargetFiles pins the shared-import
// contract. publishSharedImports runs only for a scoped generation, whose
// staging tree contains just the shared types the selected scenarios import.
// Pruning against that tree deleted every other scenario's shared types from
// gen/ — silently, since a scoped run never looks at them.
func TestPublishDirectoryMergeKeepsUnstagedTargetFiles(t *testing.T) {
	stage := t.TempDir()
	target := t.TempDir()

	// The scoped run materialized one shared type...
	if err := os.WriteFile(filepath.Join(stage, "types_pb.ts"), []byte("staged\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// ...while the repository already holds several others.
	survivors := []string{"attestation_pb.ts", "evidence_pb.ts", "metrics_pb.ts"}
	for _, name := range survivors {
		if err := os.WriteFile(filepath.Join(target, name), []byte("existing\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := publishDirectoryMerge(stage, target); err != nil {
		t.Fatal(err)
	}

	for _, name := range survivors {
		if _, err := os.Stat(filepath.Join(target, name)); err != nil {
			t.Fatalf("shared type %s must survive a scoped publish: %v", name, err)
		}
	}
	staged, err := os.ReadFile(filepath.Join(target, "types_pb.ts"))
	if err != nil {
		t.Fatalf("staged shared type must be published: %v", err)
	}
	if string(staged) != "staged\n" {
		t.Fatalf("published content = %q, want %q", staged, "staged\n")
	}
}

// The pruning publisher is still correct where the stage IS authoritative, so
// a full generation must continue to remove outputs whose sources are gone.
func TestPublishDirectoryStillPrunesWhenStageIsAuthoritative(t *testing.T) {
	stage := t.TempDir()
	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(stage, "kept_pb.ts"), []byte("kept\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"kept_pb.ts", "removed_pb.ts"} {
		if err := os.WriteFile(filepath.Join(target, name), []byte("old\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := publishDirectory(stage, target); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(target, "removed_pb.ts")); !os.IsNotExist(err) {
		t.Fatal("a full publish must remove an output the staging tree no longer produces")
	}
	if _, err := os.Stat(filepath.Join(target, "kept_pb.ts")); err != nil {
		t.Fatalf("a staged output must remain: %v", err)
	}
}

// A scoped run must never publish a partial descriptor image. The image is a whole-tree
// artifact that program-runtime's binding registry resolves every scenario against, so
// publishing a scoped build over it truncates the registry to the scoped packages and
// silently empties every other scenario's bindings.
func TestScopedPublishWithoutDescriptorLeavesTheCommittedImage(t *testing.T) {
	root := t.TempDir()
	protoRoot := filepath.Join(root, "packages", "proto")
	targetGen := filepath.Join(protoRoot, "gen")
	stageGen := filepath.Join(root, "stage", "gen")
	committed := filepath.Join(targetGen, "descriptor", "image.binpb")
	for _, path := range []string{
		filepath.Join(targetGen, "go", "alpha", "binding.go"),
		filepath.Join(targetGen, "manifests", "alpha.lock.json"),
		committed,
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("whole-tree"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{
		filepath.Join(stageGen, "go", "alpha", "binding.go"),
		filepath.Join(stageGen, "manifests", "alpha.lock.json"),
		filepath.Join(stageGen, "descriptor", "image.binpb"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("partial"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	generator := &Generator{cfg: Config{ProtoRoot: protoRoot, Logger: io.Discard}}
	if err := generator.publish(stageGen, []string{"alpha"}, false, false); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(committed); err != nil || string(got) != "whole-tree" {
		t.Fatalf("committed descriptor = %q, err=%v; want the whole-tree image left untouched", got, err)
	}
	if got, err := os.ReadFile(filepath.Join(targetGen, "go", "alpha", "binding.go")); err != nil || string(got) != "partial" {
		t.Fatalf("scoped output = %q, err=%v; want the scoped code still published", got, err)
	}
}
