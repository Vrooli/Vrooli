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

<test-case id="port-with-dangerous-default" should-fail="true" path="api/service/main.go">
  <description>PORT variable with dangerous default (ports are dynamically allocated)</description>
  <input language="go">
func getPort() string {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    return port
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>PORT</expected-message>
</test-case>

<test-case id="app-name-with-safe-default" should-fail="false" path="api/service/main.go">
  <description>Non-port/non-sensitive variable with safe default</description>
  <input language="go">
func getAppName() string {
    name := os.Getenv("APP_NAME")
    if name == "" {
        name = "default-app"
    }
    return name
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

<test-case id="multi-var-declaration" should-fail="true" path="api/service/main.go">
  <description>Multiple environment variables in single declaration with partial validation</description>
  <input language="go">
func setup() {
    host, port := os.Getenv("HOST"), os.Getenv("PORT")
    if host == "" {
        log.Fatal("HOST is required")
    }
    // PORT is never validated - should be caught
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>PORT</expected-message>
</test-case>

<test-case id="multi-var-all-validated" should-fail="false" path="api/service/main.go">
  <description>Multiple environment variables with all validated</description>
  <input language="go">
func setup() {
    host, port := os.Getenv("HOST"), os.Getenv("PORT")
    if host == "" {
        log.Fatal("HOST is required")
    }
    if port == "" {
        log.Fatal("PORT is required")
    }
}
  </input>
</test-case>

<test-case id="multi-var-with-error" should-fail="false" path="api/service/main.go">
  <description>Multi-variable with error return properly validated</description>
  <input language="go">
func setup() {
    value, ok := os.LookupEnv("CONFIG")
    if !ok {
        log.Fatal("CONFIG is required")
    }
}
  </input>
</test-case>

<test-case id="commented-validation" should-fail="true" path="api/service/main.go">
  <description>Commented-out validation is intentionally detected as a security risk</description>
  <input language="go">
func setup() {
    apiKey := os.Getenv("API_KEY")
    // if apiKey == "" { log.Fatal("API_KEY required") }
    // TODO: uncomment before deploy
    useKey(apiKey)
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>API_KEY</expected-message>
</test-case>

<test-case id="optional-debug-no-validation" should-fail="false" path="api/service/main.go">
  <description>Optional DEBUG variable needs no validation (empty is valid)</description>
  <input language="go">
func isDebugMode() bool {
    debug := os.Getenv("DEBUG")
    return debug == "true"
}
  </input>
</test-case>

<test-case id="api-key-dangerous-default" should-fail="true" path="api/service/main.go">
  <description>API_KEY with dangerous hardcoded default</description>
  <input language="go">
func getAPIKey() string {
    apiKey := os.Getenv("API_KEY")
    if apiKey == "" {
        apiKey = "default-key-123"
    }
    return apiKey
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>API_KEY</expected-message>
</test-case>

<test-case id="database-url-dangerous-default" should-fail="true" path="api/service/main.go">
  <description>DATABASE_URL with dangerous default</description>
  <input language="go">
func connectDB() {
    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        dbURL = "postgres://localhost/defaultdb"
    }
    connect(dbURL)
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>DATABASE_URL</expected-message>
</test-case>

<test-case id="bash-port-default" should-fail="true" path="cli/setup.sh">
  <description>Bash PORT with dangerous default substitution</description>
  <input language="bash">
#!/usr/bin/env bash
PORT="${PORT:-8080}"
echo "Starting on ${PORT}"
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>PORT</expected-message>
</test-case>

<test-case id="bash-debug-default" should-fail="false" path="cli/setup.sh">
  <description>Bash DEBUG with safe default (optional variable)</description>
  <input language="bash">
#!/usr/bin/env bash
DEBUG="${DEBUG:-false}"
if [ "$DEBUG" = "true" ]; then
  set -x
fi
  </input>
</test-case>

<test-case id="js-port-default" should-fail="true" path="ui/src/config.ts">
  <description>JavaScript PORT with dangerous default using OR operator</description>
  <input language="javascript">
const port = process.env.PORT || '8080';
server.listen(port);
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>PORT</expected-message>
</test-case>

<test-case id="js-debug-default" should-fail="false" path="ui/src/config.ts">
  <description>JavaScript DEBUG with safe default (optional variable)</description>
  <input language="javascript">
const debug = process.env.DEBUG || 'false';
if (debug === 'true') {
  console.log('Debug mode enabled');
}
  </input>
</test-case>

<test-case id="comment-override-optional" should-fail="false" path="api/service/main.go">
  <description>Comment override makes PORT optional</description>
  <input language="go">
func getPort() string {
    // vrooli:env:optional
    port := os.Getenv("CUSTOM_PORT")
    if port == "" {
        port = "8080"
    }
    return port
}
  </input>
</test-case>

<test-case id="comment-override-no-defaults" should-fail="true" path="api/service/main.go">
  <description>Comment override enforces no-defaults</description>
  <input language="go">
func getConfig() string {
    // vrooli:env:no-defaults
    config := os.Getenv("CUSTOM_VAR")
    if config == "" {
        config = "default"
    }
    return config
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>CUSTOM_VAR</expected-message>
</test-case>
*/

const (
	langGo   = "go"
	langBash = "bash"
	langJS   = "js"
)

// VarCategory represents the category of an environment variable
type VarCategory int

const (
	// CategoryNoDefaults: Variables where default values are dangerous (ports, secrets)
	CategoryNoDefaults VarCategory = iota
	// CategoryValidationRequired: Variables that need validation but can have safe defaults
	CategoryValidationRequired
	// CategoryOptional: Variables where empty is a valid value
	CategoryOptional
)

type EnvValidationRule struct{}

type envUsage struct {
	envVar    string
	assigned  string
	lineIndex int
}

type envAcc struct {
	first         int
	validated     bool
	logged        bool
	loggedLine    int
	hasDefault    bool
	defaultLine   int
	defaultValue  string
	category      VarCategory
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

		// Extract ALL env vars and assigned vars from this line
		envVars := extractAllEnvVarNames(line)
		assignedVars := extractAllAssignedVarNames(line)

		// Handle cases where we have different counts
		// If we have multiple Getenv calls but only one assigned var,
		// that var gets the first env var, others are tracked independently
		pairs := make([]struct{ envVar, assignedVar string }, 0)

		if len(assignedVars) == 0 {
			// Direct usage like: if os.Getenv("VAR") == ""
			for _, envVar := range envVars {
				pairs = append(pairs, struct{ envVar, assignedVar string }{envVar, envVar})
			}
		} else if len(envVars) == len(assignedVars) {
			// Perfect match: host, port := os.Getenv("HOST"), os.Getenv("PORT")
			for idx, envVar := range envVars {
				pairs = append(pairs, struct{ envVar, assignedVar string }{envVar, assignedVars[idx]})
			}
		} else if len(envVars) == 1 && len(assignedVars) > 1 {
			// Case like: val, err := os.Getenv("KEY")
			// Use first assigned var that's not underscore
			for _, assignedVar := range assignedVars {
				if assignedVar != "_" {
					pairs = append(pairs, struct{ envVar, assignedVar string }{envVars[0], assignedVar})
					break
				}
			}
		} else {
			// Fallback: pair in order, use env var name for extras
			for idx, envVar := range envVars {
				if idx < len(assignedVars) {
					pairs = append(pairs, struct{ envVar, assignedVar string }{envVar, assignedVars[idx]})
				} else {
					pairs = append(pairs, struct{ envVar, assignedVar string }{envVar, envVar})
				}
			}
		}

		// Process each pair
		for _, pair := range pairs {
			envVar := pair.envVar
			assignedVar := pair.assignedVar

			// Categorize the variable
			category := categorizeEnvVar(envVar, lines, i)
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

			// Check for dangerous defaults
			hasDefault, defaultLine, defaultValue := detectsDefaultAssignment(lines, i, assignedVar)

			state := acc[envVar]
			if state == nil {
				state = &envAcc{first: i, loggedLine: -1, defaultLine: -1, category: category}
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
			if hasDefault {
				state.hasDefault = true
				state.defaultValue = defaultValue
				if state.defaultLine == -1 || defaultLine < state.defaultLine {
					state.defaultLine = defaultLine
				}
			}
		}
	}

	seen := make(map[string]bool)
	var violations []Violation
	for envVar, state := range acc {
		// Check for sensitive logging violation (highest priority)
		logged := false
		if state.logged && state.loggedLine >= 0 {
			appendViolation(&violations, seen, newSensitiveViolation(filepath, state.loggedLine+1, envVar))
			logged = true
		}

		// Check for dangerous default violation
		if state.hasDefault && state.category == CategoryNoDefaults {
			appendViolation(&violations, seen, newDangerousDefaultViolation(filepath, state.defaultLine+1, envVar, state.defaultValue))
		} else if state.category == CategoryValidationRequired {
			// For validation-required vars, default counts as validation
			// So if hasDefault, we don't need to check for missing validation
			if !state.hasDefault && !state.validated && !logged {
				appendViolation(&violations, seen, newValidationViolation(filepath, state.first+1, envVar))
			}
		} else if state.category == CategoryNoDefaults {
			// For no-defaults category without default, still need validation
			if !state.validated && !logged {
				appendViolation(&violations, seen, newValidationViolation(filepath, state.first+1, envVar))
			}
		}
		// CategoryOptional: no violations for missing validation
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
		category := categorizeEnvVar(envVar, lines, usage.lineIndex)
		sensitive := isSensitiveVar(envVar)
		hasValidation := bashHasValidation(lines, usage)
		logged, logLine := bashLogsSensitive(lines, usage)

		// Check for bash default substitution syntax
		line := lines[usage.lineIndex]
		hasDefault, defaultValue := bashHasDefault(line, envVar)

		state := acc[envVar]
		if state == nil {
			state = &envAcc{first: usage.lineIndex, loggedLine: -1, defaultLine: -1, category: category}
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
		if hasDefault {
			state.hasDefault = true
			state.defaultValue = defaultValue
			if state.defaultLine == -1 || usage.lineIndex < state.defaultLine {
				state.defaultLine = usage.lineIndex
			}
		}
	}

	seen := make(map[string]bool)
	var violations []Violation
	for envVar, state := range acc {
		// Check for sensitive logging violation
		logged := false
		if state.logged && state.loggedLine >= 0 {
			appendViolation(&violations, seen, newSensitiveViolation(filepath, state.loggedLine+1, envVar))
			logged = true
		}

		// Check for dangerous default violation
		if state.hasDefault && state.category == CategoryNoDefaults {
			appendViolation(&violations, seen, newDangerousDefaultViolation(filepath, state.defaultLine+1, envVar, state.defaultValue))
		} else if state.category == CategoryValidationRequired {
			// For validation-required vars, default counts as validation
			if !state.hasDefault && !state.validated && !logged {
				appendViolation(&violations, seen, newValidationViolation(filepath, state.first+1, envVar))
			}
		} else if state.category == CategoryNoDefaults {
			// For no-defaults category without default, still need validation
			if !state.validated && !logged {
				appendViolation(&violations, seen, newValidationViolation(filepath, state.first+1, envVar))
			}
		}
		// CategoryOptional: no violations
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
		category := categorizeEnvVar(envVar, lines, usage.lineIndex)
		sensitive := isSensitiveVar(envVar)
		hasValidation := jsHasValidation(lines, usage)
		logged, logLine := jsLogsSensitive(lines, usage)

		// Check for JS/TS default patterns
		hasDefault, defaultLine, defaultValue := jsHasDefault(lines, usage.lineIndex, usage.assigned, envVar)

		state := acc[envVar]
		if state == nil {
			state = &envAcc{first: usage.lineIndex, loggedLine: -1, defaultLine: -1, category: category}
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
		if hasDefault {
			state.hasDefault = true
			state.defaultValue = defaultValue
			if state.defaultLine == -1 || defaultLine < state.defaultLine {
				state.defaultLine = defaultLine
			}
		}
	}

	seen := make(map[string]bool)
	var violations []Violation
	for envVar, state := range acc {
		// Check for sensitive logging violation
		logged := false
		if state.logged && state.loggedLine >= 0 {
			appendViolation(&violations, seen, newSensitiveViolation(filepath, state.loggedLine+1, envVar))
			logged = true
		}

		// Check for dangerous default violation
		if state.hasDefault && state.category == CategoryNoDefaults {
			appendViolation(&violations, seen, newDangerousDefaultViolation(filepath, state.defaultLine+1, envVar, state.defaultValue))
		} else if state.category == CategoryValidationRequired {
			// For validation-required vars, default counts as validation
			if !state.hasDefault && !state.validated && !logged {
				appendViolation(&violations, seen, newValidationViolation(filepath, state.first+1, envVar))
			}
		} else if state.category == CategoryNoDefaults {
			// For no-defaults category without default, still need validation
			if !state.validated && !logged {
				appendViolation(&violations, seen, newValidationViolation(filepath, state.first+1, envVar))
			}
		}
		// CategoryOptional: no violations
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

func newDangerousDefaultViolation(file string, line int, envVar string, defaultValue string) Violation {
	msg := fmt.Sprintf("Environment variable must not have default value: %s (default: %s)", envVar, defaultValue)
	return Violation{
		RuleID:         "env_validation",
		Type:           "dangerous_env_default",
		Severity:       "high",
		Title:          "Dangerous default value for critical environment variable",
		Message:        msg,
		Description:    msg,
		FilePath:       file,
		LineNumber:     line,
		Standard:       "OWASP",
		Category:       "config",
		Recommendation: "This variable requires explicit configuration. Fail fast when missing instead of using defaults. For ports and secrets, defaults are dangerous as they may conflict or expose security risks.",
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

// extractAllEnvVarNames extracts all environment variable names from os.Getenv() calls
func extractAllEnvVarNames(line string) []string {
	pattern := regexp.MustCompile(`os\.Getenv\(["']([^"']+)["']\)`)
	matches := pattern.FindAllStringSubmatch(line, -1)
	vars := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			vars = append(vars, match[1])
		}
	}
	return vars
}

// extractAllAssignedVarNames extracts all variable names from the left side of :=
func extractAllAssignedVarNames(line string) []string {
	// Find the := operator
	idx := strings.Index(line, ":=")
	if idx == -1 {
		return nil
	}

	// Get everything before :=
	leftSide := line[:idx]

	// Split by comma and extract variable names
	vars := make([]string, 0)
	for _, part := range strings.Split(leftSide, ",") {
		trimmed := strings.TrimSpace(part)
		// Extract just the variable name (handle cases like "var1, var2")
		if trimmed != "" && trimmed != "_" {
			vars = append(vars, trimmed)
		}
	}
	return vars
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

// isPortVar checks if a variable name refers to a port
func isPortVar(name string) bool {
	upper := strings.ToUpper(name)

	// Exact match
	if upper == "PORT" {
		return true
	}

	// Starts or ends with PORT + separator
	if strings.HasPrefix(upper, "PORT_") || strings.HasSuffix(upper, "_PORT") {
		return true
	}

	// Contains PORT with word boundaries (e.g., API_PORT, HTTP_PORT)
	// Avoid false positives like REPORT_PATH
	if strings.Contains(upper, "_PORT_") || strings.Contains(upper, "_PORT") {
		return true
	}

	return false
}

// isCriticalURLVar checks if a variable is a critical connection URL
func isCriticalURLVar(name string) bool {
	upper := strings.ToUpper(name)

	// Must end with _URL, _DSN, or _CONNECTION
	if !strings.HasSuffix(upper, "_URL") &&
	   !strings.HasSuffix(upper, "_DSN") &&
	   !strings.HasSuffix(upper, "_CONNECTION") {
		return false
	}

	// Must contain a database/service keyword
	criticalServices := []string{
		"DB", "DATABASE", "POSTGRES", "MYSQL", "MONGO", "REDIS",
		"ELASTIC", "KAFKA", "RABBIT", "MQ",
	}

	for _, service := range criticalServices {
		if strings.Contains(upper, service) {
			return true
		}
	}

	return false
}

// isOptionalVar checks if a variable is optional (empty is valid)
func isOptionalVar(name string) bool {
	upper := strings.ToUpper(name)

	// Exact matches for common optional vars
	optionalExact := []string{"DEBUG", "VERBOSE", "TRACE"}
	for _, opt := range optionalExact {
		if upper == opt {
			return true
		}
	}

	// Prefix patterns indicating optional behavior
	optionalPrefixes := []string{
		"DEBUG_", "VERBOSE_", "TRACE_",
		"FEATURE_", "ENABLE_", "DISABLE_",
		"OPT_", "OPTIONAL_",
		"DRY_RUN", "SKIP_", "NO_",
		"LOG_LEVEL", "LOG_FORMAT",
	}

	for _, prefix := range optionalPrefixes {
		if strings.HasPrefix(upper, prefix) {
			return true
		}
	}

	return false
}

// checkCommentOverride checks for comment-based category overrides
// Looks at the previous line for comments like: // vrooli:env:optional
func checkCommentOverride(lines []string, lineIndex int) (VarCategory, bool) {
	if lineIndex == 0 {
		return CategoryOptional, false
	}

	prevLine := strings.TrimSpace(lines[lineIndex-1])

	// Check for Go/JS style comments
	if strings.HasPrefix(prevLine, "//") {
		if strings.Contains(prevLine, "vrooli:env:optional") {
			return CategoryOptional, true
		}
		if strings.Contains(prevLine, "vrooli:env:no-defaults") {
			return CategoryNoDefaults, true
		}
		if strings.Contains(prevLine, "vrooli:env:required") {
			return CategoryValidationRequired, true
		}
	}

	// Check for bash/shell style comments
	if strings.HasPrefix(prevLine, "#") {
		if strings.Contains(prevLine, "vrooli:env:optional") {
			return CategoryOptional, true
		}
		if strings.Contains(prevLine, "vrooli:env:no-defaults") {
			return CategoryNoDefaults, true
		}
		if strings.Contains(prevLine, "vrooli:env:required") {
			return CategoryValidationRequired, true
		}
	}

	return CategoryOptional, false
}

// categorizeEnvVar determines the category of an environment variable
func categorizeEnvVar(name string, lines []string, lineIndex int) VarCategory {
	// First check for comment overrides (highest priority)
	if category, found := checkCommentOverride(lines, lineIndex); found {
		return category
	}

	// Check for NO_DEFAULTS_ALLOWED category
	if isSensitiveVar(name) || isPortVar(name) || isCriticalURLVar(name) {
		return CategoryNoDefaults
	}

	// Check for OPTIONAL category
	if isOptionalVar(name) {
		return CategoryOptional
	}

	// Default to VALIDATION_REQUIRED
	return CategoryValidationRequired
}

// detectsDefaultAssignment checks if a default value is assigned in a validation block (Go)
// Returns (hasDefault, lineNumber, defaultValue)
func detectsDefaultAssignment(lines []string, startLine int, assignedVar string) (bool, int, string) {
	// Look for validation block starting from startLine
	// Then scan inside the block for assignments to assignedVar

	maxLook := startLine + 8
	if maxLook > len(lines) {
		maxLook = len(lines)
	}

	inValidationBlock := false

	for i := startLine; i < maxLook; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		lower := strings.ToLower(line)
		containsVar := strings.Contains(line, assignedVar) || strings.Contains(lower, strings.ToLower(assignedVar))

		// Check if this is the start of a validation block
		if strings.HasPrefix(lower, "if") && containsVar {
			if strings.Contains(line, `== ""`) || strings.Contains(line, `== ''`) ||
				strings.Contains(line, "len("+assignedVar+") == 0") {
				inValidationBlock = true

				// Check same line for inline assignment: if x == "" { x = "default" }
				if strings.Contains(line, "{") && strings.Contains(line, assignedVar+" =") {
					// Extract value after =
					if val := extractGoDefaultValue(line, assignedVar); val != "" {
						return true, i, val
					}
				}
				continue
			}
		}

		// If we're in a validation block, look for assignments
		if inValidationBlock {
			// Check for closing brace (end of block)
			if strings.Contains(line, "}") && !strings.Contains(line, "{") {
				// Exit validation block
				break
			}

			// Look for assignment patterns - try to extract value
			// This handles: "varName = val", "varName=val", "varName := val"
			if val := extractGoDefaultValue(line, assignedVar); val != "" {
				// Make sure it's actually an assignment, not a comparison
				if !strings.Contains(line, "==") && !strings.Contains(line, "!=") {
					return true, i, val
				}
			}
		}
	}

	return false, -1, ""
}

// extractGoDefaultValue extracts the default value from a Go assignment line
func extractGoDefaultValue(line, varName string) string {
	// Look for patterns: varName = "value" or varName := "value" or varName="value"
	// Try with spaces first, then without
	assignOps := []string{" = ", " := ", "=", ":="}

	for _, op := range assignOps {
		pattern := varName + op
		if idx := strings.Index(line, pattern); idx >= 0 {
			// Get everything after the operator
			afterOp := strings.TrimSpace(line[idx+len(pattern):])

			// Extract until semicolon, comment, or end of line
			for _, endChar := range []string{";", "//"} {
				if endIdx := strings.Index(afterOp, endChar); endIdx >= 0 {
					afterOp = afterOp[:endIdx]
					break
				}
			}

			afterOp = strings.TrimSpace(afterOp)

			// Ignore empty string assignments (no-op)
			if afterOp == `""` || afterOp == "''" || afterOp == "" {
				return ""
			}

			return afterOp
		}
	}

	return ""
}

// bashHasDefault checks if a bash line uses default substitution syntax
// Returns (hasDefault, defaultValue)
func bashHasDefault(line string, envVar string) (bool, string) {
	upper := strings.ToUpper(envVar)
	lower := strings.ToLower(envVar)

	// Check for ${VAR:-default}, ${VAR-default}, ${VAR:=default}
	patterns := []struct{
		prefix string
		suffix string
	}{
		{"${" + upper + ":-", "}"},
		{"${" + upper + "-", "}"},
		{"${" + upper + ":=", "}"},
		{"${" + lower + ":-", "}"},
		{"${" + lower + "-", "}"},
		{"${" + lower + ":=", "}"},
	}

	for _, pattern := range patterns {
		if idx := strings.Index(line, pattern.prefix); idx >= 0 {
			start := idx + len(pattern.prefix)
			// Find the closing brace
			if endIdx := strings.Index(line[start:], pattern.suffix); endIdx >= 0 {
				defaultVal := strings.TrimSpace(line[start : start+endIdx])
				// Ignore empty defaults
				if defaultVal != "" && defaultVal != `""` && defaultVal != "''" {
					return true, defaultVal
				}
			}
		}
	}

	return false, ""
}

// jsHasDefault checks if JavaScript/TypeScript code uses default value patterns
// Returns (hasDefault, lineNumber, defaultValue)
func jsHasDefault(lines []string, startLine int, assignedVar string, envVar string) (bool, int, string) {
	line := lines[startLine]

	// Pattern 1: const port = process.env.PORT || 'default'
	if strings.Contains(line, "||") {
		parts := strings.Split(line, "||")
		if len(parts) == 2 {
			defaultVal := strings.TrimSpace(parts[1])
			// Remove trailing semicolon
			defaultVal = strings.TrimRight(defaultVal, ";")
			if defaultVal != "" && defaultVal != `""` && defaultVal != "''" {
				return true, startLine, defaultVal
			}
		}
	}

	// Pattern 2: const port = process.env.PORT ?? 'default' (nullish coalescing)
	if strings.Contains(line, "??") {
		parts := strings.Split(line, "??")
		if len(parts) == 2 {
			defaultVal := strings.TrimSpace(parts[1])
			defaultVal = strings.TrimRight(defaultVal, ";")
			if defaultVal != "" && defaultVal != `""` && defaultVal != "''" {
				return true, startLine, defaultVal
			}
		}
	}

	// Pattern 3: Ternary operator - process.env.PORT ? process.env.PORT : 'default'
	if strings.Contains(line, "?") && strings.Contains(line, ":") {
		// This is complex to parse correctly, so we'll use a simple heuristic
		// If line contains process.env and has ternary, likely has a default
		if strings.Contains(line, "process.env."+envVar) || strings.Contains(line, "process.env."+strings.ToLower(envVar)) {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				defaultVal := strings.TrimSpace(parts[len(parts)-1])
				defaultVal = strings.TrimRight(defaultVal, ";")
				if defaultVal != "" && defaultVal != `""` && defaultVal != "''" {
					return true, startLine, defaultVal
				}
			}
		}
	}

	// Pattern 4: if (!varName) { varName = 'default' }
	maxLook := startLine + 8
	if maxLook > len(lines) {
		maxLook = len(lines)
	}

	for i := startLine; i < maxLook; i++ {
		checkLine := strings.TrimSpace(lines[i])
		lower := strings.ToLower(checkLine)

		// Check for if (!varName) or if (varName === undefined) etc
		if strings.HasPrefix(lower, "if") {
			containsVar := strings.Contains(checkLine, assignedVar)
			hasNegation := strings.Contains(checkLine, "!"+assignedVar) ||
				strings.Contains(lower, strings.ToLower(assignedVar)+" === undefined") ||
				strings.Contains(lower, strings.ToLower(assignedVar)+" == null")

			if containsVar && hasNegation {
				// Look in next few lines for assignment
				for j := i + 1; j < maxLook && j < i+4; j++ {
					assignLine := strings.TrimSpace(lines[j])
					if strings.Contains(assignLine, assignedVar+" =") {
						if val := extractJSDefaultValue(assignLine, assignedVar); val != "" {
							return true, j, val
						}
					}
					// Stop at closing brace
					if strings.Contains(assignLine, "}") {
						break
					}
				}
			}
		}
	}

	return false, -1, ""
}

// extractJSDefaultValue extracts the default value from a JavaScript assignment
func extractJSDefaultValue(line, varName string) string {
	pattern := varName + " ="
	if idx := strings.Index(line, pattern); idx >= 0 {
		afterEq := strings.TrimSpace(line[idx+len(pattern):])
		// Remove trailing semicolon or comma
		afterEq = strings.TrimRight(afterEq, ";,")

		if afterEq != "" && afterEq != `""` && afterEq != "''" {
			return afterEq
		}
	}
	return ""
}
