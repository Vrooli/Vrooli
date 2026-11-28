//go:build linux

package agentsession

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// fixtureProc writes a /proc tree with the kernel's own file formats and
// points the readers at it.
func fixtureProc(t *testing.T, procs map[int]map[string]string) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "stat"), []byte("cpu  1 2 3\nbtime 1790000000\nprocesses 9\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for pid, files := range procs {
		dir := filepath.Join(root, strconv.Itoa(pid))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		for name, body := range files {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	previous := procRoot
	procRoot = root
	t.Cleanup(func() { procRoot = previous })
}

// stat builds a /proc/<pid>/stat line; fields after the comm are state,
// ppid, ... with starttime as the 20th.
func stat(pid, ppid int, comm, start string) string {
	fields := []string{"S", strconv.Itoa(ppid)}
	for len(fields) < 19 {
		fields = append(fields, "0")
	}
	fields = append(fields, start, "0", "0")
	return strconv.Itoa(pid) + " (" + comm + ") " + strings.Join(fields, " ") + "\n"
}

func TestProcessFactsReadParentAndStartPastAHostileComm(t *testing.T) {
	fixtureProc(t, map[int]map[string]string{
		42: {"stat": stat(42, 7, "evil) S 1 (name", "123456")},
	})
	fact, err := processFact(42)
	if err != nil {
		t.Fatal(err)
	}
	if fact.PPID != 7 || fact.StartTime != "123456" {
		t.Fatalf("fact = %+v, want ppid 7 start 123456", fact)
	}
	at, err := processStartedAt(42)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Unix(1790000000, 0).UTC().Add(1234560 * time.Millisecond); !at.Equal(want) {
		t.Fatalf("started at %s, want %s", at, want)
	}
	if _, err := processStartTime(99); !errors.Is(err, ErrProcessGone) {
		t.Fatalf("a missing pid = %v, want ErrProcessGone", err)
	}
}

// The app is alive while a process outside the agent slice carries its data
// directory; the leaked helpers themselves (inside the slice) do not count.
func TestDesktopAppProcessesExcludeTheAgentSlice(t *testing.T) {
	data := "/home/op/.config/web-console-desktop/runtime"
	environ := "HOME=/home/op\x00VROOLI_DESKTOP_MODE=true\x00VROOLI_DATA=" + data + "\x00"
	fixtureProc(t, map[int]map[string]string{
		10: {"environ": environ, "cgroup": "0::/user.slice/user-1000.slice/user@1000.service/vrooli-agents.slice/vrooli-agent-opencode-1.scope\n"},
		11: {"environ": "VROOLI_DATA=/other\x00", "cgroup": "0::/user.slice/app.slice/x.scope\n"},
	})
	pids, err := desktopAppProcesses(data)
	if err != nil || len(pids) != 0 {
		t.Fatalf("pids = %v, %v; want none (only leaked helpers carry the data dir)", pids, err)
	}
	fixtureProc(t, map[int]map[string]string{
		10: {"environ": environ, "cgroup": "0::/user.slice/user-1000.slice/user@1000.service/vrooli-agents.slice/vrooli-agent-opencode-1.scope\n"},
		20: {"environ": environ, "cgroup": "0::/user.slice/user-1000.slice/user@1000.service/app.slice/app-runtime.scope\n"},
	})
	pids, err = desktopAppProcesses(data)
	if err != nil || len(pids) != 1 || pids[0] != 20 {
		t.Fatalf("pids = %v, %v; want the app runtime 20", pids, err)
	}
}

func TestProcessMemoryReadsVmRSSAndVmSwap(t *testing.T) {
	fixtureProc(t, map[int]map[string]string{
		30: {"status": "Name:\tcodex\nVmPeak:\t 9999 kB\nVmRSS:\t   2048 kB\nVmSwap:\t    512 kB\n"},
		31: {"status": "Name:\tzombie\nState:\tZ (zombie)\n"},
	})
	rss, swap, err := processMemory(30)
	if err != nil || rss != 2048*1024 || swap != 512*1024 {
		t.Fatalf("processMemory(30) = %d, %d, %v", rss, swap, err)
	}
	if rss, swap, err := processMemory(31); err != nil || rss != 0 || swap != 0 {
		t.Fatalf("a zombie holds nothing: %d, %d, %v", rss, swap, err)
	}
	if _, _, err := processMemory(32); !errors.Is(err, ErrProcessGone) {
		t.Fatalf("a missing pid = %v, want ErrProcessGone", err)
	}
}
