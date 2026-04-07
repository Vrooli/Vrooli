/**
 * FocusActionsSection — Inline actions for the NodeInspectorPanel in focus lens mode.
 *
 * Renders the primary CTA (Run/Workshop/Finalize/etc.) and a collapsible
 * InlineQuestionStepper for pending decisions. Only renders for entity types
 * with actionable states (backlog, execution, capture).
 */

import { useState, useCallback } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Play,
  MessageCircle,
  Sparkles,
  Wrench,
  Eye,
  ChevronRight,
  RefreshCw,
  ClipboardCheck,
} from "lucide-react";
import { cn } from "../../../lib/utils";
import { defaultApiClient } from "../../../lib/api-client";
import { API_ENDPOINTS } from "../../../lib/api-endpoints";
import { backlogService } from "../../../services";
import { useBacklogStore } from "../../../stores/backlog-store";
import { useDetailSelectionStore } from "../../../stores/detail-selection-store";
import { RunBacklogModal, type RunBacklogTarget } from "../../../components/backlog/run-backlog-modal";
import { InlineQuestionStepper } from "../../../components/backlog/inline-question-stepper";
import { useNodeActionContext } from "../hooks/useNodeActionContext";
import { useNodePendingQuestions } from "../hooks/useNodePendingQuestions";
import { parseNodeId } from "../lib/node-id-parser";
import type {
  GraphNodeData,
  BacklogGraphNodeData,
  ExecutionGraphNodeData,
  CaptureGraphNodeData,
} from "../types";

// ---------------------------------------------------------------------------
// CTA icon/label map — mirrors ActionFeedItem.CTA_CONFIG
// ---------------------------------------------------------------------------

const CTA_CONFIG: Record<string, { label: string; icon: React.ElementType }> = {
  run: { label: "Run", icon: Play },
  workshop: { label: "Workshop", icon: MessageCircle },
  finalize: { label: "Finalize", icon: Sparkles },
  followUp: { label: "Follow Up", icon: Wrench },
  archive: { label: "Archive", icon: Eye },
};

// ---------------------------------------------------------------------------
// Backlog actions sub-component
// ---------------------------------------------------------------------------

