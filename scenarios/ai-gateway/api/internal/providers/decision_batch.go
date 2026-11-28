package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DecisionBatchConcurrency is how many states a decision resource answers at
// once after its first state has loaded the model.
const DecisionBatchConcurrency = 4

// DecisionBatchItem is one state's outcome from a decision batch: the
// resource's normalized answer for that state, or the typed reason it could
// not answer it. LatencyMS is the resource's own round trip for the state.
type DecisionBatchItem struct {
	OutputText string
	LatencyMS  int64
	Err        error
}

// ExecuteDecisionBatch asks one question set about many states in a single
// resource command. The resource resolves the role, proves rights and admits
// placement once and reports each state separately, so a batch of N states
// costs one process instead of N. req carries the role, the rights-admission
// identity and the per-decision deadline; questionsJSON is the System One
// question object and states are the raw JSON states, in order.
//
// An error is a failure of the whole batch (the command refused or did not
// run); a failure of one state is that item's Err.
func (a Adapter) ExecuteDecisionBatch(ctx context.Context, req ExecutionRequest, questionsJSON string, states []json.RawMessage) ([]DecisionBatchItem, error) {
	if a.family() != FamilyDecision || a.HTTP != nil || a.Door != nil {
		return nil, &CommandError{Code: CodeUnconfigured, Command: a.Provider, ExitCode: -1, Err: fmt.Errorf("provider %q has no decision batch command", a.Provider)}
	}
	role := strings.TrimSpace(req.Role)
	if role == "" || len(states) == 0 || strings.TrimSpace(questionsJSON) == "" {
		return nil, &CommandError{Code: "invalid_request", Command: a.Provider, ExitCode: -1, Err: errors.New("a decision batch requires a role, questions and at least one state")}
	}
	stdin, err := json.Marshal(struct {
		Questions json.RawMessage   `json:"questions"`
		States    []json.RawMessage `json:"states"`
	}{json.RawMessage(questionsJSON), states})
	if err != nil {
		return nil, &CommandError{Code: "invalid_request", Command: a.Provider, ExitCode: -1, Err: fmt.Errorf("encode decision batch: %w", err)}
	}
	perDecision := req.Timeout
	if perDecision <= 0 {
		perDecision = DefaultCommandTimeout
	}
	args := append([]string{"gateway", "decide-batch", "--role", role, "--input-stdin", "--json",
		"--timeout", perDecision.String(), "--concurrency", strconv.Itoa(DecisionBatchConcurrency)}, ollayaIdentityArgs(req)...)
	result, err := a.runner().Run(ctx, Command{
		Name: a.CommandName, Args: args, Stdin: string(stdin),
		Timeout: decisionBatchDeadline(perDecision, len(states)) + resourceCommandGrace,
	})
	if err != nil {
		return nil, mapCommandError(a.Provider, err)
	}
	return decodeDecisionBatch(a.Provider, result.Stdout, len(states))
}

// decisionBatchDeadline bounds a whole batch: the first state alone (it may
// load the model), then the rest in waves of DecisionBatchConcurrency, each
// wave within one decision's deadline.
func decisionBatchDeadline(perDecision time.Duration, states int) time.Duration {
	waves := 1 + (states-1+DecisionBatchConcurrency-1)/DecisionBatchConcurrency
	return time.Duration(waves) * perDecision
}

func decodeDecisionBatch(provider, stdout string, want int) ([]DecisionBatchItem, error) {
	var wire struct {
		Results []struct {
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &wire); err != nil {
		return nil, &CommandError{Code: CodeProviderFailed, Command: provider, ExitCode: -1, Err: fmt.Errorf("decode decision batch: %w", err)}
	}
	if len(wire.Results) != want {
		return nil, &CommandError{Code: CodeProviderFailed, Command: provider, ExitCode: -1, Err: fmt.Errorf("decision batch answered %d of %d states", len(wire.Results), want)}
	}
	items := make([]DecisionBatchItem, want)
	for index, entry := range wire.Results {
		switch {
		case entry.Error != nil:
			items[index].Err = &CommandError{Code: batchItemCode(entry.Error.Code), Command: provider, ExitCode: -1, Err: errors.New(entry.Error.Message)}
		case len(bytes.TrimSpace(entry.Result)) == 0 || bytes.Equal(bytes.TrimSpace(entry.Result), []byte("null")):
			items[index].Err = &CommandError{Code: CodeProviderFailed, Command: provider, ExitCode: -1, Err: errors.New("decision batch item carried neither a result nor an error")}
		default:
			var timing struct {
				LatencyMS int64 `json:"latency_ms"`
			}
			_ = json.Unmarshal(entry.Result, &timing)
			items[index] = DecisionBatchItem{OutputText: string(entry.Result), LatencyMS: max(timing.LatencyMS, 0)}
		}
	}
	return items, nil
}

// batchItemCode keeps a per-state failure's typed code when it is one the
// marker contract knows, keeps a per-state deadline as "timeout", and reads
// anything else as a provider failure.
func batchItemCode(code string) string {
	if normalized := normalizeProviderErrorCode(code); normalized != "" {
		return normalized
	}
	if strings.TrimSpace(code) == "timeout" {
		return "timeout"
	}
	return CodeProviderFailed
}
