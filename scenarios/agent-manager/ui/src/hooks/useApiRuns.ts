// Responsibility: read and control runs through the existing typed API.
import { useCallback, useEffect, useRef } from "react";

import { create } from "@bufbuild/protobuf";

import { buildTypedInvestigationRequest } from "../lib/typedInvestigation";

import type { ApproveResult, InvestigationFindings, Run, RunDiff, RunEvent, RunFormData, TypedInvestigation } from "../types";

import { StructuredResultStatus } from "../types";

import { RunConfigOverridesSchema } from "@vrooli/proto-types/agent-manager/v1/domain/profile_pb";

import { CreateRunRequestSchema, CreateRunResponseSchema, GetRunDiffResponseSchema, GetRunEventsResponseSchema, GetRunResponseSchema, ListRunsResponseSchema, PartialApproveRunRequestSchema, PartialApproveRunResponseSchema } from "@vrooli/proto-types/agent-manager/v1/api/service_pb";

import { ExtraFlagListSchema, FeatureFlagsSchema, SandboxConfigSchema } from "@vrooli/proto-types/agent-manager/v1/domain/types_pb";

import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { RunScope } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import type { TimeWindow } from "@vrooli/proto-types/measures/v1/measures_pb";
import { measuresClient } from "../features/stats/api/measuresClient";
import { useApiQuery } from "./useApiQuery";
import { apiRequest, parseProto, toProtoJson, durationFromMinutes, sandboxModeFromForm, networkAccessToProto } from "./useApiTransport";

export interface RunReportView {
  run_id: string;
  status: string;
  exit_code?: number;
  error?: string;
  duration_ms?: string;
  heartbeat_gap_ms?: string;
  turns: number;
  // ProtoJSON emits int64 as a decimal string; older servers emitted int32 as a number.
  tokens: number | string;
  cost_usd: number;
  result: { selection_status: string; selection_rule?: string; candidate_count: number; structured_status?: string; structured_method?: string; diagnostic_codes?: string[] };
  event_counts: Record<string, number>;
  tools: Array<{ name: string; calls: number; successes: number; failures: number; returned?: number; unknown_outcome?: number; unresolved?: number }>;
  project_owned_tool_calls: number;
  external_tool_calls: number;
  requested_model?: string;
  actual_model?: string;
  fallback_count: number;
  repeated_tool_calls: number;
  files_read_more_than_once: number;
  longest_event_gap_ms: string;
  diff: { files: number; bytes: number; available: { state: string; reason?: string } };
  events_availability: { state: string; reason?: string };
  receipts_availability: { state: string; reason?: string };
  receipt_count: number;
}

export function useRunReport(runId: string) {
  const report = useApiQuery<RunReportView>(["run-report", runId], (signal) =>
    apiRequest<RunReportView>(`/runs/${encodeURIComponent(runId)}/report`, { signal }), { enabled: !!runId });
  // With no run selected there is nothing to read, even on an explicit refresh.
  return { ...report, refetch: async () => { if (runId) await report.refetch(); } };
}

