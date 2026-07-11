package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"agent-manager/cli/domains"
	"agent-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

const (
	appName        = "agent-manager"
	appVersion     = "0.1.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

type App struct {
	core     *cliapp.ScenarioApp
	services *Services
}

func NewApp() (*App, error) {
	app := &App{}
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:             appName,
		Version:          appVersion,
		Description:      "Agent Manager CLI",
		DefaultAPIBase:   defaultAPIBase,
		ExtraAPIEnvVars:  []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		BuildFingerprint: buildFingerprint,
		BuildTimestamp:   buildTimestamp,
		BuildSourceRoot:  buildSourceRoot,
		AllowAnonymous:   true,
		CommandGroups: func(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
			app.core = core
			return app.customCommandGroups()
		},
	})
	if err != nil {
		return nil, err
	}
	app.core = core
	app.services = NewServices(core.APIClient)
	return app, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

func (a *App) customCommandGroups() []cliapp.CommandGroup {
	return domains.CommandGroups(a.dependencies())
}

func (a *App) commandGroups() []cliapp.CommandGroup {
	return append(a.core.StandardBaseCommandGroups(cliapp.StandardBaseCommandOptions{}), a.customCommandGroups()...)
}

func (a *App) dependencies() support.Dependencies {
	return support.Dependencies{
		Profile:          a.cmdProfile,
		Task:             a.cmdTask,
		Run:              a.cmdRun,
		Runner:           a.cmdRunner,
		Policy:           a.cmdPolicy,
		PermissionPolicy: a.cmdPermissionPolicy,
		Settings:         a.cmdSettings,
		Maintenance:      a.cmdMaintenance,
		Ops:              a.cmdOps,
		Health:           a.cmdHealth,
		Events:           a.cmdEvents,
	}
}

func formatEnumValue(value fmt.Stringer, prefix, separator string) string {
	if value == nil {
		return ""
	}
	name := strings.TrimPrefix(value.String(), prefix)
	name = strings.ToLower(name)
	if separator != "_" {
		name = strings.ReplaceAll(name, "_", separator)
	}
	return name
}

func formatTimestamp(timestamp *timestamppb.Timestamp) string {
	if timestamp == nil {
		return ""
	}
	return timestamp.AsTime().UTC().Format(time.RFC3339)
}

func trimTimestamp(value string) string {
	if len(value) > 19 {
		return value[:19]
	}
	return value
}

func formatDuration(duration *durationpb.Duration) string {
	if duration == nil {
		return ""
	}
	return duration.AsDuration().String()
}

func parseRunnerType(value string) domainpb.RunnerType {
	switch strings.ToLower(value) {
	case "claude-code":
		return domainpb.RunnerType_RUNNER_TYPE_CLAUDE_CODE
	case "codex":
		return domainpb.RunnerType_RUNNER_TYPE_CODEX
	case "opencode":
		return domainpb.RunnerType_RUNNER_TYPE_OPENCODE
	default:
		return domainpb.RunnerType_RUNNER_TYPE_UNSPECIFIED
	}
}

func parseRunMode(value string) domainpb.RunMode {
	switch strings.ToLower(value) {
	case "sandboxed":
		return domainpb.RunMode_RUN_MODE_SANDBOXED
	case "in_place", "in-place":
		return domainpb.RunMode_RUN_MODE_IN_PLACE
	default:
		return domainpb.RunMode_RUN_MODE_UNSPECIFIED
	}
}

func protoString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

var cliProtoMarshalOptions = protojson.MarshalOptions{
	UseProtoNames:   true,
	EmitUnpopulated: false,
}

func marshalProtoJSON(msg proto.Message) string {
	if msg == nil {
		return ""
	}
	data, err := cliProtoMarshalOptions.Marshal(msg)
	if err != nil {
		return ""
	}
	return string(data)
}

func parseSandboxConfig(value, filePath string) (*domainpb.SandboxConfig, error) {
	value = strings.TrimSpace(value)
	filePath = strings.TrimSpace(filePath)
	if value == "" && filePath == "" {
		return nil, nil
	}
	var data []byte
	if filePath != "" {
		loaded, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("read sandbox config file: %w", err)
		}
		data = loaded
	} else {
		data = []byte(value)
	}
	var cfg domainpb.SandboxConfig
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid sandbox config JSON: %w", err)
	}
	return &cfg, nil
}

