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
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"

	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
	governanceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance/dependency_governance_v1connect"
)

const Guidance = "These are dependencies already recorded as approved for the shown scope/range. This is not an exhaustive allowlist. If a better dependency is appropriate, suggest it with purpose, version/range, alternatives considered, and security/license notes so it can be reviewed and recorded."

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
}

type securityHealthExplainResponse struct {
	Vulnerability securityHealthVulnerability `json:"vulnerability"`
	Found         bool                        `json:"found"`
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
		if req.GetSurface() != "" && !containsFold(record.GetAllowedSurfaces(), req.GetSurface()) {
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
			findings = append(findings, governanceFinding(scenario, dep, "DENIED_IN_USE", "ERROR", "Denied dependency is in use", "This dependency is recorded as denied for Vrooli usage.", firstNonEmpty(record.GetReplacement(), "Replace the dependency or file an explicit governance exception with rationale and expiry."), dep.GetVersion(), "dependency absent or governance exception approved", policyMode))
		case "deprecated":
			findings = append(findings, governanceFinding(scenario, dep, "DEPRECATED_IN_USE", "WARNING", "Deprecated dependency is in use", "This dependency is recorded as deprecated.", firstNonEmpty(record.GetReplacement(), "Plan a migration to a maintained replacement."), dep.GetVersion(), "replacement dependency in use", policyMode))
		case "approved", "approved_with_constraints", "needs_review", "exception", "":
			if !versionAllowed(dep.GetEcosystem(), dep.GetVersion(), record.GetVersionRange()) {
				findings = append(findings, governanceFinding(scenario, dep, "VERSION_OUT_OF_RANGE", "WARNING", "Dependency version is outside recorded approval", "The dependency is recorded, but the observed version/range does not match the approved range.", "Review whether the observed version should be approved, constrained, or changed.", dep.GetVersion(), firstNonEmpty(record.GetVersionRange(), "recorded approved range"), policyMode))
			}
			if scopeReason := scopeViolation(scenario, dep, record); scopeReason != "" {
				findings = append(findings, governanceFinding(scenario, dep, "SCOPE_VIOLATION", "WARNING", "Dependency is outside recorded governance scope", scopeReason, "Update the dependency scope, move usage to an approved surface/group, or choose a scoped alternative.", depScope(dep), recordScope(record), policyMode))
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
		case "SCOPE_VIOLATION":
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

func securityDeniedRecord(evidence *governancev1.SecurityVulnerabilityEvidence, affectedRangeOverride, fixedRangeOverride, rationaleOverride string) *governancev1.ApprovedDependencyRecord {
	affectedRange := firstNonEmpty(strings.TrimSpace(affectedRangeOverride), firstSecurityAffectedRange(evidence), evidence.GetObservedVersion(), "*")
	fixedRange := firstNonEmpty(strings.TrimSpace(fixedRangeOverride), firstSecurityFixedRange(evidence), "a fixed version outside the affected range")
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

func firstSecurityAffectedRange(evidence *governancev1.SecurityVulnerabilityEvidence) string {
	for _, r := range evidence.GetAffectedRanges() {
		if strings.TrimSpace(r.GetRange()) != "" {
			return strings.TrimSpace(r.GetRange())
		}
		if strings.TrimSpace(r.GetLastAffected()) != "" {
			return "<= " + strings.TrimSpace(r.GetLastAffected())
		}
	}
	return ""
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

func remediationForEvidence(evidence *governancev1.SecurityVulnerabilityEvidence, affectedRange string) string {
	fixedRange := firstNonEmpty(firstSecurityFixedRange(evidence), "a fixed version outside the affected range")
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
		Ecosystem:               normalize(r.Ecosystem),
		PackageName:             strings.TrimSpace(r.PackageName),
		VersionRange:            strings.TrimSpace(r.VersionRange),
		State:                   normalize(r.State),
		AllowedSurfaces:         trimStrings(r.AllowedSurfaces),
		UseCases:                trimStrings(r.UseCases),
		Rationale:               strings.TrimSpace(r.Rationale),
		ApprovedBy:              strings.TrimSpace(r.ApprovedBy),
		ApprovedDate:            strings.TrimSpace(r.ApprovedDate),
		LastReviewed:            strings.TrimSpace(r.LastReviewed),
		ReviewExpires:           strings.TrimSpace(r.ReviewExpires),
		LicenseNotes:            strings.TrimSpace(r.LicenseNotes),
		SecurityNotes:           strings.TrimSpace(r.SecurityNotes),
		ExampleScenarios:        trimStrings(r.ExampleScenarios),
		Replacement:             strings.TrimSpace(r.Replacement),
		Keywords:                trimStrings(r.Keywords),
		AllowedScenarios:        trimStrings(r.AllowedScenarios),
		DeniedScenarios:         trimStrings(r.DeniedScenarios),
		AllowedDependencyGroups: trimStrings(r.AllowedGroups),
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
		AllowedSurfaces:  trimStrings(record.GetAllowedSurfaces()),
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
		AllowedGroups:    trimStrings(record.GetAllowedDependencyGroups()),
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
		case "SCOPE_VIOLATION":
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
			group.ObservedDependencies = append(group.GetObservedDependencies(), dep)
			scenariosByKey[key][resp.GetScenario()] = struct{}{}
		}
	}
	out := make([]*governancev1.DependencyUsageGroup, 0, len(groups))
	for key, group := range groups {
		for scenario := range scenariosByKey[key] {
			group.Scenarios = append(group.GetScenarios(), scenario)
		}
		sort.Strings(group.Scenarios)
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

func maxSeverity(left, right string) string {
	rank := map[string]int{"": 0, "INFO": 1, "WARNING": 2, "ERROR": 3, "BLOCKER": 4}
	if rank[strings.ToUpper(right)] > rank[strings.ToUpper(left)] {
		return strings.ToUpper(right)
	}
	return strings.ToUpper(left)
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

func scopeViolation(scenario string, dep *governancev1.ObservedDependency, record *governancev1.ApprovedDependencyRecord) string {
	if containsFold(record.GetDeniedScenarios(), scenario) {
		return "This dependency is denied for the current scenario."
	}
	if len(record.GetAllowedScenarios()) > 0 && !containsFold(record.GetAllowedScenarios(), scenario) {
		return "This dependency is not approved for the current scenario."
	}
	if dep.GetSurfaceId() != "" && len(record.GetAllowedSurfaces()) > 0 && !containsFold(record.GetAllowedSurfaces(), dep.GetSurfaceId()) {
		return "This dependency is not approved for the current surface."
	}
	if dep.GetDependencyGroup() != "" && len(record.GetAllowedDependencyGroups()) > 0 && !containsFold(record.GetAllowedDependencyGroups(), dep.GetDependencyGroup()) {
		return "This dependency is not approved for the current dependency group."
	}
	return ""
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

func depScope(dep *governancev1.ObservedDependency) string {
	parts := []string{}
	if dep.GetSurfaceId() != "" {
		parts = append(parts, "surface="+dep.GetSurfaceId())
	}
	if dep.GetDependencyGroup() != "" {
		parts = append(parts, "group="+dep.GetDependencyGroup())
	}
	return strings.Join(parts, ", ")
}

func recordScope(record *governancev1.ApprovedDependencyRecord) string {
	parts := []string{}
	if len(record.GetAllowedScenarios()) > 0 {
		parts = append(parts, "scenarios="+strings.Join(record.GetAllowedScenarios(), ","))
	}
	if len(record.GetAllowedSurfaces()) > 0 {
		parts = append(parts, "surfaces="+strings.Join(record.GetAllowedSurfaces(), ","))
	}
	if len(record.GetAllowedDependencyGroups()) > 0 {
		parts = append(parts, "groups="+strings.Join(record.GetAllowedDependencyGroups(), ","))
	}
	return firstNonEmpty(strings.Join(parts, "; "), "recorded scope")
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
		left.GetState() == right.GetState() &&
		left.GetRationale() == right.GetRationale() &&
		left.GetApprovedBy() == right.GetApprovedBy() &&
		left.GetApprovedDate() == right.GetApprovedDate() &&
		left.GetLastReviewed() == right.GetLastReviewed() &&
		left.GetReviewExpires() == right.GetReviewExpires() &&
		left.GetLicenseNotes() == right.GetLicenseNotes() &&
		left.GetSecurityNotes() == right.GetSecurityNotes() &&
		left.GetReplacement() == right.GetReplacement() &&
		stringSlicesEqual(left.GetAllowedSurfaces(), right.GetAllowedSurfaces()) &&
		stringSlicesEqual(left.GetUseCases(), right.GetUseCases()) &&
		stringSlicesEqual(left.GetExampleScenarios(), right.GetExampleScenarios()) &&
		stringSlicesEqual(left.GetKeywords(), right.GetKeywords()) &&
		stringSlicesEqual(left.GetAllowedScenarios(), right.GetAllowedScenarios()) &&
		stringSlicesEqual(left.GetDeniedScenarios(), right.GetDeniedScenarios()) &&
		stringSlicesEqual(left.GetAllowedDependencyGroups(), right.GetAllowedDependencyGroups())
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
	}, append(append(append(append([]string{}, record.GetAllowedSurfaces()...), record.GetUseCases()...), record.GetExampleScenarios()...), record.GetKeywords()...)...), " "))
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

func versionAllowed(ecosystem, observed, approvedRange string) bool {
	approvedRange = strings.TrimSpace(approvedRange)
	observed = strings.TrimSpace(observed)
	if approvedRange == "" || approvedRange == "*" || observed == "" || observed == approvedRange {
		return true
	}
	switch normalize(ecosystem) {
	case "npm", "go":
		return rangeAllowsVersion(approvedRange, observed)
	default:
		return false
	}
}

func rangeAllowsVersion(constraint, observed string) bool {
	observedVersion, ok := parseVersion(firstVersionToken(observed))
	if !ok {
		return false
	}
	for _, clause := range strings.Split(constraint, "||") {
		if clauseAllowsVersion(strings.TrimSpace(clause), observedVersion) {
			return true
		}
	}
	return false
}

func clauseAllowsVersion(clause string, observed semanticVersion) bool {
	if clause == "" || clause == "*" {
		return true
	}
	tokens := splitConstraintTokens(clause)
	if len(tokens) == 0 {
		return false
	}
	for _, token := range tokens {
		if !constraintTokenAllowsVersion(token, observed) {
			return false
		}
	}
	return true
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

func constraintTokenAllowsVersion(token string, observed semanticVersion) bool {
	token = strings.TrimSpace(token)
	if token == "" || token == "*" || token == "x" || token == "X" {
		return true
	}
	if strings.HasPrefix(token, "^") {
		base, ok := parseVersion(strings.TrimPrefix(token, "^"))
		return ok && compareVersion(observed, base) >= 0 && observed.Major == base.Major
	}
	if strings.HasPrefix(token, "~") {
		base, ok := parseVersion(strings.TrimPrefix(token, "~"))
		return ok && compareVersion(observed, base) >= 0 && observed.Major == base.Major && observed.Minor == base.Minor
	}
	for _, op := range []string{">=", "<=", ">", "<", "="} {
		if strings.HasPrefix(token, op) {
			want, ok := parseVersion(strings.TrimPrefix(token, op))
			if !ok {
				return false
			}
			cmp := compareVersion(observed, want)
			switch op {
			case ">=":
				return cmp >= 0
			case "<=":
				return cmp <= 0
			case ">":
				return cmp > 0
			case "<":
				return cmp < 0
			case "=":
				return cmp == 0
			}
		}
	}
	want, ok := parseVersion(token)
	return ok && compareVersion(observed, want) == 0
}

type semanticVersion struct {
	Major int
	Minor int
	Patch int
}

func firstVersionToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ' ' || r == ',' || r == '|'
	})
	if len(fields) == 0 {
		return value
	}
	return fields[0]
}

func parseVersion(value string) (semanticVersion, bool) {
	value = strings.TrimSpace(value)
	value = strings.TrimLeft(value, "^~<>= ")
	value = strings.TrimPrefix(value, "v")
	value = strings.SplitN(value, "-", 2)[0]
	value = strings.SplitN(value, "+", 2)[0]
	if value == "" || value == "*" {
		return semanticVersion{}, false
	}
	parts := strings.Split(value, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return semanticVersion{}, false
	}
	nums := []int{0, 0, 0}
	for i, part := range parts {
		if part == "x" || part == "X" || part == "*" {
			nums[i] = 0
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return semanticVersion{}, false
		}
		nums[i] = n
	}
	return semanticVersion{Major: nums[0], Minor: nums[1], Patch: nums[2]}, true
}

func compareVersion(left, right semanticVersion) int {
	if left.Major != right.Major {
		return left.Major - right.Major
	}
	if left.Minor != right.Minor {
		return left.Minor - right.Minor
	}
	return left.Patch - right.Patch
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
