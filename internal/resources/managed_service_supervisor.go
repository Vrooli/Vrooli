package resources

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/process"
	runtimestorage "github.com/vrooli/vrooli/internal/resources/runtime/storage"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

const managedServiceStateFile = "managed-service.json"

const managedServiceOwnershipTokenEnv = "VROOLI_MANAGED_SERVICE_INSTANCE_TOKEN"

// ManagedServiceState is the durable, credential-free identity record for a
// locally supervised service. Combined with OwnershipToken verification, it
// identifies the process Vrooli started; it is never inferred from a port,
// process name, or an arbitrary external endpoint.
type ManagedServiceState struct {
	InstanceID              string     `json:"instance_id"`
	OwnershipToken          string     `json:"ownership_token"`
	PID                     int        `json:"pid"`
	ArtifactPath            string     `json:"artifact_path"`
	ArtifactSHA256          string     `json:"artifact_sha256"`
	ArtifactVersion         string     `json:"artifact_version"`
	PreviousArtifactSHA256  string     `json:"previous_artifact_sha256,omitempty"`
	PreviousArtifactVersion string     `json:"previous_artifact_version,omitempty"`
	StartedAt               time.Time  `json:"started_at"`
	StoppedAt               *time.Time `json:"stopped_at,omitempty"`
}

// ManagedServiceSupervisor owns one private service process. It uses direct
// argv execution, not a shell, and keeps its control state separate from the
// resource's durable service data.
type ManagedServiceSupervisor struct {
	statePath string
	logPath   string
	now       func() time.Time
	start     func(*exec.Cmd) error
	isRunning func(int) bool
	terminate func(int) error
}

func newManagedServiceSupervisor(statePath, logPath string) *ManagedServiceSupervisor {
	return &ManagedServiceSupervisor{
		statePath: statePath,
		logPath:   logPath,
		now:       time.Now,
		start:     func(cmd *exec.Cmd) error { return cmd.Start() },
		isRunning: process.IsPIDRunning,
		terminate: terminateCompanion,
	}
}

func managedServiceSupervisorFor(resource string) (*ManagedServiceSupervisor, runtimestorage.Paths, error) {
	resolver, err := runtimestorage.NewResolver(runtimestorage.ResolverConfig{AppID: "vrooli"})
	if err != nil {
		return nil, runtimestorage.Paths{}, err
	}
	paths, err := runtimestorage.EnsureAllDirs(resolver, runtimestorage.Options{ResourceID: resource}, 0o700)
	if err != nil {
		return nil, runtimestorage.Paths{}, err
	}
	return newManagedServiceSupervisor(filepath.Join(paths.StateDir, managedServiceStateFile), filepath.Join(paths.LogsDir, "service.log")), paths, nil
}

func (s *ManagedServiceSupervisor) Status() (ManagedServiceState, bool, error) {
	state, err := s.readState()
	if err != nil {
		if os.IsNotExist(err) {
			return ManagedServiceState{}, false, nil
		}
		return ManagedServiceState{}, false, err
	}
	if state.PID <= 0 || !s.isRunning(state.PID) {
		return state, false, nil
	}
	if err := verifyManagedServiceOwnership(state); err != nil {
		return state, false, err
	}
	return state, true, nil
}

// Start verifies and starts an artifact only when no live process is already
// recorded. A stale record is atomically replaced; a live record never changes
// provider, version, or arguments behind the operator's back.
func (s *ManagedServiceSupervisor) Start(artifactPath string, artifact resourcedeployment.ServiceArtifact, args, env []string, workingDir string) (ManagedServiceState, error) {
	if err := artifact.VerifyFile(artifactPath); err != nil {
		return ManagedServiceState{}, err
	}
	existing, running, err := s.Status()
	if err != nil {
		return ManagedServiceState{}, fmt.Errorf("read managed-service state: %w", err)
	}
	if running {
		return existing, nil
	}
	if err := os.MkdirAll(filepath.Dir(s.logPath), 0o700); err != nil {
		return ManagedServiceState{}, fmt.Errorf("create managed-service log directory: %w", err)
	}
	logFile, err := os.OpenFile(s.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return ManagedServiceState{}, fmt.Errorf("open managed-service log: %w", err)
	}
	defer logFile.Close()

	ownershipToken, err := newManagedServiceToken()
	if err != nil {
		return ManagedServiceState{}, err
	}
	cmd := exec.Command(artifactPath, args...)
	cmd.Env = setEnvValue(env, managedServiceOwnershipTokenEnv, ownershipToken)
	cmd.Dir = workingDir
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = detachSysProcAttr()
	if err := s.start(cmd); err != nil {
		return ManagedServiceState{}, fmt.Errorf("start managed-service artifact: %w", err)
	}
	instanceID := existing.InstanceID
	if instanceID == "" {
		instanceID, err = newManagedServiceInstanceID()
		if err != nil {
			_ = s.terminate(cmd.Process.Pid)
			return ManagedServiceState{}, err
		}
	}
	state := ManagedServiceState{
		InstanceID:              instanceID,
		OwnershipToken:          ownershipToken,
		PID:                     cmd.Process.Pid,
		ArtifactPath:            artifactPath,
		ArtifactSHA256:          strings.ToLower(strings.TrimSpace(artifact.SHA256)),
		ArtifactVersion:         artifact.Version,
		PreviousArtifactSHA256:  existing.ArtifactSHA256,
		PreviousArtifactVersion: existing.ArtifactVersion,
		StartedAt:               s.now().UTC(),
	}
	// The service intentionally outlives the short-lived control CLI.
	_ = cmd.Process.Release()
	if err := s.writeState(state); err != nil {
		_ = s.terminate(state.PID)
		return ManagedServiceState{}, fmt.Errorf("persist managed-service state: %w", err)
	}
	return state, nil
}

