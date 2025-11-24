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
