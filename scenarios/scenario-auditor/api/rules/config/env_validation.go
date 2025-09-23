package config

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

/*
Rule: Environment Variable Validation
Description: Ensures proper handling and validation of environment variables
Reason: Unvalidated environment variables can lead to security issues and runtime errors
Category: config
Severity: medium
Standard: OWASP
Targets: api, cli, ui

<test-case id="missing-env-validation" should-fail="true" path="api/service/main.go">
  <description>Direct environment variable usage without validation</description>
  <input language="go">
func setupDatabase() {
    dbURL := os.Getenv("DATABASE_URL")
    db, err := sql.Open("postgres", dbURL)
    if err != nil {
        panic(err)
    }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>DATABASE_URL</expected-message>
</test-case>

<test-case id="proper-env-validation" should-fail="false" path="api/service/main.go">
  <description>Environment variable with proper validation</description>
  <input language="go">
func setupDatabase() {
    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        log.Fatal("DATABASE_URL environment variable is required")
    }
    db, err := sql.Open("postgres", dbURL)
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }
}
  </input>
</test-case>

<test-case id="sensitive-env-logging" should-fail="true" path="api/service/main.go">
  <description>Logging sensitive environment variables</description>
  <input language="go">
func configureAPI() {
    apiKey := os.Getenv("API_KEY")
    log.Printf("Using API key: %s", apiKey)
    client := NewClient(apiKey)
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Sensitive environment variable logged</expected-message>
</test-case>

<test-case id="env-with-defaults" should-fail="false" path="api/service/main.go">
  <description>Environment variable with sensible defaults</description>
  <input language="go">
func getPort() string {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    return port
}
  </input>
</test-case>

<test-case id="cli-missing-validation" should-fail="true" path="cli/setup.sh">
  <description>Bash script using env var without validation</description>
  <input language="bash">
#!/usr/bin/env bash
PORT="${PORT}"
echo "Starting on ${PORT}"
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>PORT</expected-message>
</test-case>

<test-case id="cli-proper-validation" should-fail="false" path="cli/setup.sh">
  <description>Bash script validating required env var</description>
  <input language="bash">
#!/usr/bin/env bash
if [[ -z "${PORT}" ]]; then
  echo "PORT is required"
  exit 1
fi
echo "Listening on ${PORT}"
  </input>
</test-case>

<test-case id="cli-sensitive-logging" should-fail="true" path="cli/setup.sh">
  <description>Logging sensitive environment variable in bash</description>
  <input language="bash">
#!/usr/bin/env bash
API_KEY="${API_KEY}"
echo "API_KEY=${API_KEY}"
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Sensitive environment variable logged</expected-message>
</test-case>

<test-case id="ui-missing-validation" should-fail="true" path="ui/src/config.ts">
  <description>TypeScript usage without validation</description>
  <input language="typescript">
export const apiUrl = process.env.API_URL;
fetch(apiUrl ?? '');
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>API_URL</expected-message>
</test-case>

<test-case id="ui-proper-validation" should-fail="false" path="ui/src/config.ts">
  <description>TypeScript validating env var before use</description>
  <input language="typescript">
export const apiUrl = process.env.API_URL;
if (!apiUrl) {
  throw new Error('API_URL environment variable is required');
}
  </input>
</test-case>

<test-case id="ui-sensitive-logging" should-fail="true" path="ui/src/config.ts">
  <description>Logging sensitive env var in JavaScript</description>
  <input language="javascript">
const apiKey = process.env.API_KEY;
console.log('API key:', apiKey);
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Sensitive environment variable logged</expected-message>
</test-case>
*/

const (
	langGo   = "go"
	langBash = "bash"
	langJS   = "js"
)

type EnvValidationRule struct{}

type envUsage struct {
	envVar    string
	assigned  string
	lineIndex int
}

type envAcc struct {
	first      int
	validated  bool
	logged     bool
	loggedLine int
}

