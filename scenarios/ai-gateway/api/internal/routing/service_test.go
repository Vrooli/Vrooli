package routing_test

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"ai-gateway/internal/providers"
	"ai-gateway/internal/providers/mocks"
	"ai-gateway/internal/routing"

	testdb "github.com/vrooli/api-core/databasetest"
	"google.golang.org/protobuf/proto"
)

func TestPreviewSelectsLocalFirstAndExplainsFallback(t *testing.T) { // [REQ:AIGW-ROUTE-PREVIEW] [REQ:AIGW-POLICY-CONSTRAINTS]
	runner := roleRunner()
	svc := routing.NewService(testAdapters(runner), nil)

	resp, err := svc.Preview(context.Background(), baseRequest(sharedv1.Profile_PROFILE_LOCAL_FIRST))
	require.NoError(t, err)
	require.True(t, resp.GetValid())
	require.Equal(t, "ollama", resp.GetSelectedProvider())
	require.True(t, resp.GetFallbackAllowed())
	require.Len(t, resp.GetCandidates(), 2)
	require.True(t, resp.GetCandidates()[0].GetSelected())
	require.True(t, resp.GetCandidates()[1].GetFallbackEligible())
	require.Contains(t, strings.Join(resp.GetPolicyReasons(), " "), "local-first")
}

func TestPreviewBlocksRemoteFallbackForSecretPrivacy(t *testing.T) { // [REQ:AIGW-POLICY-CONSTRAINTS]
	runner := roleRunner()
	svc := routing.NewService(testAdapters(runner), nil)

	req := baseRequest(sharedv1.Profile_PROFILE_PRIVACY_SENSITIVE)
	req.PrivacyClass = sharedv1.PrivacyClass_PRIVACY_CLASS_SECRET
	resp, err := svc.Preview(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.GetValid())
	require.Equal(t, "ollama", resp.GetSelectedProvider())
	require.False(t, resp.GetFallbackAllowed())
	require.Len(t, resp.GetCandidates(), 2)
	require.Contains(t, strings.Join(resp.GetCandidates()[1].GetReasons(), " "), "secret requests")
}

func TestExecutePersistsRedactedEvidenceAndKeepsInputOutOfCommandString(t *testing.T) { // [REQ:AIGW-ROUTE-EVIDENCE] [REQ:AIGW-PROVIDER-CLI-ONLY]
	runner := roleRunner()
	runner.Results["resource-ollama gateway generate --role chat.default --json --prompt-stdin --max-tokens 64"] = providers.Result{Stdout: `{"response":"ok","eval_count":1}`}
	db := newSchemaDB(t)
	svc := routing.NewService(testAdapters(runner), routing.NewSQLRepository(db))

	resp, err := svc.Execute(context.Background(), baseRequest(sharedv1.Profile_PROFILE_LOCAL_ONLY), "sensitive prompt")
	require.NoError(t, err)
	require.True(t, resp.GetValid())
	require.Equal(t, "succeeded", resp.GetEvidence().GetStatus())
	require.Equal(t, "ollama", resp.GetEvidence().GetSelectedProvider())
	require.Equal(t, "qwen3.5:9b", resp.GetEvidence().GetSelectedModel())
	require.True(t, resp.GetEvidence().GetPromptRedacted())
	require.True(t, resp.GetEvidence().GetResponseRedacted())
	require.NotContains(t, strings.Join(resp.GetEvidence().GetPolicyReasons(), " "), "sensitive prompt")
	require.NotContains(t, resp.GetOutputText(), "sensitive prompt")

	require.NotEmpty(t, runner.Commands)
	require.NotContains(t, runner.Commands[len(runner.Commands)-1].String(), "sensitive prompt")

	events, err := svc.ListEvidence(context.Background(), routing.EvidenceFilter{Limit: 10})
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, resp.GetEvidence().GetEventId(), events[0].GetEventId())
	require.Equal(t, "succeeded", events[0].GetStatus())
}

func TestExecuteFailsClosedWhenEvidenceCannotBePersisted(t *testing.T) { // [REQ:AIGW-ROUTE-EVIDENCE]
	runner := roleRunner()
	runner.Results["resource-ollama gateway generate --role chat.default --json --prompt-stdin --max-tokens 64"] = providers.Result{Stdout: `{"response":"ok"}`}
	svc := routing.NewService(testAdapters(runner), routing.NewSQLRepository(nil))

	resp, err := svc.Execute(context.Background(), baseRequest(sharedv1.Profile_PROFILE_LOCAL_ONLY), "hello")
	require.Nil(t, resp)
	require.Error(t, err)
	require.Contains(t, err.Error(), "persist successful route evidence")
}

