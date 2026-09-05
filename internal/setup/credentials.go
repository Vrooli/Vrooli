package setup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/operatorcapability"
	"github.com/vrooli/vrooli/internal/operatorinput"
	"github.com/vrooli/vrooli/internal/resources/securestore"
)

var (
	credentialStoreExecutableFn = os.Executable
	runCredentialStoreFn        = runCredentialStoreAsOperator
	runCredentialDoctorFn       = runCredentialDoctorAsOperator
)

// runCredentialStoreAsOperator is the only production credential-store
// command path. The platform seam restores the invoking user's native session
// (launchd on macOS, the user bus on Linux) before touching a keyring.
func runCredentialStoreAsOperator(name string, args []string, input string, opts hostreqkit.EnsureOptions) error {
	return platform.RunAsInvokingUserInSessionWithInput(context.Background(), name, args, []byte(input), platform.IdentityCommandOptions{
		Stdout: opts.Stdout,
		Stderr: opts.Stderr,
	})
}

// ensureUnattendedCredentialAccess is the step that makes "supply the
// passphrase once" true rather than aspirational: it converges this host on
// opening its credential store after a reboot with no human, and says which
// outcome it reached.
//
// It reports and never fails. A host with no TPM and no native wrap is fully
// working through its passphrase, and failing setup over it would break
// installation on the hardware the passphrase wrap exists to serve. What is not
// acceptable is silence, which is what shipped before: the operator could not
// tell an unattended host from one that would stop at a prompt on the next
// reboot until the reboot happened.
func ensureUnattendedCredentialAccess(executable, passphrase string, stdout io.Writer) {
	var output bytes.Buffer
	runErr := runCredentialStoreFn(executable,
		[]string{"credentials", "store", "rewrap", "--format", "json"}, passphrase,
		commandOptions(&output, io.Discard))
	var status securestore.UnattendedStatus
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &status); err != nil {
		_, _ = fmt.Fprintf(stdout, "[WARN]    Unattended credential access: could not be evaluated on this host\n")
		if runErr != nil {
			_, _ = fmt.Fprintf(stdout, "[WARN]    Reason: %s\n", runErr.Error())
		}
		return
	}
	switch {
	case status.Enabled && status.Added:
		_, _ = fmt.Fprintf(stdout, "[INFO]    Unattended credential access: enabled via the %s wrap (%s); this host no longer needs a passphrase after a reboot\n",
			status.Provider, status.KeyStore)
	case status.Enabled && status.Repaired:
		_, _ = fmt.Fprintf(stdout, "[INFO]    Unattended credential access: repaired; the %s wrap had stopped opening and was replaced (%s)\n",
			status.Provider, status.KeyStore)
	case status.Enabled:
		_, _ = fmt.Fprintf(stdout, "[INFO]    Unattended credential access: already enabled via the %s wrap (%s)\n",
			status.Provider, status.KeyStore)
	default:
		_, _ = fmt.Fprintf(stdout, "[WARN]    Unattended credential access: not available; this host needs its passphrase after every reboot\n")
		if status.Blocked != "" {
			_, _ = fmt.Fprintf(stdout, "[WARN]    Reason: %s\n", status.Blocked)
		}
	}
}

func configureCredentialBackend(stdout, stderr io.Writer) error {
	return configureCredentialBackendWithPassphrase(stdout, stderr, "")
}

func configureCredentialBackendWithPassphrase(stdout, stderr io.Writer, passphrase string) error {
	executable, err := credentialStoreExecutableFn()
	if err != nil {
		return fmt.Errorf("resolve Vrooli executable for credential diagnosis: %w", err)
	}
	status, err := securestore.EnsureSetupBackendWithNativeDiagnosis(nativeCredentialDiagnosis(executable))
	if err != nil {
		return fmt.Errorf("configure credential backend: %w", err)
	}
	if status.Ready {
		_, _ = fmt.Fprintf(stdout, "[INFO]    Credential backend selected: %s (write-ready)\n", status.SelectedBackend)
		return nil
	}

	// The encrypted backend is Vrooli's authority on hosts without a usable
	// native store. Initialize it after setup has applied host safeguards: on a
	// TPM host this is fully unattended, while a host without an unattended
	// wrap gets one Vrooli-owned passphrase prompt instead of a second manual
	// repair workflow outside setup.
	if status.SelectedBackend == securestore.BackendEncryptedFile {
		if passphrase != "" {
			description, describeErr := securestore.DescribeStore()
			if describeErr != nil {
				return fmt.Errorf("inspect encrypted credential store: %w", describeErr)
			}
			return initializeEncryptedBackendWithPassphrase(stdout, passphrase, description.Initialized)
		}
		if err := initializeEncryptedBackend(stdout, stderr); err != nil {
			return err
		}
		// Initialization and any unlock happened inside the operator's user
		// session. Re-running the probe in a sudo'd parent would ask root's
		// keyring/TPM session instead and could report a false negative.
		_, _ = fmt.Fprintf(stdout, "[INFO]    Credential backend selected: %s (write-ready)\n", status.SelectedBackend)
		return nil
	}

	_, _ = fmt.Fprintf(stdout, "[WARN]    Credential backend selected: %s, but it is not ready yet.\n", status.SelectedBackend)
	if status.Explanation != "" {
		_, _ = fmt.Fprintf(stderr, "[WARN]    Credential backend: %s\n", status.Explanation)
	}
	if status.OperatorAction != "" {
		_, _ = fmt.Fprintf(stdout, "[WARN]    Credential next step: %s\n", status.OperatorAction)
	}
	return nil
}

