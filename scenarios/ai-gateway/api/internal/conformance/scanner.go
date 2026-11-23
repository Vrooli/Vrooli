package conformance

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	conformancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/conformance"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"

	"github.com/vrooli/maturity-go/assessment"
)

type Scanner struct{}

func NewScanner() *Scanner { return &Scanner{} }

type ScanRequest struct {
	Scenario string
	Path     string
}

type ScanReport struct {
	Scenario        string
	MaturityLevel   string
	Findings        []*conformancev1.ConformanceFinding
	Recommendations []string
}

type rule struct {
	id          string
	severity    string
	message     string
	remediation string
	match       func(line string) bool
}

var (
	modelSlugPattern     = regexp.MustCompile(`(?i)\b(gpt-[0-9][a-z0-9.-]*|claude-[0-9][a-z0-9.-]*|qwen[0-9][a-z0-9.]*:[a-z0-9._-]+|llama[0-9][a-z0-9.]*:[a-z0-9._-]+|mistral-[0-9][a-z0-9.-]*|nomic-embed-text(?::latest)?)\b`)
	contextWindowPattern = regexp.MustCompile(`\b(4096|8192|16384|32768|65536|128000|200000)\b`)
	openRouterAPIURL     = regexp.MustCompile(`(?i)https?://(?:api\.)?openrouter\.ai/(?:api/)?v1(?:/|\b)`)
)

var rules = []rule{
	{
		id:          "ai.direct_ollama_http",
		severity:    "high",
		message:     "Direct Ollama HTTP usage bypasses resource and gateway policy boundaries.",
		remediation: "Route through AI Gateway, or through resource-ollama gateway commands only when a reviewed low-level exception applies.",
		match: func(line string) bool {
			l := strings.ToLower(line)
			return strings.Contains(l, "localhost:11434") || strings.Contains(l, "/api/generate") || strings.Contains(l, "/api/embeddings")
		},
	},
	{
		id:          "ai.direct_openrouter_http",
		severity:    "high",
		message:     "Direct OpenRouter HTTP usage bypasses resource-owned credential and model policy.",
		remediation: "Route hosted text inference through AI Gateway or resource-openrouter with role policy.",
		match: func(line string) bool {
			return openRouterAPIURL.MatchString(line)
		},
	},
	{
		id:          "ai.non_gateway_provider_client_import",
		severity:    "high",
		message:     "Scenario code imports a resource provider client outside AI Gateway.",
		remediation: "Expose the provider through AI Gateway and import only the gateway contract in caller scenarios.",
		match: func(line string) bool {
			l := strings.ToLower(strings.TrimSpace(line))
			if !strings.Contains(l, "import") && !strings.Contains(l, "require(") {
				return false
			}
			return strings.Contains(l, "resource-openrouter") ||
				strings.Contains(l, "resource-ollama") ||
				strings.Contains(l, "/resources/openrouter") ||
				strings.Contains(l, "/resources/ollama")
		},
	},
	{
		id:          "ai.invalid_provider_secret_env",
		severity:    "high",
		message:     "Provider secret environment variables belong to the resource, not caller scenarios.",
		remediation: "Move provider credential resolution to the owning resource and call AI Gateway/resource roles from the scenario.",
		match: func(line string) bool {
			l := strings.ToUpper(line)
			return strings.Contains(l, "OPENROUTER_API_KEY") || strings.Contains(l, "ANTHROPIC_API_KEY") || strings.Contains(l, "OPENAI_API_KEY")
		},
	},
	{
		id:          "ai.invalid_provider_url_env",
		severity:    "medium",
		message:     "Provider URL environment variables couple the scenario to provider topology.",
		remediation: "Use AI Gateway profiles or resource-owned discovery instead of provider URL env vars.",
		match: func(line string) bool {
			l := strings.ToUpper(line)
			return strings.Contains(l, "OLLAMA_URL") || strings.Contains(l, "OPENROUTER_BASE_URL") || strings.Contains(l, "OPENROUTER_API_BASE")
		},
	},
	{
		id:          "ai.concrete_model_slug",
		severity:    "high",
		message:     "Concrete model slugs should stay in resource policy, not scenario code.",
		remediation: "Replace concrete model references with AI Gateway role/profile requests or resource role policy.",
		match: func(line string) bool {
			l := strings.ToLower(line)
			if strings.Contains(l, "model-policy") || strings.Contains(l, "resource policy") {
				return false
			}
			return modelSlugPattern.MatchString(line)
		},
	},
	{
		id:          "ai.hardcoded_embedding_dimensions",
		severity:    "high",
		message:     "Embedding dimensions are hard-coded near vector/embedding code without visible role metadata.",
		remediation: "Store embedding role, resolved model metadata, dimensions, content version, and retarget strategy beside vectors.",
		match: func(line string) bool {
			l := strings.ToLower(line)
			return (strings.Contains(l, "embedding") || strings.Contains(l, "vector")) &&
				(strings.Contains(l, "768") || strings.Contains(l, "1536") || strings.Contains(l, "3072"))
		},
	},
	{
		id:          "ai.hardcoded_context_window",
		severity:    "medium",
		message:     "Context-window constants should be caller constraints or policy metadata, not provider truth.",
		remediation: "Express prompt budget as an operation constraint or read capacity from AI Gateway/resource policy metadata.",
		match: func(line string) bool {
			l := strings.ToLower(line)
			return (strings.Contains(l, "context") || strings.Contains(l, "window") || strings.Contains(l, "max_tokens")) &&
				contextWindowPattern.MatchString(line)
		},
	},
}