func TestMultimodalRouteUsesDeclaredModalityAndRedactsAttachmentEvidence(t *testing.T) { // [REQ:AIGW-MULTIMODAL-CONTRACT]
	runner := multimodalRoleRunner()
	runner.Results["resource-ollama gateway generate --role chat.default --json --input-json-stdin"] = providers.Result{Stdout: `{"response":"ok","eval_count":1}`}
	db := newSchemaDB(t)
	svc := routing.NewService(testAdapters(runner), routing.NewSQLRepository(db))
	image := []byte{137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 13, 'I', 'H', 'D', 'R', 0, 0, 0, 1, 0, 0, 0, 1}
	req := baseRequest(sharedv1.Profile_PROFILE_LOCAL_ONLY)
	req.Attachments = []*sharedv1.Attachment{{
		Modality:  sharedv1.Modality_MODALITY_IMAGE,
		MediaType: "image/png",
		Width:     1,
		Height:    1,
		Bytes:     uint64(len(image)),
		Payload:   &sharedv1.Attachment_InlineBytes{InlineBytes: image},
	}}
	resp, err := svc.Execute(context.Background(), req, "describe")
	require.NoError(t, err)
	require.True(t, resp.GetValid())
	require.Equal(t, int32(1), resp.GetEvidence().GetImageCount())
	require.Equal(t, int64(len(image)), resp.GetEvidence().GetAttachmentBytes())
	require.True(t, resp.GetEvidence().GetAttachmentsRedacted())
	require.NotContains(t, resp.GetEvidence().String(), base64.StdEncoding.EncodeToString(image))
	require.Len(t, resp.GetEvidence().GetAttachmentDimensions(), 1)
	require.Contains(t, runner.Commands[len(runner.Commands)-1].Stdin, base64.StdEncoding.EncodeToString(image))

	var stored string
	require.NoError(t, db.QueryRow("SELECT attachment_dimensions_json FROM route_events LIMIT 1").Scan(&stored))
	require.NotContains(t, stored, base64.StdEncoding.EncodeToString(image))
	require.NotContains(t, routing.Schema(), "data_b64")
	rows, err := db.Query("SELECT * FROM route_events")
	require.NoError(t, err)
	defer rows.Close()
	columns, err := rows.Columns()
	require.NoError(t, err)
	require.True(t, rows.Next())
	values := make([]any, len(columns))
	destinations := make([]any, len(values))
	for index := range values {
		destinations[index] = &values[index]
	}
	require.NoError(t, rows.Scan(destinations...))
	for _, value := range values {
		require.NotContains(t, fmt.Sprint(value), base64.StdEncoding.EncodeToString(image))
	}
}

func TestMultimodalRouteRejectsTextOnlyRoleBeforeProviderCall(t *testing.T) { // [REQ:AIGW-MULTIMODAL-CONTRACT]
	runner := roleRunner()
	svc := routing.NewService(testAdapters(runner), nil)
	req := baseRequest(sharedv1.Profile_PROFILE_LOCAL_ONLY)
	req.Attachments = []*sharedv1.Attachment{{Modality: sharedv1.Modality_MODALITY_IMAGE, MediaType: "image/png", Width: 1, Height: 1, Payload: &sharedv1.Attachment_InlineBytes{InlineBytes: []byte{137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 13, 'I', 'H', 'D', 'R', 0, 0, 0, 1, 0, 0, 0, 1}}}}
	resp, err := svc.Preview(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.GetValid())
	require.Len(t, resp.GetCandidates(), 2)
	for _, candidate := range resp.GetCandidates() {
		require.Equal(t, "capability_mismatch", candidate.GetRejectionReason())
	}
}

func roleRunner() *mocks.FakeRunner {
	return &mocks.FakeRunner{
		Results: map[string]providers.Result{
			"resource-ollama policy roles --json": {
				Stdout: `{"roles":[{"schema_version":"1","role":"chat.default","required_capabilities":["generate","chat"]}]}`,
			},
			"resource-openrouter policy roles --json": {
				Stdout: `{"roles":[{"schema_version":"1","role":"chat.default","required_capabilities":["generate","chat"]}]}`,
			},
			"resource-ollama policy resolve --role chat.default --json": {
				Stdout: `{"role":"chat.default","model":"qwen3.5:9b"}`,
			},
			"resource-openrouter policy resolve --role chat.default --json": {
				Stdout: `{"role":"chat.default","model":"google/gemini-3.1-flash-lite-preview"}`,
			},
		},
		Errors: map[string]error{},
	}
}

