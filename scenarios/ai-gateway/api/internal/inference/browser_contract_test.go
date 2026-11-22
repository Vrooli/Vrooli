package inference

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ai-gateway/internal/providers"
	providermocks "ai-gateway/internal/providers/mocks"
	"ai-gateway/internal/rolecatalog"

	"github.com/stretchr/testify/require"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestBrowserRoleCatalogMapsOnlyToExistingInferenceRoutes(t *testing.T) {
	catalog, err := rolecatalog.LoadCatalog("../../../config/inference-role-catalog.json")
	require.NoError(t, err)
	require.NoError(t, catalog.Validate())

	resourceRoles := func(name string) []string {
		role, ok := catalog.Roles[name]
		require.True(t, ok, "catalog must declare %s", name)
		roles := make([]string, 0, len(role.Candidates))
		for _, candidate := range role.Candidates {
			roles = append(roles, candidate.ResourceRole)
		}
		return roles
	}

	require.Equal(t, []string{"vision.default", "vision.default", "vision.default", "vision.default"}, resourceRoles("browser.locate"))
	require.True(t, IsVisualLocateRole("browser.locate"), "browser.locate must use locate.visual's normalized-coordinate contract")
	require.Equal(t, []string{"decide.local", "decide.jev.edenai", "decide.jev.typesafe", "decide.clef.cloudflare", "classify.fast"}, resourceRoles("browser.decide"))
	require.True(t, catalog.Roles["browser.decide"].IsDecision())
	require.Equal(t, []string{"chat.default", "extract.structured", "extract.structured", "extract.structured"}, resourceRoles("browser.plan"))
	require.Equal(t, []string{"chat.default", "extract.structured", "extract.structured", "extract.structured"}, resourceRoles("browser.extract"))
}

func TestBrowserRunRolesRequireCorrelationFieldsBeforeRepositoryDispatch(t *testing.T) {
	for _, role := range []string{"browser.locate", "browser.plan", "browser.extract"} {
		t.Run(role, func(t *testing.T) {
			repository := &browserContractRepository{}
			service := NewService(repository)
			request := ProviderRequest{
				Source: "caller supplied page content", SchemaJSON: `{"type":"object"}`, Role: role,
				PrivacyProvenance: "browser_contract.synthetic_fixture",
			}
			if role == "browser.locate" {
				request.Attachments = []*sharedv1.Attachment{visualAttachment(800, 400)}
			}

			missingRoot := service.Run(context.Background(), request)
			require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, missingRoot.GetError().GetCode())
			require.Equal(t, "root_operation_id", missingRoot.GetError().GetConstruct())
			require.Zero(t, repository.runCalls)

			request.RootOperationID = strings.Repeat("a", 64)
			missingBinding := service.Run(context.Background(), request)
			require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, missingBinding.GetError().GetCode())
			require.Equal(t, "browser_session", missingBinding.GetError().GetConstruct())
			require.Zero(t, repository.runCalls)

			request.BrowserSession = &inferencev1.BrowserSessionBinding{RunId: "run-1", SessionId: "session-1", GrantReceipt: "opaque-receipt"}
			accepted := service.Run(context.Background(), request)
			require.True(t, accepted.GetValidated(), accepted.GetError())
			require.Equal(t, 1, repository.runCalls)
		})
	}
	// Run's fields are correlation inputs, not an authorization proof. The
	// authenticated AM/BAS context required for a customer showcase remains an
	// upstream owner boundary and is not synthesized by this request shape.
}

func TestBrowserPlanAndExtractCannotEscapeSuppliedSchema(t *testing.T) {
	schema := `{"type":"object","properties":{"actions":{"type":"array","items":{"type":"object","properties":{"kind":{"enum":["observe","click","fill"]}},"required":["kind"],"additionalProperties":false}}},"required":["actions"],"additionalProperties":false}`
	for _, role := range []string{"browser.plan", "browser.extract"} {
		t.Run(role, func(t *testing.T) {
			repository := &browserContractRepository{result: ProviderResult{ValueJSON: `{"actions":[{"kind":"execute-code","raw_cdp":true}]}`, ProviderCallsMade: 1}}
			request := validBrowserRunRequest(role, schema)
			request.MaxProviderCalls = 1
			response := NewService(repository).Run(context.Background(), request)
			require.False(t, response.GetValidated())
			require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_VALIDATION_FAILED, response.GetError().GetCode())
			require.Equal(t, 1, repository.runCalls)
		})
	}
}

