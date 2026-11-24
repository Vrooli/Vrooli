package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/lpbsidentity"
	"github.com/vrooli/browser-automation-studio/middleware"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

type gatewayFixture struct {
	t                   *testing.T
	request             *inferencev1.RunRequest
	source              string
	token               string
	callerAuthorization string
	response            *inferencev1.RunResponse
	err                 error
}

func (f *gatewayFixture) Run(_ context.Context, req *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
	f.request = req.Msg
	f.source = req.Header().Get("X-Vrooli-AI-Source")
	f.token = strings.TrimPrefix(req.Header().Get("Authorization"), "Bearer ")
	f.callerAuthorization = req.Header().Get(lpbsidentity.CreditCallerAuthorizationHeader)
	if f.err != nil {
		return nil, f.err
	}
	encoded, err := json.Marshal(`[ {"text":"Search"} ]`)
	require.NoError(f.t, err)
	if f.response != nil {
		return connect.NewResponse(f.response), nil
	}
	return connect.NewResponse(&inferencev1.RunResponse{ValueJson: string(encoded)}), nil
}

func TestGatewayRolePromptClientRetainsRecoveryIDOnTransportFailure(t *testing.T) {
	fixture := &gatewayFixture{t: t, err: connect.NewError(connect.CodeUnavailable, errors.New("connection interrupted"))}
	path, handler := inferenceconnect.NewInferenceServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	operationID := "aabbccddeeff00112233445566778899"
	client := &gatewayRolePromptClient{resolveURL: func(context.Context) (string, error) { return server.URL, nil }, httpClient: server.Client()}
	_, err := client.ExecutePromptWithRole(WithGatewayRootOperationID(context.Background(), operationID), "write.default", "return JSON")
	require.Error(t, err)
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
	require.ErrorContains(t, err, "root_operation_id="+operationID)
	connectError, ok := err.(*connect.Error)
	require.True(t, ok)
	require.Equal(t, operationID, connectError.Meta().Get("X-Vrooli-Root-Operation-ID"))
}

