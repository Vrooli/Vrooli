// Package main provides the Swarm Manager CLI.
package main

import (
	"net/http"

	"swarm-manager/cli/domains"
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	appName        = "swarm-manager"
	appVersion     = "0.1.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

type App struct {
	core      *cliapp.ScenarioApp
	globalDry bool // set by preflight from --dry-run global flag
	identity  cliutil.IdentityEnv
}

func NewApp() (*App, error) {
	app := &App{}

	// Detect agent identity from environment. When present, wrap the HTTP
	// transport so every outgoing request carries the identity token header.
	identity := cliutil.DetectIdentity()
	app.identity = identity
	var httpClientOpts cliutil.HTTPClientOptions
	if identity.IsIdentityPresent() {
		httpClientOpts.Client = &http.Client{
			Transport: &identityTransport{
				base:  http.DefaultTransport,
				token: identity.Token,
			},
		}
	}

	disableStatus := true
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		HTTPClientOptions:    httpClientOpts,
		Name:                 appName,
		Version:              appVersion,
		Description:          "Swarm Manager CLI",
		DefaultAPIBase:       defaultAPIBase,
		ExtraAPIEnvVars:      []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		BuildFingerprint:     buildFingerprint,
		BuildTimestamp:       buildTimestamp,
		BuildSourceRoot:      buildSourceRoot,
		AllowAnonymous:       true,
		IncludeStatusCommand: &disableStatus,
		Preflight: func(_ cliapp.Command, global cliapp.GlobalOptions, _ *cliapp.ScenarioApp) error {
			app.globalDry = global.DryRun
			return nil
		},
		CommandGroups: func(_ *cliapp.ScenarioApp) []cliapp.CommandGroup {
			return domains.CommandGroups(app.dependencies())
		},
		SubcommandGroups: func(_ *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
			return domains.SubcommandGroups(app.dependencies())
		},
	})
	if err != nil {
		return nil, err
	}

	app.core = core
	return app, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

