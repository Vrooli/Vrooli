export interface ApiError extends Error {
  status?: number;
  body?: unknown;
}

function normalizeBase(base: string | undefined | null): string {
  if (!base) {
    return '';
  }
  const trimmed = base.trim().replace(/\/$/u, '');
  return trimmed === '/' ? '' : trimmed;
}

function guessApiBase(): string {
  if (typeof window === 'undefined') {
    return '';
  }

  const envBase = normalizeBase((import.meta.env?.VITE_API_BASE as string | undefined) ?? undefined);
  if (envBase) {
    return envBase;
  }

  const configBase = normalizeBase(window.DATA_STRUCTURER_CONFIG?.apiBase);
  if (configBase) {
    return configBase;
  }

  const currentOrigin = window.location.origin;

  if (currentOrigin.includes(':5173')) {
    const port = (import.meta.env?.VITE_API_PORT as string | undefined) ?? '15770';
    return `http://${window.location.hostname}:${port}`;
  }

  return normalizeBase(currentOrigin);
}

const API_BASE = guessApiBase();

async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const url = `${API_BASE}${path.startsWith('/') ? path : `/${path}`}`;
  const headers = new Headers(init.headers);
  if (!headers.has('Content-Type') && init.body) {
    headers.set('Content-Type', 'application/json');
  }

  try {
    const response = await fetch(url, { ...init, headers });
    const contentType = response.headers.get('content-type');
    const isJson = contentType?.includes('application/json');
    const body = isJson ? await response.json() : await response.text();

    if (!response.ok) {
      const error: ApiError = new Error(
        isJson && body && typeof body === 'object' && 'error' in (body as Record<string, unknown>)
          ? String((body as Record<string, unknown>).error)
          : `Request failed with status ${response.status}`,
      );
      error.status = response.status;
      error.body = body;
      throw error;
    }

    return body as T;
  } catch (error) {
    console.error('[data-structurer] API request failed', error);
    throw error;
  }
}

export const apiClient = {
  baseUrl: API_BASE,
  health: () => apiFetch<HealthResponse>('/health'),
  listSchemas: () => apiFetch<SchemasResponse>('/api/v1/schemas'),
  listJobs: () => apiFetch<JobListResponse>('/api/v1/jobs'),
  schemaDetails: (id: string) => apiFetch<SchemaDetails>(`/api/v1/schemas/${id}`),
  processedData: (schemaId: string, limit = 10) =>
    apiFetch<ProcessedDataResponse>(`/api/v1/data/${schemaId}?limit=${limit}`),
  createSchema: (payload: CreateSchemaPayload) =>
    apiFetch<SchemaCreatedResponse>('/api/v1/schemas', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),
  processData: (payload: ProcessDataPayload) =>
    apiFetch<ProcessingResponse>('/api/v1/process', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),
};

export interface HealthResponse {
  status: string;
  readiness: boolean;
  service: string;
  dependencies?: Record<string, { status?: string; latency_ms?: number }>;
}

export interface SchemaSummary {
  id: string;
  name: string;
  description?: string;
  version: number;
  is_active: boolean;
  usage_count?: number;
  avg_confidence?: number;
  created_at?: string;
  updated_at?: string;
}

export interface SchemasResponse {
  count: number;
  schemas: SchemaSummary[];
}

export interface SchemaDetails extends SchemaSummary {
  schema_definition?: Record<string, unknown>;
  example_data?: Record<string, unknown>;
}

export interface ProcessedRecord {
  id: string;
  created_at?: string;
  processed_at?: string;
  confidence_score?: number | null;
  processing_status?: string;
  structured_data?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
  error_message?: string | null;
}

export interface ProcessedDataResponse {
  data: ProcessedRecord[];
  count: number;
}

export interface JobSummary {
  id: string;
  schema_id: string;
  status: string;
  created_at?: string;
  updated_at?: string;
  progress?: number;
  error?: string | null;
}

export interface JobListResponse {
  jobs: JobSummary[];
}

export interface CreateSchemaPayload {
  name: string;
  description?: string;
  schema_definition: Record<string, unknown>;
  example_data?: Record<string, unknown>;
}

export interface SchemaCreatedResponse {
  id: string;
  name: string;
  status: string;
  created_at?: string;
}

export type InputType = 'text' | 'file' | 'url';

export interface ProcessDataPayload {
  schema_id: string;
  input_type: InputType;
  input_data: string;
  batch_mode?: boolean;
  batch_items?: string[];
}

export interface ProcessingResponse {
  processing_id?: string;
  status: 'queued' | 'running' | 'completed' | 'failed';
  structured_data?: Record<string, unknown>;
  confidence_score?: number;
  errors?: string[];
}
