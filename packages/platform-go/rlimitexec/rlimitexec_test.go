package rlimitexec

import (
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// TestParseArgs pins the shim's flag grammar: flags before "--" populate the
// Spec, everything after "--" is the target command, and a missing target is
// an error. OS-neutral, so it runs on the Linux dev host.
func TestParseArgs(t *testing.T) {
	cases := []struct {
		name       string
		args       []string
		wantSpec   Spec
		wantTarget []string
		wantErr    bool
	}{
		{
			name:       "all limits",
			args:       []string{"--as=1048576", "--cpu=30", "--nproc=64", "--nofile=256", "--", "/bin/echo", "hi"},
			wantSpec:   Spec{AddressSpaceBytes: 1048576, CPUTimeSec: 30, MaxProcesses: 64, MaxOpenFiles: 256},
			wantTarget: []string{"/bin/echo", "hi"},
		},
		{
			name:       "subset of limits",
			args:       []string{"--cpu=10", "--", "node", "server.js"},
			wantSpec:   Spec{CPUTimeSec: 10},
			wantTarget: []string{"node", "server.js"},
		},
		{
			name:       "no limits still execs target",
			args:       []string{"--", "true"},
			wantSpec:   Spec{},
			wantTarget: []string{"true"},
		},
		{
			name:       "target flags after -- are not parsed as shim flags",
			args:       []string{"--nofile=128", "--", "ls", "-la", "--color"},
			wantSpec:   Spec{MaxOpenFiles: 128},
			wantTarget: []string{"ls", "-la", "--color"},
		},
		{
			name:    "missing target",
			args:    []string{"--cpu=10"},
			wantErr: true,
		},
		{
			name:    "unknown flag",
			args:    []string{"--bogus=1", "--", "true"},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec, target, err := ParseArgs(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got spec=%+v target=%v", spec, target)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseArgs: %v", err)
			}
			if spec != tc.wantSpec {
				t.Errorf("spec: got %+v, want %+v", spec, tc.wantSpec)
			}
			if !reflect.DeepEqual(target, tc.wantTarget) {
				t.Errorf("target: got %v, want %v", target, tc.wantTarget)
			}
		})
	}
}

// TestSpecLimits pins the OS-neutral Spec -> limit mapping: only set fields
// produce entries, and the order is stable (address-space, cpu, nproc,
// nofile). The apply layer relies on this order being deterministic.
func TestSpecLimits(t *testing.T) {
	cases := []struct {
		name string
		spec Spec
		want []limitValue
	}{
		{"empty", Spec{}, nil},
		{
			name: "all set",
			spec: Spec{AddressSpaceBytes: 2048, CPUTimeSec: 5, MaxProcesses: 32, MaxOpenFiles: 64},
			want: []limitValue{
				{limitAddressSpace, 2048},
				{limitCPUTime, 5},
				{limitProcesses, 32},
				{limitOpenFiles, 64},
			},
		},
		{
			name: "sparse keeps order",
			spec: Spec{CPUTimeSec: 5, MaxOpenFiles: 64},
			want: []limitValue{
				{limitCPUTime, 5},
				{limitOpenFiles, 64},
			},
		},
		{
			name: "negative treated as unset",
			spec: Spec{AddressSpaceBytes: -1, CPUTimeSec: 5},
			want: []limitValue{{limitCPUTime, 5}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.spec.Limits()
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Limits(): got %v, want %v", got, tc.want)
			}
		})
	}
}

// TestRlimitShimAppliesNprocAndAsBeforeExec runs the shim for real: a child
// of this test binary applies the limits and execs `sh -c 'ulimit -u; ulimit -v'`,
// whose output must show the requested ceilings. Linux and macOS only.
func TestRlimitShimAppliesNprocAndAsBeforeExec(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("setrlimit shim runs on linux and darwin only")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is needed to read the limits back")
	}
	if os.Getenv("RLIMITEXEC_HELPER") == "1" {
		start := 0
		for i, arg := range os.Args {
			if arg == Subcommand {
				start = i
				break
			}
		}
		if !MaybeRun(append([]string{os.Args[0]}, os.Args[start:]...)) {
			t.Fatalf("MaybeRun did not route %q", os.Args)
		}
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run", "^TestRlimitShimAppliesNprocAndAsBeforeExec$", Subcommand, "--nproc=4096", "--as=1073741824", "--", "bash", "-c", "ulimit -u; ulimit -v")
	cmd.Env = append(os.Environ(), "RLIMITEXEC_HELPER=1")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("shim helper: %v\n%s", err, output)
	}
	got := strings.Fields(string(output))
	if len(got) < 2 || got[0] != "4096" || got[1] != "1048576" {
		t.Fatalf("limits after exec = %q, want nproc 4096 and 1048576 KiB address space", output)
	}
}
