package assets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPersistedImageFormatsRejectUnknownVersions(t *testing.T) {
	for _, tc := range []struct {
		name           string
		read           func(string) error
		valid, unknown string
	}{
		{"manifest", func(path string) error { _, err := readManifest(path); return err }, `{"schema_version":1}`, `{"schema_version":2}`},
		{"receipt", func(path string) error { _, err := readReceipt(path); return err }, `{"version":1}`, `{"version":2}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "state.json")
			for _, sample := range []struct {
				data      string
				wantError bool
			}{{tc.valid, false}, {tc.unknown, true}} {
				if err := os.WriteFile(path, []byte(sample.data), 0600); err != nil {
					t.Fatal(err)
				}
				if err := tc.read(path); (err != nil) != sample.wantError {
					t.Fatalf("read %s: error = %v, want error %v", sample.data, err, sample.wantError)
				}
			}
		})
	}
}
