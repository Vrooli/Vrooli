// Package manifestschema validates a scenario's .vrooli/service.json against
// the repository's canonical service.schema.json.
//
// Every other manifest-bearing target kind already has a validity rule —
// PACKAGE_MANIFEST_INVALID, RESOURCE_MANIFEST_INVALID, TOOL_MANIFEST_INVALID,
// SAFEGUARD_MANIFEST_INVALID, TEAM_MANIFEST_INVALID, DOCS_MANIFEST_INVALID.
// Scenario, the most numerous kind in the repository, had none: its 18 rules
// all parse service.json into map[string]any and hand-check individual shapes,
// so the document as a whole was never validated. That blind spot let 405
// schema violations accumulate across the scenario fleet unseen.
//
// This rule closes it by compiling the real schema and validating the real
// document, so drift between service.schema.json and what scenarios write
// becomes a reported finding instead of an accumulating silence.
package manifestschema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// maxReportedViolations bounds how many schema errors one manifest contributes.
// A single structural mistake can cascade into dozens of sibling errors, and a
// report that scrolls is a report nobody reads. The overflow is summarized
// rather than dropped silently.
const maxReportedViolations = 8

// schemaDirName and schemaFileName locate the canonical schema relative to the
// repository root discovered by walking up from the manifest under validation.
var (
	schemaRelParts = []string{".vrooli", "schemas"}
	schemaFileName = "service.schema.json"
	quotedProperty = regexp.MustCompile(`'[^']+'`)
)

type compiledSchema struct {
	schema *jsonschema.Schema
	err    error
}

var schemaCache sync.Map // schemas dir -> *compiledSchema

// ShouldCheck reports whether path names a scenario manifest this rule owns.
//
// Unlike the sibling shape rules, this one needs the repository on disk (the
// schema and everything it references), so synthetic single-segment doc-test
// paths are deliberately excluded rather than silently passing.
func ShouldCheck(path string) bool {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return false
	}
	if !strings.EqualFold(filepath.Base(trimmed), "service.json") {
		return false
	}
	normalized := filepath.ToSlash(trimmed)
	return strings.Contains(normalized, "/scenarios/") || strings.HasPrefix(normalized, "scenarios/")
}

// CheckServiceManifestSchema validates content against service.schema.json.
func CheckServiceManifestSchema(content []byte, filePath string) []Violation {
	if !ShouldCheck(filePath) {
		return nil
	}

	messages, err := ValidationMessages(content, filePath)
	if err != nil {
		return []Violation{newViolation(filePath, 1, err.Error())}
	}

	source := string(content)
	out := make([]Violation, 0, len(messages))
	for index, message := range messages {
		if index == maxReportedViolations {
			out = append(out, newViolation(filePath, 1, fmt.Sprintf(
				"service.json has %d more schema violations not listed here", len(messages)-maxReportedViolations)))
			break
		}
		out = append(out, newViolation(filePath, lineForPointer(source, message), message))
	}
	return out
}

// ValidationMessages returns every canonical-schema violation for a scenario
// manifest without applying the UI-facing finding cap used by
// CheckServiceManifestSchema. Fleet census evidence needs the complete count:
// truncating one noisy manifest would make the repository total irreproducible.
//
// Infrastructure failures (an unreachable or uncompilable schema) are returned
// as errors because no manifest can be honestly graded in that state. Malformed
// manifest JSON is a document violation and is therefore returned as one
// message, allowing the census to identify the offending manifest.
func ValidationMessages(content []byte, filePath string) ([]string, error) {
	schemaDir, ok := findSchemaDir(filePath)
	if !ok {
		return nil, fmt.Errorf("cannot locate .vrooli/schemas/service.schema.json above this manifest; scenario manifests cannot be validated")
	}

	schema, err := loadSchema(schemaDir)
	if err != nil {
		// A schema that will not compile is the failure mode that hid this
		// entire class of drift: a broken $ref made every standards-compliant
		// validator bail, so nothing validated and nothing complained.
		return nil, fmt.Errorf("service.schema.json does not compile, so no scenario manifest can be validated: %w", err)
	}

	var document any
	if err := json.Unmarshal(content, &document); err != nil {
		return []string{fmt.Sprintf("service.json is not valid JSON: %v", err)}, nil
	}

	err = schema.Validate(document)
	if err == nil {
		return nil, nil
	}
	validationErr, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return []string{fmt.Sprintf("service.json failed schema validation: %v", err)}, nil
	}

	messages := leafMessages(validationErr)
	sort.Strings(messages)
	return messages, nil
}

