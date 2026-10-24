/**
 * Operations service.
 *
 * Wraps `GET /api/v1/operations` and normalizes the snake_case wire shape
 * into the camelCase `OperationsView` UI type. Mirrors the field-by-field
 * normalization pattern established by `milestone-mode-service.ts` —
 * we deliberately avoid generic snake→camel converters so renames on the
 * backend surface here as type errors instead of silently mismatching.
 *
 * The endpoint accepts repeated query keys (`lane=execute&lane=review`)
 * for the lane / status / mode / owner_type filters; `URLSearchParams`
 * with `.append` produces exactly that wire shape. The handler also
 * accepts a `key[]` form and a comma-joined form, but the repeated-key
 * form is the most idiomatic for Go's `r.URL.Query()` and matches what
 * the handler tests assert.
 */

import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type {
  ActivityRow,
  BulkStopOutcome,
  BulkStopRequest,
  BulkStopResponse,
  LaneStatus,
  OperationsFilters,
  OperationsBriefing,
  OperationsBriefingAttentionItem,
  OperationsBriefingSummary,
  OperationsDirectorHandoff,
  OperationsDrillDownCommand,
  OperationsRecommendedAction,
  OperationsView,
  QueueStatus,
} from "../types/operations";

type RawRecord = Record<string, unknown>;

function stringValue(value: unknown, fallback = ""): string {
  return typeof value === "string" ? value : fallback;
}

function numberValue(value: unknown, fallback = 0): number {
  return typeof value === "number" && Number.isFinite(value) ? value : fallback;
}

function recordValue(value: unknown): RawRecord {
  return value && typeof value === "object" && !Array.isArray(value)
    ? (value as RawRecord)
    : {};
}

function normalizeLane(raw: unknown): LaneStatus {
  const lane = recordValue(raw);
  return {
    lane: stringValue(lane.lane),
    active: numberValue(lane.active),
    capacity: numberValue(lane.capacity),
    queue: numberValue(lane.queue),
  };
}

function normalizeQueue(raw: unknown): QueueStatus {
  const queue = recordValue(raw);
  return {
    depth: numberValue(queue.depth),
    maxDepth: numberValue(queue.max_depth ?? queue.maxDepth),
  };
}

function normalizeActivity(raw: unknown): ActivityRow {
  const row = recordValue(raw);
  return {
    activityId: stringValue(row.activity_id ?? row.activityId),
    runId: stringValue(row.run_id ?? row.runId, "") || undefined,
    ownerType: stringValue(row.owner_type ?? row.ownerType),
    ownerKind: stringValue(row.owner_kind ?? row.ownerKind, "") || undefined,
    ownerName: stringValue(row.owner_name ?? row.ownerName),
    ownerTitle: stringValue(row.owner_title ?? row.ownerTitle, "") || undefined,
    purpose: stringValue(row.purpose),
    phaseKind: stringValue(row.phase_kind ?? row.phaseKind, "") || undefined,
    lane: stringValue(row.lane, "") || undefined,
    status: stringValue(row.status),
    mode: stringValue(row.mode, "") || undefined,
    phase: stringValue(row.phase, "") || undefined,
    round: typeof row.round === "number" && row.round > 0 ? row.round : undefined,
    milestoneName: stringValue(row.milestone_name ?? row.milestoneName, "") || undefined,
    requestedAt: stringValue(row.requested_at ?? row.requestedAt),
    startedAt: stringValue(row.started_at ?? row.startedAt, "") || undefined,
    finishedAt: stringValue(row.finished_at ?? row.finishedAt, "") || undefined,
    runtimeSeconds:
      typeof row.runtime_seconds === "number"
        ? row.runtime_seconds
        : typeof row.runtimeSeconds === "number"
          ? row.runtimeSeconds
          : undefined,
    failureReason: stringValue(row.failure_reason ?? row.failureReason, "") || undefined,
    requestedBy: stringValue(row.requested_by ?? row.requestedBy, "") || undefined,
    interactionType:
      stringValue(row.interaction_type ?? row.interactionType, "") || undefined,
  };
}

function normalizeView(raw: unknown): OperationsView {
  const view = recordValue(raw);
  const lanes = view.lanes;
  const activities = view.activities;
  const finished = view.recently_finished ?? view.recentlyFinished;
  return {
    lanes: Array.isArray(lanes) ? lanes.map(normalizeLane) : [],
    queue: normalizeQueue(view.queue),
    activities: Array.isArray(activities) ? activities.map(normalizeActivity) : [],
    recentlyFinished: Array.isArray(finished) ? finished.map(normalizeActivity) : [],
    generatedAt: stringValue(view.generated_at ?? view.generatedAt),
    windowSeconds: numberValue(view.window_seconds ?? view.windowSeconds),
  };
}

