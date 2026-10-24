// Responsibility: read health, workflow, watch and policy state and request maintenance.
import { useCallback, useEffect, useRef } from "react";

import { create, fromJson, type MessageShape, type JsonValue } from "@bufbuild/protobuf";

import { ValueSchema } from "@bufbuild/protobuf/wkt";

import { runnerTypeToSlug } from "../lib/utils";

import type { HealthResponse, InvestigationContextFlags, InvestigationDepth, InvestigationSettings, InvestigationTagRule, ProbeResult, RunnerStatus, RunnerType } from "../types";

import { GetPermissionPolicyCatalogResponseSchema, GetPermissionPolicyStatusResponseSchema, GetRolePolicyCatalogResponseSchema, GetRunnerStatusResponseSchema, PurgeDataRequestSchema, PurgeDataResponseSchema, PurgeTarget, ProbeRunnerResponseSchema, DoctorPermissionPolicyResponseSchema, PlanPermissionPolicyResponseSchema, ReconcilePermissionPolicyRequestSchema, ReconcilePermissionPolicyResponseSchema, ReloadPermissionPolicyCatalogResponseSchema, ValidatePermissionPolicyCatalogResponseSchema, GetWorkflowExecutionTraceResponseSchema, ListWorkflowExecutionsResponseSchema, SignalWorkflowExecutionRequestSchema, WorkflowExecutionOperationRequestSchema, WorkflowExecutionOperationResponseSchema } from "@vrooli/proto-types/agent-manager/v1/api/service_pb";

import type { WorkflowExecution, WorkflowJournalEntry, WorkflowNodeAttempt } from "@vrooli/proto-types/agent-manager/v1/domain/workflow_pb";

import type { CohortWatch, InspectCohortWatchResponse } from "@vrooli/proto-types/agent-manager/v1/domain/watch_pb";

import { InspectCohortWatchResponseSchema, ListCohortWatchesResponseSchema } from "@vrooli/proto-types/agent-manager/v1/domain/watch_pb";

import { HealthResponseSchema } from "@vrooli/proto-types/common/v1/types_pb";

import { useApiState, apiRequest, parseProto, toProtoJson, normalizeJsonValueInput, protoReadOptions, normalizeHealthResponseJson } from "./useApiTransport";

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
  const { data, loading, error, setData, setLoading, setError } = useApiState<WorkflowExecution[]>([]);

  const refetch = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const raw = await apiRequest<unknown>("/workflow-executions?limit=100");
      const response = parseProto(ListWorkflowExecutionsResponseSchema, raw);
      setData(response.executions);
    } catch (error) {
      setError(error instanceof Error ? error.message : "Failed to load workflow executions");
    } finally {
      setLoading(false);
    }
  }, [setData, setError, setLoading]);

  useEffect(() => {
    void refetch();
  }, [refetch]);

  useEffect(() => {
    const refresh = () => void refetch();
    window.addEventListener("agent-manager:workflow-lifecycle", refresh);
    return () => window.removeEventListener("agent-manager:workflow-lifecycle", refresh);
  }, [refetch]);

  const getTrace = useCallback(async (executionId: string): Promise<WorkflowTraceView> => {
    const raw = await apiRequest<unknown>(`/workflow-executions/${encodeURIComponent(executionId)}/trace?limit=500`);
    const response = parseProto(GetWorkflowExecutionTraceResponseSchema, raw);
    return { execution: response.execution, attempts: response.attempts, journal: response.journal };
  }, []);

  const control = useCallback(async (execution: WorkflowExecution, operation: "cancel" | "retry" | "resume") => {
    const request = create(WorkflowExecutionOperationRequestSchema, {
      executionId: execution.id,
      idempotencyKey: `ui-${operation}-${execution.id}-${execution.version.toString()}`,
      expectedVersion: execution.version,
      reason: "Operator action from Agent Manager workflow console",
    });
    const raw = await apiRequest<unknown>(`/workflow-executions/${encodeURIComponent(execution.id)}/${operation}`, {
      method: "POST",
      body: JSON.stringify(toProtoJson(WorkflowExecutionOperationRequestSchema, request)),
    });
    const response = parseProto(WorkflowExecutionOperationResponseSchema, raw);
    await refetch();
    return response.execution;
  }, [refetch]);

  const signal = useCallback(async (execution: WorkflowExecution, name: string, payload: unknown) => {
    const request = create(SignalWorkflowExecutionRequestSchema, {
      executionId: execution.id,
      signal: name,
      payload: fromJson(ValueSchema, normalizeJsonValueInput(payload) as JsonValue, protoReadOptions),
      idempotencyKey: `ui-signal-${execution.id}-${execution.version.toString()}-${name}`,
      expectedVersion: execution.version,
    });
    const raw = await apiRequest<unknown>(`/workflow-executions/${encodeURIComponent(execution.id)}/signals`, {
      method: "POST",
      body: JSON.stringify(toProtoJson(SignalWorkflowExecutionRequestSchema, request)),
    });
    const response = parseProto(WorkflowExecutionOperationResponseSchema, raw);
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
  const { data, loading, error, setData, setLoading, setError } = useApiState<CohortWatch[]>([]);

  const refetch = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const raw = await apiRequest<unknown>("/cohort-watches?page_size=100");
      const response = parseProto(ListCohortWatchesResponseSchema, raw);
      setData(response.watches);
    } catch (error) {
      setError(error instanceof Error ? error.message : "Failed to load cohort watches");
    } finally {
      setLoading(false);
    }
  }, [setData, setError, setLoading]);

  useEffect(() => {
    void refetch();
  }, [refetch]);

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

