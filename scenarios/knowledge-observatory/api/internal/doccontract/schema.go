package doccontract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	repocontract "github.com/vrooli/repo-contract-go"
)

// docsManifestSchemaName is the canonical filename of the docs-manifest
// JSON Schema in .vrooli/schemas/. Mirrors the schema's $id "scenario-docs-manifest/v2".
const docsManifestSchemaName = "scenario-docs-manifest.schema.json"

// validateAgainstSchema runs JSON Schema validation against the docs manifest
// at the given path. It is a best-effort step: if the schema file cannot be
// located (e.g., this binary is running outside a repo checkout), no findings
// are produced and the caller's imperative Go rules in ValidateManifest take
// full responsibility. When the schema is found and validation fails, each
// violation is surfaced as a Finding with Code "schema_violation".
func validateAgainstSchema(manifestPath string) []Finding {
	repoRoot := repoRootFromManifest(manifestPath)
	if repoRoot == "" {
		return nil
	}
	schemaPath, err := repocontract.SchemaPath(repoRoot, docsManifestSchemaName)
	if err != nil {
		return nil
	}
	if _, err := os.Stat(schemaPath); err != nil {
		return nil
	}
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(docsManifestSchemaName, bytes.NewReader(schemaBytes)); err != nil {
		return []Finding{{Code: "schema_load_error", Severity: "error", Path: filepath.ToSlash(manifestPath), Message: fmt.Sprintf("failed to load docs-manifest schema: %v", err)}}
	}
	schema, err := compiler.Compile(docsManifestSchemaName)
	if err != nil {
		return []Finding{{Code: "schema_load_error", Severity: "error", Path: filepath.ToSlash(manifestPath), Message: fmt.Sprintf("failed to compile docs-manifest schema: %v", err)}}
	}

	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil // imperative rules will surface read errors via their own paths
	}
	var doc any
	if err := json.Unmarshal(manifestBytes, &doc); err != nil {
		return []Finding{{Code: "schema_violation", Severity: "error", Path: filepath.ToSlash(manifestPath), Message: fmt.Sprintf("manifest is not valid JSON: %v", err)}}
	}
	if err := schema.Validate(doc); err != nil {
		return flattenSchemaError(manifestPath, err)
	}
	return nil
}

func flattenSchemaError(manifestPath string, err error) []Finding {
	var findings []Finding
	if ve, ok := err.(*jsonschema.ValidationError); ok {
		walkValidationError(filepath.ToSlash(manifestPath), ve, &findings)
	} else {
		findings = append(findings, Finding{Code: "schema_violation", Severity: "error", Path: filepath.ToSlash(manifestPath), Message: err.Error()})
	}
	sort.Slice(findings, func(i, j int) bool { return findings[i].Message < findings[j].Message })
	return findings
}

func walkValidationError(manifestPath string, ve *jsonschema.ValidationError, out *[]Finding) {
	if len(ve.Causes) == 0 {
		loc := ve.InstanceLocation
		if loc == "" {
			loc = manifestPath
		} else {
			loc = manifestPath + loc
		}
		*out = append(*out, Finding{
			Code:     "schema_violation",
			Severity: "error",
			Path:     loc,
			Message:  ve.Message,
		})
		return
	}
	for _, cause := range ve.Causes {
		walkValidationError(manifestPath, cause, out)
	}
}

func repoRootFromManifest(manifestPath string) string {
	dir := filepath.Clean(manifestPath)
	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		dir = filepath.Dir(dir)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".vrooli", "repo-contract.json")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir || strings.TrimSpace(parent) == "" {
			return ""
		}
		dir = parent
	}
}
