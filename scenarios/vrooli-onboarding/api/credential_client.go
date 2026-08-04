package main

import (
	"context"
	"encoding/json"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

func onboardingCredentialClient() (credentialclient.Client, error) {
	authority, err := credentialauthority.Default()
	if err != nil {
		return nil, err
	}
	return credentialclient.NewClient(credentialclient.ClientOptions{Authority: authority})
}

func onboardingDoctorJSON(ctx context.Context) ([]byte, error) {
	client, err := onboardingCredentialClient()
	if err != nil {
		return nil, err
	}
	response, err := client.Doctor(ctx)
	if err != nil {
		return nil, err
	}
	return json.Marshal(response)
}

func onboardingKeyringJSON(ctx context.Context, action string) ([]byte, error) {
	client, err := onboardingCredentialClient()
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

func onboardingProvision(ctx context.Context, logicalID, field, value string) error {
	client, err := onboardingCredentialClient()
	if err != nil {
		return err
	}
	_, err = client.Provision(ctx, credentialclient.ProvisionRequest{Identity: logicalID, Field: field, Value: value})
	return err
}

func onboardingStatusJSON(ctx context.Context, logicalID, field string) ([]byte, error) {
	client, err := onboardingCredentialClient()
	if err != nil {
		return nil, err
	}
	status, err := client.Status(ctx, logicalID, field)
	if err != nil {
		return nil, err
	}
	return json.Marshal(status)
}
