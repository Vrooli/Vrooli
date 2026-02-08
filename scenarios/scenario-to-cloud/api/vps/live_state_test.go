package vps

import (
	"testing"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/ssh"
	"scenario-to-cloud/vps/systemmetrics"
)

func TestParsePSOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantCount  int
		checkFirst func(t *testing.T, p ProcessInfo)
	}{
		{
			name:      "empty output",
			input:     "",
			wantCount: 0,
		},
		{
			name:      "malformed line (too few fields)",
			input:     "root 1 0.0",
			wantCount: 0,
		},
		{
			name:      "normal process line",
			input:     "root         1  0.1  0.2  16892  4096 ?        Ss   Jan01   0:05 /sbin/init",
			wantCount: 1,
			checkFirst: func(t *testing.T, p ProcessInfo) {
				if p.User != "root" {
					t.Errorf("User = %q, want root", p.User)
				}
				if p.PID != 1 {
					t.Errorf("PID = %d, want 1", p.PID)
				}
				if p.Command != "/sbin/init" {
					t.Errorf("Command = %q, want /sbin/init", p.Command)
				}
				if p.CPUPercent != 0.1 {
					t.Errorf("CPUPercent = %f, want 0.1", p.CPUPercent)
				}
			},
		},
		{
			name: "multiple processes",
			input: `root         1  0.0  0.1  16892  2048 ?        Ss   Jan01   0:05 /sbin/init
www-data  1234  2.5  1.3 123456 26624 ?        Sl   10:00   0:30 node /app/server.js --port 3000`,
			wantCount: 2,
			checkFirst: func(t *testing.T, p ProcessInfo) {
				if p.PID != 1 {
					t.Errorf("first PID = %d, want 1", p.PID)
				}
			},
		},
		{
			name:      "command with spaces",
			input:     "user      999  0.0  0.0  12345  1024 ?        S    09:00   0:00 python3 -m http.server 8080",
			wantCount: 1,
			checkFirst: func(t *testing.T, p ProcessInfo) {
				if p.Command != "python3 -m http.server 8080" {
					t.Errorf("Command = %q, want 'python3 -m http.server 8080'", p.Command)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePSOutput(tt.input)
			if len(result) != tt.wantCount {
				t.Fatalf("parsePSOutput() returned %d processes, want %d", len(result), tt.wantCount)
			}
			if tt.checkFirst != nil && len(result) > 0 {
				tt.checkFirst(t, result[0])
			}
		})
	}
}

func TestParseSSOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantCount int
		checkPort func(t *testing.T, ports []domain.PortBinding)
	}{
		{
			name:      "empty output",
			input:     "",
			wantCount: 0,
		},
		{
			name:      "header only",
			input:     "State  Recv-Q Send-Q Local Address:Port  Peer Address:Port Process",
			wantCount: 0,
		},
		{
			name:      "normal listening port",
			input:     `LISTEN 0      4096         0.0.0.0:22          0.0.0.0:*    users:(("sshd",pid=1234,fd=3))`,
			wantCount: 1,
			checkPort: func(t *testing.T, ports []domain.PortBinding) {
				if ports[0].Port != 22 {
					t.Errorf("Port = %d, want 22", ports[0].Port)
				}
				if ports[0].Process != "sshd" {
					t.Errorf("Process = %q, want sshd", ports[0].Process)
				}
				if ports[0].PID == nil || *ports[0].PID != 1234 {
					t.Errorf("PID not correctly parsed")
				}
			},
		},
		{
			name: "multiple ports",
			input: `LISTEN 0      4096         0.0.0.0:22          0.0.0.0:*    users:(("sshd",pid=100,fd=3))
LISTEN 0      4096         0.0.0.0:80          0.0.0.0:*    users:(("caddy",pid=200,fd=7))
LISTEN 0      4096         0.0.0.0:443         0.0.0.0:*    users:(("caddy",pid=200,fd=8))`,
			wantCount: 3,
		},
		{
			name:      "IPv6 address",
			input:     `LISTEN 0      4096              [::]:22             [::]:*    users:(("sshd",pid=1234,fd=4))`,
			wantCount: 1,
			checkPort: func(t *testing.T, ports []domain.PortBinding) {
				if ports[0].Port != 22 {
					t.Errorf("Port = %d, want 22", ports[0].Port)
				}
			},
		},
		{
			name:      "no process info",
			input:     `LISTEN 0      4096         0.0.0.0:5432        0.0.0.0:*`,
			wantCount: 1,
			checkPort: func(t *testing.T, ports []domain.PortBinding) {
				if ports[0].Port != 5432 {
					t.Errorf("Port = %d, want 5432", ports[0].Port)
				}
				if ports[0].Process != "" {
					t.Errorf("Process = %q, want empty", ports[0].Process)
				}
			},
		},
		{
			name:      "skip non-LISTEN line",
			input:     `ESTAB  0      0      10.0.0.1:22    10.0.0.2:54321`,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSSOutput(tt.input)
			if len(result) != tt.wantCount {
				t.Fatalf("ParseSSOutput() returned %d ports, want %d", len(result), tt.wantCount)
			}
			if tt.checkPort != nil && len(result) > 0 {
				tt.checkPort(t, result)
			}
		})
	}
}