func TestBrowserLocateUsesExistingCoordinateNormalizationContract(t *testing.T) {
	request := validBrowserRunRequest("browser.locate", `{"type":"object","required":["found","bounds","confidence"],"properties":{"found":{"type":"boolean"},"bounds":{"type":"array","items":{"type":"number"}},"confidence":{"type":"number"}}}`)
	request.Attachments = []*sharedv1.Attachment{visualAttachment(800, 400)}
	repository := &browserContractRepository{result: ProviderResult{
		ValueJSON:            `{"found":true,"bounds":[100,200,400,800],"confidence":0.75}`,
		CoordinateConvention: CoordinateConventionNormalized1000,
	}}
	response := NewService(repository).Run(context.Background(), request)
	require.True(t, response.GetValidated(), response.GetError())
	require.JSONEq(t, `{"found":true,"bounds":[0.1,0.2,0.4,0.8],"confidence":0.75}`, response.GetValueJson())

	repository = &browserContractRepository{result: ProviderResult{
		ValueJSON:            `{"found":true,"bounds":[1001,200,400,800],"confidence":0.75}`,
		CoordinateConvention: CoordinateConventionNormalized1000,
	}}
	response = NewService(repository).Run(context.Background(), request)
	require.False(t, response.GetValidated())
	require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_VALIDATION_FAILED, response.GetError().GetCode())
}

func TestBrowserRolesFailClosedWhenSelectedSourceHasNoCandidate(t *testing.T) {
	for _, role := range []string{"browser.locate", "browser.plan", "browser.extract", "browser.decide"} {
		t.Run(role, func(t *testing.T) {
			runner := &providermocks.FakeRunner{}
			catalog := rolecatalog.RoleCatalog{SchemaVersion: 1, Roles: map[string]rolecatalog.InferenceRole{
				role: {Candidates: []rolecatalog.Candidate{{Provider: providers.ProviderOllama, ResourceRole: "vision.default", Reason: "local-only fixture"}}},
			}}
			if role == "browser.decide" {
				catalog.Roles[role] = rolecatalog.InferenceRole{Kind: rolecatalog.RoleKindDecision, Candidates: []rolecatalog.Candidate{{Provider: providers.ProviderOllaya, ResourceRole: "decide.local", Reason: "local-only fixture"}}}
			}
			repository, err := NewResourceRepository(catalog, []providers.Adapter{
				{Provider: providers.ProviderOllama, Locality: "local", CommandName: "resource-ollama", Runner: runner},
				{Provider: providers.ProviderOllaya, Locality: "local", CommandName: "resource-ollaya", Runner: runner},
			})
			require.NoError(t, err)
			ctx := providers.WithSource(context.Background(), "own-key")
			if role == "browser.decide" {
				response := repository.Decide(ctx, DecisionRequest{Role: role, State: "rank the declared choices", PrivacyProvenance: "browser_contract.synthetic_fixture", Questions: []*inferencev1.DecisionQuestion{choice("action", "observe", "click")}}, nil)
				require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_NO_ELIGIBLE_ROUTE, response.GetError().GetCode())
				require.Equal(t, "source_or_route_eligibility", response.GetError().GetConstruct())
			} else {
				result, err := repository.Run(ctx, ProviderRequest{
					RootOperationID: strings.Repeat("b", 64),
					BrowserSession:  &inferencev1.BrowserSessionBinding{RunId: "run-1", SessionId: "session-1", GrantReceipt: "opaque-receipt"},
					Role:            role, SchemaJSON: `{"type":"object"}`, Source: "synthetic page input",
				})
				require.ErrorIs(t, err, ErrNoEligibleRoute)
				require.Zero(t, result.ProviderCallsMade)
			}
			require.Empty(t, runner.Commands, "a source mismatch must be refused before model resolution or dispatch")
		})
	}
}

