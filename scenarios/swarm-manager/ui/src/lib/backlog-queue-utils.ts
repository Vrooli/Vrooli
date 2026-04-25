/**
 * Backlog Queue & Action Utilities
 *
 * Centralizes all action-visibility and action-disabled logic for backlog items.
 * Both BacklogCard and BacklogDetailsPage consume `getItemActions()` so that
 * the CTA funnel is defined in exactly one place.
 *
 * DOC: docs/concepts/ARCHITECTURE.md#backlog-action-funnel
 */

import type {
  AgentActivityPurpose,
  AgentActivityStatus,
  BacklogItem,
  BacklogKind,
  BacklogStatus,
  ItemBlockingInfo,
} from "../types";
import type { AttentionReason, FeedbackItem, MaturityItem } from "./feed";
import { getAttentionReasons } from "./feed";

// ---------------------------------------------------------------------------
// Status constants
// ---------------------------------------------------------------------------

export const QUEUEABLE_BACKLOG_STATUSES: BacklogStatus[] = ["backlog", "researching", "ready"];

/** Statuses where the item is mid-execution and should not be edited or re-queued. */
export const LOCKED_STATUSES = new Set<BacklogStatus>(["queued", "in_progress"]);

/** Statuses that represent a finished execution (success or failure). */
export const TERMINAL_STATUSES = new Set<BacklogStatus>(["completed", "failed"]);

// ---------------------------------------------------------------------------
// Queueability helpers
// ---------------------------------------------------------------------------

interface QueueableBacklogItem {
  kind: BacklogKind;
  status: BacklogStatus;
}

export const isBacklogQueueable = (item: QueueableBacklogItem & { archivedAt?: string }): boolean =>
  QUEUEABLE_BACKLOG_STATUSES.includes(item.status) ||
  (item.kind === "idea" && item.archivedAt != null);

export const getBacklogNotQueueableReason = (item: QueueableBacklogItem & { archivedAt?: string }): string | null => {
  if (isBacklogQueueable(item)) {
    return null;
  }
  switch (item.status) {
    case "queued":
      return "Already queued. Check Execution for run progress.";
    case "in_progress":
      return "Already in progress. Wait for it to finish before re-queueing.";
    case "completed":
      return "Completed items cannot be queued again.";
    case "failed":
      return "Reset status to retry. Check Execution History for failure details.";
    default:
      return "This item cannot be queued from its current status.";
  }
};

// ---------------------------------------------------------------------------
// Dependency relations (parent/children resolution)
// ---------------------------------------------------------------------------

/** Agent activity summary attached to a resolved dependency. */
export interface ResolvedDependencyActivity {
  purpose: AgentActivityPurpose;
  status: AgentActivityStatus;
}

/** A dependency reference resolved to its display fields. */
export interface ResolvedDependency {
  kind: string;
  name: string;
  title: string;
  status: BacklogStatus;
  /** Active agent activity on this dependency, if any. Present only when an
   *  activityByKey map is supplied to `computeDependencyRelations`. */
  activity?: ResolvedDependencyActivity;
  /** Pending-input reasons (pending decisions, plan ready, review ready).
   *  Present only when feedback/maturity maps are supplied to
   *  `computeDependencyRelations`. */
  attentionReasons?: AttentionReason[];
}

/** Parent (upstream) and children (downstream) dependencies for an item. */
export interface DependencyRelations {
  parents: ResolvedDependency[];
  children: ResolvedDependency[];
}

/** Optional extra signals to enrich each resolved dependency with. */
export interface DependencyEnrichmentMaps {
  /** Map from `${kind}/${name}` → active AgentActivity summary. */
  activityByKey?: Map<string, ResolvedDependencyActivity>;
  feedbackMap?: Map<string, FeedbackItem>;
  maturityMap?: Map<string, MaturityItem>;
}

/**
 * Compute the resolved parent and children dependencies for a backlog item.
 *
 * - Parents: items this item depends on (listed in `dependsOn`).
 * - Children: items that depend on this item (reverse lookup across `allItems`).
 *
 * Dangling refs (items listed in `dependsOn` but not found in `allItems`) are
 * returned with `"completed"` status so the chip still renders visually.
 *
 * When `enrichment` maps are provided, each resolved dependency also carries
 * an `activity` (if an agent is active on it) and `attentionReasons` (pending
 * decisions / plan ready / review ready). Without maps, those fields are left
 * undefined and consumers fall back to lifecycle status only.
 */
