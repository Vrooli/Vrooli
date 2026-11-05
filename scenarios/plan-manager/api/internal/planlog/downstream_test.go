package planlog

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	heartbeatv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat"
	heartbeatconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/heartbeat/heartbeat_v1connect"
	"google.golang.org/protobuf/types/known/structpb"

	planmodel "plan-manager/internal/planmodel"
)

type staticURLResolver map[string]string

func (r staticURLResolver) ResolveScenarioURLDefault(_ context.Context, scenario string) (string, error) {
	if u := r[scenario]; u != "" {
		return u, nil
	}
	return "", errors.New("missing scenario URL")
}

func TestScenarioQABugReporterForwardsFullPayloadAndDraftDisposition(t *testing.T) {
	var got bugCaptureRequest
	owner := &bugCaptureOwner{t: t, got: &got}
	path, handler := heartbeatconnect.NewHeartbeatServiceHandler(owner)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()

	ref, err := NewScenarioQABugReporter(server.Client(), staticURLResolver{promptManagerSystem: server.URL}).FileBug(context.Background(), Entry{ID: "entry-1", Title: "cache drift", Bug: planmodel.BugReportPayload{SignalType: "regression", Severity: "major", Repro: []string{"start service"}, Expected: "fresh cache", Actual: "stale cache", Description: "details", Context: map[string]string{"scenario": "plan-manager"}, HonestyFlags: []string{"minimal-context"}}})
	if err != nil {
		t.Fatal(err)
	}
	if got.SignalType != "regression" || got.Severity != "major" || got.Actual != "stale cache" || got.Context["scenario"] != "plan-manager" {
		t.Fatalf("payload was reshaped: %#v", got)
	}
	if got.IdempotencyKey != "entry-1" || len(got.Repro) != 1 || len(got.HonestyFlags) != 1 {
		t.Fatalf("capture identity or evidence lost: %#v", got)
	}
	if ref.Reference != "bug-1" || ref.Capture.State != "draft" || ref.Capture.DraftID != "bug-1" || len(ref.Capture.NextAction) == 0 {
		t.Fatalf("capture = %#v", ref)
	}
}

type bugCaptureOwner struct {
	heartbeatconnect.UnimplementedHeartbeatServiceHandler
	t   *testing.T
	got *bugCaptureRequest
	err error
}

func (h *bugCaptureOwner) CaptureBug(_ context.Context, req *connect.Request[heartbeatv1.TeamMutationRequest]) (*connect.Response[heartbeatv1.JsonResponse], error) {
	h.t.Helper()
	if h.err != nil {
		return nil, h.err
	}
	if req.Msg.GetTeamId() != "scenario-qa" {
		h.t.Errorf("team = %q", req.Msg.GetTeamId())
	}
	attribution, err := base64.StdEncoding.DecodeString(req.Header().Get("X-Vrooli-Attribution"))
	if err != nil || !json.Valid(attribution) {
		h.t.Errorf("missing or invalid writer attribution: %q, %v", attribution, err)
	}
	var identity map[string]any
	if err := json.Unmarshal(attribution, &identity); err != nil || identity["kind"] != "writer-skill" || identity["source_skill_id"] != "report-bug" || identity["team_id"] != "scenario-qa" {
		h.t.Errorf("writer attribution changed: %q, %v", attribution, err)
	}
	encoded, err := json.Marshal(req.Msg.GetBody().AsInterface())
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(encoded, h.got); err != nil {
		return nil, err
	}
	data, err := structpb.NewValue(map[string]any{"disposition": "draft", "draft_id": "bug-1", "needs": []any{"actual"}, "next_action": []any{"prompt-manager", "team", "bug-repair", "scenario-qa", "bug-1"}})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&heartbeatv1.JsonResponse{Data: data}), nil
}

func TestScenarioQABugReporterPreservesOwnerFailureDisposition(t *testing.T) {
	for _, test := range []struct {
		name        string
		code        connect.Code
		unavailable bool
	}{
		{"unavailable", connect.CodeUnavailable, true},
		{"deadline", connect.CodeDeadlineExceeded, true},
		{"rejected observation", connect.CodeInvalidArgument, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			owner := &bugCaptureOwner{t: t, err: connect.NewError(test.code, errors.New("owner diagnosis"))}
			path, handler := heartbeatconnect.NewHeartbeatServiceHandler(owner)
			mux := http.NewServeMux()
			mux.Handle(path, handler)
			server := httptest.NewServer(mux)
			defer server.Close()
			ref, err := NewScenarioQABugReporter(server.Client(), staticURLResolver{promptManagerSystem: server.URL}).FileBug(context.Background(), Entry{ID: "same-entry"})
			var unavailable ErrDownstreamUnavailable
			if err == nil || errors.As(err, &unavailable) != test.unavailable || ref.Reference != "" {
				t.Fatalf("owner failure became ref=%#v err=%v", ref, err)
			}
		})
	}
}

func TestSwarmRecordWriterForwardsClassificationAndPublishedDisposition(t *testing.T) {
	var got recordCaptureRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/records/capture" || r.Method != http.MethodPost {
			t.Fatalf("request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(recordCaptureResponse{Disposition: "published", Record: recordResponse{ID: "rec-1"}})
	}))
	defer server.Close()

	ref, err := NewSwarmRecordWriter(server.Client(), staticURLResolver{swarmManagerSystem: server.URL}).WriteRecord(context.Background(), Entry{ID: "entry-1", Record: planmodel.RecordPayload{Kind: "refactor", Scenario: "swarm-manager", Trigger: "test failure", Approach: "repair", Evidence: "go test", Outcome: "partial", CreatedBy: "agent-7"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != "refactor" || got.Scenario != "swarm-manager" || got.Outcome != "partial" || got.CreatedBy != "agent-7" {
		t.Fatalf("classification was reshaped: %#v", got)
	}
	if ref.Reference != "rec-1" || ref.Capture.State != "published" {
		t.Fatalf("capture = %#v", ref)
	}
}

func TestDownstreamAdaptersMapResolutionFailureToUnavailable(t *testing.T) {
	_, err := NewScenarioQABugReporter(http.DefaultClient, staticURLResolver{}).FileBug(context.Background(), Entry{ID: "entry-1"})
	var unavailable ErrDownstreamUnavailable
	if !errors.As(err, &unavailable) {
		t.Fatalf("FileBug error = %T %[1]v", err)
	}
	_, err = NewSwarmRecordWriter(http.DefaultClient, staticURLResolver{}).WriteRecord(context.Background(), Entry{ID: "entry-1"})
	if !errors.As(err, &unavailable) {
		t.Fatalf("WriteRecord error = %T %[1]v", err)
	}
}
