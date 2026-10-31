//go:build unix

package metrics

import (
	"os"
	"os/exec"
	"strconv"
	"testing"

	"github.com/vrooli/repo-contract-go/repocontracttest"
)

// TestPeakRSSIncludesChildProcesses is the regression test for a measurement
// defect, not a code-path defect: peak RSS used to be read from RUSAGE_SELF
// alone while CPU summed self and children. A provider that shells out — go
// test, gosec, govulncheck — is a thin wrapper, so the reported figure described
// the wrapper rather than the work. Capacity admission sizes its RAM
// reservation from that figure, so it under-reserved for the heaviest phases.
//
// The test re-executes this binary as a child that allocates a known amount,
// then asserts the sample reflects the child rather than the parent.
func TestPeakRSSIncludesChildProcesses(t *testing.T) {
	if os.Getenv("METRICS_CHILD_ALLOC_MIB") != "" {
		allocateAndExit(t)
		return
	}
	if !sampleRusage().ok {
		repocontracttest.SkipPlatform(t, "rusage not sampleable on this platform")
	}

	before := sampleRusage()

	const allocMiB = 256
	cmd := exec.Command(os.Args[0], "-test.run=TestPeakRSSIncludesChildProcesses")
	cmd.Env = append(os.Environ(), "METRICS_CHILD_ALLOC_MIB="+strconv.Itoa(allocMiB))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("child process failed: %v\n%s", err, out)
	}

	// The child is reaped by cmd.Wait, so its high-water mark is now visible
	// through RUSAGE_CHILDREN.
	after := sampleRusage()
	if !after.ok {
		t.Fatal("rusage sample failed after running the child")
	}

	childBytes, ok := processPeakRSSBytes(cmd.ProcessState)
	if !ok {
		repocontracttest.SkipPlatform(t, "child process peak RSS unavailable on this platform")
	}

	if childBytes < allocMiB/2*1024*1024 {
		t.Skipf("the child's peak RSS (%d bytes) is too small to distinguish; "+
			"the runtime may have reclaimed the allocation", childBytes)
	}
	if after.maxRSSBytes < childBytes {
		t.Fatalf("peak RSS ignores child processes: sample reports %d bytes, "+
			"but a reaped child peaked at %d bytes (before the child: %d)",
			after.maxRSSBytes, childBytes, before.maxRSSBytes)
	}
}

func allocateAndExit(t *testing.T) {
	t.Helper()
	mib, err := strconv.Atoi(os.Getenv("METRICS_CHILD_ALLOC_MIB"))
	if err != nil || mib <= 0 {
		return
	}
	buf := make([]byte, mib*1024*1024)
	// Touch every page so the allocation is resident rather than merely mapped.
	for i := 0; i < len(buf); i += 4096 {
		buf[i] = byte(i)
	}
	// Keep the compiler from eliminating the allocation.
	if buf[len(buf)-1] == 255 && buf[0] == 255 {
		t.Log("unreachable")
	}
}