// leafMessages flattens a ValidationError tree to its leaves. The root node
// only ever says "doesn't validate with ..."; the actionable text — which
// property, which constraint — lives at the leaves.
func leafMessages(err *jsonschema.ValidationError) []string {
	if err == nil {
		return nil
	}
	if len(err.Causes) == 0 {
		location := strings.TrimSpace(err.InstanceLocation)
		if location == "" {
			location = "/"
		}
		return []string{fmt.Sprintf("%s: %s", location, normalizeValidationMessage(err.Message))}
	}
	seen := make(map[string]struct{})
	var out []string
	for _, cause := range err.Causes {
		for _, message := range leafMessages(cause) {
			if _, exists := seen[message]; exists {
				continue
			}
			seen[message] = struct{}{}
			out = append(out, message)
		}
	}
	return out
}

// normalizeValidationMessage removes the jsonschema library's dependence on
// Go map iteration order. additionalProperties and required-property failures
// can name more than one property; their order has no semantic meaning, but it
// must be stable for census evidence to be byte-reproducible.
func normalizeValidationMessage(message string) string {
	properties := quotedProperty.FindAllString(message, -1)
	if len(properties) < 2 {
		return message
	}
	sort.Strings(properties)
	switch {
	case strings.HasPrefix(message, "additionalProperties ") && strings.HasSuffix(message, " not allowed"):
		return "additionalProperties " + strings.Join(properties, ", ") + " not allowed"
	case strings.HasPrefix(message, "missing properties: "):
		return "missing properties: " + strings.Join(properties, ", ")
	default:
		return message
	}
}

// lineForPointer makes a finding clickable by pointing at the offending key
// rather than at line 1. It resolves the last segment of the JSON pointer that
// prefixes message; a pointer that cannot be located falls back to line 1,
// which is honest about not knowing rather than guessing a nearby line.
func lineForPointer(source, message string) int {
	pointer, _, found := strings.Cut(message, ": ")
	if !found {
		return 1
	}
	segments := strings.Split(strings.Trim(pointer, "/"), "/")
	key := ""
	for index := len(segments) - 1; index >= 0; index-- {
		segment := strings.TrimSpace(segments[index])
		if segment == "" || isIndex(segment) {
			continue
		}
		key = segment
		break
	}
	if key == "" {
		return 1
	}
	offset := strings.Index(source, `"`+key+`"`)
	if offset < 0 {
		return 1
	}
	return strings.Count(source[:offset], "\n") + 1
}

func isIndex(segment string) bool {
	for _, r := range segment {
		if r < '0' || r > '9' {
			return false
		}
	}
	return segment != ""
}

// findSchemaDir walks up from the manifest looking for the repository's schema
// directory. The scenario's own .vrooli directory has no schemas/ child, so the
// walk naturally skips it and stops at the repository root.
func findSchemaDir(manifestPath string) (string, bool) {
	dir := filepath.Dir(manifestPath)
	if !filepath.IsAbs(dir) {
		if abs, err := filepath.Abs(dir); err == nil {
			dir = abs
		}
	}
	for {
		candidate := filepath.Join(append([]string{dir}, schemaRelParts...)...)
		if _, err := os.Stat(filepath.Join(candidate, schemaFileName)); err == nil {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// loadSchema compiles service.schema.json once per schema directory.
//
// Every sibling schema is registered under its canonical $id base and its bare
// filename — and deliberately nothing else. An extra alias rooted one path
// segment higher used to be registered elsewhere in the repository, which
// silently absorbed a generated parent-relative $ref and made a broken schema
// look healthy. Registering only real identities keeps a bad ref loud.
func loadSchema(schemaDir string) (*jsonschema.Schema, error) {
	if cached, ok := schemaCache.Load(schemaDir); ok {
		entry := cached.(*compiledSchema)
		return entry.schema, entry.err
	}

	entry := &compiledSchema{}
	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		entry.err = fmt.Errorf("read schema directory: %w", err)
		schemaCache.Store(schemaDir, entry)
		return nil, entry.err
	}

	compiler := jsonschema.NewCompiler()
	for _, item := range entries {
		if item.IsDir() || !strings.HasSuffix(item.Name(), ".json") {
			continue
		}
		raw, readErr := os.ReadFile(filepath.Join(schemaDir, item.Name())) // #nosec G304 -- enumerated from the repository schema directory.
		if readErr != nil {
			entry.err = fmt.Errorf("read %s: %w", item.Name(), readErr)
			schemaCache.Store(schemaDir, entry)
			return nil, entry.err
		}
		for _, id := range []string{item.Name(), "https://vrooli.com/schemas/" + item.Name()} {
			if addErr := compiler.AddResource(id, bytes.NewReader(raw)); addErr != nil {
				entry.err = fmt.Errorf("register %s: %w", item.Name(), addErr)
				schemaCache.Store(schemaDir, entry)
				return nil, entry.err
			}
		}
	}

	schema, compileErr := compiler.Compile(schemaFileName)
	if compileErr != nil {
		entry.err = fmt.Errorf("compile %s: %w", schemaFileName, compileErr)
		schemaCache.Store(schemaDir, entry)
		return nil, entry.err
	}
	entry.schema = schema
	schemaCache.Store(schemaDir, entry)
	return schema, nil
}
