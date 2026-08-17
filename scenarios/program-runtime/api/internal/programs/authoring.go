package programs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// AuthoringCase is one task in the versioned corpus. The oracle checks the
// *result* a program produced, never the program text, so the corpus survives a
// namespace change: the same task is still the same task when the syntax moves.
type AuthoringCase struct {
	ID     string          `json:"id"`
	Task   string          `json:"task"`
	Oracle AuthoringOracle `json:"oracle"`
	Notes  string          `json:"notes"`
}

type AuthoringOracle struct {
	Kind string   `json:"kind"`
	Keys []string `json:"keys"`
	Min  *int     `json:"min"`
	Text string   `json:"text"`
}

type AuthoringSuite struct {
	Name  string          `json:"name"`
	Floor int             `json:"floor"`
	Cases []AuthoringCase `json:"cases"`
}

// AuthoringCaseResult records what happened for one case, including the model
// that authored the program, so a score is always attributable.
type AuthoringCaseResult struct {
	CaseID        string
	Authored      bool
	FirstAttempt  bool
	Cause         string
	AgentBytes    int64
	Model         string
	FailureDetail string
}

type AuthoringResult struct {
	Suite       string
	Status      string
	Reason      string
	Floor       int32
	Met         int32
	Missed      int32
	WrongResult int32
	Unavailable int32
	Cases       []AuthoringCaseResult
}

// AuthoringDeps are the seams the eval needs. They are injected rather than
// constructed here so the harness is testable without a live fleet.
type AuthoringDeps struct {
	// Author asks the code-authoring model for one program. It returns the
	// program source and the resolved model id.
	Author func(ctx context.Context, instruction, task string) (source string, model string, err error)
	// RunCase submits a program into a fresh session and returns its terminal
	// stdout, cause, and agent-visible byte count.
	RunCase func(ctx context.Context, source string) (stdout string, cause string, agentBytes int64, failureDetail string, err error)
	// SuitePath locates the corpus on disk.
	SuitePath string
}

// authoringInstruction is the standing brief given to the authoring model. It
// names the namespace rules the corpus exercises, because a model that has
// never seen this runtime cannot infer them and the eval would then measure
// prompt omission rather than surface quality.
const authoringInstruction = `You write short Python programs for the Vrooli Program Runtime.

Rules:
- Scenario operations are flat top-level names: <scenario>.<group>.<command>, hyphens become underscores (search-hub is search_hub).
- vrooli. addresses the project CLI only, never a scenario.
- Results are Handle values. Use count(), head(n), filter(), group_by(), select(), sort(), meta().
- When a response has several repeated fields, pass rows="<field>".
- Runtime verbs: discover, recall, validate, capture, ai, agent, gather, describe, reachable, lib.
- print() only the small answer. Never print whole result sets.

Return only Python source. No markdown fence, no commentary.`

