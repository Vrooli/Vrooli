package monetization

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

// This package is intentionally a small, static provider. It owns the
// trust-boundary checks that do not require a live payment database.
type Handler struct{}

type declaration struct {
	Version   int       `json:"version"`
	BundleKey string    `json:"bundle_key"`
	Features  []surface `json:"features"`
	Meters    []surface `json:"meters"`
}

type surface struct {
	Key              string   `json:"key"`
	LimitKey         string   `json:"limit_key"`
	Class            string   `json:"class"`
	EnforcementPaths []string `json:"enforcement_paths"`
}

func RegisterRoutes(router *mux.Router) {
	path, handler := scenariovalidationconnect.NewScenarioValidationServiceHandler(Handler{})
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}

func (Handler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	path := req.Msg.GetPath()
	if path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario path is required"))
	}
	findings := scan(path)
	assessment := buildAssessment(req.Msg.GetScenario(), findings)
	native, _ := anypb.New(nativeStruct(req.Msg.GetScenario(), findings))
	status := scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED
	if len(findings) > 0 {
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED
	}
	return connect.NewResponse(&scenariovalidationv1.ValidateScenarioResponse{Scenario: req.Msg.GetScenario(), Status: status, Assessment: assessment, NativeDetail: native, FailureClassification: failureClass(status)}), nil
}

func (h Handler) ValidateTarget(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateTargetRequest]) (*connect.Response[scenariovalidationv1.ValidateTargetResponse], error) {
	if req.Msg.GetTarget() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("target is required"))
	}
	path := req.Msg.GetPath()
	if path == "" {
		path = req.Msg.GetTarget().GetRoot()
	}
	findings := scan(path)
	status := scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED
	if len(findings) > 0 {
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED
	}
	assessment := buildAssessment(req.Msg.GetTarget().GetId(), findings)
	native, _ := anypb.New(nativeStruct(req.Msg.GetTarget().GetId(), findings))
	return connect.NewResponse(&scenariovalidationv1.ValidateTargetResponse{Target: req.Msg.GetTarget(), Status: status, Assessment: assessment, NativeDetail: native, FailureClassification: failureClass(status)}), nil
}

func (Handler) DescribeProvider(context.Context, *connect.Request[scenariovalidationv1.DescribeProviderRequest]) (*connect.Response[scenariovalidationv1.DescribeProviderResponse], error) {
	return connect.NewResponse(&scenariovalidationv1.DescribeProviderResponse{
		Provider: "landing-page-business-suite", Phase: "monetization-conformance", SpecVersion: "1.0.0", Contract: "scenario-validation/v1",
		Capabilities: &scenariovalidationv1.ProviderCapabilities{DeliveryMode: "inline", SupportsExecution: false, SupportsFixes: false, TargetKinds: []commonv1.ValidationTargetKind{commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO}},
	}), nil
}

func (Handler) PreviewFix(context.Context, *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return connect.NewResponse(&scenariovalidationv1.FixResponse{Applied: false, Messages: []string{"Monetization conformance is detection-only; apply the remediation documented by the finding."}}), nil
}

func (Handler) ApplyFix(context.Context, *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return connect.NewResponse(&scenariovalidationv1.FixResponse{Applied: false, Messages: []string{"Monetization conformance does not perform source rewrites."}}), nil
}

func nativeStruct(scenario string, findings []*commonv1.AssessmentFinding) *structpb.Struct {
	items := make([]any, 0, len(findings))
	for _, f := range findings {
		items = append(items, map[string]any{"code": f.Code, "severity": f.Severity, "location": f.Location, "message": f.Message})
	}
	value, _ := structpb.NewStruct(map[string]any{"scenario": scenario, "findings": items})
	return value
}

