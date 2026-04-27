/**
 * Command Post Utilities
 *
 * Pure functions that aggregate actionable items across backlog, executions, and captures.
 * Composes existing utilities — does NOT duplicate logic.
 *
 * DOC: docs/concepts/ARCHITECTURE.md#command-post
 */

import type {
  BacklogItem,
  BacklogKind,
  Capture,
  ExecutionRecord,
  PendingQuestion,
  PendingQuestionsItem,
} from "../types";
import { getAttentionReasons, type AttentionReason, type FeedbackItem, type MaturityItem } from "./feed";
import { getItemActions, type ActionContext, type PrimaryCta } from "./backlog-queue-utils";

import { filterSnoozed, snoozeKeyForBacklog, snoozeKeyForCapture, snoozeKeyForExecution } from "./snooze-utils";
import { sortBacklogItems, buildCommandPostCompare } from "./backlog-sort";
import { computeUnblockingMap } from "./dependency-sort";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type ActionGroupId =
  | "needs-workshop"
  | "ready-to-run"
  | "pending-decisions"
  | "needs-review"
  | "needs-classification";

export interface ActionGroup {
  id: ActionGroupId;
  label: string;
  count: number;
  items: ActionableItem[];
}

export interface ActionableItem {
  type: "backlog" | "execution" | "capture";
  key: string; // snooze key
  title: string;
  kind?: BacklogKind;
  name?: string;
  executionId?: string;
  captureId?: string;
  reasons: AttentionReason[];
  primaryCta: PrimaryCta;
  /** Full BacklogItem reference — only populated for type: "backlog". */
  backlogItem?: BacklogItem;
}

export interface CrossItemQuestion {
  question: PendingQuestion;
  parentKind: BacklogKind;
  parentName: string;
  parentTitle: string;
}

// ---------------------------------------------------------------------------
// Group labels
// ---------------------------------------------------------------------------

const GROUP_LABELS: Record<ActionGroupId, string> = {
  "needs-workshop": "Needs Workshop",
  "ready-to-run": "Ready to Run",
  "pending-decisions": "Pending Decisions",
  "needs-review": "Needs Review",
  "needs-classification": "Needs Classification",
};

// ---------------------------------------------------------------------------
// groupActionItems
// ---------------------------------------------------------------------------

/**
 * Classify non-snoozed items into action groups.
 *
 * Uses getAttentionReasons() and getItemActions() from existing libs
 * to determine group membership and primary CTA.
 */
export function groupActionItems(
  backlogItems: BacklogItem[],
  executions: ExecutionRecord[],
  captures: Capture[],
  feedbackMap: Map<string, FeedbackItem>,
  maturityMap: Map<string, MaturityItem>,
  snoozedKeys: Set<string>,
): ActionGroup[] {
  const groups = new Map<ActionGroupId, ActionableItem[]>();
  for (const id of Object.keys(GROUP_LABELS) as ActionGroupId[]) {
    groups.set(id, []);
  }

  // --- Executions ---
  const nonSnoozedExecutions = filterSnoozed(
    executions,
    (e) => snoozeKeyForExecution(e.executionId),
    snoozedKeys,
  );
  for (const exec of nonSnoozedExecutions) {
    const key = snoozeKeyForExecution(exec.executionId);
    // All post-execution states that need human attention go to "needs-review":
    // failed, needs_review, needs_fixup, and completed (not yet archived).
    if (exec.status === "needs_review" || exec.status === "needs_fixup"
      || exec.status === "failed" || exec.status === "completed") {
      groups.get("needs-review")?.push({
        type: "execution",
        key,
        title: exec.backlogName || exec.executionId,
        executionId: exec.executionId,
        reasons: [],
        primaryCta: "review",
      });
    }
  }

  // --- Captures ---
  const nonSnoozedCaptures = filterSnoozed(
    captures,
    (c) => snoozeKeyForCapture(c.id),
    snoozedKeys,
  );
  for (const cap of nonSnoozedCaptures) {
    const key = snoozeKeyForCapture(cap.id);
    const title = cap.text.slice(0, 80) || cap.id;

    if (cap.status === "classifying") {
      groups.get("needs-classification")?.push({
        type: "capture",
        key,
        title,
        captureId: cap.id,
        reasons: [],
        primaryCta: null,
      });
    } else if (cap.status === "classified" && (cap.classification?.items.length ?? 0) > 0) {
      groups.get("needs-classification")?.push({
        type: "capture",
        key,
        title,
        captureId: cap.id,
        reasons: [],
        primaryCta: "review",
      });
    }
  }

  // --- Backlog items ---
  const nonSnoozedBacklog = filterSnoozed(
    backlogItems,
    (i) => snoozeKeyForBacklog(i.kind, i.name),
    snoozedKeys,
  );
  for (const item of nonSnoozedBacklog) {
    // Skip archived items — they shouldn't surface as actionable
    if (item.archivedAt != null) continue;

    const key = snoozeKeyForBacklog(item.kind, item.name);
    const reasons = getAttentionReasons(item, feedbackMap, maturityMap);

    // Build a minimal ActionContext to get the primary CTA
    const maturityKey = `${item.kind}/${item.name}`;
    const maturity = maturityMap.get(maturityKey);
    const feedback = feedbackMap.get(maturityKey);

    const ctx: ActionContext = {
      item,
      blockingInfo: null, // Command post doesn't need blocking state — items are grouped by CTA
      readinessReady: maturity?.ready ?? null,
      pendingSynthesis: false, // conservative default
      agentRunning: false,
      hasPendingDecisions: (feedback?.pendingDecisions ?? 0) > 0,
      hasExecutionHistory: false,
    };
    const actions = getItemActions(ctx);

    // Skip locked/terminal items with no attention reasons
    if (actions.locked) continue;

    const actionable: ActionableItem = {
      type: "backlog",
      key,
      title: item.title || item.name,
      kind: item.kind,
      name: item.name,
      reasons,
      primaryCta: actions.primaryCta,
      backlogItem: item,
    };

    // Classify into groups
    if (actions.terminal) {
      // All terminal items (completed, failed) need review before archiving
      groups.get("needs-review")?.push(actionable);
      continue;
    }

    if (ctx.hasPendingDecisions) {
      groups.get("pending-decisions")?.push({ ...actionable, primaryCta: "answer" });
    } else if (actions.primaryCta === "workshop" || actions.primaryCta === "finalize") {
      groups.get("needs-workshop")?.push(actionable);
    } else if (actions.primaryCta === "run") {
      groups.get("ready-to-run")?.push(actionable);
    }
  }

  // Build result array preserving group order. Most cards count parent items,
  // but Pending Decisions counts actual questions so it matches the stream.
  return (Object.keys(GROUP_LABELS) as ActionGroupId[]).map((id) => {
    const items = groups.get(id) ?? [];
    const count = id === "pending-decisions"
      ? items.reduce((sum, item) => {
        if (!item.kind || !item.name) return sum;
        return sum + (feedbackMap.get(`${item.kind}/${item.name}`)?.pendingDecisions ?? 0);
      }, 0)
      : items.length;
    return { id, label: GROUP_LABELS[id], count, items };
  });
}

