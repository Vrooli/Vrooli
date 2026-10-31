package installgateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/resources/agentinstall"
	"github.com/vrooli/vrooli/internal/tools/akt"
	"github.com/vrooli/vrooli/internal/tools/hyperframes"
	"github.com/vrooli/vrooli/internal/tools/kubo"
	triposrsource "github.com/vrooli/vrooli/internal/tools/triposr-source"
)

func aktHostToolFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "internal", "tools", "akt")
	if err := os.MkdirAll(source, 0o700); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "internal", "tools", "akt", "tool.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "tool.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestHostReleaseRequiresExactAktOwnerClosure(t *testing.T) {
	root := aktHostToolFixture(t)
	r, err := Resolve(root, akt.ToolName, "host-tool", "source", akt.ToolName, akt.Version)
	if err != nil {
		t.Fatal(err)
	}
	if r.HostTool || r.HostSource || !r.ExactVersion || !r.Profile.ScriptsDisabled || !r.Profile.FrozenLockfile || r.Profile.LifecycleMode != "pinned-release" {
		t.Fatalf("unsafe release resolution: %+v", r)
	}
	if r.Command() != "native-acquire akt 1.0.4" {
		t.Fatalf("unexpected command: %s", r.Command())
	}
	if err := ApplyExactVersion(&r, akt.Version); err != nil {
		t.Fatal(err)
	}
	if err := ApplyExactVersion(&r, "1.0.3"); err == nil {
		t.Fatal("changed version accepted")
	}
	for _, change := range []struct{ owner, ecosystem, pkg, version string }{
		{"../akt", "source", "akt", akt.Version}, {"akt", "go", "akt", akt.Version}, {"akt", "source", "other", akt.Version}, {"akt", "source", "akt", "latest"}, {"akt", "source", "akt", "1.0.3"},
	} {
		if _, err := Resolve(root, change.owner, "host-tool", change.ecosystem, change.pkg, change.version); err == nil {
			t.Fatalf("unreviewed release accepted: %+v", change)
		}
	}
	if err := os.WriteFile(r.ManifestPath, []byte(`{"name":"akt","version":"1.0.4"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(root, "akt", "host-tool", "source", "akt", akt.Version); err == nil {
		t.Fatal("changed source closure accepted")
	}
}

func TestHostReleaseRevalidatesBeforeAnyAcquisition(t *testing.T) {
	root := aktHostToolFixture(t)
	base, err := Resolve(root, "akt", "host-tool", "source", "akt", akt.Version)
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*Resolution){
		"command":  func(r *Resolution) { r.Argv = []string{"sh", "unreviewed"} },
		"path":     func(r *Resolution) { r.SurfaceRoot = root },
		"manifest": func(r *Resolution) { r.ManifestPath += ".other" },
		"package":  func(r *Resolution) { r.PackageName = "other" },
		"exact":    func(r *Resolution) { r.ExactVersion = false },
		"policy":   func(r *Resolution) { r.Profile.ScriptsDisabled = false },
		"source":   func(r *Resolution) { r.HostSource = true },
		"npm":      func(r *Resolution) { r.HostTool = true },
		"refresh":  func(r *Resolution) { r.HostToolRefresh = true },
		"manager":  func(r *Resolution) { r.PackageManager = "npm" },
	} {
		t.Run(name, func(t *testing.T) {
			r := base
			mutate(&r)
			if _, err := (ExecInstaller{}).Install(context.Background(), r); err == nil || !strings.Contains(err.Error(), "resolution changed") {
				t.Fatalf("changed resolution admitted: %v", err)
			}
		})
	}
	if err := os.WriteFile(base.ManifestPath, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	changedMode := base
	changedMode.Profile.LifecycleMode = "unknown"
	if _, err := (ExecInstaller{}).Install(context.Background(), changedMode); err == nil || !strings.Contains(err.Error(), "unsupported native") {
		t.Fatalf("unknown native profile executed: %v", err)
	}
	if _, err := (ExecInstaller{}).Install(context.Background(), base); err == nil {
		t.Fatal("source changed after resolution was acquired")
	}
}

func hostToolFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "internal", "tools", "hyperframes")
	if err := os.MkdirAll(source, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"package.json", "package-lock.json"} {
		b, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "internal", "tools", "hyperframes", name))
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(source, name), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestHostSourceResolvesOnlyPinnedTripoSRClosure(t *testing.T) {
	r, err := Resolve(t.TempDir(), triposrsource.ToolName, "host-tool", "source", triposrsource.ToolName, triposrsource.Commit)
	if err != nil {
		t.Fatal(err)
	}
	if !r.HostSource || !r.Profile.FrozenLockfile || r.Profile.LifecycleMode != "pinned-source-tree" {
		t.Fatalf("unexpected source profile resolution: %+v", r)
	}
	if err := ApplyExactVersion(&r, triposrsource.Commit); err != nil || !r.ExactVersion {
		t.Fatalf("exact commit rejected: %+v %v", r, err)
	}
	if _, err := Resolve(t.TempDir(), triposrsource.ToolName, "host-tool", "git", triposrsource.ToolName, "main"); err == nil {
		t.Fatal("unversioned Git source accepted")
	}
}

func TestHostToolResolvesOnlyDeclaredFrozenClosure(t *testing.T) {
	root := hostToolFixture(t)
	r, err := Resolve(root, "hyperframes", "host-tool", "npm", "hyperframes", hyperframes.Version)
	if err != nil {
		t.Fatal(err)
	}
	if !r.HostTool || !r.Profile.FrozenLockfile || !r.Profile.ScriptsDisabled || r.Profile.Governance != "scenario-dependency-analyzer" {
		t.Fatalf("unsafe profile %+v", r)
	}
	if r.Command() != "npm ci --ignore-scripts --no-audit --no-fund" {
		t.Fatalf("command %s", r.Command())
	}
	if err = ApplyExactVersion(&r, hyperframes.Version); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(r.Command(), "save-exact") {
		t.Fatal("mutating exact-save flag appended to frozen reproduction")
	}
	for _, change := range []struct{ owner, pkg, version string }{{"../escape", "hyperframes", hyperframes.Version}, {"hyperframes", "other", hyperframes.Version}, {"hyperframes", "hyperframes", "0.8.45"}} {
		if _, err = Resolve(root, change.owner, "host-tool", "npm", change.pkg, change.version); err == nil {
			t.Fatalf("accepted %+v", change)
		}
	}
	if err = os.WriteFile(filepath.Join(root, "internal", "tools", "hyperframes", "package-lock.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = Resolve(root, "hyperframes", "host-tool", "npm", "hyperframes", hyperframes.Version); err == nil {
		t.Fatal("unrecognized lock accepted")
	}
}

func kuboHostToolFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "internal", "tools", kubo.ToolName)
	if err := os.MkdirAll(source, 0o700); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "internal", "tools", kubo.ToolName, "tool.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "tool.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestKuboHostReleaseRequiresExactOwnerClosure(t *testing.T) {
	root := kuboHostToolFixture(t)
	r, err := Resolve(root, kubo.ToolName, "host-tool", "source", kubo.ToolName, kubo.Version)
	if err != nil {
		t.Fatal(err)
	}
	if r.HostTool || r.HostSource || r.HostToolRefresh || !r.ExactVersion || !r.Profile.ScriptsDisabled || !r.Profile.FrozenLockfile || r.Profile.LifecycleMode != "pinned-release" || r.Command() != "native-acquire kubo 0.43.1" {
		t.Fatalf("unsafe Kubo resolution: %+v", r)
	}
	if err := ApplyExactVersion(&r, kubo.Version); err != nil {
		t.Fatal(err)
	}
	if err := ApplyExactVersion(&r, "0.43.0"); err == nil {
		t.Fatal("changed Kubo version accepted")
	}
	for _, v := range []struct{ owner, ecosystem, pkg, version string }{{"../kubo", "source", "kubo", kubo.Version}, {"kubo", "go", "kubo", kubo.Version}, {"kubo", "source", "ipfs", kubo.Version}, {"kubo", "source", "kubo", "latest"}, {"kubo", "source", "kubo", "0.43.0"}} {
		if _, err := Resolve(root, v.owner, "host-tool", v.ecosystem, v.pkg, v.version); err == nil {
			t.Fatalf("unreviewed Kubo request: %+v", v)
		}
	}
}

func TestKuboHostReleaseRevalidatesBeforeAcquisition(t *testing.T) {
	root := kuboHostToolFixture(t)
	base, err := Resolve(root, kubo.ToolName, "host-tool", "source", kubo.ToolName, kubo.Version)
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*Resolution){
		"command":     func(r *Resolution) { r.Argv = []string{"sh", "unreviewed"} },
		"surface":     func(r *Resolution) { r.SurfaceRoot = root },
		"manifest":    func(r *Resolution) { r.ManifestPath += ".other" },
		"owner":       func(r *Resolution) { r.PackageName = "unreviewed" },
		"cross-owner": func(r *Resolution) { r.PackageName = akt.ToolName },
		"exact":       func(r *Resolution) { r.ExactVersion = false },
		"policy":      func(r *Resolution) { r.Profile.ScriptsDisabled = false },
		"host-source": func(r *Resolution) { r.HostSource = true },
		"host-npm":    func(r *Resolution) { r.HostTool = true },
		"refresh":     func(r *Resolution) { r.HostToolRefresh = true },
		"manager":     func(r *Resolution) { r.PackageManager = "go" },
	} {
		t.Run(name, func(t *testing.T) {
			r := base
			mutate(&r)
			if _, err := (ExecInstaller{}).Install(context.Background(), r); err == nil {
				t.Fatal("changed Kubo resolution executed")
			}
		})
	}
	if err := os.WriteFile(base.ManifestPath, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (ExecInstaller{}).Install(context.Background(), base); err == nil {
		t.Fatal("changed Kubo source executed")
	}
}

func TestHarnessHostToolRequiresResourcePackageAndExactVersion(t *testing.T) {
	old := hostToolInstall
	hostToolInstall = func(context.Context, string, []string) (string, error) {
		t.Fatal("invalid harness plan reached install effects")
		return "", nil
	}
	t.Cleanup(func() { hostToolInstall = old })
	root := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	for _, tc := range []struct{ name, pkg string }{{"codex", "@openai/codex"}, {"claude-code", "@anthropic-ai/claude-code"}, {"opencode", "opencode-ai"}, {"grok", "@xai-official/grok"}} {
		dir := filepath.Join(root, "resources", tc.name)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "resources", tc.name, "resource.json"))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "resource.json"), data, 0600); err != nil {
			t.Fatal(err)
		}
		r, err := Resolve(root, tc.name, "host-tool", "npm", tc.pkg, "1.2.3")
		if err != nil {
			t.Fatal(err)
		}
		if !r.HostTool || !r.ExactVersion || !r.Profile.ScriptsDisabled || r.Profile.FrozenLockfile || !strings.Contains(r.Command(), "--prefix") {
			t.Fatalf("bad resource plan: %+v", r)
		}
		for _, pkg := range []string{"other", tc.pkg + ";bad"} {
			if _, err := Resolve(root, tc.name, "host-tool", "npm", pkg, "1.2.3"); err == nil {
				t.Fatal("changed package accepted")
			}
		}
		if _, err := Resolve(root, tc.name, "host-tool", "npm", tc.pkg, "latest"); err == nil {
			t.Fatal("floating version accepted")
		}
		if err := ApplyExactVersion(&r, "1.2.4"); err == nil {
			t.Fatal("changed version accepted")
		}
		r.Argv = append(r.Argv, "--prefix", "/unowned")
		if _, err := provisionAgentResource(context.Background(), r); err == nil {
			t.Fatal("changed prefix accepted")
		}
		good, err := Resolve(root, tc.name, "host-tool", "npm", tc.pkg, "1.2.3")
		if err != nil {
			t.Fatal(err)
		}
		changed := bytes.ReplaceAll(data, []byte(tc.pkg), []byte("unreviewed-package"))
		if err := os.WriteFile(good.ManifestPath, changed, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := provisionAgentResource(context.Background(), good); err == nil {
			t.Fatal("changed manifest package accepted")
		}

	}
}

func TestHarnessCandidateOnlyPromotesAfterQualification(t *testing.T) {
	for _, mode := range []string{"qualified", "unsupported", "source-changed", "source-changed-during-probe", "cancelled", "external-absolute-link", "native-qualified", "native-bad-json", "native-wrong-name", "native-wrong-version", "native-undeclared", "native-escape-package", "native-source-change", "native-malformed-path", "native-reuse-selector"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
			dir := filepath.Join(root, "resources", "codex")
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "resources", "codex", "resource.json"))
			if err != nil {
				t.Fatal(err)
			}
			selectedRelative := "node_modules/.bin/codex"
			if strings.HasPrefix(mode, "native-") {
				if mode != "native-reuse-selector" {
					selectedRelative = "node_modules/@openai/codex-linux-x64/bin/codex"
				}
				if mode == "native-malformed-path" {
					selectedRelative = "node_modules/../escape"
				}
				var source map[string]any
				if err := json.Unmarshal(data, &source); err != nil {
					t.Fatal(err)
				}
				source["acquisition"].(map[string]any)["targets"].([]any)[0].(map[string]any)["bin_path"] = selectedRelative
				data, err = json.Marshal(source)
				if err != nil {
					t.Fatal(err)
				}
			}
			manifest := filepath.Join(dir, "resource.json")
			if err := os.WriteFile(manifest, data, 0600); err != nil {
				t.Fatal(err)
			}
			oldPrefix := filepath.Join(home, "prior")
			if err := os.MkdirAll(filepath.Join(oldPrefix, "node_modules", ".bin"), 0700); err != nil {
				t.Fatal(err)
			}
			oldBinary := filepath.Join(oldPrefix, "node_modules", ".bin", "codex")
			if err := os.WriteFile(oldBinary, []byte("prior-bytes"), 0700); err != nil {
				t.Fatal(err)
			}
			public := filepath.Join(home, ".local", "bin", "codex")
			if err := os.MkdirAll(filepath.Dir(public), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(oldBinary, public); err != nil {
				t.Fatal(err)
			}
			installBefore, probeBefore := hostToolInstall, agentBinaryProbe
			t.Cleanup(func() { hostToolInstall = installBefore; agentBinaryProbe = probeBefore })
			installCalls := 0
			hostToolInstall = func(_ context.Context, _ string, argv []string) (string, error) {
				installCalls++
				prefix := argv[3]
				pkg := filepath.Join(prefix, "node_modules", "@openai", "codex")
				if err := os.MkdirAll(pkg, 0700); err != nil {
					return "", err
				}
				main := map[string]any{"name": "@openai/codex", "version": "0.164.0"}
				if strings.HasPrefix(mode, "native-") && mode != "native-undeclared" {
					main["optionalDependencies"] = map[string]string{"@openai/codex-linux-x64": "0.164.0"}
				}
				b, _ := json.Marshal(main)
				if err := os.WriteFile(filepath.Join(pkg, "package.json"), b, 0600); err != nil {
					return "", err
				}
				if err := os.MkdirAll(filepath.Join(prefix, "node_modules", ".bin"), 0700); err != nil {
					return "", err
				}
				if err := os.WriteFile(filepath.Join(prefix, "node_modules", ".bin", "codex"), []byte("candidate-bytes"), 0700); err != nil {
					return "", err
				}
				if strings.HasPrefix(mode, "native-") {
					native := filepath.Join(prefix, "node_modules", "@openai", "codex-linux-x64")
					if err := os.MkdirAll(filepath.Join(native, "bin"), 0700); err != nil {
						return "", err
					}
					identity := map[string]string{"name": "@openai/codex-linux-x64", "version": "0.164.0"}
					if mode == "native-wrong-name" {
						identity["name"] = "other"
					}
					if mode == "native-wrong-version" {
						identity["version"] = "0.163.0"
					}
					nativeJSON, _ := json.Marshal(identity)
					if mode == "native-bad-json" {
						nativeJSON = []byte("{")
					}
					if err := os.WriteFile(filepath.Join(native, "package.json"), nativeJSON, 0600); err != nil {
						return "", err
					}
					if mode == "native-escape-package" {
						if err := os.WriteFile(filepath.Join(pkg, "other"), []byte("sibling-bytes"), 0700); err != nil {
							return "", err
						}
						if err := os.Symlink("../../codex/other", filepath.Join(native, "bin", "codex")); err != nil {
							return "", err
						}
					} else if err := os.WriteFile(filepath.Join(native, "bin", "codex"), []byte("native-bytes"), 0700); err != nil {
						return "", err
					}
				}
				if mode == "external-absolute-link" {
					external := filepath.Join(home, "external-codex")
					if err := os.WriteFile(external, []byte("#!/bin/sh\necho 0.164.0\n"), 0700); err != nil {
						return "", err
					}
					bin := filepath.Join(prefix, "node_modules", ".bin", "codex")
					if err := os.Remove(bin); err != nil {
						return "", err
					}
					if err := os.Symlink(external, bin); err != nil {
						return "", err
					}
					// Old digest logic could misresolve this absolute link as an
					// in-tree relative path if this synthetic target existed.
					synthetic := filepath.Join(filepath.Dir(bin), external)
					if err := os.MkdirAll(filepath.Dir(synthetic), 0700); err != nil {
						return "", err
					}
					if err := os.WriteFile(synthetic, []byte("synthetic"), 0700); err != nil {
						return "", err
					}
				}
				if mode == "source-changed" {
					if err := os.WriteFile(manifest, bytes.ReplaceAll(data, []byte("@openai/codex"), []byte("other-package")), 0600); err != nil {
						return "", err
					}
				}
				return "", nil
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			priorAtProbe := oldBinary
			agentBinaryProbe = func(_ context.Context, _ string, binary string, _ string, _ string) error {
				if mode == "external-absolute-link" || (strings.HasPrefix(mode, "native-") && mode != "native-qualified" && mode != "native-source-change" && mode != "native-reuse-selector") {
					t.Fatal("unverified executable must not be launched")
				}
				prefix, err := agentinstall.NPMInstallPrefix(home, "codex", "0.164.0")
				if err != nil || binary != filepath.Join(prefix, selectedRelative) {
					t.Fatalf("must qualify after relocation: %s, %v", binary, err)
				}
				if selected, err := os.Readlink(public); err != nil || selected != priorAtProbe {
					t.Fatal("selection changed before qualification")
				}
				if mode == "native-source-change" {
					return os.WriteFile(manifest, bytes.ReplaceAll(data, []byte(selectedRelative), []byte("node_modules/.bin/codex")), 0600)
				}
				if mode == "source-changed-during-probe" {
					return os.WriteFile(manifest, bytes.ReplaceAll(data, []byte("@openai/codex"), []byte("other-package")), 0600)
				}
				if mode == "cancelled" {
					cancel()
				}
				if mode == "unsupported" {
					return errors.New("no script-free binary")
				}
				return nil
			}
			r, err := Resolve(root, "codex", "host-tool", "npm", "@openai/codex", "0.164.0")
			if mode == "native-malformed-path" {
				if err == nil {
					t.Fatal("malformed executable path was admitted")
				}
				if selected, _ := os.Readlink(public); selected != oldBinary {
					t.Fatal("malformed source replaced previous selection")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			_, err = ExecInstaller{}.Install(ctx, r)
			if (err == nil) != (mode == "qualified" || mode == "native-qualified" || mode == "native-reuse-selector") {
				t.Fatalf("qualification result %v", err)
			}
			selected, err := os.Readlink(public)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "qualified" || mode == "native-qualified" || mode == "native-reuse-selector" {
				if selected == oldBinary || !strings.Contains(selected, "0.164.0-") {
					t.Fatalf("new candidate not selected: %s", selected)
				}
			} else if selected != oldBinary {
				t.Fatal("failed candidate replaced prior selection")
			}
			if mode == "native-reuse-selector" {
				priorAtProbe = selected
				selectedRelative = "node_modules/@openai/codex-linux-x64/bin/codex"
				var source map[string]any
				if err := json.Unmarshal(data, &source); err != nil {
					t.Fatal(err)
				}
				source["acquisition"].(map[string]any)["targets"].([]any)[0].(map[string]any)["bin_path"] = selectedRelative
				updated, err := json.Marshal(source)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(manifest, updated, 0600); err != nil {
					t.Fatal(err)
				}
				reused, err := Resolve(root, "codex", "host-tool", "npm", "@openai/codex", "0.164.0")
				if err != nil {
					t.Fatal(err)
				}
				if _, err := (ExecInstaller{}).Install(ctx, reused); err != nil {
					t.Fatal(err)
				}
				prefix, err := agentinstall.NPMInstallPrefix(home, "codex", "0.164.0")
				if err != nil {
					t.Fatal(err)
				}
				if got, _ := os.Readlink(public); got != filepath.Join(prefix, selectedRelative) || installCalls != 1 {
					t.Fatalf("changed native selector must reuse verified tree without another install: %s, installs=%d", got, installCalls)
				}
			}
			prior, err := os.ReadFile(oldBinary)
			if err != nil || string(prior) != "prior-bytes" {
				t.Fatal("prior bytes changed")
			}
		})
	}
}

func TestVersionProbeOutputMountIsInsideWritableScratch(t *testing.T) {
	receipts := "/owned/resource/qualification"
	spec := agentVersionProbeSpec("/usr/bin/bwrap", "/owned/candidate/bin/tool", receipts)
	tmpfs, bind := -1, -1
	for i, arg := range spec.Args {
		if arg == "--tmpfs" && i+1 < len(spec.Args) && spec.Args[i+1] == "/tmp" {
			tmpfs = i
		}
		if arg == "--bind" && i+2 < len(spec.Args) && spec.Args[i+1] == receipts {
			bind = i
			target := spec.Args[i+2]
			if !strings.HasPrefix(target, "/tmp/") {
				t.Fatalf("output mount would require mkdir under read-only host root: %s", target)
			}
			if !strings.Contains(strings.Join(spec.Args[i+3:], " "), target+"/version.txt") {
				t.Fatal("bounded version output is not written inside mounted receipt directory")
			}
		}
	}
	if tmpfs < 0 || bind <= tmpfs {
		t.Fatal("writable scratch must exist before output mount")
	}
	if len(spec.Env) != 0 || spec.Limits.WallTime <= 0 || spec.Limits.MemoryMaxBytes <= 0 || spec.Limits.TasksMax == 0 {
		t.Fatal("version probe lost bounded owner constraints")
	}
	if got := spec.Args[len(spec.Args)-2:]; got[0] != "/owned/candidate/bin/tool" || got[1] != "--version" {
		t.Fatalf("probe must invoke only exact candidate version: %v", got)
	}
}
