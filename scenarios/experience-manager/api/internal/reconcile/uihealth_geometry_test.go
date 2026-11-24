package reconcile

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"experience-manager/internal/spec"
	"github.com/stretchr/testify/require"
	visualpb "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
	visualconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth/visualhealth_v1connect"
)

type fakeVisualHealthAnalyzer struct {
	request  *visualpb.AnalyzeArtifactsRequest
	response *visualpb.AnalyzeArtifactsResponse
	err      error
}

type fakeVisualHealthService struct {
	fakeVisualHealthAnalyzer
}

func (f *fakeVisualHealthService) CompareArtifacts(context.Context, *connect.Request[visualpb.CompareArtifactsRequest]) (*connect.Response[visualpb.CompareArtifactsResponse], error) {
	return connect.NewResponse(&visualpb.CompareArtifactsResponse{}), nil
}

func (f *fakeVisualHealthService) ListRules(context.Context, *connect.Request[visualpb.ListRulesRequest]) (*connect.Response[visualpb.ListRulesResponse], error) {
	return connect.NewResponse(&visualpb.ListRulesResponse{}), nil
}

func (f *fakeVisualHealthService) RecordBrowserObservation(_ context.Context, req *connect.Request[visualpb.RecordBrowserObservationRequest]) (*connect.Response[visualpb.RecordBrowserObservationResponse], error) {
	return connect.NewResponse(&visualpb.RecordBrowserObservationResponse{Observation: req.Msg.GetObservation()}), nil
}

func (f *fakeVisualHealthService) ListBrowserObservations(context.Context, *connect.Request[visualpb.ListBrowserObservationsRequest]) (*connect.Response[visualpb.ListBrowserObservationsResponse], error) {
	return connect.NewResponse(&visualpb.ListBrowserObservationsResponse{}), nil
}

type fixedGeometryEvaluator struct {
	findings []spec.Finding
	err      error
	calls    int
}

func (f *fixedGeometryEvaluator) EvaluateGeometry(context.Context, string, CaptureTarget, Snapshot) ([]spec.Finding, error) {
	f.calls++
	return f.findings, f.err
}

func (f *fakeVisualHealthAnalyzer) AnalyzeArtifacts(_ context.Context, req *connect.Request[visualpb.AnalyzeArtifactsRequest]) (*connect.Response[visualpb.AnalyzeArtifactsResponse], error) {
	f.request = req.Msg
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(f.response), nil
}

func TestVisualStepFromAccessibilitySnapshotUsesSharedArtifactShape(t *testing.T) {
	target := CaptureTarget{Scenario: "demo", Route: "/", PageID: "home", StateID: "default", ViewportID: "mobile", ViewportWidth: 390, ViewportHeight: 844}
	snapshot := Snapshot{
		Contract: snapshotContract,
		URL:      "http://demo/",
		Root: AXNode{
			Role: "document", DOM: DOMNode{Tag: "body"}, Bounds: &Bounds{Width: 390, Height: 844},
			ComputedStyle: map[string]string{"scrollWidth": "404"},
			Children: []AXNode{{
				Role: "button", Name: "Save", DOM: DOMNode{TestID: "save\"button", Tag: "button"},
				Bounds:        &Bounds{X: 8, Y: 12, Width: 36, Height: 40},
				ComputedStyle: map[string]string{"overflow-x": "hidden", "font-size": "14px"},
			}},
		},
	}

	step, err := visualStepFromAccessibilitySnapshot(target, snapshot)
	require.NoError(t, err)
	require.Equal(t, "http://demo/", step.GetUrl())
	require.EqualValues(t, 390, step.GetViewport().GetWidth())
	var layout map[string]any
	require.NoError(t, json.Unmarshal([]byte(step.GetLayoutJson()), &layout))
	require.Equal(t, float64(404), layout["document"].(map[string]any)["scrollWidth"])
	require.Equal(t, "app-chrome-controls", layout["geometryPolicy"].(map[string]any)["unsafeEdgeScope"])
	require.Equal(t, "application-chrome", layout["geometryPolicy"].(map[string]any)["chromeScope"])
	require.Equal(t, true, layout["geometryPolicy"].(map[string]any)["touchTargetsViewportOnly"])
	elements := layout["elements"].([]any)
	require.Len(t, elements, 2)
	button := elements[1].(map[string]any)
	require.Equal(t, true, button["interactive"])
	require.Equal(t, true, button["surfaceRoot"] == false)
	require.Equal(t, "save\"button", button["testId"])
	require.Equal(t, float64(36), button["rect"].(map[string]any)["width"])
	require.Equal(t, "hidden", button["computed"].(map[string]any)["overflowX"])
}

