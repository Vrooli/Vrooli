package kdumpobservability

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// renderForSandbox rewrites the collector's absolute paths to temporary ones and
// shims the two privileged commands it uses, so the real script can be executed
// as an ordinary user. Everything else — the directory scan, the summary
// extraction, the manifest, the pruning guards — runs unmodified.
func renderForSandbox(t *testing.T, retain int, src, dst, shimBin string) string {
	t.Helper()

	script := collectorContent(retain)
	script = strings.Replace(script, `src="`+crashSourceDir+`"`, `src="`+src+`"`, 1)
	script = strings.Replace(script, `dst="`+crashExportDir+`"`, `dst="`+dst+`"`, 1)
	script = strings.Replace(script, "#!/bin/sh\nset -eu\n", "#!/bin/sh\nset -eu\nPATH=\""+shimBin+":$PATH\"\n", 1)

	// chown needs root; ownership is not what this test is proving.
	write(t, filepath.Join(shimBin, "chown"), "#!/bin/sh\nexit 0\n")
	// install -d -o root -g <group> -m <mode> <dir> likewise; keep the mkdir.
	write(t, filepath.Join(shimBin, "install"), "#!/bin/sh\nfor a in \"$@\"; do d=\"$a\"; done\nmkdir -p \"$d\"\n")

	path := filepath.Join(t.TempDir(), "kdump-collector")
	write(t, path, script)
	return path
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func seedDump(t *testing.T, src, stamp string) {
	t.Helper()
	dir := filepath.Join(src, stamp)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("seed %s: %v", stamp, err)
	}
	body := "earlier boot noise\n" +
		"kernel BUG at fs/iomap/buffered-io.c:1061!\n" +
		"CPU: 13 UID: 1000 PID: 3114672 Comm: kopia Kdump: loaded\n" +
		"RIP: 0010:iomap_write_end+0x1ea/0x1f0\n"
	write(t, filepath.Join(dir, "dmesg."+stamp), body)
	write(t, filepath.Join(dir, "dump."+stamp), strings.Repeat("x", 4096))
}

type collectorManifest struct {
	SourcePath    string `json:"sourcePath"`
	RetainVmcores int    `json:"retainVmcores"`
	DumpCount     int    `json:"dumpCount"`
	Dumps         []struct {
		Stamp   string `json:"stamp"`
		Summary string `json:"summary"`
		Reason  string `json:"reason"`
		Comm    string `json:"comm"`
		Bytes   int64  `json:"bytes"`
	} `json:"dumps"`
}

func runCollector(t *testing.T, retain int, stamps []string, extra func(src string)) (string, collectorManifest) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("collector is a Linux shell script")
	}
	root := t.TempDir()
	src := filepath.Join(root, "crash")
	dst := filepath.Join(root, "export")
	shimBin := filepath.Join(root, "bin")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	for _, stamp := range stamps {
		seedDump(t, src, stamp)
	}
	if extra != nil {
		extra(src)
	}

	script := renderForSandbox(t, retain, src, dst, shimBin)
	out, err := exec.Command("/bin/sh", script).CombinedOutput()
	if err != nil {
		t.Fatalf("collector failed: %v\n%s", err, out)
	}

	raw, err := os.ReadFile(filepath.Join(dst, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var m collectorManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("manifest is not valid JSON (%v):\n%s", err, raw)
	}
	return src, m
}

// The end-to-end shape: every kdump directory becomes a summary, and the
// manifest carries the fields an incident report leads with.
func TestCollectorExportsSummariesAndManifest(t *testing.T) {
	_, m := runCollector(t, 5, []string{"202608180101", "202608190202"}, nil)

	if m.DumpCount != 2 {
		t.Fatalf("dumpCount = %d, want 2", m.DumpCount)
	}
	for _, dump := range m.Dumps {
		if !strings.Contains(dump.Reason, "kernel BUG at fs/iomap/buffered-io.c:1061") {
			t.Errorf("reason not extracted for %s: %q", dump.Stamp, dump.Reason)
		}
		if dump.Comm != "kopia" {
			t.Errorf("comm = %q, want kopia", dump.Comm)
		}
		if dump.Bytes <= 0 {
			t.Errorf("bytes = %d, want the on-disk dump size", dump.Bytes)
		}
	}
}

// Pruning keeps the newest N dumps and every summary.
func TestCollectorPrunesOldestDumpsButKeepsSummaries(t *testing.T) {
	stamps := []string{"202608180101", "202608190202", "202608191459"}
	src, m := runCollector(t, 2, stamps, nil)

	if _, err := os.Stat(filepath.Join(src, "202608180101")); !os.IsNotExist(err) {
		t.Error("oldest vmcore directory should have been pruned")
	}
	for _, keep := range []string{"202608190202", "202608191459"} {
		if _, err := os.Stat(filepath.Join(src, keep)); err != nil {
			t.Errorf("newest dumps must survive pruning, %s: %v", keep, err)
		}
	}
	if len(m.Dumps) != 3 {
		t.Fatalf("manifest should describe all %d dumps, got %d", 3, len(m.Dumps))
	}
}

// Everything in /var/crash that kdump did not write belongs to another tool.
// The collector must neither export nor delete it.
func TestCollectorIgnoresForeignCrashArtifacts(t *testing.T) {
	src, m := runCollector(t, 1, []string{"202608190202", "202608191459"}, func(src string) {
		write(t, filepath.Join(src, "_usr_bin_node.1000.crash"), "apport report\n")
		write(t, filepath.Join(src, "not-a-kdump-dir", "keep"), "unrelated\n")
	})

	for _, dump := range m.Dumps {
		if strings.Contains(dump.Stamp, "crash") || dump.Stamp == "not-a-kdump-dir" {
			t.Errorf("foreign artifact was exported: %q", dump.Stamp)
		}
	}
	if _, err := os.Stat(filepath.Join(src, "_usr_bin_node.1000.crash")); err != nil {
		t.Errorf("apport report must survive: %v", err)
	}
	if _, err := os.Stat(filepath.Join(src, "not-a-kdump-dir", "keep")); err != nil {
		t.Errorf("unrelated directory must survive pruning: %v", err)
	}
}

// A second run must not re-export or double-count, so the timer can fire hourly
// without churning the export directory.
func TestCollectorIsIdempotent(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("collector is a Linux shell script")
	}
	root := t.TempDir()
	src := filepath.Join(root, "crash")
	dst := filepath.Join(root, "export")
	shimBin := filepath.Join(root, "bin")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	seedDump(t, src, "202608191459")
	script := renderForSandbox(t, 5, src, dst, shimBin)

	for i := range 2 {
		if out, err := exec.Command("/bin/sh", script).CombinedOutput(); err != nil {
			t.Fatalf("run %d failed: %v\n%s", i+1, err, out)
		}
	}

	entries, err := os.ReadDir(dst)
	if err != nil {
		t.Fatalf("read export dir: %v", err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if len(names) != 2 {
		t.Fatalf("export dir should hold one summary and one manifest, got %v", names)
	}
}
