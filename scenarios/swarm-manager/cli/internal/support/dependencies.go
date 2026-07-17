package support

import "github.com/vrooli/cli-core/cliapp"

type CommandFunc func(args []string) error

type Dependencies struct {
	Overview                CommandFunc
	PortfolioBrief          CommandFunc
	MigrateWorkshop         CommandFunc
	BacklogList             CommandFunc
	BacklogPendingQuestions CommandFunc
	BacklogGet              CommandFunc
	BacklogCreate           CommandFunc
	BacklogUpdate           CommandFunc
	BacklogDelete           CommandFunc
	BacklogDismiss          CommandFunc
	BacklogWorkshopReset    CommandFunc
	BacklogReWorkshop       CommandFunc
	BacklogFiles            CommandFunc
	BacklogFileGet          CommandFunc
	BacklogFileUpload       CommandFunc
	BacklogProcess          CommandFunc
	BacklogQueue            CommandFunc
	BacklogResearch         CommandFunc
	BacklogPromptTrace      CommandFunc
	BacklogBatchCreate      CommandFunc
	BacklogBatchQueue       CommandFunc
	BacklogExport           CommandFunc
	BacklogImport           CommandFunc
	BacklogClarify          CommandFunc
	BacklogClarifyGet       CommandFunc
	BacklogClarifyNext      CommandFunc
	BacklogClarifyAction    CommandFunc
	BacklogReviewDecide     CommandFunc
	BacklogRecoverReview    CommandFunc
	BacklogRetry            CommandFunc
	ScenariosList           CommandFunc
	ScenariosGet            CommandFunc
	ScenariosFixes          CommandFunc
	ScenariosUpdate         CommandFunc
	ScenariosDelete         CommandFunc
	ScenariosFiles          CommandFunc
	ScenariosSpecSync       CommandFunc
	ScenariosStart          CommandFunc
	ScenariosStop           CommandFunc
	ScenariosRestart        CommandFunc
	ScenariosReviewQueue    CommandFunc
	SettingsGet             CommandFunc
	SettingsUpdate          CommandFunc
	QueueList               CommandFunc
	QueueCreate             CommandFunc
	QueueDelete             CommandFunc
	ExecutionList           CommandFunc
	ExecutionGet            CommandFunc
	ExecutionCreate         CommandFunc
	ExecutionPolicyGet      CommandFunc
	ExecutionPolicyPut      CommandFunc
	ExecutionPromptTrace    CommandFunc
	ExecutionStart          CommandFunc
	ExecutionCancel         CommandFunc
	ExecutionRetry          CommandFunc
	CircuitBreakerReset     CommandFunc
	ReviewList              CommandFunc
	ReviewVerify            CommandFunc
	ReviewRequest           CommandFunc
	ReviewTrigger           CommandFunc
	PromptsCatalog          CommandFunc
	PromptsSkills           CommandFunc
	PromptsSkillGet         CommandFunc
	PromptsSkillUpdate      CommandFunc
	PromptsSkillVersions    CommandFunc
	PromptsSkillRevert      CommandFunc
	PromptsPreview          CommandFunc
	PromptsSimulate         CommandFunc
	PromptsExperiment       CommandFunc
	InitiativesList         CommandFunc
	InitiativesGet          CommandFunc
	InitiativesContext      CommandFunc
	InitiativesCandidates   CommandFunc
	InitiativesCreate       CommandFunc
	InitiativesUpdate       CommandFunc
	InitiativesDelete       CommandFunc
	InitiativesAddItems     CommandFunc
	InitiativesRemove       CommandFunc
	InitiativesFiles        CommandFunc
	InitiativesFileGet      CommandFunc
	InitiativesFileUp       CommandFunc
	InitiativesFileOp       CommandFunc
	// Initiative feedback round CLI (see cmd_initiatives_feedback.go).
	InitiativesFeedbackList     CommandFunc
	InitiativesFeedbackGet      CommandFunc
	InitiativesFeedbackSubmit   CommandFunc
	InitiativesFeedbackContinue CommandFunc
	InitiativesFeedbackDecide   CommandFunc
	InitiativesFeedbackCancel   CommandFunc
	InitiativesFeedbackDelete   CommandFunc
	InitiativesFeedbackLock     CommandFunc
	// Initiative review CLI (see cmd_initiatives_review.go).
	InitiativesReviewList      CommandFunc
	InitiativesReviewGet       CommandFunc
	InitiativesReviewTrigger   CommandFunc
	InitiativesReviewDecide    CommandFunc
	InitiativesReviewDecisions CommandFunc
	// Initiative operating-mode CLI (see cmd_initiatives_operating_mode.go).
	InitiativesModeList             CommandFunc
	InitiativesModeWorkspace        CommandFunc
	InitiativesModeSwitch           CommandFunc
	InitiativesModeStart            CommandFunc
	InitiativesModeRefresh          CommandFunc
	InitiativesModeCancel           CommandFunc
	InitiativesModeComplete         CommandFunc
	InitiativesModeApplyBacklogSync CommandFunc
	// Graph projection view (see cmd_initiatives_graph.go).
	InitiativesGraphShow  CommandFunc
	CapturesList          CommandFunc
	CapturesCreate        CommandFunc
	CapturesGet           CommandFunc
	CapturesDelete        CommandFunc
	CapturesClassify      CommandFunc
	RecordsList           CommandFunc
	RecordsGet            CommandFunc
	RecordsCreate         CommandFunc
	RecordsCapture        CommandFunc
	RecordsEdit           CommandFunc
	RecordsSearch         CommandFunc
	RecordsSupersede      CommandFunc
	AgentManagerStatus    CommandFunc
	AgentManagerRunGet    CommandFunc
	AgentManagerRunStop   CommandFunc
	OperationsList        CommandFunc
	OperationsBrief       CommandFunc
	SessionsList          CommandFunc
	SessionsGet           CommandFunc
	SessionsStartupBrief  CommandFunc
	SessionsDelete        CommandFunc
	StatsSummary          CommandFunc
	StatsThroughput       CommandFunc
	StatsBlocking         CommandFunc
	StatsInitiatives      CommandFunc
	StatsAgent            CommandFunc
	StatsSessions         CommandFunc
	StatsSandboxAdoption  CommandFunc
	AISearchStatus        CommandFunc
	AISearchQuery         CommandFunc
	AISearchReconcile     CommandFunc
	AISearchReconcileStat CommandFunc
	AISearchReconcileCan  CommandFunc
	BacklogSearchAI       CommandFunc
	InitiativesSearchAI   CommandFunc
	AutoFilerStatus       CommandFunc
	AutoFilerRunNow       CommandFunc
	EvidenceRun           CommandFunc
	EvidenceEntity        CommandFunc
	EvidenceReconcile     CommandFunc
	EvidenceVerify        CommandFunc
	// Top-level operating-mode catalog CLI (see cmd_operating_mode.go).
	OperatingModeList  CommandFunc
	OperatingModeGet   CommandFunc
	OperatingModeBrief CommandFunc
	OperatingModeSet   CommandFunc
	// Self-serve operating-mode authoring CLI (see cmd_operating_mode_authoring.go).
	OperatingModeScaffold CommandFunc
	OperatingModeValidate CommandFunc
	OperatingModeSimulate CommandFunc
	OperatingModeStart    CommandFunc
	// Agent-operations diagnostics + binding controls CLI (see
	// cmd_agent_operations*.go).
	AgentOpsResolveBinding     CommandFunc
	AgentOpsValidateInvocation CommandFunc
	AgentOpsInspectWorkflow    CommandFunc
	AgentOpsInspectExecution   CommandFunc
	AgentOpsCatalog            CommandFunc
	AgentOpsCompatibleModes    CommandFunc
	AgentOpsBindings           CommandFunc
	AgentOpsOverrides          CommandFunc
	AgentOpsWorkflow           CommandFunc
	AgentOpsHistory            CommandFunc
	AgentOpsMigrationStatus    CommandFunc
	AgentOpsReconcile          CommandFunc
}

func APICommand(name, description string, run CommandFunc) cliapp.Command {
	return cliapp.Command{
		Name:        name,
		NeedsAPI:    true,
		Description: description,
		Run:         run,
	}
}

// APICommandHelp is APICommand plus a --help body (flag reference, examples).
// Use it for commands whose flag surface is too large for the one-line
// description — the description elides what agents then guess wrong.
func APICommandHelp(name, description, helpText string, run CommandFunc) cliapp.Command {
	cmd := APICommand(name, description, run)
	cmd.HelpText = helpText
	return cmd
}
