package bundleruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/infra"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
)

const authenticationModeStateFile = "runtime/authentication-mode.json"

type authenticationModeState struct {
	Mode         string    `json:"mode"`
	PreviousMode string    `json:"previous_mode,omitempty"`
	TransitionID string    `json:"transition_id"`
	ChangedAt    time.Time `json:"changed_at"`
}

// sharedProviderLeaseMetadata is deliberately not a credential container.
// The broker's scoped credential is ephemeral and is injected directly into
// the resource process. Durable entitlement leases use the native credential
// authority instead of LeasePath.
type sharedProviderLeaseMetadata struct {
	Resource  string    `json:"resource,omitempty"`
	Audience  string    `json:"audience,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
}

func readSharedProviderLeaseMetadata(fileSystem infra.FileSystem, appData string, profile *manifest.AuthenticationProfile) (sharedProviderLeaseMetadata, error) {
	path := manifest.ResolvePath(appData, profile.LeasePath)
	leaseBytes, err := fileSystem.ReadFile(path)
	if err != nil {
		return sharedProviderLeaseMetadata{}, err
	}
	var lease sharedProviderLeaseMetadata
	decoder := json.NewDecoder(bytes.NewReader(leaseBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&lease); err != nil {
		return sharedProviderLeaseMetadata{}, fmt.Errorf("parse shared-provider lease metadata: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return sharedProviderLeaseMetadata{}, errors.New("parse shared-provider lease metadata: multiple JSON values")
		}
		return sharedProviderLeaseMetadata{}, fmt.Errorf("parse shared-provider lease metadata: trailing data: %w", err)
	}
	if lease.ExpiresAt.IsZero() {
		return sharedProviderLeaseMetadata{}, errors.New("shared-provider lease metadata is missing expires_at")
	}
	if lease.Resource != "" && lease.Resource != profile.Resource {
		return sharedProviderLeaseMetadata{}, fmt.Errorf("shared-provider lease metadata resource %q does not match %q", lease.Resource, profile.Resource)
	}
	if lease.Audience != "" && lease.Audience != profile.Audience {
		return sharedProviderLeaseMetadata{}, fmt.Errorf("shared-provider lease metadata audience %q does not match %q", lease.Audience, profile.Audience)
	}
	return lease, nil
}

// authenticationModeManager owns only non-secret mode selection state. It
// never provisions, stores, or clears provider credentials or durable
// entitlement leases. Those operations remain with the declared provider and
// credential authority.
type authenticationModeManager struct {
	fs      infra.FileSystem
	clock   infra.Clock
	appData string
	profile *manifest.AuthenticationProfile

	mu    sync.RWMutex
	state authenticationModeState
}

func newAuthenticationModeManager(fileSystem infra.FileSystem, clock infra.Clock, appData string, profile *manifest.AuthenticationProfile) (*authenticationModeManager, error) {
	if profile == nil {
		return nil, nil
	}
	m := &authenticationModeManager{fs: fileSystem, clock: clock, appData: appData, profile: profile}
	data, err := fileSystem.ReadFile(m.statePath())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			m.state = authenticationModeState{Mode: profile.Mode}
			return m, nil
		}
		return nil, fmt.Errorf("read authentication mode state: %w", err)
	}
	if err := json.Unmarshal(data, &m.state); err != nil {
		return nil, fmt.Errorf("parse authentication mode state: %w", err)
	}
	if _, ok := profile.ProfileForMode(m.state.Mode); !ok {
		return nil, fmt.Errorf("persisted authentication mode %q is not declared by the bundle", m.state.Mode)
	}
	return m, nil
}

func (m *authenticationModeManager) statePath() string {
	return filepath.Join(m.appData, authenticationModeStateFile)
}

func (m *authenticationModeManager) currentProfile() *manifest.AuthenticationProfile {
	profile, _ := m.profile.ProfileForMode(m.state.Mode)
	return profile
}

func (m *authenticationModeManager) stateFor(profile *manifest.AuthenticationProfile) string {
	switch profile.Mode {
	case "personal_local":
		return "offline_ready"
	case "local_multi_user":
		return "setup_required"
	case "remote_vrooli":
		return "provider_required"
	case "shared_provider":
		return m.sharedProviderState(profile)
	default:
		return "unsupported"
	}
}

func (m *authenticationModeManager) sharedProviderState(profile *manifest.AuthenticationProfile) string {
	if strings.TrimSpace(profile.LeasePath) == "" {
		return "lease_unavailable"
	}
	lease, err := readSharedProviderLeaseMetadata(m.fs, m.appData, profile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "lease_unavailable"
		}
		return "lease_invalid"
	}
	if !lease.ExpiresAt.After(m.clock.Now()) {
		return "lease_expired"
	}
	return "lease_ready"
}

func (m *authenticationModeManager) statusLocked() map[string]interface{} {
	profile := m.currentProfile()
	status := map[string]interface{}{
		"mode":                   m.state.Mode,
		"provider":               profile.Provider,
		"resource":               profile.Resource,
		"audience":               profile.Audience,
		"provider_endpoint":      profile.ProviderEndpoint,
		"human_sign_in":          profile.HumanSignIn,
		"offline":                profile.Offline,
		"lease_path":             profile.LeasePath,
		"recovery_url":           profile.RecoveryURL,
		"requires_authenticator": profile.RequiresAuthenticator,
		"state":                  m.stateFor(profile),
		"declared_modes":         m.profile.DeclaredModes(),
	}
	if profile.Mode == "shared_provider" && strings.TrimSpace(profile.LeasePath) != "" {
		lease, err := readSharedProviderLeaseMetadata(m.fs, m.appData, profile)
		if err == nil {
			status["lease_expires_at"] = lease.ExpiresAt.UTC().Format(time.RFC3339)
		}
	}
	if m.state.PreviousMode != "" {
		status["previous_mode"] = m.state.PreviousMode
	}
	if !m.state.ChangedAt.IsZero() {
		status["changed_at"] = m.state.ChangedAt.UTC().Format(time.RFC3339)
	}
	if m.state.TransitionID != "" {
		status["transition_id"] = m.state.TransitionID
	}
	return status
}

func (m *authenticationModeManager) status() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.statusLocked()
}

func (m *authenticationModeManager) options() []map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	options := make([]map[string]interface{}, 0, len(m.profile.DeclaredModes()))
	for _, mode := range m.profile.DeclaredModes() {
		profile, _ := m.profile.ProfileForMode(mode)
		options = append(options, map[string]interface{}{
			"mode":                   mode,
			"selected":               mode == m.state.Mode,
			"provider":               profile.Provider,
			"resource":               profile.Resource,
			"audience":               profile.Audience,
			"provider_endpoint":      profile.ProviderEndpoint,
			"human_sign_in":          profile.HumanSignIn,
			"offline":                profile.Offline,
			"requires_authenticator": profile.RequiresAuthenticator,
			"state":                  m.stateFor(profile),
		})
	}
	return options
}

func (m *authenticationModeManager) ensureSelectable(profile *manifest.AuthenticationProfile) error {
	if profile == nil {
		return errors.New("authentication mode is not declared by the bundle")
	}
	if profile.Mode == "shared_provider" && m.sharedProviderState(profile) != "lease_ready" {
		return fmt.Errorf("shared_provider mode cannot be selected while its scoped lease is %s", m.sharedProviderState(profile))
	}
	return nil
}

func (m *authenticationModeManager) persistLocked(next authenticationModeState) error {
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return fmt.Errorf("encode authentication mode state: %w", err)
	}
	path := m.statePath()
	if err := m.fs.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create authentication mode state directory: %w", err)
	}
	tmp := path + ".tmp"
	if err := m.fs.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write authentication mode state: %w", err)
	}
	if err := m.fs.Rename(tmp, path); err != nil {
		_ = m.fs.Remove(tmp)
		return fmt.Errorf("commit authentication mode state: %w", err)
	}
	return nil
}

func (m *authenticationModeManager) selectMode(_ context.Context, mode string) (map[string]interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	mode = strings.TrimSpace(mode)
	profile, ok := m.profile.ProfileForMode(mode)
	if !ok {
		return nil, fmt.Errorf("authentication mode %q is not declared by the bundle", mode)
	}
	if err := m.ensureSelectable(profile); err != nil {
		return nil, err
	}
	if mode == m.state.Mode {
		return m.statusLocked(), nil
	}
	next := authenticationModeState{
		Mode:         mode,
		PreviousMode: m.state.Mode,
		TransitionID: fmt.Sprintf("auth-mode-%d", m.clock.Now().UnixNano()),
		ChangedAt:    m.clock.Now().UTC(),
	}
	if err := m.persistLocked(next); err != nil {
		return nil, err
	}
	m.state = next
	return m.statusLocked(), nil
}

func (m *authenticationModeManager) rollback(_ context.Context) (map[string]interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.state.PreviousMode == "" {
		return nil, errors.New("no previous authentication mode is available for rollback")
	}
	profile, ok := m.profile.ProfileForMode(m.state.PreviousMode)
	if !ok {
		return nil, errors.New("previous authentication mode is no longer declared by the bundle")
	}
	if err := m.ensureSelectable(profile); err != nil {
		return nil, err
	}
	next := authenticationModeState{
		Mode:         m.state.PreviousMode,
		PreviousMode: m.state.Mode,
		TransitionID: fmt.Sprintf("auth-mode-rollback-%d", m.clock.Now().UnixNano()),
		ChangedAt:    m.clock.Now().UTC(),
	}
	if err := m.persistLocked(next); err != nil {
		return nil, err
	}
	m.state = next
	return m.statusLocked(), nil
}
