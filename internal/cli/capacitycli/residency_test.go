package capacitycli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseResidencyActionsRequireExactInputs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"begin", []string{"begin", "--request-file", "request.json"}, false},
		{"complete", []string{"complete", "--request-file", "request.json"}, false},
		{"reconcile", []string{"reconcile", "--request-file", "request.json"}, false},
		{"recover", []string{"recover", "--request-file", "request.json"}, false},
		{"history", []string{"history", "--transition-id", "t-1"}, false},
		{"get", []string{"get", "--transition-id", "t-1"}, false},
		{"release", []string{"release", "--request-file", "request.json"}, false},
		{"release missing identity file", []string{"release"}, true},
		{"begin missing file", []string{"begin"}, true},
		{"unknown", []string{"start", "--request-file", "x"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseResidencyRequest(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseResidencyRequest() error=%v wantErr=%t", err, tt.wantErr)
			}
		})
	}
}

func TestReadResidencyRequestRejectsMalformedPayload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "request.json")
	if err := os.WriteFile(path, []byte(`{"claim_id":`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadResidencyRequest(path); err == nil || !strings.Contains(err.Error(), "decode residency request") {
		t.Fatalf("ReadResidencyRequest() error=%v", err)
	}
}
