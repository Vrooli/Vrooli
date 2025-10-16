package config

import (
	"path/filepath"
	"regexp"
	"strings"
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
)

// CheckHardcodedValues detects hardcoded configuration values
func CheckHardcodedValues(content []byte, filePath string) []Violation {
	var violations []Violation
	contentStr := string(content)

	// Skip test files and migrations
	if strings.HasSuffix(filePath, "_test.go") ||
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
			re:       regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*(?::=|=|:=)\s*"[^"]+"`),
			severity: "critical",
		},
		{
			name:     "hardcoded_password",
			re:       regexp.MustCompile(`(?i)(password|passwd|pwd|secret|token)\s*(?::=|=|:=)\s*"[^"]+"`),
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
			if pattern.name == "hardcoded_ip" && strings.Contains(line, "127.0.0.1") {
				continue
			}

			if _, ok := lineMatches[i]; !ok {
				lineMatches[i] = make(map[string]bool)
			}
			if lineMatches[i][pattern.name] {
				continue
			}
			lineMatches[i][pattern.name] = true

			title := strings.ReplaceAll(pattern.name, "_", " ")
			title = strings.Title(title)

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
	base := filepath.Base(filePath)
	if _, ok := hardcodedValuesLockfileBasenames[base]; ok {
		return true
	}

	if strings.Contains(filePath, "node_modules/") || strings.Contains(filePath, "vendor/") {
		return true
	}

	return false
}
