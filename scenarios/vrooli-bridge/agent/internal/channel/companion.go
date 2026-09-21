package channel

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	companionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/companion"
	presencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/presence"
	presenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/presence/presence_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	bridgeexec "vrooli-bridge/agent/internal/exec"
	"vrooli-bridge/agent/internal/nodecred"
)

const maxCompanionVersionBytes = 256

type CompanionLifecycleAdapter interface {
	HandleCompanionCommand(context.Context, *companionv1.CompanionCommand) (*companionv1.CompanionResponse, error)
}

type CompanionLifecycleReporter interface {
	ReportCompanionResponse(context.Context, *companionv1.CompanionResponse) error
}

type presenceCompanionReporter struct {
	rpc    presenceconnect.PresenceServiceClient
	cred   *nodecred.Credential
	nodeID string
	now    func() time.Time
}

func (r presenceCompanionReporter) ReportCompanionResponse(ctx context.Context, response *companionv1.CompanionResponse) error {
	req := connect.NewRequest(&presencev1.ReportCompanionResponseRequest{Response: response})
	if r.cred != nil {
		now := time.Now
		if r.now != nil {
			now = r.now
		}
		for k, v := range r.cred.Headers(r.nodeID, now().UTC()) {
			req.Header().Set(k, v)
		}
	}
	_, err := r.rpc.ReportCompanionResponse(ctx, req)
	return err
}

const (
	companionLabel       = "com.vrooli.device-control.desktop"
	companionArtifactRel = ".vrooli/bin/device-control-companion"
	companionConfigRel   = "Library/Application Support/Vrooli/desktop.json"
	companionPort        = "16465"
)

// localCompanionLifecycleAdapter owns the user-scoped macOS lifecycle after a
// signed artifact placement. It never downloads bytes, accepts shell text, or
// installs a root daemon; Bridge's ArtifactDelivery frame must place the
// executable first.
type localCompanionLifecycleAdapter struct {
	homeDir string
	runner  bridgeexec.CommandRunner
}

func (a localCompanionLifecycleAdapter) HandleCompanionCommand(ctx context.Context, command *companionv1.CompanionCommand) (*companionv1.CompanionResponse, error) {
	if !validCompanionCommand(command) {
		return nil, errors.New("invalid companion command")
	}
	response := &companionv1.CompanionResponse{OperationId: command.GetOperationId(), NodeId: command.GetNodeId(), Version: command.GetVersion(), ObservedAt: timestamppb.Now()}
	if runtime.GOOS != "darwin" {
		response.State = companionv1.CompanionState_COMPANION_STATE_FAILED
		response.ReasonCode = "unsupported_platform"
		response.Recovery = "run_on_macos_user_session"
		return response, nil
	}
	home, err := a.resolveHome()
	if err != nil {
		response.State, response.ReasonCode, response.Recovery = companionv1.CompanionState_COMPANION_STATE_FAILED, "user_home_unavailable", "repair_user_session"
		return response, nil
	}
	paths := newCompanionPaths(home)
	switch command.GetKind() {
	case companionv1.CompanionOperationKind_COMPANION_OPERATION_KIND_INSPECT:
		return a.inspect(ctx, response, paths), nil
	case companionv1.CompanionOperationKind_COMPANION_OPERATION_KIND_INSTALL,
		companionv1.CompanionOperationKind_COMPANION_OPERATION_KIND_UPGRADE:
		return a.install(ctx, response, paths, command.GetDisplayId()), nil
	case companionv1.CompanionOperationKind_COMPANION_OPERATION_KIND_REVOKE:
		return a.revoke(ctx, response, paths, false), nil
	case companionv1.CompanionOperationKind_COMPANION_OPERATION_KIND_REMOVE:
		return a.revoke(ctx, response, paths, true), nil
	}
	return response, nil
}

type companionPathSet struct {
	home, artifact, config, plist string
}