// Stop asks the process to exit gracefully and waits until the caller's
// deadline. It retains a stopped record so the next verified launch preserves
// the instance identity and upgrade lineage.
func (s *ManagedServiceSupervisor) Stop(ctx context.Context) error {
	state, running, err := s.Status()
	if err != nil {
		return fmt.Errorf("read managed-service state: %w", err)
	}
	if !running {
		return nil
	}
	if err := s.terminate(state.PID); err != nil {
		return fmt.Errorf("stop managed-service process: %w", err)
	}
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for s.isRunning(state.PID) {
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for managed-service shutdown: %w", ctx.Err())
		case <-ticker.C:
		}
	}
	now := s.now().UTC()
	state.PID = 0
	state.StoppedAt = &now
	if err := s.writeState(state); err != nil {
		return fmt.Errorf("persist stopped managed-service state: %w", err)
	}
	return nil
}

func (s *ManagedServiceSupervisor) Logs(out io.Writer) error {
	file, err := os.Open(s.logPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open managed-service log: %w", err)
	}
	defer file.Close()
	_, err = io.Copy(out, file)
	return err
}

func (s *ManagedServiceSupervisor) readState() (ManagedServiceState, error) {
	data, err := os.ReadFile(s.statePath)
	if err != nil {
		return ManagedServiceState{}, err
	}
	var state ManagedServiceState
	if err := json.Unmarshal(data, &state); err != nil {
		return ManagedServiceState{}, fmt.Errorf("parse managed-service state: %w", err)
	}
	if state.PID < 0 || strings.TrimSpace(state.InstanceID) == "" || strings.TrimSpace(state.OwnershipToken) == "" || strings.TrimSpace(state.ArtifactSHA256) == "" || strings.TrimSpace(state.ArtifactVersion) == "" {
		return ManagedServiceState{}, fmt.Errorf("managed-service state is incomplete")
	}
	return state, nil
}

func newManagedServiceInstanceID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate managed-service identity: %w", err)
	}
	return fmt.Sprintf("ms-%x", value), nil
}

func newManagedServiceToken() (string, error) {
	var value [32]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate managed-service ownership token: %w", err)
	}
	return fmt.Sprintf("mst-%x", value), nil
}

func verifyManagedServiceOwnership(state ManagedServiceState) error {
	env, err := process.ReadEnvironment(state.PID)
	if err != nil {
		return fmt.Errorf("verify managed-service process ownership: %w", err)
	}
	if env[managedServiceOwnershipTokenEnv] != state.OwnershipToken {
		return fmt.Errorf("managed-service process ownership token does not match")
	}
	return nil
}

func (s *ManagedServiceSupervisor) writeState(state ManagedServiceState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return runtimestorage.WriteFileAtomic(s.statePath, data, 0o600)
}

func managedServiceArtifactPath(controller *Controller, manifest ResourceManifest) (string, error) {
	if manifest.ManagedService == nil {
		return "", fmt.Errorf("managed_service is required")
	}
	artifact := manifest.ManagedService.Artifact
	if err := artifact.Validate(); err != nil {
		return "", err
	}
	// A bundled server is delivered with the signed Vrooli release, rather
	// than the source checkout. This keeps an installed control plane usable
	// with VROOLI_INSTALL_SOURCE=0 and makes the launch path independent of a
	// mutable working tree. The checksum is still verified immediately before
	// execution by the driver and supervisor.
	if artifact.BundleArtifact != "" {
		name, err := artifact.BundleArtifactForPlatform(runtime.GOOS, runtime.GOARCH)
		if err != nil {
			return "", err
		}
		root := strings.TrimSpace(os.Getenv("VROOLI_RESOURCE_ARTIFACT_DIR"))
		if root == "" {
			root = filepath.Join(controller.Home, ".vrooli", "artifacts")
		}
		if !filepath.IsAbs(root) {
			return "", fmt.Errorf("managed-service artifact root must be absolute")
		}
		return filepath.Join(filepath.Clean(root), manifest.Name, artifact.Version, name), nil
	}
	root := filepath.Join(controller.Root, "resources", manifest.Name)
	path := filepath.Join(root, filepath.FromSlash(artifact.Path))
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("managed-service artifact path escapes resource root")
	}
	return path, nil
}

func managedServiceStopContext(parent context.Context, manifest ResourceManifest) (context.Context, context.CancelFunc) {
	seconds := manifest.Lifecycle.StopTimeoutSeconds
	if seconds <= 0 {
		seconds = 30
	}
	return context.WithTimeout(parent, time.Duration(seconds)*time.Second)
}
