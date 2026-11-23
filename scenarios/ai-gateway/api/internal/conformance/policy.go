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
	ollamaLocalURLPattern  = regexp.MustCompile(`(?i)\bhttps?://(?:localhost|127\.0\.0\.1):11434(?:/|\b)`)
	ollamaModelPattern     = regexp.MustCompile(`\b(nomic-embed-text(?::latest)?|qwen3:[0-9]+(?:\.[0-9]+)?b|llama3\.2(?::[0-9]+b)?|qwen2\.5(?::[0-9]+b)?|codellama(?::[0-9]+b)?|mistral(?::[0-9]+b)?)\b`)
	openRouterSlugPattern  = regexp.MustCompile(`"(?:openai|anthropic|google|x-ai|deepseek|mistralai|meta-llama|qwen|z-ai|minimax|bytedance-seed|moonshotai|recraft|sourceful|black-forest-labs|inception|microsoft|alibaba)/[a-z0-9._:-]*(?:gpt|claude|gemini|llama|grok|deepseek|mixtral|mistral|qwen|glm|kimi|seedream|flux|recraft|riverflow|mercury|nano-banana|veo|seed-2)[a-z0-9._:-]*"`)
	openRouterModelEnv     = regexp.MustCompile(`\b[A-Z0-9]*_?OPENROUTER_[A-Z0-9_]*MODEL\b`)
	// ollamaRuntimeOptionPattern matches Ollama runner load options (the
	// api.Runner fields) and the retired per-request gateway flags. A caller
	// that sets one forces Ollama to restart the shared runner whenever it
	// disagrees with another caller; resources/ollama runtimeprofile owns them.
	ollamaRuntimeOptionPattern = regexp.MustCompile(`"(num_ctx|num_batch|num_gpu|main_gpu|use_mmap|num_thread)"|--num-(ctx|thread|batch|gpu)\b`)
)

// ScanProject runs the three model-policy checks formerly owned by the
// repository contract checker. It intentionally scans runtime surfaces only;
// policy authority files and catalog/test surfaces are not caller defaults.
func (s *Scanner) ScanProject(ctx context.Context, root string) (ScanReport, error) {
	var findings []*conformancev1.ConformanceFinding
	if err := walkPolicyFiles(ctx, root, func(rel, line string, lineNo int) {
		findings = append(findings, policyLineFindings(rel, line, lineNo)...)
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

func policyLineFindings(rel, line string, lineNo int) []*conformancev1.ConformanceFinding {
	if isPolicyAuthority(rel) {
		return nil
	}
	code := codePortion(line)
	var findings []*conformancev1.ConformanceFinding
	if isScenarioRuntime(rel) {
		findings = append(findings, ollamaPolicyFindings(rel, code, lineNo)...)
	}
	if isOllamaRuntimeOptionSurface(rel) && ollamaRuntimeOptionPattern.MatchString(code) {
		findings = append(findings, policyFinding("AI_OLLAMA_RUNTIME_OPTIONS", "high", rel, lineNo, "An Ollama runner load option is set by a caller; differing values make Ollama restart the shared runner.", "Remove it. resource-ollama's runtime profile applies runner options to every caller; use gateway --context-budget for admission limits."))
	}
	if isOpenRouterRuntime(rel) && hasOpenRouterPolicyFact(code) {
		findings = append(findings, policyFinding("AI_OPENROUTER_POLICY_FACTS", "high", rel, lineNo, "Concrete OpenRouter model policy is hard-coded in a runtime surface.", "Resolve a role through resource-openrouter policy instead of selecting a concrete model."))
	}
	return findings
}

func ollamaPolicyFindings(rel, code string, lineNo int) []*conformancev1.ConformanceFinding {
	var findings []*conformancev1.ConformanceFinding
	if strings.Contains(code, `"/api/embeddings"`) || strings.Contains(code, `"/api/generate"`) || strings.Contains(code, `"/api/chat"`) || ollamaLocalURLPattern.MatchString(code) {
		findings = append(findings, policyFinding("AI_OLLAMA_GATEWAY_ONLY", "high", rel, lineNo, "Raw Ollama endpoint usage bypasses resource-ollama gateway policy.", "Use resource-ollama gateway or AI Gateway role routing."))
	}
	if ollamaDimensionPattern.MatchString(code) || containsOllamaModelFact(code) {
		findings = append(findings, policyFinding("AI_OLLAMA_POLICY_FACTS", "high", rel, lineNo, "Local Ollama model or embedding policy fact is hard-coded.", "Resolve model and embedding metadata through resource-ollama policy."))
	}
	return findings
}

func hasOpenRouterPolicyFact(code string) bool {
	return openRouterSlugPattern.MatchString(code) || openRouterModelEnv.MatchString(code) ||
		(strings.Contains(code, "resource-openrouter") && strings.Contains(code, "generate") && strings.Contains(code, "--model"))
}

func walkPolicyFiles(ctx context.Context, root string, visit func(string, string, int)) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		return walkPolicyEntry(ctx, root, visit, path, entry, err)
	})
}

