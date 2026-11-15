package programs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"program-runtime/internal/harness"
)

// AuthoringCase is one task in the versioned corpus. The oracle checks the
// *result* a program produced, never the program text, so the corpus survives a
// namespace change: the same task is still the same task when the syntax moves.
type AuthoringCase struct {
	ID      string            `json:"id"`
	Task    string            `json:"task"`
	Setup   string            `json:"setup,omitempty"`
	Oracle  AuthoringOracle   `json:"oracle"`
	Fixture *AuthoringFixture `json:"fixture,omitempty"`
	Notes   string            `json:"notes"`
}

type AuthoringOracle struct {
	Kind           string             `json:"kind"`
	Keys           []string           `json:"keys"`
	Min            *int               `json:"min"`
	Text           string             `json:"text"`
	Value          any                `json:"value"`
	Relation       string             `json:"relation"`
	Key            string             `json:"key"`
	Cause          string             `json:"cause"`
	Assertions     []JSONAssertion    `json:"json_assertions,omitempty"`
	MaxOutputBytes int                `json:"max_output_bytes,omitempty"`
	CallCounts     []FixtureCallCount `json:"call_counts,omitempty"`
}

type JSONAssertion struct {
	Path      string `json:"path"`
	Op        string `json:"op"`
	Value     any    `json:"value,omitempty"`
	OtherPath string `json:"other_path,omitempty"`
}

type AuthoringFixture struct {
	Bindings map[string][]FixtureReply `json:"bindings,omitempty"`
}
type FixtureReply struct {
	Status int            `json:"status"`
	Body   map[string]any `json:"body"`
}
type FixtureCallCount struct {
	BindingID string         `json:"binding_id"`
	Count     int            `json:"count"`
	Args      map[string]any `json:"args,omitempty"`
	Rows      string         `json:"rows,omitempty"`
}

type FixtureCallTrace struct {
	BindingID string
	Args      map[string]any
	Rows      string
}

type AuthoringCaseExecution struct {
	SessionID         string
	FixtureCalls      map[string]int32
	FixtureRequests   []FixtureCallTrace
	FixtureViolations []string
}

type AuthoringSuite struct {
	Name  string          `json:"name"`
	Floor int             `json:"floor"`
	Cases []AuthoringCase `json:"cases"`
}

// AuthoringCaseResult records what happened for one case, including the model
// that authored the program, so a score is always attributable.
type AuthoringCaseResult struct {
	CaseID                  string
	Authored                bool
	FirstAttempt            bool
	Cause                   string
	AgentBytes              int64
	Model                   string
	AuthorProvider          string
	FailureDetail           string
	RuleID                  string
	TaskSHA256              string
	SourceSHA256            string
	GenerationElapsedMS     int64
	ExecutionElapsedMS      int64
	OracleElapsedMS         int64
	ElapsedToFirstWorkingMS *int64
	CaseStartOffsetMS       int64
	CaseStatus              string
	AttemptCount            int32
	RetryCount              int32
	FixtureCalls            map[string]int32
}

type AuthoringRuleMiss struct {
	RuleID string
	Count  int32
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
	FloorMet    bool
	// NotAttempted counts cases skipped because the response deadline
	// approached. A partial run states this rather than reporting a low score
	// as if the whole corpus had been measured.
	NotAttempted int32
	// HarnessStamp names the brief version this score was measured against, so
	// two runs are only comparable when their stamps match.
	HarnessStamp        string
	Cases               []AuthoringCaseResult
	RuleMisses          []AuthoringRuleMiss
	ReceiptVersion      string
	CorpusSHA256        string
	InstructionSHA256   string
	RuntimeRevision     string
	AuthorRole          string
	AuthorSchemaSHA256  string
	RoutePolicyRevision string
	StartedAt           string
	ElapsedMS           int64
	TotalBudgetMS       int64
	PerCaseReserveMS    int64
}

const (
	AuthoringReceiptVersion = "program-runtime.authoring-receipt.v1"
	AuthoringRole           = "author.generator"
	AuthoringSchemaJSON     = `{"type":"object","properties":{"source":{"type":"string"}},"required":["source"]}`
	unknownRevision         = "unknown"
)

