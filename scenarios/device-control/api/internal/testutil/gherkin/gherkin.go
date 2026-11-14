// Package gherkin runs plain-text .feature files as Go subtests. It supports
// Feature, Background, Scenario, Scenario Outline with Examples tables, and
// Given/When/Then/And/But steps matched by regular expression. It is test
// support only: production code must not import it.
package gherkin

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Suite maps step text to implementations over a scenario-specific world W.
type Suite[W any] struct {
	steps []stepDef[W]
}

type stepDef[W any] struct {
	pattern *regexp.Regexp
	run     func(t *testing.T, w *W, args []string)
}

// Step registers an implementation. The pattern is anchored; each capture group
// is passed to run as an argument. Given/When/Then keywords are not part of it.
func (s *Suite[W]) Step(pattern string, run func(t *testing.T, w *W, args []string)) {
	s.steps = append(s.steps, stepDef[W]{pattern: regexp.MustCompile("^" + pattern + "$"), run: run})
}

type step struct {
	keyword string
	text    string
	line    int
}

type scenario struct {
	name  string
	steps []step
	line  int
}

type feature struct {
	name       string
	background []step
	scenarios  []scenario
}

// RunFeatures executes every *.feature file in dir. Each scenario gets a fresh world
// from newWorld. A step with no implementation, or an implementation that
// fails, fails that scenario only.
func RunFeatures[W any](t *testing.T, dir string, newWorld func(t *testing.T) *W, register func(*Suite[W])) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.feature"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no .feature files in %s (err=%v)", dir, err)
	}
	sort.Strings(files)
	suite := &Suite[W]{}
	register(suite)
	for _, file := range files {
		parsed, err := parseFile(file)
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		t.Run(parsed.name, func(t *testing.T) {
			for _, sc := range parsed.scenarios {
				t.Run(sc.name, func(t *testing.T) {
					world := newWorld(t)
					for _, st := range append(append([]step{}, parsed.background...), sc.steps...) {
						runStep(t, suite, world, file, st)
					}
				})
			}
		})
	}
}

func runStep[W any](t *testing.T, suite *Suite[W], world *W, file string, st step) {
	t.Helper()
	for _, def := range suite.steps {
		if m := def.pattern.FindStringSubmatch(st.text); m != nil {
			def.run(t, world, m[1:])
			return
		}
	}
	t.Fatalf("%s:%d: no step implementation for %q", filepath.Base(file), st.line, st.keyword+" "+st.text)
}

func parseFile(path string) (feature, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return feature{}, err
	}
	return parse(string(raw))
}

var keywords = []string{"Given ", "When ", "Then ", "And ", "But "}

func parse(text string) (feature, error) {
	var f feature
	var current *scenario
	var outline bool
	var examples [][]string // header row first
	inExamples, inBackground := false, false

	flush := func() error {
		if current == nil {
			return nil
		}
		expanded, err := expand(*current, outline, examples)
		if err != nil {
			return err
		}
		f.scenarios = append(f.scenarios, expanded...)
		current, outline, examples, inExamples = nil, false, nil, false
		return nil
	}
	for i, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		n := i + 1
		switch {
		case line == "" || strings.HasPrefix(line, "#"):
		case strings.HasPrefix(line, "Feature:"):
			f.name = strings.TrimSpace(strings.TrimPrefix(line, "Feature:"))
		case strings.HasPrefix(line, "Background:"):
			if err := flush(); err != nil {
				return f, err
			}
			inBackground = true
		case strings.HasPrefix(line, "Scenario Outline:"), strings.HasPrefix(line, "Scenario:"):
			if err := flush(); err != nil {
				return f, err
			}
			inBackground = false
			outline = strings.HasPrefix(line, "Scenario Outline:")
			name := line[strings.Index(line, ":")+1:]
			current = &scenario{name: strings.TrimSpace(name), line: n}
		case strings.HasPrefix(line, "Examples:"):
			if current == nil || !outline {
				return f, fmt.Errorf("line %d: Examples outside a Scenario Outline", n)
			}
			inExamples = true
		case inExamples && strings.HasPrefix(line, "|"):
			examples = append(examples, splitRow(line))
		default:
			kw := ""
			for _, k := range keywords {
				if strings.HasPrefix(line, k) {
					kw = strings.TrimSpace(k)
					break
				}
			}
			if kw == "" {
				if current == nil && !inBackground && f.name != "" {
					continue // free-text Feature description
				}
				return f, fmt.Errorf("line %d: unrecognized line %q", n, line)
			}
			st := step{keyword: kw, text: strings.TrimSpace(line[len(kw):]), line: n}
			switch {
			case inBackground:
				f.background = append(f.background, st)
			case current != nil:
				current.steps = append(current.steps, st)
			default:
				return f, fmt.Errorf("line %d: step outside a Scenario", n)
			}
		}
	}
	if err := flush(); err != nil {
		return f, err
	}
	if f.name == "" {
		return f, fmt.Errorf("missing Feature:")
	}
	return f, nil
}

func splitRow(line string) []string {
	cells := strings.Split(strings.Trim(line, "|"), "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

// expand turns a Scenario Outline plus its Examples table into one concrete
// scenario per row, substituting <column> placeholders.
func expand(sc scenario, outline bool, examples [][]string) ([]scenario, error) {
	if !outline {
		return []scenario{sc}, nil
	}
	if len(examples) < 2 {
		return nil, fmt.Errorf("line %d: Scenario Outline %q needs an Examples table with at least one row", sc.line, sc.name)
	}
	header := examples[0]
	var out []scenario
	for r, row := range examples[1:] {
		if len(row) != len(header) {
			return nil, fmt.Errorf("line %d: Examples row %d has %d cells, want %d", sc.line, r+1, len(row), len(header))
		}
		concrete := scenario{name: fmt.Sprintf("%s [%d]", sc.name, r+1), line: sc.line}
		for _, st := range sc.steps {
			text := st.text
			for c, column := range header {
				text = strings.ReplaceAll(text, "<"+column+">", row[c])
			}
			concrete.steps = append(concrete.steps, step{keyword: st.keyword, text: text, line: st.line})
		}
		out = append(out, concrete)
	}
	return out, nil
}
