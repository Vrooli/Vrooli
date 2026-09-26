package validation

import (
	"encoding/json"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/maturity-go/autofix"
)

// FixRegistry owns API Health's deterministic, local-only repairs.
type FixRegistry struct {
	registry *autofix.Registry
}

func NewFixRegistry() *FixRegistry {
	return &FixRegistry{registry: autofix.NewRegistry(
		autofix.Fixer{RuleID: CodeServiceHealthMissing, Preview: previewServiceHealthFix, CanFix: fileExistsCanFix(".vrooli/service.json")},
		autofix.Fixer{RuleID: CodeHealthEndpointMissing, Preview: previewEndpointHealthFix, CanFix: fileExistsCanFix(".vrooli/endpoints.json")},
		autofix.Fixer{RuleID: CodeRawStatusCode, Preview: previewRawStatusFix, CanFix: apiSourceCanFix},
		autofix.Fixer{RuleID: CodeContentTypeMissing, Preview: previewJSONContentTypeFix, CanFix: apiSourceCanFix},
		autofix.Fixer{RuleID: CodeResponseBodyUnclosed, Preview: previewResponseBodyCloseFix, CanFix: apiSourceCanFix},
	)}
}

func (r *FixRegistry) Preview(root string, ruleIDs []string) ([]autofix.Candidate, error) {
	return r.registry.Preview(root, ruleIDs)
}

// Apply previews and writes one rule at a time so same-file fixers compose.
func (r *FixRegistry) Apply(root string, ruleIDs []string) ([]autofix.Candidate, error) {
	if len(ruleIDs) == 0 {
		ruleIDs = []string{CodeServiceHealthMissing, CodeHealthEndpointMissing, CodeRawStatusCode, CodeContentTypeMissing, CodeResponseBodyUnclosed}
	}
	var applied []autofix.Candidate
	for _, ruleID := range ruleIDs {
		candidates, err := r.registry.Apply(root, []string{ruleID})
		if err != nil {
			return applied, err
		}
		applied = append(applied, candidates...)
	}
	return applied, nil
}

func previewServiceHealthFix(root string) ([]autofix.Candidate, error) {
	path := filepath.Join(root, ".vrooli", "service.json")
	beforeBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}
	before := string(beforeBytes)
	var doc map[string]any
	if err := json.Unmarshal(beforeBytes, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	ports := ensureMap(doc, "ports")
	if _, ok := ports["api"]; !ok {
		ports["api"] = map[string]any{}
	}
	lifecycle := ensureMap(doc, "lifecycle")
	health := ensureMap(lifecycle, "health")
	endpoints := ensureMap(health, "endpoints")
	endpoints["api"] = "/health"
	checks, _ := health["checks"].([]any)
	if !hasAPIHealthCheck(checks) {
		checks = append(checks, map[string]any{
			"name":   "api_endpoint",
			"type":   "http",
			"target": "http://localhost:${API_PORT}/health",
		})
		health["checks"] = checks
	}
	afterBytes, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	after := string(afterBytes) + "\n"
	if before == after {
		return nil, nil
	}
	return []autofix.Candidate{{
		RuleID:      CodeServiceHealthMissing,
		FilePath:    path,
		Description: "Normalize API lifecycle health metadata to ports.api and /health.",
		Before:      before,
		After:       after,
	}}, nil
}

func previewEndpointHealthFix(root string) ([]autofix.Candidate, error) {
	path := filepath.Join(root, ".vrooli", "endpoints.json")
	beforeBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}
	before := string(beforeBytes)
	var doc struct {
		Endpoints []map[string]any `json:"endpoints"`
	}
	if err := json.Unmarshal(beforeBytes, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	for _, ep := range doc.Endpoints {
		if ep["path"] == "/health" {
			return nil, nil
		}
	}
	doc.Endpoints = append([]map[string]any{{
		"id":             "health",
		"path":           "/health",
		"method":         "GET",
		"summary":        "Health check",
		"description":    "Standard API health readiness endpoint.",
		"category":       "system",
		"rest_exception": true,
	}}, doc.Endpoints...)
	afterBytes, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	after := string(afterBytes) + "\n"
	return []autofix.Candidate{{
		RuleID:      CodeHealthEndpointMissing,
		FilePath:    path,
		Description: "Add the standard /health endpoint descriptor.",
		Before:      before,
		After:       after,
	}}, nil
}

