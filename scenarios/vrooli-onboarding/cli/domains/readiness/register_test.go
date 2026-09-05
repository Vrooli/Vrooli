package readiness

import (
	"net/http"
	"strings"
	"testing"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

func TestExitErrorCarriesMachineCode(t *testing.T) {
	err := &ExitError{Code: 2, Text: "required item is not ready"}
	if err.Error() != "required item is not ready" || err.ExitCode() != 2 {
		t.Fatalf("exit error = %q/%d", err.Error(), err.ExitCode())
	}
}

// The API owns the completion rule and reports it as typed blockers; the CLI's
// job is to carry that verdict into an exit code a script can act on.
func TestRunAcceptsAnAcknowledgedDegradedSet(t *testing.T) {
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"degraded","blockers":[],"degraded":[{"kind":"credential","name":"vrooli/remote-desktop:username","reason":"the credential is declared and not configured","remediation":"Provide it on the credentials step."}],"degraded_digest":"abc","degraded_acknowledged":true}`))
	}))
	if err := run(core, []string{"--json"}); err != nil {
		t.Fatal(err)
	}
}

func TestRunAcceptsACleanVerdict(t *testing.T) {
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ready","blockers":[],"degraded":[]}`))
	}))
	if err := run(core, []string{"--json"}); err != nil {
		t.Fatal(err)
	}
}

func TestRunReturnsMachineReadableRequiredFailure(t *testing.T) {
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"missing","blockers":[{"kind":"credential","name":"vrooli/calendar:jwt-secret","reason":"the credential is declared and not configured","remediation":"Provide it on the credentials step."}],"degraded":[]}`))
	}))
	err := run(core, []string{"--json"})
	exit, ok := err.(*ExitError)
	if !ok || exit.Code != 2 {
		t.Fatalf("readiness error = %#v", err)
	}
	for _, want := range []string{"vrooli/calendar:jwt-secret", "the credential is declared and not configured", "Provide it on the credentials step."} {
		if !strings.Contains(exit.Error(), want) {
			t.Fatalf("readiness error = %q, want %q", exit.Error(), want)
		}
	}
}

func TestRunReturnsFailureForRequiredHostAndUnacknowledgedDegradation(t *testing.T) {
	for _, body := range []string{
		`{"status":"missing","blockers":[{"kind":"host","name":"workspace_sandbox_userns","reason":"safeguard is missing on this host","remediation":"Apply the selection."}],"degraded":[]}`,
		`{"status":"degraded","blockers":[],"degraded":[{"kind":"host","name":"docker","reason":"tool is missing on this host","remediation":"Apply the selection."}],"degraded_digest":"abc","degraded_acknowledged":false}`,
	} {
		core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(body))
		}))
		err := run(core, []string{"--json"})
		if exit, ok := err.(*ExitError); !ok || exit.Code != 2 {
			t.Fatalf("body %s returned %#v", body, err)
		}
	}
}

// A deferred optional item is not a gap: it is an optional capability the
// operator did not select. It must not block or need an acknowledgement.
func TestRunIgnoresDeferredOptionalItems(t *testing.T) {
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ready","hosts":[{"name":"clock","kind":"safeguard","status":"deferred","required":false}],"blockers":[],"degraded":[]}`))
	}))
	if err := run(core, []string{"--json"}); err != nil {
		t.Fatal(err)
	}
}

func TestAcknowledgeDegradedReadsTheCurrentDigestWhenNoneIsGiven(t *testing.T) {
	var posted string
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			body := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(body)
			posted = string(body)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"status":"acknowledged"}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"degraded","blockers":[],"degraded":[{"kind":"credential","name":"a:b"}],"degraded_digest":"digest-under-test"}`))
	}))
	if err := acknowledgeDegraded(core, []string{"--json"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(posted, "digest-under-test") {
		t.Fatalf("posted body = %q, want the current degraded digest", posted)
	}
}

func TestAcknowledgeDegradedRefusesWhenThereIsNothingToAcknowledge(t *testing.T) {
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ready","blockers":[],"degraded":[]}`))
	}))
	err := acknowledgeDegraded(core, []string{"--json"})
	if exit, ok := err.(*ExitError); !ok || exit.Code != 2 {
		t.Fatalf("acknowledge with no degraded set returned %#v", err)
	}
}

func TestRegisterExposesReadinessCommand(t *testing.T) {
	group := Register(nil)
	if len(group.Commands) != 1 || group.Commands[0].Name != "readiness" || !group.Commands[0].NeedsAPI {
		t.Fatalf("readiness registration = %+v", group)
	}
}
