package ssh

import (
	"strings"
	"testing"
)

func TestLocalSSHCommand(t *testing.T) {
	t.Parallel()

	cfg := Config{Host: "example.com", Port: 22, User: "root", KeyPath: "/home/user/.ssh/id_ed25519"}
	result := LocalSSHCommand(cfg, "uptime")

	// Must start with "ssh"
	if !strings.HasPrefix(result, "ssh ") {
		t.Errorf("LocalSSHCommand should start with 'ssh ': %s", result)
	}
	// Must contain target
	if !strings.Contains(result, "root@example.com") {
		t.Errorf("LocalSSHCommand missing target: %s", result)
	}
	// Must contain command separator
	if !strings.Contains(result, "-- bash -lc") {
		t.Errorf("LocalSSHCommand missing -- bash -lc: %s", result)
	}
	// Must contain the command (quoted)
	if !strings.Contains(result, "'uptime'") {
		t.Errorf("LocalSSHCommand missing quoted command: %s", result)
	}
	// Must contain key path
	if !strings.Contains(result, "-i /home/user/.ssh/id_ed25519") {
		t.Errorf("LocalSSHCommand missing key path: %s", result)
	}
}

func TestLocalSCPCommand(t *testing.T) {
	t.Parallel()

	cfg := Config{Host: "example.com", Port: 2222, User: "deploy", KeyPath: "/key"}
	result := LocalSCPCommand(cfg, "/tmp/bundle.tar.gz", "/opt/deploy/bundle.tar.gz")

	// Must start with "scp"
	if !strings.HasPrefix(result, "scp ") {
		t.Errorf("LocalSCPCommand should start with 'scp ': %s", result)
	}
	// Must use uppercase -P for port
	if !strings.Contains(result, "-P 2222") {
		t.Errorf("LocalSCPCommand should use -P for port: %s", result)
	}
	// Must contain local path
	if !strings.Contains(result, "/tmp/bundle.tar.gz") {
		t.Errorf("LocalSCPCommand missing local path: %s", result)
	}
	// Must contain remote target
	if !strings.Contains(result, "deploy@example.com:/opt/deploy/bundle.tar.gz") {
		t.Errorf("LocalSCPCommand missing remote target: %s", result)
	}
}

func TestFormatCommandForLog(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		cfg          Config
		cmd          string
		wantContains []string
		wontContains []string
	}{
		{
			name:         "redacts key path",
			cfg:          Config{Host: "example.com", Port: 22, User: "root", KeyPath: "/home/user/.ssh/id_ed25519"},
			cmd:          "uptime",
			wantContains: []string{"ssh", "root@example.com", "<redacted>", "uptime"},
			wontContains: []string{"/home/user/.ssh/id_ed25519"},
		},
		{
			name:         "no key path no redaction",
			cfg:          Config{Host: "example.com", Port: 22, User: "root"},
			cmd:          "ls",
			wantContains: []string{"ssh", "root@example.com", "ls"},
			wontContains: []string{"<redacted>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatCommandForLog(tt.cfg, tt.cmd)
			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("FormatCommandForLog missing %q: %s", want, result)
				}
			}
			for _, wont := range tt.wontContains {
				if strings.Contains(result, wont) {
					t.Errorf("FormatCommandForLog should not contain %q: %s", wont, result)
				}
			}
		})
	}
}
