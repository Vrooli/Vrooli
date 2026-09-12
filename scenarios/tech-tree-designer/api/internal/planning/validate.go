package planning

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/bufbuild/protocompile"
)

type CompilerValidator struct {
	SchemasRoot string
}

func NewCompilerValidator(schemasRoot string) *CompilerValidator {
	return &CompilerValidator{SchemasRoot: schemasRoot}
}

var _ ProtoValidator = (*CompilerValidator)(nil)

func (v *CompilerValidator) Validate(ctx context.Context, scenario Scenario) ([]PlanFinding, error) {
	schemasRoot, err := resolveSchemasRoot(v.SchemasRoot)
	if err != nil {
		return nil, err
	}
	tmp, err := os.MkdirTemp("", "tech-tree-designer-planned-protos-*")
	if err != nil {
		return nil, fmt.Errorf("create planned proto overlay: %w", err)
	}
	defer os.RemoveAll(tmp)

	var findings []PlanFinding
	targets := make([]string, 0, len(scenario.Files))
	plannedPaths := map[string]struct{}{}
	for _, file := range scenario.Files {
		path, err := NormalizeProtoPath(file.Path)
		if err != nil {
			findings = append(findings, errorFinding("invalid_path", file.Path, err.Error(), "Use a relative .proto path inside the planned scenario tree."))
			continue
		}
		plannedPaths[path] = struct{}{}
		targets = append(targets, path)
		dst := filepath.Join(tmp, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return nil, fmt.Errorf("prepare planned proto path: %w", err)
		}
		if err := os.WriteFile(dst, []byte(file.Text), 0o644); err != nil {
			return nil, fmt.Errorf("write planned proto overlay: %w", err)
		}
		findings = append(findings, validateTextConventions(file, plannedPaths, schemasRoot)...)
	}
	sort.Strings(targets)
	if len(targets) == 0 {
		findings = append(findings, errorFinding("no_files", scenario.Slug, "planned scenario has no proto files", "Add at least one .proto file before validating."))
		return findings, nil
	}

	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{ImportPaths: []string{tmp, schemasRoot}}),
	}
	if _, err := compiler.Compile(ctx, targets...); err != nil {
		findings = append(findings, errorFinding("compile_failed", scenario.Slug, err.Error(), "Fix parser, import, package, and descriptor errors before materializing."))
	}
	return findings, nil
}

func validateTextConventions(file ProtoFile, plannedPaths map[string]struct{}, schemasRoot string) []PlanFinding {
	var findings []PlanFinding
	location := file.Path
	if !strings.Contains(file.Text, "@stability") {
		findings = append(findings, errorFinding("missing_stability", location, "planned proto file is missing @stability", "Add a file comment containing @stability experimental."))
	} else if !regexp.MustCompile(`@stability\s+experimental\b`).MatchString(file.Text) {
		findings = append(findings, errorFinding("planned_stability_not_experimental", location, "planned proto stability must be experimental", "Use @stability experimental until the scenario is implemented."))
	}
	for _, imp := range extractImports(file.Text) {
		if _, ok := plannedPaths[imp]; ok {
			continue
		}
		if _, err := os.Stat(filepath.Join(schemasRoot, filepath.FromSlash(imp))); err != nil {
			findings = append(findings, errorFinding("unresolved_import", location, fmt.Sprintf("import %q does not resolve to a live or planned proto", imp), "Import a checked-in package proto path or add the imported file to the plan."))
		}
	}
	for _, name := range extractMessageNames(file.Text) {
		if !isPascalCase(name) {
			findings = append(findings, errorFinding("message_name_not_pascal_case", location, fmt.Sprintf("message %q is not PascalCase", name), "Use PascalCase message names."))
		}
	}
	for _, field := range extractFieldNames(file.Text) {
		if !isSnakeCase(field) {
			findings = append(findings, errorFinding("field_name_not_snake_case", location, fmt.Sprintf("field %q is not snake_case", field), "Use snake_case field names."))
		}
		if strings.Contains(field, "id") && field != "id" && !strings.HasSuffix(field, "_id") {
			findings = append(findings, warningFinding("field_id_suffix", location, fmt.Sprintf("field %q contains id without _id suffix", field), "Use id or a *_id suffix for identifiers."))
		}
		if strings.Contains(field, "time") && !strings.HasSuffix(field, "_at") {
			findings = append(findings, warningFinding("field_timestamp_suffix", location, fmt.Sprintf("field %q looks timestamp-like without _at suffix", field), "Use *_at suffixes for timestamp fields."))
		}
	}
	return findings
}

var (
	importRe  = regexp.MustCompile(`(?m)^\s*import\s+(?:public\s+|weak\s+)?\"([^\"]+)\"\s*;`)
	messageRe = regexp.MustCompile(`(?m)^\s*message\s+([A-Za-z_][A-Za-z0-9_]*)\s*\{`)
	fieldRe   = regexp.MustCompile(`(?m)^\s*(?:optional\s+|repeated\s+|map\s*<[^>]+>\s+)?(?:[A-Za-z_][A-Za-z0-9_.]*|double|float|int32|int64|uint32|uint64|sint32|sint64|fixed32|fixed64|sfixed32|sfixed64|bool|string|bytes)\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*\d+`)
)

func extractImports(text string) []string {
	matches := importRe.FindAllStringSubmatch(text, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m[1])
	}
	return out
}

func extractMessageNames(text string) []string {
	matches := messageRe.FindAllStringSubmatch(text, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m[1])
	}
	return out
}

func extractFieldNames(text string) []string {
	matches := fieldRe.FindAllStringSubmatch(text, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m[1])
	}
	return out
}

func isPascalCase(value string) bool {
	if value == "" || strings.Contains(value, "_") {
		return false
	}
	first := []rune(value)[0]
	return unicode.IsUpper(first)
}

func isSnakeCase(value string) bool {
	if value == "" || strings.HasPrefix(value, "_") || strings.HasSuffix(value, "_") || strings.Contains(value, "__") {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return false
	}
	return true
}
