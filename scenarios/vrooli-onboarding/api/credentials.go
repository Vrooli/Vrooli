package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	credentialdomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/credentials"
)

var credentialDoctorCommand = func(ctx context.Context) ([]byte, error) {
	return onboardingDoctorJSON(ctx)
}

var credentialProvisionCommand = func(ctx context.Context, logicalID, field, value string) error {
	if err := onboardingProvision(ctx, logicalID, field, value); err != nil {
		return fmt.Errorf("credential authority rejected provisioning: %w", err)
	}
	return nil
}

func listCredentials(context.Context) ([]credentialdomain.Credential, error) {
	root, err := manifestRoot()
	if err != nil {
		return nil, err
	}
	models, err := loadScenarioReadModels()
	if err != nil {
		return nil, err
	}
	closure, err := resolveClosure(root, models)
	if err != nil {
		return nil, err
	}
	return credentialMetadataInventory(closure)
}

func provisionCredential(ctx context.Context, logicalID, field, value string) error {
	requestCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return credentialProvisionCommand(requestCtx, logicalID, field, value)
}

func diagnoseCredentials(ctx context.Context) (credentialclient.DoctorResponse, error) {
	requestCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	output, err := credentialDoctorCommand(requestCtx)
	if err != nil {
		return credentialclient.DoctorResponse{}, err
	}
	var diagnosis credentialclient.DoctorResponse
	if err := json.Unmarshal(output, &diagnosis); err != nil {
		return credentialclient.DoctorResponse{}, fmt.Errorf("credential diagnosis returned invalid data: %w", err)
	}
	return diagnosis, nil
}

func credentialDescriptorField(field string) string {
	field = strings.TrimSpace(field)
	if field == "" {
		return "value"
	}
	return field
}
