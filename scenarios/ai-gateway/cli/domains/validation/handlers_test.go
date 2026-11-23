package validation

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

type validationCaptureClient struct {
	validationconnect.ScenarioValidationServiceClient
	validateReq  *v1.ValidateScenarioRequest
	fixReq       *v1.FixRequest
	validateResp *connect.Response[v1.ValidateScenarioResponse]
	providerResp *connect.Response[v1.DescribeProviderResponse]
	fixResp      *connect.Response[v1.FixResponse]
	previewCalls int
	applyCalls   int
	err          error
}

func (c *validationCaptureClient) ValidateScenario(_ context.Context, req *connect.Request[v1.ValidateScenarioRequest]) (*connect.Response[v1.ValidateScenarioResponse], error) {
	c.validateReq = req.Msg
	return c.validateResp, c.err
}
func (c *validationCaptureClient) DescribeProvider(_ context.Context, _ *connect.Request[v1.DescribeProviderRequest]) (*connect.Response[v1.DescribeProviderResponse], error) {
	return c.providerResp, c.err
}
func (c *validationCaptureClient) PreviewFix(_ context.Context, req *connect.Request[v1.FixRequest]) (*connect.Response[v1.FixResponse], error) {
	c.previewCalls++
	c.fixReq = req.Msg
	return c.fixResp, c.err
}
func (c *validationCaptureClient) ApplyFix(_ context.Context, req *connect.Request[v1.FixRequest]) (*connect.Response[v1.FixResponse], error) {
	c.applyCalls++
	c.fixReq = req.Msg
	return c.fixResp, c.err
}

func validationContext(flags map[string]string, boolFlags map[string]bool, jsonOutput bool) (cliapp.RunContext, *bytes.Buffer) {
	flagDefs := []cliapp.Flag{{Name: "scenario"}, {Name: "path"}, {Name: "rule-ids"}, {Name: "include-execution"}}
	buf := &bytes.Buffer{}
	return cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cliapp.ArgSchema{Flags: flagDefs}, Flags: flags, BoolFlags: boolFlags, JSON: jsonOutput, Stdout: buf}), buf
}

func TestValidateMapsRequestAndRendersServerResponse(t *testing.T) {
	ctx, output := validationContext(map[string]string{"scenario": "demo", "path": "src", "rule-ids": ""}, map[string]bool{"include-execution": true}, true)
	client := &validationCaptureClient{validateResp: connect.NewResponse(&v1.ValidateScenarioResponse{Scenario: "demo", Status: v1.ValidationStatus_VALIDATION_STATUS_PASSED})}
	require.NoError(t, (&handlers{client: client}).validate(ctx))
	require.Equal(t, "demo", client.validateReq.GetScenario())
	require.Equal(t, "src", client.validateReq.GetPath())
	require.True(t, client.validateReq.GetIncludeExecution())
	require.JSONEq(t, `{"scenario":"demo","status":"VALIDATION_STATUS_PASSED"}`, output.String())
}

func TestValidatePreservesEmptyOptionalSelectorsAndFalseExecution(t *testing.T) {
	ctx, output := validationContext(map[string]string{"scenario": "demo", "path": ""}, nil, false)
	client := &validationCaptureClient{validateResp: connect.NewResponse(&v1.ValidateScenarioResponse{Scenario: "demo", Status: v1.ValidationStatus_VALIDATION_STATUS_PASSED})}
	require.NoError(t, (&handlers{client: client}).validate(ctx))
	require.Equal(t, "demo", client.validateReq.GetScenario())
	require.Empty(t, client.validateReq.GetPath())
	require.False(t, client.validateReq.GetIncludeExecution())
	require.Contains(t, output.String(), "Shared AI conformance validation for demo")
	require.Contains(t, output.String(), "status=VALIDATION_STATUS_PASSED")
}

