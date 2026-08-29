export interface Resource {
  name: string;
  status: string;
  category: string;
  installed: boolean;
}

export interface ResourceHealthStatus {
  name: string;
  status: string;
  category: string;
  available: boolean;
  last_checked: string;
}

export interface ResourceHealthResponse {
  resources: ResourceHealthStatus[];
  total: number;
  healthy_count: number;
  checked_at: string;
}

export interface GlossaryEntry {
  term: string;
  description: string;
  category: string;
}

export interface GlossaryResponse {
  entries: GlossaryEntry[];
  count: number;
  query?: string;
}

export interface V2Scenario {
  name: string;
  description?: string;
  system_required: boolean;
  enabled: boolean;
  auto_restart: boolean;
  resources: string[];
}

export interface V2ScenarioResponse {
  scenarios: V2Scenario[];
  count: number;
}
export interface V2Recommendation {
  profile: string;
  scenarios: string[];
  resources: string[];
  explanation: string;
}

export interface SupervisionAttributionStep {
  name: string;
  kind: "scenario" | "resource";
  declared_by?: string;
  supervision_intent: "must_start" | "try_start";
  source: string;
}
export interface SupervisionMember {
  name: string;
  kind: "scenario" | "resource";
  supervision_intent: "must_start" | "try_start";
  attribution_chain: SupervisionAttributionStep[];
}
export interface V2CoreSetResponse {
  available: boolean;
  seed: string[];
  trusted_base: string[];
  members: SupervisionMember[];
  member_counts?: Record<string, number>;
  load_errors?: Record<string, string>;
  error?: string;
}

export interface ClosureMember {
  name: string;
  required: boolean;
  direct: boolean;
  provenance: { kind: "selected" | "required" | "try_start"; from?: string }[];
}
export interface V2ClosureResponse {
  scenarios: ClosureMember[];
  resources: ClosureMember[];
}
export interface V2Resource {
  name: string;
  display_name?: string;
  description?: string;
  category?: string;
  enabled: boolean;
  installed: boolean;
}
export interface V2ResourceResponse {
  resources: V2Resource[];
  required: V2Resource[];
  optional: V2Resource[];
  standalone: V2Resource[];
  count: number;
}

export interface CredentialReadiness {
  resource: string;
  logical_id: string;
  field: string;
  label: string;
  description?: string;
  obtain_url?: string;
  required: boolean;
  /** operator: a person provides it. derived: written once its source credential exists. generated: minted by the owning component on first start. */
  provisioning?: "operator" | "derived" | "generated";
  derived_from?: string;
  status: "configured" | "unconfigured" | "unsupported";
  detail?: string;
}

export interface V2ReadinessResponse {
  status: "ready" | "degraded" | "missing" | "unsupported";
  scenarios: string[];
  resources: string[];
  credentials: CredentialReadiness[];
  hosts: ReadinessItem[];
  integrations: ReadinessItem[];
  checked_at: string;
  credential_diagnosis?: CredentialDiagnosis;
  recovery?: RecoveryReadiness;
  /** Named reasons configuration is not complete. Metadata only. */
  blockers?: CompletionBlocker[];
  /** Optional gaps. They stop completion only until this exact set is acknowledged. */
  degraded?: CompletionBlocker[];
  /** Identifies the degraded set an acknowledgement applies to. */
  degraded_digest?: string;
  degraded_acknowledged?: boolean;
}
export interface CompletionBlocker {
  kind: "credential" | "host" | "recovery" | "apply" | "readiness";
  name: string;
  reason: string;
  remediation: string;
}
export interface RecoveryReadiness {
  receipt_exists: boolean;
  exported_at?: string;
  entry_count: number;
  uncovered: string[];
  required_absent?: string[];
  required_absent_details?: Array<{ address: string; description?: string }>;
  root_copy?: {
    path?: string;
    sink?: string;
    copied_at?: string;
    generation?: string;
  } | null;
  root_copy_issues?: string[];
}
export interface V2ApplyResponse {
  run_id: string;
  status: string;
  items: Array<{
    id: string;
    kind: string;
    name: string;
    outcome: string;
    error?: string;
    remediation?: string;
  }>;
  blockers?: CompletionBlocker[];
  degraded?: CompletionBlocker[];
  degraded_digest?: string;
}
export interface V2ApplyPlanResponse {
  items: Array<{
    id: string;
    kind: string;
    name: string;
    dependencies?: string[];
    required: boolean;
    /** True when the item is executed with elevation and changes host state. */
    privileged?: boolean;
    /** "satisfied" | "pending" | "unknown"; absent on APIs predating state. */
    state?: string;
  }>;
}
export interface V2SessionResponse {
  step: number;
  first_unsatisfied_step: number;
  completion: boolean;
}
export interface V2Step {
  id: string;
  ordinal: number;
  title: string;
  route: string;
  deferred: boolean;
}
export interface V2StepsResponse {
  steps: V2Step[];
}
export interface OperatorInputRequest {
  id: string;
  kind: "secret" | "choice" | "confirm";
  title: string;
  description?: string;
  default?: string;
  options?: string[];
  unblocks?: string[];
  validation?: string;
  required: boolean;
}
export interface OperatorInputQueue {
  version: number;
  updated_at: string;
  requests: OperatorInputRequest[];
}
export type CapabilityState =
  | "discovered"
  | "needs_operator_input"
  | "ready_to_preview"
  | "applying"
  | "verifying"
  | "ready"
  | "retryable_failure"
  | "degraded"
  | "unsupported";
