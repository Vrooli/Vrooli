package hardcodedvalues

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"structure-health/internal/packs/filekind"
)

/*
Rule: No Hardcoded Values
Description: Avoid hardcoded ports, URLs, and credentials
Reason: Improves security, flexibility, and deployment across different environments
Category: config
Severity: high
Standard: configuration-v1
Targets: api, cli, ui, test

<test-case id="hardcoded-credentials" should-fail="true">
  <description>Hardcoded passwords and API keys</description>
  <input language="go">
func connectDB() *sql.DB {
    password := "super_secret_password_123"
    apiKey := "sk-1234567890abcdef"

    connStr := fmt.Sprintf("postgres://user:%s@localhost/db", password)
    headers["API_KEY"] = apiKey

    return sql.Open("postgres", connStr)
}
  </input>
  <expected-violations>2</expected-violations>
  <expected-message>Hardcoded</expected-message>
</test-case>

<test-case id="config-key-names-not-secrets" should-fail="false">
  <description>Constants naming an env var or HTTP header are names, not secrets</description>
  <input language="go">
const (
    EnvQdrantAPIKey          = "QDRANT_API_KEY"
    headerAgentIdentityToken = "X-Agent-Identity-Token"
    headerContentType        = "Content-Type"
)
  </input>
</test-case>

<test-case id="hardcoded-ports-urls" should-fail="true">
  <description>Hardcoded ports and URLs</description>
  <input language="go">
func setupServer() {
    serverAddr := "localhost:8080"
    apiURL := "https://api.production.com/v1"
    dbHost := "192.168.1.100"

    http.ListenAndServe(":3000", nil)
}
  </input>
  <expected-violations>4</expected-violations>
  <expected-message>Hardcoded</expected-message>
</test-case>

<test-case id="environment-based-config" should-fail="true">
  <description>Proper configuration using environment variables</description>
  <input language="go">
func setupConfig() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080" // Default fallback is not allowed
    }

    dbPassword := os.Getenv("DB_PASSWORD")
    apiKey := os.Getenv("API_KEY")
    apiURL := os.Getenv("API_URL")

    if dbPassword == "" || apiKey == "" {
        log.Fatal("Required environment variables not set")
    }
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>Hardcoded port fallback</expected-message>
</test-case>


<test-case id="bash-port-fallback" should-fail="true">
  <description>Bash port fallback using parameter expansion</description>
  <input language="bash">
#!/usr/bin/env bash
: "${PORT:=8080}"  # Not allowed
PORT=${PORT:-"8080"}  # Also not allowed
  </input>
  <expected-violations>2</expected-violations>
  <expected-message>Hardcoded Port Fallback</expected-message>
</test-case>

<test-case id="bash-env-valid" should-fail="false">
  <description>Bash script reading port without fallback</description>
  <input language="bash">
#!/usr/bin/env bash
if [[ -z "${PORT}" ]]; then
  echo "PORT must be set"
  exit 1
fi
echo "Using port ${PORT}"
  </input>
</test-case>

<test-case id="js-port-fallback" should-fail="true">
  <description>JavaScript port fallback using logical OR</description>
  <input language="javascript">
const port = process.env.PORT || "8080";
const uiPort = process.env.UI_PORT ?? '35000';
  </input>
  <expected-violations>2</expected-violations>
  <expected-message>Hardcoded port fallback</expected-message>
</test-case>

<test-case id="js-port-config" should-fail="false">
  <description>JavaScript reading ports without literals</description>
  <input language="javascript">
const port = process.env.PORT;
if (!port) {
  throw new Error('PORT is required');
}
const uiPort = Number(process.env.UI_PORT);
  </input>
</test-case>

<test-case id="config-file-usage" should-fail="false">
  <description>Using configuration files instead of hardcoding</description>
  <input language="go">
type Config struct {
    Port     string `json:"port"`
    APIURL   string `json:"api_url"`
    DBHost   string `json:"db_host"`
}

func loadConfig() (*Config, error) {
    file, err := os.Open("config.json")
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var config Config
    decoder := json.NewDecoder(file)
    return &config, decoder.Decode(&config)
}
  </input>
</test-case>
*/

