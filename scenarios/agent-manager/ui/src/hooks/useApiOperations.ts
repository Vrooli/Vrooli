// Responsibility: read health, workflow, watch and policy state and request maintenance.
import { useCallback, useEffect } from "react";

import { fromJson, type JsonValue } from "@bufbuild/protobuf";

import { ValueSchema } from "@bufbuild/protobuf/wkt";

import type { HealthResponse, InvestigationContextFlags, InvestigationDepth, InvestigationSettings, InvestigationTagRule, ProbeResult, RunnerType } from "../types";

import type { GetPermissionPolicyCatalogResponse, GetPermissionPolicyStatusResponse, GetRolePolicyCatalogResponse, PurgeTarget } from "@vrooli/proto-types/agent-manager/v1/api/service_pb";

import type { WorkflowExecution, WorkflowJournalEntry, WorkflowNodeAttempt } from "@vrooli/proto-types/agent-manager/v1/domain/workflow_pb";

import type { CohortWatch, InspectCohortWatchResponse } from "@vrooli/proto-types/agent-manager/v1/domain/watch_pb";

import { InspectCohortWatchResponseSchema, ListCohortWatchesResponseSchema } from "@vrooli/proto-types/agent-manager/v1/domain/watch_pb";

import { HealthResponseSchema } from "@vrooli/proto-types/common/v1/types_pb";

import { agentManager, CALL_DEADLINE } from "../lib/api";

import { queryClient } from "../providers/QueryProvider";
import { useApiMutation, useApiQuery } from "./useApiQuery";
import { apiRequest, parseProto, normalizeJsonValueInput, protoReadOptions, normalizeHealthResponseJson, healthRequest, healthReason } from "./useApiTransport";

const INVESTIGATION_SETTINGS_KEY = ["investigation-settings"] as const;

type PurgeCounts = {
  profiles?: number;
  tasks?: number;
  runs?: number;
};

export interface WorkflowTraceView {
  execution?: WorkflowExecution;
  attempts: WorkflowNodeAttempt[];
  journal: WorkflowJournalEntry[];
}

export function useWorkflowExecutions() {
  const { data, loading, error, refetch } = useApiQuery<WorkflowExecution[]>(
    ["workflow-executions"],
    async (signal) => (await agentManager.listWorkflowExecutions({ limit: 100 }, { signal, ...CALL_DEADLINE })).executions,
    { fallback: [] },
  );

  useEffect(() => {
    const refresh = () => void refetch();
    window.addEventListener("agent-manager:workflow-lifecycle", refresh);
    return () => window.removeEventListener("agent-manager:workflow-lifecycle", refresh);
  }, [refetch]);

  const getTrace = useCallback(async (executionId: string): Promise<WorkflowTraceView> => {
    const response = await agentManager.getWorkflowExecutionTrace({ executionId, limit: 500 }, CALL_DEADLINE);
    return { execution: response.execution, attempts: response.attempts, journal: response.journal };
  }, []);

  const control = useCallback(async (execution: WorkflowExecution, operation: "cancel" | "retry" | "resume") => {
    const response = await agentManager[`${operation}WorkflowExecution`]({
      executionId: execution.id,
      idempotencyKey: `ui-${operation}-${execution.id}-${execution.version.toString()}`,
      expectedVersion: execution.version,
      reason: "Operator action from Agent Manager workflow console",
    }, CALL_DEADLINE);
    await refetch();
    return response.execution;
  }, [refetch]);

  const signal = useCallback(async (execution: WorkflowExecution, name: string, payload: unknown) => {
    const response = await agentManager.signalWorkflowExecution({
      executionId: execution.id,
      signal: name,
      payload: fromJson(ValueSchema, normalizeJsonValueInput(payload) as JsonValue, protoReadOptions),
      idempotencyKey: `ui-signal-${execution.id}-${execution.version.toString()}-${name}`,
      expectedVersion: execution.version,
    }, CALL_DEADLINE);
    await refetch();
    return response.execution;
  }, [refetch]);

  return { data, loading, error, refetch, getTrace, control, signal };
}

export interface CohortWatchInspection {
  inspection: InspectCohortWatchResponse;
  actions: CohortWatchActionView[];
}

export interface CohortWatchActionView {
  actionId: string;
  kind: number;
  targetRunId: string;
  state: number;
  status: string;
  rejectionReason: string;
}

function normalizeCohortWatchActions(raw: unknown): CohortWatchActionView[] {
  if (typeof raw !== "object" || raw === null || !("actions" in raw) || !Array.isArray(raw.actions)) return [];
  return raw.actions.flatMap((value): CohortWatchActionView[] => {
    if (typeof value !== "object" || value === null) return [];
    const action = value as Record<string, unknown>;
    const text = (camel: string, snake: string) => typeof action[camel] === "string" ? action[camel] as string : typeof action[snake] === "string" ? action[snake] as string : "";
    const numeric = (key: string) => typeof action[key] === "number" ? action[key] as number : 0;
    return [{
      actionId: text("actionId", "action_id"),
      kind: numeric("kind"),
      targetRunId: text("targetRunId", "target_run_id"),
      state: numeric("state"),
      status: text("status", "status"),
      rejectionReason: text("rejectionReason", "rejection_reason"),
    }];
  });
}

