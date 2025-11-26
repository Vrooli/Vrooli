import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

// Factory-scope type definitions

export interface Template {
  id: string;
  name: string;
  description: string;
  version: string;
  metadata?: Record<string, unknown>;
  sections?: Record<string, unknown>;
  metrics_hooks?: Array<Record<string, unknown>>;
  customization_schema?: Record<string, unknown>;
  frontend_aesthetics?: Record<string, unknown>;
}

export interface GenerationResult {
  scenario_id: string;
  name: string;
  template: string;
  path: string;
  status: string;
  plan?: { paths: string[] };
  next_steps?: string[];
}

export interface CustomizeResult {
  status: string;
  issue_id?: string;
  tracker_url?: string;
  agent?: string;
  run_id?: string;
  message?: string;
}

export interface GeneratedScenario {
  scenario_id: string;
  name: string;
  template_id?: string;
  template_version?: string;
  path: string;
  status: string;
  generated_at?: string;
}

export interface Persona {
  id: string;
  name: string;
  description: string;
  prompt_template: string;
  use_cases?: string[];
}

export interface PreviewLinks {
  scenario_id: string;
  ui_port?: number;
  links: {
    public?: string;
    admin?: string;
    admin_login?: string;
    health?: string;
  };
}

// Helper for API calls
async function apiCall<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const url = buildApiUrl(endpoint, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
    credentials: 'include',
    ...options,
  });

  if (!res.ok) {
    const errorText = await res.text();
    throw new Error(`API call failed (${res.status}): ${errorText}`);
  }

  return res.json() as Promise<T>;
}

// Health check
export async function fetchHealth() {
  return apiCall<{ status: string; service: string; timestamp: string }>('/health');
}

// Templates (factory scope)
export async function listTemplates() {
  return apiCall<Template[]>('/templates');
}

export async function getTemplate(id: string) {
  return apiCall<Template>(`/templates/${id}`);
}

export async function listGeneratedScenarios() {
  return apiCall<GeneratedScenario[]>('/generated');
}

export async function generateScenario(templateId: string, name: string, slug: string, options?: { dry_run?: boolean }) {
  const payload = {
    template_id: templateId,
    name,
    slug,
    options: options ?? {},
  };
  return apiCall<GenerationResult>('/generate', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function customizeScenario(scenarioId: string, brief: string, assets?: string[], preview?: boolean) {
  const payload = {
    scenario_id: scenarioId,
    brief,
    assets: assets ?? [],
    preview: preview ?? false,
  };
  return apiCall<CustomizeResult>('/customize', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

// Personas (agent profiles)
export async function listPersonas() {
  return apiCall<Persona[]>('/personas');
}

export async function getPersona(id: string) {
  return apiCall<Persona>(`/personas/${id}`);
}

// Preview links
export async function getPreviewLinks(scenarioId: string) {
  return apiCall<PreviewLinks>(`/preview/${scenarioId}`);
}

// Lifecycle management types
export interface LifecycleResponse {
  success: boolean;
  message: string;
  scenario_id?: string;
  output?: string;
}

export interface ScenarioStatus {
  success: boolean;
  scenario_id: string;
  running: boolean;
  status_text: string;
}

export interface ScenarioLogs {
  success: boolean;
  scenario_id: string;
  logs: string;
}

// Lifecycle management
export async function startScenario(scenarioId: string) {
  return apiCall<LifecycleResponse>(`/lifecycle/${scenarioId}/start`, {
    method: 'POST',
  });
}

export async function stopScenario(scenarioId: string) {
  return apiCall<LifecycleResponse>(`/lifecycle/${scenarioId}/stop`, {
    method: 'POST',
  });
}

export async function restartScenario(scenarioId: string) {
  return apiCall<LifecycleResponse>(`/lifecycle/${scenarioId}/restart`, {
    method: 'POST',
  });
}

export async function getScenarioStatus(scenarioId: string) {
  return apiCall<ScenarioStatus>(`/lifecycle/${scenarioId}/status`);
}

export async function getScenarioLogs(scenarioId: string, tail = 50) {
  return apiCall<ScenarioLogs>(`/lifecycle/${scenarioId}/logs?tail=${tail}`);
}

export interface PromoteResponse {
  success: boolean;
  message: string;
  scenario_id: string;
  production_path?: string;
}

export async function promoteScenario(scenarioId: string) {
  return apiCall<PromoteResponse>(`/lifecycle/${scenarioId}/promote`, {
    method: 'POST',
  });
}
