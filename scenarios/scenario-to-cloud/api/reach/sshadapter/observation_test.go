package sshadapter

import (
	"context"
	"strings"
	"testing"

	"scenario-to-cloud/reach"
)

// [REQ:STC-P0-024] A host observation program runs as its quoted argv
// outside the bound workdir and without the vrooli path: the transport's
// one quoting rule applies to every argument, so a validated argument can
// never become shell syntax; the vrooli prefix is absent because the
// program is not a target verb.
func TestObservationProgramRendersQuotedArgvWithoutVrooliPrefix(t *testing.T) {
	runner := &fakeRunner{result: Result{ExitCode: 0, Stdout: "Filesystem 1024-blocks Used Available Capacity Mounted on\n/dev/vda1 100 50 50 50% /\n"}}
	a := adapter(runner)
	res, err := a.Exec(context.Background(), target(), reach.Command{Program: "df", Args: []string{"-Pk", "/"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 0 || !strings.Contains(res.Stdout, "/dev/vda1") {
		t.Fatalf("result = %+v", res)
	}
	cmd := runner.commands[0]
	if cmd != "'df' '-Pk' '/'" {
		t.Fatalf("remote observation = %q", cmd)
	}
	if strings.Contains(cmd, "vrooli") || strings.Contains(cmd, "cd ") {
		t.Fatalf("observation must not run through the workdir vrooli binary: %q", cmd)
	}
	if _, err := a.Exec(context.Background(), target(), reach.Command{Program: "sh", Args: []string{"-c", "id"}}); !reach.IsKind(err, reach.KindInvalidArgument) {
		t.Fatalf("non-observation program must be refused before transport work, got %v (commands=%v)", err, runner.commands)
	}
	if len(runner.commands) != 1 {
		t.Fatalf("refused program reached the transport: %v", runner.commands)
	}
}
