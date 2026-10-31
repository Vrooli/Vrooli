package kubo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func fixtureFetch(t *testing.T, calls *int) func(context.Context, binaryfetch.Target, string, binaryfetch.ProgressFunc) (string, error) {
	t.Helper()
	return func(ctx context.Context, target binaryfetch.Target, tree string, _ binaryfetch.ProgressFunc) (string, error) {
		*calls++
		want, err := releaseTarget("linux", "amd64")
		if err != nil || target != want {
			t.Fatalf("unreviewed acquisition target: %+v %v", target, err)
		}
		directory := filepath.Join(tree, "kubo")
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return "", err
		}
		for _, name := range []string{"LICENSE-MIT", "LICENSE-APACHE"} {
			b, err := os.ReadFile(name)
			if err != nil {
				return "", err
			}
			if err := os.WriteFile(filepath.Join(directory, name), b, 0o644); err != nil {
				return "", err
			}
		}
		path := filepath.Join(directory, "ipfs")
		return path, os.WriteFile(path, []byte("fixture executable; never invoked"), 0o755)
	}
}

func testInstallRoot(t *testing.T) (string, string) {
	t.Helper()
	home := t.TempDir()
	return filepath.Join(home, ".vrooli", "opt", ToolName), home
}

func TestPinnedReleaseSelectionReusesAndRejectsTamperedBytes(t *testing.T) {
	ctx := context.Background()
	root, home := testInstallRoot(t)
	target, err := releaseTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	fetch := fixtureFetch(t, &calls)
	path, err := provisionAt(ctx, root, home, target, fetch)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(root, "tree", "kubo", "ipfs") {
		t.Fatalf("unexpected selected command %s", path)
	}
	if _, r, err := selectedAt(ctx, root, home); err != nil || r.ArchiveSHA256 != target.SHA256 || r.TreeBytes <= 0 {
		t.Fatalf("selection failed: %+v %v", r, err)
	}
	if again, err := provisionAt(ctx, root, home, target, fetch); err != nil || again != path || calls != 1 {
		t.Fatalf("valid reuse downloaded again: %s %d %v", again, calls, err)
	}
	if err := os.WriteFile(path, []byte("tampered"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, err := selectedAt(ctx, root, home); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("tampered selection accepted: %v", err)
	}
	if _, err := provisionAt(ctx, root, home, target, fetch); err == nil || calls != 1 {
		t.Fatalf("corrupt selection overwritten/reacquired: %d %v", calls, err)
	}
	b, _ := os.ReadFile(path)
	if string(b) != "tampered" {
		t.Fatal("existing bytes changed after refusal")
	}
}

func TestReleaseChecksumFailureDoesNotPublishOrLeaveCandidate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("wrong archive bytes")) }))
	defer srv.Close()
	root, home := testInstallRoot(t)
	target, err := releaseTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	fetch := func(ctx context.Context, got binaryfetch.Target, tree string, p binaryfetch.ProgressFunc) (string, error) {
		if got.SHA256 != target.SHA256 {
			t.Fatal("pinned digest changed")
		}
		got.URL = srv.URL // Local transport only; retain the production expected digest.
		return binaryfetch.FetchDir(ctx, got, tree, p)
	}
	if _, err := provisionAt(context.Background(), root, home, target, fetch); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("wrong archive accepted: %v", err)
	}
	if _, err := os.Lstat(root); !os.IsNotExist(err) {
		t.Fatalf("failed acquisition published selection: %v", err)
	}
	left, err := filepath.Glob(filepath.Join(filepath.Dir(root), ".kubo-release-*"))
	if err != nil || len(left) != 0 {
		t.Fatalf("candidate leak: %v %v", left, err)
	}
}

