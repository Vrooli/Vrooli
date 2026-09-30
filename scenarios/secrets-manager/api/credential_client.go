package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

func resolveSecretsManagerCredential(envName, logicalID, field string) (string, error) {
	if value := strings.TrimSpace(os.Getenv(envName)); value != "" {
		return value, nil
	}
	identity, err := credentialauthority.ParseIdentity(logicalID)
	if err != nil {
		return "", fmt.Errorf("parse credential identity: %w", err)
	}
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", fmt.Errorf("credential authority unavailable: %w", err)
	}
	value, err := authority.Require(identity, field)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func configuredSecretsManagerOwnerToken() (string, error) {
	return resolveSecretsManagerCredential("SECRETS_MANAGER_OWNER_TOKEN", "vrooli/secrets-manager/owner", "owner-token")
}

// configuredSecretsManagerDeploymentToken is intentionally separate from the
// owner token. Deployment consumers only need to read tier-specific manifest
// metadata; they must never receive the owner capability used by the vault.
func configuredSecretsManagerDeploymentToken() (string, error) {
	return resolveSecretsManagerCredential("SECRETS_MANAGER_DEPLOYMENT_TOKEN", "vrooli/secrets-manager/deployment", "service-token")
}

func configuredSecretsManagerNativeHostTransport() (string, error) {
	return resolveSecretsManagerCredential("SECRETS_MANAGER_NATIVE_HOST_TOKEN", "vrooli/secrets-manager/native-host", "transport-token")
}

func secretsCredentialClient() (credentialclient.Client, error) {
	authority, err := credentialauthority.Default()
	if err != nil {
		return nil, err
	}
	return credentialclient.NewClient(credentialclient.ClientOptions{Authority: authority})
}

func secretsDoctorJSON(ctx context.Context) ([]byte, error) {
	client, err := secretsCredentialClient()
	if err != nil {
		return nil, err
	}
	response, err := client.Doctor(ctx)
	if err != nil {
		return nil, err
	}
	return json.Marshal(response)
}

func secretsKeyringJSON(ctx context.Context, action string) ([]byte, error) {
	client, err := secretsCredentialClient()
	if err != nil {
		return nil, err
	}
	var report credentialclient.KeyringReport
	if action == "inspect" {
		report, err = client.KeyringInspect(ctx, "")
	} else {
		report, err = client.KeyringRepair(ctx, "")
	}
	if err != nil {
		return nil, err
	}
	return json.Marshal(report)
}

func secretsProvision(ctx context.Context, logicalID, field, value string) error {
	client, err := secretsCredentialClient()
	if err != nil {
		return err
	}
	_, err = client.Provision(ctx, credentialclient.ProvisionRequest{Identity: logicalID, Field: field, Value: value})
	return err
}

func secretsStatusJSON(ctx context.Context, logicalID, field string) ([]byte, error) {
	client, err := secretsCredentialClient()
	if err != nil {
		return nil, err
	}
	status, err := client.Status(ctx, logicalID, field)
	if err != nil {
		return nil, err
	}
	return json.Marshal(status)
}
