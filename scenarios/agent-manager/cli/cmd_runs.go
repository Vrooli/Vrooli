package main

import (
	"fmt"
)

func (a *App) cmdRun(args []string) error {
	if len(args) == 0 {
		return nil
	}
	if err := rejectRunIdentityLifecycleCommand(args[0]); err != nil {
		return err
	}

	switch args[0] {
	case "list":
		return a.runList(args[1:])
	case "get":
		return a.runGet(args[1:])
	case "identity":
		return a.runIdentity(args[1:])
	case "report":
		return a.runReport(args[1:])
	case "attach":
		return a.runAttach(args[1:])
	case "detach":
		return a.runDetach(args[1:])
	case "recent":
		return a.runRecent(args[1:])
	case "cohort-report":
		return a.runCohortReport(args[1:])
	case "goal-cohort":
		return a.runGoalCohort(args[1:])
	case "cohort-compare":
		return a.runCohortCompare(args[1:])
	case "invocation-facts":
		return a.runInvocationFacts(args[1:])
	case "episodes":
		return a.runEpisodes(args[1:])
	case "messages-friction":
		return a.runMessageFriction(args[1:])
	case "episode-cohort":
		return a.runEpisodeCohort(args[1:])
	case "episode-trend":
		return a.runEpisodeTrend(args[1:])
	case "publish-recurring-friction":
		return a.runPublishRecurringFriction(args[1:])
	case "ledger":
		return a.runLedger(args[1:])
	case "import-transcript":
		return a.runImportTranscript(args[1:])
	case "import-session-corpus":
		return a.runImportSessionCorpus(args[1:])
	case "import-sweep":
		return a.runImportSweep(args[1:])
	case "backfill-labels":
		return a.runBackfillLabels(args[1:])
	case "backfill-subjects":
		return a.runBackfillSubjects(args[1:])
	case "mine-self-report-vocabulary":
		return a.runMineSelfReportVocabulary(args[1:])
	case "replay-invocation-facts":
		return a.runReplayInvocationFacts(args[1:])
	case "refresh-invocation-facts":
		return a.runRefreshInvocationFacts(args[1:])
	case "replay-invocation-corpus":
		return a.runReplayInvocationCorpus(args[1:])
	case "invocation-aggregate":
		return a.runAggregateInvocationFacts(args[1:])
	case "invocation-cohort":
		return a.runSelectInvocationCohort(args[1:])
	case "invocation-metrics":
		return a.runInvocationMetrics(args[1:])
	case "stats":
		return a.runStats(args[1:])
	case "efficiency":
		return a.runEfficiency(args[1:])
	case "result":
		return a.runResult(args[1:])
	case "tools":
		return a.runTools(args[1:])
	case "messages":
		return a.runMessages(args[1:])
	case "receipts":
		return a.runReceipts(args[1:])
	case "get-by-tag":
		return a.runGetByTag(args[1:])
	case "create":
		return a.runCreate(args[1:])
	case "delete":
		return a.runDelete(args[1:])
	case "stop":
		return a.runStop(args[1:])
	case "stop-by-tag":
		return a.runStopByTag(args[1:])
	case "stop-all":
		return a.runStopAll(args[1:])
	case "quiesce":
		return a.runQuiesce(args[1:])
	case "continue":
		return a.runContinue(args[1:])
	case "park":
		return a.runPark(args[1:])
	case "tokens":
		return a.runTokens(args[1:])
	case "wake":
		return a.runWake(args[1:])
	case "await-result":
		return a.runAwaitResult(args[1:])
	case "recover":
		return a.runRecover(args[1:])
	case "investigate":
		return a.runInvestigate(args[1:])
	case "apply-investigation":
		return a.runApplyInvestigation(args[1:])
	case "sandbox-sync":
		return a.runSandboxSync(args[1:])
	case "approve":
		return a.runApprove(args[1:])
	case "reject":
		return a.runReject(args[1:])
	case "diff":
		return a.runDiff(args[1:])
	case "events":
		return a.runEvents(args[1:])
	case "help", "-h", "--help":
		return nil
	default:
		return fmt.Errorf("unknown run subcommand: %s\n\nRun 'agent-manager run help' for usage", args[0])
	}
}
