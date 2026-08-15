package packagegov

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRunCommandsWithOptionsUsesExplicitEnvironmentAndStdin(t *testing.T) {
	var output bytes.Buffer
	err := RunCommandsWithOptions(t.TempDir(), []CommandSpec{{
		Name: "probe",
		Run:  []string{"/bin/sh", "-c", "printf 'env=%s\\n' \"$VROOLI_TEST_VALUE\"; if read -r value; then printf 'stdin=value\\n'; else printf 'stdin=eof\\n'; fi"},
	}}, &output, &output, CommandOptions{
		Env:   []string{"VROOLI_TEST_VALUE=explicit", "PATH=" + os.Getenv("PATH")},
		Stdin: strings.NewReader(""),
	})
	if err != nil {
		t.Fatalf("RunCommandsWithOptions: %v", err)
	}
	if got := output.String(); !strings.Contains(got, "env=explicit") || !strings.Contains(got, "stdin=eof") {
		t.Fatalf("output = %q, want explicit env and EOF stdin", got)
	}
}
