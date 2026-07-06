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
	modelSlugPattern     = regexp.MustCompile(`(?i)\b(gpt-[a-z0-9.-]+|claude-[a-z0-9.-]+|qwen[0-9a-z:.-]+|llama[0-9a-z:.-]+|mistral[0-9a-z:.-]+|nomic-embed-text[:a-z0-9.-]*)\b`)
	contextWindowPattern = regexp.MustCompile(`\b(4096|8192|16384|32768|65536|128000|200000)\b`)
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
			return strings.Contains(strings.ToLower(line), "api.openrouter.ai")
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
		severity:    "medium",
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
	var findings []*conformancev1.ConformanceFinding
	usesGateway := false
	aiSignal := false

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() {
			if skipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !scanFile(path) {
			return nil
		}
		fileFindings, fileUsesGateway, fileAISignal, err := scanOneFile(root, path)
		if err != nil {
			return err
		}
		findings = append(findings, fileFindings...)
		usesGateway = usesGateway || fileUsesGateway
		aiSignal = aiSignal || fileAISignal || len(fileFindings) > 0
		return nil
	})
	if err != nil {
		return ScanReport{}, err
	}
	if aiSignal && !usesGateway {
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
		Scenario:        firstNonEmpty(scenario, filepath.Base(root)),
		MaturityLevel:   maturityLevel(findings),
		Findings:        findings,
		Recommendations: recommendations(findings),
	}, nil
}

func scanOneFile(root, path string) ([]*conformancev1.ConformanceFinding, bool, bool, error) {
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
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		lower := strings.ToLower(line)
		usesGateway = usesGateway || lineUsesGateway(lower)
		aiSignal = aiSignal || lineAISignal(lower)
		findings = append(findings, lineFindings(rel, lineNo, line)...)
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

func lineFindings(rel string, lineNo int, line string) []*conformancev1.ConformanceFinding {
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
	return append(findings, ruleFindings(location, line)...)
}

func ruleFindings(location, line string) []*conformancev1.ConformanceFinding {
	findings := make([]*conformancev1.ConformanceFinding, 0, len(rules))
	for _, r := range rules {
		if r.match(line) {
			findings = append(findings, conformanceFinding(r.id, r.severity, location, r.message, r.remediation))
		}
	}
	return findings
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
	return !strings.Contains(l, "owner=") || !strings.Contains(l, "reason=") ||
		(!strings.Contains(l, "expires=") && !strings.Contains(l, "expiry=")) ||
		!strings.Contains(l, "replacement=")
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
	case ".git", "node_modules", "dist", "build", "coverage", "data", ".vrooli":
		return true
	default:
		return strings.HasSuffix(name, ".egg-info")
	}
}

func scanFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".json", ".yaml", ".yml", ".toml", ".env", ".md", ".sql":
		return true
	default:
		return false
	}
}

func maturityLevel(findings []*conformancev1.ConformanceFinding) string {
	if len(findings) == 0 {
		return "gateway-ready"
	}
	for _, f := range findings {
		if f.GetSeverity() == "high" {
			return "blocked-needs-investigation"
		}
	}
	return "resource-boundary-review"
}

func severityToken(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "high":
		return "SEVERITY_ERROR"
	case "medium":
		return "SEVERITY_WARNING"
	case "low", "advisory":
		return "SEVERITY_INFO"
	default:
		return "SEVERITY_UNSPECIFIED"
	}
}

func titleForRule(ruleID string) string {
	switch ruleID {
	case "ai.direct_ollama_http":
		return "Direct Ollama HTTP usage"
	case "ai.direct_openrouter_http":
		return "Direct OpenRouter HTTP usage"
	case "ai.invalid_provider_secret_env":
		return "Provider secret owned by scenario"
	case "ai.invalid_provider_url_env":
		return "Provider URL owned by scenario"
	case "ai.concrete_model_slug":
		return "Concrete model slug in scenario code"
	case "ai.hardcoded_embedding_dimensions":
		return "Hard-coded embedding dimensions"
	case "ai.hardcoded_context_window":
		return "Hard-coded context window"
	case "ai.embedding_metadata_missing":
		return "Embedding metadata missing"
	case "ai.resource_gateway_missing_role":
		return "Resource role policy missing"
	case "ai.gateway_not_adopted":
		return "AI Gateway adoption signal missing"
	case "ai.unreviewed_exception":
		return "Unreviewed AI exception"
	default:
		return "AI conformance finding"
	}
}

func recommendations(findings []*conformancev1.ConformanceFinding) []string {
	if len(findings) == 0 {
		return []string{"no blocking AI conformance findings detected"}
	}
	seen := map[string]struct{}{}
	var out []string
	for _, f := range findings {
		if _, ok := seen[f.GetRuleId()]; ok {
			continue
		}
		seen[f.GetRuleId()] = struct{}{}
		out = append(out, f.GetRemediation())
	}
	sort.Strings(out)
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