export function useCohortWatches() {
  const { data, loading, error, refetch } = useApiQuery<CohortWatch[]>(
    ["cohort-watches"],
    async (signal) => parseProto(ListCohortWatchesResponseSchema, await apiRequest<unknown>("/cohort-watches?page_size=100", { signal })).watches,
    { fallback: [] },
  );

  const inspect = useCallback(async (watchId: string): Promise<CohortWatchInspection> => {
    const encodedID = encodeURIComponent(watchId);
    const [inspectionRaw, actionsRaw] = await Promise.all([
      apiRequest<unknown>(`/cohort-watches/${encodedID}/inspect?event_limit=100`),
      apiRequest<unknown>(`/cohort-watches/${encodedID}/actions?limit=100`),
    ]);
    return {
      inspection: parseProto(InspectCohortWatchResponseSchema, inspectionRaw),
      actions: normalizeCohortWatchActions(actionsRaw),
    };
  }, []);

  return { data, loading, error, refetch, inspect };
}

// Health hook: polled every 30 s; the reason a degraded answer gives rides along with the document.
const HEALTH_POLL_MS = 30_000;

export function useHealth() {
  const query = useApiQuery<{ health: HealthResponse; reason: string }>(["health"], async (signal) => {
    const raw = await healthRequest(signal);
    return { health: parseProto(HealthResponseSchema, normalizeHealthResponseJson(raw)) as HealthResponse, reason: healthReason(raw) };
  }, { refetchInterval: HEALTH_POLL_MS });
  return { data: query.data?.health ?? null, reason: query.data?.reason ?? "", loading: query.loading, error: query.error, refetch: query.refetch };
}

// Role-policy catalog hook. This is a read-only projection of Git-managed
// declared state; reload and validation remain explicit operator commands.
export function useRolePolicyCatalog(options?: { enabled?: boolean }) {
	const enabled = options?.enabled ?? true;
	return useApiQuery<GetRolePolicyCatalogResponse>(["role-policy-catalog"], (signal) =>
    agentManager.getRolePolicyCatalog({}, { signal, ...CALL_DEADLINE }), { enabled });
}

type PermissionPolicyData = {
  status: GetPermissionPolicyStatusResponse;
  catalog: GetPermissionPolicyCatalogResponse;
};

// Permission-policy actions are intentionally whole-document operations. The
// UI never exposes resource-native patterns or individual rule mutation.
export function usePermissionPolicy(options?: { enabled?: boolean }) {
  const enabled = options?.enabled ?? true;
  const { data, loading, error, refetch } = useApiQuery<PermissionPolicyData>(["permission-policy"], async (signal) => {
    const [status, catalog] = await Promise.all([
      agentManager.getPermissionPolicyStatus({}, { signal, ...CALL_DEADLINE }),
      agentManager.getPermissionPolicyCatalog({}, { signal, ...CALL_DEADLINE }),
    ]);
    return { status, catalog };
  }, { enabled });

  const validate = useCallback(async () => {
    const response = await agentManager.validatePermissionPolicyCatalog({}, CALL_DEADLINE);
    await refetch();
    return response;
  }, [refetch]);

  const reload = useCallback(async () => {
    const response = await agentManager.reloadPermissionPolicyCatalog({}, CALL_DEADLINE);
    await refetch();
    return response;
  }, [refetch]);

  const plan = useCallback(async () => {
    return agentManager.planPermissionPolicy({}, CALL_DEADLINE);
  }, []);

  const doctor = useCallback(async () => {
    return agentManager.doctorPermissionPolicy({}, CALL_DEADLINE);
  }, []);

  const reconcile = useCallback(async () => {
    const response = await agentManager.reconcilePermissionPolicy({ explicitlyAuthorized: true }, CALL_DEADLINE);
    await refetch();
    return response;
  }, [refetch]);

  return { data, loading, error, refetch, validate, reload, plan, doctor, reconcile };
}

// Probe runner function (standalone for use in components)
export async function probeRunner(runnerType: RunnerType): Promise<ProbeResult> {
  const message = await agentManager.probeRunner({ runnerType }, CALL_DEADLINE);
  return message.result as ProbeResult;
}

// Investigation Settings hook
type InvestigationSettingsPatch = Partial<{
  promptTemplate: string;
  applyPromptTemplate: string;
  defaultDepth: InvestigationDepth;
  defaultContext: InvestigationContextFlags;
  investigationTagAllowlist: InvestigationTagRule[];
}>;

export function useInvestigationSettings() {
  const query = useApiQuery<InvestigationSettings>(INVESTIGATION_SETTINGS_KEY, (signal) =>
    apiRequest<InvestigationSettings>("/investigation-settings", { signal }));
  // One mutation for both writes, so `error` is always the most recent write's.
  const write = useApiMutation(async (patch?: InvestigationSettingsPatch) => {
    const result = patch
      ? await apiRequest<InvestigationSettings>("/investigation-settings", { method: "PUT", body: JSON.stringify(patch) })
      : await apiRequest<InvestigationSettings>("/investigation-settings/reset", { method: "POST" });
    queryClient.setQueryData(INVESTIGATION_SETTINGS_KEY, result);
    return result;
  });

  return {
    data: query.data,
    loading: query.loading || write.loading,
    error: write.error ?? query.error,
    refetch: query.refetch,
    updateSettings: (patch: InvestigationSettingsPatch) => write.run(patch),
    resetSettings: () => write.run(),
  };
}

// Maintenance hook
export function useMaintenance() {
  const previewPurge = useCallback(async (pattern: string, targets: PurgeTarget[]): Promise<PurgeCounts> => {
    const message = await agentManager.purgeData({ pattern, targets, dryRun: true }, CALL_DEADLINE);
    return message.matched ?? {};
  }, []);

  const executePurge = useCallback(async (pattern: string, targets: PurgeTarget[]): Promise<PurgeCounts> => {
    const message = await agentManager.purgeData({ pattern, targets, dryRun: false }, CALL_DEADLINE);
    return message.deleted ?? {};
  }, []);

  return {
    previewPurge,
    executePurge,
  };
}
