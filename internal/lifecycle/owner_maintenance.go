package lifecycle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/scenario"
	amdomain "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

// ErrOwnerMaintenanceRequired means no lifecycle effect may follow this read.
// It is deliberately not an instruction to acquire elevated owner authority.
var (
	ErrOwnerMaintenanceRequired = errors.New("execution owner maintenance precondition failed")
)

type ownerAdmissionReadError struct {
	status int
	body   []byte
}

func (e *ownerAdmissionReadError) Error() string {
	return fmt.Sprintf("owner read returned HTTP %d", e.status)
}

// This is the existing AM admission projection, not a lifecycle configuration
// framework. Pointers distinguish a missing observation from a measured zero.
type ownerMaintenanceStanding struct {
	Closed             bool   `json:"closed"`
	Revision           int64  `json:"revision"`
	Admitting          *int   `json:"admitting"`
	Remaining          *int   `json:"remaining"`
	Drained            bool   `json:"drained"`
	LifecycleInterlock string `json:"lifecycleInterlock"`
	Inventory          *struct {
		Remaining *int              `json:"remaining"`
		Work      []json.RawMessage `json:"work"`
		Executors []json.RawMessage `json:"executors"`
		Unknown   []string          `json:"unknown"`
	} `json:"inventory"`
}

// requireOwnerMaintenance is called only while holding this instance's lifecycle
// lock. AM resume must take the same lock before its admission mutex; the owner
// advertises that installed contract in LifecycleInterlock. A GET alone cannot
// prevent admission from reopening between the observation and Stop.
func (r *Runner) requireOwnerMaintenance(ctx context.Context, item scenario.Scenario, revision int64) (int64, error) {
	if item.Slug != "agent-manager" {
		return 0, nil
	}
	refuse := func(reason string) (int64, error) {
		return 0, fmt.Errorf("%w for %q: %s; use the owner's explicit maintenance status/begin/drain and recovery protocol", ErrOwnerMaintenanceRequired, recordSlug(item), reason)
	}
	// Even a false empty owner inventory must not authorize an executor to
	// interrupt the owner of its own acceptance, continuation and result writes.
	if currentExecutorIdentified() {
		return refuse("the current executor is an identified agent")
	}
	read := r.deps.readOwnerMaintenance
	if read == nil {
		read = r.readOwnerMaintenance
	}
	state, err := read(ctxOrBackground(ctx), item)
	if err != nil {
		return refuse("owner admission is unknown: " + err.Error())
	}
	if !state.Closed || state.Revision <= 0 || state.Admitting == nil || *state.Admitting != 0 || state.Remaining == nil || *state.Remaining < 0 {
		return refuse("admission is not closed with a complete observation")
	}
	if state.LifecycleInterlock != "scenario-lock-v1" {
		return refuse("owner resume does not attest the lifecycle lock interlock")
	}
	inv := state.Inventory
	if inv == nil || inv.Remaining == nil || *inv.Remaining != *state.Remaining ||
		len(inv.Executors) != 0 || len(inv.Unknown) != 0 {
		return refuse("complete durable and physical executor exclusion is unavailable")
	}
	if revision != 0 && state.Revision != revision {
		return refuse("maintenance revision changed during lifecycle preparation")
	}
	if *state.Remaining == 0 {
		if !state.Drained || len(inv.Work) != 0 {
			return refuse("admission is not closed and completely drained")
		}
	} else {
		ids, err := accountingOnlyWork(state)
		if err != nil {
			return refuse(err.Error())
		}
		verify := r.deps.verifyOwnerAccounting
		if verify == nil {
			verify = r.verifyOwnerAccounting
		}
		if err := verify(ctx, item, ids); err != nil {
			return refuse("accounting-only restart proof unavailable: " + err.Error())
		}
	}
	return state.Revision, nil
}

// Accounting retention is not active execution. This exception never changes
// accounting or the owner's drained projection; every original child must be
// proved terminal through the owner, with physical exclusion already established.
func accountingOnlyWork(state ownerMaintenanceStanding) ([]string, error) {
	inv := state.Inventory
	if inv == nil || state.Remaining == nil || *state.Remaining <= 0 || *state.Remaining > 64 || len(inv.Work) != *state.Remaining || state.Drained {
		return nil, fmt.Errorf("remaining work is not a complete accounting-only inventory")
	}
	seen := map[string]bool{}
	var ids []string
	for _, raw := range inv.Work {
		var work struct{ ID, Kind, Status string }
		if err := json.Unmarshal(raw, &work); err != nil || work.Kind != "workflow" || work.Status != "cancelling" || seen[work.ID] {
			return nil, fmt.Errorf("remaining work includes active, unknown or duplicate identities")
		}
		if _, err := uuid.Parse(work.ID); err != nil {
			return nil, fmt.Errorf("remaining workflow identity is invalid")
		}
		seen[work.ID] = true
		ids = append(ids, work.ID)
	}
	return ids, nil
}

func (r *Runner) verifyOwnerAccounting(parent context.Context, item scenario.Scenario, ids []string) error {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	view, err := r.lookupRegistryRuntime(ctx, item)
	if err != nil || !view.Authoritative || view.Ports["API_PORT"] <= 0 {
		return fmt.Errorf("authoritative local owner binding unavailable")
	}
	return verifyAccountingOnlyOwner(ctx, view.Ports["API_PORT"], ids)
}

