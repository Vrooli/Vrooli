package agentinstall

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/cli-core/upstreamcheck"
)

func npmAcquisition(packageName, version string) *binaryfetch.Acquisition {
	return &binaryfetch.Acquisition{
		Kind: "npm",
		Targets: []binaryfetch.AcquisitionTarget{{
			Package: packageName,
			Version: version,
			SHA256:  strings.Repeat("a", 64),
		}},
	}
}

func managedUpdateDeps(binDir string, latest, installed string) (updateDeps, *string) {
	installedRef := new(string)
	return updateDeps{
		resolveLatest: func(context.Context, upstreamcheck.SourceKind, string) (string, error) {
			return latest, nil
		},
		installed: func(context.Context, string) (string, error) { return installed, nil },
		install: func(_ context.Context, _, gotBinDir, _, packageRef string) error {
			if gotBinDir != binDir {
				return errors.New("unexpected bin dir " + gotBinDir)
			}
			*installedRef = packageRef
			return nil
		},
		lookPath: func(string) (string, error) { return filepath.Join(binDir, "codex"), nil },
		stat:     func(string) (os.FileInfo, error) { return nil, os.ErrNotExist },
		stdout:   io.Discard,
	}, installedRef
}

func TestUpdateInstallsLatestWhenBehind(t *testing.T) {
	binDir := filepath.Join(t.TempDir(), "bin")
	deps, installedRef := managedUpdateDeps(binDir, "0.161.0", "codex-cli 0.141.0")
	spec := Spec{Binary: "codex", BinDir: binDir, Acquisition: npmAcquisition("@openai/codex", "0.141.0")}

	if err := update(context.Background(), spec, deps); err != nil {
		t.Fatal(err)
	}
	if *installedRef != "@openai/codex@0.161.0" {
		t.Fatalf("installed ref = %q, want latest", *installedRef)
	}
}

func TestUpdateNoOpWhenCurrent(t *testing.T) {
	binDir := filepath.Join(t.TempDir(), "bin")
	deps, installedRef := managedUpdateDeps(binDir, "0.161.0", "codex-cli 0.161.0")
	spec := Spec{Binary: "codex", BinDir: binDir, Acquisition: npmAcquisition("@openai/codex", "0.161.0")}

	if err := update(context.Background(), spec, deps); err != nil {
		t.Fatal(err)
	}
	if *installedRef != "" {
		t.Fatalf("install must not run when up to date, ref = %q", *installedRef)
	}
}

func TestUpdateUsesExplicitUpstreamOverride(t *testing.T) {
	binDir := filepath.Join(t.TempDir(), "bin")
	var gotKind upstreamcheck.SourceKind
	var gotID string
	deps, _ := managedUpdateDeps(binDir, "1.2.3", "")
	deps.resolveLatest = func(_ context.Context, kind upstreamcheck.SourceKind, id string) (string, error) {
		gotKind, gotID = kind, id
		return "1.2.3", nil
	}
	spec := Spec{
		Binary:             "codex",
		BinDir:             binDir,
		Acquisition:        npmAcquisition("@openai/codex", "0.1.0"),
		UpstreamSourceKind: "github",
		UpstreamSourceID:   "openai/codex",
	}

	if err := update(context.Background(), spec, deps); err != nil {
		t.Fatal(err)
	}
	if gotKind != upstreamcheck.SourceGitHub || gotID != "openai/codex" {
		t.Fatalf("resolved %s/%s, want github/openai/codex", gotKind, gotID)
	}
}

func TestUpdateRejectsNonNPMResource(t *testing.T) {
	spec := Spec{
		Binary:      "k6",
		BinDir:      filepath.Join(t.TempDir(), "bin"),
		Acquisition: &binaryfetch.Acquisition{Kind: "url", Targets: []binaryfetch.AcquisitionTarget{{URL: "https://example.invalid/k6", SHA256: strings.Repeat("a", 64)}}},
	}
	err := update(context.Background(), spec, defaultUpdateDeps())
	if err == nil || !strings.Contains(err.Error(), "only npm-acquired") {
		t.Fatalf("error = %v, want npm-only refusal", err)
	}
}

func TestUpdateSurfacesResolveFailure(t *testing.T) {
	binDir := filepath.Join(t.TempDir(), "bin")
	deps, _ := managedUpdateDeps(binDir, "", "")
	deps.resolveLatest = func(context.Context, upstreamcheck.SourceKind, string) (string, error) {
		return "", errors.New("registry down")
	}
	spec := Spec{Binary: "codex", BinDir: binDir, Acquisition: npmAcquisition("@openai/codex", "0.1.0")}

	err := update(context.Background(), spec, deps)
	if err == nil || !strings.Contains(err.Error(), "registry down") {
		t.Fatalf("error = %v, want wrapped resolve failure", err)
	}
}
