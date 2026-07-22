/**
 * Operations Center wire types.
 *
 * Mirrors the backend `OperationsView` shape returned from
 * `GET /api/v1/operations` (see scenarios/swarm-manager/api/internal/operations).
 *
 * The backend keys are snake_case; the UI types are camelCase. The
 * `operations-service` module performs explicit field-by-field
 * normalization following the same pattern as `milestone-mode-service`.
 *
 * Lane vocabulary mirrors `agentactivity.Lane` on the backend — the four
 * canonical lanes are always present in `OperationsView.lanes`, even when
 * a lane has zero active activities, because the header renders four
 * utilization bars unconditionally.
 */

/**
 * The four canonical lanes that organize agentic activity for both
 * concurrency caps (the `LaneStatus.capacity` field) and column placement
 * in the Ops Center by-phase view.
 */
export type OperationsLane =
  | "investigate"
  | "execute"
  | "review"
  | "reconcile";

/**
 * Exhaustive ordered list of lane identifiers. Order matches the backend's
 * canonical iteration order so the Ops Center header renders bars in the
 * same sequence as the API returns them.
 */
export const OPERATIONS_LANES = [
  "investigate",
  "execute",
  "review",
  "reconcile",
] as const satisfies readonly OperationsLane[];

/**
 * Status surface mirrors `agentactivity.Status` plus the `pending` value
 * that the queue-side records carry. Kept as a string union so the UI can
 * render unknown values defensively without type errors.
 */
export type OperationsActivityStatus =
  | "pending"
  | "starting"
  | "running"
  | "needs_review"
  | "complete"
  | "failed"
  | "cancelled"
  | string;

/**
 * Subset of `AgentActivityOwnerType` the operations endpoint surfaces.
 * Kept loose (`string`) for forward compatibility — the by-milestone view
 * groups by `OwnerType === "milestone"` and falls back to the standalone
 * bucket otherwise.
 */
export type OperationsOwnerType = "backlog" | "milestone" | "scenario" | "capture" | "session" | string;

export interface LaneStatus {
  lane: OperationsLane | string;
  active: number;
  capacity: number;
  queue: number;
}

export interface QueueStatus {
  depth: number;
  maxDepth: number;
}

/**
 * Single per-run summary the Operations Center renders. `runId` is the
 * stable identity across status transitions; `activityId` is the
 * underlying ledger row's identity (used as a row key when `runId` is
 * absent — true for queue rows that haven't been spawned yet).
 */
export interface ActivityRow {
  activityId: string;
  runId?: string;
  ownerType: OperationsOwnerType;
  ownerKind?: string;
  ownerName: string;
  ownerTitle?: string;
  purpose: string;
  phaseKind?: string;
  lane?: string;
  status: OperationsActivityStatus;
  mode?: string;
  phase?: string;
  round?: number;
  milestoneName?: string;
  requestedAt: string;
  startedAt?: string;
  finishedAt?: string;
  runtimeSeconds?: number;
  failureReason?: string;
  requestedBy?: string;
  interactionType?: string;
}

export interface OperationsView {
  lanes: LaneStatus[];
  queue: QueueStatus;
  activities: ActivityRow[];
  recentlyFinished: ActivityRow[];
  generatedAt: string;
  /** Echoes the effective query window (in seconds) the aggregator used. */
  windowSeconds: number;
}

export interface OperationsBriefingSummary {
  activeActivityCount: number;
  recentlyFinishedCount: number;
  queueDepth: number;
  maxQueueDepth: number;
  saturatedLanes: string[];
  activeLaneCountByLane: Record<string, number>;
  totalBacklogItems: number;
  activeGoals: number;
  blockedItems: number;
  activeSessions: number;
}

export interface OperationsBriefingAttentionItem {
  id: string;
  severity: string;
  reason: string;
  title: string;
  status?: string;
  lane?: string;
  ref?: string;
  command?: string;
}

export interface OperationsDirectorHandoff {
  sourcePath: string;
  title: string;
  observedAt?: string;
  excerpt: string;
}

export interface OperationsRecommendedAction {
  id: string;
  label: string;
  reason: string;
  command?: string;
  uiPath?: string;
}

export interface OperationsDrillDownCommand {
  label: string;
  command: string;
}

export interface OperationsBriefing {
  generatedAt: string;
  freshnessSeconds: number;
  windowSeconds: number;
  summary: OperationsBriefingSummary;
  activeWork: ActivityRow[];
  needsAttention: OperationsBriefingAttentionItem[];
  recentCompletions: ActivityRow[];
  directorHandoffs: OperationsDirectorHandoff[];
  recommendedNextActions: OperationsRecommendedAction[];
  drillDownCommands: OperationsDrillDownCommand[];
  warnings: string[];
}

/**
 * Optional filter payload accepted by `fetchOperations`. Empty arrays and
 * empty strings are dropped before the query string is built so they do
 * not collide with backend "no filter" defaults.
 */
export interface OperationsFilters {
  /** Window in seconds. The service formats this as an ISO-8601 PT-duration. */
  windowSeconds?: number;
  statuses?: string[];
  lanes?: string[];
  modes?: string[];
  ownerTypes?: string[];
  q?: string;
}

/**
 * View the Ops Center body renders. P6 ships only `by-milestone`; P7a
 * adds `by-phase`. Stored both in the operations-store and as a `view=`
 * URL query parameter for deep-linkability.
 */
export type OperationsViewMode = "by-milestone" | "by-phase";

export const OPERATIONS_VIEW_MODES: readonly OperationsViewMode[] = [
  "by-milestone",
  "by-phase",
] as const;

/**
 * Filter accepted by the `bulk-stop` "stop all" path. Mirrors the backend
 * `BulkStopFilter` wire shape. Either field may be omitted; an empty
 * filter targets every active run.
 */
export interface BulkStopFilter {
  lane?: string;
  status?: string;
}

/**
 * Wire shape POSTed to `/api/v1/operations/bulk-stop`. Exactly one of
 * `runIds` and `filter` is set; the backend rejects both and neither.
 */
export type BulkStopRequest =
  | { runIds: string[]; filter?: never }
  | { runIds?: never; filter: BulkStopFilter };

/** Per-run outcome from a bulk-stop call. */
export interface BulkStopOutcome {
  runId: string;
  success: boolean;
  error?: string;
}

/** Top-level response from `/api/v1/operations/bulk-stop`. */
export interface BulkStopResponse {
  outcomes: BulkStopOutcome[];
  total: number;
  stopped: number;
  failed: number;
}
