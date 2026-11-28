package config

import "testing"

func TestSanitizeAppName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"normal name", "My App", "my-app"},
		{"lowercase", "myapp", "myapp"},
		{"spaces replaced", "My Cool App", "my-cool-app"},
		{"already lowercase with dashes", "my-app", "my-app"},
		{"empty string", "", "desktop-app"},
		{"whitespace only", "   ", "desktop-app"},
		{"leading/trailing whitespace", "  My App  ", "my-app"},
		{"uppercase", "MYAPP", "myapp"},
		{"mixed case", "MyApp", "myapp"},
		{"dot", "My App 2.0", "my-app-2-0"},
		{"punctuation", "Acme Ltd.", "acme-ltd"},
		{"unicode", "Café Tools", "café-tools"},
		{"underscore", "app_v1", "app-v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeAppName(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeAppName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
