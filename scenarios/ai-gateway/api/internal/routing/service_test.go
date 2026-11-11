package routing_test

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/ai-provider-go/policy"
	"github.com/vrooli/api-core/database"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"ai-gateway/internal/providers"
	"ai-gateway/internal/providers/mocks"
	"ai-gateway/internal/routing"

	testdb "github.com/vrooli/api-core/databasetest"
	"google.golang.org/protobuf/proto"
)

func newService(adapters []providers.Adapter, repo routing.Repository, opts ...routing.Option) *routing.Service {
	ctx := policy.RightsContext{ID: "fixture-context", Version: "v1", EntityScope: "fixture", RevenueStatus: "pre_revenue", AsOf: "2026-11-11", Source: "test_fixture"}
	opts = append(opts, routing.WithRightsContextSource(func(context.Context) (policy.RightsContext, error) { return ctx, nil }))
	return routing.NewService(adapters, repo, opts...)
}

type inventoryMutatingRunner struct {
	inner       *mocks.FakeRunner
	mutateAt    int
	calls       int
	replacement providers.Result
}

func (r *inventoryMutatingRunner) Run(ctx context.Context, command providers.Command) (providers.Result, error) {
	if strings.Contains(command.String(), " models inventory ") {
		r.calls++
		if r.calls == r.mutateAt {
			r.inner.Results[command.String()] = r.replacement
		}
	}
	return r.inner.Run(ctx, command)
}

