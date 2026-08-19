package promptmanager

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	httpx "github.com/vrooli/api-core/apihttptest"
	experimentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/experiments"
	experimentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/experiments/experiments_v1connect"
	skillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills"
	skillsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills/skills_v1connect"
)

type fakeSkillsService struct {
	skillsconnect.UnimplementedSkillsServiceHandler
	read     func(*skillsv1.ReadSkillsRequest) (*skillsv1.ReadSkillsResponse, error)
	list     func(*skillsv1.ListSkillsRequest) (*skillsv1.ListSkillsResponse, error)
	get      func(*skillsv1.GetSkillRequest) (*skillsv1.GetSkillResponse, error)
	update   func(*skillsv1.UpdateSkillRequest) (*skillsv1.UpdateSkillResponse, error)
	versions func(*skillsv1.ListSkillVersionsRequest) (*skillsv1.ListSkillVersionsResponse, error)
	revert   func(*skillsv1.RevertSkillRequest) (*skillsv1.RevertSkillResponse, error)
}

type fakeExperimentsService struct {
	experimentsconnect.UnimplementedExperimentsServiceHandler
	assign func(*experimentsv1.AssignExperimentRequest) (*experimentsv1.ExperimentAssignment, error)
	record func(*experimentsv1.RecordOutcomeRequest) (*experimentsv1.ExperimentOutcome, error)
}

func (f fakeExperimentsService) AssignExperiment(_ context.Context, req *connect.Request[experimentsv1.AssignExperimentRequest]) (*connect.Response[experimentsv1.ExperimentAssignment], error) {
	return unaryResponse(f.assign, req)
}

func (f fakeExperimentsService) RecordOutcome(_ context.Context, req *connect.Request[experimentsv1.RecordOutcomeRequest]) (*connect.Response[experimentsv1.ExperimentOutcome], error) {
	return unaryResponse(f.record, req)
}

func experimentTestHandler(service fakeExperimentsService) http.Handler {
	_, handler := experimentsconnect.NewExperimentsServiceHandler(service)
	return handler
}

