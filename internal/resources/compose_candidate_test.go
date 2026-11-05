package resources

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/binaryfetch"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	testresource "github.com/vrooli/vrooli/internal/resources/resourcestest"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func candidateFixture(t *testing.T) (ResourceManifest, []byte, binaryfetch.AcquisitionTarget, string) {
	t.Helper()
	root := t.TempDir()
	m := testresource.ResourceManifest("candidate-fixture", testresource.WithResourceDriver("managed-service"))
	m.ManagedService.Artifact.Layout = "dir"
	m.ManagedService.Artifact.EntryPath = "source/server.py"
	m.ManagedService.Artifact.SHA256 = ""
	m.ManagedService.Artifact.SHA256ByPlatform = map[string]string{runtime.GOOS + "-" + runtime.GOARCH: ""}
	target := binaryfetch.AcquisitionTarget{Layout: "dir", Compose: []binaryfetch.ComposeStep{{Kind: "local", Role: "wrapper", Source: "server.py", Dest: "source/server.py"}}}
	m.ManagedService.Acquisition = &binaryfetch.Acquisition{Kind: "composed", Targets: []binaryfetch.AcquisitionTarget{target}}
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "resource.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "server.py"), []byte("not executed; original source"), 0o600); err != nil {
		t.Fatal(err)
	}
	return m, raw, target, root
}

func TestComposeCandidateLoadsOnlyPreparationAndReusesCanonicalComposer(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux atomic publisher qualification")
	}
	m, raw, target, root := candidateFixture(t)
	if _, err := manifestpkg.Load(filepath.Join(root, "resource.json")); err == nil {
		t.Fatal("normal load admitted missing output pins")
	}
	loaded, err := manifestpkg.ParseForComposeCandidate(filepath.Join(root, "resource.json"), raw)
	if err != nil || loaded.Name != m.Name {
		t.Fatalf("candidate loader %+v %v", loaded, err)
	}
	var first string
	for _, name := range []string{"one", "two"} {
		out := filepath.Join(t.TempDir(), name)
		receipt, err := composeCandidate(context.Background(), m, raw, target, root, out, func(ctx context.Context, target binaryfetch.AcquisitionTarget, input, artifact string) error {
			if input == root {
				t.Fatal("builder used mutable original inputs")
			}
			body, err := os.ReadFile(filepath.Join(input, "server.py"))
			if err != nil || string(body) != "not executed; original source" {
				t.Fatal("snapshot missing", err)
			}
			return composeManagedServiceArtifact(ctx, target, input, artifact)
		})
		if err != nil {
			t.Fatal(err)
		}
		actual, err := binaryfetch.TreeDigestContext(context.Background(), filepath.Join(out, "artifact"))
		if err != nil || receipt.Status != "candidate-only" || receipt.ArtifactSHA256 != actual || receipt.Inputs["server.py"] == "" {
			t.Fatalf("receipt %+v %v", receipt, err)
		}
		if first != "" && first != actual {
			t.Fatal("two independent composes differ")
		}
		first = actual
		if _, err := os.Stat(filepath.Join(out, "receipt.json")); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(out, "artifact", "receipt.json")); !os.IsNotExist(err) {
			t.Fatal("receipt contaminated artifact identity")
		}
	}
}

func TestComposeCandidateNeverPublishesFailureDriftCancellationOrReplacement(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux atomic publisher qualification")
	}
	for _, kind := range []string{"builder-error", "source-drift", "manifest-drift", "cancel", "competing-empty-output", "existing-output"} {
		t.Run(kind, func(t *testing.T) {
			m, raw, target, root := candidateFixture(t)
			out := filepath.Join(t.TempDir(), "candidate")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			if kind == "existing-output" {
				if err := os.Mkdir(out, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			_, err := composeCandidate(ctx, m, raw, target, root, out, func(ctx context.Context, target binaryfetch.AcquisitionTarget, input, artifact string) error {
				calls++
				if kind == "builder-error" {
					return errors.New("injected")
				}
				if err := composeManagedServiceArtifact(ctx, target, input, artifact); err != nil {
					return err
				}
				switch kind {
				case "source-drift":
					return os.WriteFile(filepath.Join(root, "server.py"), []byte("changed"), 0o600)
				case "manifest-drift":
					return os.WriteFile(filepath.Join(root, "resource.json"), []byte("{}"), 0o600)
				case "cancel":
					cancel()
				case "competing-empty-output":
					return os.Mkdir(out, 0o700)
				}
				return nil
			})
			if err == nil {
				t.Fatal("failed operation published")
			}
			if kind == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal("lost cancellation", err)
			}
			if kind == "existing-output" && calls != 0 {
				t.Fatal("existing output ran builder")
			}
			_, statErr := os.Stat(out)
			if kind == "existing-output" || kind == "competing-empty-output" {
				if statErr != nil {
					t.Fatal("prior/concurrent directory removed", statErr)
				}
				entries, _ := os.ReadDir(out)
				if len(entries) != 0 {
					t.Fatal("candidate replaced concurrent empty directory")
				}
			} else if !os.IsNotExist(statErr) {
				t.Fatal("failure published output")
			}
			stages, _ := filepath.Glob(filepath.Join(filepath.Dir(out), ".vrooli-compose-candidate-*"))
			if len(stages) != 0 {
				t.Fatal("owned temp leaked", stages)
			}
		})
	}
}

