// Package catalogvalidate validates the catalog SSOT where it is authored.
// It reports authored-document defects as findings and returns an error only
// when the validator itself cannot be constructed or cannot read its inputs.
package catalogvalidate

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"react-component-library/internal/gates"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type Finding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Location string `json:"location"`
	Message  string `json:"message"`
}

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
	catalogDir := filepath.Join(v.RepoRoot, "scenarios", "react-component-library", "catalog")
	paths, err := catalogPaths(catalogDir)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("read %s: %w", path, readErr)
		}
		var document any
		if err := json.Unmarshal(data, &document); err != nil {
			findings = append(findings, Finding{Code: "catalog.document_parse_error", Severity: "error", Location: rel(v.RepoRoot, path), Message: err.Error()})
			continue
		}
		if err := v.schema.Validate(document); err != nil {
			findings = append(findings, schemaFindings(rel(v.RepoRoot, path), err)...)
		}
	}
	findings = append(findings, crossRegistryFindings(v.RepoRoot, catalogDir)...)
	findings = append(findings, blockingRunnerFindings(v.RepoRoot, catalogDir)...)
	findings = append(findings, implementationIdentityFindings(v.RepoRoot)...)
	findings = append(findings, vacuousAllowlistFindings(v.RepoRoot)...)
	findings = append(findings, tautologicalAllowlistFindings(v.RepoRoot)...)
	findings = append(findings, storyCoverageAllowlistFindings(v.RepoRoot)...)
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

// blockingRunnerFindings keeps the catalog's blocking contract honest at the
// boundary where declarations are loaded. A declaration-only gate may remain
// advisory, but it cannot claim to block a release when the executable gate
// registry has no in-process runner for it.
func blockingRunnerFindings(repoRoot, catalogDir string) []Finding {
	data, err := os.ReadFile(filepath.Join(catalogDir, "config.json"))
	if err != nil {
		return nil
	}
	var config struct {
		Gates []struct {
			ID       string `json:"id"`
			Blocking bool   `json:"blocking"`
		} `json:"gates"`
	}
	if json.Unmarshal(data, &config) != nil {
		return nil
	}
	var findings []Finding
	for _, gate := range config.Gates {
		definition, ok := gates.Lookup(gate.ID)
		if gate.Blocking && (!ok || definition.Run == nil) {
			findings = append(findings, Finding{
				Code: "catalog.blocking_gate_without_runner", Severity: "error",
				Location: rel(repoRoot, filepath.Join(catalogDir, "config.json")),
				Message:  fmt.Sprintf("blocking gate %q has no in-process runner; add one or set blocking to false and document the advisory gate", gate.ID),
			})
		}
	}
	return findings
}

// implementationIdentityFindings closes the join between authored library
// manifests and the desired-state catalog. An implementation may be outside
// the catalog, but only when that choice is explicit in its manifest.
func implementationIdentityFindings(repoRoot string) []Finding {
	libraryRoot := filepath.Join(repoRoot, "scenarios", "react-component-library", "library")
	paths, _ := filepath.Glob(filepath.Join(libraryRoot, "*", "*", "component.json"))
	var findings []Finding
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			findings = append(findings, Finding{Code: "catalog.implementation_manifest_unreadable", Severity: "error", Location: rel(repoRoot, path), Message: err.Error()})
			continue
		}
		var manifest struct {
			CatalogID    *string `json:"catalogId"`
			Supplemental bool    `json:"supplemental"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			continue
		}
		if manifest.CatalogID != nil && strings.TrimSpace(*manifest.CatalogID) != "" {
			continue
		}
		if manifest.Supplemental {
			continue
		}
		findings = append(findings, Finding{
			Code:     "catalog.implementation_identity_missing",
			Severity: "error",
			Location: rel(repoRoot, path),
			Message:  "implementation must declare a catalogId or supplemental=true",
		})
	}
	return findings
}

func (v *Validator) compile() error {
	v.once.Do(func() {
		path := filepath.Join(v.RepoRoot, ".vrooli", "schemas", "catalog-asset.schema.json")
		data, err := os.ReadFile(path)
		if err != nil {
			v.err = fmt.Errorf("read catalog schema: %w", err)
			return
		}
		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource("catalog-asset.schema.json", bytes.NewReader(data)); err != nil {
			v.err = fmt.Errorf("add catalog schema: %w", err)
			return
		}
		v.schema, v.err = compiler.Compile("catalog-asset.schema.json")
		if v.err != nil {
			v.err = fmt.Errorf("compile catalog schema: %w", v.err)
		}
	})
	return v.err
}

func catalogPaths(dir string) ([]string, error) {
	paths := []string{filepath.Join(dir, "config.json")}
	assets, err := filepath.Glob(filepath.Join(dir, "assets", "*", "*.json"))
	if err != nil {
		return nil, fmt.Errorf("scan catalog assets: %w", err)
	}
	paths = append(paths, assets...)
	// The legacy directory is an archival compatibility surface, not part of
	// the catalog SSOT. Its historical pointer documents intentionally use a
	// different schema and must not be mistaken for current asset declarations.
	filtered := paths[:1]
	for _, path := range paths[1:] {
		if filepath.Base(filepath.Dir(path)) == "legacy" {
			continue
		}
		filtered = append(filtered, path)
	}
	paths = filtered
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("catalog input %s: %w", path, err)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func schemaFindings(location string, err error) []Finding {
	var validation *jsonschema.ValidationError
	if !errors.As(err, &validation) {
		return []Finding{{Code: "catalog.schema_error", Severity: "error", Location: location, Message: err.Error()}}
	}
	if len(validation.Causes) == 0 {
		return []Finding{{Code: "catalog.schema_error", Severity: "error", Location: location, Message: validation.Message}}
	}
	var out []Finding
	for _, cause := range validation.Causes {
		out = append(out, schemaFindings(location, cause)...)
	}
	return out
}

func rel(root, path string) string {
	value, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(value)
}
