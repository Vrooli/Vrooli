package lifecycle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func retainedNodeFixture(t *testing.T) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	cwd := filepath.Join(root, "ui")
	script := filepath.Join(cwd, "server.js")
	retainedNodeWrite(t, script, "export {}")
	return root, cwd, script
}
func retainedNodeWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}
func TestRetainedNodeRuntimeDeclaredGraph(t *testing.T) {
	root, cwd, script := retainedNodeFixture(t)
	retainedNodeWrite(t, filepath.Join(cwd, "package.json"), `{"type":"module","dependencies":{"@vrooli/api-base":"*"},"optionalDependencies":{"absent-platform":"*"}}`)
	retainedNodeWrite(t, filepath.Join(cwd, "node_modules/@vrooli/api-base/package.json"), `{"dependencies":{"nested":"*"},"peerDependencies":{"express":"*"}}`)
	retainedNodeWrite(t, filepath.Join(cwd, "node_modules/@vrooli/api-base/node_modules/nested/package.json"), `{"dependencies":{"@vrooli/api-base":"*"}}`)
	retainedNodeWrite(t, filepath.Join(cwd, "node_modules/express/package.json"), `{}`)
	if err := validateRetainedNodeRuntime(root, cwd, script); err != nil {
		t.Fatal(err)
	}
}
func TestRetainedNodeRuntimeRejectsIncompleteGraph(t *testing.T) {
	for _, tc := range []struct {
		name, manifest string
		want           string
	}{
		{"missing declaration", `{}`, "explicit dependencies"},
		{"missing direct", `{"dependencies":{"missing":"*"}}`, "package absent"},
		{"missing peer", `{"dependencies":{},"peerDependencies":{"missing":"*"}}`, "package absent"},
		{"traversal", `{"dependencies":{"../outside":"*"}}`, "unsupported"},
		{"malformed", `{"dependencies":[]}`, "invalid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, cwd, script := retainedNodeFixture(t)
			retainedNodeWrite(t, filepath.Join(cwd, "package.json"), tc.manifest)
			err := validateRetainedNodeRuntime(root, cwd, script)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v, want %q", err, tc.want)
			}
		})
	}
}
func TestRetainedNodeRuntimeNeverUsesOutsideFallback(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "release")
	cwd := filepath.Join(root, "ui")
	script := filepath.Join(cwd, "server.js")
	retainedNodeWrite(t, script, "export {}")
	retainedNodeWrite(t, filepath.Join(cwd, "package.json"), `{"dependencies":{"outside":"*"}}`)
	retainedNodeWrite(t, filepath.Join(parent, "node_modules/outside/package.json"), `{}`)
	t.Setenv("NODE_PATH", filepath.Join(parent, "node_modules"))
	if err := validateRetainedNodeRuntime(root, cwd, script); err == nil {
		t.Fatal("accepted ancestor/global package")
	}
}
func TestRetainedNodeRuntimeRejectsEscapingLinks(t *testing.T) {
	for _, kind := range []string{"package", "manifest", "script", "optional"} {
		t.Run(kind, func(t *testing.T) {
			root, cwd, script := retainedNodeFixture(t)
			outside := t.TempDir()
			retainedNodeWrite(t, filepath.Join(outside, "package.json"), `{}`)
			manifest := `{"dependencies":{"dep":"*"}}`
			if kind == "optional" {
				manifest = `{"dependencies":{},"optionalDependencies":{"dep":"*"}}`
			}
			retainedNodeWrite(t, filepath.Join(cwd, "package.json"), manifest)
			var link, target string
			switch kind {
			case "package", "optional":
				link = filepath.Join(cwd, "node_modules/dep")
				target = outside
			case "manifest":
				link = filepath.Join(cwd, "node_modules/dep/package.json")
				target = filepath.Join(outside, "package.json")
			case "script":
				if err := os.Remove(script); err != nil {
					t.Fatal(err)
				}
				retainedNodeWrite(t, filepath.Join(outside, "server.js"), "export {}")
				link = script
				target = filepath.Join(outside, "server.js")
			}
			if err := os.MkdirAll(filepath.Dir(link), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, link); err != nil {
				t.Skip(err)
			}
			if err := validateRetainedNodeRuntime(root, cwd, script); err == nil {
				t.Fatal("accepted escaping symlink")
			}
		})
	}
}
func TestRetainedNodeRuntimeInternalSymlinkAndRequiredPeer(t *testing.T) {
	root, cwd, script := retainedNodeFixture(t)
	retainedNodeWrite(t, filepath.Join(cwd, "package.json"), `{"dependencies":{"dep":"*"}}`)
	target := filepath.Join(cwd, "node_modules/.pnpm/dep/node_modules/dep")
	retainedNodeWrite(t, filepath.Join(target, "package.json"), `{"peerDependencies":{"peer":"*"}}`)
	if err := os.Symlink(filepath.Join(".pnpm", "dep", "node_modules", "dep"), filepath.Join(cwd, "node_modules/dep")); err != nil {
		t.Skip(err)
	}
	if err := validateRetainedNodeRuntime(root, cwd, script); err == nil {
		t.Fatal("missing required peer must fail closed")
	}
	retainedNodeWrite(t, filepath.Join(cwd, "node_modules/peer/package.json"), `{}`)
	if err := validateRetainedNodeRuntime(root, cwd, script); err != nil {
		t.Fatal(err)
	}
}
func TestRetainedNodeRuntimeOptionalAbsentAndEmptyGraph(t *testing.T) {
	root, cwd, script := retainedNodeFixture(t)
	retainedNodeWrite(t, filepath.Join(cwd, "package.json"), `{"dependencies":{},"optionalDependencies":{"absent":"*"}}`)
	if err := validateRetainedNodeRuntime(root, cwd, script); err != nil {
		t.Fatal(err)
	}
}

