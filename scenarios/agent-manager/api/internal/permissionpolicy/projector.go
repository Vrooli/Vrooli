package permissionpolicy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"github.com/vrooli/envkit-go"
)

const (
	resourcePermissionSchemaVersion = "v1"
	defaultResourceTimeout          = 10 * time.Second
)

var (
	// ErrResourceUnavailable means an owning resource CLI could not provide a
	// plan or reconcile result. The aggregate service can continue with other
	// resources but must never claim a global success.
	ErrResourceUnavailable = errors.New("resource permission adapter unavailable")
	// ErrInvalidResourceResponse means a resource response cannot safely become
	// permission audit evidence.
	ErrInvalidResourceResponse = errors.New("invalid resource permission response")
	// ErrAuthorizationRequired prevents a caller from accidentally reconciling
	// native files without the explicit human authorization signal.
	ErrAuthorizationRequired = errors.New("explicit human authorization is required")
)

// CommandExecutor is the subprocess seam. Production uses CombinedOutput so
// failed resource diagnostics are retained; tests use deterministic fakes.
type CommandExecutor interface {
	Run(ctx context.Context, command string, args ...string) ([]byte, error)
}

type commandExecutor struct{}

func (commandExecutor) Run(ctx context.Context, command string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = []string(envkit.WithOverlay(envkit.Env(os.Environ()), envkit.Resource, nil))
	return cmd.CombinedOutput()
}

// Projector delegates projection to the resource that owns native permission
// syntax and files. It must never inspect or write those native files itself.
type Projector interface {
	Plan(ctx context.Context, request ProjectionRequest) (ProjectionResult, error)
	Reconcile(ctx context.Context, request ProjectionRequest, explicitlyAuthorized bool) (ProjectionResult, error)
}

type ProjectionRequest struct {
	Runner   domain.RunnerType
	Document ResourceDocument
}

type ProjectionResult struct {
	Runner             domain.RunnerType
	Scope              string
	DesiredDigest      string
	DesiredFingerprint string
	LiveFingerprint    string
	Drift              bool
	Changes            []string
	NativePaths        []string
	Enforcement        EnforcementPosture
}

type EnforcementPosture struct {
	Permissions string
	Caveats     []string
}

// ResourcePermissionProjector is the production resource CLI adapter.
type ResourcePermissionProjector struct {
	executor CommandExecutor
	timeout  time.Duration
}

func NewResourcePermissionProjector(executor CommandExecutor) *ResourcePermissionProjector {
	return newResourcePermissionProjector(executor, defaultResourceTimeout)
}

func newResourcePermissionProjector(executor CommandExecutor, timeout time.Duration) *ResourcePermissionProjector {
	if executor == nil {
		executor = commandExecutor{}
	}
	if timeout <= 0 {
		timeout = defaultResourceTimeout
	}
	return &ResourcePermissionProjector{executor: executor, timeout: timeout}
}

func (p *ResourcePermissionProjector) Plan(ctx context.Context, request ProjectionRequest) (ProjectionResult, error) {
	return p.project(ctx, "plan", request, false)
}

func (p *ResourcePermissionProjector) Reconcile(ctx context.Context, request ProjectionRequest, explicitlyAuthorized bool) (ProjectionResult, error) {
	if !explicitlyAuthorized {
		return ProjectionResult{}, ErrAuthorizationRequired
	}
	return p.project(ctx, "reconcile", request, true)
}

func (p *ResourcePermissionProjector) project(ctx context.Context, operation string, request ProjectionRequest, explicitlyAuthorized bool) (ProjectionResult, error) {
	if p == nil || p.executor == nil {
		return ProjectionResult{}, fmt.Errorf("%w: projector is not configured", ErrResourceUnavailable)
	}
	if !request.Runner.IsValid() {
		return ProjectionResult{}, fmt.Errorf("%w: runner %q is unsupported", ErrInvalidResourceResponse, request.Runner)
	}
	if !validScope(request.Document.Scope) || request.Document.SchemaVersion != resourcePermissionSchemaVersion {
		return ProjectionResult{}, fmt.Errorf("%w: invalid resource document", ErrInvalidResourceResponse)
	}
	data, err := json.Marshal(request.Document)
	if err != nil {
		return ProjectionResult{}, fmt.Errorf("%w: serialize resource document: %v", ErrInvalidResourceResponse, err)
	}
	expectedDigest := digest(data)
	documentPath, err := writeTemporaryDocument(data)
	if err != nil {
		return ProjectionResult{}, fmt.Errorf("%w: write temporary document: %v", ErrResourceUnavailable, err)
	}
	defer os.Remove(documentPath)

	timeoutCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	args := resourceArguments(request.Runner, operation, documentPath, request.Document.Scope, explicitlyAuthorized)
	output, err := p.executor.Run(timeoutCtx, resourceCommand(request.Runner), args...)
	if err != nil {
		if timeoutCtx.Err() != nil {
			return ProjectionResult{}, fmt.Errorf("%w: %s %s timed out: %v", ErrResourceUnavailable, request.Runner, operation, timeoutCtx.Err())
		}
		return ProjectionResult{}, fmt.Errorf("%w: %s %s: %v", ErrResourceUnavailable, request.Runner, operation, err)
	}
	response, err := parseResourcePermissionResponse(output)
	if err != nil {
		return ProjectionResult{}, fmt.Errorf("%w: %s %s: %v", ErrInvalidResourceResponse, request.Runner, operation, err)
	}
	if response.Runner != string(request.Runner) {
		return ProjectionResult{}, fmt.Errorf("%w: response runner %q does not match requested runner %q", ErrInvalidResourceResponse, response.Runner, request.Runner)
	}
	if response.Scope != request.Document.Scope {
		return ProjectionResult{}, fmt.Errorf("%w: response scope %q does not match requested scope %q", ErrInvalidResourceResponse, response.Scope, request.Document.Scope)
	}
	if response.DesiredDigest != expectedDigest {
		return ProjectionResult{}, fmt.Errorf("%w: response desired digest %q does not match document digest %q", ErrInvalidResourceResponse, response.DesiredDigest, expectedDigest)
	}
	return response.toProjectionResult(), nil
}