func BuildMaturityAssessment(report ScanReport, spec assessment.Spec) (*commonv1.MaturityAssessment, error) {
	findings := make([]assessment.Finding, 0, len(report.Findings))
	for _, finding := range report.Findings {
		findings = append(findings, assessment.Finding{
			Code:        finding.GetRuleId(),
			Severity:    severityToken(finding.GetSeverity()),
			Title:       titleForRule(finding.GetRuleId()),
			Message:     finding.GetMessage(),
			Location:    finding.GetPath(),
			Remediation: finding.GetRemediation(),
			Phase:       spec.Phase,
		})
	}
	return assessment.BuildProtoAssessment(assessment.BuildInput{
		Scenario: report.Scenario,
		Spec:     spec,
		Findings: findings,
	})
}

func (s *Scanner) Scan(ctx context.Context, req ScanRequest) (ScanReport, error) {
	scenario := strings.TrimSpace(req.Scenario)
	root, err := resolveTargetPath(scenario, req.Path)
	if err != nil {
		return ScanReport{}, err
	}
	state, err := scanScenarioTree(ctx, root)
	if err != nil {
		return ScanReport{}, err
	}
	roleFindings, err := scanGatewayRoleReachability(ctx, root)
	if err != nil {
		return ScanReport{}, err
	}
	state.findings = append(state.findings, roleFindings...)
	return buildScenarioScanReport(scenario, root, state), nil
}

type scenarioScanState struct {
	findings    []*conformancev1.ConformanceFinding
	usesGateway bool
	aiSignal    bool
}

func scanScenarioTree(ctx context.Context, root string) (scenarioScanState, error) {
	state := scenarioScanState{}
	// A scenario may own a typed model registry for cataloguing model facts.
	// Those facts are only exempt when the registry has an executable loader,
	// validation, generated-document drift coverage, and a test tying its
	// policy roles to the owning resource. Provider catalogs may then reference
	// IDs from that registry, but arbitrary caller literals remain findings.
	registry := governedModelRegistryIDs(root)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			if skipDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		return state.scanFile(root, path, registry)
	})
	return state, err
}

func (state *scenarioScanState) scanFile(root, path string, registry governedModelRegistry) error {
	if !scanFile(path) {
		return nil
	}
	fileFindings, fileUsesGateway, fileAISignal, err := scanOneFile(root, path, registry)
	if err != nil {
		return err
	}
	state.findings = append(state.findings, fileFindings...)
	state.usesGateway = state.usesGateway || fileUsesGateway
	state.aiSignal = state.aiSignal || fileAISignal || len(fileFindings) > 0
	return nil
}

func buildScenarioScanReport(scenario, root string, state scenarioScanState) ScanReport {
	findings := state.findings
	if state.aiSignal && !state.usesGateway {
		findings = append(findings, &conformancev1.ConformanceFinding{
			RuleId:      "ai.gateway_not_adopted",
			Severity:    "advisory",
			Path:        ".",
			Message:     "AI usage signals exist, but no AI Gateway adoption signal was found.",
			Remediation: "Evaluate migration to AI Gateway profiles, or document a reviewed exception for direct resource usage.",
		})
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].GetPath() == findings[j].GetPath() {
			return findings[i].GetRuleId() < findings[j].GetRuleId()
		}
		return findings[i].GetPath() < findings[j].GetPath()
	})
	return ScanReport{
		Scenario: firstNonEmpty(scenario, filepath.Base(root)), MaturityLevel: maturityLevel(findings),
		Findings: findings, Recommendations: recommendations(findings),
	}
}

