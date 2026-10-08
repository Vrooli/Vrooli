package components

import "testing"

func TestParseLibrarySpecifier(t *testing.T) {
	tests := []struct {
		name, specifier, wantName, wantVersion, wantDiagnostic string
	}{
		{name: "latest", specifier: "@vrooli/react-component-library/Button", wantName: "Button"},
		{name: "pinned", specifier: "@vrooli/react-component-library/useLocale/1.0.1", wantName: "useLocale", wantVersion: "1.0.1"},
		{name: "relative", specifier: "../Button", wantDiagnostic: "relative import"},
		{name: "foreign", specifier: "react", wantDiagnostic: "not a React Component Library"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, version, diagnostic := parseLibrarySpecifier(tt.specifier)
			if name != tt.wantName || version != tt.wantVersion {
				t.Fatalf("parseLibrarySpecifier(%q) = %q, %q; want %q, %q", tt.specifier, name, version, tt.wantName, tt.wantVersion)
			}
			if tt.wantDiagnostic != "" && !containsDiagnostic(diagnostic, tt.wantDiagnostic) {
				t.Fatalf("diagnostic %q does not contain %q", diagnostic, tt.wantDiagnostic)
			}
		})
	}
}

func containsDiagnostic(value, fragment string) bool {
	for i := 0; i+len(fragment) <= len(value); i++ {
		if value[i:i+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
