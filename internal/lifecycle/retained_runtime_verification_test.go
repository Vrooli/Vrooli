package lifecycle

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/lifecycle/retainedruntime"
	"github.com/vrooli/vrooli/internal/scenario"
)

func recaptureRetainedVerificationFixture(t *testing.T, r *Runner, item scenario.Scenario) retainedruntime.Release {
	t.Helper()
	raw, err := os.ReadFile(item.ServicePath)
	if err != nil {
		t.Fatal(err)
	}
	release, err := r.runtimeReleaseStore().Capture(context.Background(), retainedruntime.CaptureSpec{Scenario: item.Slug, Variant: item.Variant, SourceDir: item.Path, BuildIdentity: "verification-fixture", ServiceManifest: raw, ForbiddenPaths: []string{"data", "logs"}})
	if err != nil {
		t.Fatal(err)
	}
	return release
}

func TestRetainedRuntimeVerificationRoutesRecheckBinding(t *testing.T) {
	for _, route := range []string{"verify", "selected-capture"} {
		for _, mutation := range []string{"none", "state-contract", "ancestor-optional", "internal-directory-with-ancestor", "escaping-main"} {
			t.Run(route+"/"+mutation, func(t *testing.T) {
				r, item, release := retainedIntegrationFixture(t, nil)
				t.Setenv("NODE_OPTIONS", "")
				t.Setenv("NODE_PATH", "")
				if mutation == "ancestor-optional" || mutation == "internal-directory-with-ancestor" {
					retainedIntegrationWrite(t, filepath.Join(item.Path, "worker/package.json"), []byte(`{"dependencies":{},"optionalDependencies":{"external-fixture":"*"}}`), 0600)
					if mutation == "internal-directory-with-ancestor" {
						retainedIntegrationWrite(t, filepath.Join(item.Path, "worker/node_modules/external-fixture/package.json"), []byte(`{}`), 0600)
					}
					release = recaptureRetainedVerificationFixture(t, r, item)
				}
				if mutation == "escaping-main" {
					retainedIntegrationWrite(t, filepath.Join(item.Path, "worker/package.json"), []byte(`{"dependencies":{},"main":"../../outside.js"}`), 0600)
					release = recaptureRetainedVerificationFixture(t, r, item)
				}
				if route == "selected-capture" {
					if _, err := r.runtimeReleaseStore().Select(context.Background(), item.Slug, item.Variant, release.ID, "", "fixture-only-selection"); err != nil {
						t.Fatal(err)
					}
				}
				before, err := r.runtimeReleaseStore().ReadSelection(item.Slug, item.Variant)
				if err != nil {
					t.Fatal(err)
				}
				switch mutation {
				case "state-contract":
					retainedIntegrationWrite(t, filepath.Join(item.Path, ".vrooli/runtime-release.json"), []byte(`{"version":1,"state_contract":"changed"}`), 0600)
				case "ancestor-optional", "internal-directory-with-ancestor":
					retainedIntegrationWrite(t, filepath.Join(filepath.Dir(filepath.Dir(release.RuntimeDir)), "node_modules/external-fixture/package.json"), []byte(`{}`), 0600)
				}
				// File/interpreter hashes remain valid. Only owner binding/closure can
				// detect these conditions; neither route is allowed to launch anything.
				if _, err := r.runtimeReleaseStore().Verify(context.Background(), item.Slug, item.Variant, release.ID); err != nil {
					t.Fatalf("byte fixture invalid: %v", err)
				}
				var got retainedruntime.Release
				if route == "verify" {
					got, err = r.VerifyRuntimeRelease(context.Background(), item.Slug, release.ID)
				} else {
					got, err = r.CaptureRuntimeRelease(context.Background(), item.Slug, "")
				}
				if mutation == "none" {
					if err != nil || got.ID != release.ID {
						t.Fatalf("valid release rejected: %+v %v", got, err)
					}
				} else {
					want := "state compatibility"
					if strings.Contains(mutation, "ancestor") {
						want = "external ancestor"
					}
					if mutation == "escaping-main" {
						want = "main/index escapes"
					}
					if err == nil || !strings.Contains(err.Error(), want) || got.ID != "" {
						t.Fatalf("mutation=%s release=%+v error=%v, want %s", mutation, got, err, want)
					}
				}
				after, err := r.runtimeReleaseStore().ReadSelection(item.Slug, item.Variant)
				if err != nil || after != before {
					t.Fatalf("verification changed selection: before=%+v after=%+v error=%v", before, after, err)
				}
			})
		}
	}
}

func TestRetainedRuntimeVerificationPreservesCaptureSourcePath(t *testing.T) {
	r, item, _ := retainedIntegrationFixture(t, nil)
	t.Setenv("NODE_OPTIONS", "")
	t.Setenv("NODE_PATH", "")
	original := item.Path
	custom := filepath.Join(t.TempDir(), "custom-capture")
	if err := os.Rename(item.Path, custom); err != nil {
		t.Fatal(err)
	}
	item.Path, item.ServicePath = custom, filepath.Join(custom, ".vrooli/service.json")
	release := recaptureRetainedVerificationFixture(t, r, item)
	if got, err := r.VerifyRuntimeRelease(context.Background(), item.Slug, release.ID); err != nil || got.SourceDir != custom {
		t.Fatalf("verify lost captured source path: %+v %v", got, err)
	}
	if _, err := r.runtimeReleaseStore().Select(context.Background(), item.Slug, item.Variant, release.ID, "", "custom-path-fixture"); err != nil {
		t.Fatal(err)
	}
	if got, err := r.CaptureRuntimeRelease(context.Background(), item.Slug, custom); err != nil || got.SourceDir != custom {
		t.Fatalf("selected capture lost explicit path: %+v %v", got, err)
	}
	raw, err := os.ReadFile(item.ServicePath)
	if err != nil {
		t.Fatal(err)
	}
	retainedIntegrationWrite(t, filepath.Join(original, ".vrooli/service.json"), raw, 0600)
	if _, err := r.CaptureRuntimeRelease(context.Background(), item.Slug, original); err == nil || !strings.Contains(err.Error(), "different canonical scenario path") {
		t.Fatalf("accepted selected capture through wrong source: %v", err)
	}
	if selected, err := r.SelectedRuntimeReleaseID(context.Background(), item.Slug); err != nil || selected != release.ID {
		t.Fatalf("verification changed selection: %q %v", selected, err)
	}
}
