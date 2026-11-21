package ai

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

type gatewayInferenceFixture struct {
	inferenceconnect.UnimplementedInferenceServiceHandler
	request       *inferencev1.RunRequest
	source        string
	authorization string
	consent       string
}

func (f *gatewayInferenceFixture) Run(_ context.Context, request *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
	f.request = request.Msg
	f.source = request.Header().Get("X-Vrooli-AI-Source")
	f.authorization = request.Header().Get("Authorization")
	f.consent = request.Header().Get("X-Vrooli-AI-Consent-Grant")
	return connect.NewResponse(&inferencev1.RunResponse{ValueJson: `"echo safe"`, Validated: true}), nil
}

func TestGatewayProviderUsesAuthorRoleAndSourceBoundProfile(t *testing.T) {
	tests := []struct {
		source        string
		profile       sharedv1.Profile
		privacy       sharedv1.PrivacyClass
		consumerToken string
		authorization string
		consentReceipt string
		consentSource  string
		wantConsent    string
	}{
		{source: "local", profile: sharedv1.Profile_PROFILE_LOCAL_ONLY, privacy: webConsolePrivacyClass},
		{source: "own-key", profile: sharedv1.Profile_PROFILE_REMOTE_ONLY, privacy: webConsolePrivacyClass, consentReceipt: "receipt-own-key", consentSource: "own-key", wantConsent: "receipt-own-key"},
		{source: "own-key", profile: sharedv1.Profile_PROFILE_REMOTE_ONLY, privacy: webConsolePrivacyClass, consentReceipt: "receipt-credits", consentSource: "vrooli-credits"},
		{source: "vrooli-credits", profile: sharedv1.Profile_PROFILE_REMOTE_ONLY, privacy: webConsolePrivacyClass, consumerToken: "signed-access", authorization: "Bearer signed-access"},
		{source: "vrooli-credits", profile: sharedv1.Profile_PROFILE_REMOTE_ONLY, privacy: webConsolePrivacyClass, consumerToken: "Bearer signed-access", authorization: "Bearer signed-access"},
	}
	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {
			fixture := &gatewayInferenceFixture{}
			path, handler := inferenceconnect.NewInferenceServiceHandler(fixture)
			mux := http.NewServeMux()
			mux.Handle(path, handler)
			server := httptest.NewServer(mux)
			defer server.Close()

			provider := &GatewayProvider{
				Source: test.source, ProviderName: test.source,
				PrivacyClass:  test.privacy,
				PrivacySource: webConsolePrivacySource,
				ResolveURL:    func(context.Context) (string, error) { return server.URL, nil },
			}
			ctx := WithRootOperationID(WithPrivacyConsentGrant(WithConsumerToken(context.Background(), test.consumerToken), test.consentReceipt, test.consentSource), "web-console-operation-123456789012")
			got, err := provider.Generate(ctx, "Create a reversible command", "restart scenario foo")
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if got != "echo safe" {
				t.Fatalf("Generate = %q, want parsed command string", got)
			}
			if fixture.source != test.source {
				t.Fatalf("source header = %q, want %q", fixture.source, test.source)
			}
			if fixture.authorization != test.authorization {
				t.Fatalf("authorization header = %q, want %q", fixture.authorization, test.authorization)
			}
			if fixture.consent != test.wantConsent {
				t.Fatalf("consent grant header = %q, want %q", fixture.consent, test.wantConsent)
			}
			if fixture.request == nil || fixture.request.GetRole() != "author.generator" {
				t.Fatalf("task role = %v, want author.generator", fixture.request.GetRole())
			}
			if fixture.request.GetProfile() != test.profile {
				t.Fatalf("profile = %s, want %s", fixture.request.GetProfile(), test.profile)
			}
			if fixture.request.GetPrivacyClass() != test.privacy {
				t.Fatalf("privacy class = %s, want sourced class %s", fixture.request.GetPrivacyClass(), test.privacy)
			}
			if fixture.request.GetPrivacyProvenance() != webConsolePrivacySource {
				t.Fatalf("privacy provenance = %q, want Web Console user/session source", fixture.request.GetPrivacyProvenance())
			}
			if fixture.request.GetRootOperationId() != "web-console-operation-123456789012" {
				t.Fatalf("root operation ID = %q, want caller's retry identity", fixture.request.GetRootOperationId())
			}
		})
	}
}

