import { buildApiUrl } from "@vrooli/api-base";
import { API_BASE } from "./api";
import { parseApiError } from "./apiErrors";
import type {
  AuthzMatrix,
  CompiledPlanResponse,
  DeploymentOperationsResponse,
  DurableOperationPointer,
  OperationStanding,
  PlanApplyResult,
  RecoveryPoint,
  RollbackVerdict,
} from "../types/console";

/**
 * requestJson performs one API call and turns every non-2xx answer into a
 * typed ApiError (code, retryable, next_action, details) so the console can
 * branch on codes and render remediation without reading message text.
 */
export async function requestJson<T>(path: string, init: RequestInit = {}): Promise<T> {
  const url = buildApiUrl(path, { baseUrl: API_BASE });
  const res = await fetch(url, {
    ...init,
    headers: { "Content-Type": "application/json", ...((init.headers ?? {}) as Record<string, string>) },
    cache: "no-store",
  });
  if (!res.ok) {
    throw parseApiError(res.status, await res.text());
  }
  return (await res.json()) as T;
}

export function getAuthzMatrix(): Promise<AuthzMatrix> {
  return requestJson<AuthzMatrix>("/authz/matrix");
}

/** Compile and preview the executable plan. Read scope; no records, no target effects. */
export function compileDeploymentPlan(deploymentId: string, scope?: string): Promise<CompiledPlanResponse> {
  return requestJson<CompiledPlanResponse>(`/deployments/${encodeURIComponent(deploymentId)}/plan`, {
    method: "POST",
    body: JSON.stringify(scope ? { scope } : {}),
  });
}

/** Admit the reviewed plan digest as a durable operation. */
export function applyDeploymentPlan(
  deploymentId: string,
  body: { plan_digest: string; request_key: string },
): Promise<PlanApplyResult> {
  return requestJson<PlanApplyResult>(`/deployments/${encodeURIComponent(deploymentId)}/plan/apply`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function getOperation(operationId: string): Promise<OperationStanding> {
  return requestJson<OperationStanding>(`/operations/${encodeURIComponent(operationId)}`);
}

/** Server-side wait; returns the standing when terminal or when the bound elapses. */
export function waitOperation(operationId: string, timeoutSeconds: number): Promise<OperationStanding> {
  return requestJson<OperationStanding>(
    `/operations/${encodeURIComponent(operationId)}/wait?timeout=${encodeURIComponent(String(timeoutSeconds))}`,
  );
}

export function cancelOperation(operationId: string): Promise<OperationStanding> {
  return requestJson<OperationStanding>(`/operations/${encodeURIComponent(operationId)}/cancel`, { method: "POST" });
}

export function listDeploymentOperations(deploymentId: string): Promise<DeploymentOperationsResponse> {
  return requestJson<DeploymentOperationsResponse>(`/deployments/${encodeURIComponent(deploymentId)}/operations`);
}

export function listRecoveryPoints(deploymentId: string): Promise<{ schema_version: string; recovery_points: RecoveryPoint[] }> {
  return requestJson(`/deployments/${encodeURIComponent(deploymentId)}/recovery-points`);
}

export function rollbackAdmission(
  deploymentId: string,
  body: { current_schema: string; target_schema: string; schema_strategy?: string; code_rollback?: boolean },
): Promise<{ schema_version: string; verdict: RollbackVerdict }> {
  return requestJson(`/deployments/${encodeURIComponent(deploymentId)}/recovery-points/rollback-admission`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function restoreRecoveryPoint(
  deploymentId: string,
  recoveryPointId: string,
  body: { into: Record<string, string>; bindings?: string[] },
): Promise<{ schema_version: string; receipt: unknown }> {
  return requestJson(
    `/deployments/${encodeURIComponent(deploymentId)}/recovery-points/${encodeURIComponent(recoveryPointId)}/restore`,
    { method: "POST", body: JSON.stringify({ deployment_id: deploymentId, recovery_point_id: recoveryPointId, ...body }) },
  );
}

/** Mint the idempotency key for one apply. Every retry of the same review reuses it. */
export function newRequestKey(): string {
  const c = globalThis.crypto;
  if (c && typeof c.randomUUID === "function") {
    return c.randomUUID();
  }
  const bytes = new Uint8Array(16);
  if (c && typeof c.getRandomValues === "function") {
    c.getRandomValues(bytes);
  } else {
    for (let i = 0; i < bytes.length; i += 1) bytes[i] = Math.floor(Math.random() * 256);
  }
  bytes[6] = ((bytes[6] ?? 0) & 0x0f) | 0x40;
  bytes[8] = ((bytes[8] ?? 0) & 0x3f) | 0x80;
  const hex = Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}

/* Durable pointer storage: the triple the console and the wizard resume from. */

export function operationPointerKey(deploymentId: string): string {
  return `stc.console.operation.${deploymentId}`;
}

export function saveOperationPointer(pointer: DurableOperationPointer): void {
  try {
    sessionStorage.setItem(operationPointerKey(pointer.deployment_id), JSON.stringify(pointer));
  } catch {
    // Storage may be unavailable; the API remains the durable source.
  }
}

export function loadOperationPointer(deploymentId: string): DurableOperationPointer | null {
  try {
    const raw = sessionStorage.getItem(operationPointerKey(deploymentId));
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<DurableOperationPointer>;
    if (
      typeof parsed.deployment_id !== "string" ||
      typeof parsed.operation_id !== "string" ||
      typeof parsed.plan_digest !== "string"
    ) {
      return null;
    }
    return { deployment_id: parsed.deployment_id, operation_id: parsed.operation_id, plan_digest: parsed.plan_digest };
  } catch {
    return null;
  }
}

export function clearOperationPointer(deploymentId: string): void {
  try {
    sessionStorage.removeItem(operationPointerKey(deploymentId));
  } catch {
    // ignore
  }
}
