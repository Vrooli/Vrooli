//nolint:goconst // test data deliberately reuses stable command fixtures.
package setup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/securestore"
)

type credentialStoreCall struct {
	ctx   context.Context
	args  []string
	input string
}

// stubCredentialStore records the commands setup runs against the credential
// store and lets a test decide what each one answers.
func stubCredentialStore(t *testing.T, reply func(call credentialStoreCall, stdout io.Writer) error) *[]credentialStoreCall {
	t.Helper()
	calls := &[]credentialStoreCall{}
	previousRun, previousExecutable := runCredentialStoreFn, credentialStoreExecutableFn
	credentialStoreExecutableFn = func() (string, error) { return "/usr/local/bin/vrooli", nil }
	runCredentialStoreFn = func(_ string, args []string, input string, opts hostreqkit.EnsureOptions) error {
		*calls = append(*calls, credentialStoreCall{ctx: opts.Context, args: append([]string(nil), args...), input: input})
		if reply == nil {
			return nil
		}
		return reply(credentialStoreCall{args: args, input: input}, opts.Stdout)
	}
	t.Cleanup(func() {
		runCredentialStoreFn, credentialStoreExecutableFn = previousRun, previousExecutable
	})
	return calls
}

func verbOf(call credentialStoreCall) string { return strings.Join(call.args, " ") }

func TestNativeCredentialDiagnosisUsesOperatorResultForElevatedSetup(t *testing.T) {
	previousDoctor, previousRoot := runCredentialDoctorFn, hostreqkit.RunningAsRootFn
	t.Cleanup(func() {
		runCredentialDoctorFn, hostreqkit.RunningAsRootFn = previousDoctor, previousRoot
	})
	hostreqkit.RunningAsRootFn = func() bool { return true }
	want := securestore.Diagnosis{Platform: "linux", Adapter: "secret-service", Condition: "available", WriteCondition: "ready"}
	runCredentialDoctorFn = func(context.Context, string) ([]byte, error) {
		payload, err := json.Marshal(struct {
			Provider securestore.Diagnosis `json:"provider"`
		}{Provider: want})
		return payload, err
	}

	if got := nativeCredentialDiagnosis(context.Background(), "/fixture/vrooli"); got != want {
		t.Fatalf("nativeCredentialDiagnosis() = %+v, want invoking-user result %+v", got, want)
	}
}

func TestNativeCredentialDiagnosisDoesNotProbeRootAfterOperatorSessionFailure(t *testing.T) {
	previousDoctor, previousRoot := runCredentialDoctorFn, hostreqkit.RunningAsRootFn
	t.Cleanup(func() {
		runCredentialDoctorFn, hostreqkit.RunningAsRootFn = previousDoctor, previousRoot
	})
	hostreqkit.RunningAsRootFn = func() bool { return true }
	calls := 0
	runCredentialDoctorFn = func(context.Context, string) ([]byte, error) {
		calls++
		return nil, fmt.Errorf("operator session unavailable")
	}

	got := nativeCredentialDiagnosis(context.Background(), "/fixture/vrooli")
	if calls != 1 {
		t.Fatalf("operator doctor calls = %d, want one", calls)
	}
	if got.Condition != "unavailable" || got.WriteCondition != "unavailable" || !strings.Contains(got.Explanation, "operator-session") {
		t.Fatalf("diagnosis = %+v, want unavailable without a root-account fallback", got)
	}
}

