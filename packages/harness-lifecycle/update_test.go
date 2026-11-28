package agentinstall

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
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
		install: func(_ context.Context, gotSpec Spec, packageRef string, _ platformDeps) error {
			gotBinDir := gotSpec.BinDir
			if gotBinDir != binDir {
				return errors.New("unexpected bin dir " + gotBinDir)
			}
			*installedRef = packageRef
			installed = "codex-cli " + latest
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

func TestUpdateRefusesSuccessWhenSelectedBinaryIsShadowed(t *testing.T) {
	binDir := filepath.Join(t.TempDir(), "bin")
	deps, _ := managedUpdateDeps(binDir, "0.164.0", "0.164.0")
	deps.lookPath = func(string) (string, error) { return "/other/codex", nil }
	if err := update(context.Background(), Spec{Binary: "codex", BinDir: binDir, Acquisition: npmAcquisition("@openai/codex", "0.161.0")}, deps); err == nil || !strings.Contains(err.Error(), "PATH selects") {
		t.Fatalf("shadowed installation appeared ready: %v", err)
	}
}

func TestUpdateRefusesSuccessWhenInstalledVersionDiffers(t *testing.T) {
	binDir := filepath.Join(t.TempDir(), "bin")
	deps, _ := managedUpdateDeps(binDir, "0.164.0", "0.161.0")
	deps.install = func(context.Context, Spec, string, platformDeps) error { return nil }
	if err := update(context.Background(), Spec{Binary: "codex", BinDir: binDir, Acquisition: npmAcquisition("@openai/codex", "0.161.0")}, deps); err == nil || !strings.Contains(err.Error(), "expected 0.164.0") {
		t.Fatalf("ineffective installation appeared successful: %v", err)
	}
}

func TestUpdateAcceptsAliasOfTheVerifiedInstallation(t *testing.T) {
	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(binDir, "codex")
	if err := os.WriteFile(target, []byte("fixture"), 0o700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "codex")
	if err := os.Symlink(target, alias); err != nil {
		t.Fatal(err)
	}
	deps, _ := managedUpdateDeps(binDir, "0.164.0", "0.164.0")
	deps.lookPath = func(string) (string, error) { return alias, nil }
	deps.stat = os.Stat
	if err := update(context.Background(), Spec{Binary: "codex", BinDir: binDir, Acquisition: npmAcquisition("@openai/codex", "0.161.0")}, deps); err != nil {
		t.Fatalf("same executable identity rejected: %v", err)
	}
}

func TestNPMInstallRequiresMatchingGovernedReceipt(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		fail   bool
	}{
		{"installed", func(map[string]any) {}, false},
		{"bare", func(m map[string]any) {
			for k := range m {
				delete(m, k)
			}
			m["installed"] = true
		}, true},
		{"blocked", func(m map[string]any) { m["blocked"] = true }, true},
		{"preview", func(m map[string]any) { m["dry_run"] = true }, true},
		{"source-update", func(m map[string]any) { m["source_updated"] = true }, true},
		{"unapproved", func(m map[string]any) { m["verdict"] = "unrecorded" }, true},
		{"wrong-manager", func(m map[string]any) { m["package_manager"] = "pnpm" }, true},
		{"wrong-owner", func(m map[string]any) { m["manifest_path"] = "/other/resource.json" }, true},
		{"wrong-command", func(m map[string]any) { m["command"] = "npm install other@1.2.3" }, true},
		{"scripts-enabled", func(m map[string]any) {
			m["command"] = strings.ReplaceAll(m["command"].(string), " --ignore-scripts", "")
		}, true},
		{"unknown-shape", func(m map[string]any) { m["unknown-field"] = true }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			ref := "@openai/codex@0.164.0"
			prefix, err := NPMInstallPrefix(home, "codex", "0.164.0")
			if err != nil {
				t.Fatal(err)
			}
			m := map[string]any{"installed": true, "verdict": "approved", "package_manager": "npm", "manifest_path": filepath.Join(cliutil.ResolveRepoRoot(), "resources", "codex", "resource.json"), "command": cliutil.DisplayCommand(NPMInstallArgs(prefix, ref))}
			tc.mutate(m)
			receipt, err := json.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			tools := t.TempDir()
			script := "#!/bin/sh\nprintf '%s' '" + strings.ReplaceAll(string(receipt), "'", "'\\''") + "'\n"
			if err := os.WriteFile(filepath.Join(tools, "scenario-dependency-analyzer"), []byte(script), 0o700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", tools)
			err = installNPM(context.Background(), Spec{ResourceName: "codex", BinDir: filepath.Join(home, ".local", "bin")}, ref)
			if (err != nil) != tc.fail {
				t.Fatalf("error=%v fail=%t", err, tc.fail)
			}
		})
	}
}

func TestUpdateDoesNotDowngradeAheadInstallation(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "bin")
	deps, installed := managedUpdateDeps(bin, "0.164.0", "0.165.0")
	if err := update(context.Background(), Spec{ResourceName: "codex", Binary: "codex", BinDir: bin, Acquisition: npmAcquisition("@openai/codex", "0.161.0")}, deps); err != nil {
		t.Fatal(err)
	}
	if *installed != "" {
		t.Fatal("ahead installation was replaced")
	}
}

func TestInstallCommandsRejectTrailingArgumentsBeforeEffects(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, cmd := range []cliapp.Command{UpdateCommand(Spec{}), DirectInstallCommand(Spec{})} {
		for _, args := range [][]string{{"--dry-run"}, {"--version"}, {"unexpected"}} {
			if err := cmd.Run(args); err == nil || !strings.Contains(err.Error(), "accepts no arguments") {
				t.Fatalf("%s accepted args%v: %v", cmd.Name, args, err)
			}
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".local")); !os.IsNotExist(err) {
		t.Fatal("rejected invocation changed install state")
	}
}