var (
	portFallbackPattern     = regexp.MustCompile(`(?i)([A-Za-z_][A-Za-z0-9_]*)\s*(?::=|=|:=)\s*"(\d{2,5})"`)
	bashPortFallbackPattern = regexp.MustCompile(`(?i)\$\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*[:?][-=]\s*"?(\d{2,5})"?\s*\}`)
	jsPortFallbackPattern   = regexp.MustCompile(`(?i)process\.env\.([A-Za-z_][A-Za-z0-9_]*)\s*(?:\|\||\?\?)\s*['"]?(\d{2,5})['"]?`)

	// envVarNameValue matches SCREAMING_SNAKE_CASE identifiers — env-var / config-key
	// NAMES such as "QDRANT_API_KEY" or "DB_PASSWORD". A literal secret value is
	// effectively never spelled this way.
	envVarNameValue = regexp.MustCompile(`^[A-Z][A-Z0-9]*(_[A-Z0-9]+)*$`)
	// httpHeaderNameValue matches canonical HTTP header NAMES such as
	// "X-Agent-Identity-Token" or "Content-Type": each dash-separated segment is a
	// capitalized word. A high-entropy secret like "sk-1234567890abcdef" has a
	// lowercase prefix and therefore does NOT match.
	httpHeaderNameValue = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*(-[A-Z][A-Za-z0-9]*)+$`)
)

// looksLikeConfigName reports whether v is plainly the NAME of a configuration key
// (an env-var name or an HTTP header name) rather than a literal secret VALUE.
// The credential detectors match on the left-hand identifier, so a constant such as
// `headerAgentIdentityToken = "X-Agent-Identity-Token"` would otherwise be flagged
// even though its value is a header name, not a credential.
func looksLikeConfigName(v string) bool {
	return envVarNameValue.MatchString(v) || httpHeaderNameValue.MatchString(v)
}

// looksLikeShellReference distinguishes a value sourced from the caller from
// a literal credential. Bash assignments such as API_TOKEN="$value" are not
// secrets, but the lexical detector cannot otherwise tell them apart.
func looksLikeShellReference(v string) bool {
	return strings.Contains(v, "$")
}

var ipv4Candidate = regexp.MustCompile(`\b\d{1,3}(?:\.\d{1,3}){3}\b`)

func containsValidIPv4(line string) bool {
	for _, candidate := range ipv4Candidate.FindAllString(line, -1) {
		valid := true
		for _, octet := range strings.Split(candidate, ".") {
			n, err := strconv.Atoi(octet)
			if err != nil || n > 255 {
				valid = false
				break
			}
		}
		if valid {
			return true
		}
	}
	return false
}

// isSVGPathData identifies inline SVG geometry. Path data is a compact sequence
// of coordinates, so values such as "4.25 17 2.94" can accidentally form a
// valid dotted quad even though they are not network configuration.
func isSVGPathData(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "<path") && (strings.Contains(lower, ` d="`) || strings.Contains(lower, ` d='`))
}

