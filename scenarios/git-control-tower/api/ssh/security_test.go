package ssh

import (
	"testing"
)

// FakePlatform implements Platform for testing.
type FakePlatform struct {
	SSHDirPath  string
	HomeDirPath string
	Err         error
}

func (p *FakePlatform) GetSSHDir() (string, error) {
	if p.Err != nil {
		return "", p.Err
	}
	return p.SSHDirPath, nil
}

func (p *FakePlatform) GetHomeDir() (string, error) {
	if p.Err != nil {
		return "", p.Err
	}
	return p.HomeDirPath, nil
}

func (p *FakePlatform) SSHKeygenPath() string {
	return "ssh-keygen"
}

func (p *FakePlatform) SSHPath() string {
	return "ssh"
}

func TestValidateSSHPath(t *testing.T) {
	platform := &FakePlatform{
		SSHDirPath:  "/home/user/.ssh",
		HomeDirPath: "/home/user",
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid path in ssh dir",
			path:    "/home/user/.ssh/id_ed25519",
			wantErr: false,
		},
		{
			name:    "valid path with tilde",
			path:    "~/.ssh/id_ed25519",
			wantErr: false,
		},
		{
			name:    "path outside ssh dir",
			path:    "/home/user/id_ed25519",
			wantErr: true,
		},
		{
			name:    "path traversal attempt",
			path:    "/home/user/.ssh/../.ssh/id_ed25519",
			wantErr: true,
		},
		{
			name:    "root path",
			path:    "/etc/passwd",
			wantErr: true,
		},
		{
			name:    "ssh dir itself",
			path:    "/home/user/.ssh",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSSHPath(platform, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSSHPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateKeyFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{
			name:     "valid filename",
			filename: "id_ed25519",
			wantErr:  false,
		},
		{
			name:     "valid filename with underscore start",
			filename: "_mykey",
			wantErr:  false,
		},
		{
			name:     "valid filename alphanumeric",
			filename: "key123",
			wantErr:  false,
		},
		{
			name:     "empty filename",
			filename: "",
			wantErr:  true,
		},
		{
			name:     "filename with path separator",
			filename: "sub/key",
			wantErr:  true,
		},
		{
			name:     "filename with backslash",
			filename: "sub\\key",
			wantErr:  true,
		},
		{
			name:     "filename with double dot",
			filename: "key..old",
			wantErr:  true,
		},
		{
			name:     "filename starting with dot",
			filename: ".hidden",
			wantErr:  true,
		},
		{
			name:     "filename starting with dash",
			filename: "-key",
			wantErr:  true,
		},
		{
			name:     "filename too long",
			filename: "a123456789012345678901234567890123456789012345678901234567890123456789",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKeyFilename(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateKeyFilename() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsProtectedFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{
			name:     "authorized_keys",
			filename: "authorized_keys",
			want:     true,
		},
		{
			name:     "known_hosts",
			filename: "known_hosts",
			want:     true,
		},
		{
			name:     "config",
			filename: "config",
			want:     true,
		},
		{
			name:     "environment",
			filename: "environment",
			want:     true,
		},
		{
			name:     "regular key file",
			filename: "id_ed25519",
			want:     false,
		},
		{
			name:     "custom key file",
			filename: "github_key",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsProtectedFile(tt.filename); got != tt.want {
				t.Errorf("IsProtectedFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	platform := &FakePlatform{
		HomeDirPath: "/home/user",
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "path with tilde",
			path: "~/.ssh/id_ed25519",
			want: "/home/user/.ssh/id_ed25519",
		},
		{
			name: "absolute path",
			path: "/home/user/.ssh/id_ed25519",
			want: "/home/user/.ssh/id_ed25519",
		},
		{
			name: "relative path",
			path: ".ssh/id_ed25519",
			want: ".ssh/id_ed25519",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExpandPath(platform, tt.path); got != tt.want {
				t.Errorf("ExpandPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