func multimodalRoleRunner() *mocks.FakeRunner {
	return &mocks.FakeRunner{
		Results: map[string]providers.Result{
			"resource-ollama policy roles --json":                           {Stdout: `{"roles":[{"schema_version":"1","role":"chat.default","required_capabilities":["generate","chat"],"modalities":{"input":["text","image"],"output":["text"]}}]}`},
			"resource-openrouter policy roles --json":                       {Stdout: `{"roles":[{"schema_version":"1","role":"chat.default","required_capabilities":["generate","chat"],"modalities":{"input":["text","image"],"output":["text"]}}]}`},
			"resource-ollama policy resolve --role chat.default --json":     {Stdout: `{"role":"chat.default","model":"qwen3.5:9b"}`},
			"resource-openrouter policy resolve --role chat.default --json": {Stdout: `{"role":"chat.default","model":"google/gemini-3.1-flash-lite-preview"}`},
		},
		Errors: map[string]error{},
	}
}

func testAdapters(runner providers.CommandRunner) []providers.Adapter {
	return []providers.Adapter{
		{Provider: providers.ProviderOllama, CommandName: "resource-ollama", Locality: "local", Runner: runner},
		{Provider: providers.ProviderOpenRouter, CommandName: "resource-openrouter", Locality: "remote", Runner: runner},
	}
}

func baseRequest(profile sharedv1.Profile) *sharedv1.GatewayRequest {
	return &sharedv1.GatewayRequest{
		Kind:            sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION,
		Role:            "chat.default",
		Profile:         profile,
		PrivacyClass:    sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL,
		Operation:       "summarize",
		Scenario:        "fixture-scenario",
		RequestId:       "req-1",
		MaxOutputTokens: 64,
	}
}

func newSchemaDB(t *testing.T) *sql.DB {
	t.Helper()
	db := testdb.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(routing.Schema)))
	return db
}

func TestExecuteSendsCallerTemperatureAndRecordsItInEvidence(t *testing.T) { // [REQ:AIGW-ROUTE-EVIDENCE]
	runner := roleRunner()
	runner.Results["resource-ollama policy resolve --role chat.default --json"] = providers.Result{
		Stdout: `{"role":"chat.default","model":"qwen3.5:9b","sampling_support":{"temperature":"honored"}}`,
	}
	runner.Results["resource-ollama gateway generate --role chat.default --json --prompt-stdin --max-tokens 64 --temperature 1.1"] = providers.Result{Stdout: `{"response":"ok","eval_count":1}`}
	db := newSchemaDB(t)
	svc := routing.NewService(testAdapters(runner), routing.NewSQLRepository(db))

	req := baseRequest(sharedv1.Profile_PROFILE_LOCAL_ONLY)
	req.Sampling = &sharedv1.SamplingControls{Temperature: proto.Float64(1.1)}
	resp, err := svc.Execute(context.Background(), req, "hello")
	require.NoError(t, err)
	require.Equal(t, "succeeded", resp.GetEvidence().GetStatus())
	require.Equal(t, 1.1, resp.GetEvidence().GetSamplingTemperature())
	require.Equal(t, sharedv1.SamplingSupport_SAMPLING_SUPPORT_HONORED, resp.GetEvidence().GetSamplingTemperatureSupport())
	require.Equal(t, 1.1, resp.GetApplied().GetTemperatureSent())
	require.Equal(t, sharedv1.SamplingSupport_SAMPLING_SUPPORT_HONORED, resp.GetApplied().GetTemperatureSupport())
	require.Equal(t, sharedv1.OutputCapSource_OUTPUT_CAP_SOURCE_REQUEST, resp.GetApplied().GetMaxOutputTokensSource())

	// The durable record must survive the round trip, not only the response.
	events, err := svc.ListEvidence(context.Background(), routing.EvidenceFilter{Limit: 10})
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, 1.1, events[0].GetSamplingTemperature())
	require.Equal(t, sharedv1.SamplingSupport_SAMPLING_SUPPORT_HONORED, events[0].GetSamplingTemperatureSupport())
}

