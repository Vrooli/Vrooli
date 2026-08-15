package capabilityledger

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/deployability"
)

func TestGenerateIncludesVocabulary(t *testing.T) {
	root := repoRoot(t)
	ledger, err := Generate(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(ledger.Capabilities) == 0 {
		t.Fatal("ledger is empty")
	}
	for _, entry := range ledger.Capabilities {
		if len(entry.Platforms) != 3 {
			t.Fatalf("%s has %d platform entries", entry.Capability, len(entry.Platforms))
		}
	}
}

func TestGenerateClassifiesAllFourCapabilitySituations(t *testing.T) {
	ledger, err := Generate(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[CapabilitySituation]bool{}
	for _, entry := range ledger.Capabilities {
		seen[entry.Situation] = true
	}
	for _, situation := range []CapabilitySituation{
		SituationBuiltEverywhere,
		SituationNoWorkRequired,
		SituationNoEquivalentEver,
		SituationPeerNobodyWired,
	} {
		if !seen[situation] {
			t.Errorf("ledger did not classify any capability as %s", situation)
		}
	}
}

func TestGenerateChangesWhenManifestCapabilityChanges(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "source-control")
	before, err := Generate(root)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "internal", "tools", "git", "tool.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var item map[string]interface{}
	if err := json.Unmarshal(data, &item); err != nil {
		t.Fatal(err)
	}
	item["capability"] = "developer-utility"
	data, err = json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := Generate(root)
	if err != nil {
		t.Fatal(err)
	}
	beforeJSON, _ := json.Marshal(before)
	afterJSON, _ := json.Marshal(after)
	if string(beforeJSON) == string(afterJSON) {
		t.Fatal("ledger did not change after manifest capability changed")
	}
}

func TestGenerateRejectsCapabilityOutsideVocabulary(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "not-in-vocabulary")
	if _, err := Generate(root); err == nil {
		t.Fatal("ledger generation accepted a capability outside the checked-in vocabulary")
	}
}