func scan(root string) []*commonv1.AssessmentFinding {
	manifest := filepath.Join(root, ".vrooli", "monetization.json")
	var findings []*commonv1.AssessmentFinding
	var declarationData declaration
	manifestBytes, manifestErr := os.ReadFile(manifest)
	if manifestErr != nil {
		findings = append(findings, finding("money.undeclared_monetization", "Monetization is undeclared", manifest, "Add .vrooli/monetization.json using the governed schema."))
	} else if err := json.Unmarshal(manifestBytes, &declarationData); err != nil {
		findings = append(findings, finding("money.undeclared_monetization", "Monetization declaration is invalid", manifest, "Fix .vrooli/monetization.json to conform to the governed schema."))
	} else {
		findings = append(findings, validateDeclaration(root, manifest, declarationData)...)
	}
	findings = append(findings, scanServiceSecrets(root)...)
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		source := string(data)
		if strings.Contains(source, "LPBS_SERVICE_SECRET") || strings.Contains(source, "BAS_LPBS_SERVICE_SECRET") || strings.Contains(source, "ValidateServiceToken") {
			findings = append(findings, finding("money.service_token_in_client_bundle", "Shared service credential remains in client source", path, "Use the signed-in user's short-lived consumer access token."))
		}
		if strings.Contains(source, "user_identity") && (strings.Contains(source, "HandleReportUsage") || strings.Contains(source, "ReportUsageRequest")) && strings.Contains(source, "json") {
			findings = append(findings, finding("money.identity_from_request_body", "Usage identity is accepted from request data", path, "Derive identity from verified access-token claims and overwrite request data."))
		}
		if strings.Contains(source, "HS256") || strings.Contains(source, "JWT_SECRET") {
			findings = append(findings, finding("money.symmetric_token_verification", "Symmetric token verification is present", path, "Verify LPBS leases with the published asymmetric JWKS."))
		}
		return nil
	})
	return findings
}

func validateDeclaration(root, manifest string, declaration declaration) []*commonv1.AssessmentFinding {
	var findings []*commonv1.AssessmentFinding
	if declaration.Version != 1 || strings.TrimSpace(declaration.BundleKey) == "" {
		findings = append(findings, finding("money.undeclared_monetization", "Monetization declaration has invalid identity", manifest, "Use version 1 and provide a non-empty bundle_key."))
	}
	seenFeatures := map[string]struct{}{}
	for _, feature := range declaration.Features {
		key := strings.TrimSpace(feature.Key)
		if key == "" {
			findings = append(findings, finding("money.feature_not_enforced", "Feature declaration is missing a key", manifest, "Give each feature a non-empty key."))
			continue
		}
		if feature.Class != "A" && feature.Class != "B" {
			findings = append(findings, finding("money.feature_not_enforced", "Feature declaration has an invalid class", manifest, "Set class to A for cost-bearing features or B for local-capacity features."))
		}
		if len(feature.EnforcementPaths) == 0 {
			findings = append(findings, finding("money.feature_not_enforced", "Feature has no enforcement path", manifest, "Declare at least one real enforcement path."))
		}
		if _, exists := seenFeatures[key]; exists {
			findings = append(findings, finding("money.feature_not_enforced", "Feature is declared more than once", manifest, "Declare each feature key once and keep one authoritative enforcement surface."))
		}
		seenFeatures[key] = struct{}{}
		if feature.Class == "A" && hasClientPath(feature.EnforcementPaths) {
			findings = append(findings, finding("money.cost_bearing_meter_client_executed", "A cost-bearing feature is enforced in client code", manifest, "Move the cost-bearing operation and wallet check behind LPBS."))
		}
		if missing := missingPaths(root, feature.EnforcementPaths); missing != "" {
			findings = append(findings, finding("money.feature_not_enforced", "Declared feature enforcement path is missing", missing, "Add the declared enforcement path or correct the manifest."))
		}
	}
	seenMeters := map[string]struct{}{}
	for _, meter := range declaration.Meters {
		key := strings.TrimSpace(meter.LimitKey)
		if key == "" {
			findings = append(findings, finding("money.meter_missing_tier_limits", "Meter declaration is missing a limit key", manifest, "Give each meter a non-empty limit_key."))
			continue
		}
		if meter.Class != "A" && meter.Class != "B" {
			findings = append(findings, finding("money.meter_missing_tier_limits", "Meter declaration has an invalid class", manifest, "Set class to A for cost-bearing meters or B for local-capacity meters."))
		}
		if len(meter.EnforcementPaths) == 0 {
			findings = append(findings, finding("money.meter_missing_tier_limits", "Meter has no enforcement path", manifest, "Declare at least one real enforcement path."))
		}
		if _, exists := seenMeters[key]; exists {
			findings = append(findings, finding("money.meter_missing_tier_limits", "Meter is declared more than once", manifest, "Declare each limit key once."))
		}
		seenMeters[key] = struct{}{}
		if meter.Class == "A" && hasClientPath(meter.EnforcementPaths) {
			findings = append(findings, finding("money.cost_bearing_meter_client_executed", "A cost-bearing meter is client-executed", manifest, "Charge and reserve the meter server-side before returning work."))
		}
		if missing := missingPaths(root, meter.EnforcementPaths); missing != "" {
			findings = append(findings, finding("money.feature_not_enforced", "Declared meter enforcement path is missing", missing, "Add the declared enforcement path or correct the manifest."))
		}
	}
	return findings
}

