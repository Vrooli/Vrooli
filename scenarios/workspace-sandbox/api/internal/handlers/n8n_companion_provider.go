package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	companionOwnerTimeout     = 30 * time.Second
	companionOwnerOutputLimit = 32 * 1024
)

var privateServiceAlias = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)
var ownerCompanionRuntimeID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$`)
var companionDatabaseName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

// N8NCompanionOwner is the control-plane lifecycle contract used by the
// Workspace Sandbox provider. Its implementation must use the n8n resource
// owner; Sandbox does not own Docker objects or networks.
type N8NCompanionOwner interface {
	StartPostgres(context.Context, uuid.UUID, string, string, string) (N8NCompanionReceipt, error)
	StopCompanion(context.Context, uuid.UUID, string) error
}

type N8NCompanionReceipt struct {
	SandboxID          string `json:"sandboxId"`
	CompanionRuntimeID string `json:"companionRuntimeId"`
	NetworkAlias       string `json:"networkAlias"`
	Status             string `json:"status"`
	NoEgressVerified   bool   `json:"noEgressVerified"`
}

// N8NCompanionServiceProvider connects the Sandbox lifecycle to the n8n
// resource owner's fixed Postgres companion operations. No command, image,
// host path, network, or credential is supplied by a sandbox request.
type N8NCompanionServiceProvider struct{ Owner N8NCompanionOwner }

func (p N8NCompanionServiceProvider) Start(ctx context.Context, sandboxID uuid.UUID, spec CompanionServiceSpec) (CompanionRuntime, error) {
	if p.Owner == nil {
		return CompanionRuntime{}, errors.New("n8n companion owner is unavailable")
	}
	if spec.Kind != "postgres" {
		return CompanionRuntime{}, fmt.Errorf("n8n owner currently supports only synthetic postgres companions")
	}
	if !companionName.MatchString(spec.Name) || !companionVersion.MatchString(spec.Version) {
		return CompanionRuntime{}, errors.New("companion name or pinned version is invalid")
	}
	database := spec.Options["database"]
	if database == "" {
		database = "workflow"
	}
	if !companionDatabaseName.MatchString(database) {
		return CompanionRuntime{}, errors.New("companion database must be a simple identifier")
	}
	receipt, err := p.Owner.StartPostgres(ctx, sandboxID, spec.Name, spec.Version, database)
	if err != nil {
		return CompanionRuntime{}, err
	}
	if receipt.SandboxID != sandboxID.String() || receipt.Status != "attached" || !receipt.NoEgressVerified || !privateServiceAlias.MatchString(receipt.NetworkAlias) || !ownerCompanionRuntimeID.MatchString(receipt.CompanionRuntimeID) {
		// Best effort cleanup if an owner response is incomplete. The owner validates
		// identity before every mutation, so the untrusted response cannot target a
		// foreign runtime.
		if receipt.CompanionRuntimeID != "" {
			_ = p.Owner.StopCompanion(ctx, sandboxID, receipt.CompanionRuntimeID)
		}
		return CompanionRuntime{}, errors.New("n8n owner returned an invalid or unattached companion receipt")
	}
	return CompanionRuntime{ID: receipt.CompanionRuntimeID, Address: receipt.NetworkAlias + ":5432"}, nil
}

func (p N8NCompanionServiceProvider) Stop(ctx context.Context, sandboxID uuid.UUID, state CompanionServiceState) error {
	if p.Owner == nil {
		return errors.New("n8n companion owner is unavailable")
	}
	if state.Spec.Kind != "postgres" {
		return fmt.Errorf("n8n owner currently supports only synthetic postgres companions")
	}
	if !ownerCompanionRuntimeID.MatchString(strings.TrimSpace(state.RuntimeID)) {
		return errors.New("valid companion runtime ID is required for cleanup")
	}
	return p.Owner.StopCompanion(ctx, sandboxID, state.RuntimeID)
}

// N8NCompanionCLI bridges to the supported vrooli CLI command surface. It
// invokes a fixed command directly (never through a shell), places strict
// bounds on time and output, and does not forward the API process environment.
// The command is intentionally not discovered from request data.
type N8NCompanionCLI struct{ Binary string }

func (c N8NCompanionCLI) StartPostgres(ctx context.Context, sandboxID uuid.UUID, name, version, database string) (N8NCompanionReceipt, error) {
	if c.Binary == "" {
		return N8NCompanionReceipt{}, errors.New("vrooli control-plane executable is not configured")
	}
	args := []string{"resource", "n8n", "runtime", "companion-start", "--sandbox-id", sandboxID.String(), "--companion-name", name, "--service-version", version, "--database", database, "--json"}
	output, err := c.run(ctx, args...)
	if err != nil {
		return N8NCompanionReceipt{}, err
	}
	var receipt N8NCompanionReceipt
	if err := json.Unmarshal(output, &receipt); err != nil {
		return N8NCompanionReceipt{}, fmt.Errorf("decode n8n companion-start receipt: %w", err)
	}
	return receipt, nil
}

func (c N8NCompanionCLI) StopCompanion(ctx context.Context, sandboxID uuid.UUID, runtimeID string) error {
	if c.Binary == "" {
		return errors.New("vrooli control-plane executable is not configured")
	}
	_, err := c.run(ctx, "resource", "n8n", "runtime", "companion-stop", "--sandbox-id", sandboxID.String(), "--companion-runtime-id", runtimeID, "--json")
	return err
}

func (c N8NCompanionCLI) run(ctx context.Context, args ...string) ([]byte, error) {
	callCtx, cancel := context.WithTimeout(ctx, companionOwnerTimeout)
	defer cancel()
	cmd := exec.CommandContext(callCtx, c.Binary, args...)
	cmd.Env = []string{"PATH=/usr/bin:/bin"}
	for _, key := range []string{"HOME", "VROOLI_HOME", "VROOLI_ROOT", "XDG_CONFIG_HOME"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}
	stdout, stderr := &boundedOutput{}, &boundedOutput{}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("n8n companion owner command failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

type boundedOutput struct{ bytes.Buffer }

func (b *boundedOutput) Write(data []byte) (int, error) {
	if b.Len()+len(data) > companionOwnerOutputLimit {
		return 0, fmt.Errorf("owner command output exceeds %d bytes", companionOwnerOutputLimit)
	}
	return b.Buffer.Write(data)
}

var _ CompanionServiceAPI = N8NCompanionServiceProvider{}
var _ N8NCompanionOwner = N8NCompanionCLI{}
var _ io.Writer = (*boundedOutput)(nil)
