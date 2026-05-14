package ai

import "testing"

func TestShellBasename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/bin/bash", "bash"},
		{"/usr/bin/zsh", "zsh"},
		{"bash", "bash"},
		{"", ""},
	}
	for _, tt := range tests {
		got := shellBasename(tt.input)
		if got != tt.want {
			t.Errorf("shellBasename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
