// Package targetpack contains structure checks whose authority is a
// repository target manifest rather than a scenario service.json.
package targetpack

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"structure-health/internal/rules"
)

var digestPattern = regexp.MustCompile(`@sha256:[0-9a-fA-F]{64}$`)

// ParseUnit is the subset of Code Facts evidence needed by package rules.
// The package evaluator never guesses language from extensions when a proven
// unit is available.
type ParseUnit struct {
	Language   string
	RootPath   string
	ConfigPath string
	Status     string
}

// Evaluate runs the pack owned by kind. Unknown kinds intentionally produce no
// findings here; the provider contract is the authority for declared support.
func Evaluate(kind, root, id string) []rules.Finding {
	return EvaluateWithParseUnits(kind, root, id, nil)
}

func EvaluateWithParseUnits(kind, root, id string, parseUnits []ParseUnit) []rules.Finding {
	kind = strings.ToLower(strings.TrimSpace(kind))
	var findings []rules.Finding
	switch kind {
	case "resource":
		findings = evaluateResource(root, id)
	case "tool":
		findings = evaluateTool(root, id)
	case "safeguard":
		findings = evaluateSafeguard(root, id)
	case "package":
		findings = evaluatePackage(root, id, parseUnits)
	case "control-plane":
		findings = evaluateControlPlane(root)
	case "docs":
		findings = evaluateDocs(root)
	case "team":
		findings = evaluateTeam(root, id)
	case "project":
		findings = evaluateProject(root)
	}
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Code != findings[j].Code {
			return findings[i].Code < findings[j].Code
		}
		return findings[i].Location < findings[j].Location
	})
	return findings
}

func evaluatePackage(root, id string, parseUnits []ParseUnit) []rules.Finding {
	manifestPath := filepath.Join(root, ".vrooli", "package.json")
	_, doc, ok := loadObject(manifestPath)
	if !ok {
		return []rules.Finding{finding("PACKAGE_MANIFEST_MISSING", "error", "package manifest is missing or invalid", filepath.ToSlash(filepath.Join(".vrooli", "package.json")), "Add a valid .vrooli/package.json package governance manifest.")}
	}
	var out []rules.Finding
	if stringValue(doc["$schema"]) != "schemas/package.schema.json" {
		out = append(out, finding("PACKAGE_MANIFEST_INVALID", "error", "package manifest schema reference is invalid", ".vrooli/package.json#/$schema", "Use schemas/package.schema.json as the package manifest schema."))
	}
	entry, _ := doc["package"].(map[string]any)
	name := stringValue(entry["name"])
	if strings.TrimSpace(name) == "" {
		out = append(out, finding("PACKAGE_MANIFEST_INVALID", "error", "package manifest has no package name", ".vrooli/package.json#/package/name", "Declare package.name."))
	} else if id != "" && name != id {
		out = append(out, finding("PACKAGE_NAME_MISMATCH", "error", "package name does not match its target id", ".vrooli/package.json#/package/name", "Set package.name to the canonical package id."))
	}
	identifiers, _ := entry["module_identifiers"].([]any)
	if len(identifiers) == 0 || strings.TrimSpace(stringValue(entry["kind"])) == "" || entry["adoption"] == nil || entry["lifecycle"] == nil || entry["refresh"] == nil {
		out = append(out, finding("PACKAGE_MANIFEST_INVALID", "error", "package manifest is incomplete", ".vrooli/package.json#/package", "Declare package kind and at least one module identifier."))
	}
	if !fileExists(filepath.Join(root, "README.md")) || (!fileExists(filepath.Join(root, "go.mod")) && !fileExists(filepath.Join(root, "package.json"))) {
		out = append(out, finding("PACKAGE_LAYOUT_MISSING", "error", "package layout is incomplete", ".", "Provide README.md and a go.mod or package.json at the package root, or record an intentional module-boundary exception."))
	}
	if module := goModule(root); module != "" && !stringArrayContains(identifiers, module) {
		out = append(out, finding("PACKAGE_MODULE_PATH_MISMATCH", "error", "Go module is absent from package governance identifiers", "go.mod", "Add the go.mod module path to package.module_identifiers."))
	}
	if npmName := packageJSONName(root); npmName != "" && !stringArrayContains(identifiers, npmName) {
		out = append(out, finding("PACKAGE_MODULE_PATH_MISMATCH", "error", "JavaScript package name is absent from package governance identifiers", "package.json#/name", "Add the package.json name to package.module_identifiers."))
	}
	out = append(out, packageParseUnitRules(root, parseUnits)...)
	return out
}