func previewRawStatusFix(root string) ([]autofix.Candidate, error) {
	return rewriteProductionGoFiles(root, CodeRawStatusCode, "Replace raw HTTP status literals with net/http constants.", func(path, before string) (string, bool, error) {
		after := before
		keys := make([]string, 0, len(statusConstantByCode))
		for code := range statusConstantByCode {
			keys = append(keys, code)
		}
		sort.Strings(keys)
		for _, code := range keys {
			constant := statusConstantByCode[code]
			after = replaceRawStatusLiteral(after, code, "http."+constant)
		}
		if after == before {
			return "", false, nil
		}
		return ensureHTTPImport(path, after)
	})
}

func previewJSONContentTypeFix(root string) ([]autofix.Candidate, error) {
	return rewriteProductionGoFiles(root, CodeContentTypeMissing, "Set application/json before obvious JSON response writes.", func(_ string, before string) (string, bool, error) {
		lines := strings.SplitAfter(before, "\n")
		changed := false
		for i, line := range lines {
			if !strings.Contains(line, "json.NewEncoder(") || !strings.Contains(line, ".Encode(") {
				continue
			}
			writer := writerFromJSONEncoderLine(line)
			if writer == "" || precedingContentType(lines, i, writer) {
				continue
			}
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			lines[i] = indent + writer + `.Header().Set("Content-Type", "application/json")` + "\n" + line
			changed = true
		}
		if !changed {
			return "", false, nil
		}
		return strings.Join(lines, ""), true, nil
	})
}

func previewResponseBodyCloseFix(root string) ([]autofix.Candidate, error) {
	return rewriteProductionGoFiles(root, CodeResponseBodyUnclosed, "Close outbound HTTP response bodies after successful requests.", func(_ string, before string) (string, bool, error) {
		lines := strings.SplitAfter(before, "\n")
		changed := false
		for i, line := range lines {
			resp := assignedResponseName(line)
			if resp == "" || strings.Contains(before, resp+".Body.Close()") {
				continue
			}
			insertAt := i + 1
			for insertAt < len(lines) && strings.TrimSpace(lines[insertAt]) == "" {
				insertAt++
			}
			if insertAt < len(lines) && strings.Contains(lines[insertAt], "err != nil") {
				insertAt = blockEndLine(lines, insertAt) + 1
			}
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			deferLine := indent + "defer " + resp + ".Body.Close()\n"
			lines = append(lines[:insertAt], append([]string{deferLine}, lines[insertAt:]...)...)
			changed = true
		}
		if !changed {
			return "", false, nil
		}
		return strings.Join(lines, ""), true, nil
	})
}

func rewriteProductionGoFiles(root, ruleID, description string, rewrite func(path, before string) (string, bool, error)) ([]autofix.Candidate, error) {
	files, err := productionGoFiles(filepath.Join(root, "api"))
	if err != nil {
		return nil, err
	}
	var out []autofix.Candidate
	for _, path := range files {
		beforeBytes, err := os.ReadFile(path)
		if err != nil {
			return out, err
		}
		before := string(beforeBytes)
		after, changed, err := rewrite(path, before)
		if err != nil {
			return out, err
		}
		if !changed || after == before {
			continue
		}
		if formatted, err := format.Source([]byte(after)); err == nil {
			after = string(formatted)
		}
		out = append(out, autofix.Candidate{
			RuleID:      ruleID,
			FilePath:    path,
			Description: description,
			Before:      before,
			After:       after,
		})
	}
	return out, nil
}

func ensureMap(parent map[string]any, key string) map[string]any {
	if existing, ok := parent[key].(map[string]any); ok {
		return existing
	}
	m := map[string]any{}
	parent[key] = m
	return m
}

func hasAPIHealthCheck(checks []any) bool {
	for _, raw := range checks {
		check, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if strings.EqualFold(fmt.Sprint(check["type"]), "http") && strings.Contains(fmt.Sprint(check["target"]), "${API_PORT}") && strings.Contains(fmt.Sprint(check["target"]), "/health") {
			return true
		}
	}
	return false
}

