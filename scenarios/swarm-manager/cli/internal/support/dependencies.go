package support

import "github.com/vrooli/cli-core/cliapp"

type CommandFunc func(args []string) error

type Dependencies struct {
	Overview                CommandFunc
	MigrateWorkshop         CommandFunc
	BacklogList             CommandFunc
	BacklogPendingQuestions CommandFunc
	BacklogGet              CommandFunc
	BacklogCreate           CommandFunc
	BacklogUpdate           CommandFunc
	BacklogDelete           CommandFunc
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
	AgentManagerStatus    CommandFunc
	AgentManagerRunGet    CommandFunc
	AgentManagerRunStop   CommandFunc
	SessionsList          CommandFunc
	SessionsGet           CommandFunc
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
	// Top-level operating-mode catalog CLI (see cmd_operating_mode.go).
	OperatingModeList CommandFunc
	OperatingModeGet  CommandFunc
	OperatingModeSet  CommandFunc
}

func APICommand(name, description string, run CommandFunc) cliapp.Command {
	return cliapp.Command{
		Name:        name,
		NeedsAPI:    true,
		Description: description,
		Run:         run,
	}
}
