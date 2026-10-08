// Responsibility: retain services declarations within their original package.
package main

import (
	"encoding/json"
	"fmt"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"net/url"
	"strconv"
	"time"
)

// RunService handles run-related API operations.
type RunService struct {
	api *cliutil.APIClient
}

// List retrieves runs with optional filters.
func (s *RunService) List(limit, offset int, taskID, profileID, status, tagPrefix string) ([]byte, []*domainpb.Run, error) {
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		query.Set("offset", fmt.Sprintf("%d", offset))
	}
	if taskID != "" {
		query.Set("taskId", taskID)
	}
	if profileID != "" {
		query.Set("profileId", profileID)
	}
	if status != "" {
		query.Set("status", status)
	}
	if tagPrefix != "" {
		query.Set("tagPrefix", tagPrefix)
	}

	body, err := s.api.Get("/api/v1/runs", query)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.ListRunsResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Runs, nil
}

// Get retrieves a single run by ID.
// ListChildren lists the direct child runs of one parent run.
func (s *RunService) ListChildren(parentID string, limit int) ([]byte, []*domainpb.Run, error) {
	query := url.Values{}
	query.Set("parentRunId", parentID)
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	body, err := s.api.Get("/api/v1/runs", query)
	if err != nil {
		return body, nil, err
	}
	var resp apipb.ListRunsResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, err
	}
	return body, resp.Runs, nil
}

// Accounting reads one run's metered usage, including non-cache tokens.
func (s *RunService) Accounting(id string) (*apipb.RunAccounting, error) {
	body, err := s.api.Get("/api/v1/runs/"+id+"/accounting", nil)
	if err != nil {
		return nil, err
	}
	var accounting apipb.RunAccounting
	if err := unmarshalProtoResponse(body, &accounting); err != nil {
		return nil, err
	}
	return &accounting, nil
}

func (s *RunService) Get(id string) ([]byte, *domainpb.Run, error) {
	body, err := s.api.Get("/api/v1/runs/"+id, nil)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.GetRunResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Run, nil
}