function buildRunConfigOverrides(run: RunFormData) {
  const payload: Record<string, unknown> = {};
  if (run.roleRef !== undefined) {
    payload.roleRef = run.roleRef.trim();
  }
  if (run.maxTurns !== undefined) {
    payload.maxTurns = run.maxTurns;
  }
  if (run.timeoutMinutes !== undefined) {
    payload.timeout = durationFromMinutes(run.timeoutMinutes);
  }
  if (run.effort !== undefined) {
    payload.effort = run.effort;
  }
  if (run.allowedTools !== undefined) {
    payload.allowedTools = run.allowedTools;
    if (run.allowedTools.length === 0) {
      payload.clearAllowedTools = true;
    }
  }
  if (run.deniedTools !== undefined) {
    payload.deniedTools = run.deniedTools;
    if (run.deniedTools.length === 0) {
      payload.clearDeniedTools = true;
    }
  }
  if (typeof run.skipPermissionPrompt === "boolean") {
    payload.skipPermissionPrompt = run.skipPermissionPrompt;
  }
  if (run.sandboxMode !== undefined) {
    // SandboxConfig.mode is the single source of truth for sandbox
    // selection — see agent-manager DeriveRunMode. Pass it via
    // sandboxConfig (the request struct's field), letting the
    // orchestrator's resolveSandboxConfig backfill the rest of the
    // contract defaults (auto-apply, manual-review, etc).
    payload.sandboxConfig = create(SandboxConfigSchema, {
      mode: sandboxModeFromForm(run.sandboxMode),
    });
  }
  if (run.networkAccess !== undefined) {
    payload.networkAccess = networkAccessToProto(run.networkAccess);
  }
  if (run.allowedPaths !== undefined) {
    payload.allowedPaths = run.allowedPaths;
    if (run.allowedPaths.length === 0) {
      payload.clearAllowedPaths = true;
    }
  }
  if (run.deniedPaths !== undefined) {
    payload.deniedPaths = run.deniedPaths;
    if (run.deniedPaths.length === 0) {
      payload.clearDeniedPaths = true;
    }
  }
  if (run.features !== undefined) {
    payload.features = create(FeatureFlagsSchema, {
      enableBrowser: run.features.enableBrowser ?? false,
    });
  }
  if (run.extraFlags !== undefined) {
    payload.extraFlags = Object.fromEntries(
      Object.entries(run.extraFlags).map(([rt, flags]) => [
        rt,
        create(ExtraFlagListSchema, { flags }),
      ])
    );
    if (Object.keys(run.extraFlags).length === 0) {
      payload.clearExtraFlags = true;
    }
  }
  return create(RunConfigOverridesSchema, payload);
}

function hasInlineConfig(run: RunFormData): boolean {
  return Boolean(
    run.roleRef !== undefined ||
      run.maxTurns !== undefined ||
      run.timeoutMinutes !== undefined ||
      run.effort !== undefined ||
      run.allowedTools !== undefined ||
      run.deniedTools !== undefined ||
      typeof run.skipPermissionPrompt === "boolean" ||
      run.sandboxMode !== undefined ||
      run.networkAccess !== undefined ||
      run.allowedPaths !== undefined ||
      run.deniedPaths !== undefined ||
      run.features !== undefined ||
      run.extraFlags !== undefined
  );
}

// Runs hook
/** Who started a run, as the runs list API filters it; empty lists every origin. */
export type RunOrigin = "" | "agent_manager" | "operator_sessions" | "workflows";