func TestComposeCandidateRejectsUnsafeInputsBeforeBuilder(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux atomic publisher qualification")
	}
	for _, kind := range []string{"escape", "symlink", "unhashed", "sdist", "index-credentials", "overlap"} {
		t.Run(kind, func(t *testing.T) {
			m, raw, target, root := candidateFixture(t)
			switch kind {
			case "escape":
				target.Compose[0].Source = "../outside"
			case "symlink":
				if err := os.Symlink("resource.json", filepath.Join(root, "link.py")); err != nil {
					t.Fatal(err)
				}
				target.Compose[0].Source = "link.py"
			case "unhashed", "sdist", "index-credentials":
				if err := os.WriteFile(filepath.Join(root, "requirements.lock"), []byte("pkg==1.0\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				step := binaryfetch.ComposeStep{Kind: "python-wheels", Role: "wheels", Dest: "wheels", Lockfile: "requirements.lock", IndexURL: "https://pypi.org/simple"}
				if kind == "sdist" {
					step.AllowSDists = true
				}
				if kind == "index-credentials" {
					step.IndexURL = "https://secret@pypi.org/simple"
				}
				target.Compose = append(target.Compose, step)
			case "overlap":
				target.Compose = append(target.Compose, binaryfetch.ComposeStep{Kind: "local", Role: "duplicate", Source: "server.py", Dest: "source"})
			}
			calls := 0
			_, err := composeCandidate(context.Background(), m, raw, target, root, filepath.Join(t.TempDir(), "candidate"), func(context.Context, binaryfetch.AcquisitionTarget, string, string) error { calls++; return nil })
			if err == nil || calls != 0 {
				t.Fatalf("unsafe input reached builder %v calls%d", err, calls)
			}
		})
	}
}

func TestComposeCandidateManifestRejectsInvalidInputAndNonemptyOutputDigest(t *testing.T) {
	m, _, _, root := candidateFixture(t)
	m.ManagedService.Acquisition.Targets[0].ArtifactSHA256 = "invalid"
	body, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(root, "resource.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := manifestpkg.ParseForComposeCandidate(filepath.Join(root, "resource.json"), body); err == nil {
		t.Fatal("malformed nonempty output accepted")
	}
	m.ManagedService.Acquisition.Targets[0].ArtifactSHA256 = ""
	m.ManagedService.Acquisition.Targets[0].Compose[0].Source = "../escape"
	body, _ = json.Marshal(m)
	_ = os.WriteFile(filepath.Join(root, "resource.json"), body, 0o600)
	if _, err := manifestpkg.ParseForComposeCandidate(filepath.Join(root, "resource.json"), body); err == nil || !strings.Contains(err.Error(), "source") {
		t.Fatal("input validation weakened", err)
	}
}

// The real SA3/ACE compose shape overlays reviewed wrappers onto a pinned source tree.
func TestComposeCandidateArchiveThenLocalChildrenReusesCanonicalComposer(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux atomic publisher qualification")
	}
	m, _, target, root := candidateFixture(t)
	archiveBytes := candidateArchive(t)
	target.Compose = []binaryfetch.ComposeStep{
		{Kind: "url", Role: "source", URL: "https://candidate.invalid/source.tar", SHA256: fmt.Sprintf("%x", sha256.Sum256(archiveBytes)), Archive: "tar.gz", BinPath: "upstream", Dest: "source"},
		{Kind: "local", Role: "wrapper", Source: "server.py", Dest: "source/server.py"},
		{Kind: "local", Role: "loader", Source: "loader.py", Dest: "source/loader.py"},
	}
	if err := os.WriteFile(filepath.Join(root, "loader.py"), []byte("reviewed loader"), 0o600); err != nil {
		t.Fatal(err)
	}
	m.ManagedService.Acquisition.Targets = []binaryfetch.AcquisitionTarget{target}
	raw, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(root, "resource.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	prior := binaryfetch.HTTPClient
	binaryfetch.HTTPClient = &http.Client{Transport: candidateRoundTrip(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != target.Compose[0].URL {
			t.Fatalf("unexpected URL %s", req.URL)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(archiveBytes))}, nil
	})}
	t.Cleanup(func() { binaryfetch.HTTPClient = prior })
	first := ""
	for _, name := range []string{"one", "two"} {
		out := filepath.Join(t.TempDir(), name)
		receipt, err := composeCandidate(context.Background(), m, raw, target, root, out, composeManagedServiceArtifact)
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range []string{"core.py", "server.py", "loader.py"} {
			if _, err := os.Stat(filepath.Join(out, "artifact", "source", file)); err != nil {
				t.Fatal(err)
			}
		}
		if first != "" && first != receipt.ArtifactSHA256 {
			t.Fatal("ordered archive/local candidate differs")
		}
		first = receipt.ArtifactSHA256
	}
}

