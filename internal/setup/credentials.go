package setup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/resources/securestore"
)

var (
	credentialStoreExecutableFn = os.Executable
	runCredentialStoreFn        = hostreqkit.RunAsInvokingUserWithInput
)

func configureCredentialBackend(stdout, stderr io.Writer) error {
	status, err := securestore.EnsureSetupBackend()
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
				maybeAddHostBoundWrap(executable)
				return nil
			}
		}
	}

	passphrase, err := promptForSetupPassphrase(stdout, stderr, description.Initialized)
	if err != nil {
		return err
	}
	if description.Initialized {
		if err := runCredentialStoreFn(executable,
			[]string{"credentials", "store", "unlock"}, passphrase,
			commandOptions(stdout, stderr)); err != nil {
			return fmt.Errorf("unlock encrypted credential store: %w", err)
		}
		maybeAddHostBoundWrap(executable)
		return nil
	}
	if err := runCredentialStoreFn(executable,
		[]string{"credentials", "store", "init", "--format", "json"}, passphrase,
		commandOptions(stdout, stderr)); err != nil {
		return fmt.Errorf("initialize encrypted credential store: %w", err)
	}
	return nil
}

func commandOptions(stdout, stderr io.Writer) hostreqkit.EnsureOptions {
	return hostreqkit.EnsureOptions{Stdout: stdout, Stderr: stderr}
}

func maybeAddHostBoundWrap(executable string) {
	// Rewrap is intentionally best-effort here. A host without a usable TPM is
	// already fully ready through its operator passphrase; when setup's
	// safeguard has made the TPM available, this convergent step upgrades the
	// existing store without asking the operator to run a second command.
	_ = runCredentialStoreFn(executable,
		[]string{"credentials", "store", "rewrap"}, "",
		commandOptions(io.Discard, io.Discard))
}
