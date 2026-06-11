package signals

import "testing"

func TestServiceCollectorCategory(t *testing.T) {
	tests := []struct {
		name               string
		manifest           string // empty = no file
		wantCategory       string
		wantSystemRequired bool
		wantErr            bool
	}{
		{name: "declared category", manifest: `{"category":"ai_tools"}`, wantCategory: "ai_tools"},
		{name: "system required", manifest: `{"category":"platform","system_required":true}`, wantCategory: "platform", wantSystemRequired: true},
		{name: "undeclared category defaults", manifest: `{"version":"1.0.0"}`, wantCategory: "utility"},
		{name: "missing manifest defaults", manifest: "", wantCategory: "utility"},
		{name: "malformed manifest errors", manifest: `{`, wantCategory: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if tt.manifest != "" {
				writeFile(t, root, ".vrooli/service.json", tt.manifest)
			}

			snap := Snapshot{Root: root}
			err := serviceCollector{}.Collect(&snap)
			if tt.wantErr {
				if err == nil {
					t.Fatal("want error for malformed manifest")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if snap.Category != tt.wantCategory {
				t.Fatalf("category = %q, want %q", snap.Category, tt.wantCategory)
			}
			if snap.SystemRequired != tt.wantSystemRequired {
				t.Fatalf("system_required = %v, want %v", snap.SystemRequired, tt.wantSystemRequired)
			}
		})
	}
}
