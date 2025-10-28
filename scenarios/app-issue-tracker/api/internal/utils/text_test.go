package utils

import "testing"

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain text",
			input: "Hello World",
			want:  "Hello World",
		},
		{
			name:  "text with colors",
			input: "\x1b[31mRed Text\x1b[0m",
			want:  "Red Text",
		},
		{
			name:  "text with multiple ANSI sequences",
			input: "\x1b[1m\x1b[32mBold Green\x1b[0m Normal",
			want:  "Bold Green Normal",
		},
		{
			name:  "complex ANSI with cursor positioning",
			input: "\x1b[H\x1b[2JCleared\x1b[0m",
			want:  "Cleared",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only ANSI codes",
			input: "\x1b[0m\x1b[31m\x1b[0m",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripANSI(tt.input)
			if got != tt.want {
				t.Errorf("StripANSI() = %q, want %q", got, tt.want)
			}
		})
	}
}
