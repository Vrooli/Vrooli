package phases

import (
	"strings"
	"testing"
)

func TestExtractCapturedJSONObjectToleratesCommandPreamble(t *testing.T) {
	raw := []byte("rebuilding delegated CLI\n{\"scenario\":\"demo\",\"passed\":true}\n")

	payload, err := extractCapturedJSONObject("delegated", raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(payload); got != `{"scenario":"demo","passed":true}` {
		t.Fatalf("payload = %q, want JSON object", got)
	}
}

func TestExtractCapturedJSONObjectReportsMissingJSON(t *testing.T) {
	_, err := extractCapturedJSONObject("delegated", []byte("rebuilding delegated CLI\n"))
	if err == nil || !strings.Contains(err.Error(), "did not contain a JSON object") {
		t.Fatalf("err = %v, want missing JSON object error", err)
	}
}
