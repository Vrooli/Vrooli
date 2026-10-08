package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// The control-plane command owns the secure backend. Values travel only on
// stdin and are never included in argv, durable state, or structured logs.
type cliCredentialStore struct{}

func (cliCredentialStore) Provision(ctx context.Context, request CredentialProvisionRequest) error {
	command := exec.CommandContext(ctx, "vrooli", "credentials", "provision", "--identity", request.Identity, "--field", request.Field)
	command.Stdin = bytes.NewBufferString(request.Value)
	if _, err := command.Output(); err != nil {
		return fmt.Errorf("credential authority provision failed: %w", err)
	}
	return nil
}

func (cliCredentialStore) Status(ctx context.Context, identity, field string) (CredentialStatus, error) {
	command := exec.CommandContext(ctx, "vrooli", "credentials", "status", "--identity", identity, "--field", field, "--format", "json")
	output, err := command.Output()
	if err != nil {
		return CredentialStatus{}, fmt.Errorf("credential authority status failed: %w", err)
	}
	var status CredentialStatus
	if err := json.Unmarshal(output, &status); err != nil {
		return CredentialStatus{}, fmt.Errorf("decode credential authority status: %w", err)
	}
	return status, nil
}

func (cliCredentialStore) Delete(ctx context.Context, identity, field string) error {
	command := exec.CommandContext(ctx, "vrooli", "credentials", "delete", "--identity", identity, "--field", field, "--yes")
	if err := command.Run(); err != nil {
		return fmt.Errorf("credential authority delete failed: %w", err)
	}
	return nil
}