// perCaseReserve is the headroom the eval keeps before starting another case.
// One case is an authoring round-trip plus a kernel spawn plus the program's
// own work; starting one with less than this remaining risks being cut off
// mid-write, which is the failure mode that made this eval unreadable.
const perCaseReserve = 75 * time.Second

// AuthoringDeps are the seams the eval needs. They are injected rather than
// constructed here so the harness is testable without a live fleet.
type AuthoringDeps struct {
	// Author asks the code-authoring model for one program. It returns the
	// program source and the resolved model id.
	Author               func(ctx context.Context, instruction, task string) (source string, model string, err error)
	AuthorWithProvenance func(ctx context.Context, instruction, task string) (source, provider, model string, err error)
	// RunCase submits an optional harness setup program and then the authored
	// program into one fresh session. Setup is used only by corpus cases that
	// explicitly test session persistence; it is never authored by the model.
	RunCase            func(ctx context.Context, setup, source string) (stdout string, cause string, agentBytes int64, failureDetail string, err error)
	RunCaseWithFixture func(ctx context.Context, setup, source string, fixture *AuthoringFixture) (stdout string, cause string, agentBytes int64, failureDetail string, execution AuthoringCaseExecution, err error)
	// SuitePath locates the corpus on disk.
	SuitePath string
}

// authoringInstruction renders the standing brief from the versioned harness
// contract.
//
// It was a string constant until the contract existed. A constant cannot be
// checked against the construction guide or the skill, so the brief silently
// omitted rules those surfaces documented — `gather`'s callable contract among
// them — and the eval then measured prompt omission rather than surface
// quality. internal/harness owns the rules, and its drift test keeps the three
// surfaces aligned.
func authoringInstruction() string { return harness.Load().Instruction() }

// authoringHarnessStamp identifies the brief a measurement was taken against, so
// a score is attributable to a harness version rather than to "the prompt at the
// time".
func authoringHarnessStamp() string { return harness.Load().Stamp() }

