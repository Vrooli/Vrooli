//go:build linux

package procsampler

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// fakeProcFS is an in-memory /proc used to drive the Linux sampler without the
// host's live process table.
type fakeProcFS struct {
	pids    []int
	files   map[string][]byte // key "<pid>/<name>"
	links   map[string]string // key "<pid>/<name>"
	missing map[int]bool      // pids that vanished mid-walk (ESRCH)
	pidsErr error
}

func (f *fakeProcFS) PIDs() ([]int, error) { return f.pids, f.pidsErr }

func (f *fakeProcFS) ReadFile(pid int, name string) ([]byte, error) {
	if f.missing[pid] {
		return nil, os.ErrNotExist
	}
	b, ok := f.files[fmt.Sprintf("%d/%s", pid, name)]
	if !ok {
		return nil, os.ErrNotExist
	}
	return b, nil
}

func (f *fakeProcFS) Readlink(pid int, name string) (string, error) {
	if f.missing[pid] {
		return "", os.ErrNotExist
	}
	s, ok := f.links[fmt.Sprintf("%d/%s", pid, name)]
	if !ok {
		return "", os.ErrNotExist
	}
	return s, nil
}

// statLine builds a /proc/<pid>/stat line with the fields the parser reads.
// Layout after the closing paren (index): [0]=state [1]=ppid ... [11]=utime
// [12]=stime ... [17]=threads ... [21]=rss(pages).
func statLine(comm string, ppid int, utime, stime uint64, threads, rssPages int) string {
	fields := make([]string, 22)
	for i := range fields {
		fields[i] = "0"
	}
	fields[0] = "S"
	fields[1] = fmt.Sprintf("%d", ppid)
	fields[11] = fmt.Sprintf("%d", utime)
	fields[12] = fmt.Sprintf("%d", stime)
	fields[17] = fmt.Sprintf("%d", threads)
	fields[21] = fmt.Sprintf("%d", rssPages)
	line := "1234 (" + comm + ")"
	for _, f := range fields {
		line += " " + f
	}
	return line
}

