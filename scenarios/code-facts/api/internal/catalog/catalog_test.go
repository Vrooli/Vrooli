package catalog

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestStableEvidenceIdentityIgnoresMutableLinePosition(t *testing.T) {
	before := StableContentAnchorID("go", "catalog.Repository", "func Activate(ctx context.Context, id string) error")
	after := StableContentAnchorID("go", "catalog.Repository", "func   Activate(ctx context.Context, id string) error")
	if before != after {
		t.Fatalf("whitespace/line shift changed content anchor: %q != %q", before, after)
	}
	if before == StableContentAnchorID("go", "catalog.Repository", "func Delete(ctx context.Context, id string) error") {
		t.Fatal("different declaration reused a content anchor")
	}
}

func TestCorpusClassificationGolden(t *testing.T) {
	tests := []struct {
		path       string
		prefix     string
		language   string
		role       Role
		searchable bool
	}{
		{"scenarios/code-facts/api/main.go", "package main", "go", RoleImplementation, true},
		{"packages/proto/schemas/code-facts/v1/facts.proto", "syntax = proto3;", "protobuf", RoleContract, true},
		{"packages/proto/gen/go/code-facts/facts.pb.go", "// Code generated. DO NOT EDIT.", "go", RoleGeneratedAlias, false},
		{"scenarios/code-facts/api/main_test.go", "package main", "go", RoleTest, false},
		{"scenarios/code-facts/api/testdata/report.json", "{}", "json", RoleFixture, false},
		{"scenarios/code-facts/docs/README.md", "# Docs", "markdown", RoleDocumentation, false},
		{"packages/proto/.verify-123/output.go", "package verify", "go", RoleTransient, false},
	}
	for _, test := range tests {
		got := Classify(test.path, []byte(test.prefix))
		if got.Language.Name != test.language || got.Role != test.role || got.Searchable != test.searchable {
			t.Errorf("Classify(%q) = language=%s role=%s searchable=%t", test.path, got.Language.Name, got.Role, got.Searchable)
		}
	}
}

type fakeInspector struct {
	snapshots map[string]FileSnapshot
}

func (f fakeInspector) Inspect(_ context.Context, path string) (FileSnapshot, error) {
	for suffix, snapshot := range f.snapshots {
		if strings.HasSuffix(path, suffix) {
			return snapshot, nil
		}
	}
	return FileSnapshot{}, errors.New("missing fixture snapshot")
}

type fakeStarter struct {
	output []byte
	dir    string
	name   string
	args   []string
}

func (s *fakeStarter) Start(_ context.Context, dir, name string, args ...string) (CommandProcess, error) {
	s.dir, s.name, s.args = dir, name, append([]string(nil), args...)
	return &fakeProcess{reader: bytes.NewReader(s.output)}, nil
}

type fakeProcess struct{ reader io.Reader }

func (p *fakeProcess) Stdout() io.Reader { return p.reader }
func (p *fakeProcess) Wait() error       { return nil }
func (p *fakeProcess) Close() error      { return nil }

func TestGitDiscovererStreamsGovernedRootsAndClassifies(t *testing.T) {
	starter := &fakeStarter{output: []byte("scenarios/demo/api/main.go\x00packages/proto/schemas/demo.proto\x00")}
	discoverer := GitDiscoverer{
		RepoRoot: "/repo", Roots: []string{"packages", "scenarios"}, Starter: starter,
		Inspector: fakeInspector{snapshots: map[string]FileSnapshot{
			"main.go":    {Size: 10, ModTime: time.Unix(1, 0), Hash: "sha256:go", Prefix: []byte("package main")},
			"demo.proto": {Size: 20, ModTime: time.Unix(2, 0), Hash: "sha256:proto", Prefix: []byte("syntax = proto3;")},
		}},
	}
	iterator, err := discoverer.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer iterator.Close()
	var files []SourceFile
	for {
		file, ok, err := iterator.Next(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			break
		}
		files = append(files, file)
	}
	if len(files) != 2 || files[0].Scope != "scenario:demo" || files[1].Role != RoleContract {
		t.Fatalf("unexpected discovery: %+v", files)
	}
	wantTail := []string{"--", "packages", "scenarios"}
	if len(starter.args) < len(wantTail) || !reflect.DeepEqual(starter.args[len(starter.args)-len(wantTail):], wantTail) {
		t.Fatalf("git roots not normalized: %v", starter.args)
	}
}

func TestLanguageRegistryDoesNotClaimUnsupportedGraphProof(t *testing.T) {
	python, ok := LanguageForPath("tool.py")
	if !ok || python.Capabilities.Graph || python.Capabilities.Proof {
		t.Fatalf("python capabilities overclaim graph/proof: %+v", python.Capabilities)
	}
	unknown, ok := LanguageForPath("opaque.xyz")
	if ok || !unknown.Capabilities.Catalog || unknown.Capabilities.Lexical {
		t.Fatalf("unknown language capabilities are not conservative: %+v", unknown.Capabilities)
	}
}

