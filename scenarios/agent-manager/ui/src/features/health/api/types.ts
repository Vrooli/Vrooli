// Mirrors the API health audit handler responses.
//
// Source of truth on the Go side:
//   scenarios/agent-manager/api/internal/handlers/health_audit.go
//   scenarios/agent-manager/api/internal/health/types.go

export type HealthStatus = "ok" | "unknown" | "failed";

export interface ModelHealthRow {
  runner: string;
  model: string;
  status: HealthStatus;
  last_checked: string;
  reason?: string;
  message?: string;
}

export interface ModelHealthListResponse {
  models: ModelHealthRow[];
}

export interface RunnerHealthRow {
  runner: string;
  status: HealthStatus;
  last_checked: string;
  reason?: string;
  message?: string;
  catalog?: {
    observed_at?: string;
    age_days: number;
    budget_days: number;
    status: "fresh" | "stale" | "hard_stale" | "invalid" | "unknown";
  };
}

export interface RunnerHealthListResponse {
  runners: RunnerHealthRow[];
}

export interface ModelPolicyDriftSnapshot {
  last_run?: string;
  status: "healthy" | "warning" | "critical" | "not_measured";
  measured: number;
  total: number;
  findings?: Array<{ runner: string; type: string; severity: string; role?: string; model?: string; message: string; fingerprint: string }>;
  interval_hours: number;
}

export interface HealthAuditRow {
  id: number;
  timestamp: string;
  runnerType: string;
  modelId?: string;
  status: HealthStatus;
  reason?: string;
  message?: string;
  triggeredBy: string;
}

export interface HealthAuditResponse {
  rows: HealthAuditRow[];
  limit: number;
  scope: "model" | "runner";
}

export interface HealthAuditFilters {
  scope: "model" | "runner";
  runner?: string;
  model?: string;
  status?: HealthStatus;
  since?: string;
  until?: string;
  limit?: number;
}
