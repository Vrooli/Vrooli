package rules

import (
	"regexp"
	"strings"
)

/*
Rule: Environment Variable Naming
Description: Environment variables should follow VROOLI_ prefix convention
Reason: Prevents naming conflicts and clearly identifies application-specific configuration
Category: config
Severity: low
Standard: configuration-v1

<test-case id="non-standard-env-vars" should-fail="true">
  <description>Custom environment variables without VROOLI_ prefix</description>
  <input language="go">
func loadConfig() {
    apiEndpoint := os.Getenv("API_ENDPOINT")
    secretKey := os.Getenv("SECRET_KEY")
    customSetting := os.Getenv("MY_CUSTOM_SETTING")

    if apiEndpoint == "" {
        log.Fatal("API_ENDPOINT not set")
    }
}
  </input>
  <expected-violations>3</expected-violations>
  <expected-message>Non-Standard Environment Variable</expected-message>
</test-case>

<test-case id="proper-vrooli-prefix" should-fail="false">
  <description>Environment variables with proper VROOLI_ prefix</description>
  <input language="go">
func loadVrooliConfig() {
    apiEndpoint := os.Getenv("VROOLI_API_ENDPOINT")
    secretKey := os.Getenv("VROOLI_SECRET_KEY")
    lifecycleManaged := os.Getenv("VROOLI_LIFECYCLE_MANAGED")

    // Standard vars don't need prefix
    port := os.Getenv("PORT")
    dbURL := os.Getenv("DATABASE_URL")
}
  </input>
</test-case>

<test-case id="standard-vars-allowed" should-fail="false">
  <description>Standard environment variables are allowed without prefix</description>
  <input language="go">
func setupEnvironment() {
    // All standard vars that don't need VROOLI_ prefix
    home := os.Getenv("HOME")
    port := os.Getenv("PORT")
    dbHost := os.Getenv("DB_HOST")
    dbPassword := os.Getenv("DB_PASSWORD")
    postgresUser := os.Getenv("POSTGRES_USER")
    redisURL := os.Getenv("REDIS_URL")
    awsRegion := os.Getenv("AWS_REGION")
}
  </input>
</test-case>

<test-case id="mixed-env-vars" should-fail="true">
  <description>Mix of proper and improper environment variable names</description>
  <input language="go">
func configure() {
    // Proper VROOLI prefix
    vrooliRoot := os.Getenv("VROOLI_ROOT")

    // Standard allowed vars
    dbURL := os.Getenv("DATABASE_URL")

    // Custom vars without prefix (violations)
    customTimeout := os.Getenv("CUSTOM_TIMEOUT")
    serviceURL := os.Getenv("SERVICE_URL")
}
  </input>
  <expected-violations>2</expected-violations>
  <expected-message>Non-Standard Environment Variable</expected-message>
</test-case>
*/

// CheckEnvironmentVariableNaming validates environment variable naming conventions
func CheckEnvironmentVariableNaming(content []byte, filePath string) []Violation {
	var violations []Violation
	contentStr := string(content)

	// Pattern to find environment variable access
	envPatterns := []*regexp.Regexp{
		regexp.MustCompile(`os\.Getenv\("([^"]+)"\)`),
		regexp.MustCompile(`os\.LookupEnv\("([^"]+)"\)`),
		regexp.MustCompile(`viper\.GetString\("([^"]+)"\)`),
		regexp.MustCompile(`\$\{([A-Z_]+)\}`), // Shell variable references
	}

	// Standard exceptions that don't need VROOLI_ prefix
	standardVars := map[string]bool{
		"HOME":      true,
		"PATH":      true,
		"USER":      true,
		"SHELL":     true,
		"TERM":      true,
		"LANG":      true,
		"LC_ALL":    true,
		"TZ":        true,
		"PORT":      true, // Common in cloud environments
		"NODE_ENV":  true,
		"GO_ENV":    true,
		"DEBUG":     true,
		"LOG_LEVEL": true,
		// Database standard vars
		"DATABASE_URL": true,
		"DB_HOST":      true,
		"DB_PORT":      true,
		"DB_USER":      true,
		"DB_PASSWORD":  true,
		"DB_NAME":      true,
		// Common service vars
		"REDIS_URL":   true,
		"MONGODB_URI": true,
		"AWS_REGION":  true,
	}

	lines := strings.Split(contentStr, "\n")

	for i, line := range lines {
		for _, pattern := range envPatterns {
			matches := pattern.FindAllStringSubmatch(line, -1)
			for _, match := range matches {
				if len(match) > 1 {
					envVar := match[1]

					// Check if it's a custom var that should have VROOLI_ prefix
					if !standardVars[envVar] &&
						!strings.HasPrefix(envVar, "VROOLI_") &&
						!strings.HasPrefix(envVar, "POSTGRES_") && // Allow standard postgres vars
						!strings.HasPrefix(envVar, "REDIS_") && // Allow standard redis vars
						!strings.HasPrefix(envVar, "AWS_") { // Allow AWS vars

						violations = append(violations, Violation{
							Type:           "env_var_naming",
							Severity:       "low",
							Title:          "Non-Standard Environment Variable",
							Description:    "Non-Standard Environment Variable: " + envVar,
							FilePath:       filePath,
							LineNumber:     i + 1,
							CodeSnippet:    line,
							Recommendation: "Rename to VROOLI_" + envVar + " for consistency",
							Standard:       "configuration-v1",
						})
					}
				}
			}
		}
	}

	return violations
}