func TestBrowserNoEligibleRouteOffersNextStepWithoutInferenceOrFunding(t *testing.T) {
	response := applyProviderError(&inferencev1.RunResponse{Usage: &inferencev1.Usage{}}, validBrowserRunRequest("browser.extract", `{"type":"object"}`), ErrNoEligibleRoute)
	require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_NO_ELIGIBLE_ROUTE, response.GetError().GetCode())
	require.Contains(t, response.GetError().GetMessage(), "select an available source or ask the account owner to enable or upgrade an eligible route")
	require.Contains(t, response.GetError().GetMessage(), "No model inference was dispatched and funding was not changed")
	require.Empty(t, response.GetProvider())
	require.Empty(t, response.GetModel())
	require.Zero(t, response.GetUsage().GetCreditsCharged())
}

func TestBrowserRunRolesDoNotFallbackAfterUncertainMeteredOutcome(t *testing.T) {
	for _, test := range []struct {
		role        string
		meteredRole string
		attachments []*sharedv1.Attachment
	}{
		{role: "browser.locate", meteredRole: "vision.default", attachments: []*sharedv1.Attachment{visualAttachment(640, 480)}},
		{role: "browser.plan", meteredRole: "extract.structured"},
		{role: "browser.extract", meteredRole: "extract.structured"},
	} {
		t.Run(test.role, func(t *testing.T) {
			var meteredCalls int
			meteredServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				meteredCalls++
				w.WriteHeader(http.StatusBadGateway)
				_, _ = w.Write([]byte(`{"error":"uncertain after dispatch"}`))
			}))
			defer meteredServer.Close()

			runner := &providermocks.FakeRunner{DefaultOK: true, Default: providers.Result{Stdout: `{"choices":[{"message":{"content":"{}"}}]}`}}
			catalog := rolecatalog.RoleCatalog{SchemaVersion: 1, Roles: map[string]rolecatalog.InferenceRole{
				test.role: {Candidates: []rolecatalog.Candidate{
					{Provider: providers.ProviderMetered, ResourceRole: test.meteredRole, Reason: "fixture account route"},
					{Provider: providers.ProviderOpenRouter, ResourceRole: test.meteredRole, Reason: "fixture BYOK fallback"},
				}},
			}}
			repository, err := NewResourceRepository(catalog, []providers.Adapter{
				{Provider: providers.ProviderMetered, Locality: "remote", Metered: providers.NewMeteredClient(providers.MeteredClientOptions{BaseURL: meteredServer.URL})},
				{Provider: providers.ProviderOpenRouter, Locality: "remote", CommandName: "fixture-openrouter", Runner: runner},
			})
			require.NoError(t, err)
			result, err := repository.Run(providers.WithAccessToken(context.Background(), "Bearer fixture"), ProviderRequest{
				RootOperationID: strings.Repeat("d", 64),
				BrowserSession:  &inferencev1.BrowserSessionBinding{RunId: "run-1", SessionId: "session-1", GrantReceipt: "opaque-receipt"},
				Role:            test.role, SchemaJSON: `{"type":"object"}`, Instruction: "Return a bounded browser result.", Source: "synthetic page data", Attachments: test.attachments,
				Profile: sharedv1.Profile_PROFILE_REMOTE_ONLY, PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_PUBLIC,
			})
			require.ErrorIs(t, err, ErrOutcomeUncertain)
			require.Equal(t, 1, meteredCalls)
			require.EqualValues(t, 1, result.ProviderCallsMade)
			require.Empty(t, runner.Commands, "an uncertain funded attempt must not silently fall back to BYOK")
		})
	}
}

