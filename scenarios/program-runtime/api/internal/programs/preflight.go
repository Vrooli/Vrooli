package programs

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
)

var pythonIdentifier = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)

var pythonKeywords = map[string]struct{}{
	"and": {}, "as": {}, "assert": {}, "async": {}, "await": {}, "break": {}, "case": {}, "class": {}, "continue": {},
	"def": {}, "del": {}, "elif": {}, "else": {}, "except": {}, "finally": {}, "for": {}, "from": {}, "global": {},
	"if": {}, "import": {}, "in": {}, "is": {}, "lambda": {}, "match": {}, "nonlocal": {}, "not": {}, "or": {},
	"pass": {}, "raise": {}, "return": {}, "try": {}, "while": {}, "with": {}, "yield": {},
}

// ResolveSource performs the lightweight, deterministic part of Python name
// resolution needed before execution. It intentionally resolves only roots;
// proto field validation remains owned by the binding registry.
func ResolveSource(source string, known []string) []*programsv1.Diagnostic {
	knownSet := make(map[string]struct{}, len(known))
	for _, name := range known {
		knownSet[name] = struct{}{}
	}
	assigned := assignedNames(source)
	assigned = mergeNames(assigned, importedNames(source))
	seen := make(map[string]struct{})
	var diagnostics []*programsv1.Diagnostic
	lines := strings.Split(source, "\n")
	for lineNo, line := range lines {
		clean := stripPythonCommentAndStrings(line)
		if assignment := assignmentTarget(clean); assignment != "" {
			if _, ok := knownSet[assignment]; ok {
				if isProtectedRuntimeName(assignment) {
					diagnostics = append(diagnostics, &programsv1.Diagnostic{Severity: "error", Line: int32(lineNo + 1), Name: assignment, Message: fmt.Sprintf("protected runtime name %q cannot be assigned", assignment)})
				} else {
					diagnostics = append(diagnostics, &programsv1.Diagnostic{Severity: "warning", Line: int32(lineNo + 1), Name: assignment, Message: fmt.Sprintf("scenario namespace %q is shadowed; use __vrooli__.%s to reach the binding", assignment, assignment)})
				}
			}
		}
		ids := pythonIdentifier.FindAllStringIndex(clean, -1)
		for _, index := range ids {
			name := clean[index[0]:index[1]]
			if _, keyword := pythonKeywords[name]; keyword || isPythonLiteral(name) || isAttributeToken(clean, index[0]) || isAssignedToken(clean, index[1]) {
				continue
			}
			if _, local := assigned[name]; local && name != "test_geni" {
				continue
			}
			if _, ok := knownSet[name]; ok || isBuiltin(name) || isBuiltinVariable(name) {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			nearest := nearestName(name, known)
			message := fmt.Sprintf("name %q does not resolve to a governed binding namespace or a built-in", name)
			if nearest != "" {
				message += fmt.Sprintf("; nearest match: %q", nearest)
			}
			diagnostics = append(diagnostics, &programsv1.Diagnostic{Severity: "error", Line: int32(lineNo + 1), Name: name, Message: message, NearestMatch: nearest})
		}
	}
	return diagnostics
}

func assignmentTarget(line string) string {
	index := strings.Index(line, "=")
	if index < 0 || strings.HasPrefix(strings.TrimSpace(line[index:]), "==") {
		return ""
	}
	return pythonIdentifier.FindString(strings.TrimSpace(line[:index]))
}

func isProtectedRuntimeName(name string) bool {
	_, ok := map[string]struct{}{"discover": {}, "recall": {}, "guide": {}, "validate": {}, "capture": {}, "ai": {}, "agent": {}, "gather": {}, "describe": {}, "reachable": {}, "lib": {}, "vrooli": {}, "__vrooli__": {}}[name]
	return ok
}

func mergeNames(left, right map[string]struct{}) map[string]struct{} {
	for name := range right {
		left[name] = struct{}{}
	}
	return left
}

func importedNames(source string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, line := range strings.Split(source, "\n") {
		clean := strings.TrimSpace(stripPythonCommentAndStrings(line))
		if strings.HasPrefix(clean, "import ") {
			for _, item := range strings.Split(strings.TrimSpace(strings.TrimPrefix(clean, "import ")), ",") {
				parts := strings.Fields(item)
				if len(parts) >= 3 && parts[len(parts)-2] == "as" {
					result[parts[len(parts)-1]] = struct{}{}
				} else if len(parts) > 0 {
					result[strings.Split(parts[0], ".")[0]] = struct{}{}
				}
			}
		}
		if strings.HasPrefix(clean, "from ") && strings.Contains(clean, " import ") {
			items := strings.SplitN(clean, " import ", 2)[1]
			for _, item := range strings.Split(items, ",") {
				parts := strings.Fields(item)
				if len(parts) >= 3 && parts[len(parts)-2] == "as" {
					result[parts[len(parts)-1]] = struct{}{}
				} else if len(parts) > 0 {
					result[parts[0]] = struct{}{}
				}
			}
		}
	}
	return result
}

func assignedNames(source string) map[string]struct{} {
	assigned := make(map[string]struct{})
	for _, line := range strings.Split(source, "\n") {
		clean := stripPythonCommentAndStrings(line)
		if index := strings.Index(clean, "="); index >= 0 && !strings.Contains(clean[index:], "==") {
			lhs := strings.TrimSpace(clean[:index])
			if match := pythonIdentifier.FindString(lhs); match != "" {
				assigned[match] = struct{}{}
			}
		}
	}
	return assigned
}

func stripPythonCommentAndStrings(line string) string {
	var out strings.Builder
	inSingle, inDouble := false, false
	for index := 0; index < len(line); index++ {
		char := line[index]
		if char == '\\' && (inSingle || inDouble) {
			index++
			continue
		}
		if char == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if char == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if char == '#' && !inSingle && !inDouble {
			break
		}
		if !inSingle && !inDouble {
			out.WriteByte(char)
		} else {
			out.WriteByte(' ')
		}
	}
	return out.String()
}

func isAttributeToken(line string, start int) bool {
	for start > 0 && line[start-1] == ' ' {
		start--
	}
	return start > 0 && line[start-1] == '.'
}

func isAssignedToken(line string, end int) bool {
	rest := strings.TrimSpace(line[end:])
	return strings.HasPrefix(rest, "=") && !strings.HasPrefix(rest, "==")
}

func isPythonLiteral(name string) bool {
	return name == "True" || name == "False" || name == "None"
}

func isBuiltin(name string) bool {
	_, ok := map[string]struct{}{"print": {}, "len": {}, "min": {}, "max": {}, "range": {}, "sorted": {}, "sum": {}, "list": {}, "dict": {}, "set": {}, "str": {}, "int": {}, "float": {}, "bool": {}, "enumerate": {}, "isinstance": {}, "object": {}}[name]
	return ok
}

func isBuiltinVariable(name string) bool {
	return name == "Handle" || name == "result" || name == "rows" || name == "value" || name == "item" || name == "items"
}

func nearestName(name string, known []string) string {
	sorted := append([]string(nil), known...)
	sort.Strings(sorted)
	best, score := "", 0.0
	for _, candidate := range sorted {
		value := similarity(name, candidate)
		if value > score {
			best, score = candidate, value
		}
	}
	if score < 0.45 {
		return ""
	}
	return best
}

func similarity(left, right string) float64 {
	if left == right {
		return 1
	}
	common := 0
	for _, char := range left {
		if strings.ContainsRune(right, char) {
			common++
		}
	}
	return float64(common) / float64(maxInt(len(left), len(right)))
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