func TestValidateRendersAssessmentAndFindingsSummary(t *testing.T) {
	ctx, output := validationContext(map[string]string{"scenario": "demo"}, nil, false)
	client := &validationCaptureClient{validateResp: connect.NewResponse(&v1.ValidateScenarioResponse{
		Scenario: "demo", Status: v1.ValidationStatus_VALIDATION_STATUS_PASSED,
		Assessment: &commonv1.MaturityAssessment{Phase: "conformance", Local: &commonv1.LocalMaturityAssessment{CurrentLevel: "L2"}, Findings: []*commonv1.AssessmentFinding{{Code: "F1"}}},
	})}
	require.NoError(t, (&handlers{client: client}).validate(ctx))
	require.Contains(t, output.String(), "assessment=conformance level=L2 findings=1")
}

func TestDescribeProviderRendersAvailableCapabilities(t *testing.T) {
	ctx, output := validationContext(nil, nil, false)
	client := &validationCaptureClient{providerResp: connect.NewResponse(&v1.DescribeProviderResponse{
		Provider: "ai-gateway", Phase: "conformance", Contract: "scenario-validation/v1",
		Capabilities: &v1.ProviderCapabilities{SupportsExecution: true, SupportsFixes: true, DeliveryMode: "inline"},
	})}
	require.NoError(t, (&handlers{client: client}).describeProvider(ctx))
	require.Contains(t, output.String(), "AI Gateway validation provider descriptor.")
	require.Contains(t, output.String(), "provider=ai-gateway phase=conformance contract=scenario-validation/v1")
	require.Contains(t, output.String(), "supports_execution=true supports_fixes=true delivery=inline")
}

func TestDescribeProviderAllowsCapabilitiesToBeUnspecified(t *testing.T) {
	ctx, output := validationContext(nil, nil, false)
	client := &validationCaptureClient{providerResp: connect.NewResponse(&v1.DescribeProviderResponse{Provider: "ai-gateway"})}
	require.NoError(t, (&handlers{client: client}).describeProvider(ctx))
	require.Contains(t, output.String(), "provider=ai-gateway phase= contract=")
	require.NotContains(t, output.String(), "supports_execution=")
}

func TestFixPreservesEmptySelectorsAndServerMessage(t *testing.T) {
	ctx, output := validationContext(map[string]string{"scenario": "demo", "path": "", "rule-ids": " , "}, nil, false)
	client := &validationCaptureClient{fixResp: connect.NewResponse(&v1.FixResponse{Scenario: "demo", Messages: []string{"No candidates are available for this scenario."}})}
	require.NoError(t, (&handlers{client: client}).previewFix(ctx))
	require.Equal(t, "demo", client.fixReq.GetScenario())
	require.Empty(t, client.fixReq.GetPath())
	require.Empty(t, client.fixReq.GetRuleIds())
	require.Equal(t, 1, client.previewCalls)
	require.Zero(t, client.applyCalls)
	require.Contains(t, output.String(), "No candidates are available for this scenario.")
	require.NotContains(t, output.String(), "rule-1")
}

func TestApplyFixUsesApplyRPCAndMapsRuleSelectors(t *testing.T) {
	ctx, output := validationContext(map[string]string{"scenario": "demo", "path": "pkg", "rule-ids": " R1, ,R2 "}, nil, false)
	client := &validationCaptureClient{fixResp: connect.NewResponse(&v1.FixResponse{Scenario: "demo", Applied: false, Messages: []string{"AI conformance apply is intentionally a no-op until safe deterministic migrations exist."}})}
	require.NoError(t, (&handlers{client: client}).applyFix(ctx))
	require.Equal(t, 1, client.applyCalls)
	require.Zero(t, client.previewCalls)
	require.Equal(t, "demo", client.fixReq.GetScenario())
	require.Equal(t, "pkg", client.fixReq.GetPath())
	require.Equal(t, []string{"R1", "R2"}, client.fixReq.GetRuleIds())
	require.Contains(t, output.String(), "AI conformance fix applied=false.")
	require.Contains(t, output.String(), "AI conformance apply is intentionally a no-op until safe deterministic migrations exist.")
	require.NotContains(t, output.String(), "applied=true")
}

