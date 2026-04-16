package env

import "testing"

func TestRuntimeExport(t *testing.T) {
	t.Parallel()

	runtime := Runtime{
		DataRoot:         "/tmp/gemini",
		CredentialsFile:  "/tmp/gemini/credentials.json",
		TokenLogFile:     "/tmp/gemini/logs/token_usage.log",
		APIBaseURL:       "https://generativelanguage.googleapis.com/v1beta",
		DefaultModel:     "gemini-pro",
		HealthCheckModel: "gemini-pro",
		CachePrefix:      "gemini:cache",
	}

	exported := runtime.Export()
	if exported["GEMINI_API_BASE"] != "https://generativelanguage.googleapis.com/v1beta" {
		t.Fatalf("Export() api base = %q", exported["GEMINI_API_BASE"])
	}
	if exported["GEMINI_CREDENTIALS_FILE"] != "/tmp/gemini/credentials.json" {
		t.Fatalf("Export() credentials file = %q", exported["GEMINI_CREDENTIALS_FILE"])
	}
}
