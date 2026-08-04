//go:build !linux

package setup

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// The platform-specific terminal APIs are intentionally kept out of setup's
// authority logic. On non-Linux hosts the OS native backend normally wins; if
// an encrypted fallback is required, this portable reader still keeps the
// passphrase inside the Vrooli setup flow and never puts it in argv or logs.
func promptForSetupPassphrase(stdout, stderr io.Writer, existing bool) (string, error) {
	_, _ = stderr.Write([]byte(""))
	reader := bufio.NewReader(os.Stdin)
	if existing {
		_, _ = fmt.Fprint(stdout, "[ACTION]  The encrypted credential store needs its passphrase: ")
	} else {
		_, _ = fmt.Fprint(stdout, "[ACTION]  Create the encrypted credential store with a passphrase: ")
	}
	first, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read credential store passphrase: %w", err)
	}
	first = strings.TrimSpace(first)
	if first == "" {
		return "", fmt.Errorf("credential store passphrase cannot be empty")
	}
	if !existing {
		_, _ = fmt.Fprint(stdout, "[ACTION]  Repeat the credential store passphrase: ")
		second, readErr := reader.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return "", fmt.Errorf("read credential store passphrase confirmation: %w", readErr)
		}
		if strings.TrimSpace(second) != first {
			return "", fmt.Errorf("credential store passphrases do not match")
		}
	}
	return first, nil
}