func newCompanionPaths(home string) companionPathSet {
	return companionPathSet{
		home: home, artifact: filepath.Join(home, companionArtifactRel),
		config: filepath.Join(home, companionConfigRel),
		plist:  filepath.Join(home, "Library", "LaunchAgents", companionLabel+".plist"),
	}
}

func (a localCompanionLifecycleAdapter) resolveHome() (string, error) {
	if home := strings.TrimSpace(a.homeDir); home != "" {
		return filepath.Abs(home)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Abs(home)
}

func (a localCompanionLifecycleAdapter) inspect(ctx context.Context, response *companionv1.CompanionResponse, paths companionPathSet) *companionv1.CompanionResponse {
	info, err := os.Stat(paths.artifact)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		response.State, response.ReasonCode, response.Recovery = companionv1.CompanionState_COMPANION_STATE_ABSENT, "artifact_not_installed", "deliver_verified_companion_artifact"
		return response
	}
	if _, err := os.Stat(paths.plist); err != nil {
		response.State, response.ReasonCode, response.Recovery = companionv1.CompanionState_COMPANION_STATE_DEGRADED, "launch_agent_missing", "install_user_launch_agent"
		return response
	}
	if code, runErr := a.run(ctx, []string{"launchctl", "print", launchDomain(paths.home) + "/" + companionLabel}); runErr != nil || code != 0 {
		response.State, response.ReasonCode, response.Recovery = companionv1.CompanionState_COMPANION_STATE_DEGRADED, "launch_agent_not_running", "restart_user_launch_agent"
		return response
	}
	response.State, response.UserLaunchAgent = companionv1.CompanionState_COMPANION_STATE_READY, true
	return response
}

func (a localCompanionLifecycleAdapter) install(ctx context.Context, response *companionv1.CompanionResponse, paths companionPathSet, displayID string) *companionv1.CompanionResponse {
	info, err := os.Stat(paths.artifact)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		response.State, response.ReasonCode, response.Recovery = companionv1.CompanionState_COMPANION_STATE_FAILED, "artifact_not_installed", "deliver_verified_companion_artifact"
		return response
	}
	if err := os.MkdirAll(filepath.Dir(paths.config), 0o700); err != nil {
		return companionFailure(response, "config_directory_failed", "repair_user_session")
	}
	config, _ := json.Marshal(struct {
		DisplayID string `json:"display_id,omitempty"`
	}{DisplayID: strings.TrimSpace(displayID)})
	if err := atomicWrite(paths.config, config, 0o600); err != nil {
		return companionFailure(response, "config_write_failed", "repair_user_session")
	}
	if err := os.MkdirAll(filepath.Dir(paths.plist), 0o700); err != nil {
		return companionFailure(response, "launch_agent_directory_failed", "repair_user_session")
	}
	plist := renderLaunchAgent(paths.artifact, paths.config)
	if err := atomicWrite(paths.plist, []byte(plist), 0o600); err != nil {
		return companionFailure(response, "launch_agent_write_failed", "repair_user_session")
	}
	// Bootout is idempotent for the upgrade path; its failure is intentionally
	// ignored because a first install has no existing service.
	_, _ = a.run(ctx, []string{"launchctl", "bootout", launchDomain(paths.home) + "/" + companionLabel})
	if code, err := a.run(ctx, []string{"launchctl", "bootstrap", launchDomain(paths.home), paths.plist}); err != nil || code != 0 {
		return companionFailure(response, "launch_agent_bootstrap_failed", "approve_user_session_permissions")
	}
	response.State, response.UserLaunchAgent = companionv1.CompanionState_COMPANION_STATE_READY, true
	return response
}

