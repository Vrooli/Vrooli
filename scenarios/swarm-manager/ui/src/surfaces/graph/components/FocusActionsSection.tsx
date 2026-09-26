/**
 * FocusActionsSection — Inline actions for the NodeInspectorPanel in focus lens mode.
 *
 * Renders the primary CTA and a collapsible
 * InlineQuestionStepper for pending decisions. Only renders for entity types
 * with actionable states (backlog, execution, capture).
 */

import { useState, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useActionMutation } from "../../../hooks/useActionMutation";
import { useTransitionKind } from "../../../hooks/useTransitionCatalog";
import { ActionButton } from "../../../components/ui/action-button";
import {
  Play,
  Wrench,
  Eye,
  ChevronRight,
  RefreshCw,
  ClipboardCheck,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { cn } from "../../../lib/utils";
import { defaultApiClient } from "../../../lib/api-client";
import { backlogService, transitionService } from "../../../services";
import { API_ENDPOINTS } from "../../../lib/api-endpoints";
import { useBacklogStore } from "../../../stores/backlog-store";
import { executionDetailPath } from "../../../app/routes/route-paths";
import { RunSheet, type RunSheetTarget } from "../../../components/backlog/run-sheet";
import { InlineQuestionStepper } from "../../../components/backlog/inline-question-stepper";
import { useNodePendingQuestions } from "../hooks/useNodePendingQuestions";
import { parseNodeId } from "../lib/node-id-parser";
import type {
  GraphNodeData,
  BacklogGraphNodeData,
  ExecutionGraphNodeData,
  CaptureGraphNodeData,
} from "../types";

// ---------------------------------------------------------------------------
// CTA icon/label map
// ---------------------------------------------------------------------------

const CTA_CONFIG: Record<string, { label: string; icon: LucideIcon }> = {
  run: { label: "Run", icon: Play },
  followUp: { label: "Follow Up", icon: Wrench },
  archive: { label: "Archive", icon: Eye },
};

// ---------------------------------------------------------------------------
// Backlog actions sub-component
// ---------------------------------------------------------------------------

function BacklogActions({ nodeData, nodeId }: { nodeData: BacklogGraphNodeData; nodeId: string }) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const fetchBacklog = useBacklogStore((s) => s.fetchBacklog);

  const { data: nextAction } = useQuery({
    queryKey: ["backlog", nodeData.kind, nodeData.name, "next-action"],
    queryFn: () => backlogService.getNextAction(nodeData.kind, nodeData.name),
  });
  const pendingQuestions = useNodePendingQuestions(nodeData.kind, nodeData.name);

  const [runModalTarget, setRunModalTarget] = useState<RunSheetTarget | undefined>();
  const [isDecisionsExpanded, setDecisionsExpanded] = useState(false);

  const invalidateAfterAction = useCallback(() => {
    void fetchBacklog({ force: true });
    void queryClient.invalidateQueries({ queryKey: ["backlog-summary"] });
    void queryClient.invalidateQueries({ queryKey: ["backlog-list"] });
  }, [fetchBacklog, queryClient]);

  const archiveMutation = useActionMutation({
    mutationFn: () => defaultApiClient.patch(API_ENDPOINTS.backlogArchiveItem(nodeData.kind, nodeData.name), {}),
    errorMessage: `Couldn't archive ${nodeData.kind}/${nodeData.name}`,
    successMessage: `Archived ${nodeData.kind}/${nodeData.name}`,
    source: "FocusActions.archive",
    onSuccess: invalidateAfterAction,
  });

  const followUpMutation = useActionMutation({
    mutationFn: (executionId: string) =>
      defaultApiClient.post(API_ENDPOINTS.executionFollowUp(executionId), {}),
    errorMessage: "Couldn't start the follow-up run",
    successMessage: "Follow-up run started",
    successKind: "progress",
    source: "FocusActions.followUp",
    onSuccess: invalidateAfterAction,
  });

  const handleCtaClick = useCallback(() => {
    const cta = nextAction?.id;
    if (cta === "run") {
      setRunModalTarget({ kind: nodeData.kind, name: nodeData.name, title: nodeData.title });
    } else if (cta === "retry") {
      // Find the latest execution for this item to follow up on.
      const parsed = parseNodeId(nodeId);
      if (parsed?.identifier) {
        followUpMutation.mutate(parsed.identifier);
      }
    } else if (cta === "archive") {
      archiveMutation.mutate();
    } else if (cta === "author_plan" || cta === "accept_plan" || cta === "repair_plan" || cta === "review" || cta === "resolve_dependencies" || cta === "view_execution") {
      navigate(`/backlog/${nodeData.kind}/${nodeData.name}`);
    }
  }, [nextAction?.id, nodeData, nodeId, navigate, followUpMutation, archiveMutation]);

  // The projection is authoritative, including active-work states.
  if (!nextAction || nextAction.id === "none") return null;
  const displayedAction = nextAction;

  const ctaConfig = {
    label: displayedAction.compactLabel,
    icon: CTA_CONFIG[displayedAction.id === "run" ? "run" : displayedAction.id === "archive" ? "archive" : "followUp"]?.icon ?? Play,
  };
  const isMutating = archiveMutation.isPending || followUpMutation.isPending;
  const showStepper = pendingQuestions.length > 0;

  return (
    <>
      {/* Primary CTA */}
      {ctaConfig && (
        <ActionButton
          actionId={displayedAction.id}
          effect={displayedAction.effect}
          destructive={displayedAction.destructive}
          icon={ctaConfig.icon}
          label={ctaConfig.label}
          pending={isMutating}
          pendingLabel="Working..."
          disabled={!nextAction?.enabled}
          onClick={handleCtaClick}
          className="w-full rounded-lg px-3 py-1.5 text-xs h-auto"
          data-testid="focus-cta-button"
        />
      )}

      {/* Collapsible pending decisions */}
      {showStepper && (
        <div className="flex flex-col gap-1">
          <button
            type="button"
            onClick={() => setDecisionsExpanded(!isDecisionsExpanded)}
            className="flex items-center gap-1.5 rounded px-1 py-0.5 text-xs text-amber-200 transition-colors hover:bg-white/5"
            data-testid="focus-decisions-toggle"
          >
            <ChevronRight
              className={cn("h-3 w-3 transition-transform", isDecisionsExpanded && "rotate-90")}
            />
            {pendingQuestions.length} pending decision{pendingQuestions.length !== 1 ? "s" : ""}
          </button>
          {isDecisionsExpanded && (
            <div className="mt-1">
              <InlineQuestionStepper
                questions={pendingQuestions}
                backlogKind={nodeData.kind}
                backlogName={nodeData.name}
                onAllAnswered={() => {
                  setDecisionsExpanded(false);
                  invalidateAfterAction();
                }}
              />
            </div>
          )}
        </div>
      )}

      <RunSheet
        isOpen={!!runModalTarget}
        onClose={() => setRunModalTarget(undefined)}
        target={runModalTarget}
        onSuccess={() => {
          setRunModalTarget(undefined);
          invalidateAfterAction();
        }}
      />
    </>
  );
}

