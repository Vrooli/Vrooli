/**
 * Execution Details Page
 *
 * Tabbed detail view for a single execution record. Tabs:
 * - Overview: metadata, failure reason, post-run status, actions
 * - Changes: sandbox changed files grouped by scenario
 * - Review: post-run checks, scenario reviews, evidence
 * - Prompt: prompt trace
 *
 * Data fetching is delegated to useExecutionDetailData.
 * Tab state is URL-synced via useUrlState.
 */

import { useState } from "react";
import { useParams } from "react-router-dom";
import {
  CircleHelp,
  ClipboardCheck,
  GitCompare,
  Loader2,
  MessageSquarePlus,
  RotateCcw,
  Sparkles,
  XCircle,
} from "lucide-react";
import { Button } from "../components/ui/button";
import { ActionMenuSheetContent, type ActionMenuItem } from "../components/ui/action-menu";
import { Tabs, TabsList, TabsTrigger } from "../components/ui/tabs";
import { DetailPageHeader } from "../components/detail/DetailPageHeader";
import { DetailPageLayout } from "../components/detail/DetailPageLayout";
import { ErrorState } from "../components/ui/error-state";
import { PageLoadingState } from "../components/ui/loading-states";
import { FollowUpSheet } from "../components/review/follow-up-sheet";
import { ExecutionOverviewTab } from "../components/execution/execution-overview-tab";
import { ExecutionChangesTab } from "../components/execution/execution-changes-tab";
import { ExecutionReviewTab } from "../components/execution/execution-review-tab";
import { ExecutionPromptTab } from "../components/execution/execution-prompt-tab";
import { useEmbeddedServiceUrl } from "../hooks/useEmbeddedServiceUrl";
import { useExecutionDetailData } from "../hooks/useExecutionDetailData";
import { useUrlState } from "../hooks/use-url-state";
import { useReviewStore } from "../stores/review-store";
import { reviewService } from "../services/review-service";
import { EvidenceRequestPanel } from "../components/backlog/evidence-request-panel";
import { useQueryClient } from "@tanstack/react-query";
import { EXECUTION_LENSES } from "../components/detail/lens-options";
import { selectors } from "../consts/selectors";
import { canRunPostRunChecks } from "../lib/finalization";
import { ENTITY_TYPE_ICONS } from "../types/constants";
import type { ExecutionRecord } from "../types";
import { routeTargetToNodeId } from "../app/routes/route-paths";
import { useAttachToSessionAction } from "../components/session/context/useAttachToSessionAction";
import { executionOption } from "../components/session/context/session-context-refs";

type ExecutionTab = "overview" | "changes" | "review" | "prompt";

