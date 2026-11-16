package lifecycleinference

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// TypeScript and JSDoc support is intentionally lexical and conservative. It
// emits review candidates from declarations and finite vocabularies;
// persistence remains unknown unless structural source evidence proves a sink.
var (
	tsTypeDecl   = regexp.MustCompile(`(?m)(?:export\s+)?(?:type|enum)\s+([A-Za-z_$][\w$]*)\s*(?:=|\{)`)
	tsOwnerDecl  = regexp.MustCompile(`(?m)(?:export\s+)?(?:interface|class)\s+([A-Za-z_$][\w$]*)`)
	tsField      = regexp.MustCompile(`^\s*(?:readonly\s+)?([A-Za-z_$][\w$]*)\??\s*:\s*([^;=]+)`)
	tsUnion      = regexp.MustCompile(`['"]([^'"]+)['"]`)
	tsEnumValue  = regexp.MustCompile(`([A-Za-z_$][\w$]*)\s*=\s*['"]([^'"]+)['"]`)
	jsTypeAlias  = regexp.MustCompile(`(?m)@typedef\s*\{([^}]+)\}\s*([A-Za-z_$][\w$]*)`)
	jsObjectType = regexp.MustCompile(`(?m)@typedef\s*\{Object\}\s*([A-Za-z_$][\w$]*)`)
	jsProperty   = regexp.MustCompile(`@property\s*\{([^}]+)\}\s*([A-Za-z_$][\w$]*)`)
)

