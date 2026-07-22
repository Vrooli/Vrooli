package bundleruntime

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/api"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/infra"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/secrets"
)

// =============================================================================
// Service Status
// =============================================================================

// setStatus updates the status for a service.
func (s *Supervisor) setStatus(id string, status ServiceStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.clock.Now()
	if prev, ok := s.serviceStatus[id]; ok {
		if status.StartedAt.IsZero() {
			status.StartedAt = prev.StartedAt
		}
		if status.ReadyAt.IsZero() {
			status.ReadyAt = prev.ReadyAt
		}
	}
	if status.StartedAt.IsZero() {
		if proc, ok := s.procs[id]; ok && !proc.started.IsZero() {
			status.StartedAt = proc.started
		}
	}
	if status.Ready && status.ReadyAt.IsZero() {
		status.ReadyAt = now
	}
	status.UpdatedAt = now
	s.serviceStatus[id] = status
}

// getStatus retrieves the status for a service.
func (s *Supervisor) getStatus(id string) (ServiceStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st, ok := s.serviceStatus[id]
	return st, ok
}

// setProc stores a service process reference.
func (s *Supervisor) setProc(id string, proc *serviceProcess) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.procs[id] = proc
}

// getProc retrieves a service process reference.
func (s *Supervisor) getProc(id string) *serviceProcess {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.procs[id]
}

// =============================================================================
// Auth Token
// =============================================================================

// loadOrCreateToken loads an existing auth token or creates a new one.
func (s *Supervisor) loadOrCreateToken(path string) (string, error) {
	if data, err := s.fs.ReadFile(path); err == nil && len(strings.TrimSpace(string(data))) > 0 {
		return strings.TrimSpace(string(data)), nil
	}

	if err := s.fs.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}

	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)
	if err := s.fs.WriteFile(path, []byte(token), 0o600); err != nil {
		return "", err
	}
	return token, nil
}

// =============================================================================
// Public Accessors
// =============================================================================

// AppDataDir returns the resolved app data directory.
func (s *Supervisor) AppDataDir() string {
	return s.appData
}

// TelemetryPath returns the telemetry file path.
func (s *Supervisor) TelemetryPath() string {
	return s.telemetryPath
}

// TelemetryUploadURL returns the telemetry upload URL.
func (s *Supervisor) TelemetryUploadURL() string {
	return s.opts.Manifest.Telemetry.UploadTo
}

// Manifest returns the bundle manifest.
func (s *Supervisor) Manifest() *manifest.Manifest {
	return s.opts.Manifest
}

// FileSystem returns the file system abstraction.
func (s *Supervisor) FileSystem() infra.FileSystem {
	return s.fs
}

// SecretStore returns the secret store for API interactions.
func (s *Supervisor) SecretStore() api.SecretStore {
	return s.secretStore
}

// StartServicesIfReady triggers service startup if secrets are ready.
func (s *Supervisor) StartServicesIfReady() {
	if !s.servicesStarted {
		s.startServicesAsync()
	}
}

// RecordTelemetry records a telemetry event (public interface for api package).
func (s *Supervisor) RecordTelemetry(event string, details map[string]interface{}) error {
	return s.recordTelemetry(event, details)
}

// AuthToken returns the current auth token for the control API.
func (s *Supervisor) AuthToken() string {
	return s.authToken
}

// IsStarted returns whether the supervisor has been started.
func (s *Supervisor) IsStarted() bool {
	return s.started
}

// AllServicesReady returns true if all services are ready.
func (s *Supervisor) AllServicesReady() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	hasReadyCandidate := false
	for _, st := range s.serviceStatus {
		if st.Skipped {
			continue
		}
		hasReadyCandidate = true
		if !st.Ready {
			return false
		}
	}
	return hasReadyCandidate
}