func TestVisualStepConversionCarriesNestedClippingEvidence(t *testing.T) {
	target := CaptureTarget{ViewportWidth: 390, ViewportHeight: 844, ViewportID: "mobile"}
	snapshot := Snapshot{Contract: snapshotContract, Root: AXNode{
		Role: "document", DOM: DOMNode{Tag: "body"}, Bounds: &Bounds{Width: 390, Height: 844},
		Children: []AXNode{{
			Role: "list", Bounds: &Bounds{X: 0, Y: 200, Width: 390, Height: 56},
			ComputedStyle: map[string]string{"overflow-x": "auto"}, Children: []AXNode{{
				Role: "listitem", Name: "Off-canvas chip", Bounds: &Bounds{X: 324, Y: 212, Width: 99, Height: 44},
			}},
		}},
	}}
	step, err := visualStepFromAccessibilitySnapshot(target, snapshot)
	require.NoError(t, err)
	var layout map[string]any
	require.NoError(t, json.Unmarshal([]byte(step.GetLayoutJson()), &layout))
	elements := layout["elements"].([]any)
	chip := elements[2].(map[string]any)
	require.Equal(t, true, chip["inScrollContainer"], "the clipping-aware migration must carry reachability to the canonical evaluator")
}

func TestVisualStepConversionPreservesClippingFloorCases(t *testing.T) {
	target := CaptureTarget{ViewportWidth: 390, ViewportHeight: 844, ViewportID: "mobile"}
	inside := Bounds{X: 0, Y: 200, Width: 390, Height: 56}
	cases := []struct {
		name                string
		style               map[string]string
		width               float64
		nested              bool
		nestedAncestorWidth float64
		nestedAncestorStyle map[string]string
		wantClip            bool
	}{
		{name: "auto", style: map[string]string{"overflow-x": "auto"}, width: 390, wantClip: true},
		{name: "scroll", style: map[string]string{"overflow-x": "scroll"}, width: 390, wantClip: true},
		{name: "hidden", style: map[string]string{"overflow-x": "hidden"}, width: 390, wantClip: true},
		{name: "clip shorthand", style: map[string]string{"overflow": "auto hidden"}, width: 390, wantClip: true},
		{name: "deep descendant", style: map[string]string{"overflow-x": "auto"}, width: 390, nested: true, wantClip: true},
		{name: "wide inner clipper remains reachable through outer scroller", style: map[string]string{"overflow-x": "auto"}, width: 390, nested: true, nestedAncestorWidth: 700, nestedAncestorStyle: map[string]string{"overflow-x": "hidden"}, wantClip: true},
		{name: "visible does not clip", style: map[string]string{"overflow-x": "visible"}, width: 390},
		{name: "missing evidence does not clip", width: 390},
		{name: "outside ancestor cannot excuse", style: map[string]string{"overflow-x": "auto"}, width: 700},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			chip := AXNode{Role: "listitem", Name: "Dismissed", DOM: DOMNode{TestID: "chip"}, Bounds: &Bounds{X: 324, Y: 212, Width: 99, Height: 44}}
			if tc.nested {
				width := tc.nestedAncestorWidth
				if width == 0 {
					width = 600
				}
				chip = AXNode{Role: "group", Bounds: &Bounds{X: 16, Y: 200, Width: width, Height: 56}, ComputedStyle: tc.nestedAncestorStyle, Children: []AXNode{chip}}
			}
			snapshot := Snapshot{Contract: snapshotContract, Root: AXNode{
				Role: "document", DOM: DOMNode{Tag: "body"}, Bounds: &Bounds{Width: 390, Height: 844},
				Children: []AXNode{{Role: "list", Bounds: &inside, ComputedStyle: tc.style, Children: []AXNode{chip}}},
			}}
			snapshot.Root.Children[0].Bounds.Width = tc.width
			step, err := visualStepFromAccessibilitySnapshot(target, snapshot)
			require.NoError(t, err)
			var layout map[string]any
			require.NoError(t, json.Unmarshal([]byte(step.GetLayoutJson()), &layout))
			var got bool
			for _, raw := range layout["elements"].([]any) {
				element := raw.(map[string]any)
				if element["testId"] == "chip" {
					got, _ = element["inScrollContainer"].(bool)
				}
			}
			require.Equal(t, tc.wantClip, got)
		})
	}
}