// ---------------------------------------------------------------------------
// sortedGroupActionItems
// ---------------------------------------------------------------------------

/**
 * Classify items into action groups with dependency-aware ordering.
 *
 * Sorts backlog items before classification so items within each group
 * appear in dependency-aware priority order (deps before dependents,
 * then by priority ascending / recency descending).
 */
export function sortedGroupActionItems(
  backlogItems: BacklogItem[],
  executions: ExecutionRecord[],
  captures: Capture[],
  feedbackMap: Map<string, FeedbackItem>,
  maturityMap: Map<string, MaturityItem>,
  snoozedKeys: Set<string>,
): ActionGroup[] {
  const unblockingMap = computeUnblockingMap(backlogItems);
  const sorted = sortBacklogItems(backlogItems, buildCommandPostCompare(unblockingMap), backlogItems);
  return groupActionItems(sorted, executions, captures, feedbackMap, maturityMap, snoozedKeys);
}

// ---------------------------------------------------------------------------
// aggregateCrossItemQuestions
// ---------------------------------------------------------------------------

/**
 * Flatten per-item pending question lists into a single ordered queue.
 * Filters out questions belonging to snoozed items and items not in the
 * active backlog (e.g., archived/deleted items with leftover workshop files).
 */
export function aggregateCrossItemQuestions(
  pendingQuestionsItems: PendingQuestionsItem[],
  snoozedKeys: Set<string>,
  activeItemKeys?: Set<string>,
): CrossItemQuestion[] {
  const result: CrossItemQuestion[] = [];

  for (const pqi of pendingQuestionsItems) {
    const itemKey = snoozeKeyForBacklog(pqi.kind, pqi.name);
    if (snoozedKeys.has(itemKey)) continue;

    // Skip items that are not in the active backlog (archived/deleted)
    if (activeItemKeys && !activeItemKeys.has(`${pqi.kind}/${pqi.name}`)) continue;

    for (const question of pqi.questions) {
      if (question.source === "workshop" && question.selected?.trim()) continue;
      if (question.source === "review" && (question.review_status === "approved" || question.review_status === "flagged")) continue;

      result.push({
        question,
        parentKind: pqi.kind,
        parentName: pqi.name,
        parentTitle: question.title ?? pqi.name,
      });
    }
  }

  return result;
}

// ---------------------------------------------------------------------------
// computeBadgeCount
// ---------------------------------------------------------------------------

/** Sum of all group item counts — used for the HUD badge. */
export function computeBadgeCount(groups: ActionGroup[]): number {
  return groups.reduce((sum, g) => sum + g.count, 0);
}