func TestRetainedNodeRuntimePeerRequirementCombinations(t *testing.T) {
	for _, dependency := range []bool{false, true} {
		for _, optionalDependency := range []bool{false, true} {
			for _, peer := range []string{"absent", "required", "explicit-required", "optional"} {
				t.Run(fmt.Sprintf("dependency=%t/optional-dependency=%t/peer=%s", dependency, optionalDependency, peer), func(t *testing.T) {
					root, cwd, script := retainedNodeFixture(t)
					manifest := map[string]any{"dependencies": map[string]string{}}
					if dependency {
						manifest["dependencies"] = map[string]string{"dep": "*"}
					}
					if optionalDependency {
						manifest["optionalDependencies"] = map[string]string{"dep": "*"}
					}
					if peer != "absent" {
						manifest["peerDependencies"] = map[string]string{"dep": "*"}
					}
					if peer == "optional" || peer == "explicit-required" {
						manifest["peerDependenciesMeta"] = map[string]any{"dep": map[string]bool{"optional": peer == "optional"}}
					}
					data, err := json.Marshal(manifest)
					if err != nil {
						t.Fatal(err)
					}
					retainedNodeWrite(t, filepath.Join(cwd, "package.json"), string(data))
					// optionalDependencies override regular dependencies. Optional
					// peers do not override any independent required dependency.
					wantMissing := dependency && !optionalDependency || peer == "required" || peer == "explicit-required"
					err = validateRetainedNodeRuntime(root, cwd, script)
					if wantMissing {
						if err == nil || !strings.Contains(err.Error(), "package absent") {
							t.Fatalf("error=%v, want missing required dependency", err)
						}
					} else if err != nil {
						t.Fatalf("absent optional dependency rejected: %v", err)
					}
					retainedNodeWrite(t, filepath.Join(cwd, "node_modules/dep/package.json"), `{}`)
					if err := validateRetainedNodeRuntime(root, cwd, script); err != nil {
						t.Fatalf("valid installed dependency rejected: %v", err)
					}
				})
			}
		}
	}
}