function normalizeBriefingSummary(raw: unknown): OperationsBriefingSummary {
  const summary = recordValue(raw);
  const laneCounts = recordValue(summary.active_lane_count_by_lane ?? summary.activeLaneCountByLane);
  const saturatedLanes = summary.saturated_lanes ?? summary.saturatedLanes;
  return {
    activeActivityCount: numberValue(summary.active_activity_count ?? summary.activeActivityCount),
    recentlyFinishedCount: numberValue(summary.recently_finished_count ?? summary.recentlyFinishedCount),
    queueDepth: numberValue(summary.queue_depth ?? summary.queueDepth),
    maxQueueDepth: numberValue(summary.max_queue_depth ?? summary.maxQueueDepth),
    saturatedLanes: Array.isArray(saturatedLanes)
      ? saturatedLanes.filter((value): value is string => typeof value === "string")
      : [],
    activeLaneCountByLane: Object.fromEntries(
      Object.entries(laneCounts).filter((entry): entry is [string, number] => typeof entry[1] === "number"),
    ),
    totalBacklogItems: numberValue(summary.total_backlog_items ?? summary.totalBacklogItems),
    activeGoals: numberValue(summary.active_goals ?? summary.activeGoals),
    blockedItems: numberValue(summary.blocked_items ?? summary.blockedItems),
    activeSessions: numberValue(summary.active_sessions ?? summary.activeSessions),
  };
}

function normalizeAttentionItem(raw: unknown): OperationsBriefingAttentionItem {
  const item = recordValue(raw);
  return {
    id: stringValue(item.id),
    severity: stringValue(item.severity),
    reason: stringValue(item.reason),
    title: stringValue(item.title),
    status: stringValue(item.status, "") || undefined,
    lane: stringValue(item.lane, "") || undefined,
    ref: stringValue(item.ref, "") || undefined,
    command: stringValue(item.command, "") || undefined,
  };
}

function normalizeHandoff(raw: unknown): OperationsDirectorHandoff {
  const item = recordValue(raw);
  return {
    sourcePath: stringValue(item.source_path ?? item.sourcePath),
    title: stringValue(item.title),
    observedAt: stringValue(item.observed_at ?? item.observedAt, "") || undefined,
    excerpt: stringValue(item.excerpt),
  };
}

function normalizeRecommendedAction(raw: unknown): OperationsRecommendedAction {
  const item = recordValue(raw);
  return {
    id: stringValue(item.id),
    label: stringValue(item.label),
    reason: stringValue(item.reason),
    command: stringValue(item.command, "") || undefined,
    uiPath: stringValue(item.ui_path ?? item.uiPath, "") || undefined,
  };
}

function normalizeDrillDownCommand(raw: unknown): OperationsDrillDownCommand {
  const item = recordValue(raw);
  return {
    label: stringValue(item.label),
    command: stringValue(item.command),
  };
}

function normalizeBriefing(raw: unknown): OperationsBriefing {
  const briefing = recordValue(raw);
  const activeWork = briefing.active_work ?? briefing.activeWork;
  const recent = briefing.recent_completions ?? briefing.recentCompletions;
  const attention = briefing.needs_attention ?? briefing.needsAttention;
  const handoffs = briefing.director_handoffs ?? briefing.directorHandoffs;
  const actions = briefing.recommended_next_actions ?? briefing.recommendedNextActions;
  const commands = briefing.drill_down_commands ?? briefing.drillDownCommands;
  const warnings = briefing.warnings;
  return {
    generatedAt: stringValue(briefing.generated_at ?? briefing.generatedAt),
    freshnessSeconds: numberValue(briefing.freshness_seconds ?? briefing.freshnessSeconds),
    windowSeconds: numberValue(briefing.window_seconds ?? briefing.windowSeconds),
    summary: normalizeBriefingSummary(briefing.summary),
    activeWork: Array.isArray(activeWork) ? activeWork.map(normalizeActivity) : [],
    needsAttention: Array.isArray(attention) ? attention.map(normalizeAttentionItem) : [],
    recentCompletions: Array.isArray(recent) ? recent.map(normalizeActivity) : [],
    directorHandoffs: Array.isArray(handoffs) ? handoffs.map(normalizeHandoff) : [],
    recommendedNextActions: Array.isArray(actions) ? actions.map(normalizeRecommendedAction) : [],
    drillDownCommands: Array.isArray(commands) ? commands.map(normalizeDrillDownCommand) : [],
    warnings: Array.isArray(warnings) ? warnings.filter((value): value is string => typeof value === "string") : [],
  };
}