func scanOneFile(root, path string, registry governedModelRegistry) ([]*conformancev1.ConformanceFinding, bool, bool, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return nil, false, false, err
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return nil, false, false, fmt.Errorf("scan path escapes root: %s", path)
	}
	f, err := os.Open(path) // #nosec G304 -- path comes from WalkDir under root and is checked with filepath.Rel before opening.
	if err != nil {
		return nil, false, false, err
	}
	defer f.Close()

	rel = filepath.ToSlash(rel)
	var findings []*conformancev1.ConformanceFinding
	usesGateway := false
	aiSignal := false
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		lower := strings.ToLower(line)
		usesGateway = usesGateway || lineUsesGateway(lower)
		aiSignal = aiSignal || lineAISignal(lower)
		findings = append(findings, lineFindings(rel, lineNo, line, registry)...)
	}
	return findings, usesGateway, aiSignal, scanner.Err()
}

func lineUsesGateway(lower string) bool {
	return strings.Contains(lower, "ai-gateway") || strings.Contains(lower, "ai_gateway")
}

func lineAISignal(lower string) bool {
	return lineUsesGateway(lower) ||
		strings.Contains(lower, "resource-ollama") ||
		strings.Contains(lower, "resource-openrouter") ||
		strings.Contains(lower, "embedding") ||
		strings.Contains(lower, "openrouter") ||
		strings.Contains(lower, "ollama")
}

func lineFindings(rel string, lineNo int, line string, registry governedModelRegistry) []*conformancev1.ConformanceFinding {
	location := fmt.Sprintf("%s:%d", rel, lineNo)
	findings := make([]*conformancev1.ConformanceFinding, 0, 4)
	if hasUnreviewedException(line) {
		findings = append(findings, conformanceFinding(
			"ai.unreviewed_exception",
			"medium",
			location,
			"AI conformance exception markers need owner, reason, expiry, and replacement-plan metadata.",
			"Add reviewed exception metadata or remove the exception by routing through AI Gateway/resource policy.",
		))
	}
	if usesResourceCommandWithoutRole(line) {
		findings = append(findings, conformanceFinding(
			"ai.resource_gateway_missing_role",
			"medium",
			location,
			"Direct resource command usage is missing visible role/profile policy selection.",
			"Call resource role/policy commands or AI Gateway profiles so provider selection remains resource-owned.",
		))
	}
	if embeddingMetadataMissing(line) {
		findings = append(findings, conformanceFinding(
			"ai.embedding_metadata_missing",
			"high",
			location,
			"Vector/embedding storage appears to lack role, model, dimension, content-version, or migration metadata.",
			"Persist embedding role, resolved model evidence, dimensions, content version, and retarget strategy beside stored vectors.",
		))
	}
	return append(findings, ruleFindings(rel, location, line, registry)...)
}

func ruleFindings(rel, location, line string, registry governedModelRegistry) []*conformancev1.ConformanceFinding {
	findings := make([]*conformancev1.ConformanceFinding, 0, len(rules))
	for _, r := range rules {
		// Documentation may name a provider secret while describing migration
		// history or an unsafe pattern. Only runtime/configuration surfaces can
		// actually read or declare that secret.
		if r.id == "ai.invalid_provider_secret_env" && strings.EqualFold(filepath.Ext(rel), ".md") {
			continue
		}
		matchLine := line
		if r.id == "ai.concrete_model_slug" {
			if !modelTagRuntimeFile(rel) {
				continue
			}
			matchLine = codePortion(line)
			if rel == "api/internal/modelcatalog/models.seed.json" && len(registry.ids) > 0 {
				matchLine = removeKnownRegistryIDs(matchLine, registry.ids)
			}
			if rel == "api/internal/modelcatalog/catalog.go" && len(registry.ids) > 0 {
				matchLine = removeKnownRegistryIDs(matchLine, registry.ids)
			}
			if rel == "api/internal/providercatalog/catalog.seed.json" && registry.providerCatalogChecked {
				matchLine = removeKnownRegistryIDs(matchLine, registry.ids)
			}
			if rel == "ui/src/features/providers/providerCatalog.generated.ts" && registry.providerCatalogRendered {
				matchLine = removeKnownRegistryIDs(matchLine, registry.ids)
			}
		}
		if r.match(matchLine) {
			findings = append(findings, conformanceFinding(r.id, r.severity, location, r.message, r.remediation))
		}
	}
	return findings
}

