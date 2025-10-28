import { resolveApiBase } from '@vrooli/api-base'

export const DEFAULT_API_PORT = (import.meta.env.VITE_API_PORT as string | undefined)?.trim() || '16430'
export const DEFAULT_API_TOKEN =
  (import.meta.env.VITE_API_TOKEN as string | undefined)?.trim() || 'math-tools-api-token'
export const DEFAULT_API_BASE_URL = resolveApiBase({
  explicitUrl: (import.meta.env.VITE_API_BASE_URL as string | undefined)?.trim(),
  defaultPort: DEFAULT_API_PORT,
  appendSuffix: false,
})

export interface ApiSettings {
  baseUrl: string
  token: string
}

export interface ApiErrorShape {
  error: string
  success?: boolean
}

export interface CalculationResponse {
  result: unknown
  operation: string
  execution_time_ms: number
  precision_used: number
  algorithm: string
}

export interface StatisticsResponse {
  results: Record<string, unknown>
  data_points: number
}

export interface SolveResponse {
  solutions: number[]
  solution_type: string
  method_used: string
  convergence_info: {
    converged: boolean
    iterations: number
    final_error: number
  }
}

export interface OptimizeResponse {
  optimal_solution: Record<string, number>
  optimal_value: number
  status: string
  iterations: number
  algorithm_used: string
  sensitivity_analysis?: {
    gradient?: Record<string, number>
    hessian_eigenvalues?: number[]
  }
}

export interface ForecastResponse {
  forecast: number[]
  model_metrics: Record<string, number>
  model_parameters: Record<string, unknown>
  confidence_intervals?: {
    lower: number[]
    upper: number[]
  }
}

export interface HealthResponse {
  status: string
  service: string
  timestamp: number
  version: string
  database?: string
}

export interface ModelSummary {
  id: string
  name: string
  model_type: string
  created_at: string
}

export interface ModelDetail extends ModelSummary {
  formula?: string
  parameters?: unknown
}

export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'DELETE'

export interface ApiRequestOptions {
  method?: HttpMethod
  body?: unknown
  headers?: Record<string, string>
  settings: ApiSettings
}

export async function apiRequest<TResponse>(path: string, options: ApiRequestOptions): Promise<TResponse> {
  const { method = 'GET', body, headers, settings } = options
  const normalizedBase = settings.baseUrl.replace(/\/+$/u, '')
  const normalizedPath = path.startsWith('http') ? path : `${normalizedBase}${path.startsWith('/') ? '' : '/'}${path}`

  const requestInit: RequestInit = {
    method,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${settings.token}`,
      ...headers,
    },
    body: body ? JSON.stringify(body) : undefined,
  }

  const response = await fetch(normalizedPath, requestInit)

  const contentType = response.headers.get('content-type') || ''
  const isJson = contentType.includes('application/json')
  const payload = isJson ? await response.json() : await response.text()

  if (!response.ok) {
    const message = typeof payload === 'string' ? payload : (payload?.error as string) || 'Unexpected API error'
    throw new Error(message)
  }

  return payload as TResponse
}

export function buildApiSettings(partial?: Partial<ApiSettings>): ApiSettings {
  return {
    baseUrl: partial?.baseUrl?.trim() || DEFAULT_API_BASE_URL,
    token: partial?.token?.trim() || DEFAULT_API_TOKEN,
  }
}

export async function fetchHealth(settings: ApiSettings): Promise<HealthResponse> {
  return apiRequest<HealthResponse>('/api/health', { settings })
}

export async function performCalculation(
  payload: Record<string, unknown>,
  settings: ApiSettings,
): Promise<CalculationResponse> {
  return apiRequest<CalculationResponse>('/api/v1/math/calculate', { method: 'POST', body: payload, settings })
}

export async function runStatistics(
  payload: Record<string, unknown>,
  settings: ApiSettings,
): Promise<StatisticsResponse> {
  return apiRequest<StatisticsResponse>('/api/v1/math/statistics', {
    method: 'POST',
    body: payload,
    settings,
  })
}

export async function solveEquation(
  payload: Record<string, unknown>,
  settings: ApiSettings,
): Promise<SolveResponse> {
  return apiRequest<SolveResponse>('/api/v1/math/solve', {
    method: 'POST',
    body: payload,
    settings,
  })
}

export async function optimize(
  payload: Record<string, unknown>,
  settings: ApiSettings,
): Promise<OptimizeResponse> {
  return apiRequest<OptimizeResponse>('/api/v1/math/optimize', {
    method: 'POST',
    body: payload,
    settings,
  })
}

export async function forecast(
  payload: Record<string, unknown>,
  settings: ApiSettings,
): Promise<ForecastResponse> {
  return apiRequest<ForecastResponse>('/api/v1/math/forecast', {
    method: 'POST',
    body: payload,
    settings,
  })
}

export async function listModels(settings: ApiSettings): Promise<ModelSummary[]> {
  return apiRequest<ModelSummary[]>('/api/v1/models', { settings })
}

export async function createModel(
  payload: Record<string, unknown>,
  settings: ApiSettings,
): Promise<{ id: string; created_at: string }> {
  return apiRequest<{ id: string; created_at: string }>('/api/v1/models', {
    method: 'POST',
    body: payload,
    settings,
  })
}

export async function updateModel(
  id: string,
  payload: Record<string, unknown>,
  settings: ApiSettings,
): Promise<{ updated: boolean }> {
  return apiRequest<{ updated: boolean }>(`/api/v1/models/${id}`, {
    method: 'PUT',
    body: payload,
    settings,
  })
}

export async function deleteModel(id: string, settings: ApiSettings): Promise<{ deleted: boolean; id: string }> {
  return apiRequest<{ deleted: boolean; id: string }>(`/api/v1/models/${id}`, {
    method: 'DELETE',
    settings,
  })
}