func (a localCompanionLifecycleAdapter) revoke(ctx context.Context, response *companionv1.CompanionResponse, paths companionPathSet, remove bool) *companionv1.CompanionResponse {
	code, runErr := a.run(ctx, []string{"launchctl", "bootout", launchDomain(paths.home) + "/" + companionLabel})
	if !remove && (runErr != nil || code != 0) {
		return companionFailure(response, "launch_agent_revoke_failed", "inspect_user_launch_agent")
	}
	if remove {
		for _, path := range []string{paths.plist, paths.config, paths.artifact} {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return companionFailure(response, "companion_remove_failed", "remove_companion_artifact_manually")
			}
		}
		response.State = companionv1.CompanionState_COMPANION_STATE_REMOVED
	} else {
		response.State = companionv1.CompanionState_COMPANION_STATE_REVOKED
	}
	response.UserLaunchAgent = false
	return response
}

func (a localCompanionLifecycleAdapter) run(ctx context.Context, argv []string) (int, error) {
	if a.runner != nil {
		return a.runner.Run(ctx, argv, "", nil)
	}
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...) // #nosec G204 -- argv is a closed, internally constructed launchctl vocabulary.
	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), err
	}
	return -1, err
}

func launchDomain(home string) string {
	return "gui/" + strconv.Itoa(os.Getuid())
}

func companionFailure(response *companionv1.CompanionResponse, reason, recovery string) *companionv1.CompanionResponse {
	response.State, response.ReasonCode, response.Recovery = companionv1.CompanionState_COMPANION_STATE_FAILED, reason, recovery
	return response
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".vrooli-companion-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func renderLaunchAgent(program, config string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict><key>Label</key><string>%s</string><key>ProgramArguments</key><array><string>%s</string><string>--config</string><string>%s</string></array><key>EnvironmentVariables</key><dict><key>DEVICE_CONTROL_COMPANION_PORT</key><string>%s</string></dict><key>RunAtLoad</key><true/><key>KeepAlive</key><true/></dict></plist>
`, xmlEscape(companionLabel), xmlEscape(program), xmlEscape(config), companionPort)
}

func xmlEscape(value string) string {
	var escaped strings.Builder
	_ = xml.EscapeText(&escaped, []byte(value))
	return escaped.String()
}

func (c *Client) handleCompanionCommand(command *companionv1.CompanionCommand) {
	if !validCompanionCommand(command) || command.GetNodeId() != c.cfg.NodeID {
		return
	}
	response := &companionv1.CompanionResponse{OperationId: command.GetOperationId(), NodeId: c.cfg.NodeID, Version: command.GetVersion(), State: companionv1.CompanionState_COMPANION_STATE_FAILED, ReasonCode: "companion_unavailable", Recovery: "inspect_companion_health", ObservedAt: timestamppb.Now()}
	if c.companionAdapter != nil {
		out, err := c.companionAdapter.HandleCompanionCommand(c.baseCtxOrBackground(), command)
		if err != nil {
			response.ReasonCode = boundedDesktopError(err)
		} else if out != nil {
			response = out
			response.OperationId = command.GetOperationId()
			response.NodeId = c.cfg.NodeID
		}
	}
	if c.companionReporter != nil {
		if err := c.companionReporter.ReportCompanionResponse(c.baseCtxOrBackground(), response); err != nil {
			c.logger.Printf("channel: companion response report failed: %v", err)
		}
	}
}

func validCompanionCommand(command *companionv1.CompanionCommand) bool {
	if command == nil || strings.TrimSpace(command.GetOperationId()) == "" || strings.TrimSpace(command.GetNodeId()) == "" || len(command.GetVersion()) > maxCompanionVersionBytes {
		return false
	}
	switch command.GetKind() {
	case companionv1.CompanionOperationKind_COMPANION_OPERATION_KIND_INSTALL,
		companionv1.CompanionOperationKind_COMPANION_OPERATION_KIND_INSPECT,
		companionv1.CompanionOperationKind_COMPANION_OPERATION_KIND_UPGRADE,
		companionv1.CompanionOperationKind_COMPANION_OPERATION_KIND_REVOKE,
		companionv1.CompanionOperationKind_COMPANION_OPERATION_KIND_REMOVE:
		return true
	default:
		return false
	}
}