func TestGenerateFleetUsesLiveResourceAndScenarioManifests(t *testing.T) {
	readout, err := GenerateFleet(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if readout.DesktopBundling.Resources == 0 {
		t.Fatal("fleet readout did not discover resources")
	}
	if readout.DesktopBundling.HostRequired == 0 || readout.DesktopBundling.Vendorable == 0 {
		t.Fatalf("desktop bundling did not reflect the live resource fleet: %+v", readout.DesktopBundling)
	}
	if len(readout.DockerBlocked) == 0 {
		t.Fatal("fleet readout did not identify any Docker-backed scenario dependency")
	}
	resolverReason := false
	for _, block := range readout.DockerBlocked {
		for _, dependency := range block.Dependencies {
			for _, reason := range dependency.Reasons {
				if reason.Code == "host_requirement" && reason.Requirement == "docker" {
					resolverReason = true
				}
			}
		}
	}
	if !resolverReason {
		t.Fatal("Docker fleet view did not expose a resolver-derived host requirement reason")
	}
}

func TestResourceDeclarationPreservesMissingManifestAsUnknown(t *testing.T) {
	dep, ok, err := resourceDeclaration(map[string]resourceInput{}, "missing-resource", json.RawMessage(`{"required":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("missing resource dependency was dropped")
	}
	resolution := deployability.Resolve(deployability.ResolutionInput{
		Target: deployability.TargetDeclaration{Name: "fixture", Dependencies: []deployability.DependencyDeclaration{dep}},
		Tier:   deployability.TierLocal,
		OS:     deployability.HostOSLinux,
	})
	if resolution.Verdict != deployability.VerdictUnknown {
		t.Fatalf("verdict = %s, want unknown: %#v", resolution.Verdict, resolution)
	}
	if len(resolution.Dependencies) != 1 || resolution.Dependencies[0].Name != "missing-resource" {
		t.Fatalf("dependencies = %#v, want preserved missing dependency", resolution.Dependencies)
	}
}

func TestReadResourceInputsRejectsDuplicateManifestNames(t *testing.T) {
	root := t.TempDir()
	for _, directory := range []string{"first", "second"} {
		path := filepath.Join(root, "resources", directory)
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		manifest := []byte(`{"name":"same-resource","bundling":"vendorable"}`)
		if err := os.WriteFile(filepath.Join(path, "resource.json"), manifest, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := readResourceInputs(root); err == nil {
		t.Fatal("duplicate resource manifest names were accepted")
	}
}

func TestCapabilityRegistryDefinitionsDeclarePlatformVerdicts(t *testing.T) {
	root := repoRoot(t)
	paths, err := filepath.Glob(filepath.Join(root, "scenarios", "*", "api", "internal", "capabilities", "registry.go"))
	if err != nil {
		t.Fatal(err)
	}
	templatePaths, err := filepath.Glob(filepath.Join(root, "templates", "scenarios", "*", "api", "internal", "capabilities", "registry.go"))
	if err != nil {
		t.Fatal(err)
	}
	paths = append(paths, templatePaths...)
	if len(paths) == 0 {
		t.Fatal("no capability registries discovered")
	}

	for _, path := range paths {
		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			decl, ok := node.(*ast.GenDecl)
			if !ok || decl.Tok != token.VAR {
				return true
			}
			for _, spec := range decl.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for index, name := range valueSpec.Names {
					if name.Name != "Known" && name.Name != "knownDefinitions" {
						continue
					}
					if index >= len(valueSpec.Values) {
						continue
					}
					registry, ok := valueSpec.Values[index].(*ast.CompositeLit)
					if !ok {
						continue
					}
					for _, element := range registry.Elts {
						definition, ok := element.(*ast.CompositeLit)
						if !ok {
							continue
						}
						hasPlatform := false
						for _, field := range definition.Elts {
							keyValue, ok := field.(*ast.KeyValueExpr)
							if !ok {
								continue
							}
							key, keyOK := keyValue.Key.(*ast.Ident)
							if keyOK && key.Name == "Platform" {
								hasPlatform = true
								break
							}
						}
						if !hasPlatform {
							t.Errorf("%s: %s definition at %s lacks PlatformVerdict", path, name.Name, fileSet.Position(definition.Pos()))
							continue
						}
						for _, field := range definition.Elts {
							keyValue, ok := field.(*ast.KeyValueExpr)
							if !ok {
								continue
							}
							key, keyOK := keyValue.Key.(*ast.Ident)
							if !keyOK || key.Name != "Platform" {
								continue
							}
							platform, platformOK := keyValue.Value.(*ast.CompositeLit)
							if !platformOK {
								t.Errorf("%s: %s Platform is not a struct literal", path, name.Name)
								continue
							}
							fields := map[string]bool{}
							for _, platformField := range platform.Elts {
								platformKeyValue, ok := platformField.(*ast.KeyValueExpr)
								if !ok {
									continue
								}
								platformKey, ok := platformKeyValue.Key.(*ast.Ident)
								if ok && (platformKey.Name == "Support" || platformKey.Name == "Reason") {
									fields[platformKey.Name] = true
								}
							}
							for _, requiredField := range []string{"Support", "Reason"} {
								if !fields[requiredField] {
									t.Errorf("%s: %s PlatformVerdict omits %s", path, name.Name, requiredField)
								}
							}
							break
						}
					}
				}
			}
			return false
		})
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func writeFixture(t *testing.T, root, capability string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "internal", "tools", "git"), 0o755); err != nil {
		t.Fatal(err)
	}
	vocabulary := []byte(`{"version":1,"capabilities":["source-control","developer-utility"]}`)
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "capability-vocabulary.json"), vocabulary, 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := []byte(`{"name":"git","capability":"` + capability + `","capability_role":"primary","platforms":["linux"]}`)
	if err := os.WriteFile(filepath.Join(root, "internal", "tools", "git", "tool.json"), manifest, 0o644); err != nil {
		t.Fatal(err)
	}
}
