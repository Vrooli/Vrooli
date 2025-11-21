import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

declare global {
  interface Window {
    __SECRETS_MANAGER_CONFIG__?: {
      apiBaseUrl?: string;
    };
  }
}

let API_BASE_URL: string | null = null;

const getApiBaseUrl = () => {
  if (API_BASE_URL === null) {
    const explicitUrl = typeof window !== "undefined" ? window.__SECRETS_MANAGER_CONFIG__?.apiBaseUrl : undefined;
    API_BASE_URL = explicitUrl && explicitUrl.trim().length > 0 ? explicitUrl : resolveApiBase({ appendSuffix: true });
  }

  return API_BASE_URL;
};

const jsonFetch = async <T>(path: string, init?: RequestInit) => {
  const response = await fetch(
    buildApiUrl(path, {
      baseUrl: getApiBaseUrl()
    }),
    {
      headers: {
        "Content-Type": "application/json"
      },
      cache: "no-store",
      ...init
    }
  );

  if (!response.ok) {
    throw new Error(`Request failed (${response.status}): ${response.statusText}`);
  }

  return (await response.json()) as T;
};

export interface HealthResponse {
  status: string;
  service: string;
  version?: string;
  timestamp: string;
  readiness?: boolean;
  status_notes?: string[];
  dependencies?: {
    database?: {
      connected: boolean;
      error?: {
        category: string;
        code: string;
        message: string;
        retryable: boolean;
      };
      latency_ms: number;
    };
    [key: string]: any;
  };
}

export interface VaultMissingSecret {
  resource_name: string;
  secret_name: string;
  secret_path: string;
  required: boolean;
  description: string;
}

export interface VaultResourceStatus {
  resource_name: string;
  secrets_total: number;
  secrets_found: number;
  secrets_missing: number;
  secrets_optional: number;
  health_status: string;
  last_checked: string;
}

export interface VaultSecretsStatus {
  total_resources: number;
  configured_resources: number;
  missing_secrets: VaultMissingSecret[];
  resource_statuses: VaultResourceStatus[];
  last_updated: string;
}

export interface ComplianceResponse {
  overall_score: number;
  vault_secrets_health: number;
  vulnerability_summary: Record<string, number>;
  remediation_progress: {
    configured_components: number;
    critical_issues: number;
    high_issues: number;
    medium_issues: number;
    low_issues: number;
    security_score: number;
    vault_secrets_health: number;
    overall_compliance: number;
  };
  total_resources: number;
  configured_resources: number;
  configured_components: number;
  total_components: number;
  total_vulnerabilities: number;
  components_summary?: {
    resources_scanned: number;
    scenarios_scanned: number;
    total_components: number;
    configured_count: number;
  };
  last_updated: string;
}

export interface SecurityVulnerability {
  id: string;
  component_type: string;
  component_name: string;
  file_path: string;
  line_number: number;
  severity: "critical" | "high" | "medium" | "low";
  type: string;
  title: string;
  description: string;
  recommendation: string;
  can_auto_fix: boolean;
  discovered_at: string;
  status?: string;
  fingerprint?: string;
  last_observed_at?: string;
}

export interface VulnerabilityResponse {
  vulnerabilities: SecurityVulnerability[];
  total_count: number;
  scan_id: string;
  scan_duration: number;
  risk_score: number;
  recommendations?: Array<{ vulnerability_type: string; description: string; priority: string }>;
}

export interface JourneyCard {
  id: string;
  title: string;
  description: string;
  status: string;
  cta_label: string;
  cta_action: string;
  primers: string[];
  badge: string;
}

export interface TierReadiness {
  tier: string;
  label: string;
  strategized: number;
  total: number;
  ready_percent: number;
  blocking_secret_sample: string[];
}

export interface ResourceSecretInsight {
  secret_key: string;
  secret_type: string;
  classification: string;
  required: boolean;
  tier_strategies: Record<string, string>;
}

export interface ResourceInsight {
  resource_name: string;
  total_secrets: number;
  valid_secrets: number;
  missing_secrets: number;
  invalid_secrets: number;
  last_validation?: string;
  secrets: ResourceSecretInsight[];
}

export interface OrientationSummary {
  hero_stats: {
    vault_configured: number;
    vault_total: number;
    missing_secrets: number;
    risk_score: number;
    overall_score: number;
    last_scan: string;
    readiness_label: string;
    confidence: number;
  };
  journeys: JourneyCard[];
  tier_readiness: TierReadiness[];
  resource_insights: ResourceInsight[];
  vulnerability_insights: { severity: string; count: number; message: string }[];
  updated_at: string;
}

export interface ResourceSecretDetail {
  id: string;
  secret_key: string;
  secret_type: string;
  description: string;
  classification: string;
  required: boolean;
  owner_team: string;
  owner_contact: string;
  tier_strategies: Record<string, string>;
  validation_state: string;
  last_validated?: string;
}

export interface ResourceDetail {
  resource_name: string;
  valid_secrets: number;
  missing_secrets: number;
  total_secrets: number;
  last_validation?: string;
  secrets: ResourceSecretDetail[];
  open_vulnerabilities: SecurityVulnerability[];
}

