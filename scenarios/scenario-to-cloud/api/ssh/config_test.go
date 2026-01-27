package ssh

import (
	"testing"

	"scenario-to-cloud/domain"
)

func TestNewConfig_DefaultValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		host     string
		port     int
		user     string
		keyPath  string
		wantPort int
		wantUser string
	}{
		{
			name:     "all defaults applied when zeros/empty",
			host:     "192.168.1.1",
			port:     0,
			user:     "",
			keyPath:  "/path/to/key",
			wantPort: DefaultPort,
			wantUser: DefaultUser,
		},
		{
			name:     "custom port preserved",
			host:     "example.com",
			port:     2222,
			user:     "",
			keyPath:  "/key",
			wantPort: 2222,
			wantUser: DefaultUser,
		},
		{
			name:     "custom user preserved",
			host:     "example.com",
			port:     0,
			user:     "deployer",
			keyPath:  "/key",
			wantPort: DefaultPort,
			wantUser: "deployer",
		},
		{
			name:     "keyPath whitespace trimmed",
			host:     "example.com",
			port:     22,
			user:     "root",
			keyPath:  "  /path/with/spaces  ",
			wantPort: 22,
			wantUser: "root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewConfig(tt.host, tt.port, tt.user, tt.keyPath)

			if cfg.Host != tt.host {
				t.Errorf("Host = %q, want %q", cfg.Host, tt.host)
			}
			if cfg.Port != tt.wantPort {
				t.Errorf("Port = %d, want %d", cfg.Port, tt.wantPort)
			}
			if cfg.User != tt.wantUser {
				t.Errorf("User = %q, want %q", cfg.User, tt.wantUser)
			}
		})
	}
}

func TestConfigFromManifest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		manifest domain.CloudManifest
		want     Config
	}{
		{
			name: "extracts VPS config from manifest",
			manifest: domain.CloudManifest{
				Target: domain.ManifestTarget{
					VPS: &domain.ManifestVPS{
						Host:    "prod.example.com",
						Port:    22,
						User:    "deploy",
						KeyPath: "/home/user/.ssh/id_rsa",
					},
				},
			},
			want: Config{
				Host:    "prod.example.com",
				Port:    22,
				User:    "deploy",
				KeyPath: "/home/user/.ssh/id_rsa",
			},
		},
		{
			name: "trims whitespace from keyPath",
			manifest: domain.CloudManifest{
				Target: domain.ManifestTarget{
					VPS: &domain.ManifestVPS{
						Host:    "vps.example.com",
						Port:    2222,
						User:    "admin",
						KeyPath: "  /keys/ssh_key  ",
					},
				},
			},
			want: Config{
				Host:    "vps.example.com",
				Port:    2222,
				User:    "admin",
				KeyPath: "/keys/ssh_key",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConfigFromManifest(tt.manifest)

			if got.Host != tt.want.Host {
				t.Errorf("Host = %q, want %q", got.Host, tt.want.Host)
			}
			if got.Port != tt.want.Port {
				t.Errorf("Port = %d, want %d", got.Port, tt.want.Port)
			}
			if got.User != tt.want.User {
				t.Errorf("User = %q, want %q", got.User, tt.want.User)
			}
			if got.KeyPath != tt.want.KeyPath {
				t.Errorf("KeyPath = %q, want %q", got.KeyPath, tt.want.KeyPath)
			}
		})
	}
}

func TestDefaultConstants(t *testing.T) {
	t.Parallel()

	if DefaultPort != 22 {
		t.Errorf("DefaultPort = %d, want 22", DefaultPort)
	}
	if DefaultUser != "root" {
		t.Errorf("DefaultUser = %q, want 'root'", DefaultUser)
	}
}