// ServiceStatuses returns a copy of all service statuses.
func (s *Supervisor) ServiceStatuses() map[string]ServiceStatus {
	s.mu.RLock()
	out := make(map[string]ServiceStatus)
	for id, st := range s.serviceStatus {
		out[id] = st
	}
	resourceServer := s.resourceServer
	s.mu.RUnlock()
	if resourceServer != nil {
		for resource, status := range resourceServer.Statuses() {
			out["resource:"+resource] = ServiceStatus{Ready: status.Running, Message: status.Message}
		}
	}
	return out
}

// ResourceLogPath returns a managed resource log only when the resource was
// selected by this bundle's verified deployment plan. It deliberately does not
// accept an arbitrary filesystem path from the control API.
func (s *Supervisor) ResourceLogPath(resource string) (string, bool) {
	s.mu.RLock()
	resourceServer := s.resourceServer
	s.mu.RUnlock()
	if resourceServer == nil {
		return "", false
	}
	status, ok := resourceServer.Statuses()[resource]
	return status.LogPath, ok && status.LogPath != ""
}

// recordTelemetry writes a telemetry event if the recorder is initialized.
func (s *Supervisor) recordTelemetry(event string, details map[string]interface{}) error {
	if s.telemetry == nil {
		return nil
	}
	return s.telemetry.Record(event, details)
}

// =============================================================================
// Secret Management
// =============================================================================

// applySecrets injects secrets into the environment for a service.
func (s *Supervisor) applySecrets(env map[string]string, svc manifest.Service) error {
	injector := secrets.NewInjector(s.secretStore, s.fs, s.appData)
	return injector.Apply(env, svc)
}

// UpdateSecrets merges new secrets and persists them.
// Triggers service startup if all required secrets are now available.
func (s *Supervisor) UpdateSecrets(newSecrets map[string]string) error {
	merged := s.secretStore.Merge(newSecrets)

	missing := s.secretStore.MissingRequiredFrom(merged)
	if len(missing) > 0 {
		_ = s.recordTelemetry("secrets_missing", map[string]interface{}{"missing": missing})
		return fmt.Errorf("missing required secrets: %s", strings.Join(missing, ", "))
	}

	if err := s.secretStore.Persist(merged); err != nil {
		return err
	}

	s.secretStore.Set(merged)

	_ = s.recordTelemetry("secrets_updated", map[string]interface{}{"count": len(merged)})

	// Trigger service startup if not already started.
	if !s.servicesStarted {
		s.startServicesAsync()
	}
	return nil
}

// =============================================================================
// Template Expansion (delegates to env package)
// =============================================================================

// renderEnvMap builds the environment variable map for a service.
func (s *Supervisor) renderEnvMap(svc manifest.Service, bin manifest.Binary) (map[string]string, error) {
	return s.envRenderer.RenderEnvMap(svc, bin)
}

// renderArgs expands template variables in command arguments.
func (s *Supervisor) renderArgs(args []string) []string {
	return s.envRenderer.RenderArgs(args)
}

// renderValue expands template variables in a string.
func (s *Supervisor) renderValue(input string) string {
	return s.envRenderer.RenderValue(input)
}

// GPUStatus returns the current GPU detection status.
func (s *Supervisor) GPUStatus() GPUStatus {
	return s.gpuStatus
}

// PortMap returns allocated ports for all services.
func (s *Supervisor) PortMap() map[string]map[string]int {
	return s.portAllocator.Map()
}

// RuntimeInfo returns metadata about the running supervisor instance.
func (s *Supervisor) RuntimeInfo() api.RuntimeInfo {
	return api.RuntimeInfo{
		InstanceID:   s.instanceID,
		StartedAt:    s.startedAt,
		AppDataDir:   s.appData,
		BundleRoot:   s.opts.BundlePath,
		DryRun:       s.opts.DryRun,
		ManifestHash: s.manifestHash,
	}
}

// =============================================================================
// Utility Functions
// =============================================================================

func newInstanceID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	return hex.EncodeToString(buf)
}

func hashManifest(m *manifest.Manifest) string {
	if m == nil {
		return ""
	}
	data, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
