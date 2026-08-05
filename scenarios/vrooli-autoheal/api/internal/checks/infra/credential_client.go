package infra

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

var keyringInspectOutput = autohealKeyringInspect
var keyringRepairOutput = autohealKeyringRepair

func autohealCredentialClient() (credentialclient.Client, error) {
	authority, err := credentialauthority.Default()
	if err != nil {
		return nil, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	stateDir, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return nil, err
	}
	root := repositoryRoot()
	return credentialclient.NewClient(credentialclient.ClientOptions{
		Authority:   authority,
		StateDir:    stateDir,
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
		if pathExists(filepath.Join(current, ".vrooli")) && pathExists(filepath.Join(current, "scenarios")) {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func autohealKeyringInspect(ctx context.Context, path string) ([]byte, error) {
	client, err := autohealCredentialClient()
	if err != nil {
		return nil, err
	}
	report, err := client.KeyringInspect(ctx, path)
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Reports []credentialclient.KeyringReport `json:"reports"`
	}{Reports: []credentialclient.KeyringReport{report}})
}

func autohealKeyringRepair(ctx context.Context, path string) ([]byte, error) {
	client, err := autohealCredentialClient()
	if err != nil {
		return nil, err
	}
	report, err := client.KeyringRepair(ctx, path)
	if err != nil {
		return nil, err
	}
	return json.Marshal(report)
}