func TestUIHealthGeometryEvaluatorUsesCanonicalFindingEnvelope(t *testing.T) {
	client := &fakeVisualHealthAnalyzer{response: &visualpb.AnalyzeArtifactsResponse{Findings: []*visualpb.VisualFinding{
		{Code: "visual_touch_target_too_small", Severity: visualpb.VisualSeverity_VISUAL_SEVERITY_ERROR, Message: "button is too small", Location: "[data-testid=save]", Evidence: "36x40px", Remediation: "Expand it."},
		{Code: "visual_vision_opinion", Severity: visualpb.VisualSeverity_VISUAL_SEVERITY_ERROR, Message: "advisory finding"},
	}}}
	evaluator := UIHealthGeometryEvaluator{analyze: client}
	target := CaptureTarget{PageID: "home", Route: "/", StateID: "default", ViewportID: "mobile", ViewportWidth: 390, ViewportHeight: 844}
	findings, err := evaluator.EvaluateGeometry(context.Background(), "demo", target, Snapshot{Contract: snapshotContract, Root: AXNode{Role: "document", Bounds: &Bounds{Width: 390, Height: 844}}})
	require.NoError(t, err)
	require.Len(t, findings, 1)
	require.Equal(t, "visual_touch_target_too_small", findings[0].Code)
	require.Equal(t, spec.SeverityError, findings[0].Severity)
	require.Equal(t, "button is too small: 36x40px", findings[0].Message)
	require.Equal(t, []string{"[data-testid=save]"}, findings[0].Locations)
	require.Contains(t, client.request.GetSteps()[0].GetLayoutJson(), "surfaceRoot")
}

func TestUIHealthGeometryEvaluatorPreservesServiceFailure(t *testing.T) {
	client := &fakeVisualHealthAnalyzer{err: errors.New("ui-health unavailable")}
	evaluator := UIHealthGeometryEvaluator{analyze: client}
	target := CaptureTarget{PageID: "home", StateID: "default", ViewportID: "mobile", ViewportWidth: 390, ViewportHeight: 844}
	_, err := evaluator.EvaluateGeometry(context.Background(), "demo", target, Snapshot{Contract: snapshotContract, Root: AXNode{Bounds: &Bounds{Width: 390, Height: 844}}})
	require.ErrorContains(t, err, "ui-health geometry analysis: ui-health unavailable")
}

func TestUIHealthGeometryEvaluatorUsesConnectServiceEndpoint(t *testing.T) {
	service := &fakeVisualHealthService{fakeVisualHealthAnalyzer: fakeVisualHealthAnalyzer{response: &visualpb.AnalyzeArtifactsResponse{Findings: []*visualpb.VisualFinding{{
		Code: "visual_viewport_overflow", Severity: visualpb.VisualSeverity_VISUAL_SEVERITY_ERROR, Message: "page is too wide",
	}}}}}
	path, handler := visualconnect.NewVisualHealthServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	evaluator := &UIHealthGeometryEvaluator{
		resolveBaseURL: func(context.Context) (string, error) { return server.URL, nil },
		httpClient:     server.Client(),
	}
	target := CaptureTarget{PageID: "home", Route: "/", StateID: "default", ViewportID: "mobile", ViewportWidth: 390, ViewportHeight: 844}
	findings, err := evaluator.EvaluateGeometry(context.Background(), "demo", target, Snapshot{Contract: snapshotContract, Root: AXNode{Role: "document", Bounds: &Bounds{Width: 390, Height: 844}}})
	require.NoError(t, err)
	require.Len(t, findings, 1)
	require.Equal(t, "visual_viewport_overflow", findings[0].Code)
	require.Equal(t, "demo", service.request.GetScenario())
	require.Equal(t, "home::/:default:mobile::", service.request.GetRunId())
}

