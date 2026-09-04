package programs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
)

// analyzeTimeout bounds the static analysis subprocess. Analysis is a parse and
// a symbol-table walk, so a program that exceeds this is pathological; a slow
// analyzer degrades to "no diagnostics" rather than delaying every submission.
const analyzeTimeout = 10 * time.Second

// nearestMatchFloor is the similarity a candidate must reach before it is
// offered as a suggestion. It is deliberately high: a wrong suggestion is worse
// than none, because a model acts on it. `KeyError` against this namespace
// scores far below the floor and therefore returns no suggestion at all.
const nearestMatchFloor = 0.62

// analysis is the contract emitted by kernel/host/analyze.py.
type analysis struct {
	OK         bool           `json:"ok"`
	Degraded   string         `json:"degraded"`
	Bound      []analysisName `json:"bound"`
	Free       []analysisName `json:"free"`
	Imports    []analysisName `json:"imports"`
	Shadowed   []analysisName `json:"shadowed"`
	Attributes []analysisName `json:"attributes"`
}

type analysisName struct {
	Name string `json:"name"`
	Line int32  `json:"line"`
}

// unresolvedNamePrefix marks the one diagnostic class that represents an agent
// reaching for a capability that does not exist. A protected-name assignment
// and a shadow warning are program-authoring mistakes about names that resolve
// perfectly well, so recording them would answer the Act denominator's question
// — "what did an agent try to invoke and could not" — with a false positive.
const unresolvedNamePrefix = "name "

// IsUnresolvedNameDiagnostic reports whether a diagnostic represents an
// unreachable capability rather than a misuse of a name that does resolve.
func IsUnresolvedNameDiagnostic(diagnostic *programsv1.Diagnostic) bool {
	return diagnostic.GetSeverity() == "error" && strings.HasPrefix(diagnostic.GetMessage(), unresolvedNamePrefix)
}

func IsProtectedNameMisuseDiagnostic(diagnostic *programsv1.Diagnostic) bool {
	return diagnostic.GetSeverity() == "error" && (strings.HasPrefix(diagnostic.GetMessage(), "import ") || strings.HasPrefix(diagnostic.GetMessage(), "protected runtime name "))
}

var protectedRuntimeNames = map[string]struct{}{
	"discover": {}, "recall": {}, "guide": {}, "validate": {}, "capture": {}, "ai": {},
	"agent": {}, "gather": {}, "describe": {}, "reachable": {}, "lib": {}, "vrooli": {},
	"__vrooli__": {},
}

// ResolveSource reports the names a program reads that resolve to nothing, and
// the assignments that shadow a governed binding.
//
// Scope analysis is delegated to kernel/host/analyze.py, which uses Python's own
// `symtable`. Re-implementing Python scoping here is what previously refused
// every program containing a `def`, `lambda`, comprehension, or `for` loop: the
// prior regex pass treated their bound names as unresolved globals.
//
// The pass is conservative by contract. When analysis is unavailable, degraded,
// or reports a syntax error, this returns no diagnostics and lets the kernel
// report the failure with its real cause. A false refusal of a correct program
// is the expensive error; a missed diagnostic only costs a runtime error the
// kernel already produces.
func ResolveSource(source string, known []string, analyzerPath string) []*programsv1.Diagnostic {
	result, err := runAnalyzer(source, analyzerPath)
	if err != nil || result == nil || !result.OK || result.Degraded != "" {
		return nil
	}

	knownSet := make(map[string]struct{}, len(known))
	for _, name := range known {
		knownSet[name] = struct{}{}
	}

	var diagnostics []*programsv1.Diagnostic
	for _, entry := range result.Imports {
		if _, protected := protectedRuntimeNames[entry.Name]; !protected {
			continue
		}
		diagnostics = append(diagnostics, &programsv1.Diagnostic{
			Severity: "error",
			Line:     entry.Line,
			Name:     entry.Name,
			Message:  fmt.Sprintf("import %q is unavailable because the runtime names are already bound; use the bound names directly", entry.Name),
		})
	}
	for _, entry := range result.Shadowed {
		if _, ok := knownSet[entry.Name]; !ok {
			continue
		}
		if _, protected := protectedRuntimeNames[entry.Name]; protected {
			diagnostics = append(diagnostics, &programsv1.Diagnostic{
				Severity: "error",
				Line:     entry.Line,
				Name:     entry.Name,
				Message:  fmt.Sprintf("protected runtime name %q cannot be assigned", entry.Name),
			})
			continue
		}
		diagnostics = append(diagnostics, &programsv1.Diagnostic{
			Severity: "warning",
			Line:     entry.Line,
			Name:     entry.Name,
			Message:  fmt.Sprintf("scenario namespace %q is shadowed for the rest of this session; reach the binding as __vrooli__.%s", entry.Name, entry.Name),
		})
	}

	for _, entry := range result.Free {
		if _, ok := knownSet[entry.Name]; ok || hasKnownDescendant(entry.Name, known) {
			continue
		}
		nearest := nearestName(entry.Name, known)
		message := fmt.Sprintf(unresolvedNamePrefix+"%q does not resolve to a governed binding namespace or a built-in", entry.Name)
		if nearest != "" {
			message += fmt.Sprintf("; nearest match: %q", nearest)
		}
		diagnostics = append(diagnostics, &programsv1.Diagnostic{
			Severity:     "error",
			Line:         entry.Line,
			Name:         entry.Name,
			Message:      message,
			NearestMatch: nearest,
		})
	}
	knownPaths := make(map[string]struct{}, len(known))
	knownRoots := make(map[string]struct{}, len(known))
	for _, name := range known {
		knownPaths[name] = struct{}{}
		knownRoots[strings.SplitN(name, ".", 2)[0]] = struct{}{}
	}
	for _, entry := range result.Attributes {
		if _, ok := knownPaths[entry.Name]; ok || hasKnownDescendant(entry.Name, known) {
			continue
		}
		root := strings.SplitN(entry.Name, ".", 2)[0]
		if _, ok := knownRoots[root]; !ok {
			continue
		}
		candidates := make([]string, 0)
		for name := range knownPaths {
			if strings.HasPrefix(name, root+".") {
				candidates = append(candidates, name)
			}
		}
		if len(candidates) == 0 {
			continue
		}
		nearest := nearestName(entry.Name, candidates)
		message := fmt.Sprintf(unresolvedNamePrefix+"%q does not resolve to a governed binding namespace or command", entry.Name)
		if nearest != "" {
			message += fmt.Sprintf("; nearest match: %q", nearest)
		}
		diagnostics = append(diagnostics, &programsv1.Diagnostic{Severity: "error", Line: entry.Line, Name: entry.Name, Message: message, NearestMatch: nearest})
	}

	sort.SliceStable(diagnostics, func(left, right int) bool {
		return diagnostics[left].GetLine() < diagnostics[right].GetLine()
	})
	return diagnostics
}

