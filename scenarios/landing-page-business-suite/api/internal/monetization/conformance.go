package monetization

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

// Handler owns static trust-boundary checks and live reconciliation against
// the LPBS catalog. The live reader is injected at the composition root so
// the provider never invents a second catalog authority.
type Handler struct {
	catalog   catalogReader
	bundleKey func() string
}

type catalogReader interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type declaration struct {
	Version             int       `json:"version"`
	BundleKey           string    `json:"bundle_key"`
	AppKey              string    `json:"app_key"`
	RequiresEntitlement bool      `json:"requires_entitlement"`
	Features            []surface `json:"features"`
	Meters              []surface `json:"meters"`
	AccountSurfacePaths []string  `json:"account_surface_paths"`
	JourneyProbePaths   []string  `json:"journey_probe_paths"`
}

type surface struct {
	Key              string   `json:"key"`
	LimitKey         string   `json:"limit_key"`
	Class            string   `json:"class"`
	Outbox           string   `json:"outbox"`
	Byok             bool     `json:"byok"`
	EnforcementPaths []string `json:"enforcement_paths"`
}

func RegisterRoutes(router *mux.Router, catalog catalogReader, bundleKey func() string) {
	path, handler := scenariovalidationconnect.NewScenarioValidationServiceHandler(Handler{catalog: catalog, bundleKey: bundleKey})
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}

func (h Handler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	path := req.Msg.GetPath()
	if path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario path is required"))
	}
	findings := scan(path)
	status := scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED
	if liveFindings, liveErr := h.liveFindings(ctx, path); liveErr != nil {
		findings = append(findings, finding("money.live_catalog_unavailable", "LPBS monetization catalog is unavailable", path, liveErr.Error()))
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED
	} else {
		findings = append(findings, liveFindings...)
	}
	assessment := buildAssessment(req.Msg.GetScenario(), findings)
	native, _ := anypb.New(nativeStruct(req.Msg.GetScenario(), findings))
	if status != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED && len(findings) > 0 {
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
	if liveFindings, liveErr := h.liveFindings(ctx, path); liveErr != nil {
		findings = append(findings, finding("money.live_catalog_unavailable", "LPBS monetization catalog is unavailable", path, liveErr.Error()))
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED
	} else {
		findings = append(findings, liveFindings...)
	}
	if status != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED && len(findings) > 0 {
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED
	}
	assessment := buildAssessment(req.Msg.GetTarget().GetId(), findings)
	native, _ := anypb.New(nativeStruct(req.Msg.GetTarget().GetId(), findings))
	return connect.NewResponse(&scenariovalidationv1.ValidateTargetResponse{Target: req.Msg.GetTarget(), Status: status, Assessment: assessment, NativeDetail: native, FailureClassification: failureClass(status)}), nil
}

func (Handler) DescribeProvider(context.Context, *connect.Request[scenariovalidationv1.DescribeProviderRequest]) (*connect.Response[scenariovalidationv1.DescribeProviderResponse], error) {
	return connect.NewResponse(&scenariovalidationv1.DescribeProviderResponse{
		Provider: "landing-page-business-suite", Phase: "monetization-conformance", SpecVersion: "1.1.0", Contract: "scenario-validation/v1",
		Capabilities: &scenariovalidationv1.ProviderCapabilities{DeliveryMode: "inline", SupportsExecution: false, SupportsFixes: false, TargetKinds: []commonv1.ValidationTargetKind{commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO}},
	}), nil
}

func (Handler) PreviewFix(context.Context, *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return connect.NewResponse(&scenariovalidationv1.FixResponse{Applied: false, Messages: []string{"Monetization conformance is detection-only; apply the remediation documented by the finding."}}), nil
}

func (Handler) ApplyFix(context.Context, *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return connect.NewResponse(&scenariovalidationv1.FixResponse{Applied: false, Messages: []string{"Monetization conformance does not perform source rewrites."}}), nil
}

func (h Handler) liveFindings(ctx context.Context, root string) ([]*commonv1.AssessmentFinding, error) {
	if h.catalog == nil {
		return nil, nil
	}
	declarationData, err := readDeclaration(root)
	if err != nil {
		return nil, nil
	}
	snapshot, err := h.catalogSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	findings := make([]*commonv1.AssessmentFinding, 0)
	for _, candidate := range reconciliationFindings(declarationData, snapshot, root) {
		findings = append(findings, finding(candidate.Code, liveFindingTitle(candidate.Code), candidate.Location, candidate.Message))
	}
	return findings, nil
}