func unaryResponse[Req, Res any](fn func(*Req) (*Res, error), req *connect.Request[Req]) (*connect.Response[Res], error) {
	response, err := fn(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func (f fakeSkillsService) ReadSkills(_ context.Context, req *connect.Request[skillsv1.ReadSkillsRequest]) (*connect.Response[skillsv1.ReadSkillsResponse], error) {
	return unaryResponse(f.read, req)
}

func (f fakeSkillsService) ListSkills(_ context.Context, req *connect.Request[skillsv1.ListSkillsRequest]) (*connect.Response[skillsv1.ListSkillsResponse], error) {
	return unaryResponse(f.list, req)
}

func (f fakeSkillsService) GetSkill(_ context.Context, req *connect.Request[skillsv1.GetSkillRequest]) (*connect.Response[skillsv1.GetSkillResponse], error) {
	return unaryResponse(f.get, req)
}

func (f fakeSkillsService) UpdateSkill(_ context.Context, req *connect.Request[skillsv1.UpdateSkillRequest]) (*connect.Response[skillsv1.UpdateSkillResponse], error) {
	return unaryResponse(f.update, req)
}

func (f fakeSkillsService) ListSkillVersions(_ context.Context, req *connect.Request[skillsv1.ListSkillVersionsRequest]) (*connect.Response[skillsv1.ListSkillVersionsResponse], error) {
	return unaryResponse(f.versions, req)
}

func (f fakeSkillsService) RevertSkill(_ context.Context, req *connect.Request[skillsv1.RevertSkillRequest]) (*connect.Response[skillsv1.RevertSkillResponse], error) {
	return unaryResponse(f.revert, req)
}

func newSkillTestClient(t *testing.T, service fakeSkillsService, fallback http.Handler) *HTTPClient {
	t.Helper()
	_, connectHandler := skillsconnect.NewSkillsServiceHandler(service)
	if fallback == nil {
		fallback = http.NotFoundHandler()
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/vrooli.prompt_manager.v1.skills.SkillsService/") {
			connectHandler.ServeHTTP(w, r)
			return
		}
		fallback.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	return NewHTTPClientWithResolver(func(context.Context) (string, error) { return server.URL, nil }, server.Client())
}

func TestNewHTTPClient_Defaults(t *testing.T) {
	client := NewHTTPClient()
	if client == nil {
		t.Fatal("expected client")
	}
	if client.baseURLResolver == nil {
		t.Fatal("expected baseURLResolver")
	}
	if client.httpClient == nil {
		t.Fatal("expected httpClient")
	}
}

func TestNewHTTPClientWithResolver_Defaults(t *testing.T) {
	client := NewHTTPClientWithResolver(nil, nil)
	if client.baseURLResolver == nil {
		t.Fatal("expected default resolver")
	}
	if client.httpClient == nil {
		t.Fatal("expected default http client")
	}
}

func TestHTTPClient_ReadSkill(t *testing.T) {
	client := newSkillTestClient(t, fakeSkillsService{read: func(req *skillsv1.ReadSkillsRequest) (*skillsv1.ReadSkillsResponse, error) {
		if len(req.GetIdentifiers()) != 1 || req.GetIdentifiers()[0] != "test-skill" {
			t.Errorf("expected identifier test-skill, got %v", req.GetIdentifiers())
		}
		if req.GetVariables()["ITEM_NAME"] != "my-item" {
			t.Errorf("expected variable ITEM_NAME=my-item, got %s", req.GetVariables()["ITEM_NAME"])
		}
		if req.GetOutput() != "combined" {
			t.Errorf("expected output combined, got %s", req.GetOutput())
		}
		return &skillsv1.ReadSkillsResponse{Combined: "rendered prompt content"}, nil
	}}, nil)

	result, err := client.ReadSkill(context.Background(), "test-skill", map[string]string{"ITEM_NAME": "my-item"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "rendered prompt content" {
		t.Errorf("expected 'rendered prompt content', got %q", result)
	}
}

func TestHTTPClient_ReadSkill_ServerError(t *testing.T) {
	client := newSkillTestClient(t, fakeSkillsService{read: func(*skillsv1.ReadSkillsRequest) (*skillsv1.ReadSkillsResponse, error) {
		return nil, connect.NewError(connect.CodeInternal, context.Canceled)
	}}, nil)

	_, err := client.ReadSkill(context.Background(), "test-skill", nil, false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "internal") {
		t.Errorf("expected Connect internal error, got %v", err)
	}
}

func TestHTTPClient_PublishFrictionUsesCanonicalIntakeAndAttribution(t *testing.T) {
	client := NewHTTPClientWithResolver(
		func(_ context.Context) (string, error) { return "http://localhost:12345", nil },
		httpx.DoerFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost || req.URL.Path != "/api/v1/teams/meta-optimization/knowledge" {
				t.Fatalf("request = %s %s", req.Method, req.URL.Path)
			}
			var body struct {
				Topic   string `json:"topic"`
				Content string `json:"content"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if want := "friction-inbox/toolchain/agent-manager-finding-abcdef012345"; body.Topic != want {
				t.Fatalf("topic = %q, want %q", body.Topic, want)
			}
			if !strings.Contains(body.Content, "honesty_flags: [auto-generated]") {
				t.Fatalf("content lacks required honesty flag: %q", body.Content)
			}
			raw, err := base64.StdEncoding.DecodeString(req.Header.Get("X-Vrooli-Attribution"))
			if err != nil {
				t.Fatal(err)
			}
			var attribution map[string]string
			if err := json.Unmarshal(raw, &attribution); err != nil {
				t.Fatal(err)
			}
			if attribution["kind"] != "investigation" || attribution["run_id"] != "run-123" || attribution["spawn_origin"] != "investigation" {
				t.Fatalf("attribution = %#v", attribution)
			}
			return httpx.Response(http.StatusCreated, `{}`), nil
		}),
	)
	topic, err := client.PublishFriction(context.Background(), FrictionReport{InvestigationRunID: "run-123", Fingerprint: "abcdef0123456789", Category: "Tooling", Recommendation: "Fix the command", Evidence: "The command rejected valid input."})
	if err != nil {
		t.Fatal(err)
	}
	if topic != "friction-inbox/toolchain/agent-manager-finding-abcdef012345" {
		t.Fatalf("topic = %q", topic)
	}
}

func TestHTTPClient_ReadSkillSource_PinsWhenNoExperiment(t *testing.T) {
	client := newSkillTestClient(t, fakeSkillsService{read: func(req *skillsv1.ReadSkillsRequest) (*skillsv1.ReadSkillsResponse, error) {
		if req.GetVariantPolicy() != "pinned" {
			t.Errorf("expected variantPolicy pinned, got %q", req.GetVariantPolicy())
		}
		if req.GetExperimentId() != "" {
			t.Errorf("expected empty experimentId, got %q", req.GetExperimentId())
		}
		return &skillsv1.ReadSkillsResponse{
			Combined:     "content",
			CombinedHash: "sha256:abc",
			Skills:       []*skillsv1.Skill{{Id: "test-skill", Revision: 3}},
		}, nil
	}}, nil)

	snap, err := client.ReadSkillSource(context.Background(), "test-skill", "", nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.VariantID != "control" {
		t.Errorf("expected control variant, got %q", snap.VariantID)
	}
	if snap.ExperimentID != "" {
		t.Errorf("expected empty experimentId, got %q", snap.ExperimentID)
	}
}

func TestHTTPClient_ReadSkillSource_ArmsExplicitExperiment(t *testing.T) {
	client := newSkillTestClient(t, fakeSkillsService{read: func(req *skillsv1.ReadSkillsRequest) (*skillsv1.ReadSkillsResponse, error) {
		if req.GetExperimentId() != "exp-1" {
			t.Errorf("expected experimentId exp-1, got %q", req.GetExperimentId())
		}
		if req.GetVariantPolicy() != "" {
			t.Errorf("expected no variantPolicy when armed, got %q", req.GetVariantPolicy())
		}
		return &skillsv1.ReadSkillsResponse{
			Combined:          "variant content",
			CombinedHash:      "sha256:def",
			SelectedVariantId: "variant-a",
			ExperimentId:      "exp-1",
			Skills:            []*skillsv1.Skill{{Id: "test-skill", Revision: 4}},
		}, nil
	}}, nil)

	snap, err := client.ReadSkillSource(context.Background(), "test-skill", "exp-1", nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.VariantID != "variant-a" || snap.ExperimentID != "exp-1" {
		t.Errorf("expected armed variant-a/exp-1, got %q/%q", snap.VariantID, snap.ExperimentID)
	}
}

func TestHTTPClient_RecordExperimentOutcome(t *testing.T) {
	client := newSkillTestClient(t, fakeSkillsService{}, experimentTestHandler(fakeExperimentsService{record: func(outcome *experimentsv1.RecordOutcomeRequest) (*experimentsv1.ExperimentOutcome, error) {
		if outcome.GetExperimentId() != "exp-1" || outcome.GetVariantId() != "variant-a" || outcome.GetSource() != "agent-manager" {
			t.Errorf("unexpected outcome %+v", outcome)
		}
		if got := outcome.GetData().GetStructValue().GetFields()["status"].GetStringValue(); got != "complete" {
			t.Errorf("unexpected outcome data status %q", got)
		}
		return &experimentsv1.ExperimentOutcome{VariantId: outcome.GetVariantId(), Source: outcome.GetSource()}, nil
	}}))

	err := client.RecordExperimentOutcome(context.Background(), "exp-1", ExperimentOutcome{
		VariantID: "variant-a",
		Source:    "agent-manager",
		Data:      json.RawMessage(`{"status":"complete"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHTTPClient_ReadSkill_ResolverError(t *testing.T) {
	client := NewHTTPClientWithResolver(
		func(_ context.Context) (string, error) {
			return "", context.DeadlineExceeded
		},
		nil,
	)

	_, err := client.ReadSkill(context.Background(), "test-skill", nil, false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "resolve URL") {
		t.Errorf("expected resolve URL in error, got %v", err)
	}
}

func TestHTTPClientAdminOperationsAndExperimentAssignment(t *testing.T) {
	service := fakeSkillsService{
		list: func(req *skillsv1.ListSkillsRequest) (*skillsv1.ListSkillsResponse, error) {
			if req.GetTag() != "ops tag" {
				t.Fatalf("list tag=%q", req.GetTag())
			}
			return &skillsv1.ListSkillsResponse{Skills: []*skillsv1.Skill{{Id: "skill/a", Name: "A"}}}, nil
		},
		get: func(*skillsv1.GetSkillRequest) (*skillsv1.GetSkillResponse, error) {
			return &skillsv1.GetSkillResponse{Skill: &skillsv1.Skill{Id: "skill/a", Content: "old"}}, nil
		},
		update: func(req *skillsv1.UpdateSkillRequest) (*skillsv1.UpdateSkillResponse, error) {
			if req.Content == nil || req.GetContent() != "new" {
				t.Fatalf("update=%+v", req)
			}
			return &skillsv1.UpdateSkillResponse{Skill: &skillsv1.Skill{Id: "skill/a", Content: "new"}}, nil
		},
		versions: func(*skillsv1.ListSkillVersionsRequest) (*skillsv1.ListSkillVersionsResponse, error) {
			return &skillsv1.ListSkillVersionsResponse{SkillId: "skill/a", Current: 2, Versions: []*skillsv1.SkillVersion{{Version: 2, Content: "new"}}}, nil
		},
		revert: func(req *skillsv1.RevertSkillRequest) (*skillsv1.RevertSkillResponse, error) {
			if req.GetVersion() != 1 {
				t.Fatalf("revert=%+v", req)
			}
			return &skillsv1.RevertSkillResponse{SkillId: "skill/a", RevertedTo: 1}, nil
		},
	}
	client := newSkillTestClient(t, service, experimentTestHandler(fakeExperimentsService{assign: func(req *experimentsv1.AssignExperimentRequest) (*experimentsv1.ExperimentAssignment, error) {
		if req.GetExperimentId() != "exp/1" || req.GetExecutionId() != "execution" || req.GetNodeId() != "node" || req.GetAttemptKey() != "attempt" || req.GetIdempotencyKey() != "key" {
			t.Fatalf("assignment request=%+v", req)
		}
		return &experimentsv1.ExperimentAssignment{ExperimentId: "exp/1", SkillId: "skill/a", VariantId: "variant", Content: "prompt", ContentHash: "sha256:1"}, nil
	}}))
	ctx := context.Background()
	if skills, err := client.ListSkills(ctx, "ops tag"); err != nil || len(skills) != 1 {
		t.Fatalf("skills=%+v err=%v", skills, err)
	}
	if skill, err := client.GetSkill(ctx, "skill/a"); err != nil || skill.Content != "old" {
		t.Fatalf("skill=%+v err=%v", skill, err)
	}
	content := "new"
	if skill, err := client.UpdateSkill(ctx, "skill/a", PromptSkillUpdate{Content: &content}); err != nil || skill.Content != "new" {
		t.Fatalf("updated=%+v err=%v", skill, err)
	}
	if versions, err := client.GetSkillVersions(ctx, "skill/a"); err != nil || versions.Current != 2 {
		t.Fatalf("versions=%+v err=%v", versions, err)
	}
	if err := client.RevertSkillVersion(ctx, "skill/a", 1); err != nil {
		t.Fatal(err)
	}
	assignment := AssignmentRequest{ExperimentID: "exp/1", SkillID: "skill/a", ExecutionID: "execution", NodeID: "node", AttemptKey: "attempt", IdempotencyKey: "key"}
	if snap, err := client.AssignExperimentPrompt(ctx, assignment); err != nil || snap.VariantID != "variant" || snap.Content != "prompt" {
		t.Fatalf("assignment=%+v err=%v", snap, err)
	}
	if _, err := client.AssignExperimentPrompt(ctx, AssignmentRequest{}); err == nil {
		t.Fatal("incomplete assignment accepted")
	}
}
