package setup

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/runtime"
)

func requiredBlockedReport(reason hostreqkit.BlockingReason) runtime.Report {
	return runtime.Report{
		MissingRequired: []string{"buf"},
		Tools: []runtime.ItemStatus{{
			Name:           "buf",
			Required:       true,
			BlockingReason: reason,
		}},
	}
}

func TestSetupTerminalResultCategories(t *testing.T) {
	tests := []struct {
		name      string
		stage     string
		report    runtime.Report
		err       error
		category  string
		retryable bool
	}{
		{name: "ordinary success", stage: "finalize", category: SetupCategorySuccess},
		{name: "original mac requirement failure", stage: "requirements", report: requiredBlockedReport(hostreqkit.BlockingNeedsSudo), err: errors.New("buf requires privilege"), category: SetupCategoryRequiredRequirementBlocked, retryable: true},
		{name: "real unsupported host", stage: "validation", err: runtime.ErrUnsupportedPlatform, category: SetupCategoryUnsupportedPlatform},
		{name: "network or checksum failure", stage: "requirements", report: requiredBlockedReport(hostreqkit.BlockingNone), err: errors.New("checksum mismatch"), category: SetupCategoryRequiredRequirementBlocked, retryable: true},
		{name: "invalid configuration", stage: "resolution", err: errors.New("invalid selector"), category: SetupCategoryInvalidConfiguration},
		{name: "partial state", stage: "resources", err: errors.New("resource install failed"), category: SetupCategoryPartialState, retryable: true},
		{name: "transient bootstrap", stage: "bootstrap", err: errors.New("network unavailable"), category: SetupCategoryTransientFailure, retryable: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := setupTerminalResult(tt.stage, tt.report, tt.err)
			if result.Version != SetupResultVersion || result.Category != tt.category || result.Retryable != tt.retryable {
				t.Fatalf("result = %#v", result)
			}
			if tt.category == SetupCategoryRequiredRequirementBlocked && len(result.BlockedRequirements) != 1 {
				t.Fatalf("blocked requirements = %#v", result.BlockedRequirements)
			}
		})
	}
}

func TestWriteSetupResultProducesPrivateSeparateJSONTransport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "setup-result.json")
	want := SetupResult{Version: SetupResultVersion, Status: SetupStatusFailed, Category: SetupCategoryRequiredRequirementBlocked, Stage: "requirements", Retryable: true, BlockedRequirements: []string{"buf"}, Remediation: "retry"}
	if err := writeSetupResult(path, want); err != nil {
		t.Fatalf("writeSetupResult: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	var got SetupResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("result is not JSON: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("result = %#v, want %#v", got, want)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("result permissions = %v, %v; want 0600", info.Mode(), err)
	}
}
