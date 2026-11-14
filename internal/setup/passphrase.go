package setup

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
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
