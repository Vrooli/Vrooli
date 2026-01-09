import type {
  BulkIssueReportResult,
  ScenarioIssuesSummary,
  ScenarioIssueReportRequest,
  ScenarioIssueReportResponse,
} from '../types'
import { buildApiUrl } from '../utils/apiClient'

const JSON_HEADERS = { 'Content-Type': 'application/json' }

export async function fetchScenarioIssuesStatus(
  entityType: string,
  entityName: string,
  options?: { useCache?: boolean },
): Promise<ScenarioIssuesSummary> {
  if (!entityType || !entityName) {
    throw new Error('entityType and entityName are required')
  }

  const useCache = options?.useCache ?? true
  const url = buildApiUrl(
    `/issues/status?entity_type=${encodeURIComponent(entityType)}&entity_name=${encodeURIComponent(entityName)}&use_cache=${useCache ? 'true' : 'false'}`,
  )

  const response = await fetch(url)
  if (!response.ok) {
    const message = (await response.text()) || `Failed to load issues for ${entityType}/${entityName}`
    throw new Error(message)
  }

  return response.json()
}

export async function submitIssueReport(
  payload: ScenarioIssueReportRequest,
): Promise<ScenarioIssueReportResponse> {
  const response = await fetch(buildApiUrl('/issues/report'), {
    method: 'POST',
    headers: JSON_HEADERS,
    body: JSON.stringify(payload),
  })

  if (!response.ok) {
    const message = (await response.text()) || 'Failed to submit issue report'
    throw new Error(message)
  }

  return response.json()
}

export type BulkIssueReportProgressHandler = (result: BulkIssueReportResult, index: number) => void

export async function bulkSubmitIssueReports(
  requests: ScenarioIssueReportRequest[],
  onProgress?: BulkIssueReportProgressHandler,
): Promise<BulkIssueReportResult[]> {
  const results: BulkIssueReportResult[] = []

  for (let index = 0; index < requests.length; index += 1) {
    const request = requests[index]
    try {
      const response = await submitIssueReport(request)
      const entry: BulkIssueReportResult = { request, response }
      results.push(entry)
      onProgress?.(entry, index)
    } catch (error) {
      const entry: BulkIssueReportResult = {
        request,
        error: error instanceof Error ? error.message : 'Unknown error',
      }
      results.push(entry)
      onProgress?.(entry, index)
    }
  }

  return results
}