type sliceDiscoverer struct {
	files []SourceFile
	errAt int
}

func (d sliceDiscoverer) Open(context.Context) (FileIterator, error) {
	return &sliceIterator{files: d.files, errAt: d.errAt}, nil
}

type sliceIterator struct {
	files []SourceFile
	index int
	errAt int
}

func (i *sliceIterator) Next(context.Context) (SourceFile, bool, error) {
	if i.errAt > 0 && i.index == i.errAt {
		return SourceFile{}, false, errors.New("fixture discovery failed")
	}
	if i.index == len(i.files) {
		return SourceFile{}, false, nil
	}
	file := i.files[i.index]
	i.index++
	return file, true, nil
}

func (i *sliceIterator) Close() error { return nil }

func TestBuilderStreamsBoundedBatchesAndCompletesDigest(t *testing.T) {
	db := openCatalogDB(t)
	if err := Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	clock := fixedClock{value: time.Unix(200, 0)}
	repository := NewSQLiteRepository(db, clock)
	files := []SourceFile{fixtureSource("a.go"), fixtureSource("b.proto"), fixtureSource("c_test.go"), fixtureSource("d.md"), fixtureSource("e.ts")}
	result, err := (Builder{Repository: repository, Discoverer: sliceDiscoverer{files: files}, Clock: clock, BatchSize: 2}).Build(
		context.Background(), Generation{ID: "bounded", Policy: "corpus-v1", DescriptorDigest: "sha256:descriptor"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Files != 5 || result.MaxBatch != 2 || result.SourceDigest == "" {
		t.Fatalf("unexpected bounded build result: %+v", result)
	}
	var sourceDigest, descriptorDigest, state string
	if err := db.QueryRow(`SELECT source_digest, descriptor_digest, state FROM code_facts_generations WHERE id='bounded'`).Scan(&sourceDigest, &descriptorDigest, &state); err != nil {
		t.Fatal(err)
	}
	if sourceDigest != result.SourceDigest || descriptorDigest != "sha256:descriptor" || state != GenerationShadow {
		t.Fatalf("generation completion mismatch: source=%q descriptor=%q state=%q", sourceDigest, descriptorDigest, state)
	}
}

func TestBuilderMarksShadowFailedWhenDiscoveryStops(t *testing.T) {
	db := openCatalogDB(t)
	if err := Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	clock := fixedClock{value: time.Unix(200, 0)}
	repository := NewSQLiteRepository(db, clock)
	_, err := (Builder{Repository: repository, Discoverer: sliceDiscoverer{files: []SourceFile{fixtureSource("a.go"), fixtureSource("b.go")}, errAt: 1}, Clock: clock, BatchSize: 1}).Build(
		context.Background(), Generation{ID: "failed", Policy: "corpus-v1"},
	)
	if err == nil {
		t.Fatal("expected discovery failure")
	}
	var state, failure string
	if err := db.QueryRow(`SELECT state, failure FROM code_facts_generations WHERE id='failed'`).Scan(&state, &failure); err != nil {
		t.Fatal(err)
	}
	if state != GenerationFailed || !strings.Contains(failure, "fixture discovery failed") {
		t.Fatalf("failed generation not preserved honestly: state=%q failure=%q", state, failure)
	}
}

func TestRepositoryCorpusInventory(t *testing.T) {
	if os.Getenv("CODE_FACTS_REPO_INTEGRATION") != "1" {
		t.Skip("set CODE_FACTS_REPO_INTEGRATION=1 for the governed repository inventory")
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	iterator, err := (GitDiscoverer{RepoRoot: repoRoot, Starter: ExecCommandStarter{}, Inspector: OSFileInspector{}}).Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer iterator.Close()
	rootCounts := map[string]int{}
	roleCounts := map[Role]int{}
	var files int
	var bytes int64
	for {
		file, ok, err := iterator.Next(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			break
		}
		files++
		bytes += file.Size
		roleCounts[file.Role]++
		rootCounts[strings.Split(file.Path, "/")[0]]++
		if file.Role == RoleTransient {
			t.Fatalf("transient file entered governed Git corpus: %s", file.Path)
		}
	}
	for _, root := range []string{"scenarios", "packages", "resources", "internal", "cmd"} {
		if rootCounts[root] == 0 {
			t.Errorf("governed root %q has no catalog files", root)
		}
	}
	for _, role := range []Role{RoleImplementation, RoleContract, RoleGeneratedAlias, RoleTest, RoleFixture, RoleDocumentation} {
		if roleCounts[role] == 0 {
			t.Errorf("governed catalog has no %s role", role)
		}
	}
	t.Logf("governed inventory files=%d bytes=%d roots=%v roles=%v", files, bytes, rootCounts, SortedRoleCounts(roleCounts))
}