export function ExecutionDetailsPage() {
  const queryClient = useQueryClient();

  // --- Navigation / selection ---
  const { executionId } = useParams<{ executionId: string }>();
  const nodeId = routeTargetToNodeId({ entityType: "execution", identifier: executionId });

  // --- Tab state (URL-synced) ---
  const [activeTab, setActiveTab] = useUrlState<ExecutionTab>("tab", "overview", {
    validate: (v): v is ExecutionTab =>
      ["overview", "changes", "review", "prompt"].includes(v),
  });

  // --- Data ---
  const data = useExecutionDetailData({ executionId });
  const {
    execution,
    trace,
    isTraceLoading,
    reviewRounds,
    isGatheringEvidence,
    isAwaitingManualReview,
    targetScenarios: _targetScenarios,
    isLoading,
    error,
    isActive,
    isTerminal,
    postRunBadgeExecution,
    cancel,
    retry,
    triggerReview,
    refetch,
    actionBusy,
  } = data;
  const attachToSession = useAttachToSessionAction(execution ? executionOption(execution) : null);

  // --- Agent manager URL ---
  const { url: agentManagerUiUrl } = useEmbeddedServiceUrl("agent-manager");

  // --- Follow-up dialog state ---
  const [followUpTarget, setFollowUpTarget] = useState<ExecutionRecord | null>(null);

  // --- Loading / error states ---
  if (isLoading) return <PageLoadingState label="Loading execution..." />;
  if (error || !execution) {
    return (
      <DetailPageLayout
        header={
          <DetailPageHeader
            entityType="execution"
            entityIcon={ENTITY_TYPE_ICONS.execution}
            title={executionId ?? "Unknown"}
            nodeId={null}
            lenses={[]}
          />
        }
      >
        <div className="md:mx-auto md:max-w-3xl">
          <ErrorState
            error={error}
            message={`Could not load execution "${executionId}".`}
            onRetry={refetch}
          />
        </div>
      </DetailPageLayout>
    );
  }

  // --- Header primary action ---
  const runChecksLabel = execution.finalization ? "Rerun Checks" : "Run Checks";
  const primaryAction = isActive ? (
    <Button
      variant="destructive"
      size="sm"
      disabled={actionBusy}
      onClick={() => void cancel()}
    >
      {actionBusy ? <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" /> : <XCircle className="mr-1 h-3.5 w-3.5" />}
      Cancel
    </Button>
  ) : execution.status === "failed" ? (
    <Button
      variant="outline"
      size="sm"
      disabled={actionBusy}
      onClick={() => void retry()}
    >
      {actionBusy ? <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" /> : <RotateCcw className="mr-1 h-3.5 w-3.5" />}
      Retry
    </Button>
  ) : canRunPostRunChecks(execution) ? (
    <Button
      variant="outline"
      size="sm"
      disabled={actionBusy}
      onClick={() => void triggerReview()}
    >
      {actionBusy ? <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" /> : <ClipboardCheck className="mr-1 h-3.5 w-3.5" />}
      {runChecksLabel}
    </Button>
  ) : null;

  // --- Tab bar ---
  const tabBar = (
    <div
      className="border-t border-slate-800/50"
      data-testid={selectors.executionDetails.tabRow}
    >
      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as ExecutionTab)}>
        <TabsList className="w-full flex-nowrap justify-start gap-1 overflow-x-auto no-scrollbar rounded-none bg-transparent p-0 px-3">
          <TabsTrigger
            value="overview"
            className="gap-2"
            data-testid={selectors.executionDetails.tabOverview}
          >
            <CircleHelp className="h-4 w-4" />
            Overview
          </TabsTrigger>
          <TabsTrigger
            value="changes"
            className="gap-2"
            data-testid={selectors.executionDetails.tabChanges}
          >
            <GitCompare className="h-4 w-4" />
            Changes
          </TabsTrigger>
          <TabsTrigger
            value="review"
            className="gap-2"
            data-testid={selectors.executionDetails.tabReview}
          >
            <ClipboardCheck className="h-4 w-4" />
            Review
            {(isGatheringEvidence || isAwaitingManualReview) && (
              <span className="relative flex h-2 w-2">
                {isGatheringEvidence && (
                  <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-cyan-400 opacity-75" />
                )}
                <span className={`relative inline-flex h-2 w-2 rounded-full ${isAwaitingManualReview ? "bg-amber-400" : "bg-cyan-500"}`} />
              </span>
            )}
          </TabsTrigger>
          <TabsTrigger
            value="prompt"
            className="gap-2"
            data-testid={selectors.executionDetails.tabPrompt}
          >
            <Sparkles className="h-4 w-4" />
            Prompt
          </TabsTrigger>
        </TabsList>
      </Tabs>
    </div>
  );

  const mobileActionItems: ActionMenuItem[] = [];
  mobileActionItems.push(attachToSession.actionItem);
  if (isActive) {
    mobileActionItems.push({
      label: "Cancel",
      icon: <XCircle />,
      loading: actionBusy,
      disabled: actionBusy,
      destructive: true,
      onSelect: () => void cancel(),
    });
  } else if (execution.status === "failed") {
    mobileActionItems.push({
      label: "Retry",
      icon: <RotateCcw />,
      loading: actionBusy,
      disabled: actionBusy,
      onSelect: () => void retry(),
    });
  } else if (canRunPostRunChecks(execution)) {
    mobileActionItems.push({
      label: runChecksLabel,
      icon: <ClipboardCheck />,
      loading: actionBusy,
      disabled: actionBusy,
      onSelect: () => void triggerReview(),
    });
  }
  if (isTerminal) {
    mobileActionItems.push({
      label: "Follow-up",
      icon: <MessageSquarePlus />,
      disabled: actionBusy,
      onSelect: () => setFollowUpTarget(execution),
    });
  }

  // --- Mobile actions ---
  const mobileActions = mobileActionItems.length > 0
    ? <ActionMenuSheetContent items={mobileActionItems} />
    : undefined;

  return (
    <DetailPageLayout
      header={
        <DetailPageHeader
          entityType="execution"
          entityIcon={ENTITY_TYPE_ICONS.execution}
          title={`${execution.backlogKind}/${execution.backlogName}`}
          subtitle={execution.operation ?? undefined}
          status={execution.status}
          nodeId={nodeId}
          lenses={EXECUTION_LENSES}
          actions={
            <div className="flex items-center gap-2">
              {primaryAction}
              {attachToSession.button}
            </div>
          }
          tabBar={tabBar}
        />
      }
      mobileActions={mobileActions}
      mobileActionsTitle="Execution Actions"
    >
      {attachToSession.sheet}
      <div className="space-y-0 md:mx-auto md:max-w-3xl">
        {activeTab === "overview" && (
          <ExecutionOverviewTab
            execution={execution}
            isActive={isActive}
            isTerminal={isTerminal}
            actionBusy={actionBusy}
            postRunBadgeExecution={postRunBadgeExecution}
            agentManagerUiUrl={agentManagerUiUrl}
            onFollowUp={() => setFollowUpTarget(execution)}
            onCancel={() => void cancel()}
            onRetry={() => void retry()}
            onRunPostRunChecks={() => void triggerReview()}
          />
        )}
        {activeTab === "changes" && (
          <ExecutionChangesTab
            finalization={execution.finalization}
            isActive={isActive}
          />
        )}
        {activeTab === "review" && (
          <ExecutionReviewTab
            execution={execution}
            reviewRounds={reviewRounds}
            isGatheringEvidence={isGatheringEvidence}
            isAwaitingManualReview={isAwaitingManualReview}
            isActive={isActive}
            agentManagerUiUrl={agentManagerUiUrl}
            onFollowUp={() => setFollowUpTarget(execution)}
            onVerifyEvidence={(round, evidenceId, verified) => {
              void reviewService.verifyEvidence(
                execution.backlogKind,
                execution.backlogName,
                round,
                evidenceId,
                verified,
                execution.executionId,
              );
            }}
            onRequestMoreEvidence={(round, evidenceId) => {
              useReviewStore.getState().openRequestPanel(round, evidenceId);
            }}
          />
        )}
        {activeTab === "prompt" && (
          <ExecutionPromptTab trace={trace} isLoading={isTraceLoading} />
        )}
      </div>

      {/* Evidence request panel */}
      <EvidenceRequestPanel
        backlogKind={execution?.backlogKind ?? ""}
        backlogName={execution?.backlogName ?? ""}
        reviewRounds={reviewRounds}
        onAction={() => void queryClient.invalidateQueries({ queryKey: ["review-rounds", execution?.backlogKind, execution?.backlogName] })}
      />

      {/* Follow-up sheet */}
      {followUpTarget && (
        <FollowUpSheet
          isOpen={Boolean(followUpTarget)}
          onClose={() => setFollowUpTarget(null)}
          execution={followUpTarget}
          reviewRounds={reviewRounds}
          onSuccess={() => {
            setFollowUpTarget(null);
            refetch();
          }}
        />
      )}
    </DetailPageLayout>
  );
}
