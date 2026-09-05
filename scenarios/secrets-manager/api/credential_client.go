package main

import (
	"context"
	"encoding/json"
	"fmt"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

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

func ensureCredentialClient(ctx context.Context) (credentialclient.Client, error) {
	client, err := secretsCredentialClient()
	if err != nil {
		return nil, fmt.Errorf("credential client unavailable: %w", err)
	}
	return client, nil
}