func verifyAccountingOnlyOwner(ctx context.Context, port int, ids []string) error {
	seenRuns := map[string]bool{}
	totalAttempts := 0
	for _, id := range ids {
		if _, err := uuid.Parse(id); err != nil {
			return fmt.Errorf("invalid workflow identity")
		}
		body, err := fetchOwnerBytes(ctx, port, "/api/v1/workflow-executions/"+id+"/trace?limit=1")
		if err != nil {
			return err
		}
		var envelope struct {
			Execution json.RawMessage
			Attempts  []json.RawMessage
		}
		if err := json.Unmarshal(body, &envelope); err != nil {
			return fmt.Errorf("decode original workflow trace: %w", err)
		}
		var x amdomain.WorkflowExecution
		if err := protojson.Unmarshal(envelope.Execution, &x); err != nil {
			return fmt.Errorf("decode original workflow: %w", err)
		}
		if x.GetId() != id || x.GetStatus() != amdomain.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLING || x.GetTerminalReason().GetCode() != "cancelled" || len(envelope.Attempts) == 0 || int(x.GetBudgetUsage().GetNodeAttempts()) != len(envelope.Attempts) {
			return fmt.Errorf("workflow %s has incomplete or non-cancelling execution evidence", id)
		}
		seenAttempts := map[string]bool{}
		for _, raw := range envelope.Attempts {
			var attempt amdomain.WorkflowNodeAttempt
			if err := protojson.Unmarshal(raw, &attempt); err != nil {
				return fmt.Errorf("decode original attempt: %w", err)
			}
			totalAttempts++
			if totalAttempts > 64 || attempt.GetExecutionId() != id || attempt.GetId() == "" || seenAttempts[attempt.GetId()] || attempt.GetRunId() == "" || attempt.GetChildExecutionId() != "" || (attempt.GetStatus() != "dispatched" && attempt.GetStatus() != "completed" && attempt.GetStatus() != "failed") {
				return fmt.Errorf("workflow %s has an incomplete, nested or uncertain dispatch binding", id)
			}
			seenAttempts[attempt.GetId()] = true
			runID := attempt.GetRunId()
			if _, err := uuid.Parse(runID); err != nil {
				return fmt.Errorf("invalid original child identity")
			}
			if seenRuns[runID] {
				continue // Continuations retain one original run identity.
			}
			seenRuns[runID] = true
			body, err := fetchOwnerBytes(ctx, port, "/api/v1/runs/"+runID)
			if err != nil {
				return err
			}
			var response struct{ Run json.RawMessage }
			if err := json.Unmarshal(body, &response); err != nil {
				return fmt.Errorf("decode original child: %w", err)
			}
			var run amdomain.Run
			if err := protojson.Unmarshal(response.Run, &run); err != nil {
				return fmt.Errorf("decode original run: %w", err)
			}
			terminal := run.GetStatus() == amdomain.RunStatus_RUN_STATUS_COMPLETE || run.GetStatus() == amdomain.RunStatus_RUN_STATUS_FAILED || run.GetStatus() == amdomain.RunStatus_RUN_STATUS_CANCELLED
			finalizing := run.GetFinalizationStatus() != amdomain.RunFinalizationStatus_RUN_FINALIZATION_STATUS_NONE && run.GetFinalizationStatus() != amdomain.RunFinalizationStatus_RUN_FINALIZATION_STATUS_SUCCEEDED && run.GetFinalizationStatus() != amdomain.RunFinalizationStatus_RUN_FINALIZATION_STATUS_FAILED
			if run.GetId() != runID || !terminal || run.GetEndedAt() == nil || finalizing {
				return fmt.Errorf("child %s is not an observed terminal run with inactive finalization", runID)
			}
		}
	}
	return nil
}

func currentExecutorIdentified() bool {
	if strings.TrimSpace(os.Getenv(cliutil.EnvIdentityToken)) != "" || strings.TrimSpace(os.Getenv("VROOLI_RUN_ID")) != "" {
		return true
	}
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok && strings.HasSuffix(key, "AGENT_TAG") && strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func (r *Runner) readOwnerMaintenance(parent context.Context, item scenario.Scenario) (ownerMaintenanceStanding, error) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	view, err := r.lookupRegistryRuntime(ctx, item)
	if err != nil {
		return ownerMaintenanceStanding{}, err
	}
	port := view.Ports["API_PORT"]
	if !view.Authoritative || port <= 0 {
		return ownerMaintenanceStanding{}, fmt.Errorf("no authoritative local owner API binding")
	}
	return fetchOwnerMaintenance(ctx, port)
}

func fetchOwnerMaintenance(ctx context.Context, port int) (ownerMaintenanceStanding, error) {
	var state ownerMaintenanceStanding
	body, err := fetchOwnerBytes(ctx, port, "/api/v1/maintenance/admission")
	if err != nil {
		return state, err
	}
	if err := json.Unmarshal(body, &state); err != nil {
		return state, fmt.Errorf("decode owner admission: %w", err)
	}
	return state, nil
}

func fetchOwnerBytes(ctx context.Context, port int, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d%s", port, path), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cache-Control", "no-cache")
	// This read is local, never follows redirects or proxies, and never sends
	// credentials. An unavailable contract is a refusal, not an auth exchange.
	transport := &http.Transport{Proxy: nil}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	const maxBytes = 256 << 10
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxBytes {
		return nil, fmt.Errorf("owner read exceeds bounded response size")
	}
	if response.StatusCode != http.StatusOK {
		return nil, &ownerAdmissionReadError{status: response.StatusCode, body: body}
	}
	return body, nil
}