// The operator-supplied passphrase is the path almost every install actually
// takes, through setup's stdin flag or through onboarding. Returning from it
// without converging on an unattended wrap is what made this host ask for the
// same secret at every boot, so the convergence is asserted on that exact path.
func TestPassphrasePathConvergesOnUnattendedAccess(t *testing.T) {
	calls := stubCredentialStore(t, func(call credentialStoreCall, stdout io.Writer) error {
		if verbOf(call) == "credentials store rewrap --format json" {
			fmt.Fprintln(stdout, `{"enabled":true,"provider":"host-bound","key_store":"tpm2","added":true}`)
		}
		return nil
	})

	var out bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := initializeEncryptedBackendWithPassphrase(ctx, &out, "the operator passphrase", true); err != nil {
		t.Fatalf("initializeEncryptedBackendWithPassphrase: %v", err)
	}

	if len(*calls) != 2 {
		t.Fatalf("commands run = %v, want an unlock followed by a convergence", *calls)
	}
	if got := verbOf((*calls)[0]); got != "credentials store unlock" {
		t.Fatalf("first command = %q, want the unlock", got)
	}
	if got := verbOf((*calls)[1]); got != "credentials store rewrap --format json" {
		t.Fatalf("second command = %q, want the unattended convergence", got)
	}
	// The passphrase has to reach the convergence: on a host whose store was
	// locked, the wrap cannot be added without opening it first.
	if (*calls)[1].input != "the operator passphrase" {
		t.Fatalf("convergence input = %q, want the operator passphrase", (*calls)[1].input)
	}
	if (*calls)[0].ctx != ctx || (*calls)[1].ctx != ctx {
		t.Fatal("credential commands did not receive the setup operation context")
	}
	if !strings.Contains(out.String(), "Unattended credential access: enabled") {
		t.Fatalf("setup output = %q, want it to report that this host no longer needs a passphrase", out.String())
	}
}

// A host with no TPM and no native wrap is fully working through its
// passphrase. Failing setup over it would break installation on the hardware
// the passphrase wrap exists to serve — but silence is what let an operator
// believe a host was unattended until the reboot proved otherwise.
func TestBlockedUnattendedAccessIsReportedAndNotFatal(t *testing.T) {
	stubCredentialStore(t, func(call credentialStoreCall, stdout io.Writer) error {
		if verbOf(call) == "credentials store rewrap --format json" {
			fmt.Fprintln(stdout, `{"enabled":false,"blocked":"host-bound: systemd-creds is not installed"}`)
			return fmt.Errorf("exit status 1")
		}
		return nil
	})

	var out bytes.Buffer
	if err := initializeEncryptedBackendWithPassphrase(context.Background(), &out, "the operator passphrase", true); err != nil {
		t.Fatalf("a host that cannot hold an unattended wrap failed setup: %v", err)
	}
	rendered := out.String()
	if !strings.Contains(rendered, "needs its passphrase after every reboot") {
		t.Fatalf("setup output = %q, want the attended outcome stated plainly", rendered)
	}
	if !strings.Contains(rendered, "systemd-creds is not installed") {
		t.Fatalf("setup output = %q, want it to carry the reason", rendered)
	}
}

func TestCredentialSetupStopsBeforeLaunchingWhenContextIsCanceled(t *testing.T) {
	calls := stubCredentialStore(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := initializeEncryptedBackendWithPassphrase(ctx, io.Discard, "passphrase", true)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("initializeEncryptedBackendWithPassphrase error = %v, want cancellation", err)
	}
	if len(*calls) != 0 {
		t.Fatalf("credential commands = %v, want none after cancellation", *calls)
	}
}

// An already-unattended host must not be told anything changed, or a rerun of
// setup reads as a repair that did not happen.
func TestAlreadyUnattendedHostIsReportedAsUnchanged(t *testing.T) {
	stubCredentialStore(t, func(call credentialStoreCall, stdout io.Writer) error {
		if verbOf(call) == "credentials store rewrap --format json" {
			fmt.Fprintln(stdout, `{"enabled":true,"provider":"native-wrap","key_store":"keychain"}`)
		}
		return nil
	})

	var out bytes.Buffer
	if err := initializeEncryptedBackendWithPassphrase(context.Background(), &out, "passphrase", true); err != nil {
		t.Fatalf("initializeEncryptedBackendWithPassphrase: %v", err)
	}
	if !strings.Contains(out.String(), "already enabled via the native-wrap wrap") {
		t.Fatalf("setup output = %q, want an unchanged host reported as already enabled", out.String())
	}
}