// Attach registers an operator-started harness session and returns its
// scoped, one-time identity token.
func (s *RunService) Attach(req *apipb.AttachRunRequest) ([]byte, *apipb.AttachRunResponse, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/runs/attach", nil, payload)
	if err != nil {
		return body, nil, err
	}
	var resp apipb.AttachRunResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

// Detach closes an attached harness session without terminating its process.
func (s *RunService) Detach(id, reason string) ([]byte, *apipb.DetachRunResponse, error) {
	req := &apipb.DetachRunRequest{RunId: id}
	if reason != "" {
		req.Reason = proto.String(reason)
	}
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/runs/"+id+"/detach", nil, payload)
	if err != nil {
		return body, nil, err
	}
	var resp apipb.DetachRunResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

// GetReport retrieves the bounded shared run-report projection.
func (s *RunService) GetReport(id string) ([]byte, error) {
	return s.api.Get("/api/v1/runs/"+id+"/report", nil)
}

// Durability retrieves the evidence-bounded durability projection for one run.
func (s *RunService) Durability(id string) ([]byte, error) {
	return s.api.Get("/api/v1/runs/"+id+"/durability", nil)
}

// Stats returns the existing filtered run summary projection.
func (s *RunService) Stats(query url.Values) ([]byte, error) {
	return s.api.Get("/api/v1/stats/summary", query)
}

// Efficiency returns the bounded, read-only invocation efficiency projection.
func (s *RunService) Efficiency(query url.Values) ([]byte, error) {
	return s.api.Get("/api/v1/stats/efficiency", query)
}

// GetReceipts retrieves platform observations for one run.
func (s *RunService) GetReceipts(id string) ([]byte, error) {
	return s.api.Get("/api/v1/runs/"+id+"/observed-receipts", nil)
}

// Create creates a new run.
func (s *RunService) Create(req *apipb.CreateRunRequest) ([]byte, *domainpb.Run, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/runs", nil, payload)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.CreateRunResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Run, nil
}

// Stop stops a running execution.
func (s *RunService) Stop(id string) ([]byte, *apipb.StopRunResponse, error) {
	body, err := s.api.Request("POST", "/api/v1/runs/"+id+"/stop", nil, nil)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.StopRunResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

// GetByTag retrieves a run by its custom tag.
func (s *RunService) GetByTag(tag string) ([]byte, *domainpb.Run, error) {
	body, err := s.api.Get("/api/v1/runs/tag/"+tag, nil)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.GetRunByTagResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Run, nil
}

// StopByTag stops a run identified by its custom tag.
func (s *RunService) StopByTag(tag string) ([]byte, *apipb.StopRunByTagResponse, error) {
	body, err := s.api.Request("POST", "/api/v1/runs/tag/"+tag+"/stop", nil, nil)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.StopRunByTagResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

// StopAll stops all running runs, optionally filtered by tag prefix.
func (s *RunService) StopAll(req *apipb.StopAllRunsRequest) ([]byte, *domainpb.StopAllResult, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/runs/stop-all", nil, payload)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.StopAllRunsResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Result, nil
}

// Quiesce drains in-flight runs targeting a scenario so a Baseline Modes promote
// can re-point and restart its live instance (Baseline Modes P6).
func (s *RunService) Quiesce(req *apipb.QuiesceScenarioRequest) ([]byte, *apipb.QuiesceResult, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	// The owner enforces the requested drain timeout (five minutes by default).
	// Reuse the same request-local wait transport as workflow execution-wait.
	body, err := s.api.WithoutTimeout().Request("POST", "/api/v1/runs/quiesce", nil, payload)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.QuiesceScenarioResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Result, nil
}

// Approve approves a run.
func (s *RunService) Approve(id string, req *apipb.ApproveRunRequest) ([]byte, *domainpb.ApproveResult, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/runs/"+id+"/approve", nil, payload)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.ApproveRunResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Result, nil
}

// Reject rejects a run.
func (s *RunService) Reject(id string, req *apipb.RejectRunRequest) ([]byte, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, err
	}
	return s.api.Request("POST", "/api/v1/runs/"+id+"/reject", nil, payload)
}

// GetDiff retrieves the diff for a run.
func (s *RunService) GetDiff(id string) ([]byte, *domainpb.RunDiff, error) {
	body, err := s.api.Get("/api/v1/runs/"+id+"/diff", nil)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.GetRunDiffResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Diff, nil
}

// GetEvents retrieves events for a run.
func (s *RunService) GetEvents(id string, limit int, afterSequence *int64) ([]byte, []*domainpb.RunEvent, error) {
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	if afterSequence != nil {
		query.Set("after_sequence", fmt.Sprintf("%d", *afterSequence))
	}

	body, err := s.api.Get("/api/v1/runs/"+id+"/events", query)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.GetRunEventsResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Events, nil
}

// Delete removes a run.
func (s *RunService) Delete(id string) error {
	_, err := s.api.Request("DELETE", "/api/v1/runs/"+id, nil, nil)
	return err
}

// Continue continues an existing run with a follow-up message.
func (s *RunService) Continue(id string, req *domainpb.ContinueRunRequest) ([]byte, *domainpb.Run, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/runs/"+id+"/continue", nil, payload)
	if err != nil {
		return body, nil, err
	}

	var resp domainpb.ContinueRunResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Run, nil
}

// Park parks a run on externally-owned async work (durable park/resume).
func (s *RunService) Park(id string, req *domainpb.ParkRunRequest) ([]byte, *domainpb.ParkRunResponse, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/runs/"+id+"/park", nil, payload)
	if err != nil {
		return body, nil, err
	}

	var resp domainpb.ParkRunResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

// Wake wakes a parked run with a result (ops/manual recovery).
func (s *RunService) Wake(id string, req *domainpb.WakeRunRequest) ([]byte, *domainpb.WakeRunResponse, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/runs/"+id+"/wake", nil, payload)
	if err != nil {
		return body, nil, err
	}

	var resp domainpb.WakeRunResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

// WakeByKey wakes every run parked on one producer/key await handle and
// returns the IDs the server moved from parked to running.
func (s *RunService) WakeByKey(req *domainpb.WakeParkedRunsRequest) ([]string, error) {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return nil, err
	}
	body, err := s.api.Request("POST", "/api/v1/runs/wake-by-key", nil, payload)
	if err != nil {
		return nil, err
	}
	var resp domainpb.WakeParkedRunsResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return nil, err
	}
	return resp.GetWokenRunIds(), nil
}

// AwaitResult fetches a run's most recently resolved await result (the
// non-blocking re-fetch path). Pure read; never parks.
func (s *RunService) AwaitResult(id string) ([]byte, *domainpb.GetAwaitResultResponse, error) {
	body, err := s.api.Request("GET", "/api/v1/runs/"+id+"/await-result", nil, nil)
	if err != nil {
		return body, nil, err
	}

	var resp domainpb.GetAwaitResultResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

func (s *RunService) Recover(id string) ([]byte, *apipb.RecoverRunResponse, error) {
	body, err := s.api.Request("POST", "/api/v1/runs/"+id+"/recover", nil, nil)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.RecoverRunResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, &resp, nil
}

// CohortReport reads the bounded multi-run projection. The service deliberately
// accepts only explicit run IDs; it never requests a bulk transcript endpoint.
func (s *RunService) CohortReport(runIDs string) ([]byte, error) {
	return s.api.Request("GET", "/api/v1/runs/cohort-report", url.Values{"run_ids": []string{runIDs}}, nil)
}

func (s *RunService) CohortReportByName(name string) ([]byte, error) {
	return s.api.Request("GET", "/api/v1/runs/cohort-report", url.Values{"cohort": []string{name}}, nil)
}

func (s *RunService) GoalCohort(name string, limit int) ([]byte, error) {
	values := url.Values{"cohort": []string{name}}
	if limit > 0 {
		values.Set("limit", strconv.Itoa(limit))
	}
	return s.api.Request("GET", "/api/v1/runs/goal-cohort", values, nil)
}

func (s *RunService) InvocationFacts(id string) ([]byte, error) {
	return s.api.Request("GET", "/api/v1/runs/"+id+"/invocation-facts", nil, nil)
}

func (s *RunService) Episodes(id string) ([]byte, error) {
	return s.api.Request("GET", "/api/v1/runs/"+id+"/episodes", nil, nil)
}

func (s *RunService) MessageFriction(id string) ([]byte, error) {
	return s.api.Request("GET", "/api/v1/runs/"+id+"/messages-friction", nil, nil)
}

func (s *RunService) Ledger(id string, withProjections bool) ([]byte, error) {
	values := url.Values{}
	if withProjections {
		values.Set("with_projections", "true")
	}
	return s.api.Request("GET", "/api/v1/runs/"+id+"/ledger", values, nil)
}

func (s *RunService) EpisodeCohort(values url.Values) ([]byte, error) {
	return s.api.Request("GET", "/api/v1/runs/episode-cohort", values, nil)
}

func (s *RunService) CompareEpisodeCohorts(payload []byte, limit int) ([]byte, error) {
	values := url.Values{}
	if limit > 0 {
		values.Set("limit", strconv.Itoa(limit))
	}
	return s.api.Request("POST", "/api/v1/runs/episode-cohort/compare", values, payload)
}

func (s *RunService) EpisodeTrend(values url.Values) ([]byte, error) {
	return s.api.Request("GET", "/api/v1/runs/episode-trend", values, nil)
}

func (s *RunService) PublishRecurringFriction(values url.Values) ([]byte, error) {
	return s.api.Request("POST", "/api/v1/runs/episodes/publish-recurring", values, nil)
}

func (s *RunService) ImportTranscript(payload []byte) ([]byte, error) {
	return s.api.Request("POST", "/api/v1/runs/import-transcript", nil, payload)
}

func (s *RunService) ImportSessionCorpus(payload []byte) ([]byte, error) {
	return s.api.Request("POST", "/api/v1/runs/import-session-corpus", nil, payload)
}

func (s *RunService) BackfillLabels() ([]byte, error) {
	return s.api.Request("POST", "/api/v1/runs/backfill-labels", nil, nil)
}

func (s *RunService) BackfillSubjects() ([]byte, error) {
	return s.api.Request("POST", "/api/v1/runs/backfill-subjects", nil, nil)
}

// importSweepTimeout bounds the on-demand sweep generously. A sweep re-reads
// every governed harness transcript — measured at ~90s over ~4.9k files — so
// the CLI default would abort it partway and report a transport failure for
// work the server was completing normally.
const importSweepTimeout = 15 * time.Minute

// ImportSweep runs the scheduled importer's sweep now. It is idempotent, so a
// caller diagnosing a stale corpus can repeat it safely.
func (s *RunService) ImportSweep() ([]byte, error) {
	return s.api.WithTimeout(importSweepTimeout).Request("POST", "/api/v1/runs/import-sweep", nil, nil)
}

func (s *RunService) ReplayInvocationFacts(id string) ([]byte, error) {
	return s.api.Request("POST", "/api/v1/runs/"+id+"/invocation-facts/replay", nil, nil)
}

func (s *RunService) RefreshInvocationFacts(id string) ([]byte, error) {
	return s.api.Request("POST", "/api/v1/runs/"+id+"/invocation-facts/refresh", nil, nil)
}

func (s *RunService) ReplayInvocationCorpus(values url.Values) ([]byte, error) {
	return s.api.Request("POST", "/api/v1/runs/invocation-facts/replay", values, nil)
}

func (s *RunService) AggregateInvocationFacts(values url.Values) ([]byte, error) {
	return s.api.Request("GET", "/api/v1/runs/invocation-facts/aggregate", values, nil)
}

func (s *RunService) SelectInvocationCohort(values url.Values) ([]byte, error) {
	return s.api.Request("GET", "/api/v1/runs/invocation-facts/cohort", values, nil)
}

func (s *RunService) InvocationMetrics(values url.Values) ([]byte, error) {
	return s.api.Request("GET", "/api/v1/runs/invocation-facts/metrics", values, nil)
}

func (s *RunService) DefineCohort(payload []byte) ([]byte, error) {
	return s.api.Request("POST", "/api/v1/runs/cohorts", nil, payload)
}

func (s *RunService) ListCohorts() ([]byte, error) {
	return s.api.Request("GET", "/api/v1/runs/cohorts", nil, nil)
}

func (s *RunService) ShowCohort(name string, limit int) ([]byte, error) {
	values := url.Values{}
	if limit > 0 {
		values.Set("limit", strconv.Itoa(limit))
	}
	return s.api.Request("GET", "/api/v1/runs/cohorts/"+url.PathEscape(name), values, nil)
}

func (s *RunService) DeleteCohort(name string) ([]byte, error) {
	return s.api.Request("DELETE", "/api/v1/runs/cohorts/"+url.PathEscape(name), nil, nil)
}

// InvestigationApply creates a run that applies investigation recommendations.
func (s *RunService) InvestigationApply(req json.RawMessage) ([]byte, *domainpb.Run, error) {
	body, err := s.api.Request("POST", "/api/v1/runs/investigation-apply", nil, req)
	if err != nil {
		return body, nil, err
	}

	var resp apipb.CreateRunResponse
	if err := unmarshalProtoResponse(body, &resp); err != nil {
		return body, nil, nil
	}
	return body, resp.Run, nil
}

// SandboxSync syncs run state from a sandbox.
func (s *RunService) SandboxSync(id string, req json.RawMessage) ([]byte, error) {
	return s.api.Request("POST", "/api/v1/runs/"+id+"/sandbox-sync", nil, req)
}

// =============================================================================
// Runner Service
// =============================================================================