func scanServiceSecrets(root string) []*commonv1.AssessmentFinding {
	path := filepath.Join(root, ".vrooli", "service.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var document any
	if json.Unmarshal(data, &document) != nil {
		return nil
	}
	var findings []*commonv1.AssessmentFinding
	var visit func(any, string)
	visit = func(value any, location string) {
		switch typed := value.(type) {
		case map[string]any:
			classification, _ := typed["classification"].(string)
			lowerLocation := strings.ToLower(location)
			if classification == "service" && (strings.Contains(lowerLocation, "tier-2") || strings.Contains(lowerLocation, "tier-3")) {
				findings = append(findings, finding("money.service_token_in_client_bundle", "Service-classified secret is declared for a desktop tier", path, "Use a user-scoped consumer token; no shared service secret may ship in tier 2 or tier 3."))
			}
			for key, child := range typed {
				visit(child, location+"/"+key)
			}
		case []any:
			for index, child := range typed {
				visit(child, fmt.Sprintf("%s/%d", location, index))
			}
		}
	}
	visit(document, "")
	return findings
}

func missingPaths(root string, paths []string) string {
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			return "manifest"
		}
		if _, err := os.Stat(filepath.Join(root, filepath.Clean(path))); err != nil {
			return filepath.Join(root, filepath.Clean(path))
		}
	}
	return ""
}

func hasClientPath(paths []string) bool {
	for _, path := range paths {
		lower := strings.ToLower(path)
		if strings.HasPrefix(lower, "ui/") || strings.Contains(lower, "client") || strings.HasSuffix(lower, ".tsx") || strings.HasSuffix(lower, ".ts") {
			return true
		}
	}
	return false
}

func finding(code, title, location, remediation string) *commonv1.AssessmentFinding {
	globalImpact := commonv1.GlobalImpact_GLOBAL_IMPACT_FOUNDATION_BLOCKER
	if code == "money.service_token_in_client_bundle" || code == "money.symmetric_token_verification" {
		globalImpact = commonv1.GlobalImpact_GLOBAL_IMPACT_SAFETY_BLOCKER
	}
	return &commonv1.AssessmentFinding{
		Code:        code,
		Severity:    "SEVERITY_ERROR",
		Title:       title,
		Message:     title,
		Location:    location,
		Remediation: remediation,
		FixClass:    "detection_only",
		Maturity: &commonv1.FindingMaturity{
			CapabilityId:     "monetization_boundary",
			LocalLevel:       "L1",
			GlobalImpact:     globalImpact,
			Dimension:        "operational-targets",
			CleanRequirement: commonv1.CleanRequirement_CLEAN_REQUIREMENT_REQUIRED,
		},
	}
}

func buildAssessment(scenario string, findings []*commonv1.AssessmentFinding) *commonv1.MaturityAssessment {
	blocking := make([]string, 0, len(findings))
	for _, f := range findings {
		blocking = append(blocking, f.Code)
	}
	level := "L4"
	if len(findings) > 0 {
		level = "L1"
	}
	return &commonv1.MaturityAssessment{Scenario: scenario, Provider: "landing-page-business-suite", Phase: "monetization-conformance", Version: "1.0.0", Findings: findings, Local: &commonv1.LocalMaturityAssessment{CurrentLevel: level, NextLevel: "L4", BlockingFindingCodes: blocking, Clean: len(findings) == 0}}
}

func failureClass(status scenariovalidationv1.ValidationStatus) string {
	if status == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		return "static_findings"
	}
	return ""
}