export function useRuns(options?: { enabled?: boolean; limit?: number; origin?: RunOrigin }) {
  const enabled = options?.enabled ?? true;
  const limit = options?.limit;
  const origin = options?.origin ?? "";
  const { data, loading, error, refetch: fetchRuns } = useApiQuery<Run[]>(
    ["runs", limit ?? null, origin],
    async (signal) => {
      // Sub-agent runs fold under their parent (see useSubAgentRuns), so the list holds top-level runs only.
      const query = `?top_level=true${origin ? `&origin=${origin}` : ""}${limit !== undefined ? `&limit=${encodeURIComponent(String(limit))}` : ""}`;
      return parseProto(ListRunsResponseSchema, await apiRequest<unknown>("/runs" + query, { signal })).runs ?? [];
    },
    // The list refreshes quietly in the background; only the first load shows a spinner.
    { enabled, fallback: [], firstLoadOnly: true },
  );

  const createRun = useCallback(
    async (run: RunFormData): Promise<Run> => {
  const inlineConfig = hasInlineConfig(run) ? buildRunConfigOverrides(run) : undefined;
      // Mint a fresh ConversationId per Decision D7 if the caller didn't
      // supply one. Each top-level "run task" click is conceptually a new
      // conversation; multi-turn flows pass an explicit conversationId.
      const conversationId = run.conversationId ?? crypto.randomUUID();
      const request = create(CreateRunRequestSchema, {
        taskId: run.taskId,
        agentProfileId: run.agentProfileId,
        tag: run.tag,
        runMode: run.runMode,
        executionMode: run.executionMode,
        inlineConfig,
        idempotencyKey: run.idempotencyKey,
        prompt: run.prompt,
        existingSandboxId: run.existingSandboxId,
        conversationId,
        parentRunId: run.parentRunId,
      });
      const created = await apiRequest<unknown>("/runs", {
        method: "POST",
        body: JSON.stringify(toProtoJson(CreateRunRequestSchema, request)),
      });
      const message = parseProto(CreateRunResponseSchema, created);
      const mapped = message.run as Run;
      await fetchRuns();
      return mapped;
    },
    [fetchRuns]
  );

  const retryRun = useCallback(
    async (run: Run): Promise<Run> => {
      const request: RunFormData = {
        taskId: run.taskId,
        agentProfileId: run.agentProfileId,
      };
      return createRun(request);
    },
    [createRun]
  );

  const startTypedInvestigation = useCallback(
    async (
      runIds: string[],
      customContext?: string,
      depth?: "quick" | "standard" | "deep"
    ): Promise<TypedInvestigation> => {
      const request = buildTypedInvestigationRequest(
        runIds,
        customContext ?? "",
        depth ?? "standard",
        `agent-manager-ui/${crypto.randomUUID()}`,
      );
      const response = await apiRequest<{ investigation: TypedInvestigation }>("/investigations", {
        method: "POST",
        body: JSON.stringify(request),
      });
      return response.investigation;
    },
    []
  );

  const applyInvestigation = useCallback(
    async (
      investigationRunId: string,
      selected: string[],
      customContext?: string,
      attachmentIds?: string[],
		overrides?: { roleRef?: string }
    ): Promise<Run> => {
      const created = await apiRequest<unknown>("/runs/investigation-apply", {
        method: "POST",
        body: JSON.stringify({
          investigationRunId,
          decision: "completed",
          selected,
          customContext,
          attachmentIds,
			roleRef: overrides?.roleRef,
        }),
      });
      const message = parseProto(CreateRunResponseSchema, created);
      const mapped = message.run as Run;
      await fetchRuns();
      return mapped;
    },
    [fetchRuns]
  );

  const resumeFromFailedRun = useCallback(
    async (
      runId: string,
      customContext?: string,
      attachmentIds?: string[]
    ): Promise<Run> => {
      const created = await apiRequest<unknown>("/runs/resume-from-failed", {
        method: "POST",
        body: JSON.stringify({ runId, customContext, attachmentIds }),
      });
      const message = parseProto(CreateRunResponseSchema, created);
      const mapped = message.run as Run;
      await fetchRuns();
      return mapped;
    },
    [fetchRuns]
  );

  const getRun = useCallback(async (id: string): Promise<Run> => {
    const run = await apiRequest<unknown>("/runs/" + id);
    const message = parseProto(GetRunResponseSchema, run);
    return message.run as Run;
  }, []);

  const stopRun = useCallback(
    async (id: string): Promise<void> => {
      await apiRequest<void>("/runs/" + id + "/stop", { method: "POST" });
      await fetchRuns();
    },
    [fetchRuns]
  );

  const deleteRun = useCallback(
    async (id: string): Promise<void> => {
      await apiRequest<void>("/runs/" + id, { method: "DELETE" });
      await fetchRuns();
    },
    [fetchRuns]
  );

  const getRunEvents = useCallback(
    async (id: string, options?: { afterSequence?: bigint }): Promise<RunEvent[]> => {
      const params = new URLSearchParams();
      if (options?.afterSequence !== undefined) {
        params.set("after_sequence", options.afterSequence.toString());
      }
      const query = params.size > 0 ? `?${params.toString()}` : "";
      const data = await apiRequest<unknown>("/runs/" + id + "/events" + query);
      const message = parseProto(GetRunEventsResponseSchema, data);
      return message.events ?? [];
    },
    []
  );

  const getRunDiff = useCallback(async (id: string): Promise<RunDiff> => {
    const data = await apiRequest<unknown>("/runs/" + id + "/diff");
    const message = parseProto(GetRunDiffResponseSchema, data);
    return message.diff as RunDiff;
  }, []);

  // Full-run approve/reject were retired from the agent-manager UI — a human does
  // those only in the Web Console approvals inbox (the single operator surface).
  // The backend /runs/{id}/approve|reject endpoints remain, driven by the inbox's
  // resolution webhook (ResolveRunApprovalFromHub). Partial approve
  // (file-selective) is NOT offered by the inbox, so it is kept here.
  const partialApproveRun = useCallback(
    async (id: string, fileIds: string[], actor?: string, commitMsg?: string): Promise<ApproveResult> => {
      const payload = create(PartialApproveRunRequestSchema, {
        runId: id,
        fileIds,
        actor: actor?.trim() || undefined,
        commitMsg: commitMsg || undefined,
      });
      const data = await apiRequest<unknown>("/runs/" + id + "/partial-approve", {
        method: "POST",
        body: JSON.stringify(toProtoJson(PartialApproveRunRequestSchema, payload)),
      });
      const message = parseProto(PartialApproveRunResponseSchema, data);
      await fetchRuns();
      return message.result as ApproveResult;
    },
    [fetchRuns]
  );

  const continueRun = useCallback(
    async (id: string, message: string, attachmentIds?: string[]): Promise<Run> => {
      const body: Record<string, unknown> = { message };
      if (attachmentIds && attachmentIds.length > 0) {
        body.attachment_ids = attachmentIds;
      }
      const data = await apiRequest<unknown>("/runs/" + id + "/continue", {
        method: "POST",
        body: JSON.stringify(body),
      });
      const response = data as { success: boolean; run: unknown; error?: string };
      if (!response.success && response.error) {
        throw new Error(response.error);
      }
      await fetchRuns();
      return response.run as Run;
    },
    [fetchRuns]
  );

  const deleteRunMessage = useCallback(
    async (runId: string, eventId: string): Promise<void> => {
      await apiRequest<void>(`/runs/${runId}/messages/${eventId}/delete`, {
        method: "POST",
      });
      await fetchRuns();
    },
    [fetchRuns]
  );

  return {
    data, loading, error,
    refetch: fetchRuns,
    createRun,
    retryRun,
    startTypedInvestigation,
    applyInvestigation,
    resumeFromFailedRun,
    getRun,
    stopRun,
    deleteRun,
    getRunEvents,
    getRunDiff,
    partialApproveRun,
    continueRun,
    deleteRunMessage,
  };
}

