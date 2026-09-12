package setup

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
)

// [REQ:BOOT-RECOVERY-001] The JSON status report carries every safeguard's
// applied state and notes verbatim so a consumer can read one precondition
// without parsing prose.
func TestWriteSetupStatusJSONCarriesSafeguardVerdicts(t *testing.T) {
	var out bytes.Buffer
	report := vrooliruntime.Report{
		Environment: "development",
		Safeguards: []hostreqkit.SafeguardStatus{
			{Name: "autoheal_watchdog", Applied: true, ExecutionState: hostreqkit.ExecutionAlreadyPresent, Notes: []string{"validator: accepted"}},
			{Name: "runtime_supervisor", ExecutionState: hostreqkit.ExecutionPending, Notes: []string{"unit content differs from the rendered definition"}},
		},
		MissingRequired: []string{"runtime_supervisor"},
	}
	if err := writeSetupStatusJSON(&out, "readiness", report, SetupReadiness{Status: "missing", Blockers: []string{"runtime_supervisor"}, Source: ReadinessSourceInProcess}); err != nil {
		t.Fatal(err)
	}
	var decoded SetupStatusReport
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("decode: %v\n%s", err, out.String())
	}
	if decoded.Version != SetupStatusReportVersion || decoded.Phase != "readiness" || decoded.Environment != "development" {
		t.Fatalf("header = %+v", decoded)
	}
	if len(decoded.Safeguards) != 2 || !decoded.Safeguards[0].Applied || decoded.Safeguards[1].Applied {
		t.Fatalf("safeguards = %+v", decoded.Safeguards)
	}
	if decoded.Safeguards[1].Notes[0] != "unit content differs from the rendered definition" {
		t.Fatalf("notes were not carried verbatim: %+v", decoded.Safeguards[1].Notes)
	}
	if len(decoded.Missing.Required) != 1 || decoded.Readiness.Status != "missing" {
		t.Fatalf("missing/readiness = %+v / %+v", decoded.Missing, decoded.Readiness)
	}
	if bytes.Contains(out.Bytes(), []byte("[INFO]")) {
		t.Fatal("JSON output must not carry human-readable preamble lines")
	}
}