// applySandboxModeOverride parses --sandbox-mode and applies it to cfg.
// Empty input is a no-op (preserves the existing mode); non-empty input
// must match one of: off, tracking, protected (case-insensitive). When
// cfg is nil and a non-empty mode is requested the caller is expected
// to construct a SandboxConfig before calling — see profileUpdate's
// "build one if missing" branch.
func applySandboxModeOverride(cfg *domainpb.SandboxConfig, mode string) (*domainpb.SandboxConfig, error) {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		return cfg, nil
	}
	if cfg == nil {
		cfg = &domainpb.SandboxConfig{}
	}
	switch strings.ToLower(mode) {
	case "off":
		cfg.Mode = domainpb.SandboxMode_SANDBOX_MODE_OFF
	case "tracking":
		cfg.Mode = domainpb.SandboxMode_SANDBOX_MODE_TRACKING
	case "protected":
		cfg.Mode = domainpb.SandboxMode_SANDBOX_MODE_PROTECTED
	default:
		return nil, fmt.Errorf("invalid sandbox mode %q: expected one of off, tracking, protected", mode)
	}
	return cfg, nil
}

// profileSandboxMode renders the sandbox mode for human display.
// Returns "off"/"tracking"/"protected"/"-"; "-" means the profile has
// no SandboxConfig set, in which case the run-time default applies.
func profileSandboxMode(p *domainpb.AgentProfile) string {
	if p == nil || p.SandboxConfig == nil {
		return "-"
	}
	switch p.SandboxConfig.Mode {
	case domainpb.SandboxMode_SANDBOX_MODE_OFF:
		return "off"
	case domainpb.SandboxMode_SANDBOX_MODE_TRACKING:
		return "tracking"
	case domainpb.SandboxMode_SANDBOX_MODE_PROTECTED:
		return "protected"
	default:
		return "default"
	}
}

func applySandboxRetention(cfg *domainpb.SandboxConfig, mode, ttl string) (*domainpb.SandboxConfig, error) {
	mode = strings.TrimSpace(mode)
	ttl = strings.TrimSpace(ttl)
	if mode == "" && ttl == "" {
		return cfg, nil
	}
	if cfg == nil {
		cfg = &domainpb.SandboxConfig{}
	}
	if cfg.Lifecycle == nil {
		cfg.Lifecycle = &domainpb.SandboxLifecycleConfig{}
	}

	if mode != "" {
		switch strings.ToLower(mode) {
		case "keep_active":
			cfg.Lifecycle.StopOn = nil
			cfg.Lifecycle.DeleteOn = nil
		case "stop_on_terminal":
			cfg.Lifecycle.StopOn = []domainpb.SandboxLifecycleEvent{
				domainpb.SandboxLifecycleEvent_SANDBOX_LIFECYCLE_EVENT_TERMINAL,
			}
			cfg.Lifecycle.DeleteOn = nil
		case "delete_on_terminal":
			cfg.Lifecycle.StopOn = []domainpb.SandboxLifecycleEvent{
				domainpb.SandboxLifecycleEvent_SANDBOX_LIFECYCLE_EVENT_TERMINAL,
			}
			cfg.Lifecycle.DeleteOn = []domainpb.SandboxLifecycleEvent{
				domainpb.SandboxLifecycleEvent_SANDBOX_LIFECYCLE_EVENT_TERMINAL,
			}
		default:
			return nil, fmt.Errorf("invalid sandbox retention mode: %s", mode)
		}
	}

	if ttl != "" {
		parsed, err := time.ParseDuration(ttl)
		if err != nil {
			if fallback, fallbackErr := time.ParseDuration(ttl + "s"); fallbackErr == nil {
				parsed = fallback
			} else {
				return nil, fmt.Errorf("invalid sandbox retention ttl: %w", err)
			}
		}
		cfg.Lifecycle.Ttl = durationpb.New(parsed)
	}

	return cfg, nil
}