func TestRetainedNodeRuntimeOptionalPeerStillValidatesPresentPackage(t *testing.T) {
	for _, kind := range []string{"invalid-manifest", "missing-manifest", "missing-transitive", "file", "broken-link", "escaping-link"} {
		t.Run(kind, func(t *testing.T) {
			root, cwd, script := retainedNodeFixture(t)
			retainedNodeWrite(t, filepath.Join(cwd, "package.json"), `{"dependencies":{},"peerDependencies":{"dep":"*"},"peerDependenciesMeta":{"dep":{"optional":true}}}`)
			path := filepath.Join(cwd, "node_modules/dep")
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "invalid-manifest":
				retainedNodeWrite(t, filepath.Join(path, "package.json"), `{"dependencies":[]}`)
			case "missing-manifest":
				if err := os.MkdirAll(path, 0700); err != nil {
					t.Fatal(err)
				}
			case "missing-transitive":
				retainedNodeWrite(t, filepath.Join(path, "package.json"), `{"dependencies":{"missing":"*"}}`)
			case "file":
				retainedNodeWrite(t, path, "not a package directory")
			case "broken-link", "escaping-link":
				target := filepath.Join(cwd, "missing")
				if kind == "escaping-link" {
					target = t.TempDir()
					retainedNodeWrite(t, filepath.Join(target, "package.json"), `{}`)
				}
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
			}
			if err := validateRetainedNodeRuntime(root, cwd, script); err == nil {
				t.Fatal("accepted present invalid optional peer")
			}
		})
	}
}

func TestRetainedNodeRuntimeOptionalPeerMetadataValidation(t *testing.T) {
	for _, tc := range []struct {
		name, manifest, want string
	}{
		{"wrong optional type", `{"dependencies":{},"peerDependencies":{"dep":"*"},"peerDependenciesMeta":{"dep":{"optional":"true"}}}`, "invalid"},
		{"wrong metadata type", `{"dependencies":{},"peerDependencies":{"dep":"*"},"peerDependenciesMeta":[]}`, "invalid"},
		{"invalid optional peer name", `{"dependencies":{},"peerDependencies":{"../outside":"*"},"peerDependenciesMeta":{"../outside":{"optional":true}}}`, "unsupported"},
		{"unrelated metadata cannot weaken dependency", `{"dependencies":{"dep":"*"},"peerDependenciesMeta":{"dep":{"optional":true}}}`, "package absent"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, cwd, script := retainedNodeFixture(t)
			retainedNodeWrite(t, filepath.Join(cwd, "package.json"), tc.manifest)
			if err := validateRetainedNodeRuntime(root, cwd, script); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v, want %q", err, tc.want)
			}
		})
	}
}

func TestRetainedNodeRuntimeTransitiveOptionalPeers(t *testing.T) {
	root, cwd, script := retainedNodeFixture(t)
	retainedNodeWrite(t, filepath.Join(cwd, "package.json"), `{"dependencies":{"dep":"*"}}`)
	target := filepath.Join(cwd, "node_modules/.pnpm/dep/node_modules/dep")
	retainedNodeWrite(t, filepath.Join(target, "package.json"), `{"peerDependencies":{"peer":"*"},"peerDependenciesMeta":{"peer":{"optional":true}}}`)
	if err := os.Symlink(filepath.Join(".pnpm", "dep", "node_modules", "dep"), filepath.Join(cwd, "node_modules/dep")); err != nil {
		t.Fatal(err)
	}
	if err := validateRetainedNodeRuntime(root, cwd, script); err != nil {
		t.Fatalf("missing transitive optional peer rejected: %v", err)
	}
	retainedNodeWrite(t, filepath.Join(cwd, "node_modules/peer/package.json"), `{"dependencies":{"required":"*"}}`)
	if err := validateRetainedNodeRuntime(root, cwd, script); err == nil {
		t.Fatal("accepted installed optional peer with missing required dependency")
	}
	retainedNodeWrite(t, filepath.Join(cwd, "node_modules/required/package.json"), `{}`)
	if err := validateRetainedNodeRuntime(root, cwd, script); err != nil {
		t.Fatalf("valid transitive optional peer rejected: %v", err)
	}
}