type candidateRoundTrip func(*http.Request) (*http.Response, error)

func (f candidateRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestComposeCandidatePrecancelDoesNotLoadManifestOrFacts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var controller *Controller
	if _, err := controller.ComposeCandidate(ctx, "fixture", "new-output"); !errors.Is(err, context.Canceled) {
		t.Fatal("pre-cancel reached controller preparation", err)
	}
}

func TestComposeCandidateUnsafeWheelLockCreatesNothingOutsideStage(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux candidate qualification")
	}
	m, raw, target, root := candidateFixture(t)
	parent := t.TempDir()
	target.Compose = append(target.Compose, binaryfetch.ComposeStep{Kind: "python-wheels", Role: "wheels", Dest: "wheels", Lockfile: "../../escaped/new.lock", IndexURL: "https://pypi.org/simple"})
	called := 0
	_, err := composeCandidate(context.Background(), m, raw, target, root, filepath.Join(parent, "out"), func(context.Context, binaryfetch.AcquisitionTarget, string, string) error { called++; return nil })
	if err == nil || called != 0 {
		t.Fatal("unsafe lockfile reached builder", err, called)
	}
	if _, err := os.Lstat(filepath.Join(parent, "escaped")); !os.IsNotExist(err) {
		t.Fatal("unsafe lockfile created directory/file outside owned stage", err)
	}
}

func TestComposeCandidatePublicManifestGuardAndBootstrap(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux candidate qualification")
	}
	for _, kind := range []string{"regular-bootstrap", "exact-size-boundary", "symlink", "directory", "too-large", "pre-cancel"} {
		t.Run(kind, func(t *testing.T) {
			m, raw, _, fixture := candidateFixture(t)
			base := t.TempDir()
			resourceRoot := filepath.Join(base, "resources", m.Name)
			if err := os.MkdirAll(filepath.Dir(resourceRoot), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(fixture, resourceRoot); err != nil {
				t.Fatal(err)
			}
			manifestPath := filepath.Join(resourceRoot, "resource.json")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "symlink":
				other := filepath.Join(t.TempDir(), "valid.json")
				if err := os.WriteFile(other, raw, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(manifestPath); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(other, manifestPath); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Remove(manifestPath); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(manifestPath, 0o700); err != nil {
					t.Fatal(err)
				}
			case "too-large":
				if err := os.WriteFile(manifestPath, bytes.Repeat([]byte(" "), maxComposeCandidateManifestBytes+1), 0o600); err != nil {
					t.Fatal(err)
				}
			case "exact-size-boundary":
				if err := os.WriteFile(manifestPath, append(raw, bytes.Repeat([]byte(" "), maxComposeCandidateManifestBytes-len(raw))...), 0o600); err != nil {
					t.Fatal(err)
				}
			case "pre-cancel":
				cancel()
			}
			prior := composeCandidateResolveTarget
			t.Cleanup(func() { composeCandidateResolveTarget = prior })
			resolved := 0
			composeCandidateResolveTarget = func(ctx context.Context, loaded ResourceManifest) (binaryfetch.AcquisitionTarget, binaryfetch.Facts, error) {
				resolved++
				if loaded.Name != m.Name {
					t.Fatal("wrong parsed manifest", loaded.Name)
				}
				facts := binaryfetch.Facts{}
				target, err := loaded.ManagedService.Acquisition.Resolve(facts)
				return target, facts, err
			}
			out := filepath.Join(t.TempDir(), "candidate")
			receipt, err := (&Controller{Root: base}).ComposeCandidate(ctx, m.Name, out)
			if kind == "regular-bootstrap" || kind == "exact-size-boundary" {
				if err != nil || resolved != 1 || receipt.Status != "candidate-only" {
					t.Fatalf("public bootstrap failed %+v %v calls%d", receipt, err, resolved)
				}
				if _, err := os.Stat(filepath.Join(out, "artifact", "source", "server.py")); err != nil {
					t.Fatal("canonical compose output missing", err)
				}
			} else {
				if err == nil || resolved != 0 {
					t.Fatal("unsafe/pre-cancel manifest reached facts/builder", err, resolved)
				}
				if kind == "too-large" && !strings.Contains(err.Error(), "1 MiB") {
					t.Fatal("wrong size refusal", err)
				}
				if kind == "pre-cancel" && !errors.Is(err, context.Canceled) {
					t.Fatal("cancellation lost", err)
				}
				if _, err := os.Lstat(out); !os.IsNotExist(err) {
					t.Fatal("failed public operation published output", err)
				}
			}
		})
	}
}