// "No temperature was sent" must stay distinguishable from "0 was sent", or an
// omitted control reads as a deterministic one in every later query.
func TestExecuteRecordsAbsentTemperatureAsNullNotZero(t *testing.T) { // [REQ:AIGW-ROUTE-EVIDENCE]
	runner := roleRunner()
	runner.Results["resource-ollama gateway generate --role chat.default --json --prompt-stdin --max-tokens 64"] = providers.Result{Stdout: `{"response":"ok","eval_count":1}`}
	db := newSchemaDB(t)
	svc := routing.NewService(testAdapters(runner), routing.NewSQLRepository(db))

	resp, err := svc.Execute(context.Background(), baseRequest(sharedv1.Profile_PROFILE_LOCAL_ONLY), "hello")
	require.NoError(t, err)
	require.Equal(t, "succeeded", resp.GetEvidence().GetStatus())
	require.Nil(t, resp.GetEvidence().SamplingTemperature)
	require.Nil(t, resp.GetApplied().TemperatureSent)

	var stored sql.NullFloat64
	require.NoError(t, db.QueryRow("SELECT sampling_temperature FROM route_events LIMIT 1").Scan(&stored))
	require.False(t, stored.Valid, "an omitted control must persist as NULL, not 0")
}

// A candidate that cannot honor the caller's control is skipped, the walk
// continues, and the skip never trips a circuit breaker: the provider failed
// nothing.
func TestExecuteSkipsCandidateThatCannotHonorRequestedTemperature(t *testing.T) { // [REQ:AIGW-ROUTE-EVIDENCE]
	runner := roleRunner()
	runner.Results["resource-ollama policy resolve --role chat.default --json"] = providers.Result{
		Stdout: `{"role":"chat.default","model":"qwen3.5:9b","sampling_support":{"temperature":"ignored"}}`,
	}
	runner.Results["resource-openrouter policy resolve --role chat.default --json"] = providers.Result{
		Stdout: `{"role":"chat.default","model":"vendor/model","sampling_support":{"temperature":"honored"}}`,
	}
	runner.Results["resource-openrouter generate --role chat.default --json --max-tokens 64 --temperature 1.1"] = providers.Result{
		Stdout: `{"choices":[{"message":{"content":"ok"}}]}`,
	}
	db := newSchemaDB(t)
	svc := routing.NewService(testAdapters(runner), routing.NewSQLRepository(db))

	req := baseRequest(sharedv1.Profile_PROFILE_LOCAL_FIRST)
	req.Sampling = &sharedv1.SamplingControls{Temperature: proto.Float64(1.1)}
	resp, err := svc.Execute(context.Background(), req, "hello")
	require.NoError(t, err)
	require.Equal(t, "succeeded", resp.GetEvidence().GetStatus())
	require.Equal(t, "openrouter", resp.GetEvidence().GetSelectedProvider())
	require.Contains(t, strings.Join(resp.GetEvidence().GetFailureReasons(), " "), "temperature not honored")

	health, err := svc.ListProviderHealth(context.Background())
	require.NoError(t, err)
	for _, row := range health {
		require.NotEqual(t, "ollama", row.GetProvider(),
			"a sampling mismatch must not be recorded against provider health")
	}
}

func TestExecuteFailsWhenNoCandidateHonorsRequestedTemperature(t *testing.T) { // [REQ:AIGW-ROUTE-EVIDENCE]
	runner := roleRunner()
	runner.Results["resource-ollama policy resolve --role chat.default --json"] = providers.Result{
		Stdout: `{"role":"chat.default","model":"qwen3.5:9b","sampling_support":{"temperature":"rejected"}}`,
	}
	db := newSchemaDB(t)
	svc := routing.NewService(testAdapters(runner), routing.NewSQLRepository(db))

	req := baseRequest(sharedv1.Profile_PROFILE_LOCAL_ONLY)
	req.Sampling = &sharedv1.SamplingControls{Temperature: proto.Float64(1.1)}
	resp, err := svc.Execute(context.Background(), req, "hello")
	require.NoError(t, err)
	require.Equal(t, "failed", resp.GetEvidence().GetStatus())
	require.Equal(t, string(routing.FailureUnsupportedSampling), resp.GetEvidence().GetFailureClass())
	require.Empty(t, resp.GetOutputText())
}