func TestRetainedNodeRuntimeRejectsAncestorOptionalResolution(t *testing.T) {
	for _, declaration := range []string{"dependency", "optional-dependency", "optional-peer"} {
		for _, layout := range []string{"package", "broken-link", "js", "json", "node", "scoped-package", "higher-ancestor"} {
			t.Run(declaration+"/"+layout, func(t *testing.T) {
				parent := t.TempDir()
				root := filepath.Join(parent, "retained")
				if layout == "higher-ancestor" {
					root = filepath.Join(parent, "intermediate", "retained")
				}
				cwd := filepath.Join(root, "ui")
				script := filepath.Join(cwd, "entry.cjs")
				retainedNodeWrite(t, script, "module.exports = {}")
				name := "external-fixture"
				if layout == "scoped-package" {
					name = "@fixture/external"
				}
				manifest := map[string]any{"dependencies": map[string]string{}}
				switch declaration {
				case "dependency":
					manifest["dependencies"] = map[string]string{name: "*"}
				case "optional-dependency":
					manifest["optionalDependencies"] = map[string]string{name: "*"}
				case "optional-peer":
					manifest["peerDependencies"] = map[string]string{name: "*"}
					manifest["peerDependenciesMeta"] = map[string]any{name: map[string]bool{"optional": true}}
				}
				data, err := json.Marshal(manifest)
				if err != nil {
					t.Fatal(err)
				}
				retainedNodeWrite(t, filepath.Join(cwd, "package.json"), string(data))
				candidate := filepath.Join(parent, "node_modules", filepath.FromSlash(name))
				switch layout {
				case "js", "json", "node":
					retainedNodeWrite(t, candidate+"."+layout, "external file")
				case "broken-link":
					if err := os.MkdirAll(filepath.Dir(candidate), 0700); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(filepath.Join(parent, "missing"), candidate); err != nil {
						t.Fatal(err)
					}
				default:
					retainedNodeWrite(t, filepath.Join(candidate, "package.json"), `{}`)
				}
				if err := validateRetainedNodeRuntime(root, cwd, script); err == nil || !strings.Contains(err.Error(), "external") {
					t.Fatalf("accepted external ancestor candidate: %v", err)
				}
				// A directory alone does not prove a loadable entry. Reject the
				// external fallback conservatively even with an internal directory.
				retainedNodeWrite(t, filepath.Join(cwd, "node_modules", filepath.FromSlash(name), "package.json"), `{}`)
				if err := validateRetainedNodeRuntime(root, cwd, script); err == nil || !strings.Contains(err.Error(), "external") {
					t.Fatalf("internal directory masked external fallback: %v", err)
				}
			})
		}
	}
}

func TestRetainedNodeRuntimeRejectsLooseFileResolution(t *testing.T) {
	for _, extension := range []string{".js", ".json", ".node"} {
		for _, directoryPresent := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/directory=%t", extension, directoryPresent), func(t *testing.T) {
				root, cwd, script := retainedNodeFixture(t)
				retainedNodeWrite(t, filepath.Join(cwd, "package.json"), `{"dependencies":{},"optionalDependencies":{"dep":"*"}}`)
				retainedNodeWrite(t, filepath.Join(cwd, "node_modules/dep")+extension, "unexpected loose file")
				if directoryPresent {
					retainedNodeWrite(t, filepath.Join(cwd, "node_modules/dep/package.json"), `{}`)
				}
				if err := validateRetainedNodeRuntime(root, cwd, script); err == nil || !strings.Contains(err.Error(), "loose-file") {
					t.Fatalf("accepted loose file resolution: %v", err)
				}
			})
		}
	}
}

func TestRetainedNodeRuntimeRechecksAncestorOptionalResolution(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "retained")
	cwd := filepath.Join(root, "ui")
	script := filepath.Join(cwd, "entry.cjs")
	retainedNodeWrite(t, script, "module.exports = {}")
	retainedNodeWrite(t, filepath.Join(cwd, "package.json"), `{"dependencies":{},"peerDependencies":{"later":"*"},"peerDependenciesMeta":{"later":{"optional":true}}}`)
	if err := validateRetainedNodeRuntime(root, cwd, script); err != nil {
		t.Fatal(err)
	}
	retainedNodeWrite(t, filepath.Join(parent, "node_modules/later/package.json"), `{}`)
	if err := validateRetainedNodeRuntime(root, cwd, script); err == nil || !strings.Contains(err.Error(), "external") {
		t.Fatalf("ancestor change was not rechecked: %v", err)
	}
}

