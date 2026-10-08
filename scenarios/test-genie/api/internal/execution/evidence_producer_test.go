package execution

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"test-genie/internal/orchestrator"
	sharedartifacts "test-genie/internal/shared/artifacts"
)

func TestEvidenceProducerCapsLogsAndMarksTruncation(t *testing.T) {
	if _, err := exec.LookPath("bwrap"); err != nil {
		t.Skip("bubblewrap unavailable")
	}
	if err := preflightBubblewrap(context.Background()); err != nil {
		t.Skipf("PID namespace unsupported here: %v", err)
	}
	root, artifacts := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	service := NewSuiteExecutionService(nil, &serviceRecorder{})
	service.engine = nil
	result, err := service.runEvidenceProducer(context.Background(), SuiteExecutionInput{
		Request: orchestratorRequest("bounded-run"), ArtifactRoot: artifacts,
		EvidenceProducer: &EvidenceProducerCommand{Provider: "fixture", Name: "bounded", Argv: []string{"/usr/bin/printf", "0123456789abcdef"}, WorkingDirectory: filepath.Join(root, "work"), Timeout: time.Second, MaximumOutputBytes: 5, DescriptorDigest: "descriptor", SourceIdentity: "source"},
	})
	if err != nil {
		entries, _ := filepath.Glob(filepath.Join(artifacts, "*", "evidence-producer", "output.log"))
		var output []byte
		if len(entries) > 0 {
			output, _ = os.ReadFile(entries[0])
		}
		t.Fatalf("%v; child output: %s", err, output)
	}
	if result.Verdict != "PRODUCER_COMPLETED" || len(result.PlannedPhases) != 0 || len(result.Phases) != 0 {
		t.Fatalf("producer result implies suite validation: %+v", result)
	}
	logPath := filepath.Join(sharedRunDir(artifacts, "bounded-run"), "evidence-producer", "output.log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "01234" {
		t.Fatalf("bounded log = %q", content)
	}
	metadata, err := os.ReadFile(filepath.Join(filepath.Dir(logPath), "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(metadata), `"truncated":true`) || !strings.Contains(string(metadata), `"bytesWritten":5`) {
		t.Fatalf("truncation metadata = %s", metadata)
	}
}

func TestEvidenceProducerLargeFinalLogIsComplete(t *testing.T) {
	if err := preflightBubblewrap(context.Background()); err != nil {
		t.Skipf("PID namespace unsupported here: %v", err)
	}
	work, artifacts := t.TempDir(), t.TempDir()
	service := NewSuiteExecutionService(nil, &serviceRecorder{})
	service.engine = nil
	_, err := service.runEvidenceProducer(context.Background(), SuiteExecutionInput{Request: orchestratorRequest("large-log"), ArtifactRoot: artifacts, EvidenceProducer: &EvidenceProducerCommand{Provider: "fixture", Name: "large-log", Argv: []string{"/bin/sh", "-c", "head -c 900000 /dev/zero | tr '\\000' x"}, WorkingDirectory: work, Timeout: 10 * time.Second, MaximumOutputBytes: 1 << 20, DescriptorDigest: "descriptor", SourceIdentity: "source"}})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(sharedRunDir(artifacts, "large-log"), "evidence-producer", "output.log"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 900000 {
		t.Fatalf("final log length=%d, want 900000", len(data))
	}
	if !bytes.Equal(data, bytes.Repeat([]byte{'x'}, 900000)) {
		t.Fatal("large final log was incomplete or corrupted")
	}
}

func TestEvidenceProducerReadinessFailureDrainsAndReaps(t *testing.T) {
	if err := preflightBubblewrap(context.Background()); err != nil {
		t.Skipf("PID namespace unsupported here: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	missing := filepath.Join(t.TempDir(), "missing")
	cmd, ready, _, finish, closePipes, stdoutDone, err := containedProducerCommand(ctx, missing, 1024, []string{"/bin/true"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	gate := <-ready
	if gate.err == nil {
		t.Fatal("invalid work directory passed readiness")
	}
	finish()
	if copyErr := <-stdoutDone; !errors.Is(copyErr, io.EOF) {
		t.Fatalf("readiness failure did not retain EOF: %v", copyErr)
	}
	waitErr := cmd.Wait()
	closePipes()
	if waitErr == nil {
		t.Fatal("readiness failure child was not reaped with failure")
	}
}

func TestEvidenceProducerOfflineNodeToGoOwnerCommand(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Fatalf("Node runtime unavailable: %v", err)
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Fatalf("Go runtime unavailable: %v", err)
	}
	work := t.TempDir()
	artifacts := t.TempDir()
	main := `package main
import("os";"path/filepath")
func main(){ if err:=os.WriteFile(filepath.Join(os.Args[1],"owner-result.txt"),[]byte("node-to-go-offline"),0600);err!=nil{panic(err)} }
`
	if err := os.WriteFile(filepath.Join(work, "main.go"), []byte(main), 0600); err != nil {
		t.Fatal(err)
	}
	nodeScript := `const {execFileSync}=require('node:child_process'); execFileSync('go',['run','main.go',process.argv[2]],{stdio:'inherit',env:{...process.env,GOPROXY:'off',GOTOOLCHAIN:'local'}});`
	if err := os.WriteFile(filepath.Join(work, "owner.js"), []byte(nodeScript), 0600); err != nil {
		t.Fatal(err)
	}
	node, _ := exec.LookPath("node")
	service := NewSuiteExecutionService(nil, &serviceRecorder{})
	service.engine = nil
	_, err := service.runEvidenceProducer(context.Background(), SuiteExecutionInput{Request: orchestratorRequest("node-go-owner"), ArtifactRoot: artifacts, EvidenceProducer: &EvidenceProducerCommand{Provider: "fixture", Name: "node-go", Argv: []string{node, "owner.js", "/dev/shm/tg-output"}, WorkingDirectory: work, Timeout: 2 * time.Minute, MaximumOutputBytes: 1 << 20, DescriptorDigest: "descriptor", SourceIdentity: "source"}})
	if err != nil {
		t.Fatalf("offline Node->Go producer: %v", err)
	}
	result, err := os.ReadFile(filepath.Join(sharedRunDir(artifacts, "node-go-owner"), "evidence-producer", "output", "owner-result.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(result) != "node-to-go-offline" {
		t.Fatalf("retained output=%q", result)
	}
}

func TestEvidenceProducerRejectsOutputByteEntryAndLinkOverflow(t *testing.T) {
	for _, tc := range []struct {
		name, script string
		limit        int64
	}{
		{name: "bytes", script: `head -c 2048 /dev/zero > "$1/too-large"`, limit: 1024},
		{name: "entries", script: `i=0; while [ "$i" -lt 4097 ]; do : > "$1/$i"; i=$((i+1)); done`, limit: 1 << 20},
		{name: "link", script: `ln -s /etc/passwd "$1/escape"`, limit: 1024},
	} {
		t.Run(tc.name, func(t *testing.T) {
			work, artifacts := t.TempDir(), t.TempDir()
			service := NewSuiteExecutionService(nil, &serviceRecorder{})
			service.engine = nil
			_, err := service.runEvidenceProducer(context.Background(), SuiteExecutionInput{Request: orchestratorRequest("output-" + tc.name), ArtifactRoot: artifacts, EvidenceProducer: &EvidenceProducerCommand{Provider: "fixture", Name: tc.name, Argv: []string{"/bin/sh", "-c", tc.script, "owner", "/dev/shm/tg-output"}, WorkingDirectory: work, Timeout: 10 * time.Second, MaximumOutputBytes: tc.limit, DescriptorDigest: "descriptor", SourceIdentity: "source"}})
			if err == nil {
				t.Fatal("unsafe producer output was accepted")
			}
			if _, statErr := os.Stat(filepath.Join(sharedRunDir(artifacts, "output-"+tc.name), "evidence-producer", "output")); !os.IsNotExist(statErr) {
				t.Fatalf("partial output was retained: %v", statErr)
			}
		})
	}
}

func TestEvidenceProducerTimeoutKillsDescendants(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	cmd, ready, release, finish, _, stdoutDone, err := containedProducerCommand(ctx, "/", 1024, []string{"/bin/sh", "-c", "sleep 60 & wait"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if gate := <-ready; gate.err != nil {
		t.Fatal(gate.err)
	}
	if err = release(); err != nil {
		t.Fatal(err)
	}
	if !waitForDescendant(cmd.Process.Pid, time.Second) {
		cancel()
		_ = cmd.Wait()
		<-stdoutDone
		t.Fatal("timed out producer descendant did not start")
	}
	tracked := processTree(cmd.Process.Pid)
	<-ctx.Done()
	cancel()
	finish()
	if copyErr := <-stdoutDone; copyErr != nil {
		t.Fatal(copyErr)
	}
	if err = cmd.Wait(); err == nil {
		t.Fatal("timed out producer succeeded")
	}
	if !waitForTrackedExit(tracked, 3*time.Second) {
		t.Fatalf("timed out producer left processes alive: %v", tracked)
	}
}

func TestEvidenceProducerExitedLeaderKillsDescendant(t *testing.T) {
	if err := preflightBubblewrap(context.Background()); err != nil {
		t.Fatalf("containment unavailable: %v", err)
	}
	output := &bytes.Buffer{}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd, ready, release, finish, _, stdoutDone, err := containedProducerCommand(ctx, "/tmp", 1024, []string{"/bin/sh", "-c", "sleep 60 & sleep 0.5; exit 0"}, output)
	if err != nil {
		t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if gate := <-ready; gate.err != nil {
		t.Fatal(gate.err)
	}
	if err = release(); err != nil {
		t.Fatal(err)
	}
	grandchildPID := 0
	var tracked []int
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		tracked = processTree(cmd.Process.Pid)
		for _, pid := range tracked {
			data, _ := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
			args := bytes.Split(data, []byte{0})
			if len(args) >= 2 && filepath.Base(string(args[0])) == "sleep" && string(args[1]) == "60" {
				grandchildPID = pid
				break
			}
		}
		if grandchildPID != 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if grandchildPID == 0 {
		t.Fatal("descendant process was not observable")
	}
	finish()
	if copyErr := <-stdoutDone; copyErr != nil {
		t.Fatal(copyErr)
	}
	err = cmd.Wait()
	if err != nil {
		t.Fatalf("leader command failed: %v", err)
	}
	if !waitForTrackedExit(tracked, time.Second) {
		t.Fatalf("descendant %d or its process tree survived exited command leader: %v", grandchildPID, tracked)
	}
}

func TestUnsupportedContainmentRefusesBeforeProducerEffects(t *testing.T) {
	commandPath := t.TempDir()
	artifactRoot := t.TempDir()
	outputRoot := filepath.Join(t.TempDir(), "producer-output")
	t.Setenv("PATH", commandPath)
	service := NewSuiteExecutionService(nil, &serviceRecorder{})
	_, err := service.runEvidenceProducer(context.Background(), SuiteExecutionInput{Request: orchestratorRequest("unsupported"), ArtifactRoot: artifactRoot, EvidenceProducer: &EvidenceProducerCommand{Provider: "fixture", Name: "unsupported", Argv: []string{"/bin/touch", filepath.Join(outputRoot, "effect")}, WorkingDirectory: "/", OutputRoot: outputRoot, Timeout: time.Second, MaximumOutputBytes: 64, DescriptorDigest: "descriptor", SourceIdentity: "source"}})
	if err == nil {
		t.Fatal("unsupported containment ran producer")
	}
	if _, statErr := os.Stat(outputRoot); !os.IsNotExist(statErr) {
		t.Fatalf("unsupported containment created provider output: %v", statErr)
	}
	entries, readErr := os.ReadDir(artifactRoot)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("unsupported containment published artifacts before refusal: %v", entries)
	}
}

// TestEvidenceProducerOwnerDeathHelper is run as a subprocess by the owner
// death test. The outer process kills this owner after its contained tree exists.
func TestEvidenceProducerOwnerDeathHelper(t *testing.T) {
	marker := os.Getenv("TG_BWRAP_OWNER_MARKER")
	if marker == "" {
		return
	}
	cmd, ready, release, finish, _, _, err := containedProducerCommand(context.Background(), "/", 1024, []string{"/bin/sh", "-c", "sleep 60 & wait"}, io.Discard)
	if err != nil {
		os.Exit(20)
	}
	if err := cmd.Start(); err != nil {
		os.Exit(21)
	}
	if gate := <-ready; gate.err != nil {
		os.Exit(22)
	}
	if err := release(); err != nil {
		os.Exit(23)
	}
	finish()
	_ = os.WriteFile(marker, []byte(strconv.Itoa(cmd.Process.Pid)), 0o600)
	select {}
}

func TestEvidenceProducerCancellationAndOwnerDeathKillWholeTree(t *testing.T) {
	if _, err := exec.LookPath("bwrap"); err != nil {
		t.Skip("bubblewrap unavailable")
	}
	if err := preflightBubblewrap(context.Background()); err != nil {
		t.Skipf("PID namespace unsupported here: %v", err)
	}
	for _, mode := range []string{"cancel", "owner-death"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cmd, ready, release, finish, closePipes, stdoutDone, err := containedProducerCommand(ctx, "/", 1024, []string{"/bin/sh", "-c", "sleep 60 & wait"}, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			if gate := <-ready; gate.err != nil {
				t.Fatal(gate.err)
			}
			if err := release(); err != nil {
				t.Fatal(err)
			}
			finish()
			if !waitForDescendant(cmd.Process.Pid, time.Second) {
				cancel()
				_ = cmd.Wait()
				closePipes()
				<-stdoutDone
				t.Fatal("contained descendant did not start")
			}
			pid := cmd.Process.Pid
			tracked := processTree(pid)
			if mode == "cancel" {
				cancel()
				_ = cmd.Wait()
				<-stdoutDone
			} else {
				cancel()
				_ = cmd.Wait()
				marker := filepath.Join(t.TempDir(), "owner.pid")
				helper := exec.Command(os.Args[0], "-test.run=^TestEvidenceProducerOwnerDeathHelper$")
				helper.Env = append(os.Environ(), "TG_BWRAP_OWNER_MARKER="+marker)
				if err := helper.Start(); err != nil {
					t.Fatal(err)
				}
				if !waitForFile(marker, time.Second) {
					_ = helper.Process.Kill()
					_ = helper.Wait()
					t.Fatal("owner helper did not start bubblewrap")
				}
				bwrapPID, _ := os.ReadFile(marker)
				pid, _ = strconv.Atoi(string(bwrapPID))
				if !waitForDescendant(pid, time.Second) {
					_ = helper.Process.Kill()
					_ = helper.Wait()
					t.Fatal("owner descendant did not start")
				}
				tracked = processTree(pid)
				_ = helper.Process.Kill()
				_ = helper.Wait()
			}
			if !waitForTrackedExit(tracked, 3*time.Second) {
				t.Fatalf("contained process tree rooted at %d survived %s: %v", pid, mode, tracked)
			}
		})
	}
}

func waitForDescendant(root int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if descendants(root) > 1 {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func waitForTreeExit(root int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !treeAlive(root) {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return !treeAlive(root)
}

func waitForTrackedExit(pids []int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		alive := false
		for _, pid := range pids {
			if processAlive(pid) {
				alive = true
				break
			}
		}
		if !alive {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	for _, pid := range pids {
		if processAlive(pid) {
			return false
		}
	}
	return true
}

func waitForFile(path string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func treeAlive(root int) bool {
	if _, err := os.Stat(fmt.Sprintf("/proc/%d", root)); err == nil {
		return true
	}
	return false
}

func processAlive(pid int) bool {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return false
	}
	closeParen := strings.LastIndex(string(data), ")")
	if closeParen < 0 {
		return true
	}
	fields := strings.Fields(string(data)[closeParen+1:])
	return len(fields) == 0 || fields[0] != "Z"
}

func processTree(root int) []int {
	entries, _ := os.ReadDir("/proc")
	type proc struct{ pid, parent int }
	var processes []proc
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		data, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "stat"))
		if err != nil {
			continue
		}
		closeParen := strings.LastIndex(string(data), ")")
		if closeParen < 0 {
			continue
		}
		fields := strings.Fields(string(data)[closeParen+1:])
		if len(fields) < 2 {
			continue
		}
		parent, _ := strconv.Atoi(fields[1])
		processes = append(processes, proc{pid: pid, parent: parent})
	}
	seen := map[int]bool{root: true}
	result := []int{root}
	for changed := true; changed; {
		changed = false
		for _, process := range processes {
			if seen[process.parent] && !seen[process.pid] {
				seen[process.pid] = true
				result = append(result, process.pid)
				changed = true
			}
		}
	}
	return result
}

func descendants(root int) int {
	return len(processTree(root)) - 1
}

func orchestratorRequest(runID string) orchestrator.SuiteExecutionRequest {
	return orchestrator.SuiteExecutionRequest{ScenarioName: "fixture", RunID: runID, RequestedAt: time.Now().UTC()}
}

func sharedRunDir(root, runID string) string { return sharedartifacts.RunDir(root, runID) }