var statusConstantByCode = map[string]string{
	"200": "StatusOK",
	"201": "StatusCreated",
	"202": "StatusAccepted",
	"204": "StatusNoContent",
	"400": "StatusBadRequest",
	"401": "StatusUnauthorized",
	"403": "StatusForbidden",
	"404": "StatusNotFound",
	"409": "StatusConflict",
	"422": "StatusUnprocessableEntity",
	"429": "StatusTooManyRequests",
	"500": "StatusInternalServerError",
	"502": "StatusBadGateway",
	"503": "StatusServiceUnavailable",
	"504": "StatusGatewayTimeout",
}

func replaceRawStatusLiteral(src, code, replacement string) string {
	escaped := regexp.QuoteMeta(code)
	replacers := []*regexp.Regexp{
		regexp.MustCompile(`(\.WriteHeader\(\s*)` + escaped + `(\s*\))`),
		regexp.MustCompile(`(http\.Error\([^,\n]+,\s*[^,\n]+,\s*)` + escaped + `(\s*\))`),
		regexp.MustCompile(`(\.JSON\(\s*)` + escaped + `(\s*,)`),
		regexp.MustCompile(`(\.IndentedJSON\(\s*)` + escaped + `(\s*,)`),
	}
	out := src
	for _, re := range replacers {
		out = re.ReplaceAllString(out, "${1}"+replacement+"${2}")
	}
	return out
}

func ensureHTTPImport(path, src string) (string, bool, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return "", false, err
	}
	for _, imp := range file.Imports {
		if strings.Trim(imp.Path.Value, `"`) == "net/http" {
			return src, true, nil
		}
	}
	if len(file.Imports) == 0 {
		return src, true, nil
	}
	return addImport(src, "net/http"), true, nil
}

func addImport(src, importPath string) string {
	if strings.Contains(src, "import (\n") {
		return strings.Replace(src, "import (\n", "import (\n\t"+strconv.Quote(importPath)+"\n", 1)
	}
	return src
}

func writerFromJSONEncoderLine(line string) string {
	idx := strings.Index(line, "json.NewEncoder(")
	if idx < 0 {
		return ""
	}
	rest := line[idx+len("json.NewEncoder("):]
	end := strings.Index(rest, ")")
	if end < 0 {
		return ""
	}
	writer := strings.TrimSpace(rest[:end])
	if !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(writer) {
		return ""
	}
	return writer
}

func precedingContentType(lines []string, idx int, writer string) bool {
	start := idx - 6
	if start < 0 {
		start = 0
	}
	needle := writer + `.Header().Set("Content-Type"`
	alt := writer + `.Header().Add("Content-Type"`
	for i := start; i < idx; i++ {
		if strings.Contains(lines[i], needle) || strings.Contains(lines[i], alt) {
			return true
		}
	}
	return false
}

func assignedResponseName(line string) string {
	if !strings.Contains(line, ".Do(") && !strings.Contains(line, "http.Get(") && !strings.Contains(line, "http.Post(") && !strings.Contains(line, "http.Head(") {
		return ""
	}
	left, _, ok := strings.Cut(line, ":=")
	if !ok {
		left, _, ok = strings.Cut(line, "=")
	}
	if !ok {
		return ""
	}
	parts := strings.Split(left, ",")
	if len(parts) == 0 {
		return ""
	}
	name := strings.TrimSpace(parts[0])
	if name == "_" || !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(name) {
		return ""
	}
	return name
}

func blockEndLine(lines []string, start int) int {
	depth := 0
	for i := start; i < len(lines); i++ {
		depth += strings.Count(lines[i], "{")
		depth -= strings.Count(lines[i], "}")
		if depth <= 0 && strings.Contains(lines[i], "}") {
			return i
		}
	}
	return start
}

func fileExistsCanFix(rel string) func(root, _ string) bool {
	return func(root, _ string) bool {
		_, err := os.Stat(filepath.Join(root, rel))
		return err == nil
	}
}

func apiSourceCanFix(root, _ string) bool {
	files, err := productionGoFiles(filepath.Join(root, "api"))
	return err == nil && len(files) > 0
}
