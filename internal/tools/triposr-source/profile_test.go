package triposrsource

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
)

func TestTemporaryAdapterImportsTSRAndParsesPinnedCheckpointConfig(t *testing.T) {
	root, err := InstallRoot()
	if err != nil {
		t.Fatal(err)
	}
	upstream := filepath.Join(root, "source")
	if _, err := os.Stat(filepath.Join(upstream, "tsr", "system.py")); err != nil {
		t.Skip("requires governed local TripoSR upstream source")
	}
	vrooliRoot := filepath.Dir(filepath.Dir(root))
	python := filepath.Join(vrooliRoot, "data", "vrooli", "image-tools", "pyenv", "bin", "python")
	if _, err := os.Stat(python); err != nil {
		t.Skip("requires the governed Image Tools Python runtime")
	}
	modelConfig := filepath.Join(vrooliRoot, "data", "vrooli", "image-tools", "models", "triposr-vision-to-mesh", "config.yaml")
	if _, err := os.Stat(modelConfig); err != nil {
		t.Skip("requires the governed TripoSR model config")
	}
	candidate := filepath.Join(t.TempDir(), "adapter", "source")
	if _, err := applySourceAdapter(upstream, candidate); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = thawTree(candidate) })
	code := fmt.Sprintf("from dataclasses import dataclass\nfrom tsr.system import TSR\nfrom tsr.utils import BaseModule, parse_structured\nfrom tsr.config_adapter import load_config\ncfg=load_config(%q, TSR.Config)\nassert cfg.cond_image_size == 512\nassert cfg.backbone['in_channels'] == 1024\nassert parse_structured(TSR.Config, cfg) is cfg\nclass Probe(BaseModule):\n @dataclass\n class Config(BaseModule.Config):\n  value: int\n def configure(self): pass\nassert Probe({'value': 3}).cfg.value == 3\n", modelConfig)
	cmd := exec.Command(python, "-B", "-c", code)
	cmd.Env = append(os.Environ(), "PYTHONDONTWRITEBYTECODE=1", "PYTHONPATH="+candidate)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("adapted TripoSR import/config failed: %v\n%s", err, output)
	}
}

func TestTemporaryAdapterFullInstalledModelMeshQualification(t *testing.T) {
	if os.Getenv("TRIPOSR_QUALIFY_NATIVE") != "1" {
		t.Skip("set TRIPOSR_QUALIFY_NATIVE=1 to run the bounded installed-model qualification")
	}
	root, err := InstallRoot()
	if err != nil {
		t.Fatal(err)
	}
	vrooliRoot := filepath.Dir(filepath.Dir(root))
	upstream := filepath.Join(root, "source")
	model := filepath.Join(vrooliRoot, "data", "vrooli", "image-tools", "models", "triposr-vision-to-mesh")
	python := filepath.Join(vrooliRoot, "data", "vrooli", "image-tools", "pyenv", "bin", "python")
	image := filepath.Join(vrooliRoot, "plan-artifacts", "efforts", "aquila-subscription-value-2026-10-21", "stage", "device-plate.png")
	mask := filepath.Join(vrooliRoot, "plan-artifacts", "efforts", "aquila-subscription-value-2026-10-21", "stage", "device-mask.png")
	sidecar := filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(root))), "Vrooli", "scenarios", "image-tools", "api", "internal", "sidecar", "py")
	// Resolve the repository from the caller's working directory rather than
	// assuming the checkout's absolute location in the artifact layout.
	if repo, e := os.Getwd(); e == nil {
		for i := 0; i < 6; i++ {
			if _, e = os.Stat(filepath.Join(repo, "scenarios", "image-tools", "api", "internal", "sidecar", "py", "image_tools_sidecar")); e == nil {
				sidecar = filepath.Join(repo, "scenarios", "image-tools", "api", "internal", "sidecar", "py")
				break
			}
			repo = filepath.Dir(repo)
		}
	}
	for _, path := range []string{upstream, model, python, image, mask, filepath.Join(sidecar, "image_tools_sidecar", "image_to_object.py")} {
		if _, err = os.Stat(path); err != nil {
			t.Fatalf("native qualification input %s: %v", path, err)
		}
	}
	candidate := filepath.Join(t.TempDir(), "adapter", "source")
	adapterDigest, err := applySourceAdapter(upstream, candidate)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = thawTree(candidate) })
	upstreamDigest, err := TreeDigest(upstream)
	if err != nil {
		t.Fatal(err)
	}
	profile := Receipt{Schema: 2, Commit: Commit, ArchiveSHA256: ArchiveSHA256, ArchiveBytes: ArchiveBytes, TreeSHA256: adapterDigest, AdapterTreeSHA256: adapterDigest, UpstreamTreeSHA256: upstreamDigest, Adapter: AdapterVersion, License: "MIT"}
	profileBytes, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(filepath.Dir(filepath.Dir(candidate)), "source-profile.json"), profileBytes, 0600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "triposr-qualification.zip")
	cmd := exec.Command(python, "-B", "-m", "image_tools_sidecar.image_to_object", "--model", model, "--source-dir", candidate, "--image", image, "--mask", mask, "--out", out, "--device", "cuda", "--mesh-resolution", "128")
	cmd.Env = append(os.Environ(), "PYTHONDONTWRITEBYTECODE=1", "PYTHONPATH="+sidecar)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("full installed-model TripoSR qualification failed: %v\n%s", err, output)
	}
	if info, err := os.Stat(out); err != nil || info.Size() == 0 {
		t.Fatalf("qualification archive missing or empty: stat=%v", err)
	}
}