export type CapabilityInputKind =
  "secret" | "path" | "enum" | "boolean" | "duration" | "confirmation";
export interface CapabilityCandidate {
  id: string;
  kind: string;
  label: string;
  location?: string;
  stable_identity?: string;
  device_identity?: string;
  writable: boolean;
  physical_independence?: string;
  status: string;
  risk?: string;
  remediation?: string;
  metadata?: Record<string, string>;
}
export interface CapabilityInput {
  id: string;
  kind: CapabilityInputKind;
  label: string;
  description?: string;
  required: boolean;
  options?: string[];
  default?: string;
  candidates?: CapabilityCandidate[];
  validation?: string;
}
export interface CapabilityDescriptor {
  version: string;
  id: string;
  owner: string;
  title: string;
  description?: string;
  risk?: string;
  inputs?: CapabilityInput[];
  prerequisites?: string[];
  policy: {
    requires_confirmation: boolean;
    idempotent: boolean;
    retryable: boolean;
    protected_roots?: string[];
    remediation?: string;
  };
  evidence: {
    kinds?: string[];
    required_fields?: string[];
    secret_free: boolean;
    freshness?: string;
  };
  remediation?: string;
}
export interface CapabilityEvidence {
  kind: string;
  artifact_identity: string;
  source_generation?: string;
  checksum?: string;
  coverage?: string[];
  observed_at: string;
  verified: boolean;
  remediation?: string;
}
export interface CapabilityStatus {
  descriptor: CapabilityDescriptor;
  state: CapabilityState;
  candidates?: CapabilityCandidate[];
  missing_inputs?: string[];
  evidence?: CapabilityEvidence[];
  remediation?: string;
  updated_at: string;
}
export interface CapabilityStatusResponse {
  capabilities: CapabilityStatus[];
  count: number;
}
export interface CapabilityPreview {
  capability_id: string;
  plan_id: string;
  state: CapabilityState;
  mutations?: Array<{ id: string; summary: string; reversible: boolean }>;
  candidates?: CapabilityCandidate[];
  remediation?: string;
  expires_at?: string;
}
export interface CapabilityResult {
  capability_id: string;
  state: CapabilityState;
  outcome: string;
  retryable: boolean;
  error_code?: string;
  remediation?: string;
  evidence?: CapabilityEvidence[];
  mutations?: Array<{ id: string; summary: string; reversible: boolean }>;
  completed_at?: string;
}
export interface CredentialDiagnosis {
  provider?: {
    backend?: string;
    condition?: string;
    explanation?: string;
    fix?: string;
    write_condition?: string;
    write_fix?: string;
    native_storage_caveat?: string;
  };
}
export interface ReadinessItem {
  name: string;
  category?: "integration" | "system";
  status: "ready" | "degraded" | "missing" | "unsupported" | "deferred";
  detail?: string;
  remediation?: string;
  kind?: "tool" | "safeguard";
  required?: boolean;
}
export interface HostConfigProperty {
  type?: string;
  title?: string;
  description?: string;
  enum?: unknown[];
  default?: unknown;
}
export interface HostConfigSchema {
  type?: string;
  properties?: Record<string, HostConfigProperty>;
}
export interface HostRequirement {
  name: string;
  required: boolean;
  reason: string;
  notes?: string;
  description?: string;
  risk?: "low" | "medium" | "high";
  privilege?: "none" | "user" | "elevated";
  bundling?: "vendorable" | "host-required" | "prohibited";
  platforms?: string[];
  commands?: string[];
  config_schema?: HostConfigSchema;
  config?: Record<string, unknown>;
  status: "required" | "optional" | "opted_in";
}
export interface V2HostRequirementsResponse {
  tools: HostRequirement[];
  safeguards: HostRequirement[];
}

export interface OperatorState {
  version: string;
  updated_at: string;
  core?: { seed: string[]; trusted_base: string[] };
  scenarios?: Record<string, { enabled?: boolean; auto_restart?: boolean }>;
  resources?: Record<string, { enabled?: boolean }>;
  host_tools?: Record<string, { opted_in?: boolean }>;
  host_safeguards?: Record<
    string,
    { opted_in?: boolean; config?: Record<string, unknown> }
  >;
}

export type OperatorStatePatch = Partial<
  Pick<
    OperatorState,
    "core" | "scenarios" | "resources" | "host_tools" | "host_safeguards"
  >
> & {
  trust_posture?: "personal" | "shared" | "hosted";
  active_profile?: string | null;
};