// CheckHardcodedValues detects hardcoded configuration values
func CheckHardcodedValues(content []byte, filePath string) []Violation {
	var violations []Violation
	contentStr := string(content)

	// Skip test files and migrations
	if filekind.IsTestSupportFile(filePath) ||
		strings.Contains(filePath, "migration") {
		return violations
	}

	if shouldSkipHardcodedValuesFile(filePath) {
		return violations
	}

	lines := strings.Split(contentStr, "\n")

	// Patterns to detect hardcoded values
	type patternDef struct {
		name     string
		re       *regexp.Regexp
		severity string
	}

	patterns := []patternDef{
		{
			name:     "hardcoded_api_key",
			re:       regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*(?::=|=|:=)\s*"([^"]+)"`),
			severity: "critical",
		},
		{
			name:     "hardcoded_password",
			re:       regexp.MustCompile(`(?i)(password|passwd|pwd|secret|token)\s*(?::=|=|:=)\s*"([^"]+)"`),
			severity: "critical",
		},
		{
			name:     "hardcoded_localhost",
			re:       regexp.MustCompile(`localhost:\d+`),
			severity: "medium",
		},
		{
			name:     "hardcoded_port",
			re:       regexp.MustCompile(`:\d{4,5}`),
			severity: "medium",
		},
		{
			name:     "hardcoded_ip",
			re:       regexp.MustCompile(`\b\d{1,3}(?:\.\d{1,3}){3}\b`),
			severity: "high",
		},
		{
			name:     "hardcoded_url",
			re:       regexp.MustCompile(`https?://[^\s"']+`),
			severity: "medium",
		},
	}

	lineMatches := make(map[int]map[string]bool)

	for i, line := range lines {
		// Skip comments and strings in logs
		if strings.TrimSpace(line) == "" ||
			strings.HasPrefix(strings.TrimSpace(line), "//") ||
			strings.Contains(line, "fmt.") ||
			strings.Contains(line, "log.") {
			continue
		}
		if isSVGPathData(line) {
			continue
		}

		matched := false
		if match := portFallbackPattern.FindStringSubmatch(line); len(match) > 0 {
			variable := strings.ToLower(match[1])
			if strings.Contains(variable, "port") {
				if _, ok := lineMatches[i]; !ok {
					lineMatches[i] = make(map[string]bool)
				}
				if !lineMatches[i]["hardcoded_port_fallback"] {
					lineMatches[i]["hardcoded_port_fallback"] = true
					violations = append(violations, Violation{
						Type:           "hardcoded_values",
						Severity:       "medium",
						Title:          "Hardcoded Port Fallback",
						Description:    "Hardcoded port fallback detected; rely on configuration or environment variables for defaults",
						FilePath:       filePath,
						LineNumber:     i + 1,
						CodeSnippet:    line,
						Recommendation: "Remove literal port fallbacks and source defaults from configuration or environment variables",
						Standard:       "configuration-v1",
					})
				}
				matched = true
			}
		}
		if !matched {
			if match := bashPortFallbackPattern.FindStringSubmatch(line); len(match) > 0 {
				variable := strings.ToLower(match[1])
				if strings.Contains(variable, "port") {
					if _, ok := lineMatches[i]; !ok {
						lineMatches[i] = make(map[string]bool)
					}
					if !lineMatches[i]["hardcoded_port_fallback"] {
						lineMatches[i]["hardcoded_port_fallback"] = true
						violations = append(violations, Violation{
							Type:           "hardcoded_values",
							Severity:       "medium",
							Title:          "Hardcoded Port Fallback",
							Description:    "Hardcoded port fallback detected; rely on configuration or environment variables for defaults",
							FilePath:       filePath,
							LineNumber:     i + 1,
							CodeSnippet:    line,
							Recommendation: "Remove literal port fallbacks and source defaults from configuration or environment variables",
							Standard:       "configuration-v1",
						})
					}
					matched = true
				}
			}
		}
		if !matched {
			if match := jsPortFallbackPattern.FindStringSubmatch(line); len(match) > 0 {
				variable := strings.ToLower(match[1])
				if strings.Contains(variable, "port") {
					if _, ok := lineMatches[i]; !ok {
						lineMatches[i] = make(map[string]bool)
					}
					if !lineMatches[i]["hardcoded_port_fallback"] {
						lineMatches[i]["hardcoded_port_fallback"] = true
						violations = append(violations, Violation{
							Type:           "hardcoded_values",
							Severity:       "medium",
							Title:          "Hardcoded Port Fallback",
							Description:    "Hardcoded port fallback detected; rely on configuration or environment variables for defaults",
							FilePath:       filePath,
							LineNumber:     i + 1,
							CodeSnippet:    line,
							Recommendation: "Remove literal port fallbacks and source defaults from configuration or environment variables",
							Standard:       "configuration-v1",
						})
					}
					matched = true
				}
			}
		}
		if matched {
			continue
		}
		for _, pattern := range patterns {
			if pattern.name == "hardcoded_port" && strings.Contains(line, "localhost:") {
				continue
			}
			if !pattern.re.MatchString(line) {
				continue
			}

			// Ignore benign examples
			if pattern.name == "hardcoded_url" && (strings.Contains(line, "example.com") || strings.Contains(line, "localhost")) {
				continue
			}
			if pattern.name == "hardcoded_ip" && (!containsValidIPv4(line) || strings.Contains(line, "127.0.0.1") || strings.Contains(strings.ToLower(line), "version")) {
				continue
			}
			if pattern.name == "hardcoded_url" && strings.Contains(line, `"$schema"`) {
				continue
			}

			// The credential detectors match on the left-hand identifier (e.g. a const
			// named "headerAgentIdentityToken" or "EnvQdrantAPIKey"). Skip when the
			// right-hand VALUE is plainly the NAME of a config key — an env-var name or
			// an HTTP header name — rather than a literal secret. This avoids flagging
			// the idiomatic `const X = "HEADER-NAME"` / `const X = "ENV_VAR_NAME"`.
			if pattern.name == "hardcoded_api_key" || pattern.name == "hardcoded_password" {
				if m := pattern.re.FindStringSubmatch(line); len(m) >= 3 && looksLikeConfigName(m[2]) {
					continue
				}
				if m := pattern.re.FindStringSubmatch(line); len(m) >= 3 && looksLikeShellReference(m[2]) {
					continue
				}
			}

			if _, ok := lineMatches[i]; !ok {
				lineMatches[i] = make(map[string]bool)
			}
			if lineMatches[i][pattern.name] {
				continue
			}
			lineMatches[i][pattern.name] = true

			titleParts := strings.Split(strings.ReplaceAll(pattern.name, "_", " "), " ")
			for partIndex, part := range titleParts {
				if part != "" {
					titleParts[partIndex] = strings.ToUpper(part[:1]) + part[1:]
				}
			}
			title := strings.Join(titleParts, " ")

			violations = append(violations, Violation{
				Type:           "hardcoded_values",
				Severity:       pattern.severity,
				Title:          title,
				Description:    "Hardcoded value detected that should be configurable",
				FilePath:       filePath,
				LineNumber:     i + 1,
				CodeSnippet:    line,
				Recommendation: "Move to environment variable or configuration file",
				Standard:       "configuration-v1",
			})

			matched = true
			break
		}
		if matched {
			continue
		}
	}

	return violations
}

var hardcodedValuesLockfileBasenames = map[string]struct{}{
	"package-lock.json": {},
	"pnpm-lock.yaml":    {},
	"yarn.lock":         {},
	"bun.lockb":         {},
	"Cargo.lock":        {},
	"go.sum":            {},
	"composer.lock":     {},
	"Gemfile.lock":      {},
	"Podfile.lock":      {},
	"poetry.lock":       {},
	"Pipfile.lock":      {},
}

func shouldSkipHardcodedValuesFile(filePath string) bool {
	// Wizard help is user-facing instructional content. External documentation
	// links and example endpoints in it are content, not deploy-time settings.
	if strings.Contains(strings.ToLower(filepath.ToSlash(filePath)), "/help-content/") {
		return true
	}

	base := filepath.Base(filePath)
	if _, ok := hardcodedValuesLockfileBasenames[base]; ok {
		return true
	}

	if strings.Contains(filePath, "node_modules/") || strings.Contains(filePath, "vendor/") {
		return true
	}

	return false
}
