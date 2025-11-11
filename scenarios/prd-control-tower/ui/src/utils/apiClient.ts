import { buildApiUrl as buildApiBaseUrl, resolveApiBase } from '@vrooli/api-base'

const API_BASE = resolveApiBase({ appendSuffix: true })

export function getApiBase(): string {
  return API_BASE
}

export function buildApiUrl(path: string): string {
  return buildApiBaseUrl(path, { baseUrl: API_BASE })
}
