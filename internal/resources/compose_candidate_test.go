package resources

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/binaryfetch"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	testresource "github.com/vrooli/vrooli/internal/resources/resourcestest"
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
	if err = os.WriteFile(filepath.Join(root, "resource.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "server.py"), []byte("not executed; original source"), 0600); err != nil {
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
	loaded, snapshot, err := manifestpkg.LoadForComposeCandidate(filepath.Join(root, "resource.json"))
	if err != nil || loaded.Name != m.Name || string(snapshot) != string(raw) {
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
				if err := os.Mkdir(out, 0700); err != nil {
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
					return os.WriteFile(filepath.Join(root, "server.py"), []byte("changed"), 0600)
				case "manifest-drift":
					return os.WriteFile(filepath.Join(root, "resource.json"), []byte("{}"), 0600)
				case "cancel":
					cancel()
				case "competing-empty-output":
					return os.Mkdir(out, 0700)
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
				if err := os.WriteFile(filepath.Join(root, "requirements.lock"), []byte("pkg==1.0\n"), 0600); err != nil {
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
	if err := os.WriteFile(filepath.Join(root, "resource.json"), body, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := manifestpkg.LoadForComposeCandidate(filepath.Join(root, "resource.json")); err == nil {
		t.Fatal("malformed nonempty output accepted")
	}
	m.ManagedService.Acquisition.Targets[0].ArtifactSHA256 = ""
	m.ManagedService.Acquisition.Targets[0].Compose[0].Source = "../escape"
	body, _ = json.Marshal(m)
	_ = os.WriteFile(filepath.Join(root, "resource.json"), body, 0600)
	if _, _, err := manifestpkg.LoadForComposeCandidate(filepath.Join(root, "resource.json")); err == nil || !strings.Contains(err.Error(), "source") {
		t.Fatal("input validation weakened", err)
	}
}
