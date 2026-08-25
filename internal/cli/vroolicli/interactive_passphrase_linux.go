//go:build linux

package vroolicli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

func readInteractivePassphrase(input *os.File, prompt io.Writer) (string, error) {
	return readInteractivePassphraseWithLabel(input, prompt, "Credential store passphrase: ")
}

func readInteractivePassphraseWithLabel(input *os.File, prompt io.Writer, label string) (string, error) {
	return readInteractiveSecret(input, prompt, label)
}

func readInteractiveCredentialValue(input *os.File, prompt io.Writer) (string, error) {
	return readInteractiveSecret(input, prompt, "Credential value: ")
}

func readInteractiveSecret(input *os.File, prompt io.Writer, label string) (string, error) {
	state, err := unix.IoctlGetTermios(int(input.Fd()), unix.TCGETS)
	if err != nil {
		return "", fmt.Errorf("inspect terminal for credential store passphrase: %w", err)
	}
	quiet := *state
	quiet.Lflag &^= unix.ECHO
	if err := unix.IoctlSetTermios(int(input.Fd()), unix.TCSETS, &quiet); err != nil {
		return "", fmt.Errorf("disable passphrase echo: %w", err)
	}
	defer func() { _ = unix.IoctlSetTermios(int(input.Fd()), unix.TCSETS, state) }()

	if prompt == nil {
		prompt = io.Discard
	}
	if _, err := fmt.Fprint(prompt, label); err != nil {
		return "", err
	}
	value, err := bufio.NewReader(input).ReadString('\n')
	if _, writeErr := fmt.Fprintln(prompt); err == nil {
		err = writeErr
	}
	if err != nil {
		return "", fmt.Errorf("read credential store passphrase: %w", err)
	}
	return strings.TrimSpace(value), nil
}
