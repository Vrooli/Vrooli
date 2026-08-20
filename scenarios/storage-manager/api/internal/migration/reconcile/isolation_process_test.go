package reconcile

import (
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

// childScenarioEnv marks a re-executed copy of this test binary as the child.
const childScenarioEnv = "VERIFY_ISOLATION_CHILD_SCENARIO"

// TestScenarioOpensItsOwnDatabaseUnderInheritedEnvironment is the regression
// test for the cross-scenario database hijack, and it deliberately does NOT
// assert on a resolved path.
//
// Every scenario's own unit tests passed while twelve of them shared one
// 9.35 GB file, because the defect lives in process environment inheritance
// rather than in a code path a test exercises. So this test starts a real child
// process carrying a supervisor's full environment, has it OPEN its database,
// and then asks the kernel — through /proc/<pid>/fd — which file the process
// actually attached to. A resolved path proves what the code computes; an open
// descriptor proves what happened.
func TestScenarioOpensItsOwnDatabaseUnderInheritedEnvironment(t *testing.T) {
	if scenario := strings.TrimSpace(os.Getenv(childScenarioEnv)); scenario != "" {
		openAndHold(t, scenario)
		return
	}
	if runtime.GOOS != "linux" {
		t.Skip("open file descriptors are read from /proc, which is Linux-only")
	}

	root := t.TempDir()
	supervisorData := filepath.Join(root, "supervisor-data")
	if err := os.MkdirAll(supervisorData, 0o755); err != nil {
		t.Fatal(err)
	}
	hijacked := filepath.Join(supervisorData, "autoheal.sqlite")

	for _, scenario := range []string{"test-genie", "plan-manager", "git-control-tower"} {
		t.Run(scenario, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=TestScenarioOpensItsOwnDatabaseUnderInheritedEnvironment")
			cmd.Env = append(os.Environ(),
				childScenarioEnv+"="+scenario,
				// Exactly what a child inherits when a supervisor exec's the CLI.
				"SQLITE_PATH="+hijacked,
				"SQLITE_DB="+hijacked,
				"SCENARIO_DATA_DIR="+supervisorData,
				"SCENARIO_NAME=vrooli-autoheal",
				"VROOLI_SCENARIO=vrooli-autoheal",
				"VROOLI_VARIANT=",
				"VROOLI_STORAGE_NAMESPACE=",
				// Keep the child's storage inside the test's temp tree.
				"VROOLI_STORAGE_ROOT="+filepath.Join(root, "storage"),
			)
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			stdin, err := cmd.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() {
				_ = stdin.Close()
				_ = cmd.Wait()
			}()

			buf := make([]byte, 6)
			if _, err := stdout.Read(buf); err != nil {
				t.Fatalf("child never reported readiness: %v", err)
			}

			opened := databaseDescriptorOf(t, cmd.Process.Pid)
			if strings.Contains(opened, "autoheal") {
				t.Fatalf("the child opened the supervisor's database: %s", opened)
			}
			if !strings.Contains(opened, scenario) {
				t.Fatalf("the child opened %s, which is not scoped to %s", opened, scenario)
			}
		})
	}
}

// openAndHold resolves and opens the scenario's database, reports readiness,
// and blocks so the parent can inspect its descriptors.
func openAndHold(t *testing.T, scenario string) {
	t.Helper()
	dsn, err := storage.SQLiteDSN(storage.SQLiteConfig{Scenario: scenario})
	if err != nil {
		t.Fatalf("resolve %s: %v", scenario, err)
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open %s: %v", scenario, err)
	}
	defer db.Close()
	// modernc opens the file lazily; a ping forces the descriptor to exist.
	if err := db.Ping(); err != nil {
		t.Fatalf("ping %s: %v", scenario, err)
	}
	if _, err := os.Stdout.WriteString("ready\n"); err != nil {
		t.Fatal(err)
	}
	var discard [1]byte
	_, _ = os.Stdin.Read(discard[:])
}

// databaseDescriptorOf asks the kernel which database the process has open.
func databaseDescriptorOf(t *testing.T, pid int) string {
	t.Helper()
	fdDir := filepath.Join("/proc", itoa(pid), "fd")
	entries, err := os.ReadDir(fdDir)
	if err != nil {
		t.Fatalf("read %s: %v", fdDir, err)
	}
	for _, entry := range entries {
		target, err := os.Readlink(filepath.Join(fdDir, entry.Name()))
		if err != nil {
			continue
		}
		// The -wal and -shm sidecars are not the answer.
		if strings.HasSuffix(target, ".db") || strings.HasSuffix(target, ".sqlite") {
			return target
		}
	}
	t.Fatalf("no database descriptor among the child's %d open descriptors", len(entries))
	return ""
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
