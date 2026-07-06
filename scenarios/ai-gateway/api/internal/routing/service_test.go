package routing_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"ai-gateway/internal/providers"
	"ai-gateway/internal/providers/mocks"
	"ai-gateway/internal/routing"
	testdb "ai-gateway/internal/testutil/db"
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

func roleRunner() *mocks.FakeRunner {
	return &mocks.FakeRunner{
		Results: map[string]providers.Result{
			"resource-ollama policy roles --json": {
				Stdout: `{"roles":[{"schema_version":"1","role":"chat.default","required_capabilities":["generate","chat"]}]}`,
			},
			"resource-openrouter policy roles --json": {
				Stdout: `{"roles":[{"schema_version":"1","role":"chat.default","required_capabilities":["generate","chat"]}]}`,
			},
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
