package main

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"web-console/internal/sessionstore"
)

// nativeObservationMetadata selects the persisted transport contract, never a
// filesystem guess. A managed stdio owner cannot fall through to a TUI daemon.
func (s *Server) nativeObservationMetadata(ctx context.Context, id string) (sessionstore.Metadata, error) {
	if s.sessionStore == nil {
		return sessionstore.Metadata{}, connect.NewError(connect.CodeUnavailable, errors.New("session store unavailable"))
	}
	meta, err := s.sessionStore.Get(ctx, id)
	if err != nil {
		code := connect.CodeUnavailable
		if errors.Is(err, sql.ErrNoRows) {
			code = connect.CodeNotFound
		}
		return meta, connect.NewError(code, errors.New("native session metadata unavailable"))
	}
	if meta.AgentType != sessionstore.AgentCodex || (meta.LaunchMode != sessionstore.LaunchModeCodexAppServer && meta.LaunchMode != sessionstore.LaunchModeTerminalPTY) {
		return meta, connect.NewError(connect.CodeUnavailable, errors.New("session has no recognized native Codex owner transport"))
	}
	if strings.TrimSpace(meta.LaunchDescriptorJSON) != "" {
		d, err := sessionstore.ParseLaunchDescriptor(meta.LaunchDescriptorJSON)
		if err != nil || d.LaunchMode != string(meta.LaunchMode) || !strings.EqualFold(d.Agent, "codex") {
			return meta, connect.NewError(connect.CodeUnavailable, errors.New("native session launch binding is inconsistent"))
		}
	}
	return meta, nil
}

func nativeUsageHome(meta sessionstore.Metadata, ids []string) (string, error) {
	if meta.LaunchMode != sessionstore.LaunchModeCodexAppServer {
		return "", errors.New("managed native usage transport required")
	}
	// Audit identity survives retirement, which deliberately clears the live
	// owner generation. The persisted thread lineage and launch home remain.
	if meta.NativeThreadID == "" {
		return "", errors.New("managed native usage ownership is unavailable")
	}
	for _, id := range ids {
		if id != meta.NativeThreadID && id != meta.ForkedFromNativeSession {
			return "", errors.New("native usage thread is outside the session lineage")
		}
	}
	d, err := sessionstore.ParseLaunchDescriptor(meta.LaunchDescriptorJSON)
	if err != nil {
		return "", err
	}
	home := strings.TrimSpace(d.Environment["CODEX_HOME"])
	if home == "" || !filepath.IsAbs(home) {
		return "", errors.New("managed native audit home is unavailable")
	}
	return filepath.Clean(home), nil
}
