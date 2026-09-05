package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/vrooli/api-core/targetmodel"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/shared"
	sessionsH "web-console/handlers/sessions"
	"web-console/internal/backend"
	"web-console/internal/policy"
	intSessions "web-console/internal/sessions"
	"web-console/session"
)

// Create implements sessions.RemoteService. Remote sessions use the same
// session.Manager and typed SessionsService as local sessions; the only
// remote-specific data is the Bridge launch specification stored in the
// manager's backend registry.
func (s *Server) Create(ctx context.Context, in sessionsH.CreateInput) (sessionsH.Session, error) {
	target, ok := s.targetByID(strings.TrimSpace(in.TargetID))
	if !ok {
		return sessionsH.Session{}, fmt.Errorf("%w: %s", sessionsH.ErrTargetNotFound, in.TargetID)
	}
	if target.DeviceKind == "local" {
		return sessionsH.Session{}, fmt.Errorf("%w: local target must use the local session backend", sessionsH.ErrTargetUnavailable)
	}
	if !target.Available {
		reason := target.Reason
		if reason == "" {
			reason = "target is not dispatchable"
		}
		return sessionsH.Session{}, fmt.Errorf("%w: %s", sessionsH.ErrTargetUnavailable, reason)
	}
	if in.ExecuteLaunchCommand {
		if err := ensureLaunchCapability(target, in.LaunchCommand); err != nil {
			return sessionsH.Session{}, err
		}
	}
	if s.sessions == nil {
		return sessionsH.Session{}, fmt.Errorf("%w: remote session manager is not configured", sessionsH.ErrRemoteUnavailable)
	}

	cols, rows := in.Cols, in.Rows
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}
	var sessionPolicy *policy.Policy
	if in.HasPolicy {
		sessionPolicy = &policy.Policy{Mode: policy.Mode(in.Policy.Mode), Duration: in.Policy.Duration}
	}
	created, err := s.sessions.CreateRemote(ctx, session.RemoteLaunch{
		BaseURL: target.BaseURL, NodeID: target.NodeID, OwnerToken: target.OwnerToken,
		ReauthToken: target.ReauthToken,
		Cols:        uint16(cols), Rows: uint16(rows), LaunchCommand: in.LaunchCommand,
		ExecuteLaunchCommand: in.ExecuteLaunchCommand,
	}, sessionPolicy)
	if err != nil {
		return sessionsH.Session{}, err
	}
	if s.sessionStore != nil {
		_ = s.sessionStore.SetProvenance(ctx, created.ID, "remote", "target:"+target.ID, target.Label)
	}
	response := intSessions.FromSession(created)
	return sessionsH.Session{
		ID: response.ID, Shell: response.Shell, CreatedAt: response.CreatedAt,
		Cols: response.Cols, Rows: response.Rows, Backend: string(response.Backend),
		SurvivesRestart: true, Policy: sessionsH.Policy{Mode: string(response.Policy.Mode), Duration: response.Policy.Duration},
		Origin: "remote", Owner: "target:" + target.ID, DisplayLabel: target.Label,
		Target: targetToProto(target),
	}, nil
}

// ensureLaunchCapability prevents a remote session from being created when
// the command would immediately fail because the selected node does not have
// the requested coding agent. Commands outside the governed agent set remain
// valid shell commands and do not require a capability inventory entry.
func ensureLaunchCapability(target targetConnection, command string) error {
	capability, recognized := launchCapability(command)
	if !recognized {
		return nil
	}
	identity := targetmodel.ReadinessCapabilityPrefix + capability
	for _, fact := range target.Readiness {
		if fact.Identity != identity {
			continue
		}
		state := fact.State
		if state == "" {
			if fact.Passed {
				state = targetmodel.ReadinessReady
			} else {
				state = targetmodel.ReadinessMissing
			}
		}
		if state == targetmodel.ReadinessReady {
			return nil
		}
		return fmt.Errorf("%w: capability %q on %s is %s; %s", sessionsH.ErrTargetUnavailable, capability, target.Label, state, fact.RecoveryAction)
	}
	return fmt.Errorf("%w: capability %q on %s is unknown; refresh the target inventory before launching", sessionsH.ErrTargetUnavailable, capability, target.Label)
}