func packageParseUnitRules(root string, parseUnits []ParseUnit) []rules.Finding {
	var out []rules.Finding
	root = filepath.Clean(root)
	for _, unit := range parseUnits {
		if !strings.EqualFold(unit.Status, "proven") && unit.Status != "" {
			continue
		}
		unitRoot := filepath.Clean(unit.RootPath)
		// A Go module is the package boundary this rule governs. TypeScript,
		// JavaScript, Rust, and Python parse units are useful evidence for their
		// own profile packs, but do not imply that a Go package module is missing.
		if !strings.EqualFold(unit.Language, "go") {
			continue
		}
		if unitRoot != root && !packageHasModuleException(root) {
			out = append(out, finding("PACKAGE_OWN_MODULE_MISSING", "error", "package is covered by a module rooted elsewhere", ".", "Add a module configuration at the package root or record an intentional exception."))
			break
		}
		if hasRootInternalImport(root) {
			out = append(out, finding("PACKAGE_INTERNAL_IMPORT", "error", "package imports the root control plane internal package", ".", "Promote or duplicate the shared capability and remove the root internal import."))
		}
		out = append(out, missingRootReplaces(root)...)
		break
	}
	return out
}

func packageHasModuleException(root string) bool {
	_, doc, ok := loadObject(filepath.Join(root, ".vrooli", "package.json"))
	if !ok {
		return false
	}
	entry, _ := doc["package"].(map[string]any)
	return intentionalModuleException(entry)
}

func intentionalModuleException(entry map[string]any) bool {
	boundary, _ := entry["module_boundary"].(map[string]any)
	return stringValue(boundary["status"]) == "intentional_exception" && strings.TrimSpace(stringValue(boundary["reason"])) != ""
}

func hasRootInternalImport(root string) bool {
	found := false
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || found || entry.IsDir() || filepath.Ext(path) != ".go" {
			return err
		}
		raw, readErr := os.ReadFile(path)
		if readErr == nil && strings.Contains(string(raw), "github.com/vrooli/vrooli/internal/") {
			found = true
		}
		return nil
	})
	return found
}

func missingRootReplaces(root string) []rules.Finding {
	rootGoMod := ancestorGoMod(root)
	if rootGoMod == "" {
		return nil
	}
	rootModRaw, err := os.ReadFile(rootGoMod)
	if err != nil {
		return nil
	}
	localReplaces := moduleDirectives(string(rootModRaw), "replace")
	modRaw, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return nil
	}
	requires := moduleDirectives(string(modRaw), "require")
	consumerReplaces := moduleDirectives(string(modRaw), "replace")
	var out []rules.Finding
	for module := range requires {
		if localReplaces[module] && !consumerReplaces[module] {
			out = append(out, finding("PACKAGE_GO_REPLACE_MISSING", "error", "module is missing a required local replace", "go.mod", "Use Scenario Dependency Analyzer to reconcile the module's local replaces."))
		}
	}
	return out
}