func (*gatewayFixture) RunBatch(context.Context, *connect.Request[inferencev1.RunBatchRequest]) (*connect.Response[inferencev1.RunBatchResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (*gatewayFixture) Embed(context.Context, *connect.Request[inferencev1.EmbedRequest]) (*connect.Response[inferencev1.EmbedResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (*gatewayFixture) Decide(context.Context, *connect.Request[inferencev1.DecideRequest]) (*connect.Response[inferencev1.DecideResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
func (*gatewayFixture) DecideBatch(context.Context, *connect.Request[inferencev1.DecideBatchRequest]) (*connect.Response[inferencev1.DecideBatchResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func TestGatewayRolePromptClientUsesSharedGatewayAndSession(t *testing.T) {
	fixture := &gatewayFixture{t: t}
	path, handler := inferenceconnect.NewInferenceServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()

	ctx := monetization.WithAccessToken(context.Background(), "session-token")
	ctx = middleware.WithOwnKeySourceHint(ctx, true)
	ctx = WithGatewayBrowserSessionBinding(ctx, &inferencev1.BrowserSessionBinding{RunId: "87026a72-90ba-4e0b-9897-b2b09dc8a750", SessionId: "bas-session-1", GrantReceipt: "grant-receipt"})
	client := &gatewayRolePromptClient{resolveURL: func(context.Context) (string, error) { return server.URL, nil }, httpClient: server.Client()}
	got, err := client.ExecutePromptWithRole(ctx, "write.default", "return JSON")
	require.NoError(t, err)
	require.Equal(t, `[ {"text":"Search"} ]`, got)
	require.Equal(t, "own-key", fixture.source)
	require.Equal(t, "session-token", fixture.token)
	require.Equal(t, "return JSON", fixture.request.GetSource())
	require.Equal(t, "write.default", fixture.request.GetRole())
	require.Equal(t, sharedv1.Profile_PROFILE_REMOTE_ONLY, fixture.request.GetProfile())
	require.Equal(t, sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL, fixture.request.GetPrivacyClass())
	require.Equal(t, basAIPrivacySource, fixture.request.GetPrivacyProvenance())
	require.Len(t, fixture.request.GetRootOperationId(), 32)
	require.Equal(t, "bas-session-1", fixture.request.GetBrowserSession().GetSessionId())
	require.NotContains(t, fixture.request.GetSource(), "legacy-raw-key")
}

func TestGatewayRolePromptClientForwardsScenarioPrincipalSeparatelyFromUser(t *testing.T) {
	fixture := &gatewayFixture{t: t}
	path, handler := inferenceconnect.NewInferenceServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := &gatewayRolePromptClient{
		resolveURL:                       func(context.Context) (string, error) { return server.URL, nil },
		httpClient:                       server.Client(),
		resolveCreditCallerAuthorization: func(context.Context) (string, error) { return "signed-bas-principal", nil },
	}
	ctx := monetization.WithAccessToken(monetization.WithSnapshot(context.Background(), &monetization.Snapshot{
		Features:        []string{entitlement.FeatureAI},
		AppEntitlements: []entitlementclient.AppEntitlement{{AppKey: "browser-automation-studio", Status: "active", Features: []string{entitlement.FeatureAI}}},
	}), "human-session-token")
	ctx = context.WithValue(ctx, privacyConsentSourceContextKey{}, "vrooli-credits")
	if _, err := client.ExecutePromptWithRole(ctx, "write.default", "synthetic prompt"); err != nil {
		t.Fatalf("ExecutePromptWithRole: %v", err)
	}
	if fixture.token != "human-session-token" {
		t.Fatalf("user Authorization = %q", fixture.token)
	}
	if fixture.callerAuthorization != "Bearer signed-bas-principal" {
		t.Fatalf("caller principal = %q", fixture.callerAuthorization)
	}
}

func TestGatewayRolePromptClientReceiptPreservesGatewayAccountingAndAppliedSettings(t *testing.T) {
	credits := int64(17)
	temperature := 0.25
	fixture := &gatewayFixture{t: t, response: &inferencev1.RunResponse{
		ValueJson: `"ok"`, BillingOperationId: "lpbs-op-123", Provider: "hosted-provider", Model: "model-x", ProviderCallsMade: 1,
		Usage: &inferencev1.Usage{InputTokens: 13, OutputTokens: 5, CostMicros: 29,
			InputTokensProvenance:  inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED,
			OutputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_GATEWAY_ESTIMATE,
			CostProvenance:         inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED,
			CreditsCharged:         &credits, AccountingStatus: "settled"},
		Applied: &sharedv1.AppliedSettings{MaxOutputTokensEffective: 321,
			MaxOutputTokensSource: sharedv1.OutputCapSource_OUTPUT_CAP_SOURCE_ROLE_POLICY,
			TemperatureSent:       &temperature, TemperatureSupport: sharedv1.SamplingSupport_SAMPLING_SUPPORT_HONORED},
	}}
	path, handler := inferenceconnect.NewInferenceServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := &gatewayRolePromptClient{resolveURL: func(context.Context) (string, error) { return server.URL, nil }, httpClient: server.Client()}
	ctx := middleware.WithOwnKeySourceHint(monetization.WithAccessToken(context.Background(), "session-token"), true)
	got, err := client.ExecutePromptWithReceipt(WithGatewayRootOperationID(ctx, "receipt-operation"), "write.default", "return string")
	require.NoError(t, err)
	require.Equal(t, "ok", got.Value)
	require.Equal(t, "receipt-operation", got.Receipt.RootOperationID)
	require.Equal(t, "lpbs-op-123", got.Receipt.BillingOperationID)
	require.Equal(t, "own-key", got.Receipt.Source)
	require.Equal(t, "hosted-provider", got.Receipt.Provider)
	require.Equal(t, "model-x", got.Receipt.Model)
	require.EqualValues(t, 1, got.Receipt.ProviderCallsMade)
	require.EqualValues(t, 13, got.Receipt.InputTokens)
	require.EqualValues(t, 5, got.Receipt.OutputTokens)
	require.EqualValues(t, 29, got.Receipt.CostMicros)
	require.Equal(t, "USAGE_PROVENANCE_PROVIDER_REPORTED", got.Receipt.InputTokensProvenance)
	require.Equal(t, "USAGE_PROVENANCE_GATEWAY_ESTIMATE", got.Receipt.OutputTokensProvenance)
	require.Equal(t, "settled", got.Receipt.AccountingStatus)
	require.NotNil(t, got.Receipt.CreditsCharged)
	require.EqualValues(t, 17, *got.Receipt.CreditsCharged)
	require.NotNil(t, got.Receipt.Applied)
	require.EqualValues(t, 321, got.Receipt.Applied.MaxOutputTokensEffective)
	require.Equal(t, "OUTPUT_CAP_SOURCE_ROLE_POLICY", got.Receipt.Applied.MaxOutputTokensSource)
	require.NotNil(t, got.Receipt.Applied.TemperatureSent)
	require.Equal(t, 0.25, *got.Receipt.Applied.TemperatureSent)
	require.Equal(t, "SAMPLING_SUPPORT_HONORED", got.Receipt.Applied.TemperatureSupport)
	encoded, err := json.Marshal(got.Receipt)
	require.NoError(t, err)
	var shape map[string]any
	require.NoError(t, json.Unmarshal(encoded, &shape))
	require.Equal(t, "receipt-operation", shape["rootOperationId"])
	require.Equal(t, "lpbs-op-123", shape["billingOperationId"])
	require.Equal(t, "own-key", shape["source"])
	require.Equal(t, "13", shape["inputTokens"], "int64 accounting values remain exact across JavaScript adapter boundaries")
	require.Equal(t, "29", shape["costMicros"])
	require.Equal(t, "17", shape["creditsCharged"])
	_, snakeCase := shape["root_operation_id"]
	require.False(t, snakeCase, "BAS adapter receipt uses the Playwright camelCase contract")
	applied, ok := shape["applied"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(321), applied["maxOutputTokensEffective"])
}

func TestGatewayPromptReceiptKeepsZeroUsageFieldsPresent(t *testing.T) {
	fixture := &gatewayFixture{t: t, response: &inferencev1.RunResponse{ValueJson: `"ok"`}}
	path, handler := inferenceconnect.NewInferenceServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := &gatewayRolePromptClient{resolveURL: func(context.Context) (string, error) { return server.URL, nil }, httpClient: server.Client()}
	result, err := client.ExecutePromptWithReceipt(context.Background(), "write.default", "return string")
	require.NoError(t, err)
	encoded, err := json.Marshal(result.Receipt)
	require.NoError(t, err)
	var shape map[string]any
	require.NoError(t, json.Unmarshal(encoded, &shape))
	require.Equal(t, "0", shape["inputTokens"])
	require.Equal(t, "0", shape["outputTokens"])
}

func TestPrivacyConsentContextMiddlewareValidatesSourcePreferenceSeparatelyFromReceipt(t *testing.T) {
	for _, tc := range []struct {
		name       string
		source     string
		wantStatus int
		wantValue  string
	}{
		{name: "own-key preference without receipt", source: "own-key", wantStatus: http.StatusNoContent, wantValue: "own-key"},
		{name: "credits preference", source: "vrooli-credits", wantStatus: http.StatusNoContent, wantValue: "vrooli-credits"},
		{name: "local preference", source: "local", wantStatus: http.StatusNoContent, wantValue: "local"},
		{name: "invalid preference fails closed", source: "provider-that-does-not-exist", wantStatus: http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				called = true
				require.Equal(t, tc.wantValue, PrivacyConsentSourceFromContext(req.Context()))
				if req.Header.Get("X-Vrooli-AI-Consent-Grant") == "" {
					// An absent receipt does not erase the selected source preference.
					require.NotEmpty(t, PrivacyConsentSourceFromContext(req.Context()))
				}
				w.WriteHeader(http.StatusNoContent)
			})
			req := httptest.NewRequest(http.MethodPost, "/ai", nil)
			req.Header.Set("X-Vrooli-AI-Consent-Source", tc.source)
			recorder := httptest.NewRecorder()
			PrivacyConsentContextMiddleware(next).ServeHTTP(recorder, req)
			require.Equal(t, tc.wantStatus, recorder.Code)
			require.Equal(t, tc.wantValue != "", called)
		})
	}
}

func TestGatewayRolePromptClientExecutePromptUsesWorkflowRole(t *testing.T) {
	fixture := &gatewayFixture{t: t}
	path, handler := inferenceconnect.NewInferenceServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &gatewayRolePromptClient{resolveURL: func(context.Context) (string, error) { return server.URL, nil }, httpClient: server.Client()}
	got, err := client.ExecutePrompt(context.Background(), "return a workflow JSON object")
	require.NoError(t, err)
	require.NotEmpty(t, got)
	require.Equal(t, "write.default", fixture.request.GetRole())
	require.Equal(t, "return a workflow JSON object", fixture.request.GetSource())
	require.Equal(t, sharedv1.Profile_PROFILE_LOCAL_ONLY, fixture.request.GetProfile())
	require.Equal(t, sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL, fixture.request.GetPrivacyClass())
	require.Equal(t, basAIPrivacySource, fixture.request.GetPrivacyProvenance())
}

func TestGatewayRolePromptClientMapsInBandRefusalsAndSurfacesRecoveryID(t *testing.T) {
	for _, test := range []struct {
		name      string
		code      inferencev1.InferenceErrorCode
		want      connect.Code
		uncertain bool
		noRoute   bool
	}{
		{name: "hosted refusal", code: inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE, want: connect.CodeUnavailable},
		{name: "uncertain outcome", code: inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_OUTCOME_UNCERTAIN, want: connect.CodeUnavailable, uncertain: true},
		{name: "no eligible route", code: inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_NO_ELIGIBLE_ROUTE, want: connect.CodeFailedPrecondition, noRoute: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			message := "hosted operation refused"
			if test.noRoute {
				message = "no eligible route is configured for this browser source and profile; select an available source or ask the account owner to enable or upgrade an eligible route. No model inference was dispatched and funding was not changed"
			}
			fixture := &gatewayFixture{t: t, response: &inferencev1.RunResponse{Error: &inferencev1.InferenceError{Code: test.code, Message: message}}}
			path, handler := inferenceconnect.NewInferenceServiceHandler(fixture)
			mux := http.NewServeMux()
			mux.Handle(path, handler)
			server := httptest.NewServer(mux)
			defer server.Close()
			operationID := "aabbccddeeff00112233445566778899"
			client := &gatewayRolePromptClient{resolveURL: func(context.Context) (string, error) { return server.URL, nil }, httpClient: server.Client()}
			ctx := WithGatewayRootOperationID(context.Background(), operationID)
			_, err := client.ExecutePromptWithRole(ctx, "write.default", "return JSON")
			require.Error(t, err)
			require.Equal(t, test.want, connect.CodeOf(err))
			connectErr, ok := err.(*connect.Error)
			require.True(t, ok)
			require.Equal(t, test.code.String(), connectErr.Meta().Get("X-Vrooli-AI-Gateway-Error-Code"))
			if test.noRoute {
				require.Empty(t, connectErr.Meta().Get("X-Vrooli-Root-Operation-ID"), "no provider ran, so callers must not be nudged toward a recovery/retry flow")
				require.Equal(t, "local", connectErr.Meta().Get("X-Vrooli-AI-Source"), "the existing UI needs the selected source to explain a safe next step")
				require.ErrorContains(t, err, "select an available source or ask the account owner to enable or upgrade an eligible route")
				require.ErrorContains(t, err, "funding was not changed")
			} else {
				require.Equal(t, operationID, connectErr.Meta().Get("X-Vrooli-Root-Operation-ID"))
			}
			if test.uncertain {
				require.ErrorContains(t, err, "root_operation_id="+operationID)
			}
			require.Equal(t, operationID, fixture.request.GetRootOperationId())
		})
	}
}

func TestBASGatewayRouteSelectsOnlySharedAuthority(t *testing.T) {
	tests := []struct {
		name      string
		ctx       context.Context
		wantSrc   string
		wantProf  sharedv1.Profile
		wantToken string
		wantErr   string
	}{
		{name: "anonymous stays local", ctx: context.Background(), wantSrc: "local", wantProf: sharedv1.Profile_PROFILE_LOCAL_ONLY},
		{name: "signed-in without AI entitlement is denied for platform credits", ctx: monetization.WithAccessToken(context.Background(), "session"), wantErr: "requires the signed session's ai feature entitlement"},
		{name: "signed-in without AI in entitlement is denied for platform credits", ctx: monetization.WithAccessToken(monetization.WithSnapshot(context.Background(), &monetization.Snapshot{Features: []string{entitlement.FeatureRecording}}), "session"), wantErr: "requires the signed session's ai feature entitlement"},
		{name: "foreign app AI cannot authorize BAS credits", ctx: monetization.WithAccessToken(monetization.WithSnapshot(context.Background(), &monetization.Snapshot{Features: []string{entitlement.FeatureAI}, AppEntitlements: []entitlementclient.AppEntitlement{{AppKey: "audio-tools", Status: "active", Features: []string{entitlement.FeatureAI}}}}), "session"), wantErr: "requires the signed session's ai feature entitlement"},
		{name: "signed-in with BAS AI entitlement uses platform credits", ctx: monetization.WithAccessToken(monetization.WithSnapshot(context.Background(), &monetization.Snapshot{Features: []string{entitlement.FeatureAI}, AppEntitlements: []entitlementclient.AppEntitlement{{AppKey: "browser-automation-studio", Status: "active", Features: []string{entitlement.FeatureAI}}}}), "session"), wantSrc: "vrooli-credits", wantProf: sharedv1.Profile_PROFILE_REMOTE_ONLY, wantToken: "session"},
		{name: "saved own-key preference uses session credential authority", ctx: middleware.WithOwnKeySourceHint(monetization.WithAccessToken(context.Background(), "session"), true), wantSrc: "own-key", wantProf: sharedv1.Profile_PROFILE_REMOTE_ONLY, wantToken: "session"},
		{name: "raw BYOK is not authority", ctx: middleware.WithOwnKeySourceHint(context.Background(), true), wantErr: "requires a shared signed-in session"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, profile, token, err := basGatewayRoute(tt.ctx)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantSrc, source)
			require.Equal(t, tt.wantProf, profile)
			require.Equal(t, tt.wantToken, token)
		})
	}
}

func TestBASVrooliCreditGateIsServerAuthoritativeAndKeepsGatewayMeter(t *testing.T) {
	// Removing this feature gate must make the negative case proceed. This
	// regression assertion is intentionally attached to the production route
	// selector so a gate-removal mutation fails the conformance test.
	ctx := monetization.WithAccessToken(context.Background(), "verified-session")
	source, _, _, err := basGatewayRoute(ctx)
	require.ErrorContains(t, err, "signed session's ai feature entitlement")
	require.Empty(t, source)
	ctx = monetization.WithSnapshot(ctx, &monetization.Snapshot{Features: []string{entitlement.FeatureRecording}})
	source, _, _, err = basGatewayRoute(ctx)
	require.ErrorContains(t, err, "signed session's ai feature entitlement")
	require.Empty(t, source)

	ctx = monetization.WithSnapshot(ctx, &monetization.Snapshot{Features: []string{entitlement.FeatureAI}})
	source, profile, token, err := basGatewayRoute(ctx)
	require.NoError(t, err)
	require.Equal(t, "vrooli-credits", source)
	require.Equal(t, sharedv1.Profile_PROFILE_REMOTE_ONLY, profile)
	require.Equal(t, "verified-session", token)
}

func TestBASMonetizationDeclarationKeepsAIFeatureAndSharedMeter(t *testing.T) {
	data, err := os.ReadFile("../../../.vrooli/monetization.json")
	require.NoError(t, err)
	var declaration struct {
		ProductShape string `json:"product_shape"`
		Features     []struct {
			Key             string   `json:"key"`
			EnforcementPath []string `json:"enforcement_paths"`
		} `json:"features"`
		Meters []struct {
			LimitKey        string   `json:"limit_key"`
			EnforcementPath []string `json:"enforcement_paths"`
		} `json:"meters"`
	}
	require.NoError(t, json.Unmarshal(data, &declaration))
	require.Equal(t, "feature_gated", declaration.ProductShape)

	var aiFeature, aiCreditsMeter bool
	for _, feature := range declaration.Features {
		if feature.Key == "ai" {
			aiFeature = true
			require.Contains(t, feature.EnforcementPath, "api/handlers/ai/gateway_client.go")
			require.Contains(t, feature.EnforcementPath, "api/handlers/vision_navigation/connect_handler.go")
		}
	}
	for _, meter := range declaration.Meters {
		if meter.LimitKey == "ai_credits" {
			aiCreditsMeter = true
			require.Contains(t, meter.EnforcementPath, "api/handlers/ai/gateway_client.go")
			require.Contains(t, meter.EnforcementPath, "api/handlers/vision_navigation/connect_handler.go")
			require.NotContains(t, meter.EnforcementPath, "api/services/credits/service.go")
		}
		if meter.LimitKey == "workflow_executions" {
			require.Contains(t, meter.EnforcementPath, "api/services/workflow/executions.go")
			require.Contains(t, meter.EnforcementPath, "api/services/workflow/adhoc_execution.go")
			require.Contains(t, meter.EnforcementPath, "api/services/scheduler/scheduler.go")
			require.NotContains(t, meter.EnforcementPath, "api/services/credits/monetization_transport.go")
		}
	}
	require.True(t, aiFeature, "BAS must retain the declared AI feature")
	require.True(t, aiCreditsMeter, "BAS must retain the shared ai_credits meter")
}
