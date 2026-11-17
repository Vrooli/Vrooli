package runtimehandlers

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/vrooli/vrooli/internal/rootmetrics"
)

func TestRootObservationsReadCommandReportsMissingStore(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SUDO_USER", "")
	var out, stderr bytes.Buffer
	handled, code := RunRootUpdateReadCommand([]string{"runtime", "update", "observations", "--json"}, &out, &stderr)
	if !handled || code != 0 {
		t.Fatalf("command: handled=%v code=%d stderr=%s", handled, code, stderr.String())
	}
	var summary rootmetrics.Summary
	if err := json.Unmarshal(out.Bytes(), &summary); err != nil || summary.StorePresent || summary.Valid != 0 {
		t.Fatalf("summary=%+v err=%v", summary, err)
	}
}
