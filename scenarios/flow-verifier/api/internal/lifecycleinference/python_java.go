package lifecycleinference

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Python and Java inference is deliberately declaration-only: a finite enum
// plus a lifecycle-named field creates a review lead, never persistence proof.
var (
	pyEnumStart    = regexp.MustCompile(`^\s*class\s+([A-Za-z_]\w*)\s*\([^)]*Enum[^)]*\)\s*:`)
	pyClassStart   = regexp.MustCompile(`^\s*class\s+([A-Za-z_]\w*)\b`)
	pyField        = regexp.MustCompile(`^\s*([A-Za-z_]\w*)\s*:\s*([A-Za-z_]\w*)\b`)
	javaEnumStart  = regexp.MustCompile(`\benum\s+([A-Za-z_]\w*)\s*\{`)
	javaOwnerStart = regexp.MustCompile(`\b(?:class|record)\s+([A-Za-z_]\w*)\b`)
	javaField      = regexp.MustCompile(`^\s*(?:(?:public|private|protected|static|final|volatile|transient)\s+)*([A-Za-z_]\w*)\s+([A-Za-z_]\w*)\s*(?:[;=])`)
	javaEnumValue  = regexp.MustCompile(`^[A-Za-z_]\w*`)
)

func analyzePythonJavaScenario(scenarioRoot, scenarioID string) ([]Candidate, int, int) {
	var paths []string
	for _, root := range []string{"api", "cli", "ui/src"} {
		base := filepath.Join(scenarioRoot, root)
		_ = filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if entry.IsDir() && path != base && skipDir(entry.Name()) {
				return filepath.SkipDir
			}
			if entry.IsDir() {
				return nil
			}
			ext := filepath.Ext(path)
			if (ext == ".py" || ext == ".java") && !strings.HasSuffix(path, "_test.py") && !strings.HasSuffix(path, "Test.java") && !strings.HasSuffix(path, ".spec.py") {
				paths = append(paths, path)
			}
			return nil
		})
	}
	var out []Candidate
	parseErrors := 0
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			parseErrors++
			continue
		}
		rel, err := filepath.Rel(filepath.Dir(scenarioRoot), path)
		if err != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		text := string(data)
		if filepath.Ext(path) == ".py" {
			out = append(out, inferPython(rel, scenarioID, text)...)
		} else {
			out = append(out, inferJava(rel, scenarioID, text)...)
		}
	}
	return out, len(paths), parseErrors
}

func inferPython(path, scenarioID, source string) []Candidate {
	lines := strings.Split(source, "\n")
	enums := map[string][]StateEvidence{}
	for i, line := range lines {
		m := pyEnumStart.FindStringSubmatch(line)
		if len(m) == 0 {
			continue
		}
		indent := leadingSpace(line)
		memberIndent := -1
		var states []StateEvidence
		for j := i + 1; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) == "" || leadingSpace(lines[j]) <= indent {
				break
			}
			if memberIndent < 0 {
				memberIndent = leadingSpace(lines[j])
			}
			if leadingSpace(lines[j]) != memberIndent {
				continue
			}
			name := strings.TrimSpace(strings.SplitN(lines[j], "=", 2)[0])
			if isIdentifier(name) {
				states = append(states, StateEvidence{Name: name, Value: name, At: Evidence{Path: path, Line: j + 1, Text: strings.TrimSpace(lines[j])}})
			}
		}
		if len(states) > 1 {
			enums[m[1]] = states
		}
	}
	var out []Candidate
	for i, line := range lines {
		owner := pyClassStart.FindStringSubmatch(line)
		if len(owner) == 0 {
			continue
		}
		indent := leadingSpace(line)
		memberIndent := -1
		for j := i + 1; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) == "" || leadingSpace(lines[j]) <= indent {
				break
			}
			if memberIndent < 0 {
				memberIndent = leadingSpace(lines[j])
			}
			if leadingSpace(lines[j]) != memberIndent {
				continue
			}
			field := pyField.FindStringSubmatch(lines[j])
			if len(field) < 3 || !isLifecycleField(field[1]) || len(enums[field[2]]) < 2 {
				continue
			}
			out = append(out, declarationCandidate(path, scenarioID, "python", owner[1], field[1], field[2], j+1, lines[j], enums[field[2]]))
		}
	}
	return out
}

func inferJava(path, scenarioID, source string) []Candidate {
	lines := strings.Split(source, "\n")
	enums := map[string][]StateEvidence{}
	for i, line := range lines {
		m := javaEnumStart.FindStringSubmatch(line)
		if len(m) == 0 {
			continue
		}
		var states []StateEvidence
		body := strings.Join(lines[i:], "\n")
		open := strings.Index(body, "{")
		close := matchingBrace(body, open)
		if close < 0 {
			continue
		}
		constants := strings.SplitN(body[open+1:close], ";", 2)[0]
		for _, part := range strings.Split(constants, ",") {
			name := strings.TrimSpace(part)
			if loc := strings.IndexAny(name, "( "); loc >= 0 {
				name = name[:loc]
			}
			if javaEnumValue.MatchString(name) && javaEnumValue.FindString(name) == name {
				states = append(states, StateEvidence{Name: name, Value: name, At: Evidence{Path: path, Line: i + 1, Text: strings.TrimSpace(line)}})
			}
		}
		if len(states) > 1 {
			enums[m[1]] = states
		}
	}
	var out []Candidate
	var owner string
	depth := 0
	for i, line := range lines {
		if m := javaOwnerStart.FindStringSubmatch(line); len(m) > 0 {
			owner = m[1]
		}
		field := javaField.FindStringSubmatch(line)
		if len(field) == 3 && owner != "" && depth == 1 && isLifecycleField(field[2]) && len(enums[field[1]]) > 1 {
			out = append(out, declarationCandidate(path, scenarioID, "java", owner, field[2], field[1], i+1, line, enums[field[1]]))
		}
		depth += strings.Count(line, "{") - strings.Count(line, "}")
		if owner != "" && depth <= 0 {
			owner = ""
		}
	}
	return out
}

func declarationCandidate(path, scenarioID, language, owner, fieldName, fieldType string, line int, declaration string, states []StateEvidence) Candidate {
	packageName := filepath.ToSlash(filepath.Dir(strings.TrimPrefix(path, "scenarios/"+scenarioID+"/")))
	c := Candidate{ID: path + "#" + owner + "." + fieldName, ScenarioID: scenarioID, Package: packageName, Language: language, TypeName: owner, FieldName: fieldName, FieldType: fieldType, Declaration: Evidence{Path: path, Line: line, Text: strings.TrimSpace(declaration)}, States: states,
		Persistence: PersistenceClassification{Class: ClassUnknown, Reason: language + " declaration evidence does not prove durable persistence", Scope: "scenario production " + language + " source", ScanComplete: true}}
	c.BurdenScore, c.ScoreBreakdown = score(c)
	c.Confidence = confidence(c)
	return c
}

func leadingSpace(line string) int { return len(line) - len(strings.TrimLeft(line, " \t")) }
func isIdentifier(value string) bool {
	return value != "" && (value[0] == '_' || value[0] >= 'A' && value[0] <= 'Z' || value[0] >= 'a' && value[0] <= 'z')
}
