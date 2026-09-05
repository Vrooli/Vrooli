package credentialclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

type ClientOptions struct {
	Authority    *credentialauthority.Authority
	InProcess    Client
	Root         string
	StateDir     string
	Descriptors  func() ([]CredentialRef, error)
	RemoteTarget string
	RemoteRunner SSHRunner
}

// NewClient selects the only transport order permitted by the architecture:
// direct authority, then explicitly configured SSH.
// It never guesses a remote host and never falls back from an unavailable
// native store to a second local store.
func NewClient(options ClientOptions) (Client, error) {
	if options.InProcess != nil {
		return options.InProcess, nil
	}
	if options.Authority != nil && options.Authority.Availability() == nil {
		client, err := NewInProcess(InProcessOptions{Authority: options.Authority, Root: options.Root, StateDir: options.StateDir, Descriptors: options.Descriptors})
		if err == nil {
			return client, nil
		}
	}
	if options.RemoteTarget != "" {
		runner := options.RemoteRunner
		if runner == nil {
			runner = execSSHRunner{}
		}
		return &sshClient{target: options.RemoteTarget, runner: runner}, nil
	}
	return nil, ErrTransportUnavailable{Transport: "credential"}
}

type SSHRunner interface {
	Run(context.Context, string, []string, io.Reader) ([]byte, error)
}

type execSSHRunner struct{}

func (execSSHRunner) Run(ctx context.Context, target string, args []string, stdin io.Reader) ([]byte, error) {
	commandArgs := append([]string{target}, args...)
	command := exec.CommandContext(ctx, "ssh", commandArgs...)
	command.Stdin = stdin
	return command.CombinedOutput()
}

type sshClient struct {
	target string
	runner SSHRunner
}

func (c *sshClient) Provision(ctx context.Context, request ProvisionRequest) (ProvisionResponse, error) {
	_, err := c.run(ctx, []string{"vrooli", "credentials", "provision", "--identity", request.Identity, "--field", request.Field}, strings.NewReader(request.Value))
	if err != nil {
		return ProvisionResponse{}, err
	}
	return ProvisionResponse{Identity: request.Identity, Field: request.Field, Provider: "ssh", Status: "provisioned"}, nil
}

func (c *sshClient) Resolve(ctx context.Context, identity, field string) (string, error) {
	output, err := c.run(ctx, []string{"vrooli", "credentials", "resolve", "--identity", identity, "--field", field}, nil)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (c *sshClient) Delete(ctx context.Context, identity, field string) error {
	_, err := c.run(ctx, []string{"secrets-manager", "credentials", "delete", "--identity", identity, "--field", field, "--yes"}, nil)
	return err
}

func (c *sshClient) Status(ctx context.Context, identity, field string) (CredentialStatus, error) {
	output, err := c.run(ctx, []string{"vrooli", "credentials", "status", "--identity", identity, "--field", field, "--format", "json"}, nil)
	if err != nil {
		return CredentialStatus{}, err
	}
	var response CredentialStatus
	return response, decodeSSHJSON(output, &response)
}

func (c *sshClient) List(ctx context.Context) ([]CredentialRef, error) {
	output, err := c.run(ctx, []string{"secrets-manager", "credentials", "list", "--format", "json"}, nil)
	if err != nil {
		return nil, err
	}
	var response []CredentialRef
	return response, decodeSSHJSON(output, &response)
}

func (c *sshClient) Doctor(ctx context.Context) (DoctorResponse, error) {
	output, err := c.run(ctx, []string{"vrooli", "credentials", "doctor", "--format", "json"}, nil)
	if err != nil {
		return DoctorResponse{}, err
	}
	var response DoctorResponse
	return response, decodeSSHJSON(output, &response)
}

func (c *sshClient) KeyringInspect(ctx context.Context, path string) (KeyringReport, error) {
	return c.keyring(ctx, "inspect", path)
}

func (c *sshClient) KeyringRepair(ctx context.Context, path string) (KeyringReport, error) {
	return c.keyring(ctx, "repair", path)
}

func (c *sshClient) RecoveryExport(ctx context.Context, request RecoveryExportRequest) (RecoveryExportResponse, error) {
	args := []string{"vrooli", "credentials", "recovery", "export", "--output", request.OutputPath, "--format", "json"}
	for _, entry := range request.Entries {
		args = append(args, "--entry", entry.LogicalID+":"+entry.Field)
	}
	output, err := c.run(ctx, args, strings.NewReader(request.Passphrase))
	if err != nil {
		return RecoveryExportResponse{}, err
	}
	var response struct {
		Written int `json:"written"`
	}
	if err := decodeSSHJSON(output, &response); err != nil {
		return RecoveryExportResponse{}, err
	}
	return RecoveryExportResponse{Path: request.OutputPath, EntryCount: response.Written}, nil
}

func (c *sshClient) RecoveryVerify(ctx context.Context, request RecoveryVerifyRequest) (RecoveryVerifyResponse, error) {
	output, err := c.run(ctx, []string{"vrooli", "credentials", "recovery", "verify", "--input", request.InputPath, "--format", "json"}, strings.NewReader(request.Passphrase))
	if err != nil {
		return RecoveryVerifyResponse{}, err
	}
	var response RecoveryVerifyResponse
	return response, decodeSSHJSON(output, &response)
}

func (c *sshClient) RecoveryRestore(ctx context.Context, request RecoveryRestoreRequest) error {
	_, err := c.run(ctx, []string{"vrooli", "credentials", "recovery", "restore", "--input", request.InputPath}, strings.NewReader(request.Passphrase))
	return err
}

func (c *sshClient) StoreStatus(ctx context.Context) (StoreStatus, error) {
	output, err := c.run(ctx, []string{"vrooli", "credentials", "store", "status", "--format", "json"}, nil)
	if err != nil {
		return StoreStatus{}, err
	}
	var response StoreStatus
	return response, decodeSSHJSON(output, &response)
}

func (c *sshClient) keyring(ctx context.Context, action, path string) (KeyringReport, error) {
	args := []string{"secrets-manager", "keyring", action, "--format", "json"}
	if strings.TrimSpace(path) != "" {
		args = append(args, "--path", path)
	}
	output, err := c.run(ctx, args, nil)
	if err != nil {
		return KeyringReport{}, err
	}
	var report KeyringReport
	if err := decodeSSHJSON(output, &report); err != nil {
		return KeyringReport{}, err
	}
	if report.Path == "" {
		return KeyringReport{}, fmt.Errorf("remote keyring returned no report")
	}
	return report, nil
}

func (c *sshClient) run(ctx context.Context, args []string, stdin io.Reader) ([]byte, error) {
	output, err := c.runner.Run(ctx, c.target, args, stdin)
	if err != nil {
		return nil, fmt.Errorf("SSH credential operation failed: %w", err)
	}
	return output, nil
}

func decodeSSHJSON(output []byte, target any) error {
	if err := json.Unmarshal(output, target); err != nil {
		return fmt.Errorf("decode SSH credential response: %w", err)
	}
	return nil
}