// ---------------------------------------------------------------------------
// Execution actions sub-component
// ---------------------------------------------------------------------------

function ExecutionActions({ nodeData }: { nodeData: ExecutionGraphNodeData }) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const fetchBacklog = useBacklogStore((s) => s.fetchBacklog);

  const invalidateAfterAction = useCallback(() => {
    void fetchBacklog({ force: true });
    void queryClient.invalidateQueries({ queryKey: ["backlog-summary"] });
    void queryClient.invalidateQueries({ queryKey: ["backlog-list"] });
    void queryClient.invalidateQueries({ queryKey: ["executions"] });
    void queryClient.invalidateQueries({ queryKey: ["execution", nodeData.executionId] });
  }, [fetchBacklog, nodeData.executionId, queryClient]);

  const retryMutation = useActionMutation({
    mutationFn: () => defaultApiClient.post(API_ENDPOINTS.executionRetry(nodeData.executionId), {}),
    errorMessage: "Couldn't retry this execution",
    successMessage: "Retry started",
    successKind: "progress",
    source: "FocusActions.retry",
    onSuccess: invalidateAfterAction,
  });

  const triggerReviewMutation = useActionMutation({
    mutationFn: () => defaultApiClient.post(API_ENDPOINTS.executionTriggerReview(nodeData.executionId), {}),
    errorMessage: "Couldn't run the checks",
    successMessage: "Checks started",
    successKind: "progress",
    source: "FocusActions.triggerReview",
    onSuccess: invalidateAfterAction,
  });

  if (nodeData.status === "needs_review") {
    return (
      <button
        type="button"
        onClick={() => navigate(executionDetailPath(nodeData.executionId))}
        className="flex w-full items-center justify-center gap-1.5 rounded-lg bg-cyan-600/80 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-cyan-600"
        data-testid="focus-review-button"
      >
        <Eye className="h-3 w-3" />
        Review
      </button>
    );
  }

  if (nodeData.status === "completed" || nodeData.status === "needs_fixup") {
    return (
      <ActionButton
        actionId="review"
        icon={Eye}
        label="Run Checks"
        pendingLabel="Running..."
        pending={triggerReviewMutation.isPending}
        onClick={() => triggerReviewMutation.run()}
        className="w-full rounded-lg px-3 py-1.5 text-xs h-auto"
        data-testid="focus-run-checks-button"
      />
    );
  }

  if (nodeData.status === "failed") {
    return (
      <ActionButton
        actionId="retry"
        icon={RefreshCw}
        label="Retry"
        pendingLabel="Retrying..."
        pending={retryMutation.isPending}
        onClick={() => retryMutation.run()}
        className="w-full rounded-lg px-3 py-1.5 text-xs h-auto"
        data-testid="focus-retry-button"
      />
    );
  }

  return null;
}

