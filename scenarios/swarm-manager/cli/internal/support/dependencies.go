package support

import "github.com/vrooli/cli-core/cliapp"

type CommandFunc func(args []string) error

type Dependencies struct {
	Overview                    CommandFunc
	IntegrationStatus           CommandFunc
	PortfolioBrief              CommandFunc
	BacklogList                 CommandFunc
	BacklogPendingQuestions     CommandFunc
	BacklogGet                  CommandFunc
	BacklogCreate               CommandFunc
	BacklogUpdate               CommandFunc
	BacklogDelete               CommandFunc
	BacklogPlanWorkshop         CommandFunc
	BacklogRecreate             CommandFunc
	BacklogResetArtifacts       CommandFunc
	BacklogFiles                CommandFunc
	BacklogFileGet              CommandFunc
	BacklogFileUpload           CommandFunc
	BacklogProcess              CommandFunc
	BacklogQueue                CommandFunc
	BacklogPlanAccept           CommandFunc
	BacklogBatchCreate          CommandFunc
	BacklogBatchQueue           CommandFunc
	BacklogExport               CommandFunc
	BacklogReconcileCounts      CommandFunc
	BacklogImport               CommandFunc
	BacklogReviewDecide         CommandFunc
	BacklogRecoverReview        CommandFunc
	BacklogRetry                CommandFunc
	ScenariosList               CommandFunc
	ScenariosGet                CommandFunc
	ScenariosFixes              CommandFunc
	ScenariosUpdate             CommandFunc
	ScenariosDelete             CommandFunc
	ScenariosFiles              CommandFunc
	ScenariosSpecSync           CommandFunc
	ScenariosStart              CommandFunc
	ScenariosStop               CommandFunc
	ScenariosRestart            CommandFunc
	ScenariosReviewQueue        CommandFunc
	SettingsGet                 CommandFunc
	SettingsUpdate              CommandFunc
	QueueList                   CommandFunc
	QueueCreate                 CommandFunc
	QueueDelete                 CommandFunc
	ExecutionList               CommandFunc
	ExecutionGet                CommandFunc
	ExecutionCreate             CommandFunc
	ExecutionPolicyGet          CommandFunc
	ExecutionPolicyPut          CommandFunc
	ExecutionPromptTrace        CommandFunc
	ExecutionStart              CommandFunc
	ExecutionCancel             CommandFunc
	ExecutionRetry              CommandFunc
	CircuitBreakerReset         CommandFunc
	ReviewList                  CommandFunc
	ReviewVerify                CommandFunc
	ReviewRequest               CommandFunc
	ReviewTrigger               CommandFunc
	PromptsCatalog              CommandFunc
	PromptsSkills               CommandFunc
	PromptsSkillGet             CommandFunc
	PromptsSkillUpdate          CommandFunc
	PromptsSkillVersions        CommandFunc
	PromptsSkillRevert          CommandFunc
	PromptsPreview              CommandFunc
	PromptsExperiment           CommandFunc
	GoalsList                   CommandFunc
	GoalsGet                    CommandFunc
	GoalsCreate                 CommandFunc
	GoalsUpdate                 CommandFunc
	GoalsDelete                 CommandFunc
	GoalsArchive                CommandFunc
	GoalsUnarchive              CommandFunc
	GoalsContext                CommandFunc
	GoalsTargetsAdd             CommandFunc
	GoalsTargetsRemove          CommandFunc
	GoalsMilestoneCreate        CommandFunc
	GoalsMilestoneUpdate        CommandFunc
	GoalsMilestoneAssign        CommandFunc
	GoalsMilestoneUnassign      CommandFunc
	GoalsMilestoneArchive       CommandFunc
	GoalsPlanRun                CommandFunc
	GoalsDiscoverRun            CommandFunc
	GoalsMilestoneReviewRun     CommandFunc
	GoalsCloseOut               CommandFunc
	GoalsWorkflowPending        CommandFunc
	GoalsWorkflowApply          CommandFunc
	ProposalsList               CommandFunc
	ProposalsGet                CommandFunc
	ProposalsDecide             CommandFunc
	ProposalsAcceptKeep         CommandFunc
	ProposalsRevise             CommandFunc
	CapturesList                CommandFunc
	CapturesCreate              CommandFunc
	CapturesGet                 CommandFunc
	CapturesDelete              CommandFunc
	CapturesClassify            CommandFunc
	RecordsList                 CommandFunc
	RecordsGet                  CommandFunc
	RecordsCreate               CommandFunc
	RecordsCapture              CommandFunc
	RecordsEdit                 CommandFunc
	RecordsSearch               CommandFunc
	RecordsSupersede            CommandFunc
	AgentManagerStatus          CommandFunc
	AgentManagerRunGet          CommandFunc
	AgentManagerRunStop         CommandFunc
	OperationsList              CommandFunc
	OperationsBrief             CommandFunc
	SessionsList                CommandFunc
	SessionsGet                 CommandFunc
	SessionsCreate              CommandFunc
	SessionsCreateBatch         CommandFunc
	SessionsAttach              CommandFunc
	SessionsStart               CommandFunc
	SessionsContinue            CommandFunc
	SessionsComplete            CommandFunc
	SessionsReap                CommandFunc
	SessionsEvents              CommandFunc
	SessionsProposalApply       CommandFunc
	SessionsProposalRevise      CommandFunc
	SessionsProposalWait        CommandFunc
	SessionsProposalAcceptKeep  CommandFunc
	SessionsStartupBrief        CommandFunc
	SessionsPromptPreview       CommandFunc
	SessionsDelete              CommandFunc
	SessionsDisposition         CommandFunc
	StatsSummary                CommandFunc
	StatsThroughput             CommandFunc
	StatsBlocking               CommandFunc
	StatsMilestones             CommandFunc
	StatsAgent                  CommandFunc
	StatsSessions               CommandFunc
	AgentManagerSandboxAdoption CommandFunc
	AISearchStatus              CommandFunc
	AISearchQuery               CommandFunc
	AISearchReconcile           CommandFunc
	AISearchReconcileStat       CommandFunc
	AISearchReconcileCan        CommandFunc
	BacklogSearchAI             CommandFunc
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