// ancestorGoMod returns the nearest go.mod above a package root. A package's
// own go.mod is deliberately skipped: the caller needs the repository module
// whose local replaces must be reproduced by the dependent module.
func ancestorGoMod(root string) string {
	dir := filepath.Dir(filepath.Clean(root))
	for {
		candidate := filepath.Join(dir, "go.mod")
		if fileExists(candidate) {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// moduleDirectives extracts module names from both single-line and grouped
// go.mod require/replace directives. It intentionally reads only the module
// identity; versions and replacement paths are not needed for this rule.
func moduleDirectives(raw, directive string) map[string]bool {
	result := map[string]bool{}
	inBlock := false
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "//", 2)[0])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, directive+" (") {
			inBlock = true
			continue
		}
		if inBlock && line == ")" {
			inBlock = false
			continue
		}
		if strings.HasPrefix(line, directive+" ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, directive))
		} else if !inBlock {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 {
			result[fields[0]] = true
		}
	}
	return result
}

func evaluateControlPlane(root string) []rules.Finding {
	if !hasFileExtension(root, ".go") {
		return []rules.Finding{finding("CONTROL_PLANE_LAYOUT_MISSING", "warning", "control-plane target has no Go source", ".", "Keep control-plane cmd/internal targets backed by Go source files.")}
	}
	return nil
}

func evaluateDocs(root string) []rules.Finding {
	manifestPath := filepath.Join(root, "manifest.json")
	_, doc, ok := loadObject(manifestPath)
	if !ok {
		return []rules.Finding{finding("DOCS_MANIFEST_INVALID", "error", "docs manifest is missing or invalid", "manifest.json", "Add a valid docs/manifest.json project manifest.")}
	}
	if strings.TrimSpace(stringValue(doc["title"])) == "" || strings.TrimSpace(stringValue(doc["version"])) == "" || !nonEmptyArray(doc["sections"]) {
		return []rules.Finding{finding("DOCS_MANIFEST_INVALID", "error", "docs manifest is incomplete", "manifest.json", "Declare version, title, and at least one documentation section.")}
	}
	if !fileExists(filepath.Join(root, "README.md")) {
		return []rules.Finding{finding("DOCS_LAYOUT_MISSING", "warning", "docs target has no README", "README.md", "Add the documentation hub README.md.")}
	}
	return nil
}

func evaluateTeam(root, id string) []rules.Finding {
	manifestPath := filepath.Join(root, "manifest.json")
	_, doc, ok := loadObject(manifestPath)
	if !ok {
		return []rules.Finding{finding("TEAM_MANIFEST_INVALID", "error", "team manifest is missing or invalid", "manifest.json", "Add a valid team plan-of-record manifest.")}
	}
	contract, _ := doc["contract"].(map[string]any)
	team := strings.TrimSpace(stringValue(contract["team"]))
	if team == "" {
		return []rules.Finding{finding("TEAM_OWNER_MISSING", "error", "team manifest has no owner id", "manifest.json#/contract/team", "Declare contract.team as the stable team target id.")}
	}
	if id != "" && team != id {
		return []rules.Finding{finding("TEAM_OWNER_MISMATCH", "error", "team owner does not match its target id", "manifest.json#/contract/team", "Align contract.team with the enumerated team target id.")}
	}
	if !fileExists(filepath.Join(root, "README.md")) || !nonEmptyArray(doc["sections"]) {
		return []rules.Finding{finding("TEAM_LAYOUT_MISSING", "error", "team plan-of-record layout is incomplete", ".", "Provide README.md and at least one declared manifest section.")}
	}
	return nil
}