func TestNewGatewayChainDeclaresAPrivacyBasisForEveryConsentSource(t *testing.T) {
	chain := NewGatewayChain()
	want := map[string]sharedv1.Profile{
		"ollama":     sharedv1.Profile_PROFILE_LOCAL_ONLY,
		"openrouter": sharedv1.Profile_PROFILE_REMOTE_ONLY,
		"vrooli":     sharedv1.Profile_PROFILE_REMOTE_ONLY,
	}
	for _, provider := range chain.Providers() {
		gateway, ok := provider.(*GatewayProvider)
		if !ok {
			t.Fatalf("provider %q = %T, want GatewayProvider", provider.Name(), provider)
		}
		if _, ok := want[provider.Name()]; !ok {
			t.Fatalf("unexpected provider %q", provider.Name())
		}
		if gateway.PrivacyClass != webConsolePrivacyClass || gateway.PrivacySource != webConsolePrivacySource {
			t.Fatalf("provider %q privacy contract = %s from %q", provider.Name(), gateway.PrivacyClass, gateway.PrivacySource)
		}
		profile, err := gatewaySourceProfile(gateway.Source)
		if err != nil || profile != want[provider.Name()] {
			t.Fatalf("provider %q profile = %s err=%v, want %s", provider.Name(), profile, err, want[provider.Name()])
		}
	}
	if len(chain.Providers()) != len(want) {
		t.Fatalf("providers = %d, want %d", len(chain.Providers()), len(want))
	}
}

func TestGatewayProviderRejectsUnknownSource(t *testing.T) {
	provider := &GatewayProvider{
		Source: "implicit-fallback", ProviderName: "unknown",
		ResolveURL: func(context.Context) (string, error) { return "http://127.0.0.1", nil },
	}
	if _, err := provider.Generate(context.Background(), "system", "user"); err == nil {
		t.Fatal("unknown source must not be assigned an inferred profile")
	}
}

func TestGatewayProviderRequiresPrivacyProvenanceBeforeResolving(t *testing.T) {
	provider := &GatewayProvider{
		Source: "own-key", ProviderName: "openrouter", PrivacyClass: webConsolePrivacyClass,
		ResolveURL: func(context.Context) (string, error) {
			t.Fatal("gateway should not be resolved without privacy provenance")
			return "", nil
		},
	}
	if _, err := provider.Generate(context.Background(), "system", "user"); err == nil {
		t.Fatal("missing privacy provenance must fail before remote inference")
	}
}

func TestGatewayProviderRequiresConsumerTokenForCreditsBeforeResolvingGateway(t *testing.T) {
	provider := &GatewayProvider{
		Source: "vrooli-credits", ProviderName: "vrooli", PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL,
		PrivacySource: webConsolePrivacySource,
		ResolveURL: func(context.Context) (string, error) {
			t.Fatal("gateway should not be resolved without a consumer token")
			return "", nil
		},
	}
	if _, err := provider.Generate(context.Background(), "system", "user"); err == nil || !errors.Is(err, ErrCreditsRequired) {
		t.Fatalf("missing subscription token error = %v, want ErrCreditsRequired", err)
	}
}

func TestGatewayProviderRefusesRemoteWithoutRecognizedPrivacyClassBeforeResolvingGateway(t *testing.T) {
	for _, privacy := range []sharedv1.PrivacyClass{
		sharedv1.PrivacyClass_PRIVACY_CLASS_UNSPECIFIED,
		sharedv1.PrivacyClass(999),
		sharedv1.PrivacyClass_PRIVACY_CLASS_SECRET,
	} {
		t.Run(privacy.String(), func(t *testing.T) {
			provider := &GatewayProvider{
				Source: "own-key", ProviderName: "openrouter", PrivacyClass: privacy,
				ResolveURL: func(context.Context) (string, error) {
					t.Fatal("gateway should not be resolved for ineligible remote privacy class")
					return "", nil
				},
			}
			if _, err := provider.Generate(context.Background(), "system", "user"); err == nil {
				t.Fatal("remote request with missing, unknown or secret privacy classification must fail")
			}
		})
	}
}