// RunAuthoringEval measures first-attempt authoring correctness against the
// versioned corpus.
//
// It reports `unavailable` with a stated reason only when a dependency is
// genuinely missing — never as a placeholder, and never as a zero score. A
// degraded run must be distinguishable from a bad run, because the floor is a
// gate and a fabricated zero would pass it trivially.
func RunAuthoringEval(ctx context.Context, deps AuthoringDeps) (result AuthoringResult) {
	runStarted := time.Now()
	result = AuthoringResult{
		ReceiptVersion:      AuthoringReceiptVersion,
		StartedAt:           runStarted.UTC().Format(time.RFC3339Nano),
		RuntimeRevision:     sourceRevision(),
		AuthorRole:          AuthoringRole,
		AuthorSchemaSHA256:  sha256Hex([]byte(AuthoringSchemaJSON)),
		RoutePolicyRevision: unknownRevision,
		PerCaseReserveMS:    perCaseReserve.Milliseconds(),
	}
	if deadline, ok := ctx.Deadline(); ok {
		result.TotalBudgetMS = time.Until(deadline).Milliseconds()
	}
	defer func() { result.ElapsedMS = time.Since(runStarted).Milliseconds() }()
	suitePath := deps.SuitePath
	if strings.TrimSpace(suitePath) == "" {
		suitePath = "evals/authoring.primary.json"
	}
	result.Suite = suitePath

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
	result.CorpusSHA256 = sha256Hex(raw)
	result.Floor = int32(suite.Floor)
	if len(suite.Cases) == 0 {
		result.Status = "unavailable"
		result.Reason = fmt.Sprintf("corpus at %s declares no cases", suitePath)
		result.Unavailable = 1
		return result
	}
	if (deps.Author == nil && deps.AuthorWithProvenance == nil) || (deps.RunCase == nil && deps.RunCaseWithFixture == nil) {
		result.Status = "unavailable"
		result.Reason = "authoring or submission seam is not configured"
		result.Unavailable = 1
		return result
	}

	result.HarnessStamp = authoringHarnessStamp()
	instruction := authoringInstruction()
	result.InstructionSHA256 = sha256Hex([]byte(instruction))

	for index, item := range suite.Cases {
		// The eval is one HTTP response, so it must not outrun the write
		// deadline: a run that is cut off mid-write returns `unexpected EOF`,
		// which is byte-identical to a missing dependency and discards work
		// that was actually performed. Stopping early and *saying so* keeps a
		// partial measurement usable and distinguishable from a whole one.
		if deadline, hasDeadline := ctx.Deadline(); hasDeadline && time.Until(deadline) < perCaseReserve {
			result.NotAttempted = int32(len(suite.Cases) - index)
			for _, pending := range suite.Cases[index:] {
				result.Cases = append(result.Cases, AuthoringCaseResult{CaseID: pending.ID, TaskSHA256: sha256Hex([]byte(pending.Task)), CaseStatus: "not_attempted"})
			}
			result.Status = "partial"
			result.Reason = fmt.Sprintf(
				"stopped after %d of %d cases to stay inside the response deadline; %d not attempted",
				index, len(suite.Cases), result.NotAttempted)
			finalizeAuthoringResult(&result)
			return result
		}
		caseStarted := time.Now()
		caseResult := AuthoringCaseResult{CaseID: item.ID, TaskSHA256: sha256Hex([]byte(item.Task)), CaseStartOffsetMS: time.Since(runStarted).Milliseconds(), CaseStatus: "measured", AttemptCount: 1}
		generationStarted := time.Now()
		var source, model, provider string
		var err error
		if deps.AuthorWithProvenance != nil {
			source, provider, model, err = deps.AuthorWithProvenance(ctx, instruction, item.Task)
		} else {
			source, model, err = deps.Author(ctx, instruction, item.Task)
		}
		caseResult.GenerationElapsedMS = time.Since(generationStarted).Milliseconds()
		caseResult.Model = model
		caseResult.AuthorProvider = provider
		source = stripSourceFence(source)
		if source != "" {
			caseResult.SourceSHA256 = sha256Hex([]byte(source))
		}
		if err != nil {
			// A model route that cannot be reached at all makes the whole run
			// unavailable rather than a low score: the surface was never
			// measured, so reporting a number would be a false verdict.
			if isRouteUnavailable(err) {
				result.Status = "unavailable"
				caseResult.CaseStatus = "unavailable"
				result.Reason = fmt.Sprintf("no ai-gateway code-authoring route resolved: %v", err)
				result.Unavailable = 1
				result.Cases = append(result.Cases, caseResult)
				for _, pending := range suite.Cases[index+1:] {
					result.Cases = append(result.Cases, AuthoringCaseResult{CaseID: pending.ID, TaskSHA256: sha256Hex([]byte(pending.Task)), CaseStatus: "not_attempted"})
					result.NotAttempted++
				}
				return result
			}
			caseResult.FailureDetail = err.Error()
			attributeAuthoringFailure(&caseResult, "authoring_error", caseResult.FailureDetail)
			result.Missed++
			result.Cases = append(result.Cases, caseResult)
			continue
		}
		caseResult.Authored = true

		executionStarted := time.Now()
		var stdout, cause, detail string
		var agentBytes int64
		var execution AuthoringCaseExecution
		if deps.RunCaseWithFixture != nil {
			stdout, cause, agentBytes, detail, execution, err = deps.RunCaseWithFixture(ctx, item.Setup, source, item.Fixture)
		} else {
			stdout, cause, agentBytes, detail, err = deps.RunCase(ctx, item.Setup, source)
		}
		caseResult.FixtureCalls = execution.FixtureCalls
		caseResult.ExecutionElapsedMS = time.Since(executionStarted).Milliseconds()
		caseResult.Cause = cause
		caseResult.AgentBytes = agentBytes
		caseResult.FailureDetail = detail
		if err != nil {
			if strings.TrimSpace(caseResult.FailureDetail) == "" {
				caseResult.FailureDetail = err.Error()
			}
			if strings.TrimSpace(caseResult.Cause) == "" {
				caseResult.Cause = "execution_error"
			}
			attributeAuthoringFailure(&caseResult, caseResult.Cause, caseResult.FailureDetail)
			result.Missed++
			result.Cases = append(result.Cases, caseResult)
			continue
		}
		oracleStarted := time.Now()
		finishOracle := func() {
			caseResult.OracleElapsedMS = time.Since(oracleStarted).Milliseconds()
			if caseResult.FirstAttempt {
				elapsed := time.Since(caseStarted).Milliseconds()
				caseResult.ElapsedToFirstWorkingMS = &elapsed
			}
		}
		if cause != "" {
			if item.Oracle.Kind == "cause_equals" && cause == item.Oracle.Cause {
				caseResult.FirstAttempt = true
				result.Met++
			} else {
				if strings.TrimSpace(caseResult.FailureDetail) == "" {
					caseResult.FailureDetail = fmt.Sprintf("case=%s; runtime cause=%s", item.ID, cause)
				}
				attributeAuthoringFailure(&caseResult, cause, caseResult.FailureDetail)
				result.Missed++
				if cause == "binding_unavailable" {
					caseResult.CaseStatus = "unavailable"
					result.Unavailable++
				}
			}
			finishOracle()
			result.Cases = append(result.Cases, caseResult)
			continue
		}
		if item.Oracle.Kind == "cause_equals" && containsCause(stdout, item.Oracle.Cause) {
			caseResult.FirstAttempt = true
			result.Met++
			finishOracle()
			result.Cases = append(result.Cases, caseResult)
			continue
		}
		if fixtureOracleSatisfied(item.Oracle, stdout, execution) || satisfiesOracle(item.Oracle, stdout) {
			caseResult.FirstAttempt = true
			result.Met++
		} else {
			caseResult.FailureDetail = fmt.Sprintf("case=%s; oracle=%s was not satisfied; stdout=%q", item.ID, item.Oracle.Kind, boundedDetail(stdout, 512))
			attributeAuthoringFailure(&caseResult, "wrong_result", caseResult.FailureDetail)
			result.WrongResult++
		}
		finishOracle()
		result.Cases = append(result.Cases, caseResult)
	}
	result.Status = "measured"
	result.FloorMet = result.Met >= result.Floor
	finalizeAuthoringResult(&result)
	return result
}