// hasKnownDescendant accepts the scenario and group segments of a qualified
// two-level binding namespace. The kernel resolves those intermediate objects
// at runtime; preflight must therefore validate the leaf path without treating
// `scenario.group` itself as an unresolved command.
func hasKnownDescendant(name string, known []string) bool {
	prefix := strings.TrimSuffix(name, ".") + "."
	for _, candidate := range known {
		if strings.HasPrefix(candidate, prefix) {
			return true
		}
	}
	return false
}

// DeclaredNames returns module-scope names that a previous submission made
// available to later submissions in the same persistent kernel. Preflight is
// intentionally session-aware: rejecting a persisted local as an unresolved
// capability would make the documented session-reuse contract impossible.
// Analyzer failure returns no names; execution remains the authority in that
// degraded case and reports a real runtime error if the name is absent.
func DeclaredNames(source, analyzerPath string) []string {
	result, err := runAnalyzer(source, analyzerPath)
	if err != nil || result == nil || !result.OK || result.Degraded != "" {
		return nil
	}
	names := make([]string, 0, len(result.Bound))
	for _, entry := range result.Bound {
		if entry.Name != "" {
			names = append(names, entry.Name)
		}
	}
	return names
}

func runAnalyzer(source, analyzerPath string) (*analysis, error) {
	if strings.TrimSpace(analyzerPath) == "" {
		return nil, fmt.Errorf("analyzer path is not configured")
	}
	python, err := pythonInterpreter()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), analyzeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, python, "-I", "-S", analyzerPath)
	cmd.Stdin = strings.NewReader(source)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run analyzer: %w", err)
	}
	var result analysis
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("decode analyzer result: %w", err)
	}
	return &result, nil
}

// nearestName offers the closest known name by normalized edit distance.
//
// The prior implementation scored shared characters without regard to order or
// length, so `KeyError` matched `discover` (both contain e, r, o, c) and `text`
// matched `agent`. Those suggestions actively misdirect a model, which is why
// the metric is now positional and the floor is high enough to return nothing
// when no candidate is genuinely close.
func nearestName(name string, known []string) string {
	candidates := append([]string(nil), known...)
	sort.Strings(candidates)
	best, score := "", 0.0
	for _, candidate := range candidates {
		value := similarity(name, candidate)
		if value > score {
			best, score = candidate, value
		}
	}
	if score < nearestMatchFloor {
		return ""
	}
	return best
}

// similarity is 1 - (levenshtein / longest), case-insensitive.
func similarity(left, right string) float64 {
	if left == right {
		return 1
	}
	if left == "" || right == "" {
		return 0
	}
	lower, upper := []rune(strings.ToLower(left)), []rune(strings.ToLower(right))
	previous := make([]int, len(upper)+1)
	current := make([]int, len(upper)+1)
	for index := range previous {
		previous[index] = index
	}
	for i := 1; i <= len(lower); i++ {
		current[0] = i
		for j := 1; j <= len(upper); j++ {
			cost := 1
			if lower[i-1] == upper[j-1] {
				cost = 0
			}
			current[j] = minInt(minInt(current[j-1]+1, previous[j]+1), previous[j-1]+cost)
		}
		copy(previous, current)
	}
	longest := maxInt(len(lower), len(upper))
	return 1 - float64(previous[len(upper)])/float64(longest)
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