func (h Handler) catalogSnapshot(ctx context.Context) (CatalogSnapshot, error) {
	bundleKey := "business_suite"
	if h.bundleKey != nil && strings.TrimSpace(h.bundleKey()) != "" {
		bundleKey = strings.TrimSpace(h.bundleKey())
	}
	snapshot := CatalogSnapshot{Bundles: map[string]BundleSnapshot{bundleKey: {Apps: map[string]bool{}, Tiers: map[string]map[string]bool{}}}}
	bundle := snapshot.Bundles[bundleKey]
	rows, err := h.catalog.QueryContext(ctx, `SELECT app_key FROM download_apps WHERE bundle_key = $1`, bundleKey)
	if err != nil {
		return CatalogSnapshot{}, fmt.Errorf("query LPBS download catalog: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var appKey string
		if err := rows.Scan(&appKey); err != nil {
			_ = rows.Close()
			return CatalogSnapshot{}, fmt.Errorf("scan LPBS download catalog: %w", err)
		}
		bundle.Apps[appKey] = true
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return CatalogSnapshot{}, fmt.Errorf("read LPBS download catalog: %w", err)
	}
	if err := rows.Close(); err != nil {
		return CatalogSnapshot{}, fmt.Errorf("close LPBS download catalog: %w", err)
	}

	rows, err = h.catalog.QueryContext(ctx, `SELECT tier_id, limit_key FROM subscription_tier_limits WHERE app_bundle_key = $1`, bundleKey)
	if err != nil {
		return CatalogSnapshot{}, fmt.Errorf("query LPBS tier limits: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var tierID, limitKey string
		if err := rows.Scan(&tierID, &limitKey); err != nil {
			_ = rows.Close()
			return CatalogSnapshot{}, fmt.Errorf("scan LPBS tier limits: %w", err)
		}
		if bundle.Tiers[tierID] == nil {
			bundle.Tiers[tierID] = map[string]bool{}
		}
		bundle.Tiers[tierID][limitKey] = true
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return CatalogSnapshot{}, fmt.Errorf("read LPBS tier limits: %w", err)
	}
	if err := rows.Close(); err != nil {
		return CatalogSnapshot{}, fmt.Errorf("close LPBS tier limits: %w", err)
	}
	snapshot.Bundles[bundleKey] = bundle
	return snapshot, nil
}

func liveFindingTitle(code string) string {
	switch code {
	case "money.unknown_bundle_key":
		return "Declared bundle is not present in LPBS"
	case "money.unregistered_app_key":
		return "Declared app is not registered in LPBS"
	case "money.meter_missing_tier_limits":
		return "Declared meter is missing a tier limit"
	default:
		return "LPBS monetization catalog reconciliation failed"
	}
}

func readDeclaration(root string) (declaration, error) {
	data, err := os.ReadFile(filepath.Join(root, ".vrooli", "monetization.json"))
	if err != nil {
		return declaration{}, err
	}
	var result declaration
	if err := json.Unmarshal(data, &result); err != nil {
		return declaration{}, err
	}
	return result, nil
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
	findings = append(findings, scanLiveReconciliation(root, declarationData)...)
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if strings.Contains(filepath.ToSlash(path), "/api/internal/monetization/") {
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

func scanLiveReconciliation(root string, declaration declaration) []*commonv1.AssessmentFinding {
	snapshotPath := strings.TrimSpace(os.Getenv("LPBS_MONETIZATION_CATALOG"))
	if snapshotPath == "" {
		return nil
	}
	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		return nil // degraded: the live authority was not reachable
	}
	var snapshot CatalogSnapshot
	if json.Unmarshal(data, &snapshot) != nil {
		return nil
	}
	findings := make([]*commonv1.AssessmentFinding, 0)
	for _, candidate := range reconciliationFindings(declaration, snapshot, root) {
		var findingValue *commonv1.AssessmentFinding
		switch candidate.Code {
		case "money.unknown_bundle_key":
			findingValue = finding("money.unknown_bundle_key", "Declared bundle is not present in LPBS", candidate.Location, candidate.Message)
		case "money.unregistered_app_key":
			findingValue = finding("money.unregistered_app_key", "Declared app is not registered in LPBS", candidate.Location, candidate.Message)
		case "money.meter_missing_tier_limits":
			findingValue = finding("money.meter_missing_tier_limits", "Declared meter is missing a tier limit", candidate.Location, candidate.Message)
		case "money.no_account_surface":
			findingValue = finding("money.no_account_surface", "Entitled scenario has no account surface", candidate.Location, candidate.Message)
		case "money.no_journey_probe":
			findingValue = finding("money.no_journey_probe", "Entitled scenario has no journey probe", candidate.Location, candidate.Message)
		}
		if findingValue != nil {
			findings = append(findings, findingValue)
		}
	}
	return findings
}

func validateDeclaration(root, manifest string, declaration declaration) []*commonv1.AssessmentFinding {
	var findings []*commonv1.AssessmentFinding
	if declaration.Version != 2 || strings.TrimSpace(declaration.BundleKey) == "" || strings.TrimSpace(declaration.AppKey) == "" {
		findings = append(findings, finding("money.undeclared_monetization", "Monetization declaration has invalid identity", manifest, "Use version 2 and provide non-empty bundle_key and app_key values."))
	}
	if declaration.RequiresEntitlement {
		if len(declaration.AccountSurfacePaths) == 0 {
			findings = append(findings, finding("money.no_account_surface", "Entitled scenario has no declared account surface", manifest, "Declare account_surface_paths for the sign-in, plan, subscription, and credit surface."))
		} else if missing := missingPaths(root, declaration.AccountSurfacePaths); missing != "" {
			findings = append(findings, finding("money.no_account_surface", "Declared account surface path is missing", missing, "Add the account surface or correct account_surface_paths."))
		}
		if len(declaration.JourneyProbePaths) == 0 {
			findings = append(findings, finding("money.no_journey_probe", "Entitled scenario has no declared journey probe", manifest, "Declare journey_probe_paths for the provider-neutral monetization journey endpoint."))
		} else if missing := missingPaths(root, declaration.JourneyProbePaths); missing != "" {
			findings = append(findings, finding("money.no_journey_probe", "Declared journey probe path is missing", missing, "Add the journey probe or correct journey_probe_paths."))
		}
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
		if meter.Class == "B" {
			if strings.TrimSpace(meter.Outbox) == "" {
				findings = append(findings, finding("money.no_outbox_for_local_meter", "Class B meter has no durable outbox declaration", manifest, "Declare the durable outbox path for this local-capacity meter."))
			} else if !hasDurableOutbox(root, meter.Outbox) {
				findings = append(findings, finding("money.no_outbox_for_local_meter", "Declared Class B outbox is not durable", filepath.Join(root, filepath.Clean(meter.Outbox)), "Persist usage before delivery and retain it across process restarts."))
			}
		}
	}
	findings = append(findings, scanLocalLimitConfig(root, declaration)...)
	findings = append(findings, scanOfflineBlocking(root, manifest, declaration)...)
	return findings
}

func scanLocalLimitConfig(root string, declaration declaration) []*commonv1.AssessmentFinding {
	keys := make([]string, 0, len(declaration.Meters))
	for _, meter := range declaration.Meters {
		if key := strings.TrimSpace(meter.LimitKey); key != "" {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return nil
	}
	var findings []*commonv1.AssessmentFinding
	forEachSourceFile(root, func(path string, source string) {
		lowerPath := strings.ToLower(path)
		if !(strings.Contains(lowerPath, "config") || strings.Contains(lowerPath, ".env") || strings.Contains(lowerPath, ".vrooli")) {
			return
		}
		for _, key := range keys {
			if strings.Contains(source, key) && hardCodedNumericValue(source, key) {
				findings = append(findings, finding("money.limits_from_local_config", "Meter limit is duplicated in local configuration", path, "Read the limit from the signed entitlement lease; remove the local numeric limit."))
				return
			}
		}
	})
	return findings
}

func scanOfflineBlocking(root, manifest string, declaration declaration) []*commonv1.AssessmentFinding {
	var findings []*commonv1.AssessmentFinding
	for _, feature := range declaration.Features {
		for _, relative := range feature.EnforcementPaths {
			path := filepath.Join(root, filepath.Clean(relative))
			source, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			lower := strings.ToLower(string(source))
			if strings.Contains(lower, "offline") && strings.Contains(lower, "return false") && !strings.Contains(lower, "cache") {
				findings = append(findings, finding("money.gate_blocks_offline", "Gate denies during an offline path without a cached lease", path, "Consult a still-valid cached lease before denying; expire only when not_after passes."))
			}
		}
	}
	_ = manifest
	return findings
}

func hasDurableOutbox(root, relative string) bool {
	path := filepath.Join(root, filepath.Clean(relative))
	var durable bool
	forEachSourceFile(path, func(_ string, source string) {
		lower := strings.ToLower(source)
		if strings.Contains(lower, "outbox") && (strings.Contains(lower, "insert into") || strings.Contains(lower, "persist") || strings.Contains(lower, "sqlite")) {
			durable = true
		}
	})
	return durable
}

func forEachSourceFile(root string, visit func(path, source string)) {
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" || name == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Size() > 2<<20 {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr == nil {
			visit(path, string(data))
		}
		return nil
	})
}

func hardCodedNumericValue(source, key string) bool {
	quoted := regexp.QuoteMeta(key)
	pattern := regexp.MustCompile(`(?i)["']?` + quoted + `["']?\s*[:=]\s*-?[0-9]+`)
	return pattern.MatchString(source)
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
	return &commonv1.MaturityAssessment{Scenario: scenario, Provider: "landing-page-business-suite", Phase: "monetization-conformance", Version: "1.1.0", Findings: findings, Local: &commonv1.LocalMaturityAssessment{CurrentLevel: level, NextLevel: "L4", BlockingFindingCodes: blocking, Clean: len(findings) == 0}}
}

func failureClass(status scenariovalidationv1.ValidationStatus) string {
	if status == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED {
		return "live_reconciliation_unavailable"
	}
	if status == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		return "static_findings"
	}
	return ""
}
