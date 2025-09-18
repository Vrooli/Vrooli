package config

import (
	"strings"
)

/*
Rule: Environment Variable Validation
Description: Ensures proper handling and validation of environment variables
Reason: Unvalidated environment variables can lead to security issues and runtime errors
Category: config
Severity: medium
Standard: OWASP

<test-case id="missing-env-validation" should-fail="true">
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
  <expected-message>Environment variable used without validation</expected-message>
</test-case>

<test-case id="proper-env-validation" should-fail="false">
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

<test-case id="sensitive-env-logging" should-fail="true">
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

<test-case id="env-with-defaults" should-fail="false">
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
*/

type EnvValidationRule struct{}

// Check analyzes code for proper environment variable handling
func (r *EnvValidationRule) Check(content string, filepath string) ([]Violation, error) {
	var violations []Violation
	
	// Skip if no environment variable usage
	if !strings.Contains(content, "os.Getenv") {
		return violations, nil
	}
	
	lines := strings.Split(content, "\n")
	
	for i, line := range lines {
		// Check for os.Getenv usage
		if strings.Contains(line, "os.Getenv(") {
			envVarLine := i
			varName := extractEnvVarName(line)
			
			// Check if the variable is validated (simple heuristic)
			hasValidation := false
			sensitiveVar := isSensitiveVar(varName)
			
			// Look ahead for validation (next 5 lines)
			for j := i + 1; j < len(lines) && j < i+6; j++ {
				nextLine := lines[j]
				// Check for empty string validation
				if strings.Contains(nextLine, `== ""`) || 
				   strings.Contains(nextLine, `!= ""`) ||
				   strings.Contains(nextLine, "log.Fatal") ||
				   strings.Contains(nextLine, "panic") {
					hasValidation = true
					break
				}
			}
			
			// Check if it's being logged (potential security issue)
			if sensitiveVar {
				for j := i; j < len(lines) && j < i+10; j++ {
					if strings.Contains(lines[j], "log.") || 
					   strings.Contains(lines[j], "fmt.Print") {
						if strings.Contains(lines[j], varName) || 
						   (j == i && strings.Contains(lines[j], "os.Getenv")) {
							violations = append(violations, Violation{
								RuleID:   "env_validation",
								Severity: "high",
								Message:  "Sensitive environment variable logged: " + varName,
								File:     filepath,
								Line:     j + 1,
							})
						}
					}
				}
			}
			
			// Check for missing validation
			if !hasValidation && !strings.Contains(line, `if`) {
				// Check if there's a default value pattern
				hasDefault := false
				for j := i; j < len(lines) && j < i+3; j++ {
					if strings.Contains(lines[j], `if`) && strings.Contains(lines[j], `== ""`) {
						hasDefault = true
						break
					}
				}
				
				if !hasDefault {
					violations = append(violations, Violation{
						RuleID:   "env_validation",
						Severity: "medium",
						Message:  "Environment variable used without validation: " + varName,
						File:     filepath,
						Line:     envVarLine + 1,
					})
				}
			}
		}
	}
	
	return violations, nil
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

// Violation represents a rule violation
type Violation struct {
	RuleID   string `json:"rule_id"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}