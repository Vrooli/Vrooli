//go:build linux

package gocache

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// procfs builds a /proc-shaped tree whose files carry the exact formats the
// auditor parses, with executables that resolve to distinct real inodes.
type procfs struct {
	t        *testing.T
	root     string
	boot     time.Time
	binaries map[string]executable
	exes     string
}

func newProcfs(t *testing.T) *procfs {
	t.Helper()
	p := &procfs{t: t, root: t.TempDir(), boot: time.Now().Add(-time.Hour).Truncate(time.Second), binaries: map[string]executable{}, exes: t.TempDir()}
	p.write("self/status", "Name:\ttest\nNSpid:\t4242\n")
	p.write("stat", fmt.Sprintf("cpu  1 2 3\nbtime %d\nprocesses 9\n", p.boot.Unix()))
	return p
}

func (p *procfs) write(name, content string) {
	p.t.Helper()
	path := filepath.Join(p.root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		p.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		p.t.Fatal(err)
	}
}

func (p *procfs) binary(name string, exe executable) string {
	p.t.Helper()
	path := filepath.Join(p.exes, name)
	if err := os.WriteFile(path, []byte(name), 0o755); err != nil {
		p.t.Fatal(err)
	}
	p.binaries[path] = exe
	return path
}

type proc struct {
	pid, ppid, uid int
	comm, exe      string
	started        time.Duration // after boot
	environ        map[string]string
}

