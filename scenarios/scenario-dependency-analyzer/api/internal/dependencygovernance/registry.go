package dependencygovernance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
	Guidance string                     `json:"guidance"`
	Records  []approvedDependencyRecord `json:"records"`
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
	resp, err := h.registry().ValidateScenario(scenario)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
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

func (r *Registry) ValidateScenario(scenario string) (*governancev1.ApprovedDependencyValidationResponse, error) {
	scenarioDir := filepath.Join(filepath.Dir(r.registryPath()), "..", "scenarios", scenario)
	if r.repoRoot != "" {
		scenarioDir = filepath.Join(r.repoRoot, "scenarios", scenario)
	}
	observed, err := ScanScenarioDependencies(scenarioDir)
	if err != nil {
		return nil, err
	}
	return r.ValidateObserved(scenario, observed)
}

func (r *Registry) ValidateObserved(scenario string, observed []*governancev1.ObservedDependency) (*governancev1.ApprovedDependencyValidationResponse, error) {
	records, err := r.loadRecords()
	if err != nil {
		return nil, err
	}
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
			findings = append(findings, governanceFinding(dep, "WARNING", "Dependency needs governance review", "This dependency is not yet recorded in approved dependency memory.", "Keep the dependency if it is the right tool, and submit purpose, version/range, alternatives considered, and security/license notes for review.", dep.GetVersion(), "recorded approval, constraint, deprecation, or block decision"))
			continue
		}
		state := normalize(record.GetState())
		switch state {
		case "blocked":
			findings = append(findings, governanceFinding(dep, "ERROR", "Blocked dependency is in use", "This dependency is recorded as blocked for Vrooli usage.", firstNonEmpty(record.GetReplacement(), "Replace the dependency or file an explicit governance exception with rationale and expiry."), dep.GetVersion(), "dependency absent or governance exception approved"))
		case "deprecated":
			findings = append(findings, governanceFinding(dep, "WARNING", "Deprecated dependency is in use", "This dependency is recorded as deprecated.", firstNonEmpty(record.GetReplacement(), "Plan a migration to a maintained replacement."), dep.GetVersion(), "replacement dependency in use"))
		case "approved", "approved_with_constraints", "needs_review", "":
			if !versionAllowed(dep.GetVersion(), record.GetVersionRange()) {
				findings = append(findings, governanceFinding(dep, "WARNING", "Dependency version is outside recorded approval", "The dependency is recorded, but the observed version/range does not match the approved range.", "Review whether the observed version should be approved, constrained, or changed.", dep.GetVersion(), firstNonEmpty(record.GetVersionRange(), "recorded approved range")))
			}
			if state == "needs_review" {
				findings = append(findings, governanceFinding(dep, "WARNING", "Dependency approval still needs review", "This dependency has a governance record but has not been approved yet.", "Complete dependency review or choose an already approved alternative if appropriate.", dep.GetVersion(), "approved or approved_with_constraints"))
			}
		default:
			findings = append(findings, governanceFinding(dep, "WARNING", "Dependency has unknown governance state", "This dependency has a governance record with an unrecognized state.", "Fix the approved dependency registry state value.", state, "approved, approved_with_constraints, needs_review, blocked, or deprecated"))
		}
	}

	summary := summarizeRecords(records)
	summary.Observed = int32(len(observed))
	for _, finding := range findings {
		if strings.Contains(finding.GetDescription(), "not yet recorded") {
			summary.Unrecorded++
		}
	}
	status := "pass"
	passed := true
	for _, finding := range findings {
		if strings.EqualFold(finding.GetSeverity(), "ERROR") {
			status = "fail"
			passed = false
			break
		}
		if strings.EqualFold(finding.GetSeverity(), "WARNING") {
			status = "warn"
		}
	}
	if len(records) == 0 {
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

func (r *Registry) loadRecords() ([]*governancev1.ApprovedDependencyRecord, error) {
	data, err := os.ReadFile(r.registryPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read approved dependency registry: %w", err)
	}
	var raw registryFile
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse approved dependency registry: %w", err)
	}
	records := make([]*governancev1.ApprovedDependencyRecord, 0, len(raw.Records))
	for _, record := range raw.Records {
		records = append(records, record.toProto())
	}
	sortRecords(records)
	return records, nil
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
		AllowedSurfaces:  trimStrings(r.AllowedSurfaces),
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
			observed = append(observed, deps...)
		case "go.mod":
			deps, err := scanGoMod(path)
			if err != nil {
				return err
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

func isUnrecordedTransitiveDependency(dep *governancev1.ObservedDependency) bool {
	return sameFold(dep.GetEcosystem(), "go") && sameFold(dep.GetDependencyGroup(), "require_indirect")
}

func governanceFinding(dep *governancev1.ObservedDependency, severity, title, description, remediation, observed, expected string) *governancev1.ApprovedDependencyFinding {
	return &governancev1.ApprovedDependencyFinding{
		Id:          "governance." + slug(dep.GetEcosystem()+"."+dep.GetPackageName()+"."+title),
		Severity:    severity,
		Title:       title,
		Description: description,
		Remediation: remediation,
		FilePath:    dep.GetFilePath(),
		Ecosystem:   dep.GetEcosystem(),
		PackageName: dep.GetPackageName(),
		Observed:    observed,
		Expected:    expected,
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
		case "deprecated":
			summary.Deprecated++
		}
	}
	return summary
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

func versionAllowed(observed, approvedRange string) bool {
	approvedRange = strings.TrimSpace(approvedRange)
	observed = strings.TrimSpace(observed)
	return approvedRange == "" || approvedRange == "*" || observed == "" || observed == approvedRange
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
