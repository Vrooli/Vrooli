package env

import "testing"

func TestRuntimeExport(t *testing.T) {
	t.Parallel()

	runtime := Runtime{
		DataRoot:   "/tmp/data",
		ConfigsDir: "/tmp/data/configs",
		LogsDir:    "/tmp/data/logs",
		ConfigFile: "/tmp/data/config.json",
		StateFile:  "/tmp/data/state.json",
		APIBaseURL: "https://api.cloudflare.com/client/v4/accounts",
	}

	exported := runtime.Export()
	if exported["CLOUDFLARE_AI_GATEWAY_CONFIG_FILE"] != "/tmp/data/config.json" {
		t.Fatalf("Export() config file = %q", exported["CLOUDFLARE_AI_GATEWAY_CONFIG_FILE"])
	}
	if exported["CLOUDFLARE_AI_GATEWAY_API_BASE_URL"] != "https://api.cloudflare.com/client/v4/accounts" {
		t.Fatalf("Export() api base = %q", exported["CLOUDFLARE_AI_GATEWAY_API_BASE_URL"])
	}
}
