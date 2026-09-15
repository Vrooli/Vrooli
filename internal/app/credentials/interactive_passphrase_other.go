//go:build !linux

package credentials

import (
	"fmt"
	"io"
	"os"
)

func readInteractivePassphrase(_ *os.File, _ io.Writer) (string, error) {
	return "", fmt.Errorf("interactive credential store passphrase input is unavailable on this platform; provide it through standard input")
}

func readInteractivePassphraseWithLabel(_ *os.File, _ io.Writer, _ string) (string, error) {
	return "", fmt.Errorf("interactive credential store passphrase input is unavailable on this platform; provide it through standard input")
}

func readInteractiveCredentialValue(_ *os.File, _ io.Writer) (string, error) {
	return "", fmt.Errorf("interactive credential value input is unavailable on this platform; provide it through standard input")
}
