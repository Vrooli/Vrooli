package manifestvalidation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/santhosh-tekuri/jsonschema/v5"
	repocontract "github.com/vrooli/repo-contract-go"
)

const cliManifestSchemaFile = "cli-manifest.schema.json"

// JSONSchemaValidator compiles .vrooli/schemas/cli-manifest.schema.json
// lazily on first use and re-uses the compiled schema. It is safe for
// concurrent reads after Validate has been called once.
type JSONSchemaValidator struct {
	RepoRoot string
	schema   *jsonschema.Schema
}

// NewJSONSchemaValidator returns a validator rooted at the given repo dir.
func NewJSONSchemaValidator(repoRoot string) *JSONSchemaValidator {
	return &JSONSchemaValidator{RepoRoot: repoRoot}
}

// Validate JSON-schema-validates raw manifest bytes. Returns one Finding per
// schema violation. The (findings, nil) shape is reserved for "ran cleanly,
// here are the violations." Returning (nil, err) means the validator itself
// could not run (schema file missing, parse error) — a setup problem, not a
// validation result.
func (v *JSONSchemaValidator) Validate(_ context.Context, raw []byte) ([]Finding, error) {
	if v.schema == nil {
		schemaPath, err := repocontract.SchemaPath(v.RepoRoot, cliManifestSchemaFile)
		if err != nil {
			return nil, fmt.Errorf("resolve schema path: %w", err)
		}
		schemaBytes, err := os.ReadFile(schemaPath)
		if err != nil {
			return nil, fmt.Errorf("read schema %s: %w", schemaPath, err)
		}
		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource(cliManifestSchemaFile, bytes.NewReader(schemaBytes)); err != nil {
			return nil, fmt.Errorf("add schema resource: %w", err)
		}
		compiled, err := compiler.Compile(cliManifestSchemaFile)
		if err != nil {
			return nil, fmt.Errorf("compile schema: %w", err)
		}
		v.schema = compiled
	}

	var doc any
	manifestRel := cliManifestRel(v.RepoRoot)
	if err := json.Unmarshal(raw, &doc); err != nil {
		return []Finding{{
			Severity: SeverityError,
			Code:     CodeManifestParseError,
			Location: manifestRel,
			Message:  fmt.Sprintf("manifest is not valid JSON: %v", err),
		}}, nil
	}

	if err := v.schema.Validate(doc); err != nil {
		return schemaErrToFindings(err, manifestRel), nil
	}
	return nil, nil
}

// schemaErrToFindings flattens a jsonschema.ValidationError tree into a flat
// list of error-severity findings. Each leaf becomes one finding so callers
// see every distinct violation rather than only the root.
func schemaErrToFindings(err error, manifestRel string) []Finding {
	var ve *jsonschema.ValidationError
	if !errors.As(err, &ve) {
		return []Finding{{
			Severity: SeverityError,
			Code:     CodeManifestSchemaError,
			Location: manifestRel,
			Message:  err.Error(),
		}}
	}
	leaves := flattenSchemaErrors(ve)
	sort.Slice(leaves, func(i, j int) bool {
		if leaves[i].InstanceLocation != leaves[j].InstanceLocation {
			return leaves[i].InstanceLocation < leaves[j].InstanceLocation
		}
		return leaves[i].Message < leaves[j].Message
	})
	findings := make([]Finding, 0, len(leaves))
	for _, leaf := range leaves {
		loc := leaf.InstanceLocation
		if loc == "" {
			loc = manifestRel
		} else {
			loc = manifestRel + "#" + loc
		}
		findings = append(findings, Finding{
			Severity: SeverityError,
			Code:     CodeManifestSchemaError,
			Location: loc,
			Message:  leaf.Message,
		})
	}
	return findings
}

func cliManifestRel(repoRoot string) string {
	rel, _ := repocontract.ScenarioCLIManifestRel(repoRoot)
	return rel
}

type schemaLeaf struct {
	InstanceLocation string
	Message          string
}

func flattenSchemaErrors(ve *jsonschema.ValidationError) []schemaLeaf {
	if len(ve.Causes) == 0 {
		return []schemaLeaf{{InstanceLocation: ve.InstanceLocation, Message: ve.Message}}
	}
	var out []schemaLeaf
	for _, c := range ve.Causes {
		out = append(out, flattenSchemaErrors(c)...)
	}
	return out
}