func TestRetainedNodeRuntimeTransitiveOptionalCannotResolveAncestor(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "retained")
	cwd := filepath.Join(root, "ui")
	script := filepath.Join(cwd, "entry.cjs")
	retainedNodeWrite(t, script, "module.exports = {}")
	retainedNodeWrite(t, filepath.Join(cwd, "package.json"), `{"dependencies":{"dep":"*"}}`)
	target := filepath.Join(cwd, "node_modules/.pnpm/dep/node_modules/dep")
	retainedNodeWrite(t, filepath.Join(target, "package.json"), `{"peerDependencies":{"external-fixture":"*"},"peerDependenciesMeta":{"external-fixture":{"optional":true}}}`)
	if err := os.Symlink(filepath.Join(".pnpm", "dep", "node_modules", "dep"), filepath.Join(cwd, "node_modules/dep")); err != nil {
		t.Fatal(err)
	}
	retainedNodeWrite(t, filepath.Join(parent, "node_modules/external-fixture/package.json"), `{}`)
	if err := validateRetainedNodeRuntime(root, cwd, script); err == nil || !strings.Contains(err.Error(), "external ancestor") {
		t.Fatalf("accepted transitive external resolution: %v", err)
	}
}

func TestRetainedNodeRuntimePackageMainConfinement(t *testing.T) {
	for _, kind := range []string{"relative-escape", "absolute", "direct-symlink", "extension-symlink", "directory-index-symlink", "root-index-symlink", "missing-prefix-symlink", "broken-link", "internal-main", "internal-extension", "missing-main", "internal-shared-main"} {
		t.Run(kind, func(t *testing.T) {
			root, cwd, script := retainedNodeFixture(t)
			outside := t.TempDir()
			retainedNodeWrite(t, filepath.Join(outside, "external.js"), "module.exports = {}")
			main := "entry"
			wantError := true
			link := ""
			target := filepath.Join(outside, "external.js")
			switch kind {
			case "relative-escape":
				main = "../../external.js"
			case "absolute":
				main = target
			case "direct-symlink":
				link = filepath.Join(cwd, "entry")
			case "extension-symlink":
				link = filepath.Join(cwd, "entry.js")
			case "directory-index-symlink":
				main = "lib"
				link = filepath.Join(cwd, "lib/index.js")
			case "root-index-symlink":
				main = "missing"
				link = filepath.Join(cwd, "index.js")
			case "missing-prefix-symlink":
				main = "linked/missing"
				link = filepath.Join(cwd, "linked")
				target = outside
			case "broken-link":
				link = filepath.Join(cwd, "entry.js")
				target = filepath.Join(cwd, "missing-target")
			case "internal-main":
				main = "entry.js"
				retainedNodeWrite(t, filepath.Join(cwd, main), "module.exports = {}")
				wantError = false
			case "internal-extension":
				retainedNodeWrite(t, filepath.Join(cwd, "entry.js"), "module.exports = {}")
				wantError = false
			case "missing-main":
				wantError = false
			case "internal-shared-main":
				main = "../shared.js"
				retainedNodeWrite(t, filepath.Join(root, "shared.js"), "module.exports = {}")
				wantError = false
			}
			if link != "" {
				if err := os.MkdirAll(filepath.Dir(link), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, link); err != nil {
					t.Fatal(err)
				}
			}
			manifest, err := json.Marshal(map[string]any{"dependencies": map[string]string{}, "main": main})
			if err != nil {
				t.Fatal(err)
			}
			retainedNodeWrite(t, filepath.Join(cwd, "package.json"), string(manifest))
			err = validateRetainedNodeRuntime(root, cwd, script)
			if wantError && (err == nil || !strings.Contains(err.Error(), "main")) {
				t.Fatalf("accepted escaping/broken main candidate: %v", err)
			}
			if !wantError && err != nil {
				t.Fatalf("rejected confined main: %v", err)
			}
		})
	}
}