func TestPreviewFixRendersEachServerCandidate(t *testing.T) {
	ctx, output := validationContext(map[string]string{"scenario": "demo", "rule-ids": "R7"}, nil, false)
	client := &validationCaptureClient{fixResp: connect.NewResponse(&v1.FixResponse{Candidates: []*v1.FixCandidate{{
		RuleId: "R7", FilePath: "config.yaml", Applied: false, Description: "add required setting",
	}}})}
	require.NoError(t, (&handlers{client: client}).previewFix(ctx))
	require.Contains(t, output.String(), "AI conformance fix applied=false.")
	require.Contains(t, output.String(), "R7 config.yaml applied=false: add required setting")
}

func TestValidationHandlersMapDependencyAndEmptyResponseErrors(t *testing.T) {
	t.Run("dependency error", func(t *testing.T) {
		ctx, output := validationContext(map[string]string{}, nil, true)
		client := &validationCaptureClient{err: errors.New("backend unavailable")}
		err := (&handlers{client: client}).validate(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "backend unavailable")
		require.Empty(t, output.String())
	})
	t.Run("fix dependency error", func(t *testing.T) {
		ctx, output := validationContext(map[string]string{"scenario": "demo"}, nil, true)
		client := &validationCaptureClient{err: errors.New("fix service unavailable")}
		err := (&handlers{client: client}).previewFix(ctx)
		require.ErrorContains(t, err, "AI conformance fix")
		require.ErrorContains(t, err, "fix service unavailable")
		require.Equal(t, 1, client.previewCalls)
		require.Zero(t, client.applyCalls)
		require.Empty(t, output.String())
	})
	t.Run("apply dependency error", func(t *testing.T) {
		ctx, output := validationContext(map[string]string{"scenario": "demo"}, nil, true)
		client := &validationCaptureClient{err: errors.New("apply service unavailable")}
		err := (&handlers{client: client}).applyFix(ctx)
		require.ErrorContains(t, err, "AI conformance fix")
		require.ErrorContains(t, err, "apply service unavailable")
		require.Equal(t, 1, client.applyCalls)
		require.Zero(t, client.previewCalls)
		require.Empty(t, output.String())
	})
	t.Run("provider descriptor dependency error", func(t *testing.T) {
		ctx, output := validationContext(nil, nil, true)
		client := &validationCaptureClient{err: errors.New("descriptor service unavailable")}
		err := (&handlers{client: client}).describeProvider(ctx)
		require.ErrorContains(t, err, "describe AI conformance provider")
		require.ErrorContains(t, err, "descriptor service unavailable")
		require.Empty(t, output.String())
	})
	for _, tc := range []struct {
		name string
		call func(*handlers, cliapp.RunContext) error
		want string
	}{
		{"nil validation response", (*handlers).validate, "server returned no validation response"},
		{"nil provider response", (*handlers).describeProvider, "server returned no provider descriptor"},
		{"nil fix response", (*handlers).previewFix, "server returned no fix response"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := validationContext(map[string]string{}, nil, true)
			require.EqualError(t, tc.call(&handlers{client: &validationCaptureClient{}}, ctx), tc.want)
		})
	}
}

func TestValidationHandlersRejectResponsesWithoutMessages(t *testing.T) {
	ctx, _ := validationContext(map[string]string{}, nil, true)
	client := &validationCaptureClient{
		validateResp: &connect.Response[v1.ValidateScenarioResponse]{},
		providerResp: &connect.Response[v1.DescribeProviderResponse]{},
		fixResp:      &connect.Response[v1.FixResponse]{},
	}
	h := &handlers{client: client}
	require.EqualError(t, h.validate(ctx), "server returned no validation response")
	require.EqualError(t, h.describeProvider(ctx), "server returned no provider descriptor")
	require.EqualError(t, h.previewFix(ctx), "server returned no fix response")
}

func TestSplitCSVTrimsAndOmitsEmptySelectors(t *testing.T) {
	require.Empty(t, splitCSV("  ,  "))
	require.Equal(t, []string{"one", "two"}, splitCSV(" one, ,two "))
}
