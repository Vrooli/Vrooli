//go:build darwin && cgo

package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// This explicit target qualification exercises the real libproc SDK path and
// the global PCB API under the invoking native account. Linux fixtures and
// no-cgo cross-builds cannot substitute for this result.
func TestDarwinNativeProcessObservationLive(t *testing.T) {
	if os.Getenv("VROOLI_PLATFORM_NATIVE_LIVE") != "1" {
		t.Skip("set VROOLI_PLATFORM_NATIVE_LIVE=1 on the native qualification target")
	}
	image, err := OpenProcessExecutable(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	defer image.Close()
	actual, err := image.Stat()
	if err != nil {
		t.Fatal(err)
	}
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(actual, expected) {
		t.Fatal("kernel executable mapping does not match native test image")
	}
	cwd, err := processWorkingDir(os.Getpid())
	if err != nil || cwd == "" {
		t.Fatalf("cwd: %q %v", cwd, err)
	}
	security, err := ProcessSecurity(os.Getpid())
	if err != nil || security.EffectiveUID != os.Geteuid() {
		t.Fatalf("native security: %v %v", security, err)
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	_, portString, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portString)
	if err != nil {
		t.Fatal(err)
	}
	check := func(name string, listeners []ProcessTCPListener, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, entry := range listeners {
			if entry.PID == os.Getpid() && entry.Port == uint16(port) && entry.Address == "127.0.0.1" {
				return
			}
		}
		t.Fatalf("%s omitted known native self listener: %v", name, listeners)
	}
	own, err := ProcessTCPListeners(os.Getpid())
	check("process descriptor table", own, err)
	global, err := TCPListeners(uint16(port))
	check("global kernel PCB snapshot", global, err)
}

// Birth identity must survive intentional same-PID image replacement while the
// kernel execution version changes. This is a real target ABI gate, not a fixture.
func TestDarwinNativeExecVersionLive(t *testing.T) {
	if os.Getenv("VROOLI_PLATFORM_NATIVE_LIVE") != "1" {
		t.Skip("native opt-in qualification required")
	}
	const stepKey = "VROOLI_PLATFORM_DARWIN_EXEC_STAMP_STEP"
	step := os.Getenv(stepKey)
	if step != "" {
		stamp, err := darwinObservationStamp(os.Getpid())
		if err != nil {
			t.Fatal(err)
		}
		if err = json.NewEncoder(os.Stdout).Encode(stamp); err != nil {
			t.Fatal(err)
		}
		if step == "before" {
			env := []string{}
			for _, value := range os.Environ() {
				if !strings.HasPrefix(value, stepKey+"=") {
					env = append(env, value)
				}
			}
			env = append(env, stepKey+"=after")
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			if err = syscall.Exec(executable, []string{executable, "-test.run=^TestDarwinNativeExecVersionLive$"}, env); err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, executable, "-test.run=^TestDarwinNativeExecVersionLive$")
	command.Env = append(os.Environ(), stepKey+"=before")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("native exec helper: %v: %s", err, output)
	}
	var stamps [][4]uint64
	for _, line := range bytes.Split(output, []byte{'\n'}) {
		if len(line) > 0 && line[0] == '[' {
			var stamp [4]uint64
			if err = json.Unmarshal(line, &stamp); err != nil {
				t.Fatal(err)
			}
			stamps = append(stamps, stamp)
		}
	}
	if len(stamps) != 2 {
		t.Fatalf("expected two native exec samples: %s", output)
	}
	before, after := stamps[0], stamps[1]
	if before[0] != after[0] || before[1] != after[1] || before[2] != after[2] || before[2] == 0 || before[3] == after[3] || after[3] == 0 {
		t.Fatalf("native birth/exec ABI failed: %v -> %v", before, after)
	}
}
