package installgateway

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/gomodreconcile"
)

// firstPartyRepo is a repository with one npm workspace package, a scenario
// surface that can link to it, and the Go modules the topology reports.
type firstPartyRepo struct{ root, surface string }

func newFirstPartyRepo(t *testing.T) firstPartyRepo {
	t.Helper()
	root := t.TempDir()
	write := func(rel, text string) {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("packages/widgets/package.json", `{"name":"@vrooli/widgets","version":"1.2.3"}`)
	write("packages/loose/package.json", `{"name":"@vrooli/loose","version":"^1.0.0"}`)
	write("packages/golib/go.mod", "module github.com/vrooli/golib\n")
	write("scenarios/demo/ui/package.json", `{"name":"demo-ui"}`)
	write("scenarios/demo/api/go.mod", "module github.com/vrooli/vrooli/scenarios/demo/api\n")
	previous := loadGoTopology
	loadGoTopology = func(string) (gomodreconcile.Topology, error) {
		return gomodreconcile.Topology{
			"github.com/vrooli/golib":                     filepath.Join(root, "packages", "golib"),
			"github.com/vrooli/vrooli/scenarios/demo/api": filepath.Join(root, "scenarios", "demo", "api"),
			"github.com/vrooli/vrooli":                    root,
		}, nil
	}
	t.Cleanup(func() { loadGoTopology = previous })
	return firstPartyRepo{root: root, surface: filepath.Join(root, "scenarios", "demo", "ui")}
}

func TestProveFirstPartyAdmitsOnlyWhatTheRepositoryLayoutProves(t *testing.T) {
	repo := newFirstPartyRepo(t)
	cases := []struct {
		name, ecosystem, pkg, version string
		want                          string // "" means not first-party
	}{
		{"an npm package linked from a scenario surface", "npm", "@vrooli/widgets", "file:../../../packages/widgets", "packages/widgets"},
		{"the ecosystem alias", "node", "@vrooli/widgets", "file:../../../packages/widgets", "packages/widgets"},
		{"a Go package inside a packages module", "go", "github.com/vrooli/golib/sub/pkg", "v0.0.0", "packages/golib"},
		{"the Go module itself", "go", "github.com/vrooli/golib", "v0.0.0", "packages/golib"},
		{"an npm package under another name", "npm", "@vrooli/other", "file:../../../packages/widgets", ""},
		{"an npm version that is not exact", "npm", "@vrooli/loose", "file:../../../packages/loose", ""},
		{"a registry version of a first-party name", "npm", "@vrooli/widgets", "1.2.3", ""},
		{"a mutable tarball", "npm", "@vrooli/widgets", "https://example.com/widgets.tgz", ""},
		{"a file: link outside packages", "npm", "demo-ui", "file:../ui", ""},
		{"a file: link escaping the repository", "npm", "@vrooli/widgets", "file:../../../../packages/widgets", ""},
		{"an absolute file: link", "npm", "@vrooli/widgets", "file:" + filepath.Join(repo.root, "packages", "widgets"), ""},
		{"a scenario Go module", "go", "github.com/vrooli/vrooli/scenarios/demo/api", "v0.0.0", ""},
		{"the repository root Go module", "go", "github.com/vrooli/vrooli/cmd/vrooli", "v0.0.0", ""},
		{"a third-party Go module", "go", "github.com/stretchr/testify", "v1.9.0", ""},
		{"another ecosystem", "pip", "widgets", "1.2.3", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ProveFirstParty(repo.root, repo.surface, tc.ecosystem, tc.pkg, tc.version)
			if ok != (tc.want != "") || got.Path != tc.want {
				t.Fatalf("ProveFirstParty = %+v, %t; want path %q", got, ok, tc.want)
			}
		})
	}
}

func TestProveFirstPartyRefusesALinkedPackageDirectory(t *testing.T) {
	repo := newFirstPartyRepo(t)
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "package.json"), []byte(`{"name":"@vrooli/linked","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(repo.root, "packages", "linked")); err != nil {
		t.Fatal(err)
	}
	if got, ok := ProveFirstParty(repo.root, repo.surface, "npm", "@vrooli/linked", "file:../../../packages/linked"); ok {
		t.Fatalf("a symlinked package directory is not first-party, got %+v", got)
	}
}
