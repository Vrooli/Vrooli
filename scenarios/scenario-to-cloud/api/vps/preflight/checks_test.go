package preflight

import "testing"

func TestUfwAllowsPort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
		port int
		want bool
	}{
		{
			name: "exact port match",
			line: "80/tcp ALLOW Anywhere",
			port: 80,
			want: true,
		},
		{
			name: "port in range",
			line: "443/tcp ALLOW Anywhere",
			port: 443,
			want: true,
		},
		{
			name: "port not present",
			line: "22/tcp ALLOW Anywhere",
			port: 80,
			want: false,
		},
		{
			name: "partial match not allowed (8080 should not match 80)",
			line: "8080/tcp ALLOW Anywhere",
			port: 80,
			want: false,
		},
		{
			name: "partial match not allowed (80 should not match 8080)",
			line: "80/tcp ALLOW Anywhere",
			port: 8080,
			want: false,
		},
		{
			name: "port at beginning of line",
			line: "443 ALLOW",
			port: 443,
			want: true,
		},
		{
			name: "port at end of line",
			line: "ALLOW 22",
			port: 22,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ufwAllowsPort(tt.line, tt.port)
			if got != tt.want {
				t.Errorf("ufwAllowsPort(%q, %d) = %v, want %v", tt.line, tt.port, got, tt.want)
			}
		})
	}
}

func TestParseOSRelease(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		content     string
		wantID      string
		wantVersion string
	}{
		{
			name: "ubuntu 22.04",
			content: `PRETTY_NAME="Ubuntu 22.04.3 LTS"
NAME="Ubuntu"
VERSION_ID="22.04"
VERSION="22.04.3 LTS (Jammy Jellyfish)"
ID=ubuntu
ID_LIKE=debian`,
			wantID:      "ubuntu",
			wantVersion: "22.04",
		},
		{
			name: "debian 12",
			content: `PRETTY_NAME="Debian GNU/Linux 12 (bookworm)"
NAME="Debian GNU/Linux"
VERSION_ID="12"
ID=debian`,
			wantID:      "debian",
			wantVersion: "12",
		},
		{
			name: "quoted values",
			content: `ID="rocky"
VERSION_ID="9.3"`,
			wantID:      "rocky",
			wantVersion: "9.3",
		},
		{
			name:        "empty content",
			content:     "",
			wantID:      "",
			wantVersion: "",
		},
		{
			name: "ID only",
			content: `ID=centos
NAME="CentOS Stream"`,
			wantID:      "centos",
			wantVersion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, version := parseOSRelease(tt.content)
			if id != tt.wantID {
				t.Errorf("id = %q, want %q", id, tt.wantID)
			}
			if version != tt.wantVersion {
				t.Errorf("version = %q, want %q", version, tt.wantVersion)
			}
		})
	}
}

func TestParsePortBindings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		ssOutput     string
		wantBindings int
		wantPorts    []int
	}{
		{
			name: "port 80 and 443 bindings",
			ssOutput: `LISTEN  0  4096  *:80  *:*  users:(("caddy",pid=1234,fd=5))
LISTEN  0  4096  *:443  *:*  users:(("caddy",pid=1234,fd=6))
LISTEN  0  4096  *:22  *:*  users:(("sshd",pid=500,fd=3))`,
			wantBindings: 2,
			wantPorts:    []int{80, 443},
		},
		{
			name: "no edge ports",
			ssOutput: `LISTEN  0  4096  *:22  *:*  users:(("sshd",pid=500,fd=3))
LISTEN  0  4096  *:3000  *:*  users:(("node",pid=1000,fd=4))`,
			wantBindings: 0,
			wantPorts:    nil,
		},
		{
			name:         "empty output",
			ssOutput:     "",
			wantBindings: 0,
			wantPorts:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bindings := parsePortBindings(tt.ssOutput)
			if len(bindings) != tt.wantBindings {
				t.Errorf("parsePortBindings() returned %d bindings, want %d", len(bindings), tt.wantBindings)
			}

			for i, wantPort := range tt.wantPorts {
				if i >= len(bindings) {
					break
				}
				if bindings[i].Port != wantPort {
					t.Errorf("bindings[%d].Port = %d, want %d", i, bindings[i].Port, wantPort)
				}
			}
		})
	}
}
