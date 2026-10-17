package lifecycle

import (
	"io"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/vrooli/vrooli/internal/process"
)

// deadConsole is a terminal whose reader has gone: every write fails.
type deadConsole struct{ writes int }

func (c *deadConsole) Write([]byte) (int, error) {
	c.writes++
	return 0, syscall.EPIPE
}

// The lifecycle log is the record of a run, and a step's output must keep
// draining: a console that stops accepting writes costs neither.
func TestLifecycleLogKeepsEveryLineWhenTheConsoleGoesAway(t *testing.T) {
	home := t.TempDir()
	console := &deadConsole{}
	// Verbose, so both the orchestrator and the child writers reach the console.
	r := &Runner{Home: home, Out: console, Err: io.Discard, Verbosity: VerbosityVerbose}

	var writeErrs []error
	_, err := r.runWithLifecycleLog(lifecycleLogContext{Scenario: "console-probe", Operation: "restart", Phase: "setup"}, func(logWriter, childWriter io.Writer) error {
		for _, w := range []io.Writer{logWriter, childWriter, logWriter} {
			if _, werr := io.WriteString(w, "a step line\n"); werr != nil {
				writeErrs = append(writeErrs, werr)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(writeErrs) != 0 {
		t.Fatalf("a dead console failed the step's writes: %v", writeErrs)
	}

	path, err := process.ScenarioLifecycleLogPath(home, "console-probe")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(data), "a step line\n"); got != 3 {
		t.Fatalf("lifecycle log holds %d of 3 step lines:\n%s", got, data)
	}
	if console.writes != 1 {
		t.Fatalf("the dead console was written %d times; after its first failure it is left alone", console.writes)
	}
}
