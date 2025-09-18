package rules

import (
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

<test-case id="environment-based-config" should-fail="false">
  <description>Proper configuration using environment variables</description>
  <input language="go">
func setupConfig() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080" // Default fallback is acceptable
    }
    
    dbPassword := os.Getenv("DB_PASSWORD")
    apiKey := os.Getenv("API_KEY")
    apiURL := os.Getenv("API_URL")
    
    if dbPassword == "" || apiKey == "" {
        log.Fatal("Required environment variables not set")
    }
}
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

// CheckHardcodedValues detects hardcoded configuration values
func CheckHardcodedValues(content []byte, filePath string) []Violation {
	var violations []Violation
	contentStr := string(content)
	
	// Skip test files and migrations
	if strings.HasSuffix(filePath, "_test.go") || 
	   strings.Contains(filePath, "migration") {
		return violations
	}
	
	lines := strings.Split(contentStr, "\n")
	
	// Patterns to detect hardcoded values
	patterns := map[string]*regexp.Regexp{
		"hardcoded_port":     regexp.MustCompile(`:(\d{4,5})["\s]`),
		"hardcoded_localhost": regexp.MustCompile(`localhost:(\d+)`),
		"hardcoded_ip":      regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`),
		"hardcoded_url":     regexp.MustCompile(`https?://[^\s"]+`),
		"hardcoded_password": regexp.MustCompile(`(?i)(password|passwd|pwd|secret|token|key)\s*[:=]\s*"[^"]+"`),
		"hardcoded_api_key": regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*[:=]\s*"[^"]+"`),
	}
	
	severityMap := map[string]string{
		"hardcoded_password": "critical",
		"hardcoded_api_key":  "critical",
		"hardcoded_port":     "medium",
		"hardcoded_localhost": "medium",
		"hardcoded_ip":       "high",
		"hardcoded_url":      "medium",
	}
	
	for i, line := range lines {
		// Skip comments and strings in logs
		if strings.TrimSpace(line) == "" || 
		   strings.HasPrefix(strings.TrimSpace(line), "//") ||
		   strings.Contains(line, "fmt.") ||
		   strings.Contains(line, "log.") {
			continue
		}
		
		for patternName, pattern := range patterns {
			if pattern.MatchString(line) {
				// Special cases to ignore
				if patternName == "hardcoded_ip" && strings.Contains(line, "127.0.0.1") {
					continue // Localhost IP is often acceptable in examples
				}
				if patternName == "hardcoded_url" && 
				   (strings.Contains(line, "example.com") || 
				    strings.Contains(line, "localhost")) {
					continue // Example URLs are acceptable
				}
				
				severity := severityMap[patternName]
				title := strings.ReplaceAll(patternName, "_", " ")
				title = strings.Title(title)
				
				violations = append(violations, Violation{
					Type:        "hardcoded_values",
					Severity:    severity,
					Title:       title,
					Description: "Hardcoded value detected that should be configurable",
					FilePath:    filePath,
					LineNumber:  i + 1,
					CodeSnippet: line,
					Recommendation: "Move to environment variable or configuration file",
					Standard:    "configuration-v1",
				})
			}
		}
	}
	
	return violations
}