func TestLinuxSampler_ParsesFieldsAndCmdlineCwd(t *testing.T) {
	fs := &fakeProcFS{
		pids: []int{10},
		files: map[string][]byte{
			"10/stat":    []byte(statLine("system-monitor-api", 1, 100, 50, 7, 512)),
			"10/status":  []byte("Name:\tsystem-monitor-api\nVmSwap:\t4096 kB\n"),
			"10/cmdline": []byte("system-monitor-api\x00--port\x008080\x00"),
		},
		links: map[string]string{
			"10/cwd": "/home/u/Vrooli/scenarios/system-monitor/api",
		},
	}
	s := newSamplerWithFS(fs, func() time.Time { return time.Unix(0, 0) })
	out, err := s.Sample(context.Background())
	if err != nil {
		t.Fatalf("sample: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("got %d samples, want 1", len(out))
	}
	got := out[0]
	if got.PID != 10 || got.PPID != 1 {
		t.Errorf("pid/ppid = %d/%d, want 10/1", got.PID, got.PPID)
	}
	if got.Comm != "system-monitor-api" {
		t.Errorf("comm = %q", got.Comm)
	}
	if got.Cmdline != "system-monitor-api --port 8080" {
		t.Errorf("cmdline = %q", got.Cmdline)
	}
	if got.Cwd != "/home/u/Vrooli/scenarios/system-monitor/api" {
		t.Errorf("cwd = %q", got.Cwd)
	}
	if got.Threads != 7 {
		t.Errorf("threads = %d, want 7", got.Threads)
	}
	wantRSS := int64(512 * pageSizeKB())
	if got.RSSKB != wantRSS {
		t.Errorf("rss = %d KB, want %d", got.RSSKB, wantRSS)
	}
	if got.SwapKB != 4096 || got.MetricsStatus != "measured" {
		t.Errorf("swap/status = %d/%s", got.SwapKB, got.MetricsStatus)
	}
}

func TestLinuxSampler_MajorFaultRateAndStatusFixture(t *testing.T) {
	stat := statLine("faulting", 1, 0, 0, 1, 8)
	closeParen := strings.LastIndexByte(stat, ')')
	rest := strings.Fields(stat[closeParen+1:])
	rest[9] = "42"
	stat = stat[:closeParen+1] + " " + strings.Join(rest, " ")
	fs := &fakeProcFS{pids: []int{12}, files: map[string][]byte{
		"12/stat": []byte(stat), "12/status": []byte("VmSwap:\t8192 kB\n"),
	}}
	now := time.Unix(100, 0)
	s := newSamplerWithFS(fs, func() time.Time { return now })
	first, err := s.Sample(context.Background())
	if err != nil || len(first) != 1 {
		t.Fatalf("first sample: %v %+v", err, first)
	}
	if first[0].MajorFaults != 42 || first[0].MajorFaultsPerSecond != 0 {
		t.Fatalf("first faults = %+v", first[0])
	}
	now = now.Add(time.Second)
	rest[9] = "52"
	fs.files["12/stat"] = []byte(stat[:closeParen+1] + " " + strings.Join(rest, " "))
	second, _ := s.Sample(context.Background())
	if second[0].MajorFaultsPerSecond != 10 {
		t.Fatalf("major fault rate = %f, want 10", second[0].MajorFaultsPerSecond)
	}
}

func TestLinuxSampler_CommWithSpacesAndParens(t *testing.T) {
	fs := &fakeProcFS{
		pids:  []int{11},
		files: map[string][]byte{"11/stat": []byte(statLine("weird (proc) name", 3, 1, 1, 1, 1))},
	}
	s := newSamplerWithFS(fs, time.Now)
	out, err := s.Sample(context.Background())
	if err != nil {
		t.Fatalf("sample: %v", err)
	}
	if len(out) != 1 || out[0].Comm != "weird (proc) name" || out[0].PPID != 3 {
		t.Fatalf("comm/ppid parse failed: %+v", out)
	}
}

func TestLinuxSampler_CPUDeltaAcrossSamples(t *testing.T) {
	fs := &fakeProcFS{
		pids:  []int{20},
		files: map[string][]byte{"20/stat": []byte(statLine("worker", 1, 0, 0, 1, 0))},
	}
	now := time.Unix(100, 0)
	s := newSamplerWithFS(fs, func() time.Time { return now })

	// First cycle: no prior point -> 0%.
	out, _ := s.Sample(context.Background())
	if out[0].CPUPct != 0 {
		t.Fatalf("first cycle CPU = %f, want 0", out[0].CPUPct)
	}

	// Second cycle 1s later: +100 ticks of utime. With USER_HZ=100, that is 1
	// full CPU-second over 1 elapsed second = 100%.
	now = now.Add(time.Second)
	fs.files["20/stat"] = []byte(statLine("worker", 1, 100, 0, 1, 0))
	out, _ = s.Sample(context.Background())
	if out[0].CPUPct < 99 || out[0].CPUPct > 101 {
		t.Fatalf("second cycle CPU = %f, want ~100", out[0].CPUPct)
	}
}

func TestLinuxSampler_ToleratesVanishedPID(t *testing.T) {
	fs := &fakeProcFS{
		pids: []int{30, 31},
		files: map[string][]byte{
			"30/stat": []byte(statLine("alive", 1, 5, 5, 2, 8)),
			// pid 31 listed but its stat read fails (ESRCH) — must be skipped.
		},
		missing: map[int]bool{31: true},
	}
	s := newSamplerWithFS(fs, time.Now)
	out, err := s.Sample(context.Background())
	if err != nil {
		t.Fatalf("sample should not error on a vanished pid: %v", err)
	}
	if len(out) != 1 || out[0].PID != 30 {
		t.Fatalf("expected only the live pid, got %+v", out)
	}
}

func TestLinuxSampler_PIDReuseDropsBogusSpike(t *testing.T) {
	fs := &fakeProcFS{
		pids:  []int{40},
		files: map[string][]byte{"40/stat": []byte(statLine("a", 1, 500, 0, 1, 0))},
	}
	now := time.Unix(0, 0)
	s := newSamplerWithFS(fs, func() time.Time { return now })
	_, _ = s.Sample(context.Background())

	// pid 40 is reused by a fresh process with a LOWER cumulative tick count;
	// the negative delta must be treated as 0%, not a huge spike.
	now = now.Add(time.Second)
	fs.files["40/stat"] = []byte(statLine("b", 1, 1, 0, 1, 0))
	out, _ := s.Sample(context.Background())
	if out[0].CPUPct != 0 {
		t.Fatalf("pid reuse CPU = %f, want 0", out[0].CPUPct)
	}
}
