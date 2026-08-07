package setup

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestPromptForSetupPassphraseDoesNotEchoSecret(t *testing.T) {
	withPassphraseTestSeams(t, "secret-value", "secret-value")

	var stdout, stderr bytes.Buffer
	got, err := promptForSetupPassphrase(&stdout, &stderr, false)
	if err != nil {
		t.Fatalf("promptForSetupPassphrase: %v", err)
	}
	if got != "secret-value" {
		t.Fatalf("passphrase = %q, want secret-value", got)
	}
	if strings.Contains(stdout.String(), "secret-value") || strings.Contains(stderr.String(), "secret-value") {
		t.Fatal("typed passphrase was written to stdout or stderr")
	}
}

func TestPromptForSetupPassphraseRejectsEmpty(t *testing.T) {
	withPassphraseTestSeams(t, "", "")
	var output bytes.Buffer
	if _, err := promptForSetupPassphrase(&output, &output, true); err == nil || !strings.Contains(err.Error(), "cannot be empty") {
		t.Fatalf("error = %v, want empty-passphrase rejection", err)
	}
}

func TestPromptForSetupPassphraseRejectsMismatchedConfirmation(t *testing.T) {
	withPassphraseTestSeams(t, "first", "second")
	var output bytes.Buffer
	if _, err := promptForSetupPassphrase(&output, &output, false); err == nil || !strings.Contains(err.Error(), "do not match") {
		t.Fatalf("error = %v, want mismatched-confirmation rejection", err)
	}
}

func withPassphraseTestSeams(t *testing.T, values ...string) {
	t.Helper()
	oldOpen := openSetupTerminalFn
	oldRead := readSetupPasswordFn
	t.Cleanup(func() {
		openSetupTerminalFn = oldOpen
		readSetupPasswordFn = oldRead
	})
	openSetupTerminalFn = func() (*os.File, error) {
		return os.Open(os.DevNull)
	}
	index := 0
	readSetupPasswordFn = func(int) ([]byte, error) {
		if index >= len(values) {
			return nil, errors.New("test password exhausted")
		}
		value := values[index]
		index++
		return []byte(value), nil
	}
}
