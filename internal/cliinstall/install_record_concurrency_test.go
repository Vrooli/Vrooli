package cliinstall

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestConcurrentInstallRecordWritersPreserveEveryArtifactAndProvider(t *testing.T) {
	_, home := newUninstallTestService(t)
	const writers = 24
	start := make(chan struct{})
	errs := make(chan error, writers)
	var workers sync.WaitGroup
	for i := 0; i < writers; i++ {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			<-start
			if i%2 == 0 {
				path := filepath.Join(home, fmt.Sprintf("artifact-%d", i))
				errs <- RecordInstallEntries(home, InstallEntry{Scope: ScopeRuntime, Kind: EntryFile, Path: path, Prefix: home})
			} else {
				errs <- RecordContainerRuntime(home, fmt.Sprintf("provider-%d", i), fmt.Sprintf("endpoint-%d", i), "node", ObservedPresent, ActionAdopted)
			}
		}(i)
	}
	close(start)
	workers.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("concurrent install record write: %v", err)
		}
	}
	record, err := LoadInstallRecord(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(record.Entries) != writers/2 || len(record.RuntimeProviders) != writers/2 {
		t.Fatalf("concurrent writers lost installed ownership: artifacts=%d providers=%d; want %d each", len(record.Entries), len(record.RuntimeProviders), writers/2)
	}
}

func TestInstallRecordWriterProcessFixture(t *testing.T) {
	home := os.Getenv("VROOLI_INSTALL_RECORD_TEST_HOME")
	if home == "" {
		t.Skip("isolated install-record writer subprocess only")
	}
	writer := os.Getenv("VROOLI_INSTALL_RECORD_TEST_WRITER")
	if err := os.WriteFile(filepath.Join(home, "ready-"+writer), []byte("ready"), 0o600); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(8 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(home, "start-writers")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("writer fixture barrier timed out")
		}
		time.Sleep(5 * time.Millisecond)
	}
	for i := 0; i < 12; i++ {
		path := filepath.Join(home, "owned-"+writer+"-"+strconv.Itoa(i))
		if err := RecordInstallEntries(home, InstallEntry{Scope: ScopeRuntime, Kind: EntryFile, Path: path, Prefix: home}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInstallRecordMergesAcrossIndependentProcesses(t *testing.T) {
	_, home := newUninstallTestService(t)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	commands := make([]*exec.Cmd, 0, 2)
	for i := 0; i < 2; i++ {
		cmd := exec.CommandContext(ctx, executable, "-test.run=^TestInstallRecordWriterProcessFixture$")
		cmd.Env = append(os.Environ(), "VROOLI_INSTALL_RECORD_TEST_HOME="+home, "VROOLI_INSTALL_RECORD_TEST_WRITER="+strconv.Itoa(i), "GOMAXPROCS=2")
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		commands = append(commands, cmd)
		t.Cleanup(func() {
			if cmd.ProcessState == nil {
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
			}
		})
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		ready := 0
		for i := 0; i < 2; i++ {
			if _, err := os.Stat(filepath.Join(home, "ready-"+strconv.Itoa(i))); err == nil {
				ready++
			}
		}
		if ready == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("independent writers did not reach barrier")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err := os.WriteFile(filepath.Join(home, "start-writers"), []byte("start"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, cmd := range commands {
		if err := cmd.Wait(); err != nil {
			t.Fatalf("independent writer failed: %v", err)
		}
	}
	record, err := LoadInstallRecord(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(record.Entries) != 24 {
		t.Fatalf("independent processes lost installed ownership: got %d entries, want24", len(record.Entries))
	}
}