// Health hook
export function useHealth() {
  const { data, loading, error, setData, setLoading, setError } = useApiState<HealthResponse>();
  const abortRef = useRef<AbortController | null>(null);

  const fetchHealth = useCallback(async () => {
    if (abortRef.current) {
      abortRef.current.abort();
    }
    const controller = new AbortController();
    abortRef.current = controller;

    setLoading(true);
    setError(null);

    try {
      const data = await apiRequest<unknown>("/health", {
        signal: controller.signal,
      });
      const normalized = normalizeHealthResponseJson(data);
      const message = parseProto(HealthResponseSchema, normalized);
      setData(message as HealthResponse);
    } catch (err) {
      if ((err as Error).name !== "AbortError") {
        setError((err as Error).message);
      }
    } finally {
      setLoading(false);
    }
  }, [setData, setLoading, setError]);

  useEffect(() => {
    let timeoutId: ReturnType<typeof setTimeout>;
    let cancelled = false;

    const poll = async () => {
      await fetchHealth();
      if (!cancelled) {
        timeoutId = setTimeout(() => {
          void poll();
        }, 30000);
      }
    };

    void poll();

    return () => {
      cancelled = true;
      clearTimeout(timeoutId);
      abortRef.current?.abort();
    };
  }, [fetchHealth]);

  return { data, loading, error, refetch: fetchHealth };
}

// Runners hook
export function useRunners(options?: { enabled?: boolean }) {
  const enabled = options?.enabled ?? true;
  const { data, loading, error, setData, setLoading, setError } = useApiState<Record<string, RunnerStatus>>({});

  const fetchRunners = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiRequest<unknown>("/runners");
      const message = parseProto(GetRunnerStatusResponseSchema, data);
      const record: Record<string, RunnerStatus> = {};
      for (const runner of message.runners ?? []) {
        record[String(runner.runnerType)] = runner;
      }
      setData(record);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [setData, setLoading, setError]);

  useEffect(() => {
    if (!enabled) {
      return;
    }
    void fetchRunners();
  }, [enabled, fetchRunners]);

  return { data, loading, error, refetch: fetchRunners };
}

// Role-policy catalog hook. This is a read-only projection of Git-managed
// declared state; reload and validation remain explicit operator commands.
export function useRolePolicyCatalog(options?: { enabled?: boolean }) {
	const enabled = options?.enabled ?? true;
	const { data, loading, error, setData, setLoading, setError } = useApiState<MessageShape<typeof GetRolePolicyCatalogResponseSchema> | null>(null);

  const fetchCatalog = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
		const payload = await apiRequest<unknown>("/role-policy/catalog");
		setData(parseProto(GetRolePolicyCatalogResponseSchema, payload));
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [setData, setLoading, setError]);

  useEffect(() => {
    if (enabled) void fetchCatalog();
  }, [enabled, fetchCatalog]);

  return { data, loading, error, refetch: fetchCatalog };
}

type PermissionPolicyData = {
  status: MessageShape<typeof GetPermissionPolicyStatusResponseSchema>;
  catalog: MessageShape<typeof GetPermissionPolicyCatalogResponseSchema>;
};