func removeKnownRegistryIDs(line string, registryIDs map[string]struct{}) string {
	matches := modelSlugPattern.FindAllStringIndex(line, -1)
	if len(matches) == 0 {
		return line
	}
	var result strings.Builder
	last := 0
	for _, match := range matches {
		result.WriteString(line[last:match[0]])
		if _, known := registryIDs[line[match[0]:match[1]]]; !known {
			result.WriteString(line[match[0]:match[1]])
		}
		last = match[1]
	}
	result.WriteString(line[last:])
	return result.String()
}

// modelTagRuntimeFile limits the blocking tag rule to executable/configured
// caller surfaces. Documentation, requirements, tests, generated fixtures,
// and evaluation snapshots may name models as evidence without selecting
// them at runtime.
func modelTagRuntimeFile(rel string) bool {
	normalized := filepath.ToSlash(rel)
	lower := strings.ToLower(normalized)
	for _, part := range strings.Split(lower, "/") {
		switch part {
		case "docs", "test", "tests", "testdata", "fixtures", "evals", "requirements":
			return false
		}
	}
	base := strings.ToLower(filepath.Base(normalized))
	if base == "registry.seed.json" || strings.HasSuffix(base, ".freshness.json") {
		return false
	}
	for _, marker := range []string{"_test.", ".test.", ".spec."} {
		if strings.Contains(base, marker) {
			return false
		}
	}
	switch strings.ToLower(filepath.Ext(normalized)) {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".json", ".yaml", ".yml", ".toml", ".env":
		return true
	default:
		return false
	}
}

func conformanceFinding(ruleID, severity, path, message, remediation string) *conformancev1.ConformanceFinding {
	return &conformancev1.ConformanceFinding{
		RuleId:      ruleID,
		Severity:    severity,
		Path:        path,
		Message:     message,
		Remediation: remediation,
	}
}

func usesResourceCommandWithoutRole(line string) bool {
	l := strings.ToLower(line)
	if !strings.Contains(l, "resource-ollama") && !strings.Contains(l, "resource-openrouter") {
		return false
	}
	// Runtime build identities name the owner binary but do not invoke its CLI.
	if strings.Contains(l, "resource-ollama-gateway/") && strings.Contains(l, "runtimebuild") {
		return false
	}
	return !strings.Contains(l, "role") && !strings.Contains(l, "profile") && !strings.Contains(l, "policy")
}

func embeddingMetadataMissing(line string) bool {
	l := strings.ToLower(line)
	if !strings.Contains(l, "embedding") && !strings.Contains(l, "vector") {
		return false
	}
	if !strings.Contains(l, "create table") && !strings.Contains(l, "create index") && !strings.Contains(l, "collection") {
		return false
	}
	return !strings.Contains(l, "role") && !strings.Contains(l, "model") &&
		!strings.Contains(l, "dimension") && !strings.Contains(l, "content_version") &&
		!strings.Contains(l, "migration")
}

func hasUnreviewedException(line string) bool {
	l := strings.ToLower(line)
	if !strings.Contains(l, "ai-gateway-exception") && !strings.Contains(l, "ai conformance exception") {
		return false
	}
	fields := make(map[string]string, 4)
	for _, token := range strings.Fields(line) {
		key, value, ok := strings.Cut(strings.Trim(token, "/*#"), "=")
		if !ok {
			continue
		}
		key = strings.ToLower(key)
		if key == "owner" || key == "reason" || key == "expires" || key == "expiry" || key == "replacement" {
			fields[key] = strings.Trim(value, "\"'`,;.)")
		}
	}
	return fields["owner"] == "" || fields["reason"] == "" ||
		(fields["expires"] == "" && fields["expiry"] == "") || fields["replacement"] == ""
}

func resolveTargetPath(scenario, explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return filepath.Abs(explicit)
	}
	if strings.TrimSpace(scenario) == "" {
		return "", fmt.Errorf("scenario or path is required")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "scenarios", scenario)
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return candidate, nil
		}
		next := filepath.Dir(dir)
		if next == dir {
			break
		}
	}
	return "", fmt.Errorf("scenario %q could not be resolved; pass path explicitly", scenario)
}

func skipDir(name string) bool {
	switch name {
	case ".git", "node_modules", "dist", "dist-electron", "build", "coverage", "data", ".vrooli", "assets", "bundle", "docs", "artifacts", "evidence", "reports", "testdata", "fixtures", "evals", "tmp":
		return true
	default:
		return strings.HasSuffix(name, ".egg-info")
	}
}

func scanFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	for _, marker := range []string{"_test.", ".test.", ".spec."} {
		if strings.Contains(base, marker) {
			return false
		}
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".json", ".yaml", ".yml", ".toml", ".env", ".sql":
		return true
	default:
		return false
	}
}
