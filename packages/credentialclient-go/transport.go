package credentialclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

type ClientOptions struct {
	Authority       *credentialauthority.Authority
	InProcess       Client
	StateDir        string
	Descriptors     func() ([]CredentialRef, error)
	BundlePortFile  string
	BundleTokenFile string
	RemoteTarget    string
	RemoteRunner    SSHRunner
}

// NewClient selects the only transport order permitted by the architecture:
// direct authority, authenticated desktop IPC, then explicitly configured SSH.
// It never guesses a remote host and never falls back from an unavailable
// native store to a second local store.
func NewClient(options ClientOptions) (Client, error) {
	if options.InProcess != nil {
		return options.InProcess, nil
	}
	if options.Authority != nil && options.Authority.Availability() == nil {
		client, err := NewInProcess(InProcessOptions{Authority: options.Authority, StateDir: options.StateDir, Descriptors: options.Descriptors})
		if err == nil {
			return client, nil
		}
	}
	if fileExists(options.BundlePortFile) && fileExists(options.BundleTokenFile) {
		return &ipcClient{portFile: options.BundlePortFile, tokenFile: options.BundleTokenFile}, nil
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

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

type ipcClient struct {
	portFile  string
	tokenFile string
}

func (c *ipcClient) Provision(ctx context.Context, request ProvisionRequest) (ProvisionResponse, error) {
	var response ProvisionResponse
	err := c.request(ctx, http.MethodPost, "/credentials/provision", map[string]string{
		"identity": request.Identity,
		"field":    request.Field,
		"value":    request.Value,
	}, &response)
	if err != nil {
		return ProvisionResponse{}, err
	}
	if response.Provider == "" {
		response.Provider = "desktop-runtime"
	}
	return response, nil
}

func (c *ipcClient) Resolve(ctx context.Context, identity, field string) (string, error) {
	var response struct {
		Value string `json:"value"`
	}
	path := "/credentials/resolve?identity=" + queryEscape(identity) + "&field=" + queryEscape(field)
	if err := c.request(ctx, http.MethodGet, path, nil, &response); err != nil {
		return "", err
	}
	if strings.TrimSpace(response.Value) == "" {
		return "", credentialauthority.ErrUnconfigured
	}
	return response.Value, nil
}
func (c *ipcClient) Delete(context.Context, string, string) error { return ipcUnavailable() }
func (c *ipcClient) Status(ctx context.Context, identity, field string) (CredentialStatus, error) {
	var view desktopCredentialView
	path := "/credentials/status?identity=" + queryEscape(identity) + "&field=" + queryEscape(field)
	if err := c.request(ctx, http.MethodGet, path, nil, &view); err != nil {
		return CredentialStatus{}, err
	}
	return view.status(), nil
}

func (c *ipcClient) List(ctx context.Context) ([]CredentialRef, error) {
	var views []desktopCredentialView
	if err := c.request(ctx, http.MethodGet, "/credentials/list", nil, &views); err != nil {
		return nil, err
	}
	return credentialRefs(views), nil
}

func (c *ipcClient) Doctor(ctx context.Context) (DoctorResponse, error) {
	var response struct {
		Provider struct {
			Condition string `json:"condition"`
		} `json:"provider"`
		Credentials []desktopCredentialView `json:"credentials"`
		Recovery    RecoveryStatus          `json:"recovery"`
	}
	if err := c.request(ctx, http.MethodGet, "/credentials/doctor", nil, &response); err != nil {
		return DoctorResponse{}, err
	}
	return DoctorResponse{
		Provider:    ProviderDiagnosis{Condition: response.Provider.Condition, Available: response.Provider.Condition == "available"},
		Credentials: credentialRefs(response.Credentials),
		Recovery:    response.Recovery,
	}, nil
}

func (c *ipcClient) KeyringInspect(context.Context, string) (KeyringReport, error) {
	return KeyringReport{}, ipcUnavailable()
}

func (c *ipcClient) KeyringRepair(context.Context, string) (KeyringReport, error) {
	return KeyringReport{}, ipcUnavailable()
}

func (c *ipcClient) RecoveryExport(ctx context.Context, request RecoveryExportRequest) (RecoveryExportResponse, error) {
	var response struct {
		Bundle     string `json:"bundle"`
		EntryCount int    `json:"entry_count"`
	}
	if err := c.request(ctx, http.MethodPost, "/credentials/recovery/export", map[string]string{"passphrase": request.Passphrase}, &response); err != nil {
		return RecoveryExportResponse{}, err
	}
	bundle, err := base64.StdEncoding.DecodeString(response.Bundle)
	if err != nil {
		return RecoveryExportResponse{}, fmt.Errorf("decode desktop recovery bundle: %w", err)
	}
	if strings.TrimSpace(request.OutputPath) == "" {
		return RecoveryExportResponse{}, fmt.Errorf("recovery output path is required")
	}
	if err := os.WriteFile(request.OutputPath, bundle, 0o600); err != nil {
		return RecoveryExportResponse{}, err
	}
	return RecoveryExportResponse{Path: request.OutputPath, EntryCount: response.EntryCount}, nil
}

func (c *ipcClient) RecoveryVerify(context.Context, RecoveryVerifyRequest) (RecoveryVerifyResponse, error) {
	return RecoveryVerifyResponse{}, ipcUnavailable()
}

func (c *ipcClient) RecoveryRestore(ctx context.Context, request RecoveryRestoreRequest) error {
	bundle, err := os.ReadFile(request.InputPath)
	if err != nil {
		return err
	}
	return c.request(ctx, http.MethodPost, "/credentials/recovery/restore", map[string]string{
		"bundle":     base64.StdEncoding.EncodeToString(bundle),
		"passphrase": request.Passphrase,
	}, nil)
}

func (c *ipcClient) StoreStatus(context.Context) (StoreStatus, error) {
	return StoreStatus{}, ipcUnavailable()
}

type desktopCredentialView struct {
	Identity   string `json:"identity"`
	Field      string `json:"field"`
	Configured bool   `json:"configured"`
	Required   bool   `json:"required"`
	Label      string `json:"label"`
}

func (v desktopCredentialView) status() CredentialStatus {
	state := "missing"
	if v.Configured {
		state = "configured"
	}
	return CredentialStatus{Identity: v.Identity, Field: v.Field, Configured: v.Configured, Provider: "desktop-runtime", ProviderState: state}
}

func credentialRefs(views []desktopCredentialView) []CredentialRef {
	refs := make([]CredentialRef, 0, len(views))
	for _, view := range views {
		refs = append(refs, CredentialRef{LogicalID: view.Identity, Field: view.Field, Label: view.Label, Required: view.Required})
	}
	return refs
}

func (c *ipcClient) request(ctx context.Context, method, path string, payload any, output any) error {
	portData, err := os.ReadFile(c.portFile)
	if err != nil {
		return fmt.Errorf("read desktop IPC port: %w", err)
	}
	port := strings.TrimSpace(string(portData))
	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf("desktop IPC port is invalid: %w", err)
	}
	tokenData, err := os.ReadFile(c.tokenFile)
	if err != nil {
		return fmt.Errorf("read desktop IPC token: %w", err)
	}
	token := strings.TrimSpace(string(tokenData))
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode desktop IPC request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, "http://127.0.0.1:"+port+path, body)
	if err != nil {
		return fmt.Errorf("build desktop IPC request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("desktop IPC request: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read desktop IPC response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("desktop IPC status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	if output != nil && len(responseBody) > 0 {
		if err := json.Unmarshal(responseBody, output); err != nil {
			return fmt.Errorf("decode desktop IPC response: %w", err)
		}
	}
	return nil
}

func queryEscape(value string) string {
	return strings.NewReplacer("%", "%25", " ", "%20", "/", "%2F", "?", "%3F", "&", "%26", "=", "%3D", "+", "%2B").Replace(value)
}

func ipcUnavailable() error { return ErrTransportUnavailable{Transport: "desktop IPC"} }

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