func TestBrowserDecisionRoleIsUnavailableWithoutVerifiedOwnerBinding(t *testing.T) {
	repository := &browserContractRepository{}
	service := NewService(repository)
	response := service.Decide(context.Background(), DecisionRequest{
		Role: "browser.decide", State: "choose the next action", PrivacyProvenance: "browser_contract.synthetic_fixture",
		Questions: []*inferencev1.DecisionQuestion{choice("action", "observe", "click")},
	})
	require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_NO_ELIGIBLE_ROUTE, response.GetError().GetCode())
	require.Equal(t, "browser_session", response.GetError().GetConstruct())
	require.Contains(t, response.GetError().GetMessage(), "no inference was dispatched")
	require.Zero(t, repository.decideCalls)

	// The same guard is intentionally scoped to browser roles. Generic decision
	// roles continue through the existing decide.fast route.
	response = service.Decide(context.Background(), DecisionRequest{
		Role: "decide.fast", State: "choose the next action", PrivacyProvenance: "browser_contract.synthetic_fixture",
		Questions: []*inferencev1.DecisionQuestion{choice("action", "observe", "click")},
	})
	require.Nil(t, response.GetError())
	require.Equal(t, 1, repository.decideCalls)
}

func TestBrowserDecisionDoesNotAcceptCallerBindingAsAuthority(t *testing.T) {
	repository := &browserContractRepository{}
	service := NewService(repository)
	request := DecisionRequest{
		Role: "browser.decide", State: "choose the next action", PrivacyProvenance: "browser_contract.synthetic_fixture",
		Questions: []*inferencev1.DecisionQuestion{choice("action", "observe", "click")},
	}
	response := service.Decide(context.Background(), request)
	require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_NO_ELIGIBLE_ROUTE, response.GetError().GetCode())
	require.Zero(t, repository.decideCalls)

	// The current Decide wire contract has no AM/BAS verified browser-session
	// binding. Guessed values are unknown protobuf fields and fail decoding before
	// dispatch. Do not add caller fields as a substitute for an existing owner.
	for _, raw := range []string{
		`{"role":"browser.decide","browserSession":{"runId":"forged","sessionId":"fake","grantReceipt":"guess"}}`,
		`{"role":"browser.decide","rootOperationId":"root-a","browserSession":{"runId":"run-b","sessionId":"session-b","grantReceipt":"guess"}}`,
	} {
		var decoded inferencev1.DecideRequest
		err := protojson.Unmarshal([]byte(raw), &decoded)
		require.Error(t, err, "caller fields cannot establish a verified browser binding")
	}
	require.Zero(t, repository.decideCalls)
}

func validBrowserRunRequest(role, schema string) ProviderRequest {
	return ProviderRequest{
		RootOperationID: strings.Repeat("c", 64),
		BrowserSession:  &inferencev1.BrowserSessionBinding{RunId: "run-1", SessionId: "session-1", GrantReceipt: "opaque-receipt"},
		Role:            role, SchemaJSON: schema, Source: "synthetic browser content",
		PrivacyProvenance: "browser_contract.synthetic_fixture",
	}
}

type browserContractRepository struct {
	result      ProviderResult
	runCalls    int
	decideCalls int
}

func (r *browserContractRepository) Run(_ context.Context, request ProviderRequest) (ProviderResult, error) {
	r.runCalls++
	if r.result.ValueJSON != "" {
		return r.result, nil
	}
	if IsVisualLocateRole(request.Role) {
		return ProviderResult{ValueJSON: `{"found":false,"bounds":[0,0,0,0],"confidence":0}`, CoordinateConvention: CoordinateConventionNormalized1000, Provider: "fixture", Model: "fixture-model"}, nil
	}
	return ProviderResult{ValueJSON: `{}`, Provider: "fixture", Model: "fixture-model"}, nil
}

func (r *browserContractRepository) Decide(_ context.Context, _ DecisionRequest, _ TypedRunner) *inferencev1.DecideResponse {
	r.decideCalls++
	return &inferencev1.DecideResponse{Usage: &inferencev1.Usage{}, Method: inferencev1.DecisionMethod_DECISION_METHOD_DECISION_MODEL}
}
