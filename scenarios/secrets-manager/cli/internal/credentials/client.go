package credentials

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

func New() (credentialclient.Client, error) {
	authority, err := credentialauthority.Default()
	if err != nil {
		return nil, err
	}
	root := repositoryRoot()
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return nil, fmt.Errorf("resolve credential state directory: %w", err)
	}
	return credentialclient.NewClient(credentialclient.ClientOptions{
		Authority:   authority,
		StateDir:    state,
		Descriptors: func() ([]credentialclient.CredentialRef, error) { return credentialclient.DiscoverDescriptors(root) },
	})
}

func repositoryRoot() string {
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		return root
	}
	current, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if fileExists(filepath.Join(current, ".vrooli")) && fileExists(filepath.Join(current, "scenarios")) {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func ReadPassphrase() (string, error) {
	data, err := io.ReadAll(io.LimitReader(os.Stdin, 64*1024))
	if err != nil {
		return "", fmt.Errorf("read passphrase from stdin: %w", err)
	}
	if string(data) == "" {
		return "", fmt.Errorf("passphrase is required on stdin")
	}
	return string(data), nil
}
