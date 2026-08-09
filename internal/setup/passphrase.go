package setup

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func readCredentialPassphraseStdin() (string, error) {
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read credential store passphrase from standard input: %w", err)
	}
	passphrase := strings.TrimSpace(strings.TrimSuffix(line, "\n"))
	if passphrase == "" {
		return "", fmt.Errorf("credential store passphrase from standard input cannot be empty")
	}
	return passphrase, nil
}

var (
	openSetupTerminalFn = openSetupTerminal
	readSetupPasswordFn = func(fd int) ([]byte, error) {
		return term.ReadPassword(fd)
	}
)

func openSetupTerminal() (*os.File, error) {
	if hostreqspec.CurrentPlatform() == "windows" {
		return os.Stdin, nil
	}
	return os.OpenFile("/dev/tty", os.O_RDWR, 0)
}

// promptForSetupPassphrase reads directly from the controlling terminal so a
// piped stdin cannot bypass setup's credential prompt. term.ReadPassword
// disables terminal echo on Linux, macOS, and Windows for the duration of each
// read; typed secret material is never written to stdout or stderr.
func promptForSetupPassphrase(stdout, stderr io.Writer, existing bool) (string, error) {
	tty, err := openSetupTerminalFn()
	if err != nil {
		return "", fmt.Errorf("credential store needs a passphrase, but setup could not open the controlling terminal: %w", err)
	}
	if tty != os.Stdin {
		defer tty.Close()
	}

	prompt := "[ACTION]  Create the encrypted credential store with a passphrase: "
	if existing {
		prompt = "[ACTION]  The encrypted credential store needs its passphrase: "
	}
	_, _ = fmt.Fprint(stdout, prompt)
	first, err := readSetupPasswordFn(int(tty.Fd()))
	if err != nil {
		return "", fmt.Errorf("read credential store passphrase: %w", err)
	}
	_, _ = fmt.Fprintln(stdout)
	passphrase := strings.TrimSpace(string(first))
	if passphrase == "" {
		return "", fmt.Errorf("credential store passphrase cannot be empty")
	}

	if !existing {
		_, _ = fmt.Fprint(stdout, "[ACTION]  Repeat the credential store passphrase: ")
		second, readErr := readSetupPasswordFn(int(tty.Fd()))
		if readErr != nil {
			return "", fmt.Errorf("read credential store passphrase confirmation: %w", readErr)
		}
		_, _ = fmt.Fprintln(stdout)
		if strings.TrimSpace(string(second)) != passphrase {
			return "", fmt.Errorf("credential store passphrases do not match")
		}
	}
	_ = stderr
	return passphrase, nil
}
