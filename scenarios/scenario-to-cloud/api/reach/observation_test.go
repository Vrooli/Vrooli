package reach

import (
	"context"
	"testing"

	"scenario-to-cloud/identity"
)

// [REQ:STC-P0-024] A host observation program is a bounded read: it must
// come from the closed allowlist, can never be effectful, never carries a
// verb or stdin, and its arguments obey the same shell-syntax refusal as
// verbs. The allowlist itself holds only inspection tools.
func TestObservationProgramsAreBoundedReads(t *testing.T) {
	for program := range ObservationPrograms {
		switch program {
		case "cat", "df", "du", "find", "grep", "sudo", "journalctl", "ls", "pgrep", "ps", "ss", "stat", "uname", "head":
		default:
			t.Fatalf("observation allowlist admits %q, which is not an inspection tool", program)
		}
	}
	valid := Command{Program: "df", Args: []string{"-Pk", "/"}}
	if err := ValidateCommand(valid); err != nil {
		t.Fatalf("df -Pk / should validate: %v", err)
	}
	if got := valid.Argv(); len(got) != 3 || got[0] != "df" {
		t.Fatalf("observation argv = %v", got)
	}
	if err := ValidateCommand(Command{Program: "sudo", Args: []string{"-n", "-l"}}); err != nil {
		t.Fatalf("fixed sudo privilege observation should validate: %v", err)
	}
	refused := []Command{
		{Program: "bash", Args: []string{"-c", "true"}},
		{Program: "df", Effectful: true},
		{Program: "df", Verb: "scenario status"},
		{Program: "cat", Stdin: []byte("x")},
		{Program: "ss", Args: []string{"-ltnp; rm -rf /"}},
		{Program: "systemctl", Args: []string{"stop", "nginx"}},
		{Program: "kill", Args: []string{"1"}},
		{Program: "sudo", Args: []string{"-n", "bash"}},
	}
	for _, cmd := range refused {
		if err := ValidateCommand(cmd); !IsKind(err, KindInvalidArgument) {
			t.Fatalf("%+v should be refused as invalid_request, got %v", cmd, err)
		}
	}
}

type sessionless struct{ recording }

// [REQ:STC-P0-024] A transport without a session capability is a typed
// protocol refusal; the router never opens a session on another transport.
func TestRouterOpenSessionRefusesTransportsWithoutSessions(t *testing.T) {
	r := &Router{SSH: &sessionless{}}
	_, err := r.OpenSession(context.Background(), sshTarget(), SessionSpec{})
	if !IsKind(err, KindProtocolUnsupported) {
		t.Fatalf("expected reach_protocol_unsupported, got %v", err)
	}
	r.Revocation = revocation{revoked: true, reason: "grant withdrawn"}
	_, err = r.OpenSession(context.Background(), sshTarget(), SessionSpec{})
	if !IsKind(err, KindEnrollmentRevoked) {
		t.Fatalf("revoked target must refuse the session, got %v", err)
	}
	_, err = r.OpenSession(context.Background(), identity.TargetRef{}, SessionSpec{})
	if !IsKind(err, KindEnrollmentRevoked) && !IsKind(err, KindUnavailable) {
		t.Fatalf("unbound target must refuse the session, got %v", err)
	}
}
