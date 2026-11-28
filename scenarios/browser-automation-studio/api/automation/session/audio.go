package session

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type AudioDiagnostics struct {
	Diagnostic             any    `json:"diag"`
	Transcript             string `json:"final"`
	Interim                string `json:"interim"`
	CoverageVisible        bool   `json:"coverage_visible"`
	ExportAvailable        bool   `json:"export_available"`
	DiagnosticMetadataOnly bool   `json:"diagnostic_metadata_only"`
}

// AudioPlaybackCommand is intentionally limited to the two session-owned
// boundary controls supported by the current driver.
type AudioPlaybackCommand string

const (
	AudioPlaybackRestart AudioPlaybackCommand = "restart"
	AudioPlaybackStop    AudioPlaybackCommand = "stop"
)

// AudioPlayback forwards a typed host-audio control to this exact managed
// session lease. Broker performs the current grant, caller and URL checks
// before invoking this method; this method binds the invocation to the
// admitted profile and immutable owner/lease.
func (s *Session) AudioPlayback(ctx context.Context, ownerID, leaseID, profileID, callerID string, command AudioPlaybackCommand) error {
	if s == nil || s.isClosed() {
		return errors.New("audio session is closed")
	}
	if ownerID != s.executionID || leaseID != s.leaseID || strings.TrimSpace(callerID) == "" {
		return errors.New("audio command does not match the session owner lease")
	}
	s.mu.RLock()
	profile := s.permissionProfile
	callerResolver := s.callerResolver
	appTarget := s.appTarget
	s.mu.RUnlock()
	if appTarget || profile == nil || profile.ProfileID != profileID || profile.AudioMode != "pipewire_qualification" {
		return errors.New("audio playback controls require a PipeWire-qualified managed profile")
	}
	if callerResolver == nil {
		return errors.New("audio command has no trusted caller resolver")
	}
	resolvedCaller, err := callerResolver.BrowserCallerID(ctx)
	if err != nil || resolvedCaller != callerID {
		return errors.New("audio command caller does not match the trusted browser caller")
	}
	suffix := ""
	switch command {
	case AudioPlaybackRestart:
		suffix = "/audio/restart"
	case AudioPlaybackStop:
		suffix = "/audio/stop"
	default:
		return errors.New("unsupported audio playback command")
	}
	body, err := json.Marshal(map[string]string{"execution_id": ownerID, "lease_id": leaseID})
	if err != nil {
		return errors.New("could not encode the owner-bound audio command")
	}
	response, err := s.client.SessionRouteRequest(ctx, s.id, ownerID, leaseID, http.MethodPost, suffix, body, nil)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	var result struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil || !result.Success {
		return errors.New("driver did not confirm the owner-bound audio command")
	}
	return nil
}

// ReadAudioDiagnostics returns a fixed, typed observation from the current
// Audio Tools task surface. The caller must have authorized audio_observe via
// the Broker; arbitrary JavaScript is never accepted or forwarded.
func (s *Session) ReadAudioDiagnostics(ctx context.Context, ownerID, leaseID, profileID, callerID, surface string) (AudioDiagnostics, error) {
	if s == nil || s.isClosed() {
		return AudioDiagnostics{}, errors.New("audio session is closed")
	}
	if ownerID != s.executionID || leaseID != s.leaseID || strings.TrimSpace(callerID) == "" {
		return AudioDiagnostics{}, errors.New("audio observation does not match the session owner lease")
	}
	if surface != "audio-tools" && surface != "swarm-manager" {
		return AudioDiagnostics{}, errors.New("unsupported Audio Tools diagnostic surface")
	}
	s.mu.RLock()
	profile := s.permissionProfile
	callerResolver := s.callerResolver
	appTarget := s.appTarget
	s.mu.RUnlock()
	if appTarget || profile == nil || profile.ProfileID != profileID || profile.AudioMode == "" || !profile.View {
		return AudioDiagnostics{}, errors.New("audio diagnostics require an admitted audio-capable view profile")
	}
	if callerResolver == nil {
		return AudioDiagnostics{}, errors.New("audio observation has no trusted caller resolver")
	}
	resolvedCaller, err := callerResolver.BrowserCallerID(ctx)
	if err != nil || resolvedCaller != callerID {
		return AudioDiagnostics{}, errors.New("audio observation caller does not match the trusted browser caller")
	}
	body, err := json.Marshal(map[string]string{"execution_id": ownerID, "lease_id": leaseID, "surface": surface})
	if err != nil {
		return AudioDiagnostics{}, errors.New("could not encode the fixed audio observation request")
	}
	response, err := s.client.SessionRouteRequest(ctx, s.id, ownerID, leaseID, http.MethodPost, "/audio/diagnostics", body, nil)
	if err != nil {
		return AudioDiagnostics{}, err
	}
	defer response.Body.Close()
	var result struct {
		Success     bool             `json:"success"`
		Diagnostics AudioDiagnostics `json:"diagnostics"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil || !result.Success {
		return AudioDiagnostics{}, errors.New("driver did not confirm the fixed audio observation")
	}
	return result.Diagnostics, nil
}
