package lifecycle

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/scenario"
)

// fakeEngagementResolver is the lifecycle's test double for the engagement seam.
type fakeEngagementResolver struct {
	info    EngagementInfo
	engaged bool
	err     error
	asked   []string
}

var _ EngagementResolver = (*fakeEngagementResolver)(nil)

func (f *fakeEngagementResolver) Engagement(scenarioName string) (EngagementInfo, bool, error) {
	f.asked = append(f.asked, scenarioName)
	return f.info, f.engaged, f.err
}

func newSourceDirItem(variant string) scenario.Scenario {
	return scenario.Scenario{Slug: "demo", Path: "/repo/scenarios/demo", Variant: variant}
}

func TestEffectiveSourceDir(t *testing.T) {
	const copyDir = "/cache/vrooli/demo/baseline-x/restore-point"

	tests := []struct {
		name     string
		variant  string
		resolver EngagementResolver
		want     string
		wantErr  bool
	}{
		{
			name:     "nil resolver runs from working tree",
			variant:  "",
			resolver: nil,
			want:     "/repo/scenarios/demo",
		},
		{
			name:     "live with no engagement runs from working tree",
			variant:  "",
			resolver: &fakeEngagementResolver{engaged: false},
			want:     "/repo/scenarios/demo",
		},
		{
			name:    "live with open engagement runs from restore-point copy",
			variant: "",
			resolver: &fakeEngagementResolver{
				engaged: true,
				info:    EngagementInfo{RestorePointDir: copyDir, Slug: "x"},
			},
			want: copyDir,
		},
		{
			name:    "shadow with open engagement runs from working tree",
			variant: "shadow",
			resolver: &fakeEngagementResolver{
				engaged: true,
				info:    EngagementInfo{RestorePointDir: copyDir, Slug: "x"},
			},
			want: "/repo/scenarios/demo",
		},
		{
			name:     "resolver error fails closed",
			variant:  "",
			resolver: &fakeEngagementResolver{err: errors.New("floor unreadable")},
			wantErr:  true,
		},
		{
			name:    "engagement routing to copy with empty path errors",
			variant: "",
			resolver: &fakeEngagementResolver{
				engaged: true,
				info:    EngagementInfo{RestorePointDir: "", Slug: "x"},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &Runner{Engagements: tc.resolver}
			got, err := r.effectiveSourceDir(newSourceDirItem(tc.variant))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got dir %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("effectiveSourceDir = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestEffectiveSourceDirKeysByScenario confirms the resolver is consulted with
// the scenario slug (the engagement key), independent of the variant.
func TestEffectiveSourceDirKeysByScenario(t *testing.T) {
	f := &fakeEngagementResolver{engaged: false}
	r := &Runner{Engagements: f}
	if _, err := r.effectiveSourceDir(newSourceDirItem("shadow")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f.asked) != 1 || f.asked[0] != "demo" {
		t.Errorf("resolver asked %v, want [demo]", f.asked)
	}
}

// TestLayoutVariantTreatsNormalizedLiveAsLive covers the start path, which
// stamps the NORMALIZED variant ("live") onto the descriptor. Reading the raw
// string classified that instance as a shadow, so an engaged live instance kept
// building and running from the working tree the candidate owns.
func TestLayoutVariantTreatsNormalizedLiveAsLive(t *testing.T) {
	for _, variant := range []string{"", "live", "LIVE"} {
		if got := layoutVariant(scenario.Scenario{Slug: "demo", Variant: variant}); got != liveVariant {
			t.Errorf("layoutVariant(%q) = %q, want live", variant, got)
		}
	}
	if got := layoutVariant(scenario.Scenario{Slug: "demo", Variant: "shadow"}); got != shadowVariant {
		t.Errorf("layoutVariant(shadow) = %q, want shadow", got)
	}
}

// materializingEngagementResolver also implements ServingSourceMaterializer,
// as the production floor resolver does.
type materializingEngagementResolver struct {
	fakeEngagementResolver
	dir      string
	err      error
	repoRoot string
}

func (m *materializingEngagementResolver) MaterializeServingSource(_ string, _ EngagementInfo, repoRoot string) (string, error) {
	m.repoRoot = repoRoot
	return m.dir, m.err
}

// TestEffectiveSourceDirUsesServingTree covers the engaged live build that
// could not resolve ../../../packages from the bare restore point: when the
// resolver can materialize a repo-shaped serving tree, the live instance
// builds there, and a materialization failure fails closed instead of falling
// back to the candidate working tree.
func TestEffectiveSourceDirUsesServingTree(t *testing.T) {
	const servingDir = "/cache/vrooli/demo/baseline-x/serving-tree/scenarios/demo"
	engaged := fakeEngagementResolver{engaged: true, info: EngagementInfo{RestorePointDir: "/cache/rp", Slug: "x"}}

	resolver := &materializingEngagementResolver{fakeEngagementResolver: engaged, dir: servingDir}
	r := &Runner{Root: "/repo", Engagements: resolver}
	got, err := r.effectiveSourceDir(newSourceDirItem(""))
	if err != nil || got != servingDir {
		t.Fatalf("live source dir = %q, %v; want %q", got, err, servingDir)
	}
	if resolver.repoRoot != "/repo" {
		t.Errorf("materializer repo root = %q, want /repo", resolver.repoRoot)
	}

	shadow, err := r.effectiveSourceDir(newSourceDirItem("shadow"))
	if err != nil || shadow != "/repo/scenarios/demo" {
		t.Fatalf("shadow source dir = %q, %v; want working tree", shadow, err)
	}

	failing := &materializingEngagementResolver{fakeEngagementResolver: engaged, err: errors.New("disk full")}
	r = &Runner{Root: "/repo", Engagements: failing}
	if got, err := r.effectiveSourceDir(newSourceDirItem("")); err == nil {
		t.Fatalf("materialization failure returned %q, want error", got)
	}
}

// TestRepoCanonicalPathMapsServingTreeLinks: a serving tree reaches shared
// packages through symlinks into the repository. Shared-package provisioning
// and Go freshness must see the repository path, or they silently skip the
// package because it is "outside the repo".
func TestRepoCanonicalPathMapsServingTreeLinks(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "packages", "shared"), 0o755); err != nil {
		t.Fatal(err)
	}
	repo, _ = filepath.EvalSymlinks(repo)
	serving := t.TempDir()
	if err := os.Symlink(filepath.Join(repo, "packages"), filepath.Join(serving, "packages")); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(serving, "packages", "shared")
	if got, want := repoCanonicalPath(repo, linked), filepath.Join(repo, "packages", "shared"); got != want {
		t.Errorf("repoCanonicalPath(linked) = %q, want %q", got, want)
	}
	inRepo := filepath.Join(repo, "packages", "shared")
	if got := repoCanonicalPath(repo, inRepo); got != inRepo {
		t.Errorf("repoCanonicalPath(in repo) = %q, want unchanged", got)
	}
	outside := filepath.Join(serving, "missing")
	if got := repoCanonicalPath(repo, outside); got != outside {
		t.Errorf("repoCanonicalPath(outside) = %q, want unchanged", got)
	}
}