func launchCapability(command string) (string, bool) {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(command)))
	if len(fields) == 0 {
		return "", false
	}
	base := func(value string) string {
		value = strings.TrimSpace(value)
		if index := strings.LastIndex(value, "/"); index >= 0 {
			value = value[index+1:]
		}
		return value
	}
	for index, field := range fields {
		switch base(field) {
		case "claude", "claude-code":
			return "claude", true
		case "codex":
			return "codex", true
		case "opencode":
			return "opencode", true
		case "grok":
			return "grok", true
		case "agy", "antigravity":
			return "agy", true
		case "vrooli-agent-launcher":
			for _, arg := range fields[index+1:] {
				if strings.HasPrefix(arg, "--agent=") {
					return normalizeLaunchRunner(strings.TrimPrefix(arg, "--agent="))
				}
			}
		case "vrooli":
			if index+2 < len(fields) && fields[index+1] == "agent" && fields[index+2] == "launch" {
				for offset := index + 3; offset < len(fields); offset++ {
					if strings.HasPrefix(fields[offset], "--runner=") {
						return normalizeLaunchRunner(strings.TrimPrefix(fields[offset], "--runner="))
					}
				}
				return "claude", true
			}
		}
	}
	return "", false
}

func normalizeLaunchRunner(runner string) (string, bool) {
	switch runner {
	case "claude", "claude-code":
		return "claude", true
	case "codex":
		return "codex", true
	case "opencode":
		return "opencode", true
	case "grok":
		return "grok", true
	case "agy", "antigravity":
		return "agy", true
	default:
		return "", false
	}
}

func (s *Server) List(ctx context.Context) ([]sessionsH.Session, error) {
	items := make([]sessionsH.Session, 0)
	if s.sessions == nil {
		return items, nil
	}
	for _, current := range s.sessions.List() {
		if current.Backend == backend.Remote {
			items = append(items, s.managerRemoteSession(ctx, current))
		}
	}
	return items, nil
}

func (s *Server) Get(ctx context.Context, id string) (sessionsH.Session, error) {
	if s.sessions == nil {
		return sessionsH.Session{}, fmt.Errorf("%w: remote session manager is not configured", sessionsH.ErrRemoteUnavailable)
	}
	current, ok := s.sessions.Get(id)
	if !ok || current.Backend != backend.Remote {
		return sessionsH.Session{}, fmt.Errorf("%w: %s", sessionsH.ErrNotFound, id)
	}
	return s.managerRemoteSession(ctx, current), nil
}

func (s *Server) Delete(ctx context.Context, id string) error {
	if s.sessions == nil {
		return fmt.Errorf("%w: remote session manager is not configured", sessionsH.ErrRemoteUnavailable)
	}
	current, ok := s.sessions.Get(id)
	if !ok || current.Backend != backend.Remote {
		return fmt.Errorf("%w: %s", sessionsH.ErrNotFound, id)
	}
	return s.sessions.Delete(ctx, id)
}

func (s *Server) managerRemoteSession(ctx context.Context, current *session.Session) sessionsH.Session {
	response := intSessions.FromSession(current)
	owner, label := "target:", "Remote target"
	if s.sessionStore != nil {
		if metadata, err := s.sessionStore.Get(ctx, current.ID); err == nil {
			owner, label = metadata.Owner, metadata.DisplayLabel
		}
	}
	var target *sharedv1.Target
	if strings.HasPrefix(owner, "target:") {
		if candidate, ok := s.targetByID(strings.TrimPrefix(owner, "target:")); ok {
			target = targetToProto(candidate)
		}
	}
	return sessionsH.Session{
		ID: response.ID, Shell: response.Shell, CreatedAt: response.CreatedAt,
		Cols: response.Cols, Rows: response.Rows, Backend: string(response.Backend),
		SurvivesRestart: true, Policy: sessionsH.Policy{Mode: string(response.Policy.Mode), Duration: response.Policy.Duration},
		Origin: "remote", Owner: owner, DisplayLabel: label, Target: target,
	}
}