func analyzeTypeScriptScenario(scenarioRoot, scenarioID string) ([]Candidate, int, int) {
	var files []string
	for _, rel := range []string{"api", "cli", "ui/src"} {
		root := filepath.Join(scenarioRoot, rel)
		_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if entry.IsDir() && path != root && skipDir(entry.Name()) {
				return filepath.SkipDir
			}
			if !entry.IsDir() && (strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx") || strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".jsx")) && !strings.HasSuffix(path, ".d.ts") && !strings.HasSuffix(path, ".test.ts") && !strings.HasSuffix(path, ".spec.ts") && !strings.HasSuffix(path, ".test.tsx") && !strings.HasSuffix(path, ".spec.tsx") && !strings.HasSuffix(path, ".test.js") && !strings.HasSuffix(path, ".spec.js") && !strings.HasSuffix(path, ".test.jsx") && !strings.HasSuffix(path, ".spec.jsx") {
				files = append(files, path)
			}
			return nil
		})
	}
	var out []Candidate
	parseErrors := 0
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			parseErrors++
			continue
		}
		text := string(data)
		rel, err := filepath.Rel(filepath.Dir(scenarioRoot), path)
		if err != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		lineAt := func(offset int) int { return 1 + strings.Count(text[:min(offset, len(text))], "\n") }
		vocab := map[string][]StateEvidence{}
		for _, match := range tsTypeDecl.FindAllStringSubmatchIndex(text, -1) {
			name := text[match[2]:match[3]]
			start := match[0]
			end := min(len(text), start+800)
			chunk := text[start:end]
			var states []StateEvidence
			if strings.Contains(chunk, "enum ") {
				for _, value := range tsEnumValue.FindAllStringSubmatchIndex(chunk, -1) {
					states = append(states, StateEvidence{Name: chunk[value[2]:value[3]], Value: chunk[value[4]:value[5]], At: Evidence{Path: rel, Line: lineAt(start + value[0]), Text: strings.TrimSpace(chunk[value[0]:value[1]])}})
				}
			} else {
				for _, value := range tsUnion.FindAllStringSubmatchIndex(chunk, -1) {
					states = append(states, StateEvidence{Name: chunk[value[2]:value[3]], Value: chunk[value[2]:value[3]], At: Evidence{Path: rel, Line: lineAt(start + value[0]), Text: strings.TrimSpace(chunk[value[0]:value[1]])}})
				}
			}
			if len(states) > 1 {
				vocab[name] = states
			}
		}
		for _, match := range jsTypeAlias.FindAllStringSubmatchIndex(text, -1) {
			var states []StateEvidence
			for _, value := range tsUnion.FindAllStringSubmatchIndex(text[match[2]:match[3]], -1) {
				chunk := text[match[2]:match[3]]
				states = append(states, StateEvidence{Name: chunk[value[2]:value[3]], Value: chunk[value[2]:value[3]], At: Evidence{Path: rel, Line: lineAt(match[0]), Text: strings.TrimSpace(text[match[0]:match[1]])}})
			}
			if len(states) > 1 {
				vocab[text[match[4]:match[5]]] = states
			}
		}
		type owners struct {
			name   string
			body   string
			offset int
		}
		var declarations []owners
		for _, m := range tsOwnerDecl.FindAllStringSubmatchIndex(text, -1) {
			open := strings.Index(text[m[0]:], "{")
			if open < 0 {
				continue
			}
			begin := m[0] + open
			end := matchingBrace(text, begin)
			if end < 0 {
				// This lexical declaration scan is not a TypeScript parser. An
				// incomplete or unsupported brace shape cannot produce a review
				// candidate, but is not a source read/parse failure.
				continue
			}
			declarations = append(declarations, owners{name: text[m[2]:m[3]], body: text[begin+1 : end], offset: begin + 1})
		}
		for _, m := range jsObjectType.FindAllStringSubmatchIndex(text, -1) {
			declarations = append(declarations, owners{name: text[m[2]:m[3]], body: text, offset: m[0]})
		}
		for _, owner := range declarations {
			lines := strings.Split(owner.body, "\n")
			for i, line := range lines {
				m := tsField.FindStringSubmatch(line)
				if len(m) < 3 {
					if property := jsProperty.FindStringSubmatch(line); len(property) >= 3 {
						m = []string{line, property[2], property[1]}
					}
				}
				if len(m) < 3 || !isLifecycleField(m[1]) {
					continue
				}
				fieldName, fieldType := m[1], strings.TrimSpace(m[2])
				states := append([]StateEvidence(nil), vocab[fieldType]...)
				if len(states) < 2 {
					continue
				}
				lineNo := lineAt(owner.offset) + i
				ev := Evidence{Path: rel, Line: lineNo, Text: strings.TrimSpace(line)}
				candidate := Candidate{ID: rel + "#" + owner.name + "." + fieldName, ScenarioID: scenarioID, Package: filepath.ToSlash(filepath.Dir(strings.TrimPrefix(rel, "scenarios/"+scenarioID+"/"))), TypeName: owner.name, FieldName: fieldName, FieldType: fieldType, Declaration: ev, States: states, Persistence: PersistenceClassification{Class: ClassUnknown, Reason: "TypeScript/JavaScript persistence sinks are not yet structurally proven", Scope: "scenario production TypeScript/JavaScript", FilesScanned: len(files), ScanComplete: true}}
				if strings.HasSuffix(rel, ".js") || strings.HasSuffix(rel, ".jsx") {
					candidate.Language = "javascript"
				} else {
					candidate.Language = "typescript"
				}
				for _, other := range lines {
					lower := strings.ToLower(other)
					if strings.Contains(lower, "date") || strings.Contains(lower, "time") || strings.Contains(lower, "at:") {
						candidate.TimeFields = append(candidate.TimeFields, Evidence{Path: rel, Line: lineNo, Text: strings.TrimSpace(other)})
					}
					if (strings.Contains(lower, "boolean") && (strings.Contains(lower, "retry") || strings.Contains(lower, "failed") || strings.Contains(lower, "cancel") || strings.Contains(lower, "complete"))) || strings.Contains(lower, "event") || strings.Contains(lower, "retry") {
						candidate.RelatedFlags = append(candidate.RelatedFlags, Evidence{Path: rel, Line: lineNo, Text: strings.TrimSpace(other)})
					}
				}
				body := owner.body
				low := strings.ToLower(body)
				if strings.Contains(low, "transition") || strings.Contains(low, "set"+strings.ToLower(fieldName)) || strings.Contains(low, "update") {
					candidate.TransitionOwners = append(candidate.TransitionOwners, Evidence{Path: rel, Line: lineAt(owner.offset), Text: "transition/update owner in " + owner.name})
				}
				if strings.Contains(body, fieldName+" =") || strings.Contains(body, "."+fieldName+" =") {
					candidate.WriteSites = append(candidate.WriteSites, Evidence{Path: rel, Line: lineNo, Text: "assignment to " + fieldName})
				}
				if strings.Contains(body, fieldName+" ===") || strings.Contains(body, fieldName+" !==") {
					candidate.GuardSites = append(candidate.GuardSites, Evidence{Path: rel, Line: lineNo, Text: "guard reads " + fieldName})
				}
				candidate.BurdenScore, candidate.ScoreBreakdown = score(candidate)
				candidate.Confidence = confidence(candidate)
				out = append(out, candidate)
			}
		}
	}
	return out, len(files), parseErrors
}

func matchingBrace(source string, start int) int {
	depth := 0
	for i := start; i < len(source); i++ {
		if source[i] == '{' {
			depth++
		}
		if source[i] == '}' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