func walkPolicyEntry(ctx context.Context, root string, visit func(string, string, int), path string, entry os.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if entry.IsDir() {
		return walkPolicyDirectory(root, path, entry.Name())
	}
	return walkPolicyFile(root, path, visit)
}

func walkPolicyDirectory(root, path, name string) error {
	if skipPolicyDirectory(path, name, root) {
		return filepath.SkipDir
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || !policyRuntimeDirectory(rel) {
		return filepath.SkipDir
	}
	return nil
}

func skipPolicyDirectory(path, name, root string) bool {
	switch name {
	case ".git", "node_modules", "vendor", "build", "dist", "coverage", "data":
		return true
	}
	path = filepath.ToSlash(path)
	return strings.HasPrefix(path, filepath.ToSlash(filepath.Join(root, "resources", "ollama"))) ||
		strings.HasPrefix(path, filepath.ToSlash(filepath.Join(root, "resources", "openrouter")))
}

func walkPolicyFile(root, path string, visit func(string, string, int)) error {
	rel, err := filepath.Rel(root, path)
	if err != nil || !policyRuntimeFile(rel) {
		return nil
	}
	file, err := os.Open(path)
	if err != nil {
		return err
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
}

func policyRuntimeDirectory(rel string) bool {
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return true
	}
	parts := strings.Split(rel, "/")
	if parts[0] == "resources" {
		return true
	}
	if parts[0] != "scenarios" {
		return false
	}
	if len(parts) <= 2 {
		return true
	}
	switch parts[2] {
	case "api", "cli", "ui":
		return true
	default:
		return false
	}
}

// policyRuntimeFile narrows the project walk to executable scenario/resource
// surfaces before opening files. Test fixtures, generated evidence, and docs
// can contain arbitrarily large lines and do not select production policy.
func policyRuntimeFile(rel string) bool {
	if !policyFile(rel) || isPolicyAuthority(rel) || strings.HasSuffix(filepath.ToSlash(rel), ".freshness.json") {
		return false
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for _, part := range parts {
		switch strings.ToLower(part) {
		case "testdata", "fixtures", "evidence", "reports", "coverage", "docs", "examples":
			return false
		}
	}
	return isScenarioRuntime(rel) || isOpenRouterRuntime(rel)
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

// isOllamaRuntimeOptionSurface covers every caller that can reach Ollama:
// scenario api/cli code and other resources (resources/ollama is skipped by
// the walk because its runtime profile is the authority).
func isOllamaRuntimeOptionSurface(rel string) bool {
	return isScenarioRuntime(rel) || strings.HasPrefix(rel, "resources/")
}

func isOpenRouterRuntime(rel string) bool {
	parts := strings.Split(rel, "/")
	if len(parts) >= 3 && parts[0] == "scenarios" {
		return parts[2] == "api" || parts[2] == "cli" || parts[2] == "ui"
	}
	return strings.HasPrefix(rel, "resources/")
}

func isPolicyAuthority(rel string) bool {
	return isResourcePolicyAuthority(rel) || isScenarioPolicyAuthority(rel) || isPolicyTestFile(rel)
}

func isResourcePolicyAuthority(rel string) bool {
	return strings.HasPrefix(rel, "resources/") && filepath.Base(rel) == "model-policy.json" ||
		strings.HasPrefix(rel, "resources/ollama/cli/internal/policy/")
}

func isScenarioPolicyAuthority(rel string) bool {
	return rel == "scenarios/image-tools/api/internal/models/registry.seed.json" ||
		strings.HasPrefix(rel, "scenarios/agent-manager/") || strings.HasPrefix(rel, "scenarios/ai-gateway/")
}

func isPolicyTestFile(rel string) bool {
	return strings.HasSuffix(rel, "_test.go") || strings.HasSuffix(rel, ".test.ts") ||
		strings.HasSuffix(rel, ".test.tsx") || strings.HasSuffix(rel, ".spec.ts") || strings.HasSuffix(rel, ".spec.tsx")
}

func containsOllamaModelFact(line string) bool {
	for _, match := range ollamaModelPattern.FindAllStringIndex(line, -1) {
		// Do not treat a provider/model path or a longer catalog slug such as
		// "mistral-ocr" / "qwen2.5-vl-3b" as an Ollama model identifier.
		if match[1] < len(line) && (line[match[1]] == '-' || line[match[1]] == '/' || line[match[1]] == '_') {
			continue
		}
		return true
	}
	return false
}

func codePortion(line string) string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "#") {
		return ""
	}
	var quote byte
	escaped := false
	for i := 0; i < len(line); i++ {
		current := line[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if current == '\\' {
				escaped = true
				continue
			}
			if current == quote {
				quote = 0
			}
			continue
		}
		switch current {
		case '"', '\'', '`':
			quote = current
		case '/':
			if i+1 < len(line) && line[i+1] == '/' {
				return line[:i]
			}
		}
	}
	return line
}

func policyFinding(code, severity, rel string, line int, message, remediation string) *conformancev1.ConformanceFinding {
	return &conformancev1.ConformanceFinding{RuleId: code, Severity: severity, Path: rel + ":" + strconv.Itoa(line), Message: message, Remediation: remediation}
}
