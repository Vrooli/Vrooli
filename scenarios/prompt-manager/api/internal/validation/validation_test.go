package validation

import "testing"

func TestIsValidHexColor(t *testing.T) {
	tests := []struct {
		name  string
		color string
		want  bool
	}{
		{"valid lowercase", "#ff00ff", true},
		{"valid uppercase", "#FF00FF", true},
		{"valid mixed case", "#Ff00Ff", true},
		{"valid black", "#000000", true},
		{"valid white", "#ffffff", true},
		{"missing hash", "FF00FF", false},
		{"too short", "#FFF", false},
		{"too long", "#FF00FF00", false},
		{"invalid chars", "#GGGGGG", false},
		{"empty string", "", false},
		{"only hash", "#", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidHexColor(tt.color)
			if got != tt.want {
				t.Errorf("IsValidHexColor(%q) = %v, want %v", tt.color, got, tt.want)
			}
		})
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple lowercase", "hello", "hello"},
		{"with spaces", "hello world", "hello-world"},
		{"uppercase to lowercase", "Hello World", "hello-world"},
		{"special chars removed", "Hello! World?", "hello-world"},
		{"numbers preserved", "test123", "test123"},
		{"hyphens preserved", "hello-world", "hello-world"},
		{"multiple spaces", "hello  world", "hello--world"},
		{"underscores removed", "hello_world", "helloworld"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Slugify(tt.input)
			if got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