// RunAuthoringEval measures first-attempt authoring correctness against the
// versioned corpus.
//
// It reports `unavailable` with a stated reason only when a dependency is
// genuinely missing — never as a placeholder, and never as a zero score. A
// degraded run must be distinguishable from a bad run, because the floor is a
// gate and a fabricated zero would pass it trivially.
func RunAuthoringEval(ctx context.Context, deps AuthoringDeps) AuthoringResult {
	suitePath := deps.SuitePath
	if strings.TrimSpace(suitePath) == "" {
		suitePath = "evals/authoring.primary.json"
	}
	result := AuthoringResult{Suite: suitePath}

	raw, err := os.ReadFile(suitePath)
	if err != nil {
		result.Status = "unavailable"
		result.Reason = fmt.Sprintf("corpus is unreadable at %s: %v", suitePath, err)
		result.Unavailable = 1
		return result
	}
	var suite AuthoringSuite
	if err := json.Unmarshal(raw, &suite); err != nil {
		result.Status = "unavailable"
		result.Reason = fmt.Sprintf("corpus at %s is not valid JSON: %v", suitePath, err)
		result.Unavailable = 1
		return result
	}
	result.Floor = int32(suite.Floor)
	if len(suite.Cases) == 0 {
		result.Status = "unavailable"
		result.Reason = fmt.Sprintf("corpus at %s declares no cases", suitePath)
		result.Unavailable = 1
		return result
	}
	if deps.Author == nil || deps.RunCase == nil {
		result.Status = "unavailable"
		result.Reason = "authoring or submission seam is not configured"
		result.Unavailable = 1
		return result
	}

	for _, item := range suite.Cases {
		caseResult := AuthoringCaseResult{CaseID: item.ID}
		source, model, err := deps.Author(ctx, authoringInstruction, item.Task)
		caseResult.Model = model
		if err != nil {
			// A model route that cannot be reached at all makes the whole run
			// unavailable rather than a low score: the surface was never
			// measured, so reporting a number would be a false verdict.
			if isRouteUnavailable(err) {
				result.Status = "unavailable"
				result.Reason = fmt.Sprintf("no ai-gateway code-authoring route resolved: %v", err)
				result.Unavailable = 1
				result.Cases = append(result.Cases, caseResult)
				return result
			}
			caseResult.FailureDetail = err.Error()
			result.Missed++
			result.Cases = append(result.Cases, caseResult)
			continue
		}
		caseResult.Authored = true
		source = stripSourceFence(source)

		stdout, cause, agentBytes, detail, err := deps.RunCase(ctx, source)
		caseResult.Cause = cause
		caseResult.AgentBytes = agentBytes
		caseResult.FailureDetail = detail
		if err != nil || cause != "" {
			result.Missed++
			result.Cases = append(result.Cases, caseResult)
			continue
		}
		if satisfiesOracle(item.Oracle, stdout) {
			caseResult.FirstAttempt = true
			result.Met++
		} else {
			result.WrongResult++
		}
		result.Cases = append(result.Cases, caseResult)
	}
	result.Status = "measured"
	sort.SliceStable(result.Cases, func(left, right int) bool {
		return result.Cases[left].CaseID < result.Cases[right].CaseID
	})
	return result
}

func isRouteUnavailable(err error) bool {
	message := strings.ToLower(err.Error())
	for _, marker := range []string{"unreachable", "not governed", "no route", "unavailable", "connection refused", "no such host"} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

// stripSourceFence removes a markdown fence a model may add despite the brief.
// Refusing a fenced program would measure formatting compliance rather than
// authoring correctness.
func stripSourceFence(source string) string {
	trimmed := strings.TrimSpace(source)
	if !strings.HasPrefix(trimmed, "```") {
		return source
	}
	lines := strings.Split(trimmed, "\n")
	if len(lines) < 2 {
		return source
	}
	lines = lines[1:]
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

// satisfiesOracle checks the produced result, never the program text.
func satisfiesOracle(oracle AuthoringOracle, stdout string) bool {
	switch oracle.Kind {
	case "stdout_json_keys":
		for _, key := range oracle.Keys {
			if !strings.Contains(stdout, "'"+key+"'") && !strings.Contains(stdout, "\""+key+"\"") {
				return false
			}
		}
		return true
	case "stdout_contains":
		return strings.Contains(stdout, oracle.Text)
	case "stdout_non_empty":
		return strings.TrimSpace(stdout) != ""
	case "row_count_min":
		if oracle.Min == nil {
			return strings.TrimSpace(stdout) != ""
		}
		return countDigitsAtLeast(stdout, *oracle.Min)
	default:
		return strings.TrimSpace(stdout) != ""
	}
}

func countDigitsAtLeast(stdout string, minimum int) bool {
	fields := strings.FieldsFunc(stdout, func(r rune) bool { return r < '0' || r > '9' })
	for _, field := range fields {
		value := 0
		if _, err := fmt.Sscanf(field, "%d", &value); err == nil && value >= minimum {
			return true
		}
	}
	return false
}

// DefaultSuitePath resolves the corpus beside the running scenario.
func DefaultSuitePath(repoRoot string) string {
	return filepath.Join(repoRoot, "scenarios", "program-runtime", "evals", "authoring.primary.json")
}

// AuthoringHTTPTimeout bounds one authoring call.
const AuthoringHTTPTimeout = 3 * time.Minute

// AuthoringHTTPClient is the client used for authoring model calls.
func AuthoringHTTPClient() *http.Client { return &http.Client{Timeout: AuthoringHTTPTimeout} }
