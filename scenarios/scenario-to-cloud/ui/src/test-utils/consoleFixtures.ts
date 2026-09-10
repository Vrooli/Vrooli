import type { HealthObservation } from "../lib/api";
import { parseApiError } from "../lib/apiErrors";
import type { AuthzMatrix, CompiledPlanResponse, OperationStanding, RecoveryPoint } from "../types/console";

/** Fixture shapes mirror the API wire JSON exactly (snake_case, proto enum names). */

export const FIXTURE_DEPLOYMENT_ID = "0f4d2c1a-5b6e-4c7d-8e9f-0a1b2c3d4e5f";
export const FIXTURE_TARGET_KEY = "machine:fixture-host-203.0.113.10";

export function makeStanding(overrides: Partial<OperationStanding> = {}): OperationStanding {
  return {
    schema_version: "1",
    operation_id: "op-1234567890",
    deployment_id: FIXTURE_DEPLOYMENT_ID,
    request_key: "req-1",
    plan_digest: "sha256:plan-digest-fixture",
    state: "running",
    terminal: false,
    fence: 3,
    cancel_requested: false,
    active_step: "release.stage",
    completed_steps: ["host.prepare", "release.deliver", "release.verify"],
    step_receipts: [
      { step: "host.prepare", outcome: "succeeded", fence: 3, source: "target", completed_at: "2026-09-09T12:00:00Z" },
      { step: "release.deliver", outcome: "succeeded", fence: 3, source: "target", completed_at: "2026-09-09T12:00:10Z" },
      { step: "release.verify", outcome: "unchanged", fence: 3, source: "target", replayed: true, completed_at: "2026-09-09T12:00:20Z" },
    ],
    unknown_effects: [],
    next_action: { owner: "scenario-to-cloud", kind: "wait", reference: "/api/v1/operations/op-1234567890/wait" },
    reattach_command: "scenario-to-cloud operation wait op-1234567890",
    created_at: "2026-09-09T11:59:00Z",
    updated_at: "2026-09-09T12:00:20Z",
    ...overrides,
  };
}

export function makePlan(overrides: Partial<CompiledPlanResponse["plan"]> = {}, preview: Partial<CompiledPlanResponse["preview"]> = {}): CompiledPlanResponse {
  const plan: CompiledPlanResponse["plan"] = {
    schema_version: "1",
    deployment_id: FIXTURE_DEPLOYMENT_ID,
    scenario_id: "fixture-scenario",
    environment: "production",
    target: { machine_id: "fixture-host-203.0.113.10", node_id: "node-1", enrollment_generation: 3, transport: "bridge" },
    scope: "full",
    outcome: "apply",
    desired_revision: 7,
    release_digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    configuration_digest: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
    closure_digest: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
    policy_version: "1",
    preconditions: [{ kind: "fence", value: "3" }],
    actions: [
      {
        id: "a1",
        owner_operation: "release.stage",
        effect: "deployment_write",
        required_capability: "release.stage",
        inputs: {},
        depends_on: [],
        verification: "receipt",
        recovery: "rollback_release",
        retry: "safe_replay",
        cancel_point: true,
        downtime: { expected_seconds: 5, reason: "workload restart" },
      },
    ],
    presentation: { title: "Deploy release aaaaaaaaaaaa", summary: "Stage and activate the desired release.", downtime_note: "About 5 seconds of downtime.", recovery_note: "The predecessor release is retained." },
    ...overrides,
  };
  return {
    schema_version: "1",
    plan,
    plan_digest: "sha256:plan-digest-fixture",
    preview: {
      target: "machine:fixture-host-203.0.113.10 (enrollment 3, bridge)",
      outcome: plan.outcome,
      changes: [
        { action_id: "a1", operation: "release.stage", effect: "deployment_write", capability: "release.stage", summary: "Stage release aaaaaaaaaaaa", verification: "receipt", recovery: "rollback_release", retry: "safe_replay", cancel_point: true },
      ],
      data_effects: [{ action_id: "a1", subject: "postgres:main", effect: "backup_completed" }],
      downtime: { expected_seconds: 5, reason: "workload restart" },
      recovery_strategy: "rollback_release",
      shell_preview: [{ action_id: "a1", command: "vrooli cloud-target release stage --release sha256:aaaa --fence 3" }],
      handoff: plan.handoff,
      ...preview,
    },
    closure_status: "derived",
  };
}

