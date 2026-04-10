import type { GraphNodeData } from "../types";

export type NodeAttentionReason =
  | "pending-decisions"
  | "ready-to-run"
  | "needs-review"
  | "failed"
  | "needs-classification";

export interface NodeEnrichment {
  pendingDecisions?: number;
  /** null = unknown, true = ready, false = not ready (needs workshop). */
  maturityReady?: boolean | null;
  /** Latest workshop answers need synthesis (finalize). */
  pendingSynthesis?: boolean;
}

export interface NodeAttentionResult {
  needsAttention: boolean;
  reasons: NodeAttentionReason[];
}

function buildSnoozeKey(nodeData: GraphNodeData): string {
  switch (nodeData.entityType) {
    case "backlog":
      if (nodeData.rawType === "BacklogItem") {
        return `backlog:${nodeData.kind}/${nodeData.name}`;
      }
      return `backlog:${nodeData.label}`;
    case "execution":
      return `execution:${nodeData.executionId}`;
    case "capture":
      return `capture:${nodeData.id}`;
    default:
      return `${nodeData.entityType}:${nodeData.label}`;
  }
}

export function computeNodeAttention(
  nodeData: GraphNodeData,
  enrichment?: NodeEnrichment,
  snoozedKeys?: Set<string>,
): NodeAttentionResult {
  const key = buildSnoozeKey(nodeData);
  if (snoozedKeys?.has(key)) {
    return { needsAttention: false, reasons: [] };
  }

  const reasons: NodeAttentionReason[] = [];

  if (nodeData.entityType === "backlog") {
    const status = nodeData.status;

    // Locked (queued/in_progress) items don't need user attention.
    if (status === "queued" || status === "in_progress") {
      return { needsAttention: false, reasons: [] };
    }

    if (status === "failed") {
      reasons.push("failed");
    }

    // Pending decisions take priority — user must answer before anything else.
    if (enrichment?.pendingDecisions && enrichment.pendingDecisions > 0) {
      reasons.push("pending-decisions");
    } else if (enrichment?.pendingSynthesis) {
      // Latest workshop answers need synthesis (finalize) — treated as a decision.
      reasons.push("pending-decisions");
    } else if (enrichment?.maturityReady === false) {
      // Not mature enough — needs workshop to make progress.
      reasons.push("pending-decisions");
    } else if (status === "ready" || enrichment?.maturityReady === true) {
      reasons.push("ready-to-run");
    } else if (status === "researching") {
      // Research phase items surface as needing decisions (review/advance).
      reasons.push("pending-decisions");
    }
  }

  if (nodeData.entityType === "execution") {
    const status = nodeData.status;
    if (status === "needs_review" || status === "needs_fixup") {
      reasons.push("needs-review");
    }
    if (status === "failed") {
      reasons.push("failed");
    }
  }

  if (nodeData.entityType === "capture") {
    const status = nodeData.status;
    if (status === "classifying") {
      reasons.push("needs-classification");
    }
  }

  return {
    needsAttention: reasons.length > 0,
    reasons,
  };
}

const REASON_LABELS: Record<NodeAttentionReason, string> = {
  "pending-decisions": "Pending decisions",
  "ready-to-run": "Ready to run",
  "needs-review": "Needs review",
  failed: "Failed",
  "needs-classification": "Needs classification",
};

export function formatAttentionSummary(reasons: NodeAttentionReason[]): string {
  if (reasons.length === 0) return "";
  return reasons.map((r) => REASON_LABELS[r]).join(", ");
}
