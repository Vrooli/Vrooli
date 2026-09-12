package signals_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// [REQ:SIG-P0-001] The production journal has no mutation statements.
func TestSignalJournalHasNoUpdateOrDeleteStatements(t *testing.T) {
	t.Log("[REQ:SIG-P0-001]")
	for _, pattern := range []string{"*.go", "../enrichment/*.go"} {
		files, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") {
				continue
			}
			body, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			upper := strings.ToUpper(string(body))
			if strings.Contains(upper, "UPDATE SIGNAL") || strings.Contains(upper, "DELETE FROM SIGNAL") {
				t.Fatalf("append-only journal violation in %s", file)
			}
		}
	}
}
