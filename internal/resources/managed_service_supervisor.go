package resources

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
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
	OwnershipTokenHash      string     `json:"ownership_token_hash"`
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
	forceStop func(int) error
	signal    func(int, os.Signal) error
}

func newManagedServiceSupervisor(statePath, logPath string) *ManagedServiceSupervisor {
	return &ManagedServiceSupervisor{
		statePath: statePath,
		logPath:   logPath,
		now:       time.Now,
		start:     func(cmd *exec.Cmd) error { return cmd.Start() },
		isRunning: process.IsPIDRunning,
		terminate: terminateCompanion,
		forceStop: func(pid int) error { return platform.KillProcess(pid, true) },
		signal:    platform.SignalPIDWithSignal,
	}
}

func managedServiceSupervisorFor(resource string) (*ManagedServiceSupervisor, runtimestorage.Paths, error) {
	resolver, err := resourceStorageResolver()
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
func (s *ManagedServiceSupervisor) Start(artifactPath string, artifact resourcedeployment.ServiceArtifact, args, env []string, workingDir string, limits *resourcedeployment.ProcessLimits) (ManagedServiceState, error) {
	if err := artifact.VerifyFile(artifactPath); err != nil {
		return ManagedServiceState{}, err
	}
	launchPath, err := artifact.LaunchPath(artifactPath)
	if err != nil {
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
	cmd := exec.Command(launchPath, args...)
	cmd.Env = setEnvValue(env, managedServiceOwnershipTokenEnv, ownershipToken)
	if strings.EqualFold(strings.TrimSpace(artifact.Layout), "dir") {
		cmd.Dir = artifactPath
	} else {
		cmd.Dir = workingDir
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := platform.ConfigureCommand(cmd, platform.ProcessOptions{Detached: true}); err != nil {
		return ManagedServiceState{}, err
	}
	if err := s.start(cmd); err != nil {
		return ManagedServiceState{}, fmt.Errorf("start managed-service artifact: %w", err)
	}
	if err := applyManagedServiceProcessLimits(cmd.Process.Pid, limits); err != nil {
		_ = s.terminate(cmd.Process.Pid)
		return ManagedServiceState{}, fmt.Errorf("apply managed-service process limits: %w", err)
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
		OwnershipTokenHash:      managedServiceTokenHash(ownershipToken),
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

// Stop asks the process to exit with the historical graceful termination
// request and waits until the caller's deadline. If the process is still alive
// at that bounded deadline, it escalates to forceful termination and retains
// the state record unless the process is observed stopped.
func (s *ManagedServiceSupervisor) Stop(ctx context.Context) error {
	return s.stop(ctx, nil)
}

// StopWithSignal uses the manifest-selected first shutdown request. The
// supervisor owns escalation and durable state reconciliation; callers never
// need to decide whether a timeout means "stopped".
func (s *ManagedServiceSupervisor) StopWithSignal(ctx context.Context, signal os.Signal) error {
	return s.stop(ctx, signal)
}

func (s *ManagedServiceSupervisor) stop(ctx context.Context, signal os.Signal) error {
	state, running, err := s.Status()
	if err != nil {
		return fmt.Errorf("read managed-service state: %w", err)
	}
	if !running {
		return nil
	}
	if signal != nil {
		if err := s.signal(state.PID, signal); err != nil {
			return fmt.Errorf("stop managed-service process: %w", err)
		}
	} else if err := s.terminate(state.PID); err != nil {
		return fmt.Errorf("stop managed-service process: %w", err)
	}
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for s.isRunning(state.PID) {
		select {
		case <-ctx.Done():
			if !s.isRunning(state.PID) {
				break
			}
			if err := s.forceStop(state.PID); err != nil && s.isRunning(state.PID) {
				return fmt.Errorf("wait for managed-service shutdown: %w; forceful escalation failed: %v", ctx.Err(), err)
			}
			forceCtx, forceCancel := context.WithTimeout(context.Background(), 2*time.Second)
			forceErr := s.waitUntilStopped(forceCtx, state.PID)
			forceCancel()
			if forceErr != nil {
				return fmt.Errorf("wait for managed-service shutdown: %w; process remained alive after forceful escalation: %v", ctx.Err(), forceErr)
			}
			break
		case <-ticker.C:
		}
		if !s.isRunning(state.PID) {
			break
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

func (s *ManagedServiceSupervisor) waitUntilStopped(ctx context.Context, pid int) error {
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for s.isRunning(pid) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
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

// Attest creates non-secret broker evidence only after checking the live
// process environment against the persisted token hash. A caller must supply
// the authenticated local control capability that will manage this instance.
func (s *ManagedServiceSupervisor) Attest(endpoint, controlCapability string) (OwnershipAttestation, error) {
	state, running, err := s.Status()
	if err != nil {
		return OwnershipAttestation{}, fmt.Errorf("attest managed-service ownership: %w", err)
	}
	if !running {
		return OwnershipAttestation{}, fmt.Errorf("attest managed-service ownership: service is not running")
	}
	if !isLoopbackManagedEndpoint(endpoint) || strings.TrimSpace(controlCapability) == "" {
		return OwnershipAttestation{}, fmt.Errorf("managed-service attestation requires loopback endpoint and local control capability")
	}
	now := s.now().UTC()
	proof := managedServiceAttestationProof(state, endpoint, controlCapability)
	return OwnershipAttestation{InstanceID: state.InstanceID, ArtifactSHA256: state.ArtifactSHA256, Endpoint: endpoint, ControlCapability: controlCapability, IssuedAt: now, ExpiresAt: now.Add(30 * 24 * time.Hour), Proof: proof}, nil
}

func managedServiceAttestationProof(state ManagedServiceState, endpoint, controlCapability string) string {
	value := strings.Join([]string{state.InstanceID, state.ArtifactSHA256, endpoint, controlCapability, state.OwnershipTokenHash}, "\x00")
	return managedServiceTokenHash(value)
}

func verifyManagedServiceAttestation(resource string, attestation OwnershipAttestation) error {
	if strings.TrimSpace(attestation.InstanceID) == "" || strings.TrimSpace(attestation.ArtifactSHA256) == "" || !isLoopbackManagedEndpoint(attestation.Endpoint) || strings.TrimSpace(attestation.ControlCapability) == "" || strings.TrimSpace(attestation.Proof) == "" || attestation.IssuedAt.IsZero() || attestation.ExpiresAt.IsZero() {
		return fmt.Errorf("managed-service ownership attestation is incomplete")
	}
	now := time.Now()
	if !attestation.ExpiresAt.After(now) || attestation.IssuedAt.After(now.Add(time.Minute)) || attestation.ExpiresAt.Sub(attestation.IssuedAt) > 31*24*time.Hour {
		return fmt.Errorf("managed-service ownership attestation is stale or invalid")
	}
	supervisor, _, err := managedServiceSupervisorFor(resource)
	if err != nil {
		return err
	}
	state, running, err := supervisor.Status()
	if err != nil {
		return fmt.Errorf("attested managed-service is not running: %w", err)
	}
	if !running {
		return fmt.Errorf("attested managed-service is not running")
	}
	if state.InstanceID != attestation.InstanceID || state.ArtifactSHA256 != attestation.ArtifactSHA256 || managedServiceAttestationProof(state, attestation.Endpoint, attestation.ControlCapability) != attestation.Proof {
		return fmt.Errorf("managed-service ownership attestation does not match supervisor state")
	}
	return nil
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
	if state.PID < 0 || strings.TrimSpace(state.InstanceID) == "" || strings.TrimSpace(state.OwnershipTokenHash) == "" || strings.TrimSpace(state.ArtifactSHA256) == "" || strings.TrimSpace(state.ArtifactVersion) == "" {
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
	if managedServiceTokenHash(env[managedServiceOwnershipTokenEnv]) == state.OwnershipTokenHash {
		return nil
	}
	// Some otherwise legitimate managed services deliberately clear their
	// environment after startup (Redis does this when proc-title rewriting is
	// enabled). In that case, retain a process-identity proof: the live
	// executable must still be the verified artifact recorded by the supervisor.
	// This does not accept an arbitrary PID, port, or process name, and it also
	// keeps PID reuse from being mistaken for ownership because the supervisor
	// state is tied to its recorded artifact path.
	if managedServiceExecutableMatchesArtifact(state) {
		return nil
	}
	return fmt.Errorf("managed-service process ownership token does not match")
}

func managedServiceExecutableMatchesArtifact(state ManagedServiceState) bool {
	if state.PID <= 0 || strings.TrimSpace(state.ArtifactPath) == "" {
		return false
	}
	if runtime.GOOS != "linux" {
		return false
	}
	executable, err := os.Readlink(filepath.Join("/proc", strconv.Itoa(state.PID), "exe"))
	if err != nil {
		return false
	}
	executable = filepath.Clean(executable)
	artifact := filepath.Clean(state.ArtifactPath)
	if executable == artifact {
		return true
	}
	relative, err := filepath.Rel(artifact, executable)
	return err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

func managedServiceTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", sum[:])
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
	if strings.EqualFold(strings.TrimSpace(artifact.Verification), "host-tool") {
		path, err := exec.LookPath(strings.TrimSpace(artifact.Path))
		if err != nil {
			return "", fmt.Errorf("locate managed-service host tool %q: %w", artifact.Path, err)
		}
		return filepath.Abs(path)
	}
	// Acquired servers live in the immutable per-user artifact store. This is
	// also the default when a manifest does not need a separate desktop bundle
	// filename; the resource checkout is never treated as an install location.
	if manifest.ManagedService.Acquisition != nil && artifact.BundleArtifact == "" {
		root, err := managedServiceArtifactStoreRoot(controller.Home)
		if err != nil {
			return "", err
		}
		name := filepath.Base(filepath.FromSlash(artifact.Path))
		if name == "." || name == string(filepath.Separator) || name == "" {
			return "", fmt.Errorf("managed-service artifact path has no install name")
		}
		return filepath.Join(root, manifest.Name, artifact.Version, name), nil
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
		root, err := managedServiceArtifactStoreRoot(controller.Home)
		if err != nil {
			return "", err
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

func managedServiceArtifactStoreRoot(home string) (string, error) {
	root := strings.TrimSpace(os.Getenv("VROOLI_RESOURCE_ARTIFACT_DIR"))
	if root == "" {
		runtimeHome, err := repocontract.VrooliUserRoot(home)
		if err != nil {
			return "", fmt.Errorf("resolve runtime home: %w", err)
		}
		root = filepath.Join(runtimeHome, "artifacts")
	}
	if !filepath.IsAbs(root) {
		return "", fmt.Errorf("managed-service artifact root must be absolute")
	}
	return filepath.Clean(root), nil
}

func managedServiceStopContext(parent context.Context, manifest ResourceManifest) (context.Context, context.CancelFunc) {
	seconds := manifest.Lifecycle.StopTimeoutSeconds
	if seconds <= 0 {
		seconds = 30
	}
	return context.WithTimeout(parent, time.Duration(seconds)*time.Second)
}