/** The runs listed under a run as its sub-agents, read once the list is opened. */
export function useSubAgentRuns(parentId: string, expanded: boolean) {
  return useApiQuery<Run[]>(
    ["sub-agents", parentId],
    async (signal) => parseProto(ListRunsResponseSchema, await apiRequest<unknown>(`/runs?parent_run_id=${encodeURIComponent(parentId)}&limit=200`, { signal })).runs ?? [],
    { enabled: expanded, fallback: [], staleTime: 30_000 },
  );
}

/** Live run counters plus the durable windowed total the dashboard header shows. */
export interface DashboardRunCounts {
  /** Runs currently running or starting, counted by the API (not the loaded page). */
  active: number;
  /** Runs awaiting review, counted by the API. */
  review: number;
  /** Runs Agent Manager executed in the durable Measures window ({@link DASHBOARD_WINDOW_PRESET}). */
  windowTotal: number;
  /** Operator sessions (imported, attached, interactive) observed in the same window. */
  operatorSessions: number;
}

export const DASHBOARD_WINDOW_PRESET = "7d" as const;
const DASHBOARD_COUNTS_POLL_MS = 60_000;
const DASHBOARD_COUNTS_DEBOUNCE_MS = 750;

async function countRunsWithStatus(status: "running" | "starting" | "needs_review"): Promise<number> {
  const resp = await apiRequest<unknown>(`/runs?status=${status}&limit=1`);
  return parseProto(ListRunsResponseSchema, resp).total;
}