func evaluateResource(root, id string) []rules.Finding {
	manifestPath := filepath.Join(root, "resource.json")
	raw, doc, ok := loadObject(manifestPath)
	if !ok {
		return []rules.Finding{finding("RESOURCE_MANIFEST_INVALID", "error", "resource.json is invalid", "resource.json", "Fix resource.json so it is valid JSON and declares a resource manifest.")}
	}
	var out []rules.Finding
	if name, _ := doc["name"].(string); strings.TrimSpace(name) == "" || (id != "" && name != id) {
		out = append(out, finding("RESOURCE_MANIFEST_INVALID", "error", "resource manifest identity is invalid", "resource.json", "Set resource.json.name to the canonical resource id."))
	}
	checks, _ := doc["health_checks"].([]any)
	if len(checks) == 0 {
		out = append(out, finding("RESOURCE_HEALTH_KIND_MISSING", "error", "resource has no health checks", "resource.json#/health_checks", "Declare at least one readiness or liveness health check."))
	} else {
		for i, rawCheck := range checks {
			check, _ := rawCheck.(map[string]any)
			kind, _ := check["kind"].(string)
			if kind != "readiness" && kind != "liveness" {
				out = append(out, finding("RESOURCE_HEALTH_KIND_MISSING", "error", "resource health check kind is invalid", "resource.json#/health_checks/"+itoa(i)+"/kind", "Use readiness or liveness for every resource health check."))
			}
		}
	}
	if image := nestedString(doc, "runtime", "image"); image != "" && !pinnedImage(image) {
		out = append(out, finding("RESOURCE_IMAGE_UNPINNED", "error", "resource runtime image is not pinned", "resource.json#/runtime/image", "Pin container images with a sha256 digest."))
	}
	if shellFinding := forbiddenShell(root); shellFinding != nil {
		out = append(out, *shellFinding)
	}
	if imageFinding := unpinnedFileImage(root); imageFinding != nil {
		out = append(out, *imageFinding)
	}
	_ = raw
	return out
}

func evaluateTool(root, id string) []rules.Finding {
	manifestPath := filepath.Join(root, "tool.json")
	_, doc, ok := loadObject(manifestPath)
	if !ok {
		return []rules.Finding{finding("TOOL_MANIFEST_INVALID", "error", "tool.json is invalid", "tool.json", "Fix tool.json so it is valid JSON and declares a tool manifest.")}
	}
	var out []rules.Finding
	name, _ := doc["name"].(string)
	if strings.TrimSpace(name) == "" || (id != "" && name != id) {
		out = append(out, finding("TOOL_NAME_MISMATCH", "error", "tool name does not match its target id", "tool.json#/name", "Set tool.json.name to the canonical tool id."))
	}
	if !nonEmptyStringArray(doc["commands"]) || !nonEmptyStringArray(doc["versionArgs"]) || strings.TrimSpace(stringValue(doc["description"])) == "" || strings.TrimSpace(stringValue(doc["bundling"])) == "" {
		out = append(out, finding("TOOL_MANIFEST_INVALID", "error", "tool manifest is incomplete", "tool.json", "Declare description, commands, versionArgs, and bundling."))
	}
	if handler := strings.TrimSpace(stringValue(doc["handler"])); handler != "" {
		if _, err := os.Stat(filepath.Join(root, "handler.go")); err != nil {
			out = append(out, finding("TOOL_HANDLER_MISSING", "error", "tool handler is missing", "handler.go", "Add the Go handler declared by tool.json."))
		}
	}
	return out
}

func evaluateSafeguard(root, id string) []rules.Finding {
	manifestPath := filepath.Join(root, "safeguard.json")
	_, doc, ok := loadObject(manifestPath)
	if !ok {
		return []rules.Finding{finding("SAFEGUARD_MANIFEST_INVALID", "error", "safeguard.json is invalid", "safeguard.json", "Fix safeguard.json so it is valid JSON and declares a safeguard manifest.")}
	}
	var out []rules.Finding
	name, _ := doc["name"].(string)
	if strings.TrimSpace(name) == "" || (id != "" && name != id) {
		out = append(out, finding("SAFEGUARD_NAME_MISMATCH", "error", "safeguard name does not match its target id", "safeguard.json#/name", "Set safeguard.json.name to the canonical safeguard id."))
	}
	if strings.TrimSpace(stringValue(doc["description"])) == "" || strings.TrimSpace(stringValue(doc["handler"])) == "" || strings.TrimSpace(stringValue(doc["privilege"])) == "" || strings.TrimSpace(stringValue(doc["bundling"])) == "" || doc["deployment"] == nil {
		out = append(out, finding("SAFEGUARD_MANIFEST_INVALID", "error", "safeguard manifest is incomplete", "safeguard.json", "Declare description, handler, privilege, bundling, and deployment."))
	}
	if handler := strings.TrimSpace(stringValue(doc["handler"])); handler != "" {
		if _, err := os.Stat(filepath.Join(root, "handler.go")); err != nil {
			out = append(out, finding("SAFEGUARD_HANDLER_MISSING", "error", "safeguard handler is missing", "handler.go", "Add the Go handler declared by safeguard.json."))
		}
	}
	return out
}