export function computeDependencyRelations(
  item: Pick<BacklogItem, "kind" | "name" | "dependsOn">,
  allItems: BacklogItem[],
  enrichment?: DependencyEnrichmentMaps,
): DependencyRelations {
  const itemsByKey = new Map(allItems.map((i) => [`${i.kind}/${i.name}`, i]));
  const selfKey = `${item.kind}/${item.name}`;

  const attachEnrichment = (
    dep: ResolvedDependency,
    fullItem: BacklogItem | null,
  ): ResolvedDependency => {
    if (!enrichment) return dep;
    const key = `${dep.kind}/${dep.name}`;
    const activity = enrichment.activityByKey?.get(key);
    if (activity) dep.activity = activity;
    if (fullItem && (enrichment.feedbackMap || enrichment.maturityMap)) {
      const reasons = getAttentionReasons(
        fullItem,
        enrichment.feedbackMap ?? new Map<string, FeedbackItem>(),
        enrichment.maturityMap ?? new Map<string, MaturityItem>(),
      );
      dep.attentionReasons = reasons;
    }
    return dep;
  };

  const parents: ResolvedDependency[] = [];
  for (const dep of item.dependsOn ?? []) {
    if (dep === selfKey) continue;
    if (!dep.includes("/")) continue;
    const found = itemsByKey.get(dep);
    if (found) {
      parents.push(
        attachEnrichment(
          { kind: found.kind, name: found.name, title: found.title || found.name, status: found.status },
          found,
        ),
      );
    } else {
      const [kind = "", ...rest] = dep.split("/");
      parents.push(
        attachEnrichment(
          { kind, name: rest.join("/"), title: dep, status: "completed" as BacklogStatus },
          null,
        ),
      );
    }
  }

  const children: ResolvedDependency[] = [];
  for (const other of allItems) {
    const otherKey = `${other.kind}/${other.name}`;
    if (otherKey === selfKey) continue;
    if (other.dependsOn?.includes(selfKey)) {
      children.push(
        attachEnrichment(
          { kind: other.kind, name: other.name, title: other.title || other.name, status: other.status },
          other,
        ),
      );
    }
  }

  return { parents, children };
}

// ---------------------------------------------------------------------------
// Action resolver
// ---------------------------------------------------------------------------

/**
 * Input context for computing item actions. All fields are pre-computed
 * booleans/values so the resolver stays pure and framework-agnostic.
 */
export interface ActionContext {
  item: Pick<BacklogItem, "kind" | "name" | "status" | "dependsOn"> & { archivedAt?: string };
  /** Server-computed blocking info for this item, or null if not available. */
  blockingInfo: ItemBlockingInfo | null;
  /** Whether the item's plan is ready for execution. null = no readiness data loaded. */
  readinessReady: boolean | null;
  /** Whether the latest answered workshop round still needs a synthesis/finalize pass. */
  pendingSynthesis: boolean;
  /** Whether an agent run is currently active for this item. */
  agentRunning: boolean;
  /** Whether the item has unanswered workshop decisions. */
  hasPendingDecisions: boolean;
  /** Whether execution history exists for this item (details page only; card passes false). */
  hasExecutionHistory: boolean;
  /**
   * Whether the latest execution is in a terminal/effectively-terminal state
   * (`completed | failed | canceled | needs_fixup`). When true, retry is
   * available regardless of the item's current status — useful when a user
   * has manually flipped a failed item back to backlog/ready and now wants
   * to re-dispatch the prior attempt.
   */
  hasTerminalExecution?: boolean;
}

/** Which single CTA should receive primary visual emphasis. */
export type PrimaryCta = "run" | "workshop" | "finalize" | "followUp" | "archive" | "review" | "answer" | null;

/** Computed action states for a backlog item. */
export interface ItemActions {
  /** Item is mid-execution (queued/in_progress) — no CTAs allowed. */
  locked: boolean;
  /** Item execution finished (completed/failed). */
  terminal: boolean;
  /** Item has incomplete dependencies blocking it. */
  blocked: boolean;
  /** Keys of the specific dependencies that are blocking (for display). */
  blockingDepKeys: string[];
  /** Which CTA should be visually emphasized as the primary action. */
  primaryCta: PrimaryCta;
  /** "Run" button: visible and enabled. */
  canRun: boolean;
  /** "Run" button: visible but disabled (agent running or blocked). */
  runDisabled: boolean;
  /** "Workshop" button: visible and enabled. */
  canWorkshop: boolean;
  /** "Workshop" button: visible but disabled (agent running or blocked). */
  workshopDisabled: boolean;
  /** "Finalize" button: visible and enabled. */
  canFinalize: boolean;
  /** "Finalize" button: visible but disabled (agent running or blocked). */
  finalizeDisabled: boolean;
  /** "Follow Up" button: visible (terminal + has execution history). */
  canFollowUp: boolean;
  /** "Retry" button: visible (terminal + has execution history). Same gate as Follow-Up; semantically distinct (re-runs same scope). */
  canRetry: boolean;
  /** "Archive" button: visible (terminal items). */
  canArchive: boolean;
  /** Inline decision stepper / expanded workshop panel should render. */
  showDecisionStepper: boolean;
  /** Pass-through for label text ("Agent running..."). */
  agentRunning: boolean;
  /** Human-readable reason why the item can't be queued, if applicable. */
  notQueueableReason: string | null;
  /** Human-readable reason why the primary CTA is disabled, shown as tooltip/helper text. */
  disabledReason: string | null;
}