const DASHBOARD_WINDOW_MS = 7 * 24 * 3600_000;

/** Runs in the trailing dashboard window for one scope, summed over every status. */
async function windowTotal(scope: RunScope): Promise<number> {
  const now = new Date();
  const range = { from: timestampFromDate(new Date(now.getTime() - DASHBOARD_WINDOW_MS)), to: timestampFromDate(now) };
  const response = await measuresClient.runStatusDistribution({ window: { window: { case: "custom", value: range } } as TimeWindow, filter: { scope } });
  return response.rows.reduce((total, row) => total + Number(row.count), 0);
}

async function fetchDashboardRunCounts(): Promise<DashboardRunCounts> {
  const [running, starting, review, executed, operatorSessions] = await Promise.all([
    countRunsWithStatus("running"),
    countRunsWithStatus("starting"),
    countRunsWithStatus("needs_review"),
    windowTotal(RunScope.AGENT_MANAGER),
    windowTotal(RunScope.OPERATOR_SESSIONS),
  ]);
  return { active: running + starting, review, windowTotal: executed, operatorSessions };
}

/**
 * Dashboard counters. `refetch` is debounced so a burst of run websocket status
 * events causes one read; a slow poll covers missed events while enabled.
 */
export function useDashboardRunCounts(options?: { enabled?: boolean }) {
  const enabled = options?.enabled ?? true;
  const counts = useApiQuery<DashboardRunCounts>(["dashboard-counts"], () => fetchDashboardRunCounts(), {
    enabled,
    refetchInterval: DASHBOARD_COUNTS_POLL_MS,
  });
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const { refetch: fetchCounts } = counts;

  const refetch = useCallback(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      debounceRef.current = null;
      void fetchCounts();
    }, DASHBOARD_COUNTS_DEBOUNCE_MS);
  }, [fetchCounts]);

  useEffect(() => () => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = null;
  }, []);

  return { data: counts.data, loading: counts.loading, error: counts.error, refetch };
}

// Reads the categorized recommendations out of an investigation run's
// structured result (run.result.structured, present once the run's
// `agent-manager/investigate` node completes with a schema-validated
// output). Returns null when the run hasn't produced a successful
// structured result yet (still running, failed, or extraction was not
// SUCCESS) — callers should treat that as "not ready" rather than an error.
export function getInvestigationFindings(run: Run | null | undefined): InvestigationFindings | null {
  const structured = run?.result?.structured;
  if (!structured || structured.status !== StructuredResultStatus.SUCCESS) {
    return null;
  }

  const raw: unknown = structured.value;
  let parsed: unknown;
  if (typeof raw === "string") {
    if (!raw.trim()) return null;
    try {
      parsed = JSON.parse(raw);
    } catch {
      return null;
    }
  } else if (ArrayBuffer.isView(raw)) {
    const bytes = new Uint8Array(raw.buffer, raw.byteOffset, raw.byteLength);
    if (bytes.length === 0) return null;
    try {
      parsed = JSON.parse(new TextDecoder().decode(bytes));
    } catch {
      return null;
    }
  } else if (raw && typeof raw === "object") {
    parsed = raw;
  } else {
    return null;
  }

  if (!parsed || typeof parsed !== "object") return null;
  const findings = parsed as Partial<InvestigationFindings>;
  if (!Array.isArray(findings.categories)) return null;

  return {
    summary: typeof findings.summary === "string" ? findings.summary : "",
    primaryCategory: findings.primaryCategory ?? "Both",
    confidence: findings.confidence,
    categories: findings.categories,
  };
}