// ---------------------------------------------------------------------------
// Capture actions sub-component
// ---------------------------------------------------------------------------

function CaptureActions({ nodeData }: { nodeData: CaptureGraphNodeData }) {
  const captureClassifyKind = useTransitionKind("capture.classify");
  const classifyMutation = useActionMutation({
    mutationFn: () => transitionService.start("capture.classify", nodeData.id),
    errorMessage: "Couldn't classify this capture",
    successMessage: "Classification started",
    successKind: "progress",
    invalidateKeys: [["backlog-summary"]],
    source: "CaptureActions.classify",
  });

  if (nodeData.status !== "classifying") return null;

  return (
    <ActionButton
      actionId="classify"
      transitionKind={captureClassifyKind}
      icon={ClipboardCheck}
      label="Classify"
      pendingLabel="Classifying..."
      pending={classifyMutation.isPending}
      onClick={() => classifyMutation.run()}
      className="w-full rounded-lg px-3 py-1.5 text-xs h-auto"
      data-testid="focus-classify-button"
    />
  );
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

interface FocusActionsSectionProps {
  nodeData: GraphNodeData;
  nodeId: string;
}

export function FocusActionsSection({ nodeData, nodeId }: FocusActionsSectionProps) {
  return (
    <div className="flex flex-col gap-2" data-testid="focus-actions-section">
      {nodeData.entityType === "backlog" && (
        <BacklogActions nodeData={nodeData as BacklogGraphNodeData} nodeId={nodeId} />
      )}
      {nodeData.entityType === "execution" && (
        <ExecutionActions nodeData={nodeData} />
      )}
      {nodeData.entityType === "capture" && (
        <CaptureActions nodeData={nodeData} />
      )}
    </div>
  );
}