var (
	goAssignPattern   = regexp.MustCompile(`(?i)([A-Za-z_][A-Za-z0-9_]*)\s*(?:,.*)?:=\s*os\.Getenv\(`)
	bashAssignPattern = regexp.MustCompile(`(?i)(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=\s*\$\{?([A-Z_][A-Z0-9_]*)`)
	bashVarPattern    = regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*)\}|\$([A-Z_][A-Z0-9_]*)`)
	jsAssignPattern   = regexp.MustCompile(`(?i)(?:const|let|var)\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*process\.env\.([A-Za-z_][A-Za-z0-9_]*)`)
	jsEnvPattern      = regexp.MustCompile(`process\.env\.([A-Za-z_][A-Za-z0-9_]*)`)
)

// Check analyzes code for proper environment variable handling across supported languages
func (r *EnvValidationRule) Check(content string, filepath string) ([]Violation, error) {
	language := detectLanguage(filepath, content)

	var violations []Violation
	switch language {
	case langGo:
		violations = checkGoEnvValidation(content, filepath)
	case langBash:
		violations = checkBashEnvValidation(content, filepath)
	case langJS:
		violations = checkJSEnvValidation(content, filepath)
	default:
		violations = append(violations, checkGoEnvValidation(content, filepath)...)
		violations = append(violations, checkBashEnvValidation(content, filepath)...)
		violations = append(violations, checkJSEnvValidation(content, filepath)...)
		violations = dedupeViolations(violations)
	}

	return violations, nil
}

func detectLanguage(path, content string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return langGo
	case ".sh", ".bash":
		return langBash
	case ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs":
		return langJS
	}

	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "#!/") && strings.Contains(trimmed, "bash") {
		return langBash
	}
	if strings.Contains(content, "process.env.") {
		return langJS
	}
	if strings.Contains(content, "os.Getenv(") {
		return langGo
	}
	return ""
}

func checkGoEnvValidation(content, filepath string) []Violation {
	if !strings.Contains(content, "os.Getenv(") {
		return nil
	}

	lines := strings.Split(content, "\n")
	acc := make(map[string]*envAcc)

	for i, line := range lines {
		if !strings.Contains(line, "os.Getenv(") {
			continue
		}

		envVar := extractEnvVarName(line)
		assignedVar := extractAssignedVarName(line)
		if assignedVar == "" {
			assignedVar = envVar
		}

		sensitiveVar := isSensitiveVar(envVar)
		hasValidation := false
		logLine := -1

		for j := i + 1; j < len(lines) && j < i+8; j++ {
			nextLine := strings.TrimSpace(lines[j])
			if nextLine == "" {
				continue
			}

			lower := strings.ToLower(nextLine)
			containsVar := strings.Contains(nextLine, assignedVar) || strings.Contains(lower, strings.ToLower(assignedVar))
			if strings.HasPrefix(lower, "if") && containsVar {
				if strings.Contains(nextLine, `== ""`) || strings.Contains(nextLine, `== ''`) ||
					strings.Contains(nextLine, "len("+assignedVar+") == 0") || strings.Contains(nextLine, "len("+strings.ToLower(assignedVar)+") == 0") {
					hasValidation = true
					break
				}
			}

			if (strings.Contains(nextLine, "log.Fatal") || strings.Contains(nextLine, "panic(") || strings.Contains(nextLine, "log.Fatalf")) && containsVar {
				hasValidation = true
				break
			}
		}

		if sensitiveVar {
			for j := i; j < len(lines) && j < i+10; j++ {
				logCandidate := lines[j]
				if strings.Contains(logCandidate, "log.") || strings.Contains(logCandidate, "fmt.Print") {
					if strings.Contains(logCandidate, assignedVar) || strings.Contains(logCandidate, envVar) {
						logLine = j
						break
					}
				}
			}
		}

		state := acc[envVar]
		if state == nil {
			state = &envAcc{first: i, loggedLine: -1}
			acc[envVar] = state
		} else if i < state.first {
			state.first = i
		}
		if hasValidation {
			state.validated = true
		}
		if sensitiveVar && logLine >= 0 {
			state.logged = true
			if state.loggedLine == -1 || logLine < state.loggedLine {
				state.loggedLine = logLine
			}
		}
	}

	seen := make(map[string]bool)
	var violations []Violation
	for envVar, state := range acc {
		logged := false
		if state.logged && state.loggedLine >= 0 {
			appendViolation(&violations, seen, newSensitiveViolation(filepath, state.loggedLine+1, envVar))
			logged = true
		}
		if !state.validated && !logged {
			appendViolation(&violations, seen, newValidationViolation(filepath, state.first+1, envVar))
		}
	}

	return violations
}

func checkBashEnvValidation(content, filepath string) []Violation {
	lines := strings.Split(content, "\n")
	usages := collectBashUsages(lines)
	if len(usages) == 0 {
		return nil
	}

	acc := make(map[string]*envAcc)

	for _, usage := range usages {
		envVar := strings.ToUpper(usage.envVar)
		sensitive := isSensitiveVar(envVar)
		hasValidation := bashHasValidation(lines, usage)
		logged, logLine := bashLogsSensitive(lines, usage)

		state := acc[envVar]
		if state == nil {
			state = &envAcc{first: usage.lineIndex, loggedLine: -1}
			acc[envVar] = state
		} else if usage.lineIndex < state.first {
			state.first = usage.lineIndex
		}

		if hasValidation {
			state.validated = true
		}
		if sensitive && logged {
			state.logged = true
			if state.loggedLine == -1 || logLine < state.loggedLine {
				state.loggedLine = logLine
			}
		}
	}

	seen := make(map[string]bool)
	var violations []Violation
	for envVar, state := range acc {
		logged := false
		if state.logged && state.loggedLine >= 0 {
			appendViolation(&violations, seen, newSensitiveViolation(filepath, state.loggedLine+1, envVar))
			logged = true
		}
		if !state.validated && !logged {
			appendViolation(&violations, seen, newValidationViolation(filepath, state.first+1, envVar))
		}
	}

	return violations
}

func checkJSEnvValidation(content, filepath string) []Violation {
	if !strings.Contains(content, "process.env.") {
		return nil
	}

	lines := strings.Split(content, "\n")
	usages := collectJSUsages(lines)
	if len(usages) == 0 {
		return nil
	}

	acc := make(map[string]*envAcc)

	for _, usage := range usages {
		envVar := strings.ToUpper(usage.envVar)
		sensitive := isSensitiveVar(envVar)
		hasValidation := jsHasValidation(lines, usage)
		logged, logLine := jsLogsSensitive(lines, usage)

		state := acc[envVar]
		if state == nil {
			state = &envAcc{first: usage.lineIndex, loggedLine: -1}
			acc[envVar] = state
		} else if usage.lineIndex < state.first {
			state.first = usage.lineIndex
		}

		if hasValidation {
			state.validated = true
		}
		if sensitive && logged {
			state.logged = true
			if state.loggedLine == -1 || logLine < state.loggedLine {
				state.loggedLine = logLine
			}
		}
	}

	seen := make(map[string]bool)
	var violations []Violation
	for envVar, state := range acc {
		logged := false
		if state.logged && state.loggedLine >= 0 {
			appendViolation(&violations, seen, newSensitiveViolation(filepath, state.loggedLine+1, envVar))
			logged = true
		}
		if !state.validated && !logged {
			appendViolation(&violations, seen, newValidationViolation(filepath, state.first+1, envVar))
		}
	}

	return violations
}

func collectBashUsages(lines []string) []envUsage {
	seen := make(map[string]bool)
	var result []envUsage

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		for _, match := range bashAssignPattern.FindAllStringSubmatch(line, -1) {
			assigned := match[1]
			envVar := strings.ToUpper(match[2])
			key := fmt.Sprintf("%d:%s:%s", i, envVar, assigned)
			if !seen[key] {
				result = append(result, envUsage{envVar: envVar, assigned: assigned, lineIndex: i})
				seen[key] = true
			}
		}

		for _, match := range bashVarPattern.FindAllStringSubmatch(line, -1) {
			envVar := match[1]
			if envVar == "" {
				envVar = match[2]
			}
			if envVar == "" {
				continue
			}
			envVar = strings.ToUpper(envVar)
			key := fmt.Sprintf("%d:%s", i, envVar)
			if !seen[key] {
				result = append(result, envUsage{envVar: envVar, assigned: envVar, lineIndex: i})
				seen[key] = true
			}
		}
	}

	return result
}

func collectJSUsages(lines []string) []envUsage {
	seen := make(map[string]bool)
	var result []envUsage

	for i, line := range lines {
		for _, match := range jsAssignPattern.FindAllStringSubmatch(line, -1) {
			assigned := match[1]
			envVar := strings.ToUpper(match[2])
			key := fmt.Sprintf("%d:%s:%s", i, envVar, assigned)
			if !seen[key] {
				result = append(result, envUsage{envVar: envVar, assigned: assigned, lineIndex: i})
				seen[key] = true
			}
		}

		for _, match := range jsEnvPattern.FindAllStringSubmatch(line, -1) {
			envVar := strings.ToUpper(match[1])
			key := fmt.Sprintf("%d:%s", i, envVar)
			if !seen[key] {
				result = append(result, envUsage{envVar: envVar, assigned: envVar, lineIndex: i})
				seen[key] = true
			}
		}
	}

	return result
}

func bashHasValidation(lines []string, usage envUsage) bool {
	line := lines[usage.lineIndex]
	envVar := usage.envVar

	if strings.Contains(strings.ToLower(line), strings.ToLower("${"+envVar+":?")) {
		return true
	}

	start := usage.lineIndex
	end := usage.lineIndex + 8
	if end > len(lines) {
		end = len(lines)
	}

	for j := start; j < end; j++ {
		trimmed := strings.TrimSpace(lines[j])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		lower := strings.ToLower(trimmed)
		envLower := strings.ToLower(envVar)
		if strings.Contains(lower, "${"+envLower+":?") {
			return true
		}
		if strings.Contains(lower, "${"+envLower+"?") {
			return true
		}
		if strings.Contains(lower, "require_env "+envLower) {
			return true
		}
		if (strings.Contains(lower, "[[") || strings.Contains(lower, "[")) && (strings.Contains(lower, "-z") || strings.Contains(lower, "!")) {
			if containsVarReference(trimmed, envVar) {
				return true
			}
		}
	}

	return false
}

func bashLogsSensitive(lines []string, usage envUsage) (bool, int) {
	envVar := usage.envVar
	start := usage.lineIndex
	end := usage.lineIndex + 8
	if end > len(lines) {
		end = len(lines)
	}

	for j := start; j < end; j++ {
		line := lines[j]
		lower := strings.ToLower(line)
		if !(strings.Contains(lower, "echo") || strings.Contains(lower, "printf") || strings.Contains(lower, "logger")) {
			continue
		}
		if containsVarReference(line, envVar) {
			return true, j
		}
	}

	return false, -1
}

func jsHasValidation(lines []string, usage envUsage) bool {
	assigned := usage.assigned
	envVar := usage.envVar

	start := usage.lineIndex
	end := usage.lineIndex + 8
	if end > len(lines) {
		end = len(lines)
	}

	for j := start; j < end; j++ {
		trimmed := strings.TrimSpace(lines[j])
		if trimmed == "" {
			continue
		}

		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "if") {
			if assigned != "" && strings.Contains(trimmed, assigned) {
				if strings.Contains(trimmed, "!"+assigned) ||
					strings.Contains(lower, strings.ToLower(assigned)+" === undefined") ||
					strings.Contains(lower, strings.ToLower(assigned)+" == null") ||
					strings.Contains(lower, strings.ToLower(assigned)+"?.length === 0") {
					return true
				}
			}
			if strings.Contains(lower, "process.env."+strings.ToLower(envVar)) && strings.Contains(lower, "!") {
				return true
			}
		}
	}

	return false
}

func jsLogsSensitive(lines []string, usage envUsage) (bool, int) {
	assigned := usage.assigned
	envVar := usage.envVar
	start := usage.lineIndex
	end := usage.lineIndex + 8
	if end > len(lines) {
		end = len(lines)
	}

	for j := start; j < end; j++ {
		line := lines[j]
		lower := strings.ToLower(line)
		if !(strings.Contains(lower, "console.log") || strings.Contains(lower, "console.error") || strings.Contains(lower, "console.info") || strings.Contains(lower, "logger")) {
			continue
		}
		if (assigned != "" && strings.Contains(line, assigned)) || strings.Contains(line, "process.env."+envVar) {
			return true, j
		}
	}

	return false, -1
}

func appendViolation(violations *[]Violation, seen map[string]bool, violation Violation) {
	key := fmt.Sprintf("%s:%d:%s", violation.FilePath, violation.LineNumber, violation.Message)
	if seen[key] {
		return
	}
	seen[key] = true
	*violations = append(*violations, violation)
}

func dedupeViolations(list []Violation) []Violation {
	seen := make(map[string]bool)
	var result []Violation
	for _, v := range list {
		key := fmt.Sprintf("%s:%d:%s", v.FilePath, v.LineNumber, v.Message)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, v)
	}
	return result
}

func newValidationViolation(file string, line int, envVar string) Violation {
	msg := "Environment variable used without validation: " + envVar
	return Violation{
		RuleID:         "env_validation",
		Type:           "env_validation",
		Severity:       "medium",
		Title:          "Environment variable missing validation",
		Message:        msg,
		Description:    msg,
		FilePath:       file,
		LineNumber:     line,
		Standard:       "OWASP",
		Category:       "config",
		Recommendation: "Validate the environment variable and fail fast when it is absent",
	}
}

func newSensitiveViolation(file string, line int, envVar string) Violation {
	msg := "Sensitive environment variable logged: " + envVar
	return Violation{
		RuleID:         "env_validation",
		Type:           "env_validation",
		Severity:       "high",
		Title:          "Sensitive environment variable logged",
		Message:        msg,
		Description:    msg,
		FilePath:       file,
		LineNumber:     line,
		Standard:       "OWASP",
		Category:       "config",
		Recommendation: "Remove sensitive values from logs and redact secrets before printing",
	}
}

func containsVarReference(line, envVar string) bool {
	if strings.Contains(line, "${"+envVar+"}") {
		return true
	}

	needle := "$" + envVar
	idx := strings.Index(line, needle)
	for idx != -1 {
		end := idx + len(needle)
		if end == len(line) || !isIdentifierChar(rune(line[end])) {
			if idx == 0 || !isIdentifierChar(rune(line[idx-1])) {
				return true
			}
		}
		next := strings.Index(line[end:], needle)
		if next == -1 {
			break
		}
		idx = end + next
	}
	return false
}

func isIdentifierChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

func extractEnvVarName(line string) string {
	start := strings.Index(line, `"`)
	if start == -1 {
		start = strings.Index(line, `'`)
	}
	if start == -1 {
		return "UNKNOWN"
	}

	end := strings.Index(line[start+1:], `"`)
	if end == -1 {
		end = strings.Index(line[start+1:], `'`)
	}
	if end == -1 {
		return "UNKNOWN"
	}

	return line[start+1 : start+1+end]
}

func extractAssignedVarName(line string) string {
	matches := goAssignPattern.FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func isSensitiveVar(name string) bool {
	sensitive := []string{
		"PASSWORD", "SECRET", "KEY", "TOKEN", "API_KEY", "PRIVATE",
		"CREDENTIAL", "AUTH", "CERTIFICATE", "CERT",
	}

	upperName := strings.ToUpper(name)
	for _, s := range sensitive {
		if strings.Contains(upperName, s) {
			return true
		}
	}

	return false
}