func TestPreviewSelectsLocalFirstAndExplainsFallback(t *testing.T) { // [REQ:AIGW-ROUTE-PREVIEW] [REQ:AIGW-POLICY-CONSTRAINTS]
	runner := roleRunner()
	svc := newService(testAdapters(runner), nil)

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

func TestConditionalLocalRightsReevaluatedAfterPreviewBeforeExecution(t *testing.T) {
	runner := roleRunner()
	condition := `"classification":"conditional","conditions":[{"id":"revenue-v1","kind":"annual_revenue","comparator":"lt","threshold":10000000,"unit":"USD","entity_scope":"fixture","interpretation":"conservative fixture condition"}],"evidence":`
	for key, result := range runner.Results {
		if strings.Contains(key, "policy roles") || strings.Contains(key, "policy resolve") {
			result.Stdout = strings.Replace(result.Stdout, `"classification":"unrestricted","evidence":`, condition, 1)
			runner.Results[key] = result
		}
	}
	ctx := policy.RightsContext{ID: "fixture-context", Version: "v1", EntityScope: "fixture", RevenueStatus: "pre_revenue", AsOf: "2026-11-11", Source: "test_fixture"}
	source := func(context.Context) (policy.RightsContext, error) { return ctx, nil }
	svc := routing.NewService(testAdapters(runner), routing.NewSQLRepository(newSchemaDB(t)), routing.WithRightsContextSource(source))
	req := baseRequest(sharedv1.Profile_PROFILE_LOCAL_ONLY)
	preview, err := svc.Preview(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "ollama", preview.GetSelectedProvider())

	ctx.Version = "v2"
	ctx.RevenueStatus = "at_or_above_threshold"
	execution, err := svc.Execute(context.Background(), req, "hello")
	require.NoError(t, err)
	require.Equal(t, "blocked", execution.GetEvidence().GetStatus())
	for _, command := range runner.Commands {
		require.NotContains(t, command.String(), "gateway generate", "a changed current context must block before local inference")
	}
}

func TestChangedOllamaRuntimeArtifactAfterPlanningBlocksExecution(t *testing.T) {
	base := roleRunner()
	runner := &inventoryMutatingRunner{inner: base, mutateAt: 3, replacement: providers.Result{Stdout: `{"source":"ollama /api/tags","models":[{"name":"qwen3.5:9b","digest":"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},{"name":"nomic-embed-text:latest","digest":"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"}]}`}}
	svc := newService(testAdapters(runner), routing.NewSQLRepository(newSchemaDB(t)))
	resp, err := svc.Execute(context.Background(), baseRequest(sharedv1.Profile_PROFILE_LOCAL_ONLY), "hello")
	require.NoError(t, err)
	require.NotEqual(t, "succeeded", resp.GetEvidence().GetStatus())
	for _, command := range base.Commands {
		require.NotContains(t, command.String(), "gateway generate", "changed runtime digest must block before provider invocation")
	}
}

func TestMissingCatalogRightsBlocksLocalCandidateAtPlanning(t *testing.T) {
	runner := &mocks.FakeRunner{Results: map[string]providers.Result{
		"resource-ollama policy roles --json": {Stdout: `{"roles":[{"role":"chat.default","model":"qwen3.5:9b","required_capabilities":["generate","chat"]}]}`},
	}}
	ctx := policy.RightsContext{ID: "fixture-context", Version: "v1", EntityScope: "fixture", RevenueStatus: "pre_revenue", AsOf: "2026-11-11", Source: "test_fixture"}
	svc := routing.NewService(testAdapters(runner)[:1], nil, routing.WithRightsContextSource(func(context.Context) (policy.RightsContext, error) { return ctx, nil }))
	preview, err := svc.Preview(context.Background(), baseRequest(sharedv1.Profile_PROFILE_LOCAL_ONLY))
	require.NoError(t, err)
	require.Empty(t, preview.GetSelectedProvider())
	require.Equal(t, "local_model_rights_unknown", preview.GetCandidates()[0].GetRejectionReason())
}

func TestOtherLocalModelOwnersWithoutRightsAreBlocked(t *testing.T) {
	tests := []struct {
		name, provider, command, role, kind, capabilities string
	}{
		{"ollaya local decision model", providers.ProviderOllaya, "resource-ollaya", "decide.local", "text_generation", `"generate","chat"`},
		{"pii detector model", providers.ProviderPiiDetector, "resource-pii-detector", "pii.small", "detect_pii", `"detect_pii"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resolverOutput := `{"roles":[{"role":"` + tc.role + `","model":"fixture-model","locality":"local","required_capabilities":[` + tc.capabilities + `]}]}`
			runner := &mocks.FakeRunner{Results: map[string]providers.Result{
				tc.command + " policy roles --json": {Stdout: resolverOutput},
			}}
			adapter := providers.Adapter{Provider: tc.provider, CommandName: tc.command, Locality: "local", Runner: runner}
			ctx := policy.RightsContext{ID: "fixture-context", Version: "v1", EntityScope: "fixture", RevenueStatus: "pre_revenue", AsOf: "2026-11-11", Source: "test_fixture"}
			svc := routing.NewService([]providers.Adapter{adapter}, nil, routing.WithRightsContextSource(func(context.Context) (policy.RightsContext, error) { return ctx, nil }))
			req := baseRequest(sharedv1.Profile_PROFILE_LOCAL_ONLY)
			req.Role = tc.role
			if tc.kind == "detect_pii" {
				req.Kind = sharedv1.RequestKind_REQUEST_KIND_DETECT_PII
			}
			preview, err := svc.Preview(context.Background(), req)
			require.NoError(t, err)
			require.Empty(t, preview.GetSelectedProvider())
			require.Equal(t, "local_model_rights_unknown", preview.GetCandidates()[0].GetRejectionReason())
		})
	}
}

func TestPIIModelIdentityChangeAfterPlanningBlocksExecution(t *testing.T) {
	rights := `{"artifact_id":"fixture/pii-model","revision":"fixture-rev","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","license_id":"fixture-license","classification":"unrestricted","evidence":{"url":"https://example.invalid/license","revision":"fixture-license-rev","sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","observed_at":"2026-11-11"}}`
	roleJSON := `{"roles":[{"role":"pii.small","model":"fixture-pii-model","rights":` + rights + `,"locality":"local","required_capabilities":["detect_pii"],"modalities":{"input":["text"]}}]}`
	runner := &mocks.FakeRunner{Results: map[string]providers.Result{
		"resource-pii-detector policy roles --json":                    {Stdout: roleJSON},
		"resource-pii-detector policy resolve --role pii.small --json": {Stdout: `{"role":"pii.small","model":"fixture-pii-model","rights":` + rights + `,"locality":"local","capabilities":["detect_pii"]}`},
	}}
	ctx := policy.RightsContext{ID: "fixture-context", Version: "v1", EntityScope: "fixture", RevenueStatus: "pre_revenue", AsOf: "2026-11-11", Source: "test_fixture"}
	reads := 0
	source := func(context.Context) (policy.RightsContext, error) {
		reads++
		if reads == 2 {
			runner.Results["resource-pii-detector policy resolve --role pii.small --json"] = providers.Result{Stdout: `{"role":"pii.small","model":"changed-pii-model","rights":` + rights + `,"locality":"local","capabilities":["detect_pii"]}`}
		}
		return ctx, nil
	}
	adapter := providers.Adapter{Provider: providers.ProviderPiiDetector, CommandName: "resource-pii-detector", Locality: "local", Runner: runner}
	svc := routing.NewService([]providers.Adapter{adapter}, routing.NewSQLRepository(newSchemaDB(t)), routing.WithRightsContextSource(source))
	resp, err := svc.Execute(context.Background(), piiRequestForGatewayTest(), "mail me")
	require.NoError(t, err)
	require.Equal(t, "blocked", resp.GetEvidence().GetStatus())
	for _, command := range runner.Commands {
		require.NotContains(t, command.String(), "gateway detect", "changed model identity must be rejected before inference")
	}
}

func piiRequestForGatewayTest() *sharedv1.GatewayRequest {
	request := baseRequest(sharedv1.Profile_PROFILE_LOCAL_ONLY)
	request.Kind = sharedv1.RequestKind_REQUEST_KIND_DETECT_PII
	request.Role = "pii.small"
	return request
}

func TestPreviewBlocksRemoteFallbackForSecretPrivacy(t *testing.T) { // [REQ:AIGW-POLICY-CONSTRAINTS]
	runner := roleRunner()
	svc := newService(testAdapters(runner), nil)

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
	runner.Results["resource-ollama gateway generate --role chat.default --expected-model qwen3.5:9b --json --prompt-stdin --max-tokens 64"] = providers.Result{Stdout: `{"response":"ok","eval_count":1}`}
	db := newSchemaDB(t)
	svc := newService(testAdapters(runner), routing.NewSQLRepository(db))

	resp, err := svc.Execute(context.Background(), baseRequest(sharedv1.Profile_PROFILE_LOCAL_ONLY), "sensitive prompt")
	require.NoError(t, err)
	require.True(t, resp.GetValid())
	require.Equal(t, "succeeded", resp.GetEvidence().GetStatus())
	require.Equal(t, "ollama", resp.GetEvidence().GetSelectedProvider())
	require.Equal(t, "qwen3.5:9b", resp.GetEvidence().GetSelectedModel())
	// A TEXT_GENERATION request runs no PII-detection pass, so the redacted-
	// evidence seam reports false (it is meaningful now, not unconditionally
	// true). The real privacy guarantee is below: the input never appears in the
	// command string, policy reasons, or output.
	require.False(t, resp.GetEvidence().GetPromptRedacted())
	require.False(t, resp.GetEvidence().GetResponseRedacted())
	require.NotContains(t, strings.Join(resp.GetEvidence().GetPolicyReasons(), " "), "sensitive prompt")
	require.NotContains(t, resp.GetOutputText(), "sensitive prompt")

	require.NotEmpty(t, runner.Commands)
	command := runner.Commands[len(runner.Commands)-1].String()
	require.NotContains(t, command, "sensitive prompt")
	require.Contains(t, command, "--expected-artifact-digest cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
	require.Contains(t, command, "--require-loopback", "local-only profile must reach the resource boundary")

	events, err := svc.ListEvidence(context.Background(), routing.EvidenceFilter{Limit: 10})
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, resp.GetEvidence().GetEventId(), events[0].GetEventId())
	require.Equal(t, "succeeded", events[0].GetStatus())
}

func TestLocalOnlyProfileAndArtifactDigestReachOllamaEmbedding(t *testing.T) {
	runner := roleRunner()
	svc := newService(testAdapters(runner), routing.NewSQLRepository(newSchemaDB(t)))
	req := baseRequest(sharedv1.Profile_PROFILE_LOCAL_ONLY)
	req.Kind = sharedv1.RequestKind_REQUEST_KIND_TEXT_EMBEDDING
	req.Role = "embedding.default"
	resp, err := svc.Execute(context.Background(), req, "embed this")
	require.NoError(t, err)
	require.True(t, resp.GetValid())
	require.Equal(t, "succeeded", resp.GetEvidence().GetStatus())
	require.NotEmpty(t, runner.Commands)
	command := runner.Commands[len(runner.Commands)-1].String()
	require.Contains(t, command, "gateway embed")
	require.Contains(t, command, "--expected-artifact-digest dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd")
	require.Contains(t, command, "--require-loopback")
}

func TestExecuteFailsClosedWhenEvidenceCannotBePersisted(t *testing.T) { // [REQ:AIGW-ROUTE-EVIDENCE]
	runner := roleRunner()
	runner.Results["resource-ollama gateway generate --role chat.default --expected-model qwen3.5:9b --json --prompt-stdin --max-tokens 64"] = providers.Result{Stdout: `{"response":"ok"}`}
	svc := newService(testAdapters(runner), routing.NewSQLRepository(nil))

	resp, err := svc.Execute(context.Background(), baseRequest(sharedv1.Profile_PROFILE_LOCAL_ONLY), "hello")
	require.Nil(t, resp)
	require.Error(t, err)
	require.Contains(t, err.Error(), "persist successful route evidence")
}

func TestMultimodalRouteUsesDeclaredModalityAndRedactsAttachmentEvidence(t *testing.T) { // [REQ:AIGW-MULTIMODAL-CONTRACT]
	runner := multimodalRoleRunner()
	runner.Results["resource-ollama gateway generate --role chat.default --expected-model qwen3.5:9b --json --input-json-stdin"] = providers.Result{Stdout: `{"response":"ok","eval_count":1}`}
	db := newSchemaDB(t)
	svc := newService(testAdapters(runner), routing.NewSQLRepository(db))
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
	svc := newService(testAdapters(runner), nil)
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
	runner := &mocks.FakeRunner{
		Results: map[string]providers.Result{
			"resource-ollama policy roles --json": {
				Stdout: `{"roles":[{"schema_version":"1","role":"chat.default","model":"qwen3.5:9b","rights":{"artifact_id":"fixture/qwen3.5:9b","revision":"fixture-rev1","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","license_id":"fixture-unrestricted","classification":"unrestricted","evidence":{"url":"https://example.invalid/fixture-license","revision":"fixture-license-v1","sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","observed_at":"2026-11-11"}},"required_capabilities":["generate","chat"]}]}`,
			},
			"resource-openrouter policy roles --json": {
				Stdout: `{"roles":[{"schema_version":"1","role":"chat.default","model":"qwen3.5:9b","rights":{"artifact_id":"fixture/qwen3.5:9b","revision":"fixture-rev1","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","license_id":"fixture-unrestricted","classification":"unrestricted","evidence":{"url":"https://example.invalid/fixture-license","revision":"fixture-license-v1","sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","observed_at":"2026-11-11"}},"required_capabilities":["generate","chat"]}]}`,
			},
			"resource-ollama policy resolve --role chat.default --json": {
				Stdout: `{"role":"chat.default","model":"qwen3.5:9b","rights":{"artifact_id":"fixture/qwen3.5:9b","revision":"fixture-rev1","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","license_id":"fixture-unrestricted","classification":"unrestricted","evidence":{"url":"https://example.invalid/fixture-license","revision":"fixture-license-v1","sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","observed_at":"2026-11-11"}}}`,
			},
			"resource-openrouter policy resolve --role chat.default --json": {
				Stdout: `{"role":"chat.default","model":"google/gemini-3.1-flash-lite-preview"}`,
			},
		},
		Errors: map[string]error{},
	}
	identity := `,"runtime_identity":{"provider":"ollama","model":"qwen3.5:9b","artifact_id":"fixture/qwen3.5:9b","artifact_revision":"fixture-rev1","digest_kind":"ollama_model_contents","digest":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","source":"ollama /api/tags","observed_at":"2026-11-11"}`
	embedRights := `{"artifact_id":"fixture/nomic-embed-text:latest","revision":"fixture-rev1","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","license_id":"fixture-unrestricted","classification":"unrestricted","evidence":{"url":"https://example.invalid/fixture-license","revision":"fixture-license-v1","sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","observed_at":"2026-11-11"}}`
	embedIdentity := `{"provider":"ollama","model":"nomic-embed-text:latest","artifact_id":"fixture/nomic-embed-text:latest","artifact_revision":"fixture-rev1","digest_kind":"ollama_model_contents","digest":"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd","source":"ollama /api/tags","observed_at":"2026-11-11"}`
	for key, result := range runner.Results {
		if key == "resource-ollama policy roles --json" {
			result.Stdout = strings.Replace(result.Stdout, `,"required_capabilities"`, identity+`,"required_capabilities"`, 1)
			result.Stdout = strings.TrimSuffix(result.Stdout, `]}`) + `,{"schema_version":"1","role":"embedding.default","model":"nomic-embed-text:latest","rights":` + embedRights + `,"runtime_identity":` + embedIdentity + `,"required_capabilities":["embedding"],"capabilities":["embedding"]}]}`
			runner.Results[key] = result
		}
		if key == "resource-ollama policy resolve --role chat.default --json" {
			result.Stdout = strings.Replace(result.Stdout, `,"rights":`, identity+`,"rights":`, 1)
			runner.Results[key] = result
		}
	}
	runner.Results["resource-ollama policy resolve --role embedding.default --json"] = providers.Result{Stdout: `{"role":"embedding.default","model":"nomic-embed-text:latest","rights":` + embedRights + `,"runtime_identity":` + embedIdentity + `,"capabilities":["embedding"],"locality":"local"}`}
	runner.Results["resource-ollama models inventory --json"] = providers.Result{Stdout: `{"source":"ollama /api/tags","models":[{"name":"qwen3.5:9b","digest":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},{"name":"nomic-embed-text:latest","digest":"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"}]}`}
	runner.DefaultOK = true
	runner.Default = providers.Result{Stdout: `{"response":"ok","eval_count":1}`}
	return runner
}

func withChatRuntimeIdentity(raw string) string {
	identity := `,"runtime_identity":{"provider":"ollama","model":"qwen3.5:9b","artifact_id":"fixture/qwen3.5:9b","artifact_revision":"fixture-rev1","digest_kind":"ollama_model_contents","digest":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","source":"ollama /api/tags","observed_at":"2026-11-11"}`
	return strings.Replace(raw, `,"rights":`, identity+`,"rights":`, 1)
}

func multimodalRoleRunner() *mocks.FakeRunner {
	runner := &mocks.FakeRunner{
		Results: map[string]providers.Result{
			"resource-ollama policy roles --json":                           {Stdout: `{"roles":[{"schema_version":"1","role":"chat.default","model":"qwen3.5:9b","rights":{"artifact_id":"fixture/qwen3.5:9b","revision":"fixture-rev1","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","license_id":"fixture-unrestricted","classification":"unrestricted","evidence":{"url":"https://example.invalid/fixture-license","revision":"fixture-license-v1","sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","observed_at":"2026-11-11"}},"required_capabilities":["generate","chat"],"modalities":{"input":["text","image"],"output":["text"]}}]}`},
			"resource-openrouter policy roles --json":                       {Stdout: `{"roles":[{"schema_version":"1","role":"chat.default","model":"qwen3.5:9b","rights":{"artifact_id":"fixture/qwen3.5:9b","revision":"fixture-rev1","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","license_id":"fixture-unrestricted","classification":"unrestricted","evidence":{"url":"https://example.invalid/fixture-license","revision":"fixture-license-v1","sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","observed_at":"2026-11-11"}},"required_capabilities":["generate","chat"],"modalities":{"input":["text","image"],"output":["text"]}}]}`},
			"resource-ollama policy resolve --role chat.default --json":     {Stdout: `{"role":"chat.default","model":"qwen3.5:9b","rights":{"artifact_id":"fixture/qwen3.5:9b","revision":"fixture-rev1","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","license_id":"fixture-unrestricted","classification":"unrestricted","evidence":{"url":"https://example.invalid/fixture-license","revision":"fixture-license-v1","sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","observed_at":"2026-11-11"}}}`},
			"resource-openrouter policy resolve --role chat.default --json": {Stdout: `{"role":"chat.default","model":"google/gemini-3.1-flash-lite-preview"}`},
		},
		Errors: map[string]error{},
	}
	identity := `,"runtime_identity":{"provider":"ollama","model":"qwen3.5:9b","artifact_id":"fixture/qwen3.5:9b","artifact_revision":"fixture-rev1","digest_kind":"ollama_model_contents","digest":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","source":"ollama /api/tags","observed_at":"2026-11-11"}`
	for key, result := range runner.Results {
		if key == "resource-ollama policy roles --json" {
			result.Stdout = strings.Replace(result.Stdout, `,"required_capabilities"`, identity+`,"required_capabilities"`, 1)
			runner.Results[key] = result
		}
		if key == "resource-ollama policy resolve --role chat.default --json" {
			result.Stdout = strings.Replace(result.Stdout, `,"rights":`, identity+`,"rights":`, 1)
			runner.Results[key] = result
		}
	}
	runner.Results["resource-ollama models inventory --json"] = providers.Result{Stdout: `{"source":"ollama /api/tags","models":[{"name":"qwen3.5:9b","digest":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"}]}`}
	runner.DefaultOK = true
	runner.Default = providers.Result{Stdout: `{"response":"ok","eval_count":1}`}
	return runner
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
		Stdout: withChatRuntimeIdentity(`{"role":"chat.default","model":"qwen3.5:9b","rights":{"artifact_id":"fixture/qwen3.5:9b","revision":"fixture-rev1","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","license_id":"fixture-unrestricted","classification":"unrestricted","evidence":{"url":"https://example.invalid/fixture-license","revision":"fixture-license-v1","sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","observed_at":"2026-11-11"}},"sampling_support":{"temperature":"honored"}}`),
	}
	runner.Results["resource-ollama gateway generate --role chat.default --expected-model qwen3.5:9b --json --prompt-stdin --max-tokens 64 --temperature 1.1"] = providers.Result{Stdout: `{"response":"ok","eval_count":1}`}
	db := newSchemaDB(t)
	svc := newService(testAdapters(runner), routing.NewSQLRepository(db))

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
	runner.Results["resource-ollama gateway generate --role chat.default --expected-model qwen3.5:9b --json --prompt-stdin --max-tokens 64"] = providers.Result{Stdout: `{"response":"ok","eval_count":1}`}
	db := newSchemaDB(t)
	svc := newService(testAdapters(runner), routing.NewSQLRepository(db))

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
		Stdout: withChatRuntimeIdentity(`{"role":"chat.default","model":"qwen3.5:9b","rights":{"artifact_id":"fixture/qwen3.5:9b","revision":"fixture-rev1","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","license_id":"fixture-unrestricted","classification":"unrestricted","evidence":{"url":"https://example.invalid/fixture-license","revision":"fixture-license-v1","sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","observed_at":"2026-11-11"}},"sampling_support":{"temperature":"ignored"}}`),
	}
	runner.Results["resource-openrouter policy resolve --role chat.default --json"] = providers.Result{
		Stdout: `{"role":"chat.default","model":"vendor/model","sampling_support":{"temperature":"honored"}}`,
	}
	runner.Results["resource-openrouter generate --role chat.default --json --max-tokens 64 --temperature 1.1"] = providers.Result{
		Stdout: `{"choices":[{"message":{"content":"ok"}}]}`,
	}
	db := newSchemaDB(t)
	svc := newService(testAdapters(runner), routing.NewSQLRepository(db))

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
		Stdout: withChatRuntimeIdentity(`{"role":"chat.default","model":"qwen3.5:9b","rights":{"artifact_id":"fixture/qwen3.5:9b","revision":"fixture-rev1","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","license_id":"fixture-unrestricted","classification":"unrestricted","evidence":{"url":"https://example.invalid/fixture-license","revision":"fixture-license-v1","sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","observed_at":"2026-11-11"}},"sampling_support":{"temperature":"rejected"}}`),
	}
	db := newSchemaDB(t)
	svc := newService(testAdapters(runner), routing.NewSQLRepository(db))

	req := baseRequest(sharedv1.Profile_PROFILE_LOCAL_ONLY)
	req.Sampling = &sharedv1.SamplingControls{Temperature: proto.Float64(1.1)}
	resp, err := svc.Execute(context.Background(), req, "hello")
	require.NoError(t, err)
	require.Equal(t, "failed", resp.GetEvidence().GetStatus())
	require.Equal(t, string(routing.FailureUnsupportedSampling), resp.GetEvidence().GetFailureClass())
	require.Empty(t, resp.GetOutputText())
}