func (a *App) dependencies() support.Dependencies {
	return support.Dependencies{
		Overview:                        a.cmdOverview,
		MigrateWorkshop:                 a.cmdMigrateWorkshop,
		AISearchStatus:                  a.cmdAISearchStatus,
		AISearchQuery:                   a.cmdAISearchSearch(""),
		AISearchReconcile:               a.cmdAISearchReconcile,
		AISearchReconcileStat:           a.cmdAISearchReconcileStatus,
		AISearchReconcileCan:            a.cmdAISearchReconcileCancel,
		BacklogSearchAI:                 a.cmdAISearchSearch("backlog"),
		InitiativesSearchAI:             a.cmdAISearchSearch("initiative"),
		BacklogList:                     a.cmdBacklogList,
		BacklogPendingQuestions:         a.cmdBacklogPendingQuestions,
		BacklogGet:                      a.cmdBacklogGet,
		BacklogCreate:                   a.cmdBacklogCreate,
		BacklogUpdate:                   a.cmdBacklogUpdate,
		BacklogDelete:                   a.cmdBacklogDelete,
		BacklogWorkshopReset:            a.cmdBacklogWorkshopReset,
		BacklogReWorkshop:               a.cmdBacklogReWorkshop,
		BacklogFiles:                    a.cmdBacklogFiles,
		BacklogFileGet:                  a.cmdBacklogFileGet,
		BacklogFileUpload:               a.cmdBacklogFileUpload,
		BacklogProcess:                  a.cmdBacklogProcessPreflight,
		BacklogQueue:                    a.cmdBacklogQueue,
		BacklogResearch:                 a.cmdBacklogResearch,
		BacklogPromptTrace:              a.cmdBacklogPromptTrace,
		BacklogBatchCreate:              a.cmdBacklogBatchCreate,
		BacklogBatchQueue:               a.cmdBacklogBatchQueue,
		BacklogExport:                   a.cmdBacklogExport,
		BacklogImport:                   a.cmdBacklogImport,
		BacklogClarify:                  a.cmdBacklogClarify,
		BacklogClarifyGet:               a.cmdBacklogClarifyGet,
		BacklogClarifyNext:              a.cmdBacklogClarifyContinue,
		BacklogClarifyAction:            a.cmdBacklogClarifyAction,
		BacklogReviewDecide:             a.cmdBacklogReviewDecide,
		BacklogRetry:                    a.cmdBacklogRetry,
		ScenariosList:                   a.cmdScenariosList,
		ScenariosGet:                    a.cmdScenariosGet,
		ScenariosFixes:                  a.cmdScenariosFixes,
		ScenariosUpdate:                 a.cmdScenariosUpdate,
		ScenariosDelete:                 a.cmdScenariosDelete,
		ScenariosFiles:                  a.cmdScenariosFiles,
		ScenariosSpecSync:               a.cmdScenariosSpecSyncArchive,
		ScenariosStart:                  a.cmdScenariosStart,
		ScenariosStop:                   a.cmdScenariosStop,
		ScenariosRestart:                a.cmdScenariosRestart,
		ScenariosReviewQueue:            a.cmdScenariosReviewQueue,
		SettingsGet:                     a.cmdSettingsGet,
		SettingsUpdate:                  a.cmdSettingsUpdate,
		QueueList:                       a.cmdQueueList,
		QueueCreate:                     a.cmdQueueCreate,
		QueueDelete:                     a.cmdQueueDelete,
		ExecutionList:                   a.cmdExecutionList,
		ExecutionGet:                    a.cmdExecutionGet,
		ExecutionCreate:                 a.cmdExecutionCreate,
		ExecutionPolicyGet:              a.cmdExecutionPolicyGet,
		ExecutionPolicyPut:              a.cmdExecutionPolicyUpdate,
		ExecutionPromptTrace:            a.cmdExecutionPromptTrace,
		ExecutionStart:                  a.cmdExecutionStart,
		ExecutionCancel:                 a.cmdExecutionCancel,
		ExecutionRetry:                  a.cmdExecutionRetry,
		CircuitBreakerReset:             a.cmdCircuitBreakerReset,
		ReviewList:                      a.cmdReviewList,
		ReviewVerify:                    a.cmdReviewVerify,
		ReviewRequest:                   a.cmdReviewRequest,
		ReviewTrigger:                   a.cmdReviewTrigger,
		PromptsCatalog:                  a.cmdPromptsCatalog,
		PromptsSkills:                   a.cmdPromptsSkills,
		PromptsSkillGet:                 a.cmdPromptsSkillGet,
		PromptsSkillUpdate:              a.cmdPromptsSkillUpdate,
		PromptsSkillVersions:            a.cmdPromptsSkillVersions,
		PromptsSkillRevert:              a.cmdPromptsSkillRevert,
		PromptsPreview:                  a.cmdPromptsPreview,
		PromptsSimulate:                 a.cmdPromptsSimulate,
		PromptsExperiment:               a.cmdPromptsExperimentResults,
		InitiativesList:                 a.cmdInitiativesList,
		InitiativesGet:                  a.cmdInitiativesGet,
		InitiativesContext:              a.cmdInitiativesContext,
		InitiativesCreate:               a.cmdInitiativesCreate,
		InitiativesUpdate:               a.cmdInitiativesUpdate,
		InitiativesDelete:               a.cmdInitiativesDelete,
		InitiativesAddItems:             a.cmdInitiativesAddItems,
		InitiativesRemove:               a.cmdInitiativesRemoveItems,
		InitiativesFiles:                a.cmdInitiativesFiles,
		InitiativesFileGet:              a.cmdInitiativesFileGet,
		InitiativesFileUp:               a.cmdInitiativesFileUpload,
		InitiativesFileOp:               a.cmdInitiativesFileOp,
		InitiativesFeedbackList:         a.cmdInitiativesFeedbackList,
		InitiativesFeedbackGet:          a.cmdInitiativesFeedbackGet,
		InitiativesFeedbackSubmit:       a.cmdInitiativesFeedbackSubmit,
		InitiativesFeedbackContinue:     a.cmdInitiativesFeedbackContinue,
		InitiativesFeedbackDecide:       a.cmdInitiativesFeedbackDecide,
		InitiativesFeedbackCancel:       a.cmdInitiativesFeedbackCancel,
		InitiativesFeedbackDelete:       a.cmdInitiativesFeedbackDelete,
		InitiativesFeedbackLock:         a.cmdInitiativesFeedbackLock,
		InitiativesReviewList:           a.cmdInitiativesReviewList,
		InitiativesReviewGet:            a.cmdInitiativesReviewGet,
		InitiativesReviewTrigger:        a.cmdInitiativesReviewTrigger,
		InitiativesReviewDecide:         a.cmdInitiativesReviewDecide,
		InitiativesReviewDecisions:      a.cmdInitiativesReviewDecisions,
		InitiativesModeList:             a.cmdInitiativesModeList,
		InitiativesModeWorkspace:        a.cmdInitiativesModeWorkspace,
		InitiativesModeSwitch:           a.cmdInitiativesModeSwitch,
		InitiativesModeStart:            a.cmdInitiativesModeStart,
		InitiativesModeRefresh:          a.cmdInitiativesModeRefresh,
		InitiativesModeCancel:           a.cmdInitiativesModeCancel,
		InitiativesModeComplete:         a.cmdInitiativesModeComplete,
		InitiativesModeApplyBacklogSync: a.cmdInitiativesModeApplyBacklogSync,
		InitiativesGraphShow:            a.cmdInitiativesGraphShow,
		OperatingModeList:               a.cmdOperatingModeList,
		OperatingModeGet:                a.cmdOperatingModeGet,
		OperatingModeSet:                a.cmdOperatingModeSet,
		CapturesList:                    a.cmdCapturesList,
		CapturesCreate:                  a.cmdCapturesCreate,
		CapturesGet:                     a.cmdCapturesGet,
		CapturesDelete:                  a.cmdCapturesDelete,
		CapturesClassify:                a.cmdCapturesClassify,
		AgentManagerStatus:              a.cmdAgentManagerStatus,
		AgentManagerRunGet:              a.cmdAgentManagerRunGet,
		AgentManagerRunStop:             a.cmdAgentManagerRunStop,
		SessionsList:                    a.cmdSessionsList,
		SessionsGet:                     a.cmdSessionsGet,
		SessionsDelete:                  a.cmdSessionsDelete,
		StatsSummary:                    a.cmdStatsSummary,
		StatsThroughput:                 a.cmdStatsThroughput,
		StatsBlocking:                   a.cmdStatsBlocking,
		StatsInitiatives:                a.cmdStatsInitiatives,
		StatsAgent:                      a.cmdStatsAgent,
		StatsSessions:                   a.cmdStatsSessions,
		StatsSandboxAdoption:            a.cmdStatsSandboxAdoption,
	}
}
