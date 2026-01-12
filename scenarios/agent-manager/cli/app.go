package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
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
	env := cliapp.StandardScenarioEnv(appName, cliapp.ScenarioEnvOptions{
		ExtraAPIEnvVars:     []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		ExtraAPIPortEnvVars: []string{"API_PORT"},
	})
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:              appName,
		Version:           appVersion,
		Description:       "Agent Manager CLI",
		DefaultAPIBase:    defaultAPIBase,
		APIEnvVars:        env.APIEnvVars,
		APIPortEnvVars:    env.APIPortEnvVars,
		APIPortDetector:   cliutil.DetectPortFromVrooli(appName, "API_PORT"),
		ConfigDirEnvVars:  env.ConfigDirEnvVars,
		SourceRootEnvVars: env.SourceRootEnvVars,
		TokenEnvVars:      env.TokenEnvVars,
		BuildFingerprint:  buildFingerprint,
		BuildTimestamp:    buildTimestamp,
		BuildSourceRoot:   buildSourceRoot,
		AllowAnonymous:    true,
	})
	if err != nil {
		return nil, err
	}
	app := &App{
		core:     core,
		services: NewServices(core.APIClient),
	}
	app.core.SetCommands(app.registerCommands())
	return app, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

func (a *App) registerCommands() []cliapp.CommandGroup {
	health := cliapp.CommandGroup{
		Title: "Health",
		Commands: []cliapp.Command{
			{Name: "status", NeedsAPI: true, Description: "Check API health", Run: a.cmdStatus},
		},
	}

	profiles := cliapp.CommandGroup{
		Title: "Profiles",
		Commands: []cliapp.Command{
			{Name: "profile", NeedsAPI: true, Description: "Manage agent profiles", Run: a.cmdProfile},
		},
	}

	tasks := cliapp.CommandGroup{
		Title: "Tasks",
		Commands: []cliapp.Command{
			{Name: "task", NeedsAPI: true, Description: "Manage tasks", Run: a.cmdTask},
		},
	}

	runs := cliapp.CommandGroup{
		Title: "Runs",
		Commands: []cliapp.Command{
			{Name: "run", NeedsAPI: true, Description: "Manage run executions", Run: a.cmdRun},
		},
	}

	runners := cliapp.CommandGroup{
		Title: "Runners",
		Commands: []cliapp.Command{
			{Name: "runner", NeedsAPI: true, Description: "Manage agent runners", Run: a.cmdRunner},
		},
	}

	settings := cliapp.CommandGroup{
		Title: "Settings",
		Commands: []cliapp.Command{
			{Name: "settings", NeedsAPI: true, Description: "Manage settings", Run: a.cmdSettings},
		},
	}

	maintenance := cliapp.CommandGroup{
		Title: "Maintenance",
		Commands: []cliapp.Command{
			{Name: "maintenance", NeedsAPI: true, Description: "Maintenance operations", Run: a.cmdMaintenance},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{health, profiles, tasks, runs, runners, settings, maintenance, config}
}

func (a *App) apiPath(v1Path string) string {
	v1Path = strings.TrimSpace(v1Path)
	if v1Path == "" {
		return ""
	}
	if !strings.HasPrefix(v1Path, "/") {
		v1Path = "/" + v1Path
	}
	base := strings.TrimRight(strings.TrimSpace(a.core.HTTPClient.BaseURL()), "/")
	if strings.HasSuffix(base, "/api/v1") {
		return v1Path
	}
	return "/api/v1" + v1Path
}

type healthResponse struct {
	Status     string            `json:"status"`
	Service    string            `json:"service"`
	Version    string            `json:"version"`
	Readiness  bool              `json:"readiness"`
	Timestamp  string            `json:"timestamp"`
	Deps       map[string]string `json:"dependencies"`
	Error      string            `json:"error,omitempty"`
	Message    string            `json:"message,omitempty"`
	Operations map[string]any    `json:"operations,omitempty"`
}

func (a *App) cmdStatus(_ []string) error {
	body, err := a.core.APIClient.Get(a.apiPath("/health"), nil)
	if err != nil {
		return err
	}

	var parsed healthResponse
	if unmarshalErr := json.Unmarshal(body, &parsed); unmarshalErr == nil && parsed.Status != "" {
		fmt.Printf("Status: %s\n", parsed.Status)
		fmt.Printf("Ready: %v\n", parsed.Readiness)
		if parsed.Service != "" {
			fmt.Printf("Service: %s\n", parsed.Service)
		}
		if parsed.Version != "" {
			fmt.Printf("Version: %s\n", parsed.Version)
		}
		if len(parsed.Deps) > 0 {
			fmt.Println("Dependencies:")
			for key, value := range parsed.Deps {
				fmt.Printf("  %s: %s\n", key, value)
			}
		}
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

func formatEnumValue(value fmt.Stringer, prefix, separator string) string {
	if value == nil {
		return ""
	}
	name := value.String()
	if strings.HasPrefix(name, prefix) {
		name = strings.TrimPrefix(name, prefix)
	}
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
