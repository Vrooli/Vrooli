package vps

import (
	"encoding/json"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/ssh"
	"scenario-to-cloud/sshidentity"
	"scenario-to-cloud/vps/systemmetrics"
	"testing"
)

func TestValidRawJSON(t *testing.T) {
	t.Parallel()

	if got := validRawJSON(""); got != nil {
		t.Fatalf("expected nil for empty input")
	}
	if got := validRawJSON("not json"); got != nil {
		t.Fatalf("expected nil for invalid json")
	}
	if got := validRawJSON(`{"ok":true}`); got == nil {
		t.Fatalf("expected raw json for valid payload")
	}
}

func TestBuildProcessState_IgnoresInvalidRawJSON(t *testing.T) {
	t.Parallel()

	processes := []ProcessInfo{
		{
			User:    "root",
			PID:     123,
			Command: "/root/Vrooli/scenarios/landing-page-business-suite/api/landing-page-business-suite-api",
		},
	}

	state := buildProcessState(
		processes,
		nil,
		"[INFO] The API may not be running",
		"[INFO] resource status unavailable",
		"landing-page-business-suite",
		map[string]bool{"postgres": true},
	)

	if len(state.Scenarios) != 1 {
		t.Fatalf("expected one scenario process, got %d", len(state.Scenarios))
	}
	if len(state.Scenarios[0].VrooliStatus) != 0 {
		t.Fatalf("expected invalid raw JSON to be dropped from scenario status")
	}

	if _, err := json.Marshal(state); err != nil {
		t.Fatalf("expected process state to marshal, got error: %v", err)
	}
}

func TestBuildProcessState_DedupesResourceRowsByID(t *testing.T) {
	t.Parallel()

	processes := []ProcessInfo{
		{
			User:    "postgres",
			PID:     900,
			Command: "postgres: checkpointer process",
		},
		{
			User:    "postgres",
			PID:     800,
			Command: "postgres: writer process",
		},
	}

	state := buildProcessState(
		processes,
		nil,
		`{"scenarios":[]}`,
		`{"resources":[]}`,
		"landing-page-business-suite",
		map[string]bool{"postgres": true},
	)

	if len(state.Resources) != 1 {
		t.Fatalf("expected one postgres resource row, got %d", len(state.Resources))
	}
	if state.Resources[0].ID != "postgres" {
		t.Fatalf("expected postgres resource id, got %q", state.Resources[0].ID)
	}
	if state.Resources[0].PID != 800 {
		t.Fatalf("expected canonical postgres PID to be earliest (800), got %d", state.Resources[0].PID)
	}
}

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

func TestParseRemoteProcStatCPUUsage(t *testing.T) {
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
			result := systemmetrics.ParseCPUUsageFromProcStat(tt.input)
			if result < tt.wantMin || result > tt.wantMax {
				t.Errorf("ParseCPUUsageFromProcStat() = %f, want between %f and %f", result, tt.wantMin, tt.wantMax)
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

	state := parseSystemState(results, sshidentity.DeploymentSSHIdentity{}, "", systemmetrics.CollectorForOS("linux"))
	if state.UptimeSeconds != 389593 {
		t.Errorf("UptimeSeconds = %d, want 389593", state.UptimeSeconds)
	}
}

func TestParseSystemState_Disk(t *testing.T) {
	t.Parallel()

	results := map[string]sshCommandResult{
		"df_kb": {result: ssh.Result{Stdout: "/dev/sda1 209715200 88080384 121634816 42% /"}},
	}

	state := parseSystemState(results, sshidentity.DeploymentSSHIdentity{}, "", systemmetrics.CollectorForOS("linux"))
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
		"meminfo": {result: ssh.Result{Stdout: "MemTotal:       4038656 kB\nMemFree:        1048576 kB\nMemAvailable:   1740800 kB\nBuffers:         102400 kB\nCached:          790528 kB\nSwapTotal:       2097152 kB\nSwapFree:        1572864 kB\n"}},
	}

	state := parseSystemState(results, sshidentity.DeploymentSSHIdentity{}, "", systemmetrics.CollectorForOS("linux"))
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

	state := parseSystemState(results, sshidentity.DeploymentSSHIdentity{AuthMode: sshidentity.AuthModeDefaultSSH}, "", systemmetrics.CollectorForOS("linux"))
	if state.SSH.VerificationState != string(sshidentity.VerificationUnknown) {
		t.Fatalf("VerificationState=%q, want unknown", state.SSH.VerificationState)
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

	state := parseSystemState(results, sshidentity.DeploymentSSHIdentity{
		KeyPath:           "~/.ssh/id_ed25519",
		AuthMode:          sshidentity.AuthModeExplicitKey,
		VerificationState: sshidentity.VerificationUnknown,
	}, pubKey, systemmetrics.CollectorForOS("linux"))
	if state.SSH.VerificationState != string(sshidentity.VerificationAuthorized) {
		t.Fatalf("VerificationState=%q, want authorized", state.SSH.VerificationState)
	}
}
