package facts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

type stackManifest struct {
	Components map[string]struct {
		Role  string `json:"role"`
		Build struct {
			Kind string `json:"kind"`
			Dir  string `json:"dir"`
		} `json:"build"`
		Run struct {
			CWD string `json:"cwd"`
		} `json:"run"`
	} `json:"components"`
	Dependencies struct {
		Resources map[string]json.RawMessage `json:"resources"`
	} `json:"dependencies"`
}

// describeStackFacts reads declared component build metadata and resource
// declarations. It emits unknown values as evidence rather than guessing.
func describeStackFacts(root string) []*factsv1.GenericFact {
	path := filepath.Join(root, ".vrooli", "service.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return []*factsv1.GenericFact{{Id: "stack:manifest", Family: factsv1.FactFamily_FACT_FAMILY_STACK, Kind: "manifest", Subject: path, Evidence: []*factsv1.Evidence{evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_MISSING, "service.json is absent; stack declarations are unknown.", path)}}}
	}
	var manifest stackManifest
	if json.Unmarshal(data, &manifest) != nil {
		return []*factsv1.GenericFact{{Id: "stack:manifest", Family: factsv1.FactFamily_FACT_FAMILY_STACK, Kind: "manifest", Subject: path, Evidence: []*factsv1.Evidence{evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN, "service.json could not be parsed; stack declarations are unknown.", path)}}}
	}
	facts := make([]*factsv1.GenericFact, 0, len(manifest.Components)+len(manifest.Dependencies.Resources))
	ids := make([]string, 0, len(manifest.Components))
	for id := range manifest.Components {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		c := manifest.Components[id]
		dir := c.Build.Dir
		if dir == "" {
			dir = c.Run.CWD
		}
		if dir == "" {
			dir = "."
		}
		componentPath := filepath.Join(root, dir)
		attrs := map[string]string{"role": c.Role, "directory": dir}
		if c.Build.Kind != "" {
			attrs["build.kind"] = c.Build.Kind
		}
		language, framework, bundler := inspectComponentStack(componentPath)
		if language != "" {
			attrs["language"] = language
		}
		if framework != "" {
			attrs["ui_framework"] = framework
		}
		if bundler != "" {
			attrs["bundler"] = bundler
		}
		evidenceItems := []*factsv1.Evidence{evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, "Component role and directory are declared in service.json.", path)}
		if c.Build.Kind != "" {
			evidenceItems = append(evidenceItems, evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, "Component build.kind is declared in service.json.", path))
		} else {
			evidenceItems = append(evidenceItems, evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN, "Component build.kind is not declared.", path))
		}
		if language == "" {
			evidenceItems = append(evidenceItems, evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN, "Component language is not known from supported manifest files.", componentPath))
		}
		if framework == "" {
			evidenceItems = append(evidenceItems, evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN, "UI framework is not known from package metadata.", filepath.Join(componentPath, "package.json")))
		}
		if bundler == "" {
			evidenceItems = append(evidenceItems, evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN, "Bundler is not known from package metadata.", filepath.Join(componentPath, "package.json")))
		}
		facts = append(facts, &factsv1.GenericFact{Id: "stack:component:" + id, Family: factsv1.FactFamily_FACT_FAMILY_STACK, Kind: "component", Subject: id, Attributes: attrs, Evidence: evidenceItems})
	}
	resources := make([]string, 0, len(manifest.Dependencies.Resources))
	for id := range manifest.Dependencies.Resources {
		resources = append(resources, id)
	}
	sort.Strings(resources)
	for _, id := range resources {
		facts = append(facts, &factsv1.GenericFact{Id: "stack:resource:" + id, Family: factsv1.FactFamily_FACT_FAMILY_STACK, Kind: "resource", Subject: id, Attributes: map[string]string{"declared": "true"}, Evidence: []*factsv1.Evidence{evidence(factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN, "Resource is declared in service.json.", path)}})
	}
	return facts
}

func inspectComponentStack(root string) (language, framework, bundler string) {
	for _, candidate := range []struct{ name, language string }{{"go.mod", "go"}, {"Cargo.toml", "rust"}, {"pyproject.toml", "python"}, {"requirements.txt", "python"}, {"tsconfig.json", "typescript"}} {
		if fileExists(filepath.Join(root, candidate.name)) {
			language = candidate.language
			break
		}
	}
	packagePath := filepath.Join(root, "package.json")
	data, err := os.ReadFile(packagePath)
	if err != nil {
		return
	}
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return
	}
	deps := map[string]bool{}
	for name := range pkg.Dependencies {
		deps[name] = true
	}
	for name := range pkg.DevDependencies {
		deps[name] = true
	}
	for name, value := range map[string]string{"react": "react", "vue": "vue", "svelte": "svelte", "@angular/core": "angular"} {
		if deps[name] {
			framework = value
			break
		}
	}
	for name, value := range map[string]string{"vite": "vite", "webpack": "webpack", "esbuild": "esbuild", "parcel": "parcel"} {
		if deps[name] {
			bundler = value
			break
		}
	}
	if language == "" {
		if fileExists(filepath.Join(root, "tsconfig.json")) {
			language = "typescript"
		} else {
			language = "javascript"
		}
	}
	return
}
