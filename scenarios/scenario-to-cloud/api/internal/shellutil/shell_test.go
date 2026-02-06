package shellutil

import (
	"testing"
)

func TestQuoteSingle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty string", input: "", want: "''"},
		{name: "simple word", input: "hello", want: "'hello'"},
		{name: "embedded single quote", input: "it's", want: "'it'\"'\"'s'"},
		{name: "multiple single quotes", input: "a'b'c", want: "'a'\"'\"'b'\"'\"'c'"},
		{name: "special shell chars", input: "$(rm -rf /)", want: "'$(rm -rf /)'"},
		{name: "spaces", input: "hello world", want: "'hello world'"},
		{name: "newline", input: "line1\nline2", want: "'line1\nline2'"},
		{name: "double quotes preserved", input: `he said "hi"`, want: `'he said "hi"'`},
		{name: "backslash preserved", input: `path\to\file`, want: `'path\to\file'`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := QuoteSingle(tt.input)
			if got != tt.want {
				t.Errorf("QuoteSingle(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestVrooliCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		workdir string
		cmd     string
	}{
		{name: "simple command", workdir: "/opt/vrooli", cmd: "vrooli start"},
		{name: "workdir with spaces", workdir: "/opt/my vrooli", cmd: "ls"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VrooliCommand(tt.workdir, tt.cmd)
			// Must contain PATH setup
			if !contains(result, `export PATH=`) {
				t.Errorf("VrooliCommand missing PATH setup: %s", result)
			}
			// Must contain cd to workdir (quoted)
			if !contains(result, "cd "+QuoteSingle(tt.workdir)) {
				t.Errorf("VrooliCommand missing cd to workdir: %s", result)
			}
			// Must contain the command
			if !contains(result, tt.cmd) {
				t.Errorf("VrooliCommand missing command %q: %s", tt.cmd, result)
			}
		})
	}
}

func TestSafeRemoteJoin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		elem []string
		want string
	}{
		{name: "empty input", elem: []string{}, want: ""},
		{name: "single element", elem: []string{"/opt"}, want: "/opt"},
		{name: "two elements", elem: []string{"/opt", "vrooli"}, want: "/opt/vrooli"},
		{name: "trims whitespace", elem: []string{" /opt ", " vrooli "}, want: "/opt/vrooli"},
		{name: "skips empty", elem: []string{"/opt", "", "vrooli"}, want: "/opt/vrooli"},
		{name: "cleans path", elem: []string{"/opt/", "/vrooli/../app"}, want: "/opt/app"},
		{name: "all empty", elem: []string{"", "", ""}, want: ""},
		{name: "all whitespace", elem: []string{"  ", " "}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SafeRemoteJoin(tt.elem...)
			if got != tt.want {
				t.Errorf("SafeRemoteJoin(%v) = %q, want %q", tt.elem, got, tt.want)
			}
		})
	}
}

func TestValidateTildeExpansion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmd     string
		wantErr bool
	}{
		{name: "no tilde", cmd: "echo hello", wantErr: false},
		{name: "unquoted tilde ok", cmd: "cd ~/dir", wantErr: false},
		{name: "tilde in single quotes", cmd: "cd '~/dir'", wantErr: true},
		{name: "tilde in double quotes ok (not single)", cmd: `cd "~/dir"`, wantErr: false},
		{name: "empty command", cmd: "", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTildeExpansion(tt.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTildeExpansion(%q) error = %v, wantErr %v", tt.cmd, err, tt.wantErr)
			}
		})
	}
}

// contains is a test helper for substring check.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
