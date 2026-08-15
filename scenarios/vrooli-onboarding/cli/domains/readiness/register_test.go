package readiness

import (
	"net/http"
	"testing"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

func TestExitErrorCarriesMachineCode(t *testing.T) {
	err := &ExitError{Code: 2, Text: "required item is not ready"}
	if err.Error() != "required item is not ready" || err.ExitCode() != 2 {
		t.Fatalf("exit error = %q/%d", err.Error(), err.ExitCode())
	}
}

func TestRunAcceptsOptionalDegradation(t *testing.T) {
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"degraded","credentials":[{"required":false,"status":"missing"}],"hosts":[]}`))
	}))
	if err := run(core, []string{"--json"}); err != nil {
		t.Fatal(err)
	}
}

func TestRunReturnsMachineReadableRequiredFailure(t *testing.T) {
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"missing","credentials":[{"required":true,"status":"missing"}],"hosts":[]}`))
	}))
	err := run(core, []string{"--json"})
	exit, ok := err.(*ExitError)
	if !ok || exit.Code != 2 {
		t.Fatalf("readiness error = %#v", err)
	}
}

func TestRunReturnsFailureForRequiredHostAndUnsupportedStatus(t *testing.T) {
	for _, body := range []string{
		`{"status":"degraded","credentials":[],"hosts":[{"required":true,"status":"missing"}]}`,
		`{"status":"unsupported","credentials":[],"hosts":[]}`,
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

func TestRegisterExposesReadinessCommand(t *testing.T) {
	group := Register(nil)
	if len(group.Commands) != 1 || group.Commands[0].Name != "readiness" || !group.Commands[0].NeedsAPI {
		t.Fatalf("readiness registration = %+v", group)
	}
}