/**
 * Compute all action states for a backlog item following the CTA funnel:
 *
 * | Step | Condition                          | Primary CTA     |
 * |------|------------------------------------|-----------------|
 * | -1   | Locked (queued/in_progress)         | none            |
 * |  0   | Blocked by deps                    | disabled        |
 * |  1   | Agent running                      | disabled        |
 * |  2   | Unanswered decisions               | stepper/panel   |
 * |  3   | Latest answers pending synthesis    | finalize/workshop |
 * |  4   | Readiness not met (no decisions)    | workshop        |
 * |  5   | Ready, no active run               | run             |
 * |  6   | Terminal (completed/failed)         | follow-up/archive |
 */
export function getItemActions(ctx: ActionContext): ItemActions {
  const { item, blockingInfo, agentRunning } = ctx;

  const locked = LOCKED_STATUSES.has(item.status);
  const blocked = blockingInfo?.blocked ?? false;
  const blockingDepKeys = blockingInfo?.blockingDepKeys ?? [];
  const queueable = isBacklogQueueable(item);
  const notQueueableReason = getBacklogNotQueueableReason(item);
  // Archived ideas (status=completed + archivedAt) are re-queueable, so they
  // should not be treated as terminal even though their status is "completed".
  const terminal = TERMINAL_STATUSES.has(item.status) && !queueable;

  // Retry is gated on having a *terminal* execution to retry — independent
  // of the item's current status. A user who manually flipped a failed item
  // back to backlog/ready should still see Retry, because the prior attempt
  // is what gets re-dispatched (parented to the new attempt).
  const canRetryFromHistory = ctx.hasTerminalExecution ?? ctx.hasExecutionHistory;

  // Base result with all actions off.
  const base: ItemActions = {
    locked,
    terminal,
    blocked,
    blockingDepKeys,
    primaryCta: null,
    canRun: false,
    runDisabled: false,
    canWorkshop: false,
    workshopDisabled: false,
    canFinalize: false,
    finalizeDisabled: false,
    canFollowUp: false,
    canRetry: false,
    canArchive: false,
    showDecisionStepper: false,
    agentRunning,
    notQueueableReason,
    disabledReason: null,
  };

  // Step -1: Locked or archived — no CTAs at all.
  if (locked) return base;
  if (item.archivedAt != null) return base;

  // Step 5: Terminal — follow-up + archive. Checked before steps 0-4 because
  // terminal items should never show run/workshop regardless of other state.
  if (terminal) {
    return {
      ...base,
      canFollowUp: ctx.hasExecutionHistory,
      canRetry: canRetryFromHistory,
      canArchive: true,
      primaryCta: ctx.hasExecutionHistory ? "followUp" : "archive",
    };
  }

  // Step 0: Blocked by deps — CTAs remain available so user can override via
  // the modal (which sends confirm+force). The `blocked` and `blockingDepKeys`
  // fields on ItemActions signal the UI to show a warning, but buttons are not
  // hard-disabled. Fall through to normal CTA logic below.

  // Step 2: Unanswered decisions — stepper is primary, workshop blocked until
  // all decisions are resolved (running another round before answering existing
  // questions would just pile up more unanswered items).
  if (ctx.hasPendingDecisions) {
    return {
      ...base,
      showDecisionStepper: true,
      canWorkshop: false,
      workshopDisabled: false,
      canRetry: canRetryFromHistory,
      primaryCta: null,
    };
  }

  // Step 3a: Latest answers need synthesis — finalize if ready, otherwise run
  // another workshop round to incorporate them.
  if (queueable && ctx.pendingSynthesis) {
    if (ctx.readinessReady === true) {
      return {
        ...base,
        canFinalize: !agentRunning,
        finalizeDisabled: agentRunning,
        canWorkshop: !agentRunning,
        workshopDisabled: agentRunning,
        canRetry: canRetryFromHistory,
        primaryCta: "finalize",
        disabledReason: agentRunning ? "An agent is already running for this item." : null,
      };
    }
    return {
      ...base,
      canWorkshop: !agentRunning,
      workshopDisabled: agentRunning,
      canRetry: canRetryFromHistory,
      primaryCta: "workshop",
      disabledReason: agentRunning ? "An agent is already running for this item." : null,
    };
  }

  // Step 3: Readiness not met — workshop is primary.
  if (queueable && ctx.readinessReady === false) {
    return {
      ...base,
      canWorkshop: !agentRunning,
      workshopDisabled: agentRunning,
      canRetry: canRetryFromHistory,
      primaryCta: "workshop",
      disabledReason: agentRunning ? "An agent is already running for this item." : null,
    };
  }

  // Step 4: Ready — run is primary.
  if (queueable) {
    return {
      ...base,
      canRun: !agentRunning,
      runDisabled: agentRunning,
      canWorkshop: !agentRunning,
      workshopDisabled: agentRunning,
      canRetry: canRetryFromHistory,
      primaryCta: "run",
      disabledReason: agentRunning ? "An agent is already running for this item." : null,
    };
  }

  // Non-queueable, non-terminal status (e.g., manually moved to backlog with
  // no plan yet). Surface Retry if a terminal execution exists so the user
  // can re-dispatch the prior attempt.
  if (canRetryFromHistory) {
    return { ...base, canRetry: true };
  }

  // Fallback: no primary CTA.
  return base;
}