func candidateArchive(t *testing.T) []byte {
	t.Helper()
	var archive bytes.Buffer
	gz := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gz)
	source := []byte("upstream implementation")
	if err := tw.WriteHeader(&tar.Header{Name: "upstream/core.py", Mode: 0o644, Size: int64(len(source))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(source); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return archive.Bytes()
}

func TestComposeCacheReusesUnchangedPartsForOfflineWrapperChange(t *testing.T) {
	m, _, target, root := candidateFixture(t)
	archive := candidateArchive(t)
	step := binaryfetch.ComposeStep{Kind: "url", Role: "source", URL: "https://candidate.invalid/source.tar", SHA256: fmt.Sprintf("%x", sha256.Sum256(archive)), Archive: "tar.gz", BinPath: "upstream", Dest: "source"}
	target.Compose = append([]binaryfetch.ComposeStep{step}, target.Compose...)
	m.ManagedService.Acquisition.Targets = []binaryfetch.AcquisitionTarget{target}
	calls := 0
	old := binaryfetch.HTTPClient
	binaryfetch.HTTPClient = &http.Client{Transport: candidateRoundTrip(func(req *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(archive))}, nil
	})}
	t.Cleanup(func() { binaryfetch.HTTPClient = old })
	cache := composeStepCache{root: t.TempDir(), resource: m.Name, lockRoot: t.TempDir()}
	out1 := filepath.Join(t.TempDir(), "first")
	if err := composeManagedServiceArtifactCached(context.Background(), target, root, out1, &cache); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("first acquisition calls=%d", calls)
	}
	if err := os.WriteFile(filepath.Join(root, "server.py"), []byte("changed reviewed wrapper"), 0o600); err != nil {
		t.Fatal(err)
	}
	cache.offline = true
	out2 := filepath.Join(t.TempDir(), "second")
	if err := composeManagedServiceArtifactCached(WithOfflineCompose(context.Background()), target, root, out2, &cache); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("offline reuse acquired bytes: calls=%d", calls)
	}
	body, err := os.ReadFile(filepath.Join(out2, "source", "server.py"))
	if err != nil || string(body) != "changed reviewed wrapper" {
		t.Fatalf("wrapper change lost: %s %v", body, err)
	}
	if _, err := os.Stat(filepath.Join(out2, "source", "core.py")); err != nil {
		t.Fatal(err)
	}
	key, err := cache.key(step, root)
	if err != nil {
		t.Fatal(err)
	}
	if _, hit, err := resourcedeployment.InspectCachedArtifacts(context.Background(), cache.root, key); err != nil || !hit {
		t.Fatalf("preview miss after reuse: %v", err)
	}
	entries, err := os.ReadDir(cache.root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cache.root, entries[0].Name(), "payload", "core.py"), []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := composeManagedServiceArtifactCached(WithOfflineCompose(context.Background()), target, root, filepath.Join(t.TempDir(), "third"), &cache); err == nil || calls != 1 {
		t.Fatalf("tampered offline cache acquired or accepted: %d %v", calls, err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := resourcedeployment.InspectCachedArtifacts(cancelled, cache.root, key); !errors.Is(err, context.Canceled) {
		t.Fatalf("lost cancellation: %v", err)
	}
}