// AuthoringInferenceArgs is the single request builder for the constrained
// authoring call. Local-only is an explicit privacy boundary, not role order.
func AuthoringInferenceArgs(instruction, task string) map[string]any {
	return map[string]any{
		"source": task, "instruction": instruction, "role": AuthoringRole,
		"schema_json": AuthoringSchemaJSON, "profile": "PROFILE_LOCAL_ONLY",
	}
}

// ValidateAuthoringRoute rejects responses without attributable evidence of a
// recognized local provider and model. The gateway's local-only profile remains
// the routing authority; this allowlist makes its successful response auditable.
func ValidateAuthoringRoute(provider, model string) error {
	provider = strings.ToLower(strings.TrimSpace(provider))
	model = strings.TrimSpace(model)
	if model == "" {
		return fmt.Errorf("local-only authoring response omitted model (provider=%q)", provider)
	}
	switch provider {
	case "ollama":
		return nil
	default:
		return fmt.Errorf("local-only authoring response has unrecognized or non-local provider %q model %q", provider, model)
	}
}

func sha256Hex(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}

func sourceRevision() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return unknownRevision
	}
	var revision string
	var modified bool
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	if revision == "" || modified {
		return unknownRevision
	}
	return revision
}

func attributeAuthoringFailure(result *AuthoringCaseResult, cause, detail string) {
	result.Cause = strings.TrimSpace(cause)
	result.RuleID = harness.Load().ResolveFailure(result.Cause, detail)
}

func boundedDetail(detail string, limit int) string {
	if limit <= 0 || len(detail) <= limit {
		return detail
	}
	return detail[:limit] + "…"
}

