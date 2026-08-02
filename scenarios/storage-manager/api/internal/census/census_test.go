package census

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanClosedIdentityAndAttribution(t *testing.T) {
	root := t.TempDir()
	mustWrite := func(name, value string) {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(value), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("owned/a", "1234")
	mustWrite("unowned/b", "123456")
	report, err := Scan(root, map[string][]Declaration{"component": {{Name: "data", Path: filepath.Join(root, "owned"), Budgeted: true}}})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Closed || report.MeasuredBytes != 10 || report.AttributedBytes != 4 || report.UnattributedBytes != 6 {
		t.Fatalf("unexpected report: %+v", report)
	}
}
