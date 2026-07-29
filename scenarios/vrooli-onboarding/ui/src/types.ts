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

export interface CredentialReadiness {
  resource: string;
  logical_id: string;
  field: string;
  label: string;
  required: boolean;
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
}
export interface ReadinessItem { name: string; status: "ready" | "degraded" | "missing" | "unsupported" | "deferred"; detail?: string; remediation?: string; kind?: "tool" | "safeguard"; required?: boolean; }
export interface HostRequirement { name: string; required: boolean; reason: string; notes?: string; description?: string; risk?: "low" | "medium" | "high"; privilege?: "none" | "user" | "elevated"; bundling?: "vendorable" | "host-required" | "prohibited"; platforms?: string[]; commands?: string[]; status: "required" | "optional" | "opted_in"; }
export interface V2HostRequirementsResponse { tools: HostRequirement[]; safeguards: HostRequirement[]; }

export interface OperatorState {
  version: string;
  updated_at: string;
  scenarios?: Record<string, { enabled?: boolean; auto_restart?: boolean }>;
  resources?: Record<string, { enabled?: boolean }>;
  host_tools?: Record<string, { opted_in?: boolean }>;
  host_safeguards?: Record<string, { opted_in?: boolean }>;
}

/** Labels for each wizard step, in order. */
export const STEP_LABELS = ["Welcome", "Scenarios", "Resources", "Credentials", "Integrations", "Host", "Operating Mode", "Validation"] as const;

/** Total number of wizard steps. */
export const TOTAL_STEPS = STEP_LABELS.length;