function BacklogActions({ nodeData, nodeId }: { nodeData: BacklogGraphNodeData; nodeId: string }) {
  const queryClient = useQueryClient();
  const fetchBacklog = useBacklogStore((s) => s.fetchBacklog);
  const selectBacklog = useDetailSelectionStore((s) => s.selectBacklog);

  const itemActions = useNodeActionContext(nodeData);
  const pendingQuestions = useNodePendingQuestions(nodeData.kind, nodeData.name);

  const [runModalTarget, setRunModalTarget] = useState<RunBacklogTarget | undefined>();
  const [isDecisionsExpanded, setDecisionsExpanded] = useState(false);

  const invalidateAfterAction = useCallback(() => {
    void fetchBacklog({ force: true });
    void queryClient.invalidateQueries({ queryKey: ["backlog-summary"] });
    void queryClient.invalidateQueries({ queryKey: ["backlog-list"] });
  }, [fetchBacklog, queryClient]);

  const archiveMutation = useMutation({
    mutationFn: () => defaultApiClient.patch(API_ENDPOINTS.backlogArchiveItem(nodeData.kind, nodeData.name), {}),
    onSuccess: invalidateAfterAction,
  });

  const followUpMutation = useMutation({
    mutationFn: (executionId: string) =>
      defaultApiClient.post(API_ENDPOINTS.executionFollowUp(executionId), {}),
    onSuccess: invalidateAfterAction,
  });

  const handleCtaClick = useCallback(() => {
    const cta = itemActions.primaryCta;
    if (cta === "run") {
      setRunModalTarget({ kind: nodeData.kind, name: nodeData.name, title: nodeData.title });
    } else if (cta === "workshop" || cta === "finalize") {
      selectBacklog(nodeData.kind, nodeData.name);
    } else if (cta === "followUp") {
      // Find the latest execution for this item to follow up on.
      const parsed = parseNodeId(nodeId);
      if (parsed?.identifier) {
        followUpMutation.mutate(parsed.identifier);
      }
    } else if (cta === "archive") {
      archiveMutation.mutate();
    }
  }, [itemActions.primaryCta, nodeData, nodeId, selectBacklog, followUpMutation, archiveMutation]);

  // Nothing to show for locked items.
  if (itemActions.locked) return null;

  const ctaConfig = itemActions.primaryCta ? CTA_CONFIG[itemActions.primaryCta] : null;
  const isMutating = archiveMutation.isPending || followUpMutation.isPending;
  const showStepper = pendingQuestions.length > 0;

  return (
    <>
      {/* Primary CTA */}
      {ctaConfig && (
        <button
          type="button"
          onClick={handleCtaClick}
          disabled={isMutating}
          className="flex w-full items-center justify-center gap-1.5 rounded-lg bg-cyan-600/80 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-cyan-600 disabled:opacity-50"
          data-testid="focus-cta-button"
        >
          <ctaConfig.icon className="h-3 w-3" />
          {isMutating ? "Working..." : ctaConfig.label}
        </button>
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

      {/* Run modal */}
      <RunBacklogModal
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
  const queryClient = useQueryClient();
  const fetchBacklog = useBacklogStore((s) => s.fetchBacklog);
  const selectExecution = useDetailSelectionStore((s) => s.selectExecution);

  const invalidateAfterAction = useCallback(() => {
    void fetchBacklog({ force: true });
    void queryClient.invalidateQueries({ queryKey: ["backlog-summary"] });
    void queryClient.invalidateQueries({ queryKey: ["backlog-list"] });
  }, [fetchBacklog, queryClient]);

  const retryMutation = useMutation({
    mutationFn: () => defaultApiClient.post(API_ENDPOINTS.executionRetry(nodeData.executionId), {}),
    onSuccess: invalidateAfterAction,
  });

  if (nodeData.status === "needs_review" || nodeData.status === "needs_fixup") {
    return (
      <button
        type="button"
        onClick={() => selectExecution(nodeData.executionId)}
        className="flex w-full items-center justify-center gap-1.5 rounded-lg bg-cyan-600/80 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-cyan-600"
        data-testid="focus-review-button"
      >
        <Eye className="h-3 w-3" />
        Review
      </button>
    );
  }

  if (nodeData.status === "failed") {
    return (
      <button
        type="button"
        onClick={() => retryMutation.mutate()}
        disabled={retryMutation.isPending}
        className="flex w-full items-center justify-center gap-1.5 rounded-lg bg-cyan-600/80 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-cyan-600 disabled:opacity-50"
        data-testid="focus-retry-button"
      >
        <RefreshCw className="h-3 w-3" />
        {retryMutation.isPending ? "Retrying..." : "Retry"}
      </button>
    );
  }

  return null;
}

// ---------------------------------------------------------------------------
// Capture actions sub-component
// ---------------------------------------------------------------------------

function CaptureActions({ nodeData }: { nodeData: CaptureGraphNodeData }) {
  const queryClient = useQueryClient();

  const classifyMutation = useMutation({
    mutationFn: () => defaultApiClient.post(API_ENDPOINTS.captureClassify(nodeData.id), {}),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["backlog-summary"] });
    },
  });

  if (nodeData.status !== "classifying") return null;

  return (
    <button
      type="button"
      onClick={() => classifyMutation.mutate()}
      disabled={classifyMutation.isPending}
      className="flex w-full items-center justify-center gap-1.5 rounded-lg bg-cyan-600/80 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-cyan-600 disabled:opacity-50"
      data-testid="focus-classify-button"
    >
      <ClipboardCheck className="h-3 w-3" />
      {classifyMutation.isPending ? "Classifying..." : "Classify"}
    </button>
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
        <ExecutionActions nodeData={nodeData as ExecutionGraphNodeData} />
      )}
      {nodeData.entityType === "capture" && (
        <CaptureActions nodeData={nodeData as CaptureGraphNodeData} />
      )}
    </div>
  );
}
