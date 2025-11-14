import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });
const buildUrl = (path: string) => buildApiUrl(path, { baseUrl: API_BASE });

export interface HealthResponse {
  status: string;
  service: string;
  timestamp: string;
}

export interface SystemStatus {
  service: string;
  version: string;
  uptime: number;
  statistics: {
    total_builds: number;
    active_builds: number;
    completed_builds: number;
    failed_builds: number;
  };
}

export interface TemplateInfo {
  name: string;
  description: string;
  type: string;
  framework: string;
  use_cases: string[];
  features: string[];
  complexity: string;
}

export interface DesktopConfig {
  app_name: string;
  app_display_name: string;
  app_description: string;
  version: string;
  author: string;
  license: string;
  app_id: string;
  server_type: string;
  server_port: number;
  server_path: string;
  api_endpoint: string;
  framework: string;
  template_type: string;
  platforms: string[];
  output_path: string;
  features: Record<string, boolean>;
  window: Record<string, unknown>;
}

export interface BuildStatus {
  build_id: string;
  scenario_name: string;
  status: "building" | "ready" | "failed";
  framework: string;
  template_type: string;
  platforms: string[];
  output_path: string;
  created_at: string;
  completed_at?: string;
  error_log?: string[];
  build_log?: string[];
}

export async function fetchHealth(): Promise<HealthResponse> {
  const response = await fetch(buildUrl("/health"));
  if (!response.ok) {
    throw new Error(`Health check failed: ${response.statusText}`);
  }
  return response.json();
}

export async function fetchSystemStatus(): Promise<SystemStatus> {
  const response = await fetch(buildUrl("/status"));
  if (!response.ok) {
    throw new Error(`Failed to fetch system status: ${response.statusText}`);
  }
  return response.json();
}

export async function fetchTemplates(): Promise<{ templates: TemplateInfo[] }> {
  const response = await fetch(buildUrl("/templates"));
  if (!response.ok) {
    throw new Error(`Failed to fetch templates: ${response.statusText}`);
  }
  return response.json();
}

export async function generateDesktop(config: DesktopConfig): Promise<{
  build_id: string;
  status: string;
  desktop_path: string;
  install_instructions: string;
}> {
  const response = await fetch(buildUrl("/desktop/generate"), {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify(config)
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: response.statusText }));
    throw new Error(error.message || "Failed to generate desktop application");
  }

  return response.json();
}

export async function fetchBuildStatus(buildId: string): Promise<BuildStatus> {
  const response = await fetch(buildUrl(`/desktop/status/${buildId}`));
  if (!response.ok) {
    throw new Error(`Failed to fetch build status: ${response.statusText}`);
  }
  return response.json();
}