func thawTree(root string) error {
	var dirs []string
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			dirs = append(dirs, path)
			return nil
		}
		return os.Chmod(path, 0644)
	}); err != nil {
		return err
	}
	for i := len(dirs) - 1; i >= 0; i-- {
		if err := os.Chmod(dirs[i], 0755); err != nil {
			return err
		}
	}
	return nil
}

func TestSelectedInstallationRequiresLedgerAndMatchesSourceTree(t *testing.T) {
	home, restore := hostreqkittest.StubInvokingUser(t)
	defer restore()
	root, err := InstallRoot()
	if err != nil {
		t.Fatal(err)
	}
	upstream := filepath.Join(root, "source")
	source := filepath.Join(root, "adapter", "source")
	if err := os.MkdirAll(upstream, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "README.md"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	digest, err := TreeDigest(source)
	if err != nil {
		t.Fatal(err)
	}
	upstreamDigest, err := TreeDigest(upstream)
	if err != nil {
		t.Fatal(err)
	}
	receipt := Receipt{Schema: 2, Commit: Commit, ArchiveSHA256: ArchiveSHA256, ArchiveBytes: ArchiveBytes, TreeSHA256: digest, AdapterTreeSHA256: digest, UpstreamTreeSHA256: upstreamDigest, Adapter: AdapterVersion, License: "MIT"}
	b, _ := json.Marshal(receipt)
	if err = os.WriteFile(filepath.Join(root, "source-profile.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err = SelectedInstallation(context.Background()); err == nil {
		t.Fatal("unledgered source tree was selected")
	}
	if err = cliinstall.RecordToolArtifacts(home, cliinstall.InstallEntry{Scope: cliinstall.ScopeRuntime, Kind: cliinstall.EntryDirectory, Path: root, Prefix: filepath.Dir(root)}); err != nil {
		t.Fatal(err)
	}
	selected, got, err := SelectedInstallation(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if selected != source || got.TreeSHA256 != digest {
		t.Fatalf("unexpected selected profile %q %+v", selected, got)
	}
	if err = os.WriteFile(filepath.Join(source, "README.md"), []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err = SelectedInstallation(context.Background()); err == nil {
		t.Fatal("tampered source tree was selected")
	}
}

func TestSelectedInstallationIsReadOnlyWhenAdapterIsMissing(t *testing.T) {
	home, restore := hostreqkittest.StubInvokingUser(t)
	defer restore()
	root, err := InstallRoot()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Join(root, "source"), 0700); err != nil {
		t.Fatal(err)
	}
	digest, err := TreeDigest(filepath.Join(root, "source"))
	if err != nil {
		t.Fatal(err)
	}
	receipt := Receipt{Schema: 1, Commit: Commit, ArchiveSHA256: ArchiveSHA256, ArchiveBytes: ArchiveBytes, TreeSHA256: digest, License: "MIT"}
	b, _ := json.Marshal(receipt)
	if err = os.WriteFile(filepath.Join(root, "source-profile.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	if err = cliinstall.RecordToolArtifacts(home, cliinstall.InstallEntry{Scope: cliinstall.ScopeRuntime, Kind: cliinstall.EntryDirectory, Path: root, Prefix: filepath.Dir(root)}); err != nil {
		t.Fatal(err)
	}
	if _, _, err = SelectedInstallation(context.Background()); err == nil {
		t.Fatal("legacy profile unexpectedly selected")
	}
	if _, err = os.Lstat(filepath.Join(root, "adapter")); !os.IsNotExist(err) {
		t.Fatalf("read-only profile getter mutated adapter state: stat error=%v", err)
	}
	got, err := os.ReadFile(filepath.Join(root, "source-profile.json"))
	if err != nil || string(got) != string(b) {
		t.Fatalf("read-only profile getter changed receipt: err=%v", err)
	}
}