func TestPinnedReleaseRejectsUnsupportedPlatformAndChangedSource(t *testing.T) {
	for _, p := range [][2]string{{"linux", "arm64"}, {"darwin", "amd64"}, {"windows", "amd64"}, {"linux", "386"}} {
		if _, err := releaseTarget(p[0], p[1]); err == nil {
			t.Fatalf("unqualified platform accepted: %v", p)
		}
	}
	source := t.TempDir()
	p := filepath.Join(source, "tool.json")
	if err := os.WriteFile(p, manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSource(source); err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(manifestBytes), "3f2bf974", "0f2bf974", 1)
	if err := os.WriteFile(p, []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSource(source); err == nil {
		t.Fatal("changed archive digest accepted")
	}
}

func TestReleaseRefusesUnownedDirectoryAndCancelledAcquisition(t *testing.T) {
	target, _ := releaseTarget("linux", "amd64")
	root, home := testInstallRoot(t)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, "operator-file")
	if err := os.WriteFile(p, []byte("retain me"), 0o600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	fetch := fixtureFetch(t, &calls)
	if _, err := provisionAt(context.Background(), root, home, target, fetch); err == nil || calls != 0 {
		t.Fatalf("unowned directory touched: %d %v", calls, err)
	}
	if b, err := os.ReadFile(p); err != nil || string(b) != "retain me" {
		t.Fatal("unowned file changed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := provisionAt(ctx, root+"-cancel", home, target, fetch); !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatalf("cancelled acquisition ran: %d %v", calls, err)
	}
}

func TestReleaseLicenseMismatchNeverSelectsCandidate(t *testing.T) {
	root, home := testInstallRoot(t)
	target, _ := releaseTarget("linux", "amd64")
	calls := 0
	fetch := fixtureFetch(t, &calls)
	bad := func(ctx context.Context, target binaryfetch.Target, tree string, p binaryfetch.ProgressFunc) (string, error) {
		path, err := fetch(ctx, target, tree, p)
		if err != nil {
			return "", err
		}
		return path, os.WriteFile(filepath.Join(tree, "kubo", "LICENSE-MIT"), []byte("unreviewed license"), 0o644)
	}
	if _, err := provisionAt(context.Background(), root, home, target, bad); err == nil || !strings.Contains(err.Error(), "license") {
		t.Fatalf("changed notice selected: %v", err)
	}
	if _, err := os.Lstat(root); !os.IsNotExist(err) {
		t.Fatalf("failed notice published: %v", err)
	}
	left, err := filepath.Glob(filepath.Join(filepath.Dir(root), ".kubo-release-*"))
	if err != nil || len(left) != 0 {
		t.Fatalf("candidate leak: %v %v", left, err)
	}
}

type ambientGeneric struct{ applied *bool }

func (ambientGeneric) Name() string           { return ToolName }
func (ambientGeneric) Kind() hostreqspec.Kind { return hostreqspec.KindTool }
func (ambientGeneric) Inspect(_ hostreqkit.Host, r hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	s := hostreqkit.BaseStatus(r)
	s.SupportClass, s.ExecutionState = hostreqkit.SupportSupported, hostreqkit.ExecutionAlreadyPresent
	s.Installed, s.Command, s.Version = true, "/unowned/ambient/ipfs", "unknown"
	return s
}
func (g ambientGeneric) Apply(_ hostreqkit.Host, s hostreqkit.ItemStatus, _ hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	*g.applied = true
	return s, nil
}

func TestHandlerNeverAdoptsGenericAmbientCommandOrDownloader(t *testing.T) {
	var manifest hostreqkit.ToolManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	applied := false
	generic := ambientGeneric{applied: &applied}
	missing := func(context.Context) (string, Receipt, error) { return "", Receipt{}, errors.New("no owned selection") }
	h := newHandler(manifest, generic, missing)
	host := hostreqkit.Host{OS: "linux"}
	requirement := hostreqspec.ResolvedRequirement{Name: ToolName}
	status := h.Inspect(host, requirement)
	if status.Installed || status.Command != "" {
		t.Fatalf("adopted ambient command: %+v", status)
	}
	result, err := h.Apply(host, status, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil || applied || result.ExecutionState != hostreqkit.ExecutionWouldInstall || !strings.Contains(strings.Join(result.Notes, " "), "source/kubo@0.43.1") {
		t.Fatalf("not governed preview: %+v %v genericApplied=%v", result, err, applied)
	}
	selected := func(context.Context) (string, Receipt, error) {
		return "/owned/verified/kubo/ipfs", Receipt{Version: Version}, nil
	}
	owned := newHandler(manifest, generic, selected).Inspect(host, requirement)
	if !owned.Installed || owned.Command != "/owned/verified/kubo/ipfs" {
		t.Fatalf("owned selection lost: %+v", owned)
	}
}
