import { buildApiUrl } from '../utils/apiClient'

const JSON_HEADERS = { 'Content-Type': 'application/json' }

// BacklogFeedback is the per-item feedback contract returned by the API after
// filing (or reading) a backlog item in swarm-manager. No time-based ETA —
// queue_position (items-ahead; null when not pending) is the honest signal.
export interface BacklogFeedback {
  item_id: string
  kind: string
  name: string
  deep_link: string
  status: string
  queue_position?: number | null
  priority: number
  deduped: boolean
}

export interface IssueReportSelectionInput {
  id: string
  title: string
  detail?: string
  category?: string
  severity?: string
  reference?: string
  notes?: string
}

export interface ScenarioIssueReportRequest {
  entity_type: string
  entity_name: string
  source?: string
  title: string
  description: string
  priority?: string
  summary?: string
  tags?: string[]
  selections: IssueReportSelectionInput[]
}

export async function submitIssueReport(
  payload: ScenarioIssueReportRequest,
): Promise<BacklogFeedback> {
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

export async function fetchBacklogItemStatus(kind: string, name: string): Promise<BacklogFeedback> {
  if (!name) {
    throw new Error('name is required')
  }
  const url = buildApiUrl(
    `/issues/status?kind=${encodeURIComponent(kind || 'fix')}&name=${encodeURIComponent(name)}`,
  )
  const response = await fetch(url)
  if (!response.ok) {
    const message = (await response.text()) || `Failed to load status for ${kind}/${name}`
    throw new Error(message)
  }
  return response.json()
}

export interface BulkIssueReportResult {
  request: ScenarioIssueReportRequest
  response?: BacklogFeedback
  error?: string
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