func TestReconcileCapturesDelegatesPageGeometryToExperience(t *testing.T) {
	evaluator := &fixedGeometryEvaluator{findings: []spec.Finding{{Code: "visual_touch_target_too_small", Severity: spec.SeverityError, Message: "button is 36px"}}}
	check := Check{Capturer: fakeCapturer{snapshot: passingSnapshot()}, GeometryEvaluator: evaluator, CaptureProfiles: testProfiles()}
	findings := check.Run(context.Background(), activeReport("primary", spec.Binding{TestID: "primary-action"}))
	require.Zero(t, evaluator.calls, "page geometry belongs to canonical experience")
	require.NotContains(t, findings, evaluator.findings[0])
}

func TestReconcileCapturesRetainsComponentGeometryEvaluation(t *testing.T) {
	evaluator := &fixedGeometryEvaluator{findings: []spec.Finding{{Code: "visual_touch_target_too_small", Severity: spec.SeverityError, Message: "button is 36px"}}}
	check := Check{Capturer: fakeCapturer{snapshot: passingSnapshot()}, GeometryEvaluator: evaluator, CaptureProfiles: testProfiles()}
	report := activeComponentReport("primary", spec.Binding{TestID: "primary-action"})
	component := report.Spec.Components["button"]
	component.FloorOptOuts = nil
	report.Spec.Components["button"] = component
	findings := check.Run(context.Background(), report)
	require.Equal(t, 1, evaluator.calls)
	require.Contains(t, findings, evaluator.findings[0])
}

func TestCanonicalExperienceOwnsPageGeometryAndComponentEvaluatorHonorsOptOut(t *testing.T) {
	page := spec.PageDocument{
		Kind: "page", Archetype: "page", Page: spec.PageIdentity{ID: "home", Routes: []string{"/"}},
		Claims: []spec.Claim{{ID: "overflow", Type: "no-document-horizontal-overflow", Tier: "machine"}},
	}
	target := CaptureTarget{Scenario: "demo", PageID: "home", Route: "/", StateID: "default", ViewportID: "desktop", ViewportWidth: 1280, ViewportHeight: 720}
	snapshot := passingSnapshot()
	overflow := []spec.Finding{{Code: "visual_viewport_overflow", Severity: spec.SeverityError, Message: "document exceeds viewport"}}
	result := reconcileActivePageWithGeometry("experience/pages/home.json", page, target, snapshot, overflow)
	var failedFloor *Evidence
	for index := range result.Evidence {
		if result.Evidence[index].ClaimType == "no-document-horizontal-overflow" {
			failedFloor = &result.Evidence[index]
			break
		}
	}
	require.NotNil(t, failedFloor)
	require.Equal(t, "skipped", failedFloor.Verdict)
	require.Contains(t, failedFloor.Message, "canonical experience phase")
	for _, finding := range result.Findings {
		require.NotContains(t, finding.Code, "experience.floor_", "retired EXM floor findings must not duplicate ui-health")
	}
	require.NotContains(t, result.Findings, overflow[0], "EXM does not evaluate page geometry")

	filtered := filterOptedOutGeometry(overflow, []spec.FloorOptOut{{Floor: "no-document-horizontal-overflow", Reason: "This route intentionally has a wide embedded canvas."}})
	require.Empty(t, filtered, "an explicit floor waiver must suppress the corresponding shared result")
	waived := reconcileActivePageWithGeometry("experience/pages/home.json", page, target, snapshot, filtered)
	for _, evidence := range waived.Evidence {
		if evidence.ClaimType == "no-document-horizontal-overflow" {
			require.Equal(t, "skipped", evidence.Verdict, "the opted-out claim remains delegated")
		}
	}
	componentPage := page
	componentPage.Kind = "experience-component"
	claimEvidence, passed := evaluateGeometryFloorClaim(componentPage, spec.Claim{ID: "overflow", Type: "no-document-horizontal-overflow"}, target, snapshot.Flatten(), []spec.Finding{{Code: spec.CodeClaimUnverifiable, Severity: spec.SeverityInfo, Message: "shared geometry service unavailable"}})
	require.False(t, passed)
	require.Equal(t, "unverifiable", claimEvidence.Verdict, "component geometry outages cannot become passes")
}

