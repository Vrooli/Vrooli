package dependencygovernance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"

	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
	governanceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance/dependency_governance_v1connect"
)

const Guidance = "These are dependencies already recorded as approved for the shown package/range. This is not an exhaustive allowlist. If a better dependency is appropriate, suggest it with purpose, version/range, alternatives considered, and security/license notes so it can be reviewed and recorded."

type Surface struct {
	ID       string
	Language string
	RootPath string
}

type Registry struct {
	repoRoot string
}

type registryFile struct {
	SchemaVersion        string                     `json:"schema_version"`
	Version              string                     `json:"version"`
	Guidance             string                     `json:"guidance"`
	Policy               registryPolicy             `json:"policy"`
	ReleaseAgeExceptions []string                   `json:"release_age_exceptions"`
	Records              []approvedDependencyRecord `json:"records"`
}

type registryPolicy struct {
	Mode string `json:"mode"`
}

type approvedDependencyRecord struct {
	Ecosystem        string   `json:"ecosystem"`
	PackageName      string   `json:"package_name"`
	VersionRange     string   `json:"version_range"`
	State            string   `json:"state"`
	AllowedSurfaces  []string `json:"allowed_surfaces"`
	UseCases         []string `json:"use_cases"`
	Rationale        string   `json:"rationale"`
	ApprovedBy       string   `json:"approved_by"`
	ApprovedDate     string   `json:"approved_date"`
	LastReviewed     string   `json:"last_reviewed"`
	ReviewExpires    string   `json:"review_expires"`
	LicenseNotes     string   `json:"license_notes"`
	SecurityNotes    string   `json:"security_notes"`
	ExampleScenarios []string `json:"example_scenarios"`
	Replacement      string   `json:"replacement"`
	Keywords         []string `json:"keywords"`
	AllowedScenarios []string `json:"allowed_scenarios"`
	DeniedScenarios  []string `json:"denied_scenarios"`
	AllowedGroups    []string `json:"allowed_dependency_groups"`
	RangePolicy      string   `json:"range_policy"`
}

type securityHealthExplainResponse struct {
	Vulnerability securityHealthVulnerability `json:"vulnerability"`
	Found         bool                        `json:"found"`
}

type securityHealthListResponse struct {
	Vulnerabilities []securityHealthVulnerability `json:"vulnerabilities"`
	Total           int32                         `json:"total"`
}

type securityHealthVulnerability struct {
	VulnerabilityID      string                       `json:"vulnerability_id"`
	VulnerabilityIDCamel string                       `json:"vulnerabilityId"`
	Aliases              []string                     `json:"aliases"`
	Ecosystem            string                       `json:"ecosystem"`
	Name                 string                       `json:"name"`
	Version              string                       `json:"version"`
	AffectedRanges       []securityHealthVersionRange `json:"affected_ranges"`
	AffectedRangesCamel  []securityHealthVersionRange `json:"affectedRanges"`
	FixedRanges          []securityHealthVersionRange `json:"fixed_ranges"`
	FixedRangesCamel     []securityHealthVersionRange `json:"fixedRanges"`
	Severity             string                       `json:"severity"`
	NormalizedSeverity   string                       `json:"normalized_severity"`
	NormalizedCamel      string                       `json:"normalizedSeverity"`
	AdvisoryURL          string                       `json:"advisory_url"`
	AdvisoryURLCamel     string                       `json:"advisoryUrl"`
	Summary              string                       `json:"summary"`
	Source               string                       `json:"source"`
	Reachability         string                       `json:"reachability"`
	Confidence           string                       `json:"confidence"`
	Production           bool                         `json:"production"`
	DevOnly              bool                         `json:"dev_only"`
	DevOnlyCamel         bool                         `json:"devOnly"`
	Scenarios            []string                     `json:"scenarios"`
	SourceFiles          []string                     `json:"source_files"`
	SourceFilesCamel     []string                     `json:"sourceFiles"`
	Remediation          string                       `json:"remediation"`
}

type securityHealthVersionRange struct {
	Range        string `json:"range"`
	Version      string `json:"version"`
	Introduced   string `json:"introduced"`
	Fixed        string `json:"fixed"`
	LastAffected string `json:"last_affected"`
	LastCamel    string `json:"lastAffected"`
}

type loadedRegistry struct {
	records []*governancev1.ApprovedDependencyRecord
	policy  registryPolicy
}

func NewRegistry(repoRoot string) *Registry {
	return &Registry{repoRoot: strings.TrimSpace(repoRoot)}
}

func RegisterConnectRoutes(router *gin.Engine, scenariosDir func() string) {
	connectPath, connectHandler := governanceconnect.NewDependencyGovernanceServiceHandler(&connectHandler{
		scenariosDir: scenariosDir,
	})
	router.Any(connectPath+"*path", gin.WrapH(connectHandler))
}

type connectHandler struct {
	scenariosDir func() string
}

func (h *connectHandler) ListApprovedDependencies(_ context.Context, req *connect.Request[governancev1.ListApprovedDependenciesRequest]) (*connect.Response[governancev1.ApprovedDependencyListResponse], error) {
	records, summary, err := h.registry().List(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&governancev1.ApprovedDependencyListResponse{
		Records:  records,
		Summary:  summary,
		Guidance: Guidance,
	}), nil
}

func (h *connectHandler) SearchApprovedDependencies(_ context.Context, req *connect.Request[governancev1.SearchApprovedDependenciesRequest]) (*connect.Response[governancev1.ApprovedDependencySearchResponse], error) {
	records, summary, err := h.registry().Search(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&governancev1.ApprovedDependencySearchResponse{
		Records:  records,
		Summary:  summary,
		Guidance: Guidance,
	}), nil
}

func (h *connectHandler) ExplainApprovedDependency(_ context.Context, req *connect.Request[governancev1.ExplainApprovedDependencyRequest]) (*connect.Response[governancev1.ApprovedDependencyExplainResponse], error) {
	record, found, err := h.registry().Explain(req.Msg.GetEcosystem(), req.Msg.GetPackageName())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&governancev1.ApprovedDependencyExplainResponse{
		Record:   record,
		Found:    found,
		Guidance: Guidance,
	}), nil
}

