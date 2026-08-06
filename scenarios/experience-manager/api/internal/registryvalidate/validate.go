// Package registryvalidate validates the experience capability registry SSOT.
package registryvalidate

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type Finding struct{ Code, Severity, Location, Message string }

type Validator struct {
	RepoRoot string
	once     sync.Once
	schema   *jsonschema.Schema
	err      error
}

func New(repoRoot string) *Validator { return &Validator{RepoRoot: repoRoot} }

func (v *Validator) Validate() ([]Finding, error) {
	if err := v.compile(); err != nil {
		return nil, err
	}
	dir := filepath.Join(v.RepoRoot, "scenarios", "experience-manager", "capabilities")
	paths := []string{filepath.Join(dir, "index.json"), filepath.Join(dir, "states.json"), filepath.Join(dir, "axes.json"), filepath.Join(dir, "evidence.json")}
	groups, err := filepath.Glob(filepath.Join(dir, "capabilities", "*.json"))
	if err != nil {
		return nil, err
	}
	paths = append(paths, groups...)
	sort.Strings(paths)
	var findings []Finding
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		var document any
		if err := json.Unmarshal(data, &document); err != nil {
			findings = append(findings, Finding{"registry.document_parse_error", "error", rel(v.RepoRoot, path), err.Error()})
			continue
		}
		if err := v.schema.Validate(document); err != nil {
			findings = append(findings, schemaFindings(rel(v.RepoRoot, path), err)...)
		}
	}
	findings = append(findings, crossFindings(dir)...)
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Location != findings[j].Location {
			return findings[i].Location < findings[j].Location
		}
		if findings[i].Code != findings[j].Code {
			return findings[i].Code < findings[j].Code
		}
		return findings[i].Message < findings[j].Message
	})
	return findings, nil
}

func (v *Validator) compile() error {
	v.once.Do(func() {
		data, err := os.ReadFile(filepath.Join(v.RepoRoot, ".vrooli", "schemas", "experience-capability-registry.schema.json"))
		if err != nil {
			v.err = fmt.Errorf("read registry schema: %w", err)
			return
		}
		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource("experience-capability-registry.schema.json", bytes.NewReader(data)); err != nil {
			v.err = err
			return
		}
		v.schema, v.err = compiler.Compile("experience-capability-registry.schema.json")
		if v.err != nil {
			v.err = fmt.Errorf("compile registry schema: %w", v.err)
		}
	})
	return v.err
}

func schemaFindings(location string, err error) []Finding {
	var validation *jsonschema.ValidationError
	if !errors.As(err, &validation) {
		return []Finding{{"registry.schema_error", "error", location, err.Error()}}
	}
	if len(validation.Causes) == 0 {
		return []Finding{{"registry.schema_error", "error", location, validation.Message}}
	}
	var out []Finding
	for _, cause := range validation.Causes {
		out = append(out, schemaFindings(location, cause)...)
	}
	return out
}

func crossFindings(dir string) []Finding {
	axisValues := map[string]map[string]bool{}
	evidence := map[string]bool{}
	states := map[string]bool{}
	var axes struct {
		Axes []struct {
			ID     string `json:"id"`
			Values []struct {
				ID string `json:"id"`
			} `json:"values"`
		} `json:"axes"`
	}
	if data, err := os.ReadFile(filepath.Join(dir, "axes.json")); err == nil && json.Unmarshal(data, &axes) == nil {
		for _, axis := range axes.Axes {
			axisValues[axis.ID] = map[string]bool{}
			for _, value := range axis.Values {
				axisValues[axis.ID][value.ID] = true
			}
		}
	}
	var evidenceDoc struct {
		Evidence []struct {
			ID string `json:"id"`
		} `json:"evidence"`
	}
	if data, err := os.ReadFile(filepath.Join(dir, "evidence.json")); err == nil && json.Unmarshal(data, &evidenceDoc) == nil {
		for _, e := range evidenceDoc.Evidence {
			evidence[e.ID] = true
		}
	}
	var stateDoc struct {
		Canonical []struct {
			ID string `json:"id"`
		} `json:"canonical"`
		Views map[string]struct {
			States []string `json:"states"`
		} `json:"views"`
	}
	if data, err := os.ReadFile(filepath.Join(dir, "states.json")); err == nil && json.Unmarshal(data, &stateDoc) == nil {
		for _, s := range stateDoc.Canonical {
			states[s.ID] = true
		}
		for view, doc := range stateDoc.Views {
			for _, id := range doc.States {
				if !states[id] {
					return []Finding{{"registry.state_reference", "error", "capabilities/states.json", fmt.Sprintf("view %q references undeclared state %q", view, id)}}
				}
			}
		}
	}
	paths, _ := filepath.Glob(filepath.Join(dir, "capabilities", "*.json"))
	var findings []Finding
	seen := map[string]string{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var doc struct {
			Capabilities []struct {
				ID     string   `json:"id"`
				Facets []string `json:"facets"`
				Proves *struct {
					Axes     map[string][]string `json:"axes"`
					Evidence []string            `json:"evidence"`
				} `json:"proves"`
			} `json:"capabilities"`
		}
		if json.Unmarshal(data, &doc) != nil {
			continue
		}
		for _, item := range doc.Capabilities {
			if previous, ok := seen[item.ID]; ok {
				findings = append(findings, Finding{"registry.duplicate_capability", "error", rel(dir, path), fmt.Sprintf("capability %q is also declared in %s", item.ID, previous)})
			}
			seen[item.ID] = rel(dir, path)
			if len(item.Facets) == 0 {
				findings = append(findings, Finding{"registry.capability_facets", "error", rel(dir, path), fmt.Sprintf("capability %q declares no facet", item.ID)})
			}
			if item.Proves == nil {
				continue
			}
			for axis, values := range item.Proves.Axes {
				known := axisValues[axis]
				if known == nil {
					findings = append(findings, Finding{"registry.axis_reference", "error", rel(dir, path), fmt.Sprintf("capability %q references undeclared axis %q", item.ID, axis)})
					continue
				}
				for _, value := range values {
					if !known[value] {
						findings = append(findings, Finding{"registry.axis_value_reference", "error", rel(dir, path), fmt.Sprintf("capability %q references undeclared value %q on axis %q", item.ID, value, axis)})
					}
				}
			}
			for _, id := range item.Proves.Evidence {
				if !evidence[id] {
					findings = append(findings, Finding{"registry.evidence_reference", "error", rel(dir, path), fmt.Sprintf("capability %q references undeclared evidence %q", item.ID, id)})
				}
			}
		}
	}
	return findings
}

func rel(root, path string) string {
	value, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(value)
}