func TestRetiredBaselineFloorMappingsPreserveCanonicalGeometryParity(t *testing.T) {
	page := spec.PageDocument{Page: spec.PageIdentity{ID: "home"}}
	target := CaptureTarget{Route: "/", StateID: "default", ViewportID: "mobile", ViewportWidth: 390, ViewportHeight: 844}
	cases := []struct {
		floor string
		rule  string
	}{
		{floor: "no-document-horizontal-overflow", rule: "visual_viewport_overflow"},
		{floor: "viewport-fill", rule: "visual_surface_does_not_fill_viewport"},
		{floor: "chrome-pinned", rule: "visual_chrome_outside_viewport"},
		{floor: "safe-area-tap-targets", rule: "visual_unsafe_edge_tap_zone"},
		{floor: "single-line-chrome", rule: "visual_multiline_chrome_label"},
		{floor: "tap-target-size", rule: "visual_touch_target_too_small"},
	}
	for _, tc := range cases {
		t.Run(tc.floor, func(t *testing.T) {
			mappedRule, ok := baselineGeometryFloor(tc.floor)
			require.True(t, ok, "every authored baseline floor has one canonical mapping")
			require.Equal(t, tc.rule, mappedRule)
			require.Equal(t, tc.rule, geometryRuleForFloor(tc.floor))
			require.True(t, isBaselineFloorType(tc.floor))
			claim := spec.Claim{ID: "legacy-" + tc.floor, Type: tc.floor}
			canonicalFailure := []spec.Finding{{Code: tc.rule, Severity: spec.SeverityError, Message: "canonical geometry detected the legacy floor defect"}}
			failed, passed := evaluateGeometryFloorClaim(page, claim, target, nil, canonicalFailure)
			require.False(t, passed)
			require.Equal(t, "failed", failed.Verdict)
			require.Contains(t, failed.Message, "canonical geometry detected")

			clean, passed := evaluateGeometryFloorClaim(page, claim, target, nil, []spec.Finding{})
			require.True(t, passed)
			require.Equal(t, "passed", clean.Verdict)

			unmeasured, passed := evaluateGeometryFloorClaim(page, claim, target, nil, []spec.Finding{{
				Code: spec.CodeClaimUnverifiable, Severity: spec.SeverityInfo, Message: "canonical geometry evidence was unavailable",
			}})
			require.False(t, passed)
			require.Equal(t, "unverifiable", unmeasured.Verdict)

			wrongRule, passed := evaluateGeometryFloorClaim(page, claim, target, nil, []spec.Finding{{
				Code: "visual_vision_opinion", Severity: spec.SeverityError, Message: "unrelated advisory finding",
			}})
			require.True(t, passed, "an unrelated rule must not fail a legacy floor claim")
			require.Equal(t, "passed", wrongRule.Verdict)
		})
	}
	if _, ok := baselineGeometryFloor("unknown-floor"); ok || isBaselineFloorType("unknown-floor") || geometryRuleForFloor("unknown-floor") != "" {
		t.Fatal("unknown floors must not be admitted into the canonical geometry pack")
	}
}

func countFindingCode(findings []spec.Finding, code string) int {
	count := 0
	for _, finding := range findings {
		if finding.Code == code {
			count++
		}
	}
	return count
}

func TestReconcileSurfacesUnavailableSharedGeometryAsUnmeasured(t *testing.T) {
	evaluator := &fixedGeometryEvaluator{err: errors.New("ui-health unavailable")}
	check := Check{Capturer: fakeCapturer{snapshot: passingSnapshot()}, GeometryEvaluator: evaluator, CaptureProfiles: testProfiles()}
	report := activeComponentReport("primary", spec.Binding{TestID: "primary-action"})
	component := report.Spec.Components["button"]
	component.FloorOptOuts = nil
	report.Spec.Components["button"] = component
	findings := check.Run(context.Background(), report)
	require.Equal(t, 1, evaluator.calls)
	var unavailable bool
	for _, finding := range findings {
		if finding.Code == spec.CodeClaimUnverifiable && finding.Severity == spec.SeverityInfo && strings.Contains(finding.Message, "shared geometry evaluation was not completed") {
			unavailable = true
		}
	}
	require.True(t, unavailable, "shared evaluator outage must be visible as unmeasured geometry")
}