func runCredentialDoctorAsOperator(executable string) ([]byte, error) {
	return platform.RunAsInvokingUserInSessionOutput(context.Background(), "env", []string{
		"VROOLI_CREDENTIAL_BACKEND=native", executable, "credentials", "doctor", "--check-writes", "--format", "json",
	}, platform.IdentityCommandOptions{Stderr: io.Discard})
}

func nativeCredentialDiagnosis(executable string) securestore.Diagnosis {
	output, err := runCredentialDoctorFn(executable)
	if err == nil {
		var response struct {
			Provider securestore.Diagnosis `json:"provider"`
		}
		if decodeErr := json.Unmarshal(output, &response); decodeErr == nil && response.Provider.Adapter != "" {
			return response.Provider
		}
	}
	// A direct diagnosis remains useful for non-elevated callers and for
	// platforms whose session runner is typed unsupported. Elevated setup must
	// not fall back to probing the root account: that is the original identity
	// bug this seam closes.
	if !hostreqkit.RunningAsRootFn() {
		return securestore.DiagnoseNativeWritable()
	}
	return securestore.Diagnosis{Platform: "unknown", Adapter: "native", Condition: "unavailable", WriteCondition: "unavailable", Explanation: "operator-session native probe failed; run onboarding in the invoking user's session"}
}

func initializeEncryptedBackendWithPassphrase(stdout io.Writer, passphrase string, initialized bool) error {
	executable, err := credentialStoreExecutableFn()
	if err != nil {
		return fmt.Errorf("resolve Vrooli executable for credential-store setup: %w", err)
	}
	command := []string{"credentials", "store", "init", "--format", "json"}
	if initialized {
		command = []string{"credentials", "store", "unlock"}
	}
	if err := runCredentialStoreFn(executable, command, passphrase, commandOptions(stdout, io.Discard)); err != nil {
		return fmt.Errorf("initialize or unlock encrypted credential store: %w", err)
	}
	_, _ = fmt.Fprintln(stdout, "[INFO]    Credential backend selected: encrypted-file (write-ready)")
	ensureUnattendedCredentialAccess(executable, passphrase, stdout)
	return nil
}

func initializeEncryptedBackend(stdout, stderr io.Writer) error {
	description, err := securestore.DescribeStore()
	if err != nil {
		return fmt.Errorf("inspect encrypted credential store: %w", err)
	}
	executable, err := credentialStoreExecutableFn()
	if err != nil {
		return fmt.Errorf("resolve Vrooli executable for credential-store setup: %w", err)
	}

	if !description.Initialized {
		// Try the unattended host-bound provider first, but do so as the
		// operator. Under sudo, root can use a TPM that the operator cannot yet
		// use, which would create a store that only root can open.
		if initErr := runCredentialStoreFn(executable,
			[]string{"credentials", "store", "init", "--format", "json"}, "",
			commandOptions(io.Discard, io.Discard)); initErr == nil {
			ensureUnattendedCredentialAccess(executable, "", stdout)
			return nil
		}
	} else {
		// A store can already exist but still need an unlock. Ask the actual
		// operator session first; a host-bound wrap may make this completely
		// unattended.
		var current securestore.StoreStatus
		var output bytes.Buffer
		if statusErr := runCredentialStoreFn(executable,
			[]string{"credentials", "store", "status", "--format", "json"}, "",
			commandOptions(&output, io.Discard)); statusErr == nil {
			if decodeErr := json.Unmarshal(output.Bytes(), &current); decodeErr == nil && current.Unlocked {
				ensureUnattendedCredentialAccess(executable, "", stdout)
				return nil
			}
		}
	}

	// Setup is Phase A and is deliberately non-interactive. The request is
	// resolved by vrooli-onboarding, which can render the same typed input in
	// its browser, CLI, or API surface. Keeping this producer here means a
	// headless `sudo vrooli setup` exits with an actionable pending state rather
	// than opening a hidden terminal prompt.
	if err := enqueueCredentialStoreInput(description.Initialized, stdout); err != nil {
		return err
	}
	return nil
}

func enqueueCredentialStoreInput(initialized bool, stdout io.Writer) error {
	request := operatorinput.Request{
		ID:              "credential-store-passphrase",
		Kind:            operatorinput.KindSecret,
		ContractVersion: operatorcapability.ContractVersion,
		Owner:           "vrooli.control-plane",
		CapabilityID:    "credential-store-access",
		ActionID:        "apply",
		InputID:         "passphrase",
		Title:           "Protect the encrypted credential store",
		Description:     "Choose one passphrase in vrooli-onboarding; it is never printed or placed in a command argument.",
		Unblocks:        []string{"credential-store-initialization", "unattended-credential-access"},
		Validation:      "non-empty",
		Required:        true,
	}
	if initialized {
		request.Description = "Enter the existing encrypted-store passphrase in vrooli-onboarding; it is never printed or placed in a command argument."
	}
	if err := operatorinput.Enqueue(request); err != nil {
		return fmt.Errorf("credential store needs operator input and queueing it failed: %w", err)
	}
	_, _ = fmt.Fprintln(stdout, "[PENDING] Credential-store protection will be completed by vrooli-onboarding (credential-store-passphrase).")
	return nil
}

func commandOptions(stdout, stderr io.Writer) hostreqkit.EnsureOptions {
	return hostreqkit.EnsureOptions{Stdout: stdout, Stderr: stderr}
}
