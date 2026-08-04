//go:build linux

package setup

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

func promptForSetupPassphrase(stdout, stderr io.Writer, existing bool) (string, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", fmt.Errorf("credential store needs a passphrase, but setup could not open the controlling terminal: %w", err)
	}
	defer tty.Close()

	termios, err := unix.IoctlGetTermios(int(tty.Fd()), unix.TCGETS)
	if err != nil {
		return "", fmt.Errorf("inspect terminal for credential passphrase: %w", err)
	}
	noEcho := *termios
	noEcho.Lflag &^= unix.ECHO
	if err := unix.IoctlSetTermios(int(tty.Fd()), unix.TCSETS, &noEcho); err != nil {
		return "", fmt.Errorf("protect credential passphrase input: %w", err)
	}
	defer func() { _ = unix.IoctlSetTermios(int(tty.Fd()), unix.TCSETS, termios) }()

	if existing {
		_, _ = fmt.Fprint(stdout, "[ACTION]  The encrypted credential store needs its passphrase: ")
	} else {
		_, _ = fmt.Fprint(stdout, "[ACTION]  Create the encrypted credential store with a passphrase: ")
	}
	reader := bufio.NewReader(tty)
	first, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read credential store passphrase: %w", err)
	}
	_, _ = fmt.Fprintln(stdout)
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
		_, _ = fmt.Fprintln(stdout)
		if strings.TrimSpace(second) != first {
			return "", fmt.Errorf("credential store passphrases do not match")
		}
	}
	_ = stderr
	return first, nil
}
