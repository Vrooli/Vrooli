/**
 * Backlog Queue & Action Utilities
 *
 * Centralizes all action-visibility and action-disabled logic for backlog items.
 * Both BacklogCard and BacklogDetailsPage consume `getItemActions()` so that
 * the CTA funnel is defined in exactly one place.
 *
 * DOC: docs/concepts/ARCHITECTURE.md#backlog-action-funnel
 */

import type { BacklogItem, BacklogKind, BacklogStatus } from "../types";

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

export const isBacklogQueueable = (item: QueueableBacklogItem): boolean =>
  QUEUEABLE_BACKLOG_STATUSES.includes(item.status) ||
  (item.kind === "idea" && item.status === "archived");

export const getBacklogNotQueueableReason = (item: QueueableBacklogItem): string | null => {
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
    case "archived":
      return "Only archived ideas can be queued directly.";
    default:
      return "This item cannot be queued from its current status.";
  }
};

// ---------------------------------------------------------------------------
// Dependency blocking
// ---------------------------------------------------------------------------

/**
 * Statuses that indicate a dependency is not yet planned/ready — blocking downstream
 * items from being *queued*. This is intentionally narrow: you CAN workshop an item
 * whose dependency is `ready`, so only `backlog` and `researching` block queueing.
 *
 * For *display ordering* (sort-blocking), see `dependency-sort.ts` which uses a
 * broader definition: any dependency not `completed`/`archived` pushes the dependent
 * below it in sorted views.
 */
const BLOCKING_DEP_STATUSES = new Set<BacklogStatus>(["backlog", "researching"]);

/**
 * Check whether any of an item's dependencies are still in an unplanned state,
 * meaning this item should not be run yet.
 */
export function hasBlockingDeps(item: Pick<BacklogItem, "dependsOn">, allItems: BacklogItem[]): boolean {
  if (!item.dependsOn || item.dependsOn.length === 0) return false;
  const itemsByKey = new Map(allItems.map((i) => [`${i.kind}/${i.name}`, i]));
  return item.dependsOn.some((dep) => {
    const depItem = itemsByKey.get(dep);
    return depItem && BLOCKING_DEP_STATUSES.has(depItem.status);
  });
}

/**
 * Return the keys of dependencies that are blocking this item.
 * Used for display purposes (linking to blocking items in the UI).
 */
export function getBlockingDepKeys(item: Pick<BacklogItem, "dependsOn">, allItems: BacklogItem[]): string[] {
  if (!item.dependsOn || item.dependsOn.length === 0) return [];
  const itemsByKey = new Map(allItems.map((i) => [`${i.kind}/${i.name}`, i]));
  return item.dependsOn.filter((dep) => {
    const depItem = itemsByKey.get(dep);
    return depItem && BLOCKING_DEP_STATUSES.has(depItem.status);
  });
}

// ---------------------------------------------------------------------------
// Dependency relations (parent/children resolution)
// ---------------------------------------------------------------------------

/** A dependency reference resolved to its display fields. */
export interface ResolvedDependency {
  kind: string;
  name: string;
  title: string;
  status: BacklogStatus;
}

/** Parent (upstream) and children (downstream) dependencies for an item. */
export interface DependencyRelations {
  parents: ResolvedDependency[];
  children: ResolvedDependency[];
}

/**
 * Compute the resolved parent and children dependencies for a backlog item.
 *
 * - Parents: items this item depends on (listed in `dependsOn`).
 * - Children: items that depend on this item (reverse lookup across `allItems`).
 *
 * Dangling refs (items listed in `dependsOn` but not found in `allItems`) are
 * returned with `"archived"` status so the chip still renders visually.
 */
export function computeDependencyRelations(
  item: Pick<BacklogItem, "kind" | "name" | "dependsOn">,
  allItems: BacklogItem[],
): DependencyRelations {
  const itemsByKey = new Map(allItems.map((i) => [`${i.kind}/${i.name}`, i]));
  const selfKey = `${item.kind}/${item.name}`;

  const parents: ResolvedDependency[] = [];
  for (const dep of item.dependsOn ?? []) {
    if (dep === selfKey) continue;
    if (!dep.includes("/")) continue;
    const found = itemsByKey.get(dep);
    if (found) {
      parents.push({ kind: found.kind, name: found.name, title: found.title || found.name, status: found.status });
    } else {
      const [kind = "", ...rest] = dep.split("/");
      parents.push({ kind, name: rest.join("/"), title: dep, status: "archived" as BacklogStatus });
    }
  }

  const children: ResolvedDependency[] = [];
  for (const other of allItems) {
    const otherKey = `${other.kind}/${other.name}`;
    if (otherKey === selfKey) continue;
    if (other.dependsOn?.includes(selfKey)) {
      children.push({ kind: other.kind, name: other.name, title: other.title || other.name, status: other.status });
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
  item: Pick<BacklogItem, "kind" | "name" | "status" | "dependsOn">;
  allItems: BacklogItem[];
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
  const { item, allItems, agentRunning } = ctx;

  const locked = LOCKED_STATUSES.has(item.status);
  const terminal = TERMINAL_STATUSES.has(item.status);
  const blocked = hasBlockingDeps(item, allItems);
  const blockingDepKeys = blocked ? getBlockingDepKeys(item, allItems) : [];
  const queueable = isBacklogQueueable(item);
  const notQueueableReason = getBacklogNotQueueableReason(item);

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
    canArchive: false,
    showDecisionStepper: false,
    agentRunning,
    notQueueableReason,
    disabledReason: null,
  };

  // Step -1: Locked — no CTAs at all.
  if (locked) return base;

  // Step 5: Terminal — follow-up + archive. Checked before steps 0-4 because
  // terminal items should never show run/workshop regardless of other state.
  if (terminal) {
    return {
      ...base,
      canFollowUp: ctx.hasExecutionHistory,
      canArchive: true,
      primaryCta: ctx.hasExecutionHistory ? "followUp" : "archive",
    };
  }

  // Step 0: Blocked by deps — show actions as disabled.
  if (blocked && queueable) {
    const needsFinalize = ctx.pendingSynthesis && ctx.readinessReady === true;
    const needsWorkshop = ctx.readinessReady === false;
    return {
      ...base,
      showDecisionStepper: ctx.hasPendingDecisions,
      canWorkshop: false,
      workshopDisabled: needsWorkshop,
      canFinalize: false,
      finalizeDisabled: needsFinalize,
      canRun: false,
      runDisabled: !needsWorkshop && !needsFinalize,
      primaryCta: needsFinalize ? "finalize" : needsWorkshop ? "workshop" : "run",
      disabledReason: "Blocked by incomplete dependencies. Resolve them first.",
    };
  }

  // Step 2: Unanswered decisions — stepper is primary, workshop blocked until
  // all decisions are resolved (running another round before answering existing
  // questions would just pile up more unanswered items).
  if (ctx.hasPendingDecisions) {
    return {
      ...base,
      showDecisionStepper: true,
      canWorkshop: false,
      workshopDisabled: false,
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
        primaryCta: "finalize",
        disabledReason: agentRunning ? "An agent is already running for this item." : null,
      };
    }
    return {
      ...base,
      canWorkshop: !agentRunning,
      workshopDisabled: agentRunning,
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
      primaryCta: "run",
      disabledReason: agentRunning ? "An agent is already running for this item." : null,
    };
  }

  // Fallback: no primary CTA.
  return base;
}