func resourceArguments(runner domain.RunnerType, operation, documentPath, scope string, explicitlyAuthorized bool) []string {
	args := []string{"permissions", operation, "--document", documentPath, "--json"}
	if runner == domain.RunnerTypeCodex || runner == domain.RunnerTypeGrok {
		args = append(args, "--scope", scope)
	}
	if explicitlyAuthorized {
		args = append(args, "--i-was-explicitly-authorized")
	}
	return args
}

func resourceCommand(runner domain.RunnerType) string {
	return "resource-" + string(runner)
}

func writeTemporaryDocument(data []byte) (string, error) {
	file, err := os.CreateTemp("", "agent-manager-permission-policy-*.json")
	if err != nil {
		return "", err
	}
	path := file.Name()
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		os.Remove(path)
		return "", err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		os.Remove(path)
		return "", err
	}
	if err := file.Close(); err != nil {
		os.Remove(path)
		return "", err
	}
	return path, nil
}

type resourcePermissionResponse struct {
	SchemaVersion      string   `json:"schema_version"`
	Runner             string   `json:"runner"`
	Scope              string   `json:"scope"`
	DesiredDigest      string   `json:"desired_digest"`
	DesiredFingerprint string   `json:"desired_fingerprint"`
	LiveFingerprint    string   `json:"live_fingerprint"`
	Drift              bool     `json:"drift"`
	Changes            []string `json:"changes"`
	NativePaths        []string `json:"native_paths"`
	Enforcement        struct {
		Permissions string   `json:"permissions"`
		Caveats     []string `json:"caveats"`
	} `json:"enforcement"`
}

func parseResourcePermissionResponse(data []byte) (resourcePermissionResponse, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var response resourcePermissionResponse
	if err := decoder.Decode(&response); err != nil {
		return resourcePermissionResponse{}, fmt.Errorf("parse JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return resourcePermissionResponse{}, errors.New("multiple JSON values")
		}
		return resourcePermissionResponse{}, fmt.Errorf("parse trailing JSON: %w", err)
	}
	if err := response.validate(); err != nil {
		return resourcePermissionResponse{}, err
	}
	return response, nil
}

func (r resourcePermissionResponse) validate() error {
	if r.SchemaVersion != resourcePermissionSchemaVersion {
		return fmt.Errorf("schema_version must be %q", resourcePermissionSchemaVersion)
	}
	if !domain.RunnerType(r.Runner).IsValid() {
		return fmt.Errorf("runner %q is unsupported", r.Runner)
	}
	if !validScope(r.Scope) {
		return fmt.Errorf("scope %q is unsupported", r.Scope)
	}
	for field, value := range map[string]string{
		"desired_digest": r.DesiredDigest, "desired_fingerprint": r.DesiredFingerprint,
		"live_fingerprint": r.LiveFingerprint,
	} {
		if strings.TrimSpace(value) == "" || strings.TrimSpace(value) != value {
			return fmt.Errorf("%s is required", field)
		}
	}
	if _, err := hex.DecodeString(r.DesiredDigest); err != nil || len(r.DesiredDigest) != sha256.Size*2 {
		return errors.New("desired_digest must be a SHA-256 hex digest")
	}
	switch r.Enforcement.Permissions {
	case "native", "hook_backed", "intent_only":
	default:
		return fmt.Errorf("enforcement.permissions %q is unsupported", r.Enforcement.Permissions)
	}
	if len(r.NativePaths) == 0 {
		return errors.New("native_paths must not be empty")
	}
	for _, path := range r.NativePaths {
		if strings.TrimSpace(path) == "" || strings.TrimSpace(path) != path {
			return errors.New("native_paths must contain trimmed paths")
		}
	}
	for _, change := range r.Changes {
		if strings.TrimSpace(change) == "" || strings.TrimSpace(change) != change {
			return errors.New("changes must contain trimmed descriptions")
		}
	}
	return nil
}

func (r resourcePermissionResponse) toProjectionResult() ProjectionResult {
	nativePaths := append([]string(nil), r.NativePaths...)
	sort.Strings(nativePaths)
	return ProjectionResult{
		Runner:             domain.RunnerType(r.Runner),
		Scope:              r.Scope,
		DesiredDigest:      r.DesiredDigest,
		DesiredFingerprint: r.DesiredFingerprint,
		LiveFingerprint:    r.LiveFingerprint,
		Drift:              r.Drift,
		Changes:            append([]string(nil), r.Changes...),
		NativePaths:        nativePaths,
		Enforcement:        EnforcementPosture{Permissions: r.Enforcement.Permissions, Caveats: append([]string(nil), r.Enforcement.Caveats...)},
	}
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