export function makeObservation(overrides: Partial<HealthObservation> = {}): HealthObservation {
  return {
    deployment_id: FIXTURE_DEPLOYMENT_ID,
    target_id: FIXTURE_TARGET_KEY,
    observed_release_digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    observed_configuration_digest: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
    observed_at: "2026-09-09T12:00:00Z",
    status: "HEALTH_STATUS_HEALTHY",
    checks: [
      { id: "transport_reach", status: "CHECK_STATUS_PASSED", reason_code: "" },
      { id: "application_readiness", status: "CHECK_STATUS_PASSED", reason_code: "" },
    ],
    freshness: "FRESHNESS_CURRENT",
    producer_ref: "scenario-to-cloud:health:v1",
    partial: false,
    missing_dependencies: [],
    next_actions: [],
    ...overrides,
  };
}

export function makeRecoveryPoint(overrides: Partial<RecoveryPoint> = {}): RecoveryPoint {
  return {
    id: "rp-0001",
    deployment_id: FIXTURE_DEPLOYMENT_ID,
    binding_ids: ["postgres:main"],
    bindings: [{ id: "postgres:main", kind: "postgres", provider: "data-backup-manager", locator: "postgres://localhost:5432/app" }],
    schema_version: "12",
    configuration_digest: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
    release_digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    captured_at: "2026-09-09T11:00:00Z",
    provider: "data-backup-manager",
    encrypted: true,
    recovery_key_ref: "vault:recovery/rp-0001",
    retention_policy: "keep-last-5",
    protected: true,
    protected_by: ["release:sha256:aaaa"],
    migration_posture: "reversible",
    location: "s3://fixture/rp-0001",
    ...overrides,
  };
}

export function makeMatrix(): AuthzMatrix {
  return {
    schema_version: "1",
    scenario: "scenario-to-cloud",
    scopes: ["scenario-to-cloud:read", "scenario-to-cloud:write", "scenario-to-cloud:destructive"],
    routes: [
      { method: "POST", path: "/api/v1/deployments/{id}/plan", effect: "read", scope: "scenario-to-cloud:read", target_bound: true, service_allowed: false, description: "Compile" },
      { method: "POST", path: "/api/v1/deployments/{id}/plan/apply", effect: "workload_mutation", scope: "scenario-to-cloud:destructive", target_bound: true, service_allowed: false, description: "Apply" },
      { method: "POST", path: "/api/v1/operations/{id}/cancel", effect: "workload_mutation", scope: "scenario-to-cloud:destructive", target_bound: false, service_allowed: false, description: "Cancel" },
      { method: "GET", path: "/api/v1/deployments/{id}/recovery-points", effect: "read", scope: "scenario-to-cloud:read", target_bound: true, service_allowed: true, description: "List" },
      { method: "POST", path: "/api/v1/deployments/{id}/recovery-points/{rp}/restore", effect: "workload_mutation", scope: "scenario-to-cloud:destructive", target_bound: true, service_allowed: false, description: "Restore" },
      { method: "GET", path: "/api/v1/operations/", prefix: true, effect: "read", scope: "scenario-to-cloud:read", target_bound: false, service_allowed: true, description: "Operations" },
    ],
  };
}

/** Build a typed ApiError exactly as the API would have written it. */
export function apiErrorOf(status: number, code: string, message: string, extras: { retryable?: boolean; next_action?: Record<string, string>; details?: Record<string, unknown> } = {}) {
  return parseApiError(status, JSON.stringify({ error: { code, message, retryable: extras.retryable ?? false, next_action: extras.next_action, details: extras.details } }));
}