func finalizeAuthoringResult(result *AuthoringResult) {
	counts := map[string]int32{}
	for _, item := range result.Cases {
		if item.RuleID != "" {
			counts[item.RuleID]++
		}
	}
	result.RuleMisses = result.RuleMisses[:0]
	for ruleID, count := range counts {
		result.RuleMisses = append(result.RuleMisses, AuthoringRuleMiss{RuleID: ruleID, Count: count})
	}
	sort.Slice(result.RuleMisses, func(left, right int) bool {
		return result.RuleMisses[left].RuleID < result.RuleMisses[right].RuleID
	})
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
	case "json_assertions":
		return fixtureOracleSatisfied(oracle, stdout, AuthoringCaseExecution{})
	case "stdout_json_keys":
		for _, key := range oracle.Keys {
			if !strings.Contains(stdout, "'"+key+"'") && !strings.Contains(stdout, "\""+key+"\"") {
				return false
			}
		}
		return true
	case "stdout_contains":
		return strings.Contains(stdout, oracle.expectedText())
	case "stdout_non_empty":
		return strings.TrimSpace(stdout) != ""
	case "row_count_relation":
		expected, ok := oracle.expectedCount()
		if !ok {
			return false
		}
		return compareCount(stdout, oracle.Relation, expected)
	case "handle_metadata":
		return containsJSONKey(stdout, oracle.Key)
	case "null_verdict":
		lower := strings.ToLower(stdout)
		return strings.Contains(lower, "null") || strings.Contains(lower, "nothing") || strings.Contains(lower, "no governed") || strings.Contains(lower, "no capability")
	case "null_verdict_or_success":
		return strings.TrimSpace(stdout) != ""
	case "row_count_min":
		if oracle.Min == nil {
			return false
		}
		return compareCount(stdout, "gte", *oracle.Min)
	default:
		return false
	}
}

func fixtureOracleSatisfied(oracle AuthoringOracle, stdout string, execution AuthoringCaseExecution) bool {
	if oracle.Kind != "json_assertions" {
		return false
	}
	if len(oracle.Assertions) == 0 && len(oracle.CallCounts) == 0 {
		return false
	}
	if oracle.MaxOutputBytes > 0 && len(stdout) > oracle.MaxOutputBytes {
		return false
	}
	var root any
	decoder := json.NewDecoder(strings.NewReader(stdout))
	decoder.UseNumber()
	if err := decoder.Decode(&root); err != nil {
		return false
	}
	if _, ok := root.(map[string]any); !ok {
		return false
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return false
	}
	for _, assertion := range oracle.Assertions {
		actual, exists := jsonPath(root, assertion.Path)
		other, otherExists := jsonPath(root, assertion.OtherPath)
		if !assertionMatches(actual, exists, assertion, other, otherExists) {
			return false
		}
	}
	for _, expected := range oracle.CallCounts {
		if execution.FixtureCalls[expected.BindingID] != int32(expected.Count) {
			return false
		}
		if expected.Args != nil || expected.Rows != "" {
			matched := int32(0)
			for _, call := range execution.FixtureRequests {
				if call.BindingID == expected.BindingID {
					if (expected.Rows != "" && call.Rows != expected.Rows) || !argsContain(call.Args, expected.Args) {
						return false
					}
					matched++
				}
			}
			if matched != int32(expected.Count) {
				return false
			}
		}
	}
	return true
}

func argsContain(actual, expected map[string]any) bool {
	for key, value := range expected {
		got, ok := actual[key]
		if !ok || !jsonEqual(got, value) {
			return false
		}
	}
	return true
}

func jsonPath(root any, path string) (any, bool) {
	if path == "" {
		return root, true
	}
	var current = root
	for _, part := range strings.Split(path, ".") {
		switch node := current.(type) {
		case map[string]any:
			value, ok := node[part]
			if !ok {
				return nil, false
			}
			current = value
		case []any:
			index := 0
			if _, err := fmt.Sscanf(part, "%d", &index); err != nil || index < 0 || index >= len(node) {
				return nil, false
			}
			current = node[index]
		default:
			return nil, false
		}
	}
	return current, true
}

