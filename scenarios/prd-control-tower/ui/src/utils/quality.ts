import { buildApiUrl } from './apiClient'
import type {
  QualityScanEntity,
  QualityScanResponse,
  ScenarioQualityReport,
  QualitySummary,
} from '../types'

export async function fetchQualityReport(entityType: string, entityName: string, options?: { useCache?: boolean }): Promise<ScenarioQualityReport> {
  const useCache = options?.useCache ?? false
  const response = await fetch(buildApiUrl(`/quality/${entityType}/${entityName}?use_cache=${useCache ? 'true' : 'false'}`))
  if (!response.ok) {
    const message = await response.text()
    throw new Error(message || `Failed to load quality report for ${entityType}/${entityName}`)
  }
  return response.json()
}

export async function runQualityScan(entities: QualityScanEntity[], useCache = false): Promise<QualityScanResponse> {
  const response = await fetch(buildApiUrl('/quality/scan'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ entities, use_cache: useCache }),
  })

  if (!response.ok) {
    const message = await response.text()
    throw new Error(message || 'Quality scan failed')
  }

  return response.json()
}

export async function fetchQualitySummary(useCache = true): Promise<QualitySummary> {
  const response = await fetch(buildApiUrl(`/quality/summary?use_cache=${useCache ? 'true' : 'false'}`))
  if (!response.ok) {
    const message = await response.text()
    throw new Error(message || 'Failed to load quality summary')
  }
  return response.json()
}
