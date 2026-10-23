package routing_test

import (
	"context"
	"testing"

	"ai-gateway/internal/providers"
	"ai-gateway/internal/providers/mocks"
	"ai-gateway/internal/routing"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"github.com/stretchr/testify/require"
)

// piiDetectRunner wires the pii-detector provider's policy + detect commands.
func piiDetectRunner() *mocks.FakeRunner {
	return &mocks.FakeRunner{
		Results: map[string]providers.Result{
			"resource-pii-detector policy roles --json": {
				Stdout: `{"roles":[{"schema_version":"2026-10-21","role":"pii.small","required_capabilities":["detect_pii"]}]}`,
			},
			"resource-pii-detector policy resolve --role pii.small --json": {
				Stdout: `{"role":"pii.small","model":"gravitee-io/bert-small-pii-detection","canonical_model":"gravitee-io/bert-small-pii-detection","capabilities":["detect_pii"]}`,
			},
			"resource-pii-detector gateway detect --text-stdin --json": {
				Stdout: `{"spans":[{"category":"EMAIL_ADDRESS","start":11,"end":26,"text":"bob@example.com","confidence":0.99,"source":"regex"}],"model_available":false}`,
			},
		},
		Errors: map[string]error{},
	}
}

func piiAdapters(runner providers.CommandRunner) []providers.Adapter {
	return []providers.Adapter{
		{Provider: providers.ProviderPiiDetector, CommandName: "resource-pii-detector", Locality: "local", Runner: runner},
	}
}

func piiRequest() *sharedv1.GatewayRequest {
	return &sharedv1.GatewayRequest{
		Kind:         sharedv1.RequestKind_REQUEST_KIND_DETECT_PII,
		Role:         "pii.small",
		Profile:      sharedv1.Profile_PROFILE_LOCAL_ONLY,
		PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
		Operation:    "scan",
		Scenario:     "fixture-scenario",
		RequestId:    "req-pii-1",
	}
}

// A successful DETECT_PII request sets prompt_redacted (a detect pass ran) while
// keeping response_redacted false (no response-redaction policy exists yet), and
// returns the detector's span JSON without leaking the input into evidence.
func TestExecuteDetectPIISetsPromptRedactedSeam(t *testing.T) {
	runner := piiDetectRunner()
	db := newSchemaDB(t)
	svc := routing.NewService(piiAdapters(runner), routing.NewSQLRepository(db))

	resp, err := svc.Execute(context.Background(), piiRequest(), "mail me at bob@example.com")
	require.NoError(t, err)
	require.True(t, resp.GetValid())
	require.Equal(t, "succeeded", resp.GetEvidence().GetStatus())
	require.Equal(t, "pii-detector", resp.GetEvidence().GetSelectedProvider())
	require.True(t, resp.GetEvidence().GetPromptRedacted(), "a successful detect pass must set prompt_redacted")
	require.False(t, resp.GetEvidence().GetResponseRedacted(), "response_redacted stays false until a response-redaction policy exists")
	require.Contains(t, resp.GetOutputText(), "EMAIL_ADDRESS")
	require.NotContains(t, resp.GetOutputText(), "mail me at")
}

// roleSupportsKind must admit the detector role for DETECT_PII and reject it for
// an unrelated kind, so the detector is routable only for its capability.
func TestDetectPIIRoleGating(t *testing.T) {
	runner := piiDetectRunner()
	db := newSchemaDB(t)
	svc := routing.NewService(piiAdapters(runner), routing.NewSQLRepository(db))

	// A TEXT_GENERATION request against the detector-only inventory has no
	// eligible provider and is blocked, proving the detector is not routed for a
	// kind it does not support.
	req := piiRequest()
	req.Kind = sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION
	req.Role = "chat.default"
	resp, err := svc.Execute(context.Background(), req, "hello")
	require.NoError(t, err)
	require.Equal(t, "blocked", resp.GetEvidence().GetStatus())
}