/**
 * Format a positive integer second count as the smallest ISO-8601
 * PT-duration the backend accepts (`PTxH`, `PTxM`, or `PTxS`). Larger
 * windows decompose into hours+minutes+seconds in that order so the
 * server's strict grammar accepts every value the slider can pick.
 */
export function formatWindowSeconds(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) {
    return "PT3H";
  }
  const total = Math.floor(seconds);
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const secs = total % 60;
  let out = "PT";
  if (hours > 0) out += `${hours}H`;
  if (minutes > 0) out += `${minutes}M`;
  if (secs > 0 || (hours === 0 && minutes === 0)) out += `${secs}S`;
  return out;
}

function buildOperationsQuery(filters?: OperationsFilters): string {
  const params = new URLSearchParams();
  if (filters?.windowSeconds && filters.windowSeconds > 0) {
    params.set("window", formatWindowSeconds(filters.windowSeconds));
  }
  for (const status of filters?.statuses ?? []) {
    const trimmed = status.trim();
    if (trimmed) params.append("status", trimmed);
  }
  for (const lane of filters?.lanes ?? []) {
    const trimmed = lane.trim();
    if (trimmed) params.append("lane", trimmed);
  }
  for (const mode of filters?.modes ?? []) {
    const trimmed = mode.trim();
    if (trimmed) params.append("mode", trimmed);
  }
  for (const ownerType of filters?.ownerTypes ?? []) {
    const trimmed = ownerType.trim();
    if (trimmed) params.append("owner_type", trimmed);
  }
  if (filters?.q) {
    const trimmed = filters.q.trim();
    if (trimmed) params.set("q", trimmed);
  }
  const encoded = params.toString();
  return encoded ? `?${encoded}` : "";
}

function normalizeBulkOutcome(raw: unknown): BulkStopOutcome {
  const row = recordValue(raw);
  return {
    runId: stringValue(row.run_id ?? row.runId),
    success: row.success === true,
    error: stringValue(row.error, "") || undefined,
  };
}

function normalizeBulkResponse(raw: unknown): BulkStopResponse {
  const view = recordValue(raw);
  const outcomes = view.outcomes;
  return {
    outcomes: Array.isArray(outcomes) ? outcomes.map(normalizeBulkOutcome) : [],
    total: numberValue(view.total),
    stopped: numberValue(view.stopped),
    failed: numberValue(view.failed),
  };
}

/**
 * Convert the camelCase request shape into the snake_case wire shape the
 * backend expects. Mirrors the field-by-field normalization the rest of
 * this module uses for incoming responses — intentionally avoiding a
 * generic camel→snake converter so renames surface as type errors rather
 * than silent wire mismatches.
 */
function serializeBulkRequest(req: BulkStopRequest): Record<string, unknown> {
  if ("runIds" in req && req.runIds) {
    return { run_ids: req.runIds };
  }
  if ("filter" in req && req.filter) {
    const filter: Record<string, string> = {};
    if (req.filter.lane) filter.lane = req.filter.lane;
    if (req.filter.status) filter.status = req.filter.status;
    return { filter };
  }
  // Should be unreachable — the BulkStopRequest discriminated union forbids
  // it at the type level — but be defensive so a future caller error fails
  // fast on the wire (the backend rejects empty bodies with 400).
  return {};
}

export interface IOperationsService {
  fetchOperations(filters?: OperationsFilters): Promise<OperationsView>;
  fetchBriefing?(filters?: Pick<OperationsFilters, "windowSeconds">): Promise<OperationsBriefing>;
  bulkStop(request: BulkStopRequest): Promise<BulkStopResponse>;
}

export function createOperationsService(
  apiClient: IApiClient = defaultApiClient,
): IOperationsService {
  return {
    async fetchOperations(filters?: OperationsFilters): Promise<OperationsView> {
      const suffix = buildOperationsQuery(filters);
      const raw = await apiClient.get<unknown>(`${API_ENDPOINTS.operations}${suffix}`);
      return normalizeView(raw);
    },
    async fetchBriefing(filters?: Pick<OperationsFilters, "windowSeconds">): Promise<OperationsBriefing> {
      const suffix = buildOperationsQuery(filters);
      const raw = await apiClient.get<unknown>(`${API_ENDPOINTS.operationsBrief}${suffix}`);
      return normalizeBriefing(raw);
    },
    async bulkStop(request: BulkStopRequest): Promise<BulkStopResponse> {
      const body = serializeBulkRequest(request);
      const raw = await apiClient.post<unknown>(API_ENDPOINTS.operationsBulkStop, body);
      return normalizeBulkResponse(raw);
    },
  };
}

export const operationsService = createOperationsService();

// Exported for tests so they can assert query-string composition without
// mocking the entire IApiClient surface.
export const __test__ = { buildOperationsQuery, serializeBulkRequest };