func (p *procfs) add(process proc) {
	p.t.Helper()
	dir := fmt.Sprint(process.pid)
	p.write(dir+"/comm", process.comm+"\n")
	p.write(dir+"/status", fmt.Sprintf("Name:\t%s\nUid:\t%d\t%d\t%d\t%d\n", process.comm, process.uid, process.uid, process.uid, process.uid))
	fields := []string{"S", fmt.Sprint(process.ppid)}
	for len(fields) < 19 {
		fields = append(fields, "0")
	}
	fields = append(fields, fmt.Sprint(int64(process.started/time.Second)*clockTicks), "0", "0")
	p.write(dir+"/stat", fmt.Sprintf("%d (%s) %s", process.pid, process.comm, strings.Join(fields, " ")))
	keys := make([]string, 0, len(process.environ))
	for key := range process.environ {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var environ strings.Builder
	for _, key := range keys {
		environ.WriteString(key + "=" + process.environ[key] + "\x00")
	}
	p.write(dir+"/environ", environ.String())
	p.write(dir+"/cmdline", process.comm+"\x00build\x00./...\x00")
	if err := os.Symlink(process.exe, filepath.Join(p.root, dir, "exe")); err != nil {
		p.t.Fatal(err)
	}
}

func (p *procfs) auditor(store, program string) *auditor {
	p.t.Helper()
	info, err := os.Stat(store)
	if err != nil {
		p.t.Fatal(err)
	}
	return &auditor{
		procRoot: p.root, store: info, program: program, uid: os.Getuid(), now: time.Now,
		readExe: func(path string) (executable, error) {
			target, err := filepath.EvalSymlinks(path)
			if err != nil {
				return executable{}, err
			}
			return p.binaries[target], nil
		},
		executable: map[[2]uint64]executable{},
	}
}

func TestAuditClassifiesEveryWayAGoCommandCanReachTheStore(t *testing.T) {
	p := newProcfs(t)
	home := t.TempDir()
	store := filepath.Join(home, ".cache", "go-build")
	elsewhere := t.TempDir()
	for _, dir := range []string{store, elsewhere} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	program := "/opt/vrooli/bin/" + ProgramName
	envFile := filepath.Join(home, ".config", "go", "env")
	if err := os.MkdirAll(filepath.Dir(envFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envFile, []byte("GOCACHEPROG="+program+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	activated := p.boot.Add(10 * time.Minute)
	if err := os.Chtimes(envFile, activated, activated); err != nil {
		t.Fatal(err)
	}
	modern := p.binary("go1.26", executable{mainPath: goCommandMainPath, goVersion: "go1.26.0"})
	legacy := p.binary("go1.21", executable{mainPath: goCommandMainPath, goVersion: "go1.21.13"})
	helper := p.binary("helper", executable{mainPath: ProgramMainPath, goVersion: "go1.26.0"})
	impostor := p.binary("impostor", executable{mainPath: "example.com/notgo"})
	uid := os.Getuid()
	user := map[string]string{"HOME": home}
	with := func(extra map[string]string) map[string]string {
		out := map[string]string{"HOME": home}
		for key, value := range extra {
			out[key] = value
		}
		return out
	}
	after, before := 20*time.Minute, 5*time.Minute
	helperComm := ProgramName[:commLength-1]

	p.add(proc{pid: 10, ppid: 1, uid: uid, comm: "go", exe: modern, started: before, environ: user})        // has a helper: proven
	p.add(proc{pid: 11, ppid: 10, uid: uid, comm: helperComm, exe: helper, started: before, environ: user}) // its helper
	p.add(proc{pid: 20, ppid: 1, uid: uid, comm: "go", exe: modern, started: after, environ: user})         // will start one
	p.add(proc{pid: 30, ppid: 1, uid: uid, comm: "go", exe: modern, started: before, environ: user})        // pre-activation
	p.add(proc{pid: 40, ppid: 1, uid: uid, comm: "go", exe: modern, started: after, environ: with(map[string]string{"GOENV": "off"})})
	p.add(proc{pid: 50, ppid: 1, uid: uid, comm: "go", exe: legacy, started: after, environ: user})
	p.add(proc{pid: 60, ppid: 1, uid: uid, comm: "go", exe: modern, started: before, environ: with(map[string]string{"GOCACHE": elsewhere})})
	p.add(proc{pid: 70, ppid: 1, uid: uid, comm: "go", exe: modern, started: before, environ: with(map[string]string{"GOCACHEPROG": program})}) // explicit env: no file timing
	p.add(proc{pid: 80, ppid: 1, uid: uid, comm: "go", exe: modern, started: after, environ: with(map[string]string{"GOCACHEPROG": "/usr/bin/other-cacher"})})
	p.add(proc{pid: 90, ppid: 1, uid: uid + 1, comm: "go", exe: modern, started: after, environ: user})
	p.add(proc{pid: 95, ppid: 1, uid: uid, comm: "go", exe: impostor, started: after, environ: user}) // not cmd/go
	p.add(proc{pid: 96, ppid: 1, uid: uid, comm: "bash", exe: modern, started: after, environ: user}) // not named go

	report, err := p.auditor(store, program).run()
	if err != nil {
		t.Fatal(err)
	}
	reasons := map[int]string{}
	for _, finding := range report.Unmediated {
		reasons[finding.PID] = finding.Reason
	}
	want := map[int]string{
		30: "started before its go env file last changed",
		40: "GOCACHEPROG unset",
		50: "predates GOCACHEPROG",
		80: "other-cacher",
		90: "cannot be inspected",
	}
	if len(reasons) != len(want) {
		t.Errorf("unmediated = %v, want pids %v", reasons, want)
	}
	for pid, fragment := range want {
		if !strings.Contains(reasons[pid], fragment) {
			t.Errorf("pid %d reason %q, want it to mention %q", pid, reasons[pid], fragment)
		}
	}
	if report.Mediated != 3 || report.OtherStores != 1 {
		t.Errorf("mediated=%d other_stores=%d, want 3 and 1 (report %+v)", report.Mediated, report.OtherStores, report)
	}
}

func TestAuditTreatsAnUnverifiableEnvFileAsTheDefaultStore(t *testing.T) {
	p := newProcfs(t)
	home := t.TempDir()
	store := filepath.Join(home, ".cache", "go-build")
	if err := os.MkdirAll(store, 0o755); err != nil {
		t.Fatal(err)
	}
	envFile := filepath.Join(home, ".config", "go", "env")
	if err := os.MkdirAll(filepath.Dir(envFile), 0o755); err != nil {
		t.Fatal(err)
	}
	// The file now sends GOCACHE elsewhere, but it changed after the process
	// started; what the process read is unknown, so it may be this store.
	if err := os.WriteFile(envFile, []byte("GOCACHE="+t.TempDir()+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	modern := p.binary("go", executable{mainPath: goCommandMainPath, goVersion: "go1.26.0"})
	p.add(proc{pid: 5, ppid: 1, uid: os.Getuid(), comm: "go", exe: modern, started: time.Minute, environ: map[string]string{"HOME": home}})
	report, err := p.auditor(store, "/bin/"+ProgramName).run()
	if err != nil {
		t.Fatal(err)
	}
	if report.Clear() || report.OtherStores != 0 {
		t.Fatalf("a process whose env file changed after start was exempted: %+v", report)
	}
}

func TestAuditRefusesToRunInsideANestedPIDNamespace(t *testing.T) {
	p := newProcfs(t)
	p.write("self/status", "Name:\ttest\nNSpid:\t4242\t7\n")
	store := t.TempDir()
	if _, err := p.auditor(store, "/bin/"+ProgramName).run(); err == nil || !strings.Contains(err.Error(), "nested") {
		t.Fatalf("nested namespace audit error = %v", err)
	}
}

func TestSupportsCacheProgFollowsGoReleases(t *testing.T) {
	for version, want := range map[string]bool{
		"go1.21.13": false, "go1.23.4": false, "go1.24rc1": true, "go1.24.0": true, "go1.26.0": true,
		"devel go1.27-abcdef": true, "": false, "gccgo": false,
	} {
		if got := supportsCacheProg(version); got != want {
			t.Errorf("supportsCacheProg(%q) = %v, want %v", version, got, want)
		}
	}
}
