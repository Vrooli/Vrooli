package hardcodedvalues

import "testing"

// countByTitle returns how many violations carry the given Title.
func countByTitle(vs []Violation, title string) int {
	n := 0
	for _, v := range vs {
		if v.Title == title {
			n++
		}
	}
	return n
}

// TestCredentialNameVsValue locks in the fix for the structure-health false
// positives where the credential detectors flagged a constant whose IDENTIFIER
// contained "Token"/"APIKey"/"Password" even though its VALUE was the NAME of an
// env var or HTTP header rather than a secret. The guard must suppress those while
// still catching genuine hardcoded secrets.
func TestCredentialNameVsValue(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		wantAPIKey int
		wantPass   int
	}{
		// --- must NOT flag: value is a NAME, not a secret ---
		{
			name: "env var name as value",
			line: `	EnvQdrantAPIKey = "QDRANT_API_KEY"`,
		},
		{
			name: "http header name with Token suffix",
			line: `	headerAgentIdentityToken = "X-Agent-Identity-Token"`,
		},
		{
			name: "canonical header name",
			line: `	headerContentType = "Content-Type"`,
		},
		{
			name: "screaming snake secret-keyword name",
			line: `	envSecret = "APP_SECRET"`,
		},
		// --- must STILL flag: value is a real secret ---
		{
			name:     "literal password value",
			line:     `	password := "super_secret_password_123"`,
			wantPass: 1,
		},
		{
			name:       "openai-style api key with lowercase prefix",
			line:       `	apiKey := "sk-1234567890abcdef"`,
			wantAPIKey: 1,
		},
		{
			name:     "mixed-case token value",
			line:     `	token := "aB3xY9zQ-1234"`,
			wantPass: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CheckHardcodedValues([]byte(tc.line+"\n"), "api/internal/example/config.go")
			if n := countByTitle(got, "Hardcoded Api Key"); n != tc.wantAPIKey {
				t.Errorf("Hardcoded Api Key count = %d, want %d (violations: %+v)", n, tc.wantAPIKey, got)
			}
			if n := countByTitle(got, "Hardcoded Password"); n != tc.wantPass {
				t.Errorf("Hardcoded Password count = %d, want %d (violations: %+v)", n, tc.wantPass, got)
			}
		})
	}
}

// TestLooksLikeConfigName exercises the name classifier directly so the boundary
// between a NAME and a high-entropy secret is explicit and regression-protected.
func TestLooksLikeConfigName(t *testing.T) {
	names := []string{
		"QDRANT_API_KEY",         // env var name
		"DB_PASSWORD",            // env var name
		"X-Agent-Identity-Token", // http header name
		"Content-Type",           // http header name
		"AUTH_TOKEN",             // env var name
	}
	for _, v := range names {
		if !looksLikeConfigName(v) {
			t.Errorf("looksLikeConfigName(%q) = false, want true (it is a config-key name)", v)
		}
	}

	secrets := []string{
		"super_secret_password_123", // lowercase literal
		"sk-1234567890abcdef",       // lowercase-prefixed api key
		"aB3xY9zQ-1234",             // mixed-case high entropy
		"hunter2",                   // lowercase literal
		"p@ssw0rd",                  // symbol-bearing literal
	}
	for _, v := range secrets {
		if looksLikeConfigName(v) {
			t.Errorf("looksLikeConfigName(%q) = true, want false (it is a secret value)", v)
		}
	}
}

// TestExistingCredentialCasesStillFire guards the documented should-fail case
// (two violations: one password, one api key) from the rule's own doc spec.
func TestExistingCredentialCasesStillFire(t *testing.T) {
	src := `func connectDB() *sql.DB {
    password := "super_secret_password_123"
    apiKey := "sk-1234567890abcdef"
    return nil
}`
	got := CheckHardcodedValues([]byte(src), "api/internal/example/db.go")
	if n := countByTitle(got, "Hardcoded Password"); n != 1 {
		t.Errorf("Hardcoded Password count = %d, want 1 (violations: %+v)", n, got)
	}
	if n := countByTitle(got, "Hardcoded Api Key"); n != 1 {
		t.Errorf("Hardcoded Api Key count = %d, want 1 (violations: %+v)", n, got)
	}
}

func TestHardcodedIPRejectsVersionsAndTestFiles(t *testing.T) {
	if got := CheckHardcodedValues([]byte(`const ua = "Chrome/120.0.0.0"`), "ui/shared.test.ts"); countByTitle(got, "Hardcoded Ip") != 0 {
		t.Fatalf("test browser version produced violations: %+v", got)
	}
	if got := CheckHardcodedValues([]byte(`const ip = "10.0.0.1"`), "api/config.go"); countByTitle(got, "Hardcoded Ip") != 1 {
		t.Fatalf("valid dotted quad must remain detectable: %+v", got)
	}
	if got := CheckHardcodedValues([]byte(`const version = "1.2.3.4"`), "api/version.go"); countByTitle(got, "Hardcoded Ip") != 0 {
		t.Fatalf("semantic version produced violations: %+v", got)
	}
}

func TestHardcodedURLAllowsJSONSchemaIdentifier(t *testing.T) {
	if got := CheckHardcodedValues([]byte(`"$schema": "https://json-schema.org/draft/2020-12/schema"`), "ui/schema.json"); countByTitle(got, "Hardcoded Url") != 0 {
		t.Fatalf("schema identifier produced violations: %+v", got)
	}
}
