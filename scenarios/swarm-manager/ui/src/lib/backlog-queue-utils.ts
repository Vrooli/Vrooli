/**
 * Backlog Queue & Action Utilities
 *
 * Adapts server-owned action projections and resolves dependency display data.
 *
 * DOC: docs/concepts/ARCHITECTURE.md#backlog-action-funnel
 */

import type {
  AgentActivityPurpose,
  AgentActivityStatus,
  BacklogItem,
  BacklogStatus,
} from "../types";
import type { BacklogNextAction } from "../services/backlog/types";
import type { AttentionReason, FeedbackItem, MaturityItem } from "./attention";
import { getAttentionReasons } from "./attention";

// ---------------------------------------------------------------------------
// Status constants
// ---------------------------------------------------------------------------

/** Statuses where the item is mid-execution and should not be edited or re-queued. */
export const LOCKED_STATUSES = new Set<BacklogStatus>(["queued", "in_progress"]);

/** Statuses that represent a finished execution (success or failure). */
export const TERMINAL_STATUSES = new Set<BacklogStatus>(["completed", "failed"]);

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

/** Which single CTA should receive primary visual emphasis. */
export type PrimaryCta = "run" | "followUp" | "archive" | "review" | "answer" | null;

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
  /** "Follow Up" button: visible (terminal + has execution history). */
  canFollowUp: boolean;
  /** "Retry" button: visible (terminal + has execution history). Same gate as Follow-Up; semantically distinct (re-runs same scope). */
  canRetry: boolean;
  /** "Archive" button: visible (terminal items). */
  canArchive: boolean;
  /** Inline decision stepper should render. */
  showDecisionStepper: boolean;
  /** Pass-through for label text ("Agent running..."). */
  agentRunning: boolean;
  /** True only while the agent is actively executing. */
  agentExecuting?: boolean;
  /** Human-readable reason why the item can't be queued, if applicable. */
  notQueueableReason: string | null;
  /** Human-readable reason why the primary CTA is disabled, shown as tooltip/helper text. */
  disabledReason: string | null;
}

/**
 * Adapts the server-owned next-action projection to the legacy card/button
 * shape. Eligibility is resolved by the server; this only preserves the
 * rendering contract while those cards are migrated.
 */
export function itemActionsFromNextAction(
  item: Pick<BacklogItem, "status">,
  action: BacklogNextAction | undefined,
  options: { agentRunning?: boolean; agentExecuting?: boolean } = {},
): ItemActions {
  const agentRunning = options.agentRunning ?? false;
  const agentExecuting = options.agentExecuting ?? agentRunning;
  const base: ItemActions = {
    locked: action?.id === "none" && LOCKED_STATUSES.has(item.status),
    terminal: TERMINAL_STATUSES.has(item.status),
    blocked: (action?.blockers.length ?? 0) > 0,
    blockingDepKeys: [],
    primaryCta: null,
    canRun: false,
    runDisabled: false,
    canFollowUp: false,
    canRetry: false,
    canArchive: false,
    showDecisionStepper: action?.id === "decide",
    agentRunning,
    agentExecuting,
    notQueueableReason: null,
    disabledReason: action?.enabled === false ? action.reason ?? null : null,
  };
  if (!action) return base;
  switch (action.id) {
    case "run":
      return { ...base, canRun: action.enabled, runDisabled: !action.enabled, primaryCta: "run" };
    case "retry":
      return { ...base, canRetry: action.enabled, primaryCta: "run" };
    case "dispatch_followup":
    case "author_followup":
      return { ...base, canFollowUp: action.enabled, primaryCta: "followUp" };
    case "archive":
      return { ...base, canArchive: action.enabled, primaryCta: "archive" };
    case "review":
      return { ...base, primaryCta: "review" };
    default:
      return base;
  }
}
