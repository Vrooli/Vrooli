/**
 * Wire shapes the deployment console renders. Field names are the JSON the
 * API writes (snake_case, proto names); see docs/reference/console-experience.md
 * and the owning Go packages (api/operations, api/execplan, api/authz,
 * api/backup, api/identity).
 */

export type NextAction = {
  owner?: string;
  kind?: string;
  reference?: string;
  label?: string;
};

export type TargetRef = {
  machine_id?: string;
  node_id?: string;
  enrollment_generation?: number;
  transport?: "bridge" | "ssh" | string;
  locator?: { host?: string; port?: number; user?: string; workdir?: string };
};

/** Fields of GET /deployments/{id} the console reads beyond the legacy type. */
export type DeploymentIdentity = {
  id: string;
  name: string;
  scenario_id: string;
  environment?: string;
  target?: TargetRef;
  fence?: number;
  status: string;
};

export type OperationState =
  | "admitted"
  | "waiting_input"
  | "running"
  | "verifying"
  | "reconciling"
  | "recovering"
  | "cancel_requested"
  | "succeeded"
  | "failed"
  | "failed_recovery"
  | "cancelled";

export const TERMINAL_OPERATION_STATES: readonly OperationState[] = [
  "succeeded",
  "failed",
  "failed_recovery",
  "cancelled",
];

export type StepReceipt = {
  step: string;
  owner_operation?: string;
  outcome: "succeeded" | "failed" | "skipped" | "unchanged" | "unknown" | string;
  fence: number;
  replayed?: boolean;
  source: string;
  detail?: string;
  error?: string;
  started_at?: string;
  completed_at: string;
};

export type UnknownEffect = {
  step: string;
  fence: number;
  reason: string;
  retry: string;
  next_action: string;
  recorded_at: string;
};

export type TypedErrorBody = {
  code: string;
  message: string;
  retryable?: boolean;
  next_action?: NextAction;
  details?: Record<string, unknown>;
};

export type OperationStanding = {
  schema_version: string;
  operation_id: string;
  deployment_id: string;
  request_key: string;
  plan_digest: string;
  state: OperationState | string;
  terminal: boolean;
  fence: number;
  worker_id?: string;
  lease_expires_at?: string;
  heartbeat_at?: string;
  cancel_requested: boolean;
  active_step?: string;
  completed_steps: string[];
  step_receipts: StepReceipt[];
  unknown_effects: UnknownEffect[];
  result?: { outcome: string; recovery_outcome?: string; completed_steps: number; message?: string };
  error?: TypedErrorBody;
  next_action?: NextAction;
  reattach_command: string;
  still_pending?: boolean;
  recommended_next_check_seconds?: number;
  created_at: string;
  updated_at: string;
  terminal_at?: string;
};

export type DeploymentOperationsResponse = {
  schema_version: string;
  deployment_id: string;
  operations: OperationStanding[];
  timestamp: string;
};

export type PlanOutcome = "apply" | "no_op" | "needs_input";

export type PlanAction = {
  id: string;
  owner_operation: string;
  effect: string;
  required_capability: string;
  inputs: Record<string, string>;
  depends_on: string[];
  verification: string;
  recovery: string;
  retry: string;
  cancel_point: boolean;
  downtime?: { expected_seconds: number; reason: string };
};

export type PlanHandoff = {
  owner: string;
  kind: string;
  reference: string;
  missing: string[];
};

export type ExecutablePlan = {
  schema_version: string;
  deployment_id: string;
  scenario_id: string;
  environment: string;
  target: { machine_id?: string; node_id?: string; enrollment_generation: number; transport: string };
  scope: string;
  outcome: PlanOutcome | string;
  desired_revision: number;
  release_digest: string;
  configuration_digest: string;
  closure_digest: string;
  policy_version: string;
  preconditions: { kind: string; value: string }[];
  actions: PlanAction[];
  handoff?: PlanHandoff;
  presentation: { title: string; summary: string; downtime_note?: string; recovery_note?: string };
};

export type PlanChange = {
  action_id: string;
  operation: string;
  effect: string;
  capability: string;
  summary: string;
  verification: string;
  recovery: string;
  retry: string;
  cancel_point: boolean;
};

export type PlanDataEffect = { action_id: string; subject: string; effect: string };

export type PlanPreview = {
  target: string;
  outcome: string;
  changes: PlanChange[];
  data_effects: PlanDataEffect[];
  downtime: { expected_seconds: number; reason: string };
  recovery_strategy: string;
  handoff?: PlanHandoff;
  shell_preview: { action_id: string; command: string }[];
};

export type CompiledPlanResponse = {
  schema_version: string;
  plan: ExecutablePlan;
  plan_digest: string;
  preview: PlanPreview;
  closure_status?: string;
};

export type PlanApplyResult = {
  schema_version: string;
  operation_id?: string;
  plan_digest: string;
  state: string;
};

/** 202 body shared by execute/start/apply. */
export type OperationAccepted = {
  schema_version: string;
  operation_id: string;
  plan_digest: string;
  state: string;
  replayed: boolean;
  wait: string;
  message: string;
  timestamp: string;
};

export type AuthzRoute = {
  method: string;
  path: string;
  prefix?: boolean;
  effect: string;
  scope?: string;
  target_bound: boolean;
  body_target?: boolean;
  remote_reach?: boolean;
  service_allowed: boolean;
  no_store?: boolean;
  optional?: boolean;
  description: string;
};

export type AuthzMatrix = {
  schema_version: string;
  scenario: string;
  scopes: string[];
  routes: AuthzRoute[];
};

export type DataBinding = {
  id?: string;
  kind?: string;
  owner?: string;
  name?: string;
  [key: string]: unknown;
};

export type RecoveryPoint = {
  id: string;
  deployment_id: string;
  operation_id?: string;
  binding_ids: string[];
  bindings: DataBinding[];
  schema_version: string;
  configuration_digest: string;
  release_digest: string;
  captured_at: string;
  provider: string;
  encrypted: boolean;
  recovery_key_ref: string;
  retention_policy: string;
  protected: boolean;
  protected_by?: string[];
  migration_posture: string;
  location: string;
};

export type RecoveryPlan = {
  kind: string;
  recovery_point_id?: string;
  preconditions: string[];
  steps: string[];
};

export type RollbackVerdict = {
  compatible: boolean;
  schema_strategy: string;
  current_schema: string;
  target_schema: string;
  reason_code?: string;
  reason?: string;
  plan?: RecoveryPlan;
};

/** Durable pointer the console and wizard resume from after an interruption. */
export type DurableOperationPointer = {
  deployment_id: string;
  operation_id: string;
  plan_digest: string;
};

/** One rendered action: availability and reason come from the API only. */
export type ActionState = {
  id: string;
  label: string;
  available: boolean;
  reason?: { code: string; message: string };
  nextAction?: NextAction;
  requiredScope?: string;
};