func assertionMatches(actual any, exists bool, check JSONAssertion, other any, otherExists bool) bool {
	if check.Op == "exists" {
		return exists
	}
	if check.Op == "missing" {
		return !exists
	}
	if !exists {
		return false
	}
	if check.Op == "equal_path" {
		return otherExists && jsonEqual(actual, other)
	}
	switch check.Op {
	case "eq":
		return jsonEqual(actual, check.Value)
	case "ne":
		return !jsonEqual(actual, check.Value)
	case "contains":
		switch value := actual.(type) {
		case string:
			expected, ok := check.Value.(string)
			return ok && strings.Contains(value, expected)
		case []any:
			for _, item := range value {
				if jsonEqual(item, check.Value) {
					return true
				}
			}
		case map[string]any:
			key, ok := check.Value.(string)
			if ok {
				_, found := value[key]
				return found
			}
		}
	case "length_eq", "length_lte":
		expected, ok := numericFloat(check.Value)
		if !ok {
			return false
		}
		length := -1
		switch value := actual.(type) {
		case string:
			length = len(value)
		case []any:
			length = len(value)
		case map[string]any:
			length = len(value)
		}
		if length < 0 {
			return false
		}
		if check.Op == "length_eq" {
			return float64(length) == expected
		}
		return float64(length) <= expected
	case "gt", "gte", "lt", "lte":
		left, lok := numericFloat(actual)
		right, rok := numericFloat(check.Value)
		if !lok || !rok {
			return false
		}
		switch check.Op {
		case "gt":
			return left > right
		case "gte":
			return left >= right
		case "lt":
			return left < right
		default:
			return left <= right
		}
	}
	return false
}

func numericFloat(value any) (float64, bool) {
	switch number := value.(type) {
	case json.Number:
		f, err := number.Float64()
		return f, err == nil
	case float64:
		return number, true
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	}
	return 0, false
}

func jsonEqual(left, right any) bool {
	leftBytes, leftErr := json.Marshal(left)
	rightBytes, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftBytes) == string(rightBytes)
}

func (o AuthoringOracle) expectedText() string {
	if o.Text != "" {
		return o.Text
	}
	value, _ := o.Value.(string)
	return value
}

func (o AuthoringOracle) expectedCount() (int, bool) {
	switch value := o.Value.(type) {
	case float64:
		return int(value), value == float64(int(value))
	case int:
		return value, true
	case json.Number:
		parsed, err := value.Int64()
		return int(parsed), err == nil
	default:
		return 0, false
	}
}

func containsCause(stdout, cause string) bool {
	if cause == "" {
		return false
	}
	lower := strings.ToLower(stdout)
	if strings.Contains(lower, strings.ToLower(cause)) {
		return true
	}
	canonicalPhrases := map[string][]string{
		"ambiguous_response": {"ambiguous-response", "ambiguous response", "no determinable primary response field"},
		"refused_no_grant":   {"requires an explicit grant", "destructive binding"},
	}
	for _, phrase := range canonicalPhrases[cause] {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

func containsJSONKey(stdout, key string) bool {
	if key == "" {
		return false
	}
	return strings.Contains(stdout, fmt.Sprintf("%q:", key)) || strings.Contains(stdout, fmt.Sprintf("'%s':", key))
}

func compareCount(stdout, relation string, expected int) bool {
	for _, field := range strings.FieldsFunc(stdout, func(r rune) bool { return r < '0' || r > '9' }) {
		value := 0
		if _, err := fmt.Sscanf(field, "%d", &value); err != nil {
			continue
		}
		switch relation {
		case "gt":
			if value > expected {
				return true
			}
		case "gte":
			if value >= expected {
				return true
			}
		case "eq":
			if value == expected {
				return true
			}
		case "lt":
			if value < expected {
				return true
			}
		case "lte":
			if value <= expected {
				return true
			}
		}
	}
	return false
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
