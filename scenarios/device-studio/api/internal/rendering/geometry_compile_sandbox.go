package rendering

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"device-studio/internal/geometry"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/discovery"
)

const geometrySandboxTimeout = 90 * time.Second
const geometrySandboxWorkspaceAlias = "/workspace"

var resolveGeometrySandbox = func(ctx context.Context) (string, error) {
	return discovery.ResolveScenarioURLDefault(ctx, "workspace-sandbox")
}

// RunGeometryCompileManifest is the fixed deterministic stage run inside the
// protected workspace-sandbox process. It reads only the fixed manifest and
// canonical recipe names from the attempt workspace and writes one bounded GLB.
func RunGeometryCompileManifest(ctx context.Context, manifestPath string) error {
	if ctx == nil || !filepath.IsAbs(manifestPath) || filepath.Clean(manifestPath) != manifestPath || filepath.Base(manifestPath) != GeometryManifestName {
		return errors.New("geometry compile worker requires a fixed manifest in its owned workspace")
	}
	ctx, cancel := context.WithTimeout(ctx, geometrySandboxTimeout)
	defer cancel()
	root, err := os.OpenRoot(filepath.Dir(manifestPath))
	if err != nil {
		return err
	}
	defer root.Close()
	manifestBytes, err := readGeometryFile(ctx, root, GeometryManifestName, 64<<10)
	if err != nil {
		return err
	}
	var manifest GeometryWorkerManifest
	decoder := json.NewDecoder(bytes.NewReader(manifestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return err
	}
	canonicalManifest, err := json.Marshal(manifest)
	if err != nil || !bytes.Equal(manifestBytes, append(canonicalManifest, '\n')) || manifest.Version != 1 || !validWorkerDigest(manifest.WorkerSHA256) {
		return errors.New("geometry compile manifest is not canonical")
	}
	if err := verifyCurrentExecutable(manifest.WorkerSHA256); err != nil {
		return err
	}
	recipeBytes, err := readGeometryFile(ctx, root, GeometryRecipeName, geometry.MaxRecipeBytes)
	if err != nil {
		return err
	}
	if len(recipeBytes) > 0 && recipeBytes[len(recipeBytes)-1] == '\n' {
		recipeBytes = recipeBytes[:len(recipeBytes)-1]
	}
	recipe, canonicalRecipe, err := geometry.Decode(recipeBytes)
	if err != nil || !bytes.Equal(canonicalRecipe, recipeBytes) || recipe.Operation != "construct" {
		return errors.New("geometry compile recipe is not a canonical construct recipe")
	}
	for i := range manifest.Envelope {
		if recipe.Envelope[i] != manifest.Envelope[i] {
			return errors.New("geometry compile recipe changed its locked envelope")
		}
	}
	glb, err := geometry.Compile(recipe)
	if err != nil {
		return err
	}
	inventory, err := InspectPhysicalGLB(glb)
	if err != nil {
		return fmt.Errorf("compiled geometry failed physical GLB inspection: %w", err)
	}
	if len(glb) > geometry.MaxGLBBytes || inventory.TriangleCount > geometry.MaxTriangles {
		return errors.New("compiled geometry exceeds output bounds")
	}
	return writeGeometryFile(root, GeometryOutputName, glb)
}

type geometrySandboxCreate struct {
	ScopePath      string `json:"scopePath"`
	ProjectRoot    string `json:"projectRoot"`
	NoLock         *bool  `json:"noLock"`
	Owner          string `json:"owner"`
	IdempotencyKey string `json:"idempotencyKey"`
}
type geometrySandbox struct {
	ID            string               `json:"id"`
	WorkspacePath string               `json:"workspacePath"`
	Containment   *geometryContainment `json:"containment"`
}
type geometryContainment struct {
	Level        string   `json:"level"`
	Backend      string   `json:"backend"`
	Enforcements []string `json:"enforcements"`
}
type geometrySandboxExec struct {
	Command        string            `json:"command"`
	Args           []string          `json:"args"`
	Env            map[string]string `json:"env,omitempty"`
	IsolationLevel string            `json:"isolationLevel"`
	ExecutionMode  string            `json:"executionMode"`
	AllowNetwork   *bool             `json:"allowNetwork"`
	WorkingDir     string            `json:"workingDir"`
	MemoryLimitMB  int               `json:"memoryLimitMB"`
	CPUTimeSec     int               `json:"cpuTimeSec"`
	TimeoutSec     int               `json:"timeoutSec"`
	MaxProcesses   int               `json:"maxProcesses"`
	MaxOpenFiles   int               `json:"maxOpenFiles"`
}
type geometrySandboxExecResult struct {
	ExitCode    int                  `json:"exitCode"`
	Stderr      string               `json:"stderr"`
	TimedOut    bool                 `json:"timedOut"`
	Containment *geometryContainment `json:"containment"`
}
type geometrySandboxFile struct {
	Encoding string `json:"encoding"`
	Content  string `json:"content"`
	Size     int64  `json:"size"`
}

type geometrySandboxKillAllResult struct {
	Killed int      `json:"killed"`
	Errors []string `json:"errors"`
}

type geometrySandboxProcesses struct {
	Running int `json:"running"`
}

func runGeometryCompileSandbox(ctx context.Context, workspace string, manifest GeometryWorkerManifest) (retErr error) {
	if !filepath.IsAbs(workspace) || filepath.Clean(workspace) != workspace {
		return errors.New("geometry compile workspace path is invalid")
	}
	base, err := resolveGeometrySandbox(ctx)
	if err != nil || base == "" {
		return errors.New("workspace-sandbox is unavailable for bounded geometry compilation")
	}
	client := &http.Client{Timeout: geometrySandboxTimeout}
	noLock := true
	// A sandbox is disposable and is deleted after the run. Include a fresh
	// operation nonce so a later retry cannot be handed an earlier deleted
	// sandbox by WSS's durable idempotency table.
	key := "device-studio-geometry-compile-" + manifest.AttemptID + "-" + uuid.NewString()
	create := geometrySandboxCreate{ScopePath: workspace, ProjectRoot: workspace, NoLock: &noLock, Owner: "device-studio-geometry-compile", IdempotencyKey: key}
	var sandbox geometrySandbox
	if err := geometrySandboxJSON(ctx, client, http.MethodPost, base+"/api/v1/sandboxes", create, &sandbox); err != nil {
		return fmt.Errorf("workspace-sandbox create failed: %w", err)
	}
	if sandbox.ID != "" {
		// Install cleanup as soon as the owner returns an addressable sandbox
		// identity. Even malformed ancillary metadata must not leak that scope.
		defer func() {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			cleanupErr := cleanupGeometrySandbox(cleanupCtx, client, base, sandbox.ID)
			if cleanupErr != nil {
				retErr = errors.Join(retErr, fmt.Errorf("workspace-sandbox cleanup failed; execution may remain uncertain: %w", cleanupErr))
			}
		}()
	}
	if sandbox.ID == "" {
		return errors.New("workspace-sandbox returned no sandbox identity; the created scope cannot be safely targeted for cleanup")
	}
	if !validGeometryJobID(sandbox.ID) {
		return errors.New("workspace-sandbox returned an invalid sandbox identity; cleanup was attempted")
	}
	if sandbox.WorkspacePath == "" {
		return errors.New("workspace-sandbox returned no workspace path")
	}
	if sandbox.Containment == nil || sandbox.Containment.Backend == "" || sandbox.Containment.Backend == "none" {
		return errors.New("workspace-sandbox did not provide an active containment backend")
	}
	// workspacePath is the host-side merged directory. The full isolation
	// profile exposes it inside bwrap at the stable /workspace alias, not at
	// that host-side absolute path.
	manifestPath := filepath.Join(geometrySandboxWorkspaceAlias, GeometryManifestName)
	workerPath := filepath.Join(geometrySandboxWorkspaceAlias, "worker")
	allowNetwork := false
	// The API executable links cgo-backed services and needs several runtime
	// threads even in deterministic worker mode. Keep a modest 32-task ceiling
	// (below the existing geometry owner limit of 128), pin Go parallelism to
	// two, and set a soft heap target. The 4 GiB RLIMIT_AS is virtual address
	// space for the Go runtime; it is not a physical-memory or aggregate-RSS cap.
	req := geometrySandboxExec{Command: workerPath, Args: []string{"geometry-compile-worker", "--manifest", manifestPath}, Env: map[string]string{"GOMAXPROCS": "2", "GOMEMLIMIT": "256MiB"}, IsolationLevel: "full", ExecutionMode: "protected", AllowNetwork: &allowNetwork, WorkingDir: geometrySandboxWorkspaceAlias, MemoryLimitMB: 4096, CPUTimeSec: 45, TimeoutSec: 60, MaxProcesses: 32, MaxOpenFiles: 64}
	var result geometrySandboxExecResult
	execURL := base + "/api/v1/sandboxes/" + url.PathEscape(sandbox.ID) + "/exec"
	if err := geometrySandboxJSON(ctx, client, http.MethodPost, execURL, req, &result); err != nil {
		return fmt.Errorf("workspace-sandbox protected geometry compile failed: %w", err)
	}
	if result.ExitCode != 0 || result.TimedOut {
		detail := strings.TrimSpace(result.Stderr)
		if len(detail) > 300 {
			detail = detail[:300]
		}
		if detail != "" {
			return fmt.Errorf("workspace-sandbox geometry compiler did not succeed (exit=%d timeout=%t): %s", result.ExitCode, result.TimedOut, detail)
		}
		return fmt.Errorf("workspace-sandbox geometry compiler did not succeed (exit=%d timeout=%t)", result.ExitCode, result.TimedOut)
	}
	if !hasEnforcement(result.Containment, "filesystem-write-containment") || !hasEnforcement(result.Containment, "network-deny") {
		return errors.New("workspace-sandbox did not enforce filesystem-write containment and network denial")
	}
	fileURL := base + "/api/v1/sandboxes/" + url.PathEscape(sandbox.ID) + "/files/content?path=" + url.QueryEscape("/"+GeometryOutputName)
	var file geometrySandboxFile
	if err := geometrySandboxJSON(ctx, client, http.MethodGet, fileURL, nil, &file); err != nil {
		return fmt.Errorf("workspace-sandbox output read failed: %w", err)
	}
	if file.Encoding != "base64" || file.Size < 1 || file.Size > geometry.MaxGLBBytes {
		return errors.New("workspace-sandbox returned geometry output outside fixed bounds")
	}
	decoded, err := base64.StdEncoding.DecodeString(file.Content)
	if err != nil || int64(len(decoded)) != file.Size {
		return errors.New("workspace-sandbox geometry output encoding or size is invalid")
	}
	root, err := os.OpenRoot(workspace)
	if err != nil {
		return err
	}
	defer root.Close()
	return writeGeometryFile(root, GeometryOutputName, decoded)
}

// cleanupGeometrySandbox first stops and observes the disposable sandbox's
// tracked processes before deleting its workspace. Exec is synchronous, but
// a cancelled HTTP request can return to this process before the remote owner
// has fully recorded termination. A cleanup error is therefore a failed and
// uncertain operation, never a successful compile receipt.
func cleanupGeometrySandbox(ctx context.Context, client *http.Client, base, sandboxID string) error {
	root := base + "/api/v1/sandboxes/" + url.PathEscape(sandboxID)
	var killed geometrySandboxKillAllResult
	if err := geometrySandboxJSON(ctx, client, http.MethodPost, root+"/processes/kill-all", struct{}{}, &killed); err != nil {
		return fmt.Errorf("stop owned sandbox processes: %w", err)
	}
	if len(killed.Errors) > 0 {
		return fmt.Errorf("stop owned sandbox processes returned %d error(s)", len(killed.Errors))
	}
	var last int
	for {
		var processes geometrySandboxProcesses
		if err := geometrySandboxJSON(ctx, client, http.MethodGet, root+"/processes?running=true", nil, &processes); err != nil {
			return fmt.Errorf("verify sandbox process quiescence: %w", err)
		}
		last = processes.Running
		if last == 0 {
			break
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("sandbox still reports %d running process(es): %w", last, ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
	if err := geometrySandboxJSON(ctx, client, http.MethodDelete, root, nil, nil); err != nil {
		return fmt.Errorf("delete quiescent sandbox: %w", err)
	}
	return nil
}

func hasEnforcement(c *geometryContainment, want string) bool {
	if c == nil {
		return false
	}
	for _, value := range c.Enforcements {
		if value == want {
			return true
		}
	}
	return false
}

func geometrySandboxJSON(ctx context.Context, client *http.Client, method, endpoint string, input, output any) error {
	var body io.Reader
	if input != nil {
		raw, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return err
	}
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil || len(data) > 1<<20 {
		return errors.New("workspace-sandbox response exceeded fixed bound")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(data))
		if len(message) > 400 {
			message = message[:400]
		}
		return fmt.Errorf("workspace-sandbox returned HTTP %d: %s", resp.StatusCode, message)
	}
	if output == nil || len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, output); err != nil {
		return fmt.Errorf("workspace-sandbox response is invalid JSON: %w", err)
	}
	return nil
}
