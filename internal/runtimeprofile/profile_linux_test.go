//go:build linux

package runtimeprofile

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/api-core/runtimediagnostics"
	"github.com/vrooli/vrooli/internal/process"
)

// This is a test-process fixture, not a scenario launch. It exercises the same
// initial OS environment and process-incarnation boundary as native lifecycle.
func TestProfileProcessFixture(t *testing.T) {
	if os.Getenv("VROOLI_CPU_PROFILE_FIXTURE") != "1" {
		return
	}
	stop, err := runtimediagnostics.StartFromEnv(context.Background(), os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	fmt.Println("ready")
	_, _ = io.Copy(io.Discard, os.Stdin)
	stop()
	os.Exit(0)
}

func startProfileFixture(t *testing.T) (string, process.Record) {
	t.Helper()
	home, err := os.MkdirTemp("", "vrp-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(home) })
	directory, err := Directory(home)
	if err != nil {
		t.Fatal(err)
	}
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	command := exec.CommandContext(ctx, path, "-test.run=^TestProfileProcessFixture$")
	command.Env = append(os.Environ(), "VROOLI_CPU_PROFILE_FIXTURE=1", "VROOLI_SCENARIO=sample", "VROOLI_VARIANT=shadow", "VROOLI_RUNTIME_INSTANCE_ID=inst-fixture", "VROOLI_COMPONENT_GENERATION=fixture-generation", runtimediagnostics.DirectoryEnv+"="+directory)
	stdin, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		stdin.Close()
		if err := command.Wait(); err != nil {
			t.Errorf("profile fixture: %v: %s", err, stderr.String())
		}
	})
	scanner := bufio.NewScanner(stdout)
	if !scanner.Scan() || scanner.Text() != "ready" {
		t.Fatalf("fixture did not start: %v", scanner.Err())
	}
	incarnation, err := process.Incarnation(command.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	record := process.Record{PID: command.Process.Pid, Generation: "fixture-generation", Incarnation: incarnation, RuntimeInstanceID: "inst-fixture", Scenario: "sample", Step: "start-api", Status: "running"}
	if err := process.WriteScenarioRecord(home, "sample@shadow", "start-api", record); err != nil {
		t.Fatal(err)
	}
	return home, record
}

func TestCapturePinsLaunchAndPublishesPrivateProfile(t *testing.T) {
	home, record := startProfileFixture(t)
	output := filepath.Join(home, "sample.pprof")
	result, err := Capture(context.Background(), home, Request{Scenario: "sample@shadow", Step: "start-api", Seconds: 1, Output: output})
	if err != nil {
		t.Fatal(err)
	}
	if result.Identity.PID != record.PID || result.Identity.RuntimeInstanceID != record.RuntimeInstanceID || result.Identity.Generation != record.Generation || result.Bytes == 0 || len(result.SHA256) != 64 {
		t.Fatalf("result=%+v", result)
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("profile permissions=%v", info.Mode())
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if _, err := io.Copy(io.Discard, reader); err != nil {
		t.Fatal(err)
	}
	files, _ := filepath.Glob(filepath.Join(home, ".vrooli-cpu-*"))
	if len(files) != 0 {
		t.Fatalf("temporary profiles left behind: %v", files)
	}
}

func TestCaptureRefusesRecycledIdentityAndCancellationPublishesNothing(t *testing.T) {
	home, record := startProfileFixture(t)
	output := filepath.Join(home, "sample.pprof")
	request := Request{Scenario: "sample@shadow", Step: "start-api", Seconds: 30, Output: output}
	wrong := record
	wrong.Incarnation = "different"
	if err := process.WriteScenarioRecord(home, "sample@shadow", "start-api", wrong); err != nil {
		t.Fatal(err)
	}
	if _, err := Capture(context.Background(), home, request); err == nil {
		t.Fatal("recycled identity accepted")
	}
	if err := process.WriteScenarioRecord(home, "sample@shadow", "start-api", record); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := Capture(ctx, home, request); err == nil {
		t.Fatal("cancelled collection succeeded")
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		t.Fatalf("cancelled profile was published: %v", err)
	}
}