func loadObject(path string) ([]byte, map[string]any, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, false
	}
	var doc map[string]any
	if json.Unmarshal(raw, &doc) != nil || doc == nil {
		return raw, nil, false
	}
	return raw, doc, true
}

func forbiddenShell(root string) *rules.Finding {
	var result *rules.Finding
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || result != nil || entry.IsDir() {
			return err
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".sh" || ext == ".bash" {
			rel, _ := filepath.Rel(root, path)
			f := finding("RESOURCE_SHELL_FORBIDDEN", "error", "resource contains a shell file", filepath.ToSlash(rel), "Remove shell-owned resource lifecycle files.")
			result = &f
		}
		return nil
	})
	return result
}

func unpinnedFileImage(root string) *rules.Finding {
	var result *rules.Finding
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || result != nil {
			return err
		}
		if entry.IsDir() || (entry.Name() != "Dockerfile" && !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml")) {
			return nil
		}
		file, openErr := os.Open(path)
		if openErr != nil {
			return openErr
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		lineNumber := 0
		for scanner.Scan() {
			lineNumber++
			line := strings.TrimSpace(scanner.Text())
			image := ""
			if strings.HasPrefix(line, "FROM ") {
				image = strings.Fields(strings.TrimPrefix(line, "FROM "))[0]
			} else if strings.HasPrefix(line, "image:") {
				image = strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "image:")), "\"'")
			}
			if image != "" && image != "scratch" && !pinnedImage(image) {
				rel, _ := filepath.Rel(root, path)
				f := finding("RESOURCE_IMAGE_UNPINNED", "error", "resource file image is not pinned", filepath.ToSlash(rel)+":"+itoa(lineNumber), "Pin container images with a sha256 digest.")
				result = &f
				break
			}
		}
		return scanner.Err()
	})
	return result
}

func pinnedImage(image string) bool {
	image = strings.TrimSpace(strings.Trim(image, "\"'"))
	return digestPattern.MatchString(image)
}

func nestedString(doc map[string]any, keys ...string) string {
	var current any = doc
	for _, key := range keys {
		object, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = object[key]
	}
	return stringValue(current)
}

func stringValue(value any) string {
	result, _ := value.(string)
	return result
}

func nonEmptyStringArray(value any) bool {
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		return false
	}
	for _, item := range items {
		if strings.TrimSpace(stringValue(item)) == "" {
			return false
		}
	}
	return true
}

func nonEmptyArray(value any) bool {
	items, ok := value.([]any)
	return ok && len(items) > 0
}

func stringArrayContains(items []any, want string) bool {
	for _, item := range items {
		if stringValue(item) == want {
			return true
		}
	}
	return false
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func goModule(root string) string {
	raw, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			return fields[1]
		}
	}
	return ""
}

func packageJSONName(root string) string {
	path := filepath.Join(root, "package.json")
	_, doc, ok := loadObject(path)
	if !ok {
		return ""
	}
	return stringValue(doc["name"])
}

func hasFileExtension(root, extension string) bool {
	found := false
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || found {
			return err
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), extension) {
			found = true
		}
		return nil
	})
	return found
}

func finding(code, severity, title, location, remediation string) rules.Finding {
	if entry, ok := rules.Lookup(code); ok {
		severity = entry.Severity
		remediation = entry.Remediation
	}
	return rules.Finding{Code: code, Severity: severity, Title: title, Message: title, Location: location, Remediation: remediation}
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	return strconv.Itoa(value)
}
