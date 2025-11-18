import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

let API_BASE_URL: string | null = null;

const getApiBaseUrl = () => {
  if (API_BASE_URL === null) {
    API_BASE_URL = resolveApiBase({
      appendSuffix: true,
      explicitUrl: import.meta.env.VITE_API_BASE_URL
    });
  }

  return API_BASE_URL;
};

const jsonFetch = async <T>(path: string, init?: RequestInit) => {
  const response = await fetch(buildApiUrl(path, { baseUrl: getApiBaseUrl() }), {
    headers: {
      "Content-Type": "application/json"
    },
    cache: "no-store",
    ...init
  });

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
  dependencies?: Record<string, string>;
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
}

export interface VulnerabilityResponse {
  vulnerabilities: SecurityVulnerability[];
  total_count: number;
  scan_id: string;
  scan_duration: number;
  risk_score: number;
  recommendations?: Array<{ vulnerability_type: string; description: string; priority: string }>;
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