func TestParseCPUUsageFromTop(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantMin float64
		wantMax float64
	}{
		{
			name:    "empty input",
			input:   "",
			wantMin: 0,
			wantMax: 0,
		},
		{
			name: "two samples (idle system)",
			input: `cpu  1000 50 200 50000 100 10 5 0
cpu  1010 50 200 50100 100 10 5 0`,
			wantMin: 0,
			wantMax: 30,
		},
		{
			name: "two samples (busy system)",
			input: `cpu  1000 50 200 5000 100 10 5 0
cpu  2000 50 300 5050 100 10 5 0`,
			wantMin: 50,
			wantMax: 100,
		},
		{
			name:    "single sample fallback",
			input:   `cpu  10000 500 3000 80000 1000 100 50 0`,
			wantMin: 0,
			wantMax: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCPUUsageFromTop(tt.input)
			if result < tt.wantMin || result > tt.wantMax {
				t.Errorf("parseCPUUsageFromTop() = %f, want between %f and %f", result, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestParseHumanSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  int
	}{
		{"200G", 200},
		{"1T", 1024},
		{"512M", 0},
		{"1024M", 1},
		{"100K", 0},
		{"0G", 0},
		{"2.5T", 2560},
		{"", 0},
		{"notanumber", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseHumanSize(tt.input)
			if result != tt.want {
				t.Errorf("parseHumanSize(%q) = %d, want %d", tt.input, result, tt.want)
			}
		})
	}
}

func TestCategorizePortsWithManifest(t *testing.T) {
	t.Parallel()

	ports := []domain.PortBinding{
		{Port: 22, Process: "sshd"},
		{Port: 80, Process: "caddy"},
		{Port: 443, Process: "caddy"},
		{Port: 35000, Process: "node"},
		{Port: 5432, Process: "postgres"},
		{Port: 9999, Process: "mystery"},
	}

	processState := domain.ProcessState{
		Scenarios: []domain.ScenarioProcess{
			{
				ID:     "my-scenario",
				Status: "running",
				PID:    1000,
				Ports: []domain.ProcessPort{
					{Port: 35000, Status: "listening"},
				},
			},
		},
		Resources: []domain.ResourceProcess{
			{ID: "postgres", Status: "running", Port: 5432, PID: 2000},
		},
	}

	manifest := domain.CloudManifest{}

	result := categorizePortsWithManifest(ports, processState, manifest)

	if len(result) != len(ports) {
		t.Fatalf("expected %d ports, got %d", len(ports), len(result))
	}

	typeMap := make(map[int]string)
	for _, p := range result {
		typeMap[p.Port] = p.Type
	}

	expected := map[int]string{
		22:    "system",
		80:    "edge",
		443:   "edge",
		35000: "scenario",
		5432:  "resource",
		9999:  "unexpected",
	}

	for port, wantType := range expected {
		if typeMap[port] != wantType {
			t.Errorf("port %d: type = %q, want %q", port, typeMap[port], wantType)
		}
	}
}

func TestParseSystemState_Uptime(t *testing.T) {
	t.Parallel()

	results := map[string]sshCommandResult{
		"uptime": {result: ssh.Result{Stdout: "389593.24 1558372.96"}},
	}

	state := parseSystemState(results, "", "", systemmetrics.CollectorForOS("linux"))
	if state.UptimeSeconds != 389593 {
		t.Errorf("UptimeSeconds = %d, want 389593", state.UptimeSeconds)
	}
}

func TestParseSystemState_Disk(t *testing.T) {
	t.Parallel()

	results := map[string]sshCommandResult{
		"df": {result: ssh.Result{Stdout: "/dev/sda1      200G   84G  116G  42% /"}},
	}

	state := parseSystemState(results, "", "", systemmetrics.CollectorForOS("linux"))
	if state.Disk.TotalGB != 200 {
		t.Errorf("Disk.TotalGB = %d, want 200", state.Disk.TotalGB)
	}
	if state.Disk.UsedGB != 84 {
		t.Errorf("Disk.UsedGB = %d, want 84", state.Disk.UsedGB)
	}
	if state.Disk.FreeGB != 116 {
		t.Errorf("Disk.FreeGB = %d, want 116", state.Disk.FreeGB)
	}
	if state.Disk.UsagePercent != 42 {
		t.Errorf("Disk.UsagePercent = %f, want 42", state.Disk.UsagePercent)
	}
}

func TestParseSystemState_Memory(t *testing.T) {
	t.Parallel()

	results := map[string]sshCommandResult{
		"free": {result: ssh.Result{Stdout: "Mem:           3944        2048        1024         100         872        1700\nSwap:          2048         512        1536"}},
	}

	state := parseSystemState(results, "", "", systemmetrics.CollectorForOS("linux"))
	if state.Memory.TotalMB != 3944 {
		t.Errorf("Memory.TotalMB = %d, want 3944", state.Memory.TotalMB)
	}
	if state.Memory.UsedMB != 2244 {
		t.Errorf("Memory.UsedMB = %d, want 2244", state.Memory.UsedMB)
	}
	if state.Memory.FreeMB != 1700 {
		t.Errorf("Memory.FreeMB = %d, want 1700", state.Memory.FreeMB)
	}
	if state.Swap.TotalMB != 2048 {
		t.Errorf("Swap.TotalMB = %d, want 2048", state.Swap.TotalMB)
	}
}

func TestParseSystemState_SSHKeyAuthUnknownWithoutKey(t *testing.T) {
	t.Parallel()

	results := map[string]sshCommandResult{
		"ssh_ping":      {result: ssh.Result{ExitCode: 0}},
		"ssh_key_check": {result: ssh.Result{Stdout: "ssh-ed25519 AAAA existing-key user@host", ExitCode: 0}},
	}

	state := parseSystemState(results, "", "", systemmetrics.CollectorForOS("linux"))
	if state.SSH.KeyInAuthState != "unknown" {
		t.Fatalf("KeyInAuthState=%q, want unknown", state.SSH.KeyInAuthState)
	}
}

func TestParseSystemState_SSHKeyAuthAuthorized(t *testing.T) {
	t.Parallel()

	pubKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey generated-by-test"
	results := map[string]sshCommandResult{
		"ssh_ping": {
			result: ssh.Result{ExitCode: 0},
		},
		"ssh_key_check": {
			result: ssh.Result{Stdout: pubKey + "\nssh-ed25519 AAAA other", ExitCode: 0},
		},
	}

	state := parseSystemState(results, "~/.ssh/id_ed25519", pubKey, systemmetrics.CollectorForOS("linux"))
	if state.SSH.KeyInAuthState != "authorized" {
		t.Fatalf("KeyInAuthState=%q, want authorized", state.SSH.KeyInAuthState)
	}
	if !state.SSH.KeyInAuth {
		t.Fatal("KeyInAuth=false, want true")
	}
}
