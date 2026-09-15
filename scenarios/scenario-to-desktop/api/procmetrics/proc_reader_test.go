package procmetrics

import (
	"fmt"
	"testing"
	"testing/fstest"
)

func TestParseProcessStatAndRoleClassification(t *testing.T) {
	input := "1234 (Electron Helper (GPU)) S 1000 1234 1234 0 -1 4194304 500 0 0 0 150 30 0 0 20 0 1 0 100 1000000 200 0 0 0"
	ppid, command, utime, stime, err := parseProcessStat(input)
	if err != nil {
		t.Fatalf("parseProcessStat failed: %v", err)
	}
	if ppid != 1000 || command != "Electron Helper (GPU)" || utime != 150 || stime != 30 {
		t.Fatalf("parsed process = %d %q %d %d", ppid, command, utime, stime)
	}
	if got := classifyProcess(ProcessInfo{PID: 1235, Command: command, Cmdline: []string{"electron", "--type=gpu-process"}, Exe: "/opt/app/app"}, 1234, ProcessTreeOptions{}); got != RoleElectronGPU {
		t.Fatalf("role = %q, want %q", got, RoleElectronGPU)
	}
}

func TestLinuxProcReaderReanchorsAndClassifiesFixtureTree(t *testing.T) {
	proc := fstest.MapFS{}
	add := func(pid, ppid int, command, exe, cmdline string) {
		stat := []byte(fmt.Sprintf("%d (%s) S %d 1 1 0 -1 0 0 0 0 0 10 2 0 0 20 0 1 0 1 1 1 0 0 0", pid, command, ppid))
		status := []byte("VmPeak:\t100 kB\nVmRSS:\t50 kB\nThreads:\t2\n")
		proc[fmt.Sprintf("%d/stat", pid)] = &fstest.MapFile{Data: stat}
		proc[fmt.Sprintf("%d/status", pid)] = &fstest.MapFile{Data: status}
		proc[fmt.Sprintf("%d/cmdline", pid)] = &fstest.MapFile{Data: []byte(cmdline + "\x00")}
		proc[fmt.Sprintf("%d/exe-target", pid)] = &fstest.MapFile{Data: []byte(exe)}
	}
	add(100, 1, "sh", "/usr/bin/sh", "sh -c launch")
	add(200, 100, "hello-desktop", "/tmp/squashfs-root/hello-desktop", "hello-desktop --smoke-test")
	add(201, 200, "hello-desktop", "/tmp/squashfs-root/hello-desktop", "hello-desktop\x00--type=renderer")
	add(202, 200, "hello-desktop", "/tmp/squashfs-root/hello-desktop", "hello-desktop\x00--type=gpu-process")
	add(203, 200, "runtime", "/tmp/squashfs-root/resources/bundle/runtime/linux-x64/runtime", "runtime")
	add(204, 200, "foo-api", "/tmp/squashfs-root/resources/bundle/bin/api/linux-x64/foo-api", "foo-api")
	add(205, 100, "Xvfb", "/usr/bin/Xvfb", "Xvfb :99")

	reader := &LinuxProcReader{FS: proc}
	got, err := reader.ProcessTreeWithOptions(100, ProcessTreeOptions{AppExecutableName: "hello-desktop"})
	if err != nil {
		t.Fatal(err)
	}
	roles := map[int]ProcessRole{}
	for _, p := range got {
		roles[p.PID] = p.Role
	}
	want := map[int]ProcessRole{100: RoleLauncher, 200: RoleElectronMain, 201: RoleElectronRender, 202: RoleElectronGPU, 203: RoleBundledRuntime, 204: RoleScenarioService, 205: RoleUnknown}
	for pid, role := range want {
		if roles[pid] != role {
			t.Errorf("pid %d role = %q, want %q", pid, roles[pid], role)
		}
	}
}

func TestParseStat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		utime   int64
		stime   int64
		wantErr bool
	}{
		{
			name:  "normal process",
			input: "1234 (myapp) S 1233 1234 1234 0 -1 4194304 500 0 0 0 150 30 0 0 20 0 1 0 100 1000000 200 18446744073709551615 0 0 0 0 0 0 0 0 0 0 0 0 17 0 0 0 0 0 0",
			utime: 150,
			stime: 30,
		},
		{
			name:  "process name with spaces and parens",
			input: "5678 (my (fancy) app) S 5677 5678 5678 0 -1 4194304 100 0 0 0 42 18 0 0 20 0 4 0 200 2000000 300 18446744073709551615 0 0 0 0 0 0 0 0 0 0 0 0 17 0 0 0 0 0 0",
			utime: 42,
			stime: 18,
		},
		{
			name:    "malformed no paren",
			input:   "1234 myapp S 1233",
			wantErr: true,
		},
		{
			name:    "too few fields",
			input:   "1234 (app) S 1 2",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			utime, stime, err := parseStat(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if utime != tt.utime {
				t.Errorf("utime = %d, want %d", utime, tt.utime)
			}
			if stime != tt.stime {
				t.Errorf("stime = %d, want %d", stime, tt.stime)
			}
		})
	}
}

func TestParseStatus(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		rssBytes  int64
		peakBytes int64
		threads   int
		wantErr   bool
	}{
		{
			name: "normal status",
			input: `Name:	myapp
Umask:	0022
State:	S (sleeping)
Tgid:	1234
Ngid:	0
Pid:	1234
PPid:	1233
VmPeak:	  204800 kB
VmSize:	  180000 kB
VmRSS:	  102400 kB
Threads:	8
`,
			rssBytes:  102400 * 1024,
			peakBytes: 204800 * 1024,
			threads:   8,
		},
		{
			name: "missing VmRSS",
			input: `Name:	app
VmPeak:	1000 kB
Threads:	1
`,
			wantErr: true,
		},
		{
			name:    "empty content",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rss, peak, threads, err := parseStatus(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rss != tt.rssBytes {
				t.Errorf("rssBytes = %d, want %d", rss, tt.rssBytes)
			}
			if peak != tt.peakBytes {
				t.Errorf("peakBytes = %d, want %d", peak, tt.peakBytes)
			}
			if threads != tt.threads {
				t.Errorf("threads = %d, want %d", threads, tt.threads)
			}
		})
	}
}
