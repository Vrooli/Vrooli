package cliutil

import (
	"encoding/json"
	"fmt"
	"time"
)

// Durable park/resume — producer-side park primitive.
//
// When a producer CLI (test-genie `runs wait`, git-control-tower `baseline
// diff`, …) is invoked INSIDE an agent-manager run, it cannot reliably block on
// externally-owned async work: an LLM agentic loop has no native blocking
// primitive and the harness's tool-timeout / background semantics are outside
// our control. Instead the producer asks agent-manager to PARK the run — the
// agent process exits (zero tokens), agent-manager performs the blocking wait
// on the agent's behalf via its per-producer Waiter, and WAKES the run by
// resuming the conversation with the result injected as the next turn.
//
// This primitive is the single producer-side entry point: a producer calls
// ParkForAwait with its (producer, key) and, if parked, prints the returned
// clean tool-result message and exits 0. Outside an agent-manager run the
// primitive is a no-op (parked=false) and the producer falls back to its normal
// blocking behaviour — so raw-terminal / human callers are unchanged.

// Park producer keys. These MUST match the producer keys agent-manager registers
// for its Waiters (scenarios/agent-manager/api/internal/orchestration/waiter.go:
// ProducerTestGenie / ProducerGCT) so the await-handle is dispatched to the
// right Waiter on wake.
const (
	ParkProducerTestGenie = "test-genie"
	ParkProducerGCT       = "git-control-tower"
)

// ParkRequest describes externally-owned async work the current run wants to
// await by parking rather than blocking.
type ParkRequest struct {
	// Producer is the agent-manager Waiter key (one of ParkProducer*).
	Producer string
	// Key is the producer-scoped identifier of the awaited work, encoded as
	// "<scenario>/<id>" — the encoding agent-manager's splitProducerKey parses
	// (e.g. "web-console/20260625-..." for a test-genie run, "web-console/pre-launch"
	// for a baseline diff).
	Key string
	// Deadline optionally bounds the wait. The zero value lets agent-manager
	// apply its default park TTL.
	Deadline time.Time
}

// ParkResult is the outcome of a successful park.
type ParkResult struct {
	// Message is the clean tool-result text agent-manager wants the in-run command
	// to print before the turn ends. Printing it (and exiting 0) records the
	// "parked" outcome for the agent; agent-manager then terminates the process
	// group to end the turn.
	Message string
}

// parkResponse mirrors the snake_case protojson of agent-manager's
// ParkRunResponse (handlers write via protoconv.MarshalJSON, UseProtoNames=true).
type parkResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ParkForAwait asks agent-manager to park the current run on the given async
// work. Return contract:
//
//   - (nil, false, nil): the caller is NOT inside an agent-manager run (no
//     identity token present). The caller must fall back to its normal blocking
//     wait — behaviour for human / raw-terminal / non-AM callers is unchanged.
//   - (result, true, nil): agent-manager accepted the park. The caller should
//     print result.Message and exit 0; agent-manager ends the turn and will wake
//     the run with the result.
//   - (nil, true, err): the caller IS inside an agent-manager run but the park
//     call failed (AM unreachable, run not in a parkable state, auth error, …).
//     The caller should surface the warning and fall back to blocking — strictly
//     no worse than today's behaviour (graceful degradation).
func ParkForAwait(in ParkRequest) (*ParkResult, bool, error) {
	id := DetectIdentity()
	if !id.IsIdentityPresent() {
		// Not an agent-manager run (the strict signal park depends on). The
		// caller blocks normally.
		return nil, false, nil
	}

	// Resolve which run we are (the park endpoint keys off the run id in the
	// path). The identity token is opaque client-side, so we verify it to recover
	// the owning run id — this also confirms the token is currently valid.
	verified, err := id.VerifyIdentity()
	if err != nil {
		return nil, true, fmt.Errorf("park: resolve run identity: %w", err)
	}
	if verified == nil || !verified.Valid || verified.Claims == nil || verified.Claims.RunID == "" {
		reason := "identity token is not valid"
		if verified != nil && verified.Error != "" {
			reason = verified.Error
		}
		return nil, true, fmt.Errorf("park: %s", reason)
	}
	runID := verified.Claims.RunID

	baseURL := agentManagerAPIBase()
	if baseURL == "" {
		return nil, true, fmt.Errorf("park: agent-manager base URL not discoverable")
	}

	client := NewHTTPClient(HTTPClientOptions{
		BaseOptions: APIBaseOptions{Override: baseURL},
		Timeout:     15 * time.Second,
	})

	body := map[string]any{
		"producer":       in.Producer,
		"key":            in.Key,
		"identity_token": id.Token,
	}
	if !in.Deadline.IsZero() {
		body["deadline_unix"] = in.Deadline.Unix()
	}

	data, err := client.Do("POST", "/api/v1/runs/"+runID+"/park", nil, body)
	if err != nil {
		return nil, true, fmt.Errorf("park request: %w", err)
	}

	var resp parkResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, true, fmt.Errorf("park: parse response: %w", err)
	}
	if !resp.Success {
		return nil, true, fmt.Errorf("park: agent-manager declined to park the run")
	}
	return &ParkResult{Message: resp.Message}, true, nil
}

// agentManagerAPIBase resolves the agent-manager REST base URL for the park
// callback, mirroring IdentityEnv.VerifyIdentity's resolution so both callbacks
// agree on where agent-manager lives.
func agentManagerAPIBase() string {
	return DetermineAPIBase(APIBaseOptions{
		EnvVars: []string{
			"AGENT_MANAGER_API_BASE",
			"AGENT_MANAGER_API_URL",
		},
		PortEnvVars:  []string{"AGENT_MANAGER_API_PORT"},
		PortDetector: detectAgentManagerPort,
	})
}