func (h *connectHandler) ValidateApprovedDependencies(_ context.Context, req *connect.Request[governancev1.ValidateApprovedDependenciesRequest]) (*connect.Response[governancev1.ApprovedDependencyValidationResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	resp, err := h.registry().ValidateScenario(scenario, req.Msg.GetPolicyMode())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ValidateFleetApprovedDependencies(_ context.Context, req *connect.Request[governancev1.ValidateFleetApprovedDependenciesRequest]) (*connect.Response[governancev1.FleetApprovedDependencyValidationResponse], error) {
	resp, err := h.registry().ValidateFleet(req.Msg.GetPolicyMode())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ListApprovedDependencyFindings(_ context.Context, req *connect.Request[governancev1.ListApprovedDependencyFindingsRequest]) (*connect.Response[governancev1.ApprovedDependencyFindingsResponse], error) {
	resp, err := h.registry().ListFindings(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetApprovedDependencyTriage(_ context.Context, req *connect.Request[governancev1.GetApprovedDependencyTriageRequest]) (*connect.Response[governancev1.ApprovedDependencyTriageResponse], error) {
	resp, err := h.registry().GetTriage(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetApprovedDependencyUsage(_ context.Context, req *connect.Request[governancev1.GetApprovedDependencyUsageRequest]) (*connect.Response[governancev1.ApprovedDependencyUsageResponse], error) {
	resp, err := h.registry().GetUsage(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) UpsertApprovedDependency(_ context.Context, req *connect.Request[governancev1.UpsertApprovedDependencyRequest]) (*connect.Response[governancev1.UpsertApprovedDependencyResponse], error) {
	resp, err := h.registry().Upsert(req.Msg.GetRecord(), req.Msg.GetDryRun())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ProposeApprovedDependencyRecords(_ context.Context, req *connect.Request[governancev1.ProposeApprovedDependencyRecordsRequest]) (*connect.Response[governancev1.ApprovedDependencyProposalResponse], error) {
	resp, err := h.registry().ProposeRecords(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) BatchUpsertApprovedDependencies(_ context.Context, req *connect.Request[governancev1.BatchUpsertApprovedDependenciesRequest]) (*connect.Response[governancev1.BatchUpsertApprovedDependenciesResponse], error) {
	resp, err := h.registry().BatchUpsert(req.Msg.GetRecords(), req.Msg.GetDryRun())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ApproveObservedDependency(_ context.Context, req *connect.Request[governancev1.ApproveObservedDependencyRequest]) (*connect.Response[governancev1.DependencyGovernanceDecisionResponse], error) {
	resp, err := h.registry().ApproveObserved(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) WidenApprovedDependencyRange(_ context.Context, req *connect.Request[governancev1.WidenApprovedDependencyRangeRequest]) (*connect.Response[governancev1.DependencyGovernanceDecisionResponse], error) {
	resp, err := h.registry().WidenRange(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ListSecurityGovernanceGaps(ctx context.Context, req *connect.Request[governancev1.ListSecurityGovernanceGapsRequest]) (*connect.Response[governancev1.SecurityGovernanceGapsResponse], error) {
	resp, err := h.registry().ListSecurityGovernanceGaps(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) PreviewVulnerabilityRemediation(ctx context.Context, req *connect.Request[governancev1.PreviewVulnerabilityRemediationRequest]) (*connect.Response[governancev1.VulnerabilityRemediationResponse], error) {
	resp, err := h.registry().PreviewVulnerabilityRemediation(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) DenyVulnerableDependency(ctx context.Context, req *connect.Request[governancev1.DenyVulnerableDependencyRequest]) (*connect.Response[governancev1.VulnerabilityRemediationResponse], error) {
	resp, err := h.registry().DenyVulnerableDependency(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) registry() *Registry {
	scenariosDir := ""
	if h != nil && h.scenariosDir != nil {
		scenariosDir = h.scenariosDir()
	}
	return NewRegistry(filepath.Dir(scenariosDir))
}

func (r *Registry) List(req *governancev1.ListApprovedDependenciesRequest) ([]*governancev1.ApprovedDependencyRecord, *governancev1.DependencyGovernanceSummary, error) {
	records, err := r.loadRecords()
	if err != nil {
		return nil, nil, err
	}
	filtered := make([]*governancev1.ApprovedDependencyRecord, 0, len(records))
	for _, record := range records {
		if !matchesFilter(record.GetEcosystem(), req.GetEcosystem()) || !matchesFilter(record.GetState(), req.GetState()) {
			continue
		}
		if req.GetUseCase() != "" && !containsFold(record.GetUseCases(), req.GetUseCase()) {
			continue
		}
		filtered = append(filtered, record)
	}
	sortRecords(filtered)
	return filtered, summarizeRecords(filtered), nil
}

func (r *Registry) Search(req *governancev1.SearchApprovedDependenciesRequest) ([]*governancev1.ApprovedDependencyRecord, *governancev1.DependencyGovernanceSummary, error) {
	records, err := r.loadRecords()
	if err != nil {
		return nil, nil, err
	}
	queryTerms := queryTerms(req.GetQuery())
	matches := make([]*governancev1.ApprovedDependencyRecord, 0, len(records))
	for _, record := range records {
		if !matchesFilter(record.GetEcosystem(), req.GetEcosystem()) {
			continue
		}
		if len(queryTerms) == 0 || recordMatches(record, queryTerms) {
			matches = append(matches, record)
		}
	}
	sortRecords(matches)
	if limit := int(req.GetLimit()); limit > 0 && len(matches) > limit {
		matches = matches[:limit]
	}
	return matches, summarizeRecords(matches), nil
}

func (r *Registry) Explain(ecosystem, packageName string) (*governancev1.ApprovedDependencyRecord, bool, error) {
	records, err := r.loadRecords()
	if err != nil {
		return nil, false, err
	}
	for _, record := range records {
		if sameFold(record.GetEcosystem(), ecosystem) && sameFold(record.GetPackageName(), packageName) {
			return record, true, nil
		}
	}
	return nil, false, nil
}

func (r *Registry) ValidateScenario(scenario string, policyModeOverride ...string) (*governancev1.ApprovedDependencyValidationResponse, error) {
	scenarioDir := filepath.Join(filepath.Dir(r.registryPath()), "..", "scenarios", scenario)
	if r.repoRoot != "" {
		scenarioDir = filepath.Join(r.repoRoot, "scenarios", scenario)
	}
	observed, err := ScanScenarioDependencies(scenarioDir)
	if err != nil {
		return nil, err
	}
	return r.ValidateObserved(scenario, observed, policyModeOverride...)
}

func (r *Registry) ValidateFleet(policyModeOverride ...string) (*governancev1.FleetApprovedDependencyValidationResponse, error) {
	loaded, err := r.loadRegistry()
	if err != nil {
		return nil, err
	}
	scenarios, err := r.discoverScenarios()
	if err != nil {
		return nil, err
	}
	responses := make([]*governancev1.ApprovedDependencyValidationResponse, 0, len(scenarios))
	for _, scenario := range scenarios {
		resp, err := r.ValidateScenario(scenario, policyModeOverride...)
		if err != nil {
			resp = degradedScenarioResponse(scenario, err, r.effectivePolicyMode(policyModeOverride...))
		}
		responses = append(responses, resp)
	}
	summary := summarizeRecords(loaded.records)
	summary.Status = "pass"
	summary.PolicyMode = r.effectivePolicyMode(policyModeOverride...)
	summary.ScenarioCount = int32(len(responses))
	var findings []*governancev1.ApprovedDependencyFinding
	for _, resp := range responses {
		mergeSummary(summary, resp.GetSummary())
		findings = append(findings, resp.GetFindings()...)
	}
	summary.FindingCount = int32(len(findings))
	summary.DependencyCount = int32(len(buildUsageGroups(responses)))
	status, passed := statusFromFindings(findings, len(responses) == 0)
	summary.Status = status
	return &governancev1.FleetApprovedDependencyValidationResponse{
		Passed:      passed,
		Summary:     summary,
		Scenarios:   responses,
		UsageGroups: buildUsageGroups(responses),
		Findings:    findings,
		Guidance:    Guidance,
	}, nil
}

func (r *Registry) ListFindings(req *governancev1.ListApprovedDependencyFindingsRequest) (*governancev1.ApprovedDependencyFindingsResponse, error) {
	fleet, err := r.ValidateFleet(req.GetPolicyMode())
	if err != nil {
		return nil, err
	}
	findings := make([]*governancev1.ApprovedDependencyFinding, 0, len(fleet.GetFindings()))
	for _, finding := range fleet.GetFindings() {
		if !matchesFilter(finding.GetScenario(), req.GetScenario()) {
			continue
		}
		if !matchesFilter(finding.GetEcosystem(), req.GetEcosystem()) {
			continue
		}
		if req.GetPackageName() != "" && !sameFold(finding.GetPackageName(), req.GetPackageName()) {
			continue
		}
		if !matchesFilter(finding.GetSeverity(), req.GetSeverity()) {
			continue
		}
		if !matchesFilter(finding.GetFindingClass(), req.GetFindingClass()) {
			continue
		}
		findings = append(findings, finding)
	}
	return &governancev1.ApprovedDependencyFindingsResponse{
		Findings: findings,
		Summary:  summarizeFindings(findings, fleet.GetSummary().GetPolicyMode()),
		Guidance: Guidance,
	}, nil
}

func (r *Registry) GetTriage(req *governancev1.GetApprovedDependencyTriageRequest) (*governancev1.ApprovedDependencyTriageResponse, error) {
	fleet, err := r.ValidateFleet(req.GetPolicyMode())
	if err != nil {
		return nil, err
	}
	groups := buildTriageGroups(fleet.GetFindings(), req)
	resp := &governancev1.ApprovedDependencyTriageResponse{
		Summary:  fleet.GetSummary(),
		Guidance: Guidance,
	}
	limit := int(req.GetLimit())
	for _, group := range groups {
		switch group.GetSection() {
		case "security":
			resp.SecurityActions = appendLimited(resp.GetSecurityActions(), group, limit)
		case "seeding":
			resp.RegistrySeeding = appendLimited(resp.GetRegistrySeeding(), group, limit)
		case "ranges":
			resp.RangePolicy = appendLimited(resp.GetRangePolicy(), group, limit)
		case "expired":
			resp.StaleOrExpiredReviews = appendLimited(resp.GetStaleOrExpiredReviews(), group, limit)
		default:
			resp.ScenarioHotspots = appendLimited(resp.GetScenarioHotspots(), group, limit)
		}
	}
	return resp, nil
}

func (r *Registry) GetUsage(req *governancev1.GetApprovedDependencyUsageRequest) (*governancev1.ApprovedDependencyUsageResponse, error) {
	ecosystem := normalize(req.GetEcosystem())
	packageName := strings.TrimSpace(req.GetPackageName())
	if ecosystem == "" || packageName == "" {
		return nil, fmt.Errorf("ecosystem and package_name are required")
	}
	fleet, err := r.ValidateFleet(req.GetPolicyMode())
	if err != nil {
		return nil, err
	}
	key := recordKey(ecosystem, packageName)
	var group *governancev1.DependencyUsageGroup
	for _, candidate := range fleet.GetUsageGroups() {
		if recordKey(candidate.GetEcosystem(), candidate.GetPackageName()) == key {
			group = candidate
			break
		}
	}
	findings := make([]*governancev1.ApprovedDependencyFinding, 0)
	for _, finding := range fleet.GetFindings() {
		if recordKey(finding.GetEcosystem(), finding.GetPackageName()) == key {
			findings = append(findings, finding)
		}
	}
	summary := summarizeFindings(findings, fleet.GetSummary().GetPolicyMode())
	if group != nil {
		summary.Observed = group.GetUsageCount()
		summary.ScenarioCount = group.GetScenarioCount()
		summary.DependencyCount = 1
	}
	return &governancev1.ApprovedDependencyUsageResponse{
		Found:      group != nil,
		UsageGroup: group,
		Findings:   findings,
		Summary:    summary,
		Guidance:   Guidance,
	}, nil
}

func (r *Registry) ValidateObserved(scenario string, observed []*governancev1.ObservedDependency, policyModeOverride ...string) (*governancev1.ApprovedDependencyValidationResponse, error) {
	loaded, err := r.loadRegistry()
	if err != nil {
		return nil, err
	}
	records := loaded.records
	policyMode := normalize(firstNonEmpty(append(policyModeOverride, loaded.policy.Mode, "advisory")...))
	byKey := make(map[string]*governancev1.ApprovedDependencyRecord, len(records))
	for _, record := range records {
		byKey[recordKey(record.GetEcosystem(), record.GetPackageName())] = record
	}

	findings := make([]*governancev1.ApprovedDependencyFinding, 0)
	for _, dep := range observed {
		annotateDependencySignalCategory(dep)
		record := byKey[recordKey(dep.GetEcosystem(), dep.GetPackageName())]
		if record == nil {
			if isUnrecordedTransitiveDependency(dep) {
				continue
			}
			severity := "WARNING"
			if policyMode == "strict" {
				severity = "ERROR"
			}
			findings = append(findings, governanceFinding(scenario, dep, "UNRECORDED_DIRECT", severity, "Dependency needs governance review", "This dependency is not yet recorded in approved dependency memory.", "Keep the dependency if it is the right tool, and submit purpose, version/range, alternatives considered, and security/license notes for review.", dep.GetVersion(), "recorded approval, constraint, deprecation, or denial decision", policyMode))
			continue
		}
		state := normalize(record.GetState())
		switch state {
		case "blocked", "denied":
			if state == "denied" && normalize(record.GetRangePolicy()) == "security_denied" {
				if decision := evaluateVersionPolicy(dep.GetEcosystem(), dep.GetVersion(), record.GetVersionRange(), record.GetRangePolicy()); decision.Allowed {
					findings = append(findings, governanceFinding(scenario, dep, "DENIED_IN_USE", "ERROR", "Denied vulnerable dependency range is in use", "This dependency version matches a security-denied governance range.", firstNonEmpty(record.GetReplacement(), "Upgrade to a non-vulnerable fixed range or remove the dependency."), dep.GetVersion(), firstNonEmpty(record.GetVersionRange(), "security-denied range"), policyMode))
				} else if decision.FindingClass == "VERSION_RANGE_UNPARSEABLE" {
					findings = append(findings, governanceFinding(scenario, dep, decision.FindingClass, "WARNING", decision.Title, decision.Description, decision.Remediation, dep.GetVersion(), decision.Expected, policyMode))
				}
				continue
			}
			findings = append(findings, governanceFinding(scenario, dep, "DENIED_IN_USE", "ERROR", "Denied dependency is in use", "This dependency is recorded as denied for Vrooli usage.", firstNonEmpty(record.GetReplacement(), "Replace the dependency or file an explicit governance exception with rationale and expiry."), dep.GetVersion(), "dependency absent or governance exception approved", policyMode))
		case "deprecated":
			findings = append(findings, governanceFinding(scenario, dep, "DEPRECATED_IN_USE", "WARNING", "Deprecated dependency is in use", "This dependency is recorded as deprecated.", firstNonEmpty(record.GetReplacement(), "Plan a migration to a maintained replacement."), dep.GetVersion(), "replacement dependency in use", policyMode))
		case "approved", "approved_with_constraints", "needs_review", "exception", "":
			decision := evaluateVersionPolicy(dep.GetEcosystem(), dep.GetVersion(), record.GetVersionRange(), record.GetRangePolicy())
			if !decision.Allowed {
				findings = append(findings, governanceFinding(scenario, dep, decision.FindingClass, "WARNING", decision.Title, decision.Description, decision.Remediation, dep.GetVersion(), decision.Expected, policyMode))
			}
			if exception := scenarioExceptionViolation(scenario, record); exception.reason != "" {
				findings = append(findings, governanceFinding(scenario, dep, "SCENARIO_EXCEPTION_VIOLATION", exception.severity, "Dependency violates scenario-specific governance exception", exception.reason, "Update the dependency decision or remove usage from the scenario-specific exception.", scenario, recordScope(record), policyMode))
			}
			if expired(record.GetReviewExpires()) {
				findings = append(findings, governanceFinding(scenario, dep, "EXPIRED_APPROVAL", "WARNING", "Dependency governance review has expired", "This dependency approval or exception has passed its review expiry date.", "Review the dependency and renew, replace, or deny it.", record.GetReviewExpires(), "unexpired review date", policyMode))
			}
			if state == "needs_review" {
				findings = append(findings, governanceFinding(scenario, dep, "UNRECORDED_DIRECT", "WARNING", "Dependency approval still needs review", "This dependency has a governance record but has not been approved yet.", "Complete dependency review or choose an already approved alternative if appropriate.", dep.GetVersion(), "approved or approved_with_constraints", policyMode))
			}
		default:
			findings = append(findings, governanceFinding(scenario, dep, "REGISTRY_INVALID", "WARNING", "Dependency has unknown governance state", "This dependency has a governance record with an unrecognized state.", "Fix the approved dependency registry state value.", state, "approved, approved_with_constraints, needs_review, denied, or deprecated", policyMode))
		}
	}

	summary := summarizeRecords(records)
	summary.PolicyMode = policyMode
	summary.Observed = int32(len(observed))
	for _, finding := range findings {
		switch finding.GetFindingClass() {
		case "UNRECORDED_DIRECT":
			summary.Unrecorded++
		case "VERSION_OUT_OF_RANGE":
			summary.OutOfRange++
		case "SCENARIO_EXCEPTION_VIOLATION":
			summary.OutOfScope++
		case "EXPIRED_APPROVAL", "EXPIRED_EXCEPTION":
			summary.Expired++
		}
		switch strings.ToUpper(finding.GetSeverity()) {
		case "ERROR":
			summary.ErrorCount++
		case "WARNING":
			summary.WarningCount++
		default:
			summary.InfoCount++
		}
	}
	status, passed := statusFromFindings(findings, len(records) == 0)
	if len(records) == 0 && len(findings) == 0 {
		status = "not_configured"
	}
	summary.Status = status
	return &governancev1.ApprovedDependencyValidationResponse{
		Scenario:             scenario,
		Passed:               passed && status != "fail",
		Summary:              summary,
		Findings:             findings,
		ObservedDependencies: observed,
		Guidance:             Guidance,
	}, nil
}

func (r *Registry) ListSecurityGovernanceGaps(ctx context.Context, req *governancev1.ListSecurityGovernanceGapsRequest) (*governancev1.SecurityGovernanceGapsResponse, error) {
	if req == nil {
		req = &governancev1.ListSecurityGovernanceGapsRequest{}
	}
	evidence, total, err := r.securityVulnerabilityEvidenceList(ctx, req)
	if err != nil {
		return nil, err
	}
	records, err := r.loadRecords()
	if err != nil {
		return nil, err
	}
	recordByKey := make(map[string]*governancev1.ApprovedDependencyRecord, len(records))
	for _, record := range records {
		recordByKey[recordKey(record.GetEcosystem(), record.GetPackageName())] = record
	}

	resp := &governancev1.SecurityGovernanceGapsResponse{
		Total:    total,
		Guidance: Guidance,
	}
	minSeverity := normalize(req.GetMinimumSeverity())
	for _, vuln := range evidence {
		if minSeverity != "" && severityRank(vuln.GetNormalizedSeverity()) < severityRank(minSeverity) {
			continue
		}
		key := recordKey(vuln.GetEcosystem(), vuln.GetPackageName())
		record := recordByKey[key]
		deniedCovered := securityDeniedRecordCovers(record, vuln)
		approvedOverlap := approvedRecordOverlapsVulnerability(record, vuln)
		gap := &governancev1.SecurityGovernanceGap{
			GapId:                  "security-gap." + slug(key+"."+vuln.GetVulnerabilityId()+"."+vuln.GetObservedVersion()),
			Ecosystem:              vuln.GetEcosystem(),
			PackageName:            vuln.GetPackageName(),
			ObservedVersion:        vuln.GetObservedVersion(),
			VulnerabilityIds:       trimStrings(append([]string{vuln.GetVulnerabilityId()}, vuln.GetAliases()...)),
			Severity:               vuln.GetSeverity(),
			NormalizedSeverity:     vuln.GetNormalizedSeverity(),
			AffectedRanges:         securityRangeStrings(vuln.GetAffectedRanges()),
			FixedRanges:            securityFixedRangeStrings(vuln.GetFixedRanges()),
			Scenarios:              append([]string{}, vuln.GetScenarios()...),
			SourceFiles:            append([]string{}, vuln.GetSourceFiles()...),
			DeniedRecordCovers:     deniedCovered,
			ApprovedRecordOverlaps: approvedOverlap,
			SignalCategory:         securityGapSignalCategory(vuln),
			SuggestedCommand:       securityGapCommand(vuln),
			Remediation:            remediationForEvidence(vuln, matchingSecurityAffectedRange(vuln)),
		}
		resp.Gaps = append(resp.GetGaps(), gap)
		if deniedCovered {
			resp.DeniedCoveredCount++
		} else {
			resp.UncoveredCount++
		}
		if approvedOverlap {
			resp.ApprovedOverlapCount++
			resp.Warnings = append(resp.GetWarnings(), fmt.Sprintf("%s/%s has approved governance memory overlapping vulnerable version %s for %s", vuln.GetEcosystem(), vuln.GetPackageName(), vuln.GetObservedVersion(), vuln.GetVulnerabilityId()))
		}
	}
	sort.Slice(resp.Gaps, func(i, j int) bool {
		left := resp.Gaps[i]
		right := resp.Gaps[j]
		if left.GetDeniedRecordCovers() != right.GetDeniedRecordCovers() {
			return !left.GetDeniedRecordCovers()
		}
		if severityRank(left.GetNormalizedSeverity()) != severityRank(right.GetNormalizedSeverity()) {
			return severityRank(left.GetNormalizedSeverity()) > severityRank(right.GetNormalizedSeverity())
		}
		return left.GetGapId() < right.GetGapId()
	})
	if limit := int(req.GetLimit()); limit > 0 && len(resp.Gaps) > limit {
		resp.Gaps = resp.Gaps[:limit]
	}
	resp.WarningCount = int32(len(resp.GetWarnings()))
	return resp, nil
}

func (r *Registry) Upsert(record *governancev1.ApprovedDependencyRecord, dryRun bool) (*governancev1.UpsertApprovedDependencyResponse, error) {
	normalized := protoRecordToJSON(record).toProto()
	if err := validateRecord(normalized); err != nil {
		return nil, err
	}
	raw, err := r.loadRegistryFileForMutation()
	if err != nil {
		return nil, err
	}
	if raw.SchemaVersion == "" {
		raw.SchemaVersion = "1"
	}
	if raw.Policy.Mode == "" {
		raw.Policy.Mode = "advisory"
	}
	replacement := protoRecordToJSON(normalized)
	key := recordKey(normalized.GetEcosystem(), normalized.GetPackageName())
	var previous *governancev1.ApprovedDependencyRecord
	changed := true
	replaced := false
	for i, existing := range raw.Records {
		existingProto := existing.toProto()
		if recordKey(existingProto.GetEcosystem(), existingProto.GetPackageName()) != key {
			continue
		}
		previous = existingProto
		changed = !recordsEqual(previous, normalized)
		raw.Records[i] = replacement
		replaced = true
		break
	}
	if !replaced {
		raw.Records = append(raw.Records, replacement)
	}
	sort.Slice(raw.Records, func(i, j int) bool {
		left := raw.Records[i].toProto()
		right := raw.Records[j].toProto()
		return recordKey(left.GetEcosystem(), left.GetPackageName()) < recordKey(right.GetEcosystem(), right.GetPackageName())
	})
	if err := validateRegistryFile(raw); err != nil {
		return nil, err
	}
	if !dryRun && changed {
		if err := r.writeRegistryFile(raw); err != nil {
			return nil, err
		}
	}
	records := make([]*governancev1.ApprovedDependencyRecord, 0, len(raw.Records))
	for _, rawRecord := range raw.Records {
		records = append(records, rawRecord.toProto())
	}
	return &governancev1.UpsertApprovedDependencyResponse{
		Record:         normalized,
		PreviousRecord: previous,
		DryRun:         dryRun,
		Changed:        changed,
		Message:        mutationMessage(dryRun, changed, previous != nil, normalized),
		Summary:        summarizeRecords(records),
		Guidance:       Guidance,
	}, nil
}

func (r *Registry) ProposeRecords(req *governancev1.ProposeApprovedDependencyRecordsRequest) (*governancev1.ApprovedDependencyProposalResponse, error) {
	if req == nil {
		req = &governancev1.ProposeApprovedDependencyRecordsRequest{}
	}
	fleet, err := r.ValidateFleet(req.GetPolicyMode())
	if err != nil {
		return nil, err
	}
	groups := buildProposalGroups(fleet, req)
	limit := int(req.GetTopUnrecorded())
	if limit <= 0 {
		limit = 25
	}
	if len(groups) > limit {
		groups = groups[:limit]
	}
	records := make([]*governancev1.ApprovedDependencyRecord, 0, len(groups))
	warnings := make([]string, 0)
	for _, group := range groups {
		record := proposalRecordFromGroup(group, req.GetState(), req.GetRangeStrategy())
		records = append(records, record)
		if len(group.GetVulnerabilityIds()) > 0 {
			warnings = append(warnings, fmt.Sprintf("%s/%s has security-sensitive findings: %s", group.GetEcosystem(), group.GetPackageName(), strings.Join(group.GetVulnerabilityIds(), ",")))
		}
		if record.GetVersionRange() == "*" {
			warnings = append(warnings, fmt.Sprintf("%s/%s has multiple or unknown observed versions; proposal uses '*' and requires reviewer narrowing before approval", group.GetEcosystem(), group.GetPackageName()))
		}
	}
	return &governancev1.ApprovedDependencyProposalResponse{
		Records:        records,
		EvidenceGroups: groups,
		Warnings:       trimStrings(warnings),
		Summary:        summarizeRecords(records),
		Guidance:       Guidance,
	}, nil
}

func (r *Registry) BatchUpsert(records []*governancev1.ApprovedDependencyRecord, dryRun bool) (*governancev1.BatchUpsertApprovedDependenciesResponse, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("at least one approved dependency record is required")
	}
	raw, err := r.loadRegistryFileForMutation()
	if err != nil {
		return nil, err
	}
	if raw.SchemaVersion == "" {
		raw.SchemaVersion = "1"
	}
	if raw.Policy.Mode == "" {
		raw.Policy.Mode = "advisory"
	}

	normalized := make([]*governancev1.ApprovedDependencyRecord, 0, len(records))
	replacements := make([]approvedDependencyRecord, 0, len(records))
	seenBatch := map[string]struct{}{}
	for _, record := range records {
		candidate := protoRecordToJSON(record).toProto()
		if err := validateRecord(candidate); err != nil {
			return nil, err
		}
		key := recordKey(candidate.GetEcosystem(), candidate.GetPackageName())
		if _, ok := seenBatch[key]; ok {
			return nil, fmt.Errorf("duplicate approved dependency record in batch: %s", key)
		}
		seenBatch[key] = struct{}{}
		normalized = append(normalized, candidate)
		replacements = append(replacements, protoRecordToJSON(candidate))
	}

	mutations := make([]*governancev1.UpsertApprovedDependencyResponse, 0, len(normalized))
	changed := false
	for i, replacement := range replacements {
		record := normalized[i]
		key := recordKey(record.GetEcosystem(), record.GetPackageName())
		var previous *governancev1.ApprovedDependencyRecord
		replaced := false
		recordChanged := true
		for j, existing := range raw.Records {
			existingProto := existing.toProto()
			if recordKey(existingProto.GetEcosystem(), existingProto.GetPackageName()) != key {
				continue
			}
			previous = existingProto
			recordChanged = !recordsEqual(previous, record)
			raw.Records[j] = replacement
			replaced = true
			break
		}
		if !replaced {
			raw.Records = append(raw.Records, replacement)
		}
		changed = changed || recordChanged
		mutations = append(mutations, &governancev1.UpsertApprovedDependencyResponse{
			Record:         record,
			PreviousRecord: previous,
			DryRun:         dryRun,
			Changed:        recordChanged,
			Message:        mutationMessage(dryRun, recordChanged, previous != nil, record),
			Guidance:       Guidance,
		})
	}
	sort.Slice(raw.Records, func(i, j int) bool {
		left := raw.Records[i].toProto()
		right := raw.Records[j].toProto()
		return recordKey(left.GetEcosystem(), left.GetPackageName()) < recordKey(right.GetEcosystem(), right.GetPackageName())
	})
	if err := validateRegistryFile(raw); err != nil {
		return nil, err
	}
	if !dryRun && changed {
		if err := r.writeRegistryFile(raw); err != nil {
			return nil, err
		}
	}
	allRecords := make([]*governancev1.ApprovedDependencyRecord, 0, len(raw.Records))
	for _, rawRecord := range raw.Records {
		allRecords = append(allRecords, rawRecord.toProto())
	}
	summary := summarizeRecords(allRecords)
	for _, mutation := range mutations {
		mutation.Summary = summary
	}
	return &governancev1.BatchUpsertApprovedDependenciesResponse{
		Mutations: mutations,
		DryRun:    dryRun,
		Changed:   changed,
		Summary:   summary,
		Guidance:  Guidance,
	}, nil
}

func (r *Registry) ApproveObserved(req *governancev1.ApproveObservedDependencyRequest) (*governancev1.DependencyGovernanceDecisionResponse, error) {
	ecosystem := normalize(req.GetEcosystem())
	packageName := strings.TrimSpace(req.GetPackageName())
	if ecosystem == "" || packageName == "" {
		return nil, fmt.Errorf("ecosystem and package_name are required")
	}
	fleet, err := r.ValidateFleet(req.GetPolicyMode())
	if err != nil {
		return nil, err
	}
	usage := findUsageGroup(fleet.GetUsageGroups(), ecosystem, packageName)
	if usage == nil {
		return nil, fmt.Errorf("no observed fleet usage found for %s/%s", ecosystem, packageName)
	}
	group := triageGroupFromUsage(usage, "decision.approve_observed", "approve_observed", "seeding")
	record := proposalRecordFromGroup(group, "approved", req.GetRangeStrategy())
	if req.GetRangePolicy() != "" {
		record.RangePolicy = normalize(req.GetRangePolicy())
	}
	record.Rationale = firstNonEmpty(strings.TrimSpace(req.GetRationale()), fmt.Sprintf("Approved from observed Vrooli usage across %d scenario(s). Reviewer should confirm purpose, license posture, and alternatives.", usage.GetScenarioCount()))
	record.ApprovedBy = strings.TrimSpace(req.GetApprovedBy())
	record.Keywords = trimStrings(append(record.GetKeywords(), "decision-recipe", "approve-observed"))
	mutation, err := r.Upsert(record, req.GetDryRun())
	if err != nil {
		return nil, err
	}
	warnings := decisionWarnings(record, group)
	return &governancev1.DependencyGovernanceDecisionResponse{
		Record:        record,
		Mutation:      mutation,
		EvidenceGroup: group,
		Warnings:      warnings,
		Guidance:      Guidance,
	}, nil
}

func (r *Registry) WidenRange(req *governancev1.WidenApprovedDependencyRangeRequest) (*governancev1.DependencyGovernanceDecisionResponse, error) {
	ecosystem := normalize(req.GetEcosystem())
	packageName := strings.TrimSpace(req.GetPackageName())
	if ecosystem == "" || packageName == "" {
		return nil, fmt.Errorf("ecosystem and package_name are required")
	}
	targetPolicy := firstNonEmpty(normalize(req.GetTargetPolicy()), "major_line")
	if targetPolicy != "major_line" {
		return nil, fmt.Errorf("unsupported target_policy %q; supported value is major_line", targetPolicy)
	}
	records, err := r.loadRecords()
	if err != nil {
		return nil, err
	}
	var existing *governancev1.ApprovedDependencyRecord
	for _, record := range records {
		if recordKey(record.GetEcosystem(), record.GetPackageName()) == recordKey(ecosystem, packageName) {
			existing = record
			break
		}
	}
	if existing == nil {
		return nil, fmt.Errorf("no approved dependency record found for %s/%s", ecosystem, packageName)
	}
	fleet, err := r.ValidateFleet(req.GetPolicyMode())
	if err != nil {
		return nil, err
	}
	usage := findUsageGroup(fleet.GetUsageGroups(), ecosystem, packageName)
	if usage == nil {
		return nil, fmt.Errorf("no observed fleet usage found for %s/%s", ecosystem, packageName)
	}
	group := triageGroupFromUsage(usage, "decision.widen_range", "widen_range", "ranges")
	nextRange, err := observedSingleMajorRange(group.GetObservedVersions())
	if err != nil {
		return nil, err
	}
	record := proto.Clone(existing).(*governancev1.ApprovedDependencyRecord)
	record.VersionRange = nextRange
	record.RangePolicy = "major_line"
	record.Rationale = firstNonEmpty(strings.TrimSpace(req.GetRationale()), existing.GetRationale(), fmt.Sprintf("Widened to the observed %s major line from fleet usage.", nextRange))
	record.ApprovedBy = firstNonEmpty(strings.TrimSpace(req.GetApprovedBy()), existing.GetApprovedBy())
	record.Keywords = trimStrings(append(record.GetKeywords(), "decision-recipe", "widen-range"))
	record.ExampleScenarios = trimStrings(append(record.GetExampleScenarios(), group.GetScenarios()...))
	mutation, err := r.Upsert(record, req.GetDryRun())
	if err != nil {
		return nil, err
	}
	warnings := decisionWarnings(record, group)
	return &governancev1.DependencyGovernanceDecisionResponse{
		Record:        record,
		Mutation:      mutation,
		EvidenceGroup: group,
		Warnings:      warnings,
		Guidance:      Guidance,
	}, nil
}

func (r *Registry) PreviewVulnerabilityRemediation(ctx context.Context, req *governancev1.PreviewVulnerabilityRemediationRequest) (*governancev1.VulnerabilityRemediationResponse, error) {
	ecosystem := normalize(req.GetEcosystem())
	packageName := strings.TrimSpace(req.GetPackageName())
	vulnerabilityID := strings.TrimSpace(req.GetVulnerabilityId())
	if ecosystem == "" || packageName == "" || vulnerabilityID == "" {
		return nil, fmt.Errorf("ecosystem, package_name, and vulnerability_id are required")
	}
	evidence, found, err := r.securityVulnerabilityEvidence(ctx, ecosystem, packageName, vulnerabilityID)
	if err != nil {
		return nil, err
	}
	if !found {
		return &governancev1.VulnerabilityRemediationResponse{
			Found:    false,
			Guidance: Guidance,
		}, nil
	}
	record := securityDeniedRecord(evidence, "", "", "")
	return &governancev1.VulnerabilityRemediationResponse{
		Found:             true,
		Vulnerability:     evidence,
		SuggestedRecord:   record,
		AffectedScenarios: append([]string{}, evidenceScenarios(evidence)...),
		SourceFiles:       append([]string{}, evidenceSourceFiles(evidence)...),
		Remediation:       remediationForEvidence(evidence, record.GetVersionRange()),
		Guidance:          Guidance,
	}, nil
}

func (r *Registry) DenyVulnerableDependency(ctx context.Context, req *governancev1.DenyVulnerableDependencyRequest) (*governancev1.VulnerabilityRemediationResponse, error) {
	ecosystem := normalize(req.GetEcosystem())
	packageName := strings.TrimSpace(req.GetPackageName())
	vulnerabilityID := strings.TrimSpace(req.GetVulnerabilityId())
	if ecosystem == "" || packageName == "" || vulnerabilityID == "" {
		return nil, fmt.Errorf("ecosystem, package_name, and vulnerability_id are required")
	}
	evidence, found, err := r.securityVulnerabilityEvidence(ctx, ecosystem, packageName, vulnerabilityID)
	if err != nil {
		return nil, err
	}
	if !found {
		return &governancev1.VulnerabilityRemediationResponse{
			Found:    false,
			Guidance: Guidance,
		}, nil
	}
	record := securityDeniedRecord(evidence, req.GetAffectedRange(), req.GetFixedRange(), req.GetRationale())
	if req.GetApprovedBy() != "" {
		record.ApprovedBy = strings.TrimSpace(req.GetApprovedBy())
	}
	mutation, err := r.Upsert(record, req.GetDryRun())
	if err != nil {
		return nil, err
	}
	return &governancev1.VulnerabilityRemediationResponse{
		Found:             true,
		Vulnerability:     evidence,
		SuggestedRecord:   record,
		Mutation:          mutation,
		AffectedScenarios: append([]string{}, evidenceScenarios(evidence)...),
		SourceFiles:       append([]string{}, evidenceSourceFiles(evidence)...),
		Remediation:       remediationForEvidence(evidence, record.GetVersionRange()),
		Guidance:          Guidance,
	}, nil
}

func (r *Registry) securityVulnerabilityEvidence(ctx context.Context, ecosystem, packageName, vulnerabilityID string) (*governancev1.SecurityVulnerabilityEvidence, bool, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(timeoutCtx, "security-health", "deps", "explain", vulnerabilityID, "--ecosystem", ecosystem, "--package", packageName, "--json")
	if r.repoRoot != "" {
		cmd.Dir = r.repoRoot
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, false, fmt.Errorf("query Security Health vulnerability evidence: %w: %s", err, strings.TrimSpace(string(output)))
	}
	var resp securityHealthExplainResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, false, fmt.Errorf("parse Security Health vulnerability evidence: %w", err)
	}
	if !resp.Found {
		return nil, false, nil
	}
	evidence := securityVulnerabilityToProto(resp.Vulnerability)
	if evidence.GetVulnerabilityId() == "" {
		evidence.VulnerabilityId = vulnerabilityID
	}
	if evidence.GetEcosystem() == "" {
		evidence.Ecosystem = ecosystem
	}
	if evidence.GetPackageName() == "" {
		evidence.PackageName = packageName
	}
	return evidence, true, nil
}

func (r *Registry) securityVulnerabilityEvidenceList(ctx context.Context, req *governancev1.ListSecurityGovernanceGapsRequest) ([]*governancev1.SecurityVulnerabilityEvidence, int32, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	args := []string{"deps", "vulnerabilities", "--json"}
	if req.GetEcosystem() != "" {
		args = append(args, "--ecosystem", req.GetEcosystem())
	}
	if req.GetPackageName() != "" {
		args = append(args, req.GetPackageName())
	}
	if req.GetScenario() != "" {
		args = append(args, "--scenario", req.GetScenario())
	}
	if req.GetVulnerabilityId() != "" {
		args = append(args, "--vulnerability", req.GetVulnerabilityId())
	}
	if req.GetLimit() > 0 {
		args = append(args, "--limit", fmt.Sprintf("%d", req.GetLimit()))
	}
	cmd := exec.CommandContext(timeoutCtx, "security-health", args...)
	if r.repoRoot != "" {
		cmd.Dir = r.repoRoot
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, 0, fmt.Errorf("query Security Health vulnerabilities: %w: %s", err, strings.TrimSpace(string(output)))
	}
	var resp securityHealthListResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, 0, fmt.Errorf("parse Security Health vulnerabilities: %w", err)
	}
	out := make([]*governancev1.SecurityVulnerabilityEvidence, 0, len(resp.Vulnerabilities))
	for _, raw := range resp.Vulnerabilities {
		evidence := securityVulnerabilityToProto(raw)
		if evidence.GetEcosystem() == "" && req.GetEcosystem() != "" {
			evidence.Ecosystem = normalizeSecurityEcosystem(req.GetEcosystem())
		}
		if evidence.GetPackageName() == "" && req.GetPackageName() != "" {
			evidence.PackageName = req.GetPackageName()
		}
		out = append(out, evidence)
	}
	return out, resp.Total, nil
}

func securityVulnerabilityToProto(v securityHealthVulnerability) *governancev1.SecurityVulnerabilityEvidence {
	affected := firstRanges(v.AffectedRanges, v.AffectedRangesCamel)
	fixed := firstRanges(v.FixedRanges, v.FixedRangesCamel)
	return &governancev1.SecurityVulnerabilityEvidence{
		VulnerabilityId:    firstNonEmpty(v.VulnerabilityID, v.VulnerabilityIDCamel),
		Aliases:            trimStrings(v.Aliases),
		Ecosystem:          normalizeSecurityEcosystem(v.Ecosystem),
		PackageName:        strings.TrimSpace(v.Name),
		ObservedVersion:    strings.TrimSpace(v.Version),
		AffectedRanges:     securityRangesToProto(affected),
		FixedRanges:        securityRangesToProto(fixed),
		Severity:           strings.TrimSpace(v.Severity),
		NormalizedSeverity: strings.TrimSpace(firstNonEmpty(v.NormalizedSeverity, v.NormalizedCamel)),
		AdvisoryUrl:        strings.TrimSpace(firstNonEmpty(v.AdvisoryURL, v.AdvisoryURLCamel)),
		Summary:            strings.TrimSpace(v.Summary),
		Source:             normalizeSecurityToken(v.Source),
		Reachability:       normalizeSecurityToken(v.Reachability),
		Confidence:         normalizeSecurityToken(v.Confidence),
		Production:         v.Production,
		DevOnly:            v.DevOnly || v.DevOnlyCamel,
		Remediation:        strings.TrimSpace(v.Remediation),
		Scenarios:          trimStrings(v.Scenarios),
		SourceFiles:        trimStrings(firstStrings(v.SourceFiles, v.SourceFilesCamel)),
	}
}

func securityRangesToProto(ranges []securityHealthVersionRange) []*governancev1.SecurityVersionRange {
	out := make([]*governancev1.SecurityVersionRange, 0, len(ranges))
	for _, r := range ranges {
		out = append(out, &governancev1.SecurityVersionRange{
			Range:        strings.TrimSpace(r.Range),
			Version:      strings.TrimSpace(r.Version),
			Introduced:   strings.TrimSpace(r.Introduced),
			Fixed:        strings.TrimSpace(r.Fixed),
			LastAffected: strings.TrimSpace(firstNonEmpty(r.LastAffected, r.LastCamel)),
		})
	}
	return out
}

func securityRangeStrings(ranges []*governancev1.SecurityVersionRange) []string {
	out := make([]string, 0, len(ranges))
	for _, r := range ranges {
		out = append(out, securityAffectedRangeString(r))
	}
	return trimStrings(out)
}

func securityFixedRangeStrings(ranges []*governancev1.SecurityVersionRange) []string {
	out := make([]string, 0, len(ranges))
	for _, r := range ranges {
		out = append(out, firstNonEmpty(strings.TrimSpace(r.GetRange()), strings.TrimSpace(r.GetVersion()), strings.TrimSpace(r.GetFixed())))
	}
	return trimStrings(out)
}

func securityDeniedRecordCovers(record *governancev1.ApprovedDependencyRecord, evidence *governancev1.SecurityVulnerabilityEvidence) bool {
	if record == nil || evidence == nil {
		return false
	}
	if normalize(record.GetState()) != "denied" || normalize(record.GetRangePolicy()) != "security_denied" {
		return false
	}
	return evaluateVersionPolicy(evidence.GetEcosystem(), evidence.GetObservedVersion(), record.GetVersionRange(), record.GetRangePolicy()).Allowed
}

func approvedRecordOverlapsVulnerability(record *governancev1.ApprovedDependencyRecord, evidence *governancev1.SecurityVulnerabilityEvidence) bool {
	if record == nil || evidence == nil {
		return false
	}
	switch normalize(record.GetState()) {
	case "approved", "approved_with_constraints":
	default:
		return false
	}
	return evaluateVersionPolicy(evidence.GetEcosystem(), evidence.GetObservedVersion(), record.GetVersionRange(), record.GetRangePolicy()).Allowed
}

func securityGapSignalCategory(evidence *governancev1.SecurityVulnerabilityEvidence) string {
	if evidence.GetProduction() {
		return "security_vulnerable"
	}
	if evidence.GetDevOnly() {
		return "direct_dev"
	}
	switch normalize(evidence.GetReachability()) {
	case "reachable":
		return "security_vulnerable"
	case "lockfile_affected":
		return "lockfile_transitive"
	default:
		return "security_vulnerable"
	}
}

func securityGapCommand(evidence *governancev1.SecurityVulnerabilityEvidence) string {
	return fmt.Sprintf(
		"scenario-dependency-analyzer deps approved deny-vulnerable %s/%s --vulnerability %s --json",
		evidence.GetEcosystem(),
		evidence.GetPackageName(),
		evidence.GetVulnerabilityId(),
	)
}

func securityDeniedRecord(evidence *governancev1.SecurityVulnerabilityEvidence, affectedRangeOverride, fixedRangeOverride, rationaleOverride string) *governancev1.ApprovedDependencyRecord {
	affectedRange := firstNonEmpty(strings.TrimSpace(affectedRangeOverride), matchingSecurityAffectedRange(evidence), evidence.GetObservedVersion(), "*")
	fixedRange := firstNonEmpty(strings.TrimSpace(fixedRangeOverride), securityFixedRangeForAffectedRange(evidence, affectedRange), "a fixed version outside the affected range")
	rationale := strings.TrimSpace(rationaleOverride)
	if rationale == "" {
		rationale = fmt.Sprintf("%s is affected by %s according to Security Health evidence.", evidence.GetPackageName(), evidence.GetVulnerabilityId())
	}
	securityNotes := []string{
		"security-health vulnerability evidence",
		"vulnerability=" + evidence.GetVulnerabilityId(),
	}
	if evidence.GetSource() != "" {
		securityNotes = append(securityNotes, "source="+evidence.GetSource())
	}
	if evidence.GetConfidence() != "" {
		securityNotes = append(securityNotes, "confidence="+evidence.GetConfidence())
	}
	if evidence.GetReachability() != "" {
		securityNotes = append(securityNotes, "reachability="+evidence.GetReachability())
	}
	if evidence.GetAdvisoryUrl() != "" {
		securityNotes = append(securityNotes, "advisory="+evidence.GetAdvisoryUrl())
	}
	return &governancev1.ApprovedDependencyRecord{
		Ecosystem:        evidence.GetEcosystem(),
		PackageName:      evidence.GetPackageName(),
		VersionRange:     affectedRange,
		RangePolicy:      "security_denied",
		State:            "denied",
		Rationale:        rationale,
		ApprovedDate:     time.Now().UTC().Format("2006-01-02"),
		LastReviewed:     time.Now().UTC().Format("2006-01-02"),
		SecurityNotes:    strings.Join(securityNotes, "; "),
		Replacement:      "Update to " + fixedRange + " or record a reviewed exception with expiry if the evidence is not applicable.",
		ExampleScenarios: evidence.GetScenarios(),
		Keywords:         trimStrings(append([]string{"security", "vulnerability", evidence.GetVulnerabilityId()}, evidence.GetAliases()...)),
	}
}

func matchingSecurityAffectedRange(evidence *governancev1.SecurityVulnerabilityEvidence) string {
	observed := strings.TrimSpace(evidence.GetObservedVersion())
	if observed == "" {
		return firstSecurityAffectedRange(evidence)
	}
	if paired := matchingSecurityAffectedInterval(evidence.GetEcosystem(), observed, evidence.GetAffectedRanges()); paired != "" {
		return paired
	}
	if len(evidence.GetAffectedRanges()) == 1 {
		candidate := securityAffectedRangeString(evidence.GetAffectedRanges()[0])
		if candidate != "" && evaluateVersionPolicy(evidence.GetEcosystem(), observed, candidate, "security_denied").Allowed {
			return candidate
		}
	}
	for _, r := range evidence.GetAffectedRanges() {
		candidate := securityAffectedRangeString(r)
		if candidate == "" {
			continue
		}
		if isLowerBoundRange(candidate) && evaluateVersionPolicy(evidence.GetEcosystem(), observed, candidate, "security_denied").Allowed {
			return candidate
		}
	}
	if evaluateVersionPolicy(evidence.GetEcosystem(), observed, observed, "exact").Allowed {
		return observed
	}
	return firstSecurityAffectedRange(evidence)
}

func matchingSecurityAffectedInterval(ecosystem, observed string, ranges []*governancev1.SecurityVersionRange) string {
	var introduced string
	for _, r := range ranges {
		if value := strings.TrimSpace(r.GetIntroduced()); value != "" {
			introduced = value
			continue
		}
		candidate := securityAffectedRangeString(r)
		if isLowerBoundRange(candidate) {
			introduced = rangeBoundVersion(candidate)
			continue
		}
		fixed := firstNonEmpty(strings.TrimSpace(r.GetFixed()), fixedVersionFromRange(candidate))
		if introduced == "" || fixed == "" {
			continue
		}
		interval := ">=" + introduced + " <" + fixed
		if evaluateVersionPolicy(ecosystem, observed, interval, "security_denied").Allowed {
			return interval
		}
	}
	return ""
}

func securityAffectedRangeString(r *governancev1.SecurityVersionRange) string {
	if strings.TrimSpace(r.GetRange()) != "" {
		return strings.TrimSpace(r.GetRange())
	}
	if strings.TrimSpace(r.GetLastAffected()) != "" {
		return "<= " + strings.TrimSpace(r.GetLastAffected())
	}
	return ""
}

func firstSecurityAffectedRange(evidence *governancev1.SecurityVulnerabilityEvidence) string {
	for _, r := range evidence.GetAffectedRanges() {
		if candidate := securityAffectedRangeString(r); candidate != "" {
			return candidate
		}
	}
	return ""
}

func isLowerBoundRange(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, ">=") || strings.HasPrefix(value, ">")
}

func fixedVersionFromRange(value string) string {
	value = strings.TrimSpace(value)
	for _, prefix := range []string{"<=", "<"} {
		if strings.HasPrefix(value, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(value, prefix))
		}
	}
	return ""
}

func rangeBoundVersion(value string) string {
	value = strings.TrimSpace(value)
	for _, prefix := range []string{">=", ">", "<=", "<"} {
		if strings.HasPrefix(value, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(value, prefix))
		}
	}
	return value
}

func firstSecurityFixedRange(evidence *governancev1.SecurityVulnerabilityEvidence) string {
	for _, r := range evidence.GetFixedRanges() {
		if strings.TrimSpace(r.GetRange()) != "" {
			return strings.TrimSpace(r.GetRange())
		}
		if strings.TrimSpace(r.GetVersion()) != "" {
			return ">= " + strings.TrimSpace(r.GetVersion())
		}
		if strings.TrimSpace(r.GetFixed()) != "" {
			return ">= " + strings.TrimSpace(r.GetFixed())
		}
	}
	return ""
}

func securityFixedRangeForAffectedRange(evidence *governancev1.SecurityVulnerabilityEvidence, affectedRange string) string {
	if upperBound := upperBoundFromRange(affectedRange); upperBound != "" {
		return ">= " + upperBound
	}
	return firstSecurityFixedRange(evidence)
}

func upperBoundFromRange(value string) string {
	for _, token := range splitConstraintTokens(value) {
		token = strings.TrimSpace(token)
		for _, prefix := range []string{"<=", "<"} {
			if strings.HasPrefix(token, prefix) {
				return strings.TrimSpace(strings.TrimPrefix(token, prefix))
			}
		}
	}
	return ""
}

func remediationForEvidence(evidence *governancev1.SecurityVulnerabilityEvidence, affectedRange string) string {
	fixedRange := firstNonEmpty(securityFixedRangeForAffectedRange(evidence, affectedRange), "a fixed version outside the affected range")
	return fmt.Sprintf("Deny %s/%s versions matching %s because of %s, then update affected scenarios to %s or record a reviewed exception with expiry.", evidence.GetEcosystem(), evidence.GetPackageName(), firstNonEmpty(affectedRange, "the affected range"), evidence.GetVulnerabilityId(), fixedRange)
}

func evidenceScenarios(evidence *governancev1.SecurityVulnerabilityEvidence) []string {
	return trimStrings(evidence.GetScenarios())
}

func evidenceSourceFiles(evidence *governancev1.SecurityVulnerabilityEvidence) []string {
	return trimStrings(evidence.GetSourceFiles())
}

func firstRanges(primary, fallback []securityHealthVersionRange) []securityHealthVersionRange {
	if len(primary) > 0 {
		return primary
	}
	return fallback
}

func firstStrings(primary, fallback []string) []string {
	if len(primary) > 0 {
		return primary
	}
	return fallback
}

func normalizeSecurityEcosystem(value string) string {
	value = normalizeSecurityToken(value)
	value = strings.TrimPrefix(value, "ecosystem_")
	if value == "unspecified" {
		return ""
	}
	return value
}

func normalizeSecurityToken(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "EVIDENCE_CONFIDENCE_")
	value = strings.TrimPrefix(value, "VULNERABILITY_SOURCE_")
	value = strings.TrimPrefix(value, "REACHABILITY_")
	return strings.ToLower(strings.ReplaceAll(value, "-", "_"))
}

func (r *Registry) loadRecords() ([]*governancev1.ApprovedDependencyRecord, error) {
	loaded, err := r.loadRegistry()
	if err != nil {
		return nil, err
	}
	return loaded.records, nil
}

func (r *Registry) loadRegistry() (*loadedRegistry, error) {
	raw, err := r.loadRegistryFile()
	if err != nil {
		return nil, err
	}
	if err := validateRegistryFile(raw); err != nil {
		return nil, err
	}
	records := make([]*governancev1.ApprovedDependencyRecord, 0, len(raw.Records))
	for _, record := range raw.Records {
		records = append(records, record.toProto())
	}
	sortRecords(records)
	return &loadedRegistry{records: records, policy: raw.Policy}, nil
}

func (r *Registry) loadRegistryFile() (registryFile, error) {
	data, err := os.ReadFile(r.registryPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return registryFile{SchemaVersion: "1", Policy: registryPolicy{Mode: "advisory"}}, nil
		}
		return registryFile{}, fmt.Errorf("read approved dependency registry: %w", err)
	}
	var raw registryFile
	if err := json.Unmarshal(data, &raw); err != nil {
		return registryFile{}, fmt.Errorf("parse approved dependency registry: %w", err)
	}
	normalizeRegistryFileDefaults(&raw)
	return raw, nil
}

func (r *Registry) loadRegistryFileForMutation() (registryFile, error) {
	raw, err := r.loadRegistryFile()
	if err != nil {
		return registryFile{}, err
	}
	if err := validateRegistryFile(raw); err != nil {
		return registryFile{}, err
	}
	return raw, nil
}

func validateRegistryFile(raw registryFile) error {
	if raw.SchemaVersion != "" && raw.SchemaVersion != "1" {
		return fmt.Errorf("approved dependency registry schema_version %q is not supported", raw.SchemaVersion)
	}
	if raw.Policy.Mode == "" {
		raw.Policy.Mode = "advisory"
	}
	if !validPolicyMode(raw.Policy.Mode) {
		return fmt.Errorf("approved dependency registry policy.mode %q is not supported", raw.Policy.Mode)
	}
	seen := map[string]struct{}{}
	for _, record := range raw.Records {
		protoRecord := record.toProto()
		key := recordKey(protoRecord.GetEcosystem(), protoRecord.GetPackageName())
		if _, ok := seen[key]; ok {
			return fmt.Errorf("duplicate approved dependency record %s", key)
		}
		seen[key] = struct{}{}
		if err := validateRecord(protoRecord); err != nil {
			return err
		}
	}
	return nil
}

func normalizeRegistryFileDefaults(raw *registryFile) {
	if raw == nil {
		return
	}
	if raw.SchemaVersion == "" {
		raw.SchemaVersion = "1"
	}
	if raw.Policy.Mode == "" {
		raw.Policy.Mode = "advisory"
	}
}

func (r *Registry) writeRegistryFile(raw registryFile) error {
	path := r.registryPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("prepare approved dependency registry directory: %w", err)
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("encode approved dependency registry: %w", err)
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".approved-dependencies-*.json")
	if err != nil {
		return fmt.Errorf("create approved dependency registry temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write approved dependency registry temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close approved dependency registry temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace approved dependency registry: %w", err)
	}
	return nil
}

func (r *Registry) registryPath() string {
	return filepath.Join(r.repoRoot, ".vrooli", "dependencies", "approved-dependencies.json")
}

func (r approvedDependencyRecord) toProto() *governancev1.ApprovedDependencyRecord {
	return &governancev1.ApprovedDependencyRecord{
		Ecosystem:        normalize(r.Ecosystem),
		PackageName:      strings.TrimSpace(r.PackageName),
		VersionRange:     strings.TrimSpace(r.VersionRange),
		State:            normalize(r.State),
		UseCases:         trimStrings(r.UseCases),
		Rationale:        strings.TrimSpace(r.Rationale),
		ApprovedBy:       strings.TrimSpace(r.ApprovedBy),
		ApprovedDate:     strings.TrimSpace(r.ApprovedDate),
		LastReviewed:     strings.TrimSpace(r.LastReviewed),
		ReviewExpires:    strings.TrimSpace(r.ReviewExpires),
		LicenseNotes:     strings.TrimSpace(r.LicenseNotes),
		SecurityNotes:    strings.TrimSpace(r.SecurityNotes),
		ExampleScenarios: trimStrings(r.ExampleScenarios),
		Replacement:      strings.TrimSpace(r.Replacement),
		Keywords:         trimStrings(r.Keywords),
		AllowedScenarios: trimStrings(r.AllowedScenarios),
		DeniedScenarios:  trimStrings(r.DeniedScenarios),
		RangePolicy:      normalize(r.RangePolicy),
	}
}

func protoRecordToJSON(record *governancev1.ApprovedDependencyRecord) approvedDependencyRecord {
	if record == nil {
		return approvedDependencyRecord{}
	}
	return approvedDependencyRecord{
		Ecosystem:        normalize(record.GetEcosystem()),
		PackageName:      strings.TrimSpace(record.GetPackageName()),
		VersionRange:     strings.TrimSpace(record.GetVersionRange()),
		State:            normalize(record.GetState()),
		UseCases:         trimStrings(record.GetUseCases()),
		Rationale:        strings.TrimSpace(record.GetRationale()),
		ApprovedBy:       strings.TrimSpace(record.GetApprovedBy()),
		ApprovedDate:     strings.TrimSpace(record.GetApprovedDate()),
		LastReviewed:     strings.TrimSpace(record.GetLastReviewed()),
		ReviewExpires:    strings.TrimSpace(record.GetReviewExpires()),
		LicenseNotes:     strings.TrimSpace(record.GetLicenseNotes()),
		SecurityNotes:    strings.TrimSpace(record.GetSecurityNotes()),
		ExampleScenarios: trimStrings(record.GetExampleScenarios()),
		Replacement:      strings.TrimSpace(record.GetReplacement()),
		Keywords:         trimStrings(record.GetKeywords()),
		AllowedScenarios: trimStrings(record.GetAllowedScenarios()),
		DeniedScenarios:  trimStrings(record.GetDeniedScenarios()),
		RangePolicy:      normalize(record.GetRangePolicy()),
	}
}

func ScanScenarioDependencies(scenarioDir string) ([]*governancev1.ObservedDependency, error) {
	var observed []*governancev1.ObservedDependency
	err := filepath.WalkDir(scenarioDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "node_modules", "dist", "build", "coverage", ".turbo":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		switch entry.Name() {
		case "package.json":
			deps, err := scanPackageJSON(path)
			if err != nil {
				return err
			}
			for _, dep := range deps {
				dep.SurfaceId = inferSurfaceID(path)
			}
			observed = append(observed, deps...)
		case "go.mod":
			deps, err := scanGoMod(path)
			if err != nil {
				return err
			}
			for _, dep := range deps {
				dep.SurfaceId = inferSurfaceID(path)
			}
			observed = append(observed, deps...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(observed, func(i, j int) bool {
		return observed[i].GetEcosystem()+observed[i].GetPackageName()+observed[i].GetFilePath() < observed[j].GetEcosystem()+observed[j].GetPackageName()+observed[j].GetFilePath()
	})
	return observed, nil
}

func ScanSurfaceDependencies(surfaces []Surface) []*governancev1.ObservedDependency {
	var observed []*governancev1.ObservedDependency
	seen := map[string]struct{}{}
	for _, surface := range surfaces {
		root := strings.TrimSpace(surface.RootPath)
		if root == "" {
			continue
		}
		var deps []*governancev1.ObservedDependency
		switch normalize(surface.Language) {
		case "go":
			deps, _ = scanGoMod(filepath.Join(root, "go.mod"))
		case "javascript", "typescript":
			deps, _ = scanPackageJSON(filepath.Join(root, "package.json"))
		}
		for _, dep := range deps {
			dep.SurfaceId = surface.ID
			key := dep.GetEcosystem() + "\x00" + dep.GetPackageName() + "\x00" + dep.GetFilePath() + "\x00" + dep.GetDependencyGroup()
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			observed = append(observed, dep)
		}
	}
	sort.Slice(observed, func(i, j int) bool {
		return observed[i].GetEcosystem()+observed[i].GetPackageName()+observed[i].GetSurfaceId() < observed[j].GetEcosystem()+observed[j].GetPackageName()+observed[j].GetSurfaceId()
	})
	return observed
}

func scanPackageJSON(path string) ([]*governancev1.ObservedDependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var pkg struct {
		Dependencies         map[string]string `json:"dependencies"`
		DevDependencies      map[string]string `json:"devDependencies"`
		PeerDependencies     map[string]string `json:"peerDependencies"`
		OptionalDependencies map[string]string `json:"optionalDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("parse package.json %s: %w", path, err)
	}
	var out []*governancev1.ObservedDependency
	for _, group := range []struct {
		name string
		deps map[string]string
	}{
		{name: "dependencies", deps: pkg.Dependencies},
		{name: "devDependencies", deps: pkg.DevDependencies},
		{name: "peerDependencies", deps: pkg.PeerDependencies},
		{name: "optionalDependencies", deps: pkg.OptionalDependencies},
	} {
		for name, version := range group.deps {
			out = append(out, &governancev1.ObservedDependency{
				Ecosystem:       "npm",
				PackageName:     strings.TrimSpace(name),
				Version:         strings.TrimSpace(version),
				FilePath:        filepath.ToSlash(path),
				DependencyGroup: group.name,
			})
		}
	}
	return out, nil
}

func scanGoMod(path string) ([]*governancev1.ObservedDependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []*governancev1.ObservedDependency
	inRequire := false
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if inRequire {
			if line == ")" {
				inRequire = false
				continue
			}
			if dep := parseGoRequire(line, path); dep != nil {
				out = append(out, dep)
			}
			continue
		}
		if strings.HasPrefix(line, "require (") {
			inRequire = true
			continue
		}
		if strings.HasPrefix(line, "require ") {
			if dep := parseGoRequire(strings.TrimPrefix(line, "require "), path); dep != nil {
				out = append(out, dep)
			}
		}
	}
	return out, nil
}

func parseGoRequire(line, path string) *governancev1.ObservedDependency {
	group := "require"
	parts := strings.SplitN(line, "//", 2)
	line = parts[0]
	if len(parts) == 2 && strings.Contains(parts[1], "indirect") {
		group = "require_indirect"
	}
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return nil
	}
	return &governancev1.ObservedDependency{
		Ecosystem:       "go",
		PackageName:     fields[0],
		Version:         fields[1],
		FilePath:        filepath.ToSlash(path),
		DependencyGroup: group,
	}
}

func (r *Registry) discoverScenarios() ([]string, error) {
	scenariosRoot := filepath.Join(r.repoRoot, "scenarios")
	entries, err := os.ReadDir(scenariosRoot)
	if err != nil {
		return nil, fmt.Errorf("read scenarios directory: %w", err)
	}
	scenarios := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if _, err := os.Stat(filepath.Join(scenariosRoot, entry.Name(), ".vrooli", "service.json")); err == nil {
			scenarios = append(scenarios, entry.Name())
		}
	}
	sort.Strings(scenarios)
	return scenarios, nil
}

func (r *Registry) effectivePolicyMode(policyModeOverride ...string) string {
	loaded, err := r.loadRegistry()
	if err != nil {
		return "advisory"
	}
	mode := normalize(firstNonEmpty(append(policyModeOverride, loaded.policy.Mode, "advisory")...))
	if !validPolicyMode(mode) {
		return "advisory"
	}
	return mode
}

func degradedScenarioResponse(scenario string, err error, policyMode string) *governancev1.ApprovedDependencyValidationResponse {
	finding := &governancev1.ApprovedDependencyFinding{
		Id:           "governance." + slug(scenario+".scan.degraded"),
		Scenario:     scenario,
		Severity:     "ERROR",
		Title:        "Scenario dependency scan failed",
		Description:  err.Error(),
		Remediation:  "Fix the scenario dependency files or scanner error, then rerun fleet validation.",
		FindingClass: "SCAN_DEGRADED",
		PolicyMode:   policyMode,
	}
	return &governancev1.ApprovedDependencyValidationResponse{
		Scenario: scenario,
		Passed:   false,
		Summary: &governancev1.DependencyGovernanceSummary{
			Status:       "fail",
			PolicyMode:   policyMode,
			FindingCount: 1,
			ErrorCount:   1,
		},
		Findings: []*governancev1.ApprovedDependencyFinding{finding},
		Guidance: Guidance,
	}
}

func mergeSummary(target, source *governancev1.DependencyGovernanceSummary) {
	if target == nil || source == nil {
		return
	}
	target.Unrecorded += source.GetUnrecorded()
	target.Observed += source.GetObserved()
	target.OutOfRange += source.GetOutOfRange()
	target.OutOfScope += source.GetOutOfScope()
	target.Expired += source.GetExpired()
	target.FindingCount += source.GetFindingCount()
	target.ErrorCount += source.GetErrorCount()
	target.WarningCount += source.GetWarningCount()
	target.InfoCount += source.GetInfoCount()
}

func summarizeFindings(findings []*governancev1.ApprovedDependencyFinding, policyMode string) *governancev1.DependencyGovernanceSummary {
	summary := &governancev1.DependencyGovernanceSummary{
		PolicyMode:   firstNonEmpty(policyMode, "advisory"),
		FindingCount: int32(len(findings)),
	}
	scenarios := map[string]struct{}{}
	dependencies := map[string]struct{}{}
	for _, finding := range findings {
		if finding.GetScenario() != "" {
			scenarios[finding.GetScenario()] = struct{}{}
		}
		key := recordKey(finding.GetEcosystem(), finding.GetPackageName())
		if key != "/" {
			dependencies[key] = struct{}{}
		}
		switch finding.GetFindingClass() {
		case "UNRECORDED_DIRECT":
			summary.Unrecorded++
		case "DENIED_IN_USE", "SECURITY_AFFECTED_RANGE_DENIED":
			summary.Denied++
		case "DEPRECATED_IN_USE":
			summary.Deprecated++
		case "VERSION_OUT_OF_RANGE", "SECURITY_VULNERABLE_VERSION":
			summary.OutOfRange++
		case "SCENARIO_EXCEPTION_VIOLATION":
			summary.OutOfScope++
		case "EXPIRED_APPROVAL", "EXPIRED_EXCEPTION":
			summary.Expired++
		}
		switch strings.ToUpper(finding.GetSeverity()) {
		case "ERROR", "BLOCKER":
			summary.ErrorCount++
		case "WARNING":
			summary.WarningCount++
		default:
			summary.InfoCount++
		}
	}
	summary.ScenarioCount = int32(len(scenarios))
	summary.DependencyCount = int32(len(dependencies))
	status, _ := statusFromFindings(findings, false)
	summary.Status = status
	return summary
}

func buildUsageGroups(responses []*governancev1.ApprovedDependencyValidationResponse) []*governancev1.DependencyUsageGroup {
	groups := map[string]*governancev1.DependencyUsageGroup{}
	scenariosByKey := map[string]map[string]struct{}{}
	findingsByKey := map[string]int32{}
	severityByKey := map[string]string{}
	for _, resp := range responses {
		for _, finding := range resp.GetFindings() {
			key := recordKey(finding.GetEcosystem(), finding.GetPackageName())
			if key == "/" {
				continue
			}
			findingsByKey[key]++
			severityByKey[key] = maxSeverity(severityByKey[key], finding.GetSeverity())
		}
		for _, dep := range resp.GetObservedDependencies() {
			key := recordKey(dep.GetEcosystem(), dep.GetPackageName())
			group := groups[key]
			if group == nil {
				group = &governancev1.DependencyUsageGroup{
					Ecosystem:   dep.GetEcosystem(),
					PackageName: dep.GetPackageName(),
				}
				groups[key] = group
				scenariosByKey[key] = map[string]struct{}{}
			}
			group.UsageCount++
			annotateDependencySignalCategory(dep)
			group.ObservedDependencies = append(group.GetObservedDependencies(), dep)
			scenariosByKey[key][resp.GetScenario()] = struct{}{}
		}
	}
	out := make([]*governancev1.DependencyUsageGroup, 0, len(groups))
	for key, group := range groups {
		categories := map[string]struct{}{}
		for _, dep := range group.GetObservedDependencies() {
			if dep.GetSignalCategory() != "" {
				categories[dep.GetSignalCategory()] = struct{}{}
			}
		}
		for scenario := range scenariosByKey[key] {
			group.Scenarios = append(group.GetScenarios(), scenario)
		}
		sort.Strings(group.Scenarios)
		group.SignalCategories = sortedKeys(categories)
		group.ScenarioCount = int32(len(group.GetScenarios()))
		group.FindingCount = findingsByKey[key]
		group.HighestSeverity = firstNonEmpty(severityByKey[key], "INFO")
		out = append(out, group)
	}
	sort.Slice(out, func(i, j int) bool {
		return recordKey(out[i].GetEcosystem(), out[i].GetPackageName()) < recordKey(out[j].GetEcosystem(), out[j].GetPackageName())
	})
	return out
}

type triageAccumulator struct {
	group           *governancev1.DependencyGovernanceTriageGroup
	scenarios       map[string]struct{}
	classes         map[string]struct{}
	versions        map[string]struct{}
	vulnerabilities map[string]struct{}
}

func buildTriageGroups(findings []*governancev1.ApprovedDependencyFinding, req *governancev1.GetApprovedDependencyTriageRequest) []*governancev1.DependencyGovernanceTriageGroup {
	groups := map[string]*triageAccumulator{}
	for _, finding := range findings {
		if !matchesFilter(finding.GetEcosystem(), req.GetEcosystem()) {
			continue
		}
		if req.GetPackageName() != "" && !sameFold(finding.GetPackageName(), req.GetPackageName()) {
			continue
		}
		section, action := triageSectionAndAction(finding.GetFindingClass())
		if !matchesTriageSection(section, req.GetSection()) {
			continue
		}
		groupID := strings.Join([]string{section, action, normalize(finding.GetEcosystem()), strings.ToLower(finding.GetPackageName())}, ".")
		acc := groups[groupID]
		if acc == nil {
			acc = &triageAccumulator{
				group: &governancev1.DependencyGovernanceTriageGroup{
					GroupId:         groupID,
					Title:           triageTitle(section, finding),
					ActionType:      action,
					Section:         section,
					Ecosystem:       finding.GetEcosystem(),
					PackageName:     finding.GetPackageName(),
					HighestSeverity: "INFO",
					Rationale:       triageRationale(section, finding),
				},
				scenarios:       map[string]struct{}{},
				classes:         map[string]struct{}{},
				versions:        map[string]struct{}{},
				vulnerabilities: map[string]struct{}{},
			}
			groups[groupID] = acc
		}
		acc.group.FindingCount++
		acc.group.UsageCount++
		acc.group.HighestSeverity = maxSeverity(acc.group.GetHighestSeverity(), finding.GetSeverity())
		if finding.GetScenario() != "" {
			acc.scenarios[finding.GetScenario()] = struct{}{}
		}
		if finding.GetFindingClass() != "" {
			acc.classes[finding.GetFindingClass()] = struct{}{}
		}
		if finding.GetObserved() != "" {
			acc.versions[finding.GetObserved()] = struct{}{}
		}
		for _, vulnID := range vulnerabilityIDsFromFinding(finding) {
			acc.vulnerabilities[vulnID] = struct{}{}
		}
	}
	out := make([]*governancev1.DependencyGovernanceTriageGroup, 0, len(groups))
	for _, acc := range groups {
		acc.group.Scenarios = sortedKeys(acc.scenarios)
		acc.group.ScenarioCount = int32(len(acc.group.GetScenarios()))
		acc.group.FindingClasses = sortedKeys(acc.classes)
		acc.group.ObservedVersions = sortedKeys(acc.versions)
		acc.group.VulnerabilityIds = sortedKeys(acc.vulnerabilities)
		acc.group.RecommendedCommand = triageCommand(acc.group)
		out = append(out, acc.group)
	}
	sort.Slice(out, func(i, j int) bool {
		left := out[i]
		right := out[j]
		if severityRank(left.GetHighestSeverity()) != severityRank(right.GetHighestSeverity()) {
			return severityRank(left.GetHighestSeverity()) > severityRank(right.GetHighestSeverity())
		}
		if left.GetFindingCount() != right.GetFindingCount() {
			return left.GetFindingCount() > right.GetFindingCount()
		}
		if left.GetScenarioCount() != right.GetScenarioCount() {
			return left.GetScenarioCount() > right.GetScenarioCount()
		}
		return left.GetGroupId() < right.GetGroupId()
	})
	return out
}

func buildProposalGroups(fleet *governancev1.FleetApprovedDependencyValidationResponse, req *governancev1.ProposeApprovedDependencyRecordsRequest) []*governancev1.DependencyGovernanceTriageGroup {
	groups := map[string]*triageAccumulator{}
	usageByKey := map[string]*governancev1.DependencyUsageGroup{}
	for _, usage := range fleet.GetUsageGroups() {
		usageByKey[recordKey(usage.GetEcosystem(), usage.GetPackageName())] = usage
	}
	securityFindings := map[string][]string{}
	for _, finding := range fleet.GetFindings() {
		if section, _ := triageSectionAndAction(finding.GetFindingClass()); section == "security" {
			key := recordKey(finding.GetEcosystem(), finding.GetPackageName())
			securityFindings[key] = append(securityFindings[key], vulnerabilityIDsFromFinding(finding)...)
		}
	}
	for _, finding := range fleet.GetFindings() {
		if finding.GetFindingClass() != "UNRECORDED_DIRECT" {
			continue
		}
		if !matchesFilter(finding.GetScenario(), req.GetScenario()) {
			continue
		}
		if !matchesFilter(finding.GetEcosystem(), req.GetEcosystem()) {
			continue
		}
		if req.GetPackageName() != "" && !sameFold(finding.GetPackageName(), req.GetPackageName()) {
			continue
		}
		key := recordKey(finding.GetEcosystem(), finding.GetPackageName())
		if !proposalDependencyGroupIncluded(usageByKey[key], req) {
			continue
		}
		groupID := "proposal.approve_or_review." + normalize(finding.GetEcosystem()) + "." + strings.ToLower(finding.GetPackageName())
		acc := groups[groupID]
		if acc == nil {
			acc = &triageAccumulator{
				group: &governancev1.DependencyGovernanceTriageGroup{
					GroupId:         groupID,
					Title:           "Draft governance review record for " + finding.GetEcosystem() + "/" + finding.GetPackageName(),
					ActionType:      "propose_record",
					Section:         "seeding",
					Ecosystem:       finding.GetEcosystem(),
					PackageName:     finding.GetPackageName(),
					HighestSeverity: "INFO",
					Rationale:       "This direct dependency is used but has no reviewed governance memory yet.",
				},
				scenarios:       map[string]struct{}{},
				classes:         map[string]struct{}{},
				versions:        map[string]struct{}{},
				vulnerabilities: map[string]struct{}{},
			}
			groups[groupID] = acc
		}
		acc.group.FindingCount++
		acc.group.UsageCount++
		acc.group.HighestSeverity = maxSeverity(acc.group.GetHighestSeverity(), finding.GetSeverity())
		if finding.GetScenario() != "" {
			acc.scenarios[finding.GetScenario()] = struct{}{}
		}
		if finding.GetFindingClass() != "" {
			acc.classes[finding.GetFindingClass()] = struct{}{}
		}
		if finding.GetObserved() != "" {
			acc.versions[finding.GetObserved()] = struct{}{}
		}
		for _, vulnID := range securityFindings[key] {
			acc.vulnerabilities[vulnID] = struct{}{}
		}
	}
	out := make([]*governancev1.DependencyGovernanceTriageGroup, 0, len(groups))
	minScenarios := int(req.GetMinimumScenarioCount())
	for _, acc := range groups {
		acc.group.Scenarios = sortedKeys(acc.scenarios)
		acc.group.ScenarioCount = int32(len(acc.group.GetScenarios()))
		if minScenarios > 0 && int(acc.group.GetScenarioCount()) < minScenarios {
			continue
		}
		acc.group.FindingClasses = sortedKeys(acc.classes)
		acc.group.ObservedVersions = sortedKeys(acc.versions)
		acc.group.VulnerabilityIds = sortedKeys(acc.vulnerabilities)
		acc.group.RecommendedCommand = "scenario-dependency-analyzer deps approved upsert-batch --file proposals.json --dry-run --json"
		out = append(out, acc.group)
	}
	sort.Slice(out, func(i, j int) bool {
		left := out[i]
		right := out[j]
		if left.GetScenarioCount() != right.GetScenarioCount() {
			return left.GetScenarioCount() > right.GetScenarioCount()
		}
		if left.GetFindingCount() != right.GetFindingCount() {
			return left.GetFindingCount() > right.GetFindingCount()
		}
		return left.GetEcosystem()+"/"+left.GetPackageName() < right.GetEcosystem()+"/"+right.GetPackageName()
	})
	return out
}

func proposalDependencyGroupIncluded(group *governancev1.DependencyUsageGroup, req *governancev1.ProposeApprovedDependencyRecordsRequest) bool {
	if group == nil {
		return true
	}
	includeDev := req.GetIncludeDev()
	includeRuntime := req.GetIncludeRuntime()
	if !includeDev && !includeRuntime {
		includeDev = true
		includeRuntime = true
	}
	for _, dep := range group.GetObservedDependencies() {
		if req.GetScenario() != "" && !containsFold(group.GetScenarios(), req.GetScenario()) {
			continue
		}
		category := dependencySignalCategory(dep)
		if includeRuntime && category == "direct_runtime" {
			return true
		}
		if includeDev && category == "direct_dev" {
			return true
		}
	}
	return false
}

func findUsageGroup(groups []*governancev1.DependencyUsageGroup, ecosystem, packageName string) *governancev1.DependencyUsageGroup {
	key := recordKey(ecosystem, packageName)
	for _, group := range groups {
		if recordKey(group.GetEcosystem(), group.GetPackageName()) == key {
			return group
		}
	}
	return nil
}

func triageGroupFromUsage(group *governancev1.DependencyUsageGroup, prefix, action, section string) *governancev1.DependencyGovernanceTriageGroup {
	scenarios := map[string]struct{}{}
	versions := map[string]struct{}{}
	for _, scenario := range group.GetScenarios() {
		scenarios[scenario] = struct{}{}
	}
	for _, dep := range group.GetObservedDependencies() {
		if dep.GetVersion() != "" {
			versions[dep.GetVersion()] = struct{}{}
		}
	}
	out := &governancev1.DependencyGovernanceTriageGroup{
		GroupId:          prefix + "." + normalize(group.GetEcosystem()) + "." + strings.ToLower(group.GetPackageName()),
		Title:            fmt.Sprintf("%s %s/%s from observed usage", action, group.GetEcosystem(), group.GetPackageName()),
		ActionType:       action,
		Section:          section,
		Ecosystem:        group.GetEcosystem(),
		PackageName:      group.GetPackageName(),
		FindingCount:     group.GetFindingCount(),
		ScenarioCount:    group.GetScenarioCount(),
		UsageCount:       group.GetUsageCount(),
		HighestSeverity:  firstNonEmpty(group.GetHighestSeverity(), "INFO"),
		FindingClasses:   []string{"OBSERVED_USAGE"},
		Scenarios:        sortedKeys(scenarios),
		ObservedVersions: sortedKeys(versions),
		Rationale:        "Decision recipe built from current fleet dependency usage.",
	}
	return out
}

func observedSingleMajorRange(versions []string) (string, error) {
	versions = trimStrings(versions)
	if len(versions) == 0 {
		return "", fmt.Errorf("no observed versions are available for range widening")
	}
	major := -1
	selected := ""
	var selectedVersion semanticVersion
	for _, version := range versions {
		parsed, ok := parseVersion(firstVersionToken(version))
		if !ok {
			return "", fmt.Errorf("cannot widen range because observed version %q is not parseable", version)
		}
		if major == -1 {
			major = parsed.Major
			selected = version
			selectedVersion = parsed
			continue
		}
		if parsed.Major != major {
			return "", fmt.Errorf("cannot widen range automatically because observed versions span multiple majors: %s", strings.Join(versions, ", "))
		}
		if compareVersion(parsed, selectedVersion) < 0 {
			selected = version
			selectedVersion = parsed
		}
	}
	return selected, nil
}

func decisionWarnings(record *governancev1.ApprovedDependencyRecord, group *governancev1.DependencyGovernanceTriageGroup) []string {
	warnings := make([]string, 0)
	if record.GetVersionRange() == "*" {
		warnings = append(warnings, fmt.Sprintf("%s/%s uses '*' and should be narrowed before approval when possible", record.GetEcosystem(), record.GetPackageName()))
	}
	if len(group.GetObservedVersions()) > 1 {
		warnings = append(warnings, fmt.Sprintf("%s/%s has multiple observed versions: %s", group.GetEcosystem(), group.GetPackageName(), strings.Join(group.GetObservedVersions(), ", ")))
	}
	return warnings
}

func proposalRecordFromGroup(group *governancev1.DependencyGovernanceTriageGroup, state, rangeStrategy string) *governancev1.ApprovedDependencyRecord {
	versions := trimStrings(group.GetObservedVersions())
	return &governancev1.ApprovedDependencyRecord{
		Ecosystem:        group.GetEcosystem(),
		PackageName:      group.GetPackageName(),
		VersionRange:     proposedVersionRange(versions, rangeStrategy),
		RangePolicy:      proposedRangePolicy(rangeStrategy),
		State:            firstNonEmpty(normalize(state), "needs_review"),
		Rationale:        "Draft record generated from observed Vrooli usage. Reviewer must confirm purpose, alternatives considered, license posture, and security posture before approving.",
		SecurityNotes:    proposalSecurityNotes(group),
		ExampleScenarios: append([]string{}, group.GetScenarios()...),
		Keywords:         trimStrings([]string{"proposal", "governance-review"}),
	}
}

func proposedVersionRange(versions []string, strategy string) string {
	strategy = normalize(strategy)
	if strategy == "wildcard" || len(versions) == 0 {
		return "*"
	}
	if strategy == "exact" && len(versions) > 0 {
		return versions[0]
	}
	if len(versions) == 1 {
		return versions[0]
	}
	return "*"
}

func proposedRangePolicy(strategy string) string {
	switch normalize(strategy) {
	case "exact":
		return "exact"
	case "major", "major_line":
		return "major_line"
	case "minimum":
		return "minimum"
	case "dev", "dev_tooling":
		return "dev_tooling"
	default:
		return ""
	}
}

func proposalSecurityNotes(group *governancev1.DependencyGovernanceTriageGroup) string {
	notes := []string{"Generated proposal; run Security Health review before approving."}
	if len(group.GetVulnerabilityIds()) > 0 {
		notes = append(notes, "security findings present: "+strings.Join(group.GetVulnerabilityIds(), ","))
	}
	return strings.Join(notes, " ")
}

func dependencySignalCategory(dep *governancev1.ObservedDependency) string {
	if dep == nil {
		return "direct_runtime"
	}
	group := normalize(dep.GetDependencyGroup())
	switch {
	case sameFold(dep.GetEcosystem(), "go") && group == "require_indirect":
		return "indirect"
	case group == "devdependencies":
		return "direct_dev"
	case strings.Contains(group, "lockfile"):
		return "lockfile_transitive"
	default:
		return "direct_runtime"
	}
}

func annotateDependencySignalCategory(dep *governancev1.ObservedDependency) {
	if dep == nil || dep.GetSignalCategory() != "" {
		return
	}
	dep.SignalCategory = dependencySignalCategory(dep)
}

func appendLimited(groups []*governancev1.DependencyGovernanceTriageGroup, group *governancev1.DependencyGovernanceTriageGroup, limit int) []*governancev1.DependencyGovernanceTriageGroup {
	if limit > 0 && len(groups) >= limit {
		return groups
	}
	return append(groups, group)
}

func triageSectionAndAction(findingClass string) (string, string) {
	switch findingClass {
	case "DENIED_IN_USE", "SECURITY_AFFECTED_RANGE_DENIED", "SECURITY_VULNERABLE_VERSION":
		return "security", "deny_or_remove"
	case "UNRECORDED_DIRECT":
		return "seeding", "approve_or_review"
	case "VERSION_OUT_OF_RANGE":
		return "ranges", "widen_range"
	case "EXPIRED_APPROVAL", "EXPIRED_EXCEPTION":
		return "expired", "renew_review"
	default:
		return "hotspots", "review"
	}
}

func matchesTriageSection(section, filter string) bool {
	filter = normalize(filter)
	if filter == "" {
		return true
	}
	switch filter {
	case "security", "security_actions":
		return section == "security"
	case "seeding", "registry_seeding":
		return section == "seeding"
	case "ranges", "range_policy":
		return section == "ranges"
	case "hotspots", "scenario_hotspots":
		return section == "hotspots"
	case "expired", "stale_or_expired_reviews":
		return section == "expired"
	default:
		return section == filter
	}
}

func triageTitle(section string, finding *governancev1.ApprovedDependencyFinding) string {
	dep := finding.GetEcosystem() + "/" + finding.GetPackageName()
	switch section {
	case "security":
		return "Resolve blocked or vulnerable dependency usage for " + dep
	case "seeding":
		return "Review common unrecorded dependency " + dep
	case "ranges":
		return "Review approved range drift for " + dep
	case "expired":
		return "Renew expired governance review for " + dep
	default:
		return "Review governance hotspot for " + dep
	}
}

func triageRationale(section string, finding *governancev1.ApprovedDependencyFinding) string {
	switch section {
	case "security":
		return "This dependency has denied or security-sensitive governance findings and should be removed, upgraded, or explicitly remediated."
	case "seeding":
		return "This direct dependency is used but has no reviewed governance memory yet."
	case "ranges":
		return "The dependency is recorded, but observed versions fall outside the reviewed range."
	case "expired":
		return "The dependency decision exists, but its review expiry has passed."
	default:
		return firstNonEmpty(finding.GetDescription(), "This dependency has governance findings that should be reviewed as a grouped operator task.")
	}
}

func triageCommand(group *governancev1.DependencyGovernanceTriageGroup) string {
	dep := group.GetEcosystem() + "/" + group.GetPackageName()
	switch group.GetActionType() {
	case "approve_or_review":
		return "scenario-dependency-analyzer deps approved approve-observed " + dep + " --from-findings --json"
	case "widen_range":
		return "scenario-dependency-analyzer deps approved widen-range " + dep + " --to-major-line --json"
	case "deny_or_remove":
		if len(group.GetVulnerabilityIds()) > 0 {
			return "scenario-dependency-analyzer deps approved deny-vulnerable " + dep + " --vulnerability " + group.GetVulnerabilityIds()[0] + " --json"
		}
		return "scenario-dependency-analyzer deps approved usage " + dep + " --json"
	case "renew_review":
		return "scenario-dependency-analyzer deps approved explain " + dep + " --json"
	default:
		return "scenario-dependency-analyzer deps approved findings --ecosystem " + group.GetEcosystem() + " --package " + group.GetPackageName() + " --json"
	}
}

func vulnerabilityIDsFromFinding(finding *governancev1.ApprovedDependencyFinding) []string {
	values := []string{finding.GetObserved(), finding.GetExpected(), finding.GetDescription(), finding.GetRemediation(), finding.GetTitle()}
	var out []string
	for _, value := range values {
		for _, token := range strings.FieldsFunc(value, func(r rune) bool {
			return r == ' ' || r == ',' || r == ';' || r == '(' || r == ')' || r == '[' || r == ']'
		}) {
			token = strings.Trim(token, ".:")
			upper := strings.ToUpper(token)
			if strings.HasPrefix(upper, "GHSA-") || strings.HasPrefix(upper, "CVE-") || strings.HasPrefix(upper, "GO-") {
				out = append(out, token)
			}
		}
	}
	return trimStrings(out)
}

func sortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func maxSeverity(left, right string) string {
	if severityRank(right) > severityRank(left) {
		return strings.ToUpper(right)
	}
	return strings.ToUpper(left)
}

func severityRank(value string) int {
	rank := map[string]int{
		"":         0,
		"INFO":     1,
		"LOW":      1,
		"WARNING":  2,
		"MODERATE": 2,
		"MEDIUM":   2,
		"ERROR":    3,
		"HIGH":     3,
		"BLOCKER":  4,
		"CRITICAL": 4,
	}
	return rank[strings.ToUpper(value)]
}

func statusFromFindings(findings []*governancev1.ApprovedDependencyFinding, notConfigured bool) (string, bool) {
	if notConfigured && len(findings) == 0 {
		return "not_configured", true
	}
	status := "pass"
	for _, finding := range findings {
		switch strings.ToUpper(finding.GetSeverity()) {
		case "ERROR", "BLOCKER":
			return "fail", false
		case "WARNING":
			status = "warn"
		}
	}
	return status, true
}

func validateRecord(record *governancev1.ApprovedDependencyRecord) error {
	key := recordKey(record.GetEcosystem(), record.GetPackageName())
	if record.GetEcosystem() == "" || record.GetPackageName() == "" {
		return fmt.Errorf("approved dependency record %q must include ecosystem and package_name", key)
	}
	if !validState(record.GetState()) {
		return fmt.Errorf("approved dependency record %s has unsupported state %q", key, record.GetState())
	}
	if !supportedRangePolicy(normalize(record.GetRangePolicy())) {
		return fmt.Errorf("approved dependency record %s has unsupported range_policy %q", key, record.GetRangePolicy())
	}
	if strings.TrimSpace(record.GetRationale()) == "" {
		return fmt.Errorf("approved dependency record %s must include rationale", key)
	}
	state := normalize(record.GetState())
	if (state == "denied" || state == "blocked" || state == "deprecated") && strings.TrimSpace(record.GetReplacement()) == "" && strings.TrimSpace(record.GetRationale()) == "" {
		return fmt.Errorf("approved dependency record %s must include replacement or rationale for %s state", key, state)
	}
	if record.GetReviewExpires() != "" {
		if _, err := time.Parse("2006-01-02", record.GetReviewExpires()); err != nil {
			return fmt.Errorf("approved dependency record %s has invalid review_expires %q", key, record.GetReviewExpires())
		}
	}
	return nil
}

func validState(state string) bool {
	switch normalize(state) {
	case "", "approved", "approved_with_constraints", "needs_review", "blocked", "denied", "deprecated", "exception":
		return true
	default:
		return false
	}
}

func validPolicyMode(mode string) bool {
	switch normalize(mode) {
	case "advisory", "strict", "review_gate":
		return true
	default:
		return false
	}
}

type scenarioException struct {
	reason   string
	severity string
}

func scenarioExceptionViolation(scenario string, record *governancev1.ApprovedDependencyRecord) scenarioException {
	if containsFold(record.GetDeniedScenarios(), scenario) {
		return scenarioException{
			reason:   "This dependency is explicitly denied for the current scenario.",
			severity: "ERROR",
		}
	}
	if len(record.GetAllowedScenarios()) > 0 && !containsFold(record.GetAllowedScenarios(), scenario) {
		return scenarioException{
			reason:   "This dependency is only approved for explicitly listed scenario exceptions, and the current scenario is not listed.",
			severity: "WARNING",
		}
	}
	return scenarioException{}
}

func expired(date string) bool {
	date = strings.TrimSpace(date)
	if date == "" {
		return false
	}
	parsed, err := time.Parse("2006-01-02", date)
	if err != nil {
		return false
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	return parsed.Before(today)
}

func recordScope(record *governancev1.ApprovedDependencyRecord) string {
	parts := []string{}
	if len(record.GetAllowedScenarios()) > 0 {
		parts = append(parts, "scenarios="+strings.Join(record.GetAllowedScenarios(), ","))
	}
	return firstNonEmpty(strings.Join(parts, "; "), "global approval")
}

func inferSurfaceID(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, part := range parts {
		switch part {
		case "api", "cli", "ui", "worker":
			return part
		}
	}
	return ""
}

func isUnrecordedTransitiveDependency(dep *governancev1.ObservedDependency) bool {
	return sameFold(dep.GetEcosystem(), "go") && sameFold(dep.GetDependencyGroup(), "require_indirect")
}

func governanceFinding(scenario string, dep *governancev1.ObservedDependency, findingClass, severity, title, description, remediation, observed, expected, policyMode string) *governancev1.ApprovedDependencyFinding {
	return &governancev1.ApprovedDependencyFinding{
		Id:           "governance." + slug(scenario+"."+dep.GetEcosystem()+"."+dep.GetPackageName()+"."+findingClass),
		Severity:     severity,
		Title:        title,
		Description:  description,
		Remediation:  remediation,
		FilePath:     dep.GetFilePath(),
		Ecosystem:    dep.GetEcosystem(),
		PackageName:  dep.GetPackageName(),
		Observed:     observed,
		Expected:     expected,
		Scenario:     scenario,
		FindingClass: findingClass,
		PolicyMode:   policyMode,
	}
}

func summarizeRecords(records []*governancev1.ApprovedDependencyRecord) *governancev1.DependencyGovernanceSummary {
	summary := &governancev1.DependencyGovernanceSummary{Status: "pass"}
	if len(records) == 0 {
		summary.Status = "not_configured"
		return summary
	}
	for _, record := range records {
		switch normalize(record.GetState()) {
		case "approved":
			summary.Approved++
		case "approved_with_constraints":
			summary.ApprovedWithConstraints++
		case "needs_review":
			summary.NeedsReview++
		case "blocked":
			summary.Blocked++
			summary.Denied++
		case "denied":
			summary.Denied++
		case "deprecated":
			summary.Deprecated++
		}
	}
	return summary
}

func recordsEqual(left, right *governancev1.ApprovedDependencyRecord) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.GetEcosystem() == right.GetEcosystem() &&
		left.GetPackageName() == right.GetPackageName() &&
		left.GetVersionRange() == right.GetVersionRange() &&
		left.GetRangePolicy() == right.GetRangePolicy() &&
		left.GetState() == right.GetState() &&
		left.GetRationale() == right.GetRationale() &&
		left.GetApprovedBy() == right.GetApprovedBy() &&
		left.GetApprovedDate() == right.GetApprovedDate() &&
		left.GetLastReviewed() == right.GetLastReviewed() &&
		left.GetReviewExpires() == right.GetReviewExpires() &&
		left.GetLicenseNotes() == right.GetLicenseNotes() &&
		left.GetSecurityNotes() == right.GetSecurityNotes() &&
		left.GetReplacement() == right.GetReplacement() &&
		stringSlicesEqual(left.GetUseCases(), right.GetUseCases()) &&
		stringSlicesEqual(left.GetExampleScenarios(), right.GetExampleScenarios()) &&
		stringSlicesEqual(left.GetKeywords(), right.GetKeywords()) &&
		stringSlicesEqual(left.GetAllowedScenarios(), right.GetAllowedScenarios()) &&
		stringSlicesEqual(left.GetDeniedScenarios(), right.GetDeniedScenarios())
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func mutationMessage(dryRun, changed, replaced bool, record *governancev1.ApprovedDependencyRecord) string {
	action := "create"
	if replaced {
		action = "update"
	}
	key := recordKey(record.GetEcosystem(), record.GetPackageName())
	if !changed {
		return fmt.Sprintf("No registry change needed for %s.", key)
	}
	if dryRun {
		return fmt.Sprintf("Dry run: would %s governance decision for %s.", action, key)
	}
	return fmt.Sprintf("Governance decision %sd for %s.", action, key)
}

func matchesFilter(value, filter string) bool {
	filter = strings.TrimSpace(filter)
	return filter == "" || sameFold(value, filter)
}

func recordMatches(record *governancev1.ApprovedDependencyRecord, terms []string) bool {
	haystack := strings.ToLower(strings.Join(append([]string{
		record.GetEcosystem(),
		record.GetPackageName(),
		record.GetVersionRange(),
		record.GetState(),
		record.GetRationale(),
		record.GetLicenseNotes(),
		record.GetSecurityNotes(),
		record.GetReplacement(),
	}, append(append(append([]string{}, record.GetUseCases()...), record.GetExampleScenarios()...), record.GetKeywords()...)...), " "))
	for _, term := range terms {
		if !strings.Contains(haystack, term) {
			return false
		}
	}
	return true
}

func queryTerms(query string) []string {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if len(field) > 1 {
			out = append(out, field)
		}
	}
	return out
}

func splitConstraintTokens(clause string) []string {
	fields := strings.FieldsFunc(clause, func(r rune) bool {
		return r == ',' || r == ' '
	})
	tokens := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			tokens = append(tokens, field)
		}
	}
	return tokens
}

func recordKey(ecosystem, packageName string) string {
	return normalize(ecosystem) + "/" + strings.ToLower(strings.TrimSpace(packageName))
}

func sortRecords(records []*governancev1.ApprovedDependencyRecord) {
	sort.Slice(records, func(i, j int) bool {
		return recordKey(records[i].GetEcosystem(), records[i].GetPackageName()) < recordKey(records[j].GetEcosystem(), records[j].GetPackageName())
	})
}

func trimStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

func containsFold(values []string, needle string) bool {
	for _, value := range values {
		if sameFold(value, needle) {
			return true
		}
	}
	return false
}

func sameFold(left, right string) bool {
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

var _ governanceconnect.DependencyGovernanceServiceHandler = (*connectHandler)(nil)
