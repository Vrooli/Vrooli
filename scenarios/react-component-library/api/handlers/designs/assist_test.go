package designs_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	aiinference "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	"google.golang.org/protobuf/encoding/protojson"

	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"

	designsH "react-component-library/handlers/designs"
	localdb "react-component-library/internal/database"
	"react-component-library/internal/designinference"
	designs "react-component-library/internal/designs"
)

type fakeAssistant struct {
	calls     int
	key       string
	request   designinference.Request
	operation designinference.Operation
	err       error
}

func (f *fakeAssistant) Run(_ context.Context, key string, request designinference.Request) (designinference.Operation, error) {
	f.calls++
	f.key = key
	f.request = request
	return f.operation, f.err
}

func completed(value string) designinference.Operation {
	raw, _ := protojson.Marshal(&aiinference.RunResponse{
		ValueJson: value,
		Validated: true,
		Provider:  "fake",
		Model:     "test-model",
		Usage:     &aiinference.Usage{InputTokens: 120, OutputTokens: 40},
	})
	return designinference.Operation{ID: "inference_test", State: "completed", ResponseJSON: string(raw)}
}

func newAssistRouter(t *testing.T, assistant designsH.Assistant) *mux.Router {
	t.Helper()
	sqlDB := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), sqlDB,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(designs.Schema),
		apidb.SchemaProviderFunc(designinference.Schema),
	))
	require.NoError(t, designs.Migrate(context.Background(), sqlDB))
	module := designsH.ModuleWithAssistant(designs.NewSQLiteRepository(sqlDB), nil, assistant)
	router := mux.NewRouter()
	module.Mount(router)
	return router
}

const assistDesignBody = `{
	"name":"Assist sample",
	"theme":{"accent":"#ffb35c"},
	"frames":[{
		"id":"desktop","name":"Desktop","device":"desktop",
		"spec":{"id":"page","type":"page","props":{},"children":[
			{"id":"hero","type":"stack","props":{"gap":18},"children":[
				{"id":"title","type":"text","props":{"text":"Old headline"}},
				{"id":"cta","type":"button","props":{"text":"Open"}}
			]}
		]}
	}]
}`

func seedAssistDesign(t *testing.T, router *mux.Router) designs.Document {
	t.Helper()
	rec := do(t, router, http.MethodPost, "/api/designs", assistDesignBody)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	return decodeDesign(t, rec)
}

func assistBody(revision int, scope string) string {
	return fmt.Sprintf(`{"prompt":"Sharpen the headline.","scope":%s,"revision":%d,"idempotencyKey":"assist-key-0001"}`, scope, revision)
}

func TestAssistReturnsAnInertValidatedProposal(t *testing.T) {
	assistant := &fakeAssistant{operation: completed(`{"summary":"Sharpened the headline.","operations":[{"kind":"setText","nodeId":"title","text":"New headline"}]}`)}
	router := newAssistRouter(t, assistant)
	design := seedAssistDesign(t, router)

	rec := do(t, router, http.MethodPost, "/api/designs/"+design.ID+"/assist", assistBody(design.Revision, `{"kind":"layer","frameId":"desktop","nodeId":"title"}`))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var payload struct {
		Operation struct {
			ID           string `json:"id"`
			State        string `json:"state"`
			Provider     string `json:"provider"`
			Model        string `json:"model"`
			InputTokens  int64  `json:"inputTokens"`
			OutputTokens int64  `json:"outputTokens"`
		} `json:"operation"`
		Revision int `json:"revision"`
		Proposal struct {
			Summary    string `json:"summary"`
			Operations []struct {
				Kind   string `json:"kind"`
				NodeID string `json:"nodeId"`
				Text   string `json:"text"`
			} `json:"operations"`
		} `json:"proposal"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload), rec.Body.String())
	require.Equal(t, "inference_test", payload.Operation.ID)
	require.Equal(t, "completed", payload.Operation.State)
	require.Equal(t, "fake", payload.Operation.Provider)
	require.Equal(t, int64(120), payload.Operation.InputTokens)
	require.Equal(t, design.Revision, payload.Revision)
	require.Equal(t, "Sharpened the headline.", payload.Proposal.Summary)
	require.Len(t, payload.Proposal.Operations, 1)
	require.Equal(t, "New headline", payload.Proposal.Operations[0].Text)

	require.Equal(t, 1, assistant.calls)
	require.Equal(t, "assist-key-0001", assistant.key)
	require.Equal(t, "extract.structured", assistant.request.Role)
	require.EqualValues(t, 4096, assistant.request.MaxOutputTokens)
	require.Contains(t, assistant.request.Instruction, "Sharpen the headline.")
	var contextPayload map[string]any
	require.NoError(t, json.Unmarshal([]byte(assistant.request.Source), &contextPayload), "the model source is the scoped context JSON")
	require.Contains(t, contextPayload, "selection")

	current := do(t, router, http.MethodGet, "/api/designs/"+design.ID, "")
	require.Equal(t, design.Revision, decodeDesign(t, current).Revision, "assist never writes the design")
}

func TestAssistRejectsAStaleRevision(t *testing.T) {
	assistant := &fakeAssistant{operation: completed(`{"summary":"x","operations":[{"kind":"setText","nodeId":"title","text":"New"}]}`)}
	router := newAssistRouter(t, assistant)
	design := seedAssistDesign(t, router)

	rec := do(t, router, http.MethodPost, "/api/designs/"+design.ID+"/assist", assistBody(design.Revision+1, `{"kind":"design"}`))
	require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	require.Equal(t, 0, assistant.calls)
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
		Design designs.Document `json:"design"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, "conflict", payload.Error.Code)
	require.Equal(t, design.Revision, payload.Design.Revision)
}