export interface DeploymentManifestSecret {
  resource_name: string;
  secret_key: string;
  secret_type: string;
  required: boolean;
  classification: string;
  description?: string;
  owner_team?: string;
  owner_contact?: string;
  handling_strategy: string;
  fallback_strategy?: string;
  requires_user_input: boolean;
  prompt?: { label?: string; description?: string };
  generator_template?: Record<string, unknown> | null;
  bundle_hints?: Record<string, unknown> | null;
  tier_strategies?: Record<string, string>;
}

export interface DeploymentManifestSummary {
  total_secrets: number;
  strategized_secrets: number;
  requires_action: number;
  blocking_secrets: string[];
  classification_weights: Record<string, number>;
  strategy_breakdown: Record<string, number>;
  scope_readiness: Record<string, string>;
}

export interface DependencyRequirementSummary {
  ram_mb?: number;
  disk_mb?: number;
  cpu_cores?: number;
  network?: string;
  source?: string;
  confidence?: string;
}

export interface DependencyTierSupport {
  supported: boolean;
  fitness_score: number;
  notes?: string;
  reason?: string;
  alternatives?: string[];
}

export interface DependencyInsight {
  name: string;
  kind: string;
  resource_type?: string;
  source?: string;
  alternatives?: string[];
  requirements?: DependencyRequirementSummary;
  tier_support?: Record<string, DependencyTierSupport>;
}

export interface TierAggregateView {
  fitness_score: number;
  dependency_count?: number;
  blocking_dependencies?: string[];
  estimated_requirements?: DependencyRequirementSummary;
}

export interface DeploymentManifestResponse {
  scenario: string;
  tier: string;
  generated_at: string;
  resources: string[];
  secrets: DeploymentManifestSecret[];
  summary: DeploymentManifestSummary;
  analyzer_generated_at?: string;
  dependencies?: DependencyInsight[];
  tier_aggregates?: Record<string, TierAggregateView>;
}

export interface DeploymentManifestRequest {
  scenario: string;
  tier: string;
  resources?: string[];
  include_optional?: boolean;
}

export const fetchHealth = () => jsonFetch<HealthResponse>("/health");

export const fetchVaultStatus = (resource?: string) => {
  const search = resource ? `?resource=${encodeURIComponent(resource)}` : "";
  return jsonFetch<VaultSecretsStatus>(`/vault/secrets/status${search}`);
};

export const fetchCompliance = () => jsonFetch<ComplianceResponse>("/security/compliance");

export interface VulnerabilityFilters {
  component?: string;
  componentType?: string;
  severity?: string;
}

export const fetchVulnerabilities = (filters: VulnerabilityFilters) => {
  const params = new URLSearchParams();
  if (filters.component) {
    params.set("component", filters.component);
  }
  if (filters.componentType) {
    params.set("component_type", filters.componentType);
  }
  if (filters.severity) {
    params.set("severity", filters.severity);
  }

  const suffix = params.toString() ? `?${params.toString()}` : "";
  return jsonFetch<VulnerabilityResponse>(`/vulnerabilities${suffix}`);
};

export const fetchOrientationSummary = () => jsonFetch<OrientationSummary>("/orientation/summary");

export const generateDeploymentManifest = (payload: DeploymentManifestRequest) =>
  jsonFetch<DeploymentManifestResponse>("/deployment/secrets", {
    method: "POST",
    body: JSON.stringify(payload)
  });

export const fetchResourceDetail = (resource: string) => jsonFetch<ResourceDetail>(`/resources/${resource}`);

export interface UpdateResourceSecretPayload {
  classification?: string;
  description?: string;
  required?: boolean;
  owner_team?: string;
  owner_contact?: string;
}

export const updateResourceSecret = (resource: string, secret: string, payload: UpdateResourceSecretPayload) =>
  jsonFetch<ResourceSecretDetail>(`/resources/${resource}/secrets/${secret}`, {
    method: "PATCH",
    body: JSON.stringify(payload)
  });

export interface UpdateSecretStrategyPayload {
  tier: string;
  handling_strategy: string;
  fallback_strategy?: string;
  requires_user_input?: boolean;
  prompt_label?: string;
  prompt_description?: string;
  generator_template?: Record<string, unknown>;
  bundle_hints?: Record<string, unknown>;
}

export const updateSecretStrategy = (resource: string, secret: string, payload: UpdateSecretStrategyPayload) =>
  jsonFetch<ResourceSecretDetail>(`/resources/${resource}/secrets/${secret}/strategy`, {
    method: "POST",
    body: JSON.stringify(payload)
  });

export interface UpdateVulnerabilityStatusPayload {
  status: string;
  assigned_to?: string;
}

export const updateVulnerabilityStatus = (id: string, payload: UpdateVulnerabilityStatusPayload) =>
  jsonFetch<{ id: string; status: string }>(`/vulnerabilities/${id}/status`, {
    method: "POST",
    body: JSON.stringify(payload)
  });
