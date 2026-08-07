package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"agent-manager/cli/domains"
	"agent-manager/cli/internal/support"

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
		SubcommandGroups: func(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
			app.core = core
			return domains.SubcommandGroups(app.dependencies())
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
		Declarations:     a.cmdDeclarations,
		Workflow:         a.cmdWorkflow,
		Task:             a.cmdTask,
		Run:              a.cmdRun,
		RunCommands:      a.runCommands(),
		Runner:           a.cmdRunner,
		Policy:           a.cmdPolicy,
		PermissionPolicy: a.cmdPermissionPolicy,
		Settings:         a.cmdSettings,
		Maintenance:      a.cmdMaintenance,
		Ops:              a.cmdOps,
		Health:           a.cmdHealth,
		Events:           a.cmdEvents,
		Findings:         a.cmdFindings,
		Subscription:     a.cmdSubscription,
		ScenarioSmoke:    a.cmdScenarioSmoke,
	}
}

func (a *App) runCommands() []cliapp.Command {
	type entry struct {
		name, description, usage string
		run                      func([]string) error
	}
	entries := []entry{
		{"list", "List runs", "agent-manager run list [options]", a.runList},
		{"get", "Get a run", "agent-manager run get <id> [--json]", a.runGet},
		{"report", "Show bounded investigation diagnostics", "agent-manager run report <id> [--json]", a.runReport},
		{"recent", "Show recent work and evidence-bounded durability", "agent-manager run recent [--limit n] [--json]", a.runRecent},
		{"stats", "Show filtered aggregate run statistics", "agent-manager run stats [--profile UUID] [--since RFC3339] [--tag-prefix prefix]", a.runStats},
		{"result", "Show final-output and structured-result provenance", "agent-manager run result <id>", a.runResult},
		{"cohort-report", "Show ranked bounded cohort evidence", "agent-manager run cohort-report --run-ids id1,id2 [--json]", a.runCohortReport},
		{"goal-cohort", "Fold goal progression across a durable cohort", "agent-manager run goal-cohort --cohort name [--json]", a.runGoalCohort},
		{"cohort-compare", "Compare episode fingerprints across two durable populations", "agent-manager run cohort-compare --left-filter-json '{}' --right-filter-json '{}' [--limit n] [--json]", a.runCohortCompare},
		{"invocation-facts", "Show redacted normalized invocation evidence", "agent-manager run invocation-facts <id> [--json]", a.runInvocationFacts},
		{"episodes", "Show bounded friction episodes", "agent-manager run episodes <id> [--json]", a.runEpisodes},
		{"messages-friction", "Show deterministic self-reported friction spans", "agent-manager run messages-friction <id> [--json]", a.runMessageFriction},
		{"episode-cohort", "Fold episodes across a filtered run cohort", "agent-manager run episode-cohort [--tag-prefix value] [--limit n] [--json]", a.runEpisodeCohort},
		{"episode-trend", "Trend recurring episode fingerprints over time", "agent-manager run episode-trend [--from RFC3339] [--to RFC3339] [--bucket 24h] [--json]", a.runEpisodeTrend},
		{"publish-recurring-friction", "Route recurring deterministic friction to the meta-optimization inbox", "agent-manager run publish-recurring-friction [--cap n] [--json]", a.runPublishRecurringFriction},
		{"ledger", "Show cross-scenario receipt ledger", "agent-manager run ledger <id> [--with-projections] [--json]", a.runLedger},
		{"import-transcript", "Adopt an external harness transcript as a read-only run", "agent-manager run import-transcript <path> [--runner type] [--label value] [--json]", a.runImportTranscript},
		{"import-session-corpus", "Import a bounded, reproducible governed runner-session corpus", "agent-manager run import-session-corpus [--runners codex,claude-code] [--from RFC3339] [--to RFC3339] [--per-month n] [--limit n] [--json]", a.runImportSessionCorpus},
		{"backfill-labels", "Recover labels from retained imported transcripts", "agent-manager run backfill-labels [--json]", a.runBackfillLabels},
		{"backfill-subjects", "Project run subjects from retained invocation facts", "agent-manager run backfill-subjects [--json]", a.runBackfillSubjects},
		{"mine-self-report-vocabulary", "Mine offline self-report phrase candidates for human review", "agent-manager run mine-self-report-vocabulary [--output review.json] <transcript.jsonl> ...", a.runMineSelfReportVocabulary},
		{"replay-invocation-facts", "Rebuild durable invocation evidence from retained events", "agent-manager run replay-invocation-facts <id> [--json]", a.runReplayInvocationFacts},
		{"refresh-invocation-facts", "Refresh durable invocation evidence when events advanced", "agent-manager run refresh-invocation-facts <id> [--json]", a.runRefreshInvocationFacts},
		{"replay-invocation-corpus", "Rebuild or refresh a filtered durable invocation corpus", "agent-manager run replay-invocation-corpus [--from RFC3339] [--to RFC3339] [--refresh]", a.runReplayInvocationCorpus},
		{"invocation-aggregate", "Aggregate durable invocation evidence", "agent-manager run invocation-aggregate --dimension value [--json]", a.runAggregateInvocationFacts},
		{"invocation-cohort", "Select a durable invocation cohort", "agent-manager run invocation-cohort [--json]", a.runSelectInvocationCohort},
		{"invocation-metrics", "Calculate durable invocation metrics", "agent-manager run invocation-metrics [--json]", a.runInvocationMetrics},
		{"cohort", "Define and inspect durable named cohorts", "agent-manager run cohort <define|list|show|delete> ...", a.runCohort},
		{"cross-scenario", "Calculate declared cross-scenario command semantics", "agent-manager run cross-scenario <run-id> [--json]", a.runCrossScenario},
		{"tools", "Show tool calls and failures", "agent-manager run tools <id> [--failed]", a.runTools},
		{"messages", "Show recorded agent messages", "agent-manager run messages <id> [--all] [--range start:end] [--grep text]", a.runMessages},
		{"receipts", "Show observed receipt state and evidence", "agent-manager run receipts <id> [--json]", a.runReceipts},
		{"get-by-tag", "Get a run by tag", "agent-manager run get-by-tag <tag>", a.runGetByTag},
		{"create", "Create and start a run", "agent-manager run create [options]", a.runCreate},
		{"delete", "Delete a run", "agent-manager run delete <id>", a.runDelete},
		{"stop", "Stop a run", "agent-manager run stop <id>", a.runStop},
		{"stop-by-tag", "Stop runs by tag", "agent-manager run stop-by-tag <tag>", a.runStopByTag},
		{"stop-all", "Stop all matching runs", "agent-manager run stop-all [options]", a.runStopAll},
		{"quiesce", "Drain in-flight scenario runs", "agent-manager run quiesce --scenario <scenario>", a.runQuiesce},
		{"continue", "Continue a run", "agent-manager run continue <id> --message <message>", a.runContinue},
		{"park", "Park a run", "agent-manager run park <id>", a.runPark},
		{"wake", "Wake a parked run", "agent-manager run wake <id>", a.runWake},
		{"await-result", "Read a run's awaited result", "agent-manager run await-result <id>", a.runAwaitResult},
		{"recover", "Reconcile a run", "agent-manager run recover <id>", a.runRecover},
		{"investigate", "Create an investigation run", "agent-manager run investigate [options]", a.runInvestigate},
		{"apply-investigation", "Apply investigation recommendations", "agent-manager run apply-investigation <id>", a.runApplyInvestigation},
		{"sandbox-sync", "Sync run state from sandbox", "agent-manager run sandbox-sync <id>", a.runSandboxSync},
		{"approve", "Approve run changes", "agent-manager run approve <id>", a.runApprove},
		{"reject", "Reject run changes", "agent-manager run reject <id>", a.runReject},
		{"diff", "Show sandbox diff", "agent-manager run diff <id> [--stat]", a.runDiff},
		{"events", "Show run events", "agent-manager run events <id> [--stats] [--failed]", a.runEvents},
	}
	commands := make([]cliapp.Command, 0, len(entries))
	for _, item := range entries {
		// cliapp dispatches registered subcommands directly, bypassing cmdRun.
		// Keep the run-identity boundary here as well as in cmdRun so both the
		// legacy dispatcher and the discoverable command tree enforce it.
		name, run := item.name, item.run
		command := support.Command(name, item.description, func(args []string) error {
			if err := rejectRunIdentityLifecycleCommand(name); err != nil {
				return err
			}
			return run(args)
		})
		command.Usage = item.usage
		commands = append(commands, command)
	}
	return commands
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

// apiError enriches a failed API call's error with the human-readable message
// (and recovery hint) the agent-manager API returns in the JSON error body.
// The shared cliutil client parses the legacy `error` key, but agent-manager's
// proto error envelope carries the message under `message` (+ a hint under
// details.fields.hint), so a validation rejection would otherwise surface only
// as "api error (400): HTTP 400". The full body is preserved on the typed
// cliutil.APIError (RawResponse); when it has no usable message the original
// transport error is returned unchanged. The optional body argument (returned
// by service wrappers) is used as a fallback source.
func apiError(body []byte, err error) error {
	if err == nil {
		return nil
	}
	raw := body
	var apiErr *cliutil.APIError
	if errors.As(err, &apiErr) && len(apiErr.RawResponse) > 0 {
		raw = apiErr.RawResponse
	}
	var parsed struct {
		Message string `json:"message"`
		Details struct {
			Fields struct {
				Hint struct {
					StringValue string `json:"string_value"`
				} `json:"hint"`
			} `json:"fields"`
		} `json:"details"`
	}
	if jsonErr := json.Unmarshal(raw, &parsed); jsonErr == nil && strings.TrimSpace(parsed.Message) != "" {
		msg := strings.TrimSpace(parsed.Message)
		if hint := strings.TrimSpace(parsed.Details.Fields.Hint.StringValue); hint != "" && !strings.Contains(msg, hint) {
			msg = msg + "\nhint: " + hint
		}
		return errors.New(msg)
	}
	return err
}

func parseExecutionMode(value string) domainpb.ExecutionMode {
	switch strings.ToLower(value) {
	case "codec_pipe", "codec-pipe":
		return domainpb.ExecutionMode_EXECUTION_MODE_CODEC_PIPE
	case "interactive":
		return domainpb.ExecutionMode_EXECUTION_MODE_INTERACTIVE
	default:
		return domainpb.ExecutionMode_EXECUTION_MODE_UNSPECIFIED
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
