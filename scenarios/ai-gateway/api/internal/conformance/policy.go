package conformance

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	conformancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/conformance"
)

var (
	ollamaDimensionPattern = regexp.MustCompile(`\b(DefaultVectorSize|vectorSize|DenseSize|EmbedDimensions|vector_size)\s*[:=]\s*(768|1024|1536)\b|"size"\s*:\s*(768|1024|1536)\b|(?i)\bvector\s*\(\s*(768|1024|1536)\s*\)`)
	ollamaModelPattern     = regexp.MustCompile(`\b(nomic-embed-text(?::latest)?|qwen3:[0-9]+(?:\.[0-9]+)?b|llama3\.2(?::[0-9]+b)?|qwen2\.5(?::[0-9]+b)?|codellama(?::[0-9]+b)?|mistral(?::[0-9]+b)?)\b`)
	openRouterSlugPattern  = regexp.MustCompile(`"(?:openai|anthropic|google|x-ai|deepseek|mistralai|meta-llama|qwen|z-ai|minimax|bytedance-seed|moonshotai|recraft|sourceful|black-forest-labs|inception|microsoft|alibaba)/[a-z0-9._:-]*(?:gpt|claude|gemini|llama|grok|deepseek|mixtral|mistral|qwen|glm|kimi|seedream|flux|recraft|riverflow|mercury|nano-banana|veo|seed-2)[a-z0-9._:-]*"`)
	openRouterModelEnv     = regexp.MustCompile(`\b[A-Z0-9]*_?OPENROUTER_[A-Z0-9_]*MODEL\b`)
)

// ScanProject runs the three model-policy checks formerly owned by the
// repository contract checker. It intentionally scans runtime surfaces only;
// policy authority files and catalog/test surfaces are not caller defaults.
func (s *Scanner) ScanProject(ctx context.Context, root string) (ScanReport, error) {
	var findings []*conformancev1.ConformanceFinding
	if err := walkPolicyFiles(ctx, root, func(rel, line string, lineNo int) {
		if isPolicyAuthority(rel) {
			return
		}
		if isScenarioRuntime(rel) && (strings.Contains(line, `"/api/embeddings"`) || strings.Contains(line, `"/api/generate"`) || strings.Contains(line, `"/api/chat"`)) {
			findings = append(findings, policyFinding("AI_OLLAMA_GATEWAY_ONLY", "high", rel, lineNo, "Raw Ollama endpoint usage bypasses resource-ollama gateway policy.", "Use resource-ollama gateway or AI Gateway role routing."))
		}
		if isScenarioRuntime(rel) && (ollamaDimensionPattern.MatchString(line) || ollamaModelPattern.MatchString(line)) {
			findings = append(findings, policyFinding("AI_OLLAMA_POLICY_FACTS", "high", rel, lineNo, "Local Ollama model or embedding policy fact is hard-coded.", "Resolve model and embedding metadata through resource-ollama policy."))
		}
		if isOpenRouterRuntime(rel) && !isPolicyAuthority(rel) {
			code := codePortion(line)
			if openRouterSlugPattern.MatchString(code) || openRouterModelEnv.MatchString(code) || (strings.Contains(code, "resource-openrouter") && strings.Contains(code, "generate") && strings.Contains(code, "--model")) {
				findings = append(findings, policyFinding("AI_OPENROUTER_POLICY_FACTS", "high", rel, lineNo, "Concrete OpenRouter model policy is hard-coded in a runtime surface.", "Resolve a role through resource-openrouter policy instead of selecting a concrete model."))
			}
		}
	}); err != nil {
		return ScanReport{}, err
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].GetPath() != findings[j].GetPath() {
			return findings[i].GetPath() < findings[j].GetPath()
		}
		return findings[i].GetRuleId() < findings[j].GetRuleId()
	})
	return ScanReport{Scenario: "repo", MaturityLevel: maturityLevel(findings), Findings: findings, Recommendations: recommendations(findings)}, nil
}

func walkPolicyFiles(ctx context.Context, root string, visit func(string, string, int)) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "node_modules" || entry.Name() == "vendor" || entry.Name() == "build" || entry.Name() == "dist" || entry.Name() == "coverage" || entry.Name() == "data" || strings.HasPrefix(filepath.ToSlash(path), filepath.ToSlash(filepath.Join(root, "resources", "ollama"))) || strings.HasPrefix(filepath.ToSlash(path), filepath.ToSlash(filepath.Join(root, "resources", "openrouter"))) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil || !policyFile(rel) {
			return nil
		}
		file, openErr := os.Open(path)
		if openErr != nil {
			return openErr
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
		lineNo := 0
		for scanner.Scan() {
			lineNo++
			visit(filepath.ToSlash(rel), scanner.Text(), lineNo)
		}
		return scanner.Err()
	})
}

func policyFile(rel string) bool {
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".go", ".json", ".yaml", ".yml", ".sh", ".sql", ".ts", ".tsx", ".js":
		return true
	default:
		return false
	}
}

func isScenarioRuntime(rel string) bool {
	parts := strings.Split(rel, "/")
	return len(parts) >= 3 && parts[0] == "scenarios" && (parts[2] == "api" || parts[2] == "cli")
}

func isOpenRouterRuntime(rel string) bool {
	parts := strings.Split(rel, "/")
	if len(parts) >= 3 && parts[0] == "scenarios" {
		return parts[2] == "api" || parts[2] == "cli" || parts[2] == "ui"
	}
	return strings.HasPrefix(rel, "resources/")
}

func isPolicyAuthority(rel string) bool {
	return rel == "resources/ollama/model-policy.json" || strings.HasPrefix(rel, "resources/ollama/cli/internal/policy/") || rel == "resources/openrouter/model-policy.json" || strings.HasSuffix(rel, "_test.go") || strings.HasSuffix(rel, ".test.ts") || strings.HasSuffix(rel, ".test.tsx") || strings.HasSuffix(rel, ".spec.ts") || strings.HasSuffix(rel, ".spec.tsx") || strings.HasPrefix(rel, "scenarios/agent-manager/") || strings.HasPrefix(rel, "scenarios/ai-gateway/")
}

func codePortion(line string) string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "#") {
		return ""
	}
	if idx := strings.Index(line, "//"); idx >= 0 {
		return line[:idx]
	}
	return line
}

func policyFinding(code, severity, rel string, line int, message, remediation string) *conformancev1.ConformanceFinding {
	return &conformancev1.ConformanceFinding{RuleId: code, Severity: severity, Path: rel + ":" + strconv.Itoa(line), Message: message, Remediation: remediation}
}