// Permission-policy actions are intentionally whole-document operations. The
// UI never exposes resource-native patterns or individual rule mutation.
export function usePermissionPolicy(options?: { enabled?: boolean }) {
  const enabled = options?.enabled ?? true;
  const { data, loading, error, setData, setLoading, setError } = useApiState<PermissionPolicyData | null>(null);

  const refetch = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [statusPayload, catalogPayload] = await Promise.all([
        apiRequest<unknown>("/permission-policy/status"),
        apiRequest<unknown>("/permission-policy/catalog"),
      ]);
      setData({
        status: parseProto(GetPermissionPolicyStatusResponseSchema, statusPayload),
        catalog: parseProto(GetPermissionPolicyCatalogResponseSchema, catalogPayload),
      });
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [setData, setLoading, setError]);

  useEffect(() => {
    if (enabled) void refetch();
  }, [enabled, refetch]);

  const validate = useCallback(async () => {
    const payload = await apiRequest<unknown>("/permission-policy/validate", { method: "POST" });
    const response = parseProto(ValidatePermissionPolicyCatalogResponseSchema, payload);
    await refetch();
    return response;
  }, [refetch]);

  const reload = useCallback(async () => {
    const payload = await apiRequest<unknown>("/permission-policy/reload", { method: "POST" });
    const response = parseProto(ReloadPermissionPolicyCatalogResponseSchema, payload);
    await refetch();
    return response;
  }, [refetch]);

  const plan = useCallback(async () => {
    const payload = await apiRequest<unknown>("/permission-policy/plan", { method: "POST" });
    return parseProto(PlanPermissionPolicyResponseSchema, payload);
  }, []);

  const doctor = useCallback(async () => {
    const payload = await apiRequest<unknown>("/permission-policy/doctor", { method: "POST" });
    return parseProto(DoctorPermissionPolicyResponseSchema, payload);
  }, []);

  const reconcile = useCallback(async () => {
    const request = create(ReconcilePermissionPolicyRequestSchema, { explicitlyAuthorized: true });
    const payload = await apiRequest<unknown>("/permission-policy/reconcile", {
      method: "POST",
      body: JSON.stringify(toProtoJson(ReconcilePermissionPolicyRequestSchema, request)),
    });
    const response = parseProto(ReconcilePermissionPolicyResponseSchema, payload);
    await refetch();
    return response;
  }, [refetch]);

  return { data, loading, error, refetch, validate, reload, plan, doctor, reconcile };
}

// Probe runner function (standalone for use in components)
export async function probeRunner(runnerType: RunnerType): Promise<ProbeResult> {
  const data = await apiRequest<unknown>(`/runners/${runnerTypeToSlug(runnerType)}/probe`, {
    method: "POST",
  });
  const message = parseProto(ProbeRunnerResponseSchema, data);
  return message.result as ProbeResult;
}

// Investigation Settings hook
export function useInvestigationSettings() {
  const { data, loading, error, setData, setLoading, setError } = useApiState<InvestigationSettings | null>(null);

  const fetchSettings = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiRequest<InvestigationSettings>("/investigation-settings");
      setData(data);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [setData, setLoading, setError]);

  const updateSettings = useCallback(async (settings: Partial<{
    promptTemplate: string;
    applyPromptTemplate: string;
    defaultDepth: InvestigationDepth;
    defaultContext: InvestigationContextFlags;
    investigationTagAllowlist: InvestigationTagRule[];
  }>): Promise<InvestigationSettings> => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiRequest<InvestigationSettings>("/investigation-settings", {
        method: "PUT",
        body: JSON.stringify(settings),
      });
      setData(data);
      return data;
    } catch (err) {
      setError((err as Error).message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [setData, setLoading, setError]);

  const resetSettings = useCallback(async (): Promise<InvestigationSettings> => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiRequest<InvestigationSettings>("/investigation-settings/reset", {
        method: "POST",
      });
      setData(data);
      return data;
    } catch (err) {
      setError((err as Error).message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [setData, setLoading, setError]);

  useEffect(() => {
    fetchSettings();
  }, [fetchSettings]);

  return {
    data, loading, error,
    refetch: fetchSettings,
    updateSettings,
    resetSettings,
  };
}

// Maintenance hook
export function useMaintenance() {
  const previewPurge = useCallback(async (pattern: string, targets: PurgeTarget[]): Promise<PurgeCounts> => {
    const payload = create(PurgeDataRequestSchema, {
      pattern,
      targets,
      dryRun: true,
    });
    const data = await apiRequest<unknown>("/maintenance/purge", {
      method: "POST",
      body: JSON.stringify(toProtoJson(PurgeDataRequestSchema, payload)),
    });
    const message = parseProto(PurgeDataResponseSchema, data);
    return message.matched ?? {};
  }, []);

  const executePurge = useCallback(async (pattern: string, targets: PurgeTarget[]): Promise<PurgeCounts> => {
    const payload = create(PurgeDataRequestSchema, {
      pattern,
      targets,
      dryRun: false,
    });
    const data = await apiRequest<unknown>("/maintenance/purge", {
      method: "POST",
      body: JSON.stringify(toProtoJson(PurgeDataRequestSchema, payload)),
    });
    const message = parseProto(PurgeDataResponseSchema, data);
    return message.deleted ?? {};
  }, []);

  return {
    previewPurge,
    executePurge,
  };
}
