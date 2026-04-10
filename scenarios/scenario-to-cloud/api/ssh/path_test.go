package ssh

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateKeyFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		wantErr  string // empty means no error expected
	}{
		{name: "valid simple", filename: "id_ed25519", wantErr: ""},
		{name: "valid underscore start", filename: "_key", wantErr: ""},
		{name: "valid numeric start", filename: "1key", wantErr: ""},
		{name: "empty", filename: "", wantErr: "cannot be empty"},
		{name: "forward slash", filename: "foo/bar", wantErr: "path separators"},
		{name: "backslash", filename: `foo\bar`, wantErr: "path separators"},
		{name: "double dot", filename: "foo..bar", wantErr: "'..'"},
		{name: "too long", filename: strings.Repeat("a", 65), wantErr: "too long"},
		{name: "max length ok", filename: strings.Repeat("a", 64), wantErr: ""},
		{name: "starts with dash", filename: "-key", wantErr: "must start with alphanumeric"},
		{name: "starts with dot", filename: ".key", wantErr: "must start with alphanumeric"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKeyFilename(tt.filename)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("ValidateKeyFilename(%q) unexpected error: %v", tt.filename, err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateKeyFilename(%q) expected error containing %q, got nil", tt.filename, tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("ValidateKeyFilename(%q) error = %q, want containing %q", tt.filename, err.Error(), tt.wantErr)
				}
			}
		})
	}
}

func TestValidateSSHPath(t *testing.T) {
	t.Parallel()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("cannot get home dir: %v", err)
	}
	sshDir := filepath.Join(homeDir, ".ssh")

	tests := []struct {
		name    string
		path    string
		wantErr string // empty means no error expected
	}{
		{name: "valid path within .ssh", path: filepath.Join(sshDir, "id_ed25519"), wantErr: ""},
		{name: "tilde-prefixed path", path: "~/.ssh/id_rsa", wantErr: ""},
		{name: ".ssh directory itself", path: sshDir, wantErr: ""},
		{name: "path traversal outside .ssh", path: filepath.Join(sshDir, "..", "passwd"), wantErr: "within ~/.ssh"},
		{name: "dotdot within .ssh still rejected", path: sshDir + "/subdir/../id_rsa", wantErr: "traversal"},
		{name: "absolute path outside .ssh", path: "/tmp/key", wantErr: "within ~/.ssh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSSHPath(tt.path)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("ValidateSSHPath(%q) unexpected error: %v", tt.path, err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateSSHPath(%q) expected error containing %q, got nil", tt.path, tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("ValidateSSHPath(%q) error = %q, want containing %q", tt.path, err.Error(), tt.wantErr)
				}
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	t.Parallel()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("cannot get home dir: %v", err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "tilde path", path: "~/.ssh/id_ed25519", want: filepath.Join(homeDir, ".ssh/id_ed25519")},
		{name: "absolute path unchanged", path: "/root/.ssh/key", want: "/root/.ssh/key"},
		{name: "relative path unchanged", path: "keys/mykey", want: "keys/mykey"},
		{name: "just tilde slash", path: "~/", want: homeDir},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandPath(tt.path)
			if got != tt.want {
				t.Errorf("ExpandPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