func TestAssistRejectsAProposalTheDesignCannotApply(t *testing.T) {
	assistant := &fakeAssistant{operation: completed(`{"summary":"x","operations":[{"kind":"setText","nodeId":"missing","text":"New"}]}`)}
	router := newAssistRouter(t, assistant)
	design := seedAssistDesign(t, router)

	rec := do(t, router, http.MethodPost, "/api/designs/"+design.ID+"/assist", assistBody(design.Revision, `{"kind":"design"}`))
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), "invalid_proposal")
}

func TestAssistMapsGatewayFailures(t *testing.T) {
	t.Run("unreachable", func(t *testing.T) {
		router := newAssistRouter(t, &fakeAssistant{err: context.DeadlineExceeded})
		design := seedAssistDesign(t, router)
		rec := do(t, router, http.MethodPost, "/api/designs/"+design.ID+"/assist", assistBody(design.Revision, `{"kind":"design"}`))
		require.Equal(t, http.StatusServiceUnavailable, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "ai_unavailable")
	})

	t.Run("no answer", func(t *testing.T) {
		router := newAssistRouter(t, &fakeAssistant{operation: designinference.Operation{ID: "x", State: "dispatch_unknown"}})
		design := seedAssistDesign(t, router)
		rec := do(t, router, http.MethodPost, "/api/designs/"+design.ID+"/assist", assistBody(design.Revision, `{"kind":"design"}`))
		require.Equal(t, http.StatusServiceUnavailable, rec.Code, rec.Body.String())
	})

	t.Run("gateway failed", func(t *testing.T) {
		router := newAssistRouter(t, &fakeAssistant{operation: designinference.Operation{ID: "x", State: "failed", Detail: "model refused"}})
		design := seedAssistDesign(t, router)
		rec := do(t, router, http.MethodPost, "/api/designs/"+design.ID+"/assist", assistBody(design.Revision, `{"kind":"design"}`))
		require.Equal(t, http.StatusBadGateway, rec.Code, rec.Body.String())
		require.Contains(t, rec.Body.String(), "model refused")
	})
}

func TestAssistValidatesTheEnvelope(t *testing.T) {
	router := newAssistRouter(t, &fakeAssistant{})
	design := seedAssistDesign(t, router)

	for name, body := range map[string]string{
		"missing revision": `{"prompt":"x","scope":{"kind":"design"},"idempotencyKey":"assist-key-0001"}`,
		"short key":        `{"prompt":"x","scope":{"kind":"design"},"revision":1,"idempotencyKey":"short"}`,
		"empty prompt":     `{"prompt":"  ","scope":{"kind":"design"},"revision":1,"idempotencyKey":"assist-key-0001"}`,
		"bad scope":        `{"prompt":"x","scope":{"kind":"everything"},"revision":1,"idempotencyKey":"assist-key-0001"}`,
		"unknown field":    `{"prompt":"x","scope":{"kind":"design"},"revision":1,"idempotencyKey":"assist-key-0001","model":"other"}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec := do(t, router, http.MethodPost, "/api/designs/"+design.ID+"/assist", body)
			require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
		})
	}

	rec := do(t, router, http.MethodPost, "/api/designs/design-missing/assist", assistBody(1, `{"kind":"design"}`))
	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
}

func TestAssistWithoutARunnerIsHonest(t *testing.T) {
	sqlDB := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), sqlDB,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(designs.Schema),
	))
	require.NoError(t, designs.Migrate(context.Background(), sqlDB))
	module := designsH.ModuleFromRepository(designs.NewSQLiteRepository(sqlDB), nil)
	router := mux.NewRouter()
	module.Mount(router)
	design := seedAssistDesign(t, router)

	rec := do(t, router, http.MethodPost, "/api/designs/"+design.ID+"/assist", assistBody(design.Revision, `{"kind":"design"}`))
	require.Equal(t, http.StatusServiceUnavailable, rec.Code, rec.Body.String())
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, "ai_unavailable", payload.Error.Code)
}
