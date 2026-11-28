import { createClient } from '@connectrpc/connect'
import { createScenarioConnectTransport } from '@vrooli/api-base'
import { resolvePromptManagerConnectBase } from '@/lib/api'
import { useQuery } from '@tanstack/react-query'
import { AgentManagerService } from '@vrooli/proto-types/agent-manager/v1/api/service_pb'
import type { EffortBoard } from '@vrooli/proto-types/agent-manager/v1/domain/effort_pb'

export const EFFORT_PAGE_SIZE = 10
export const LINKED_EFFORT_PAGE_SIZE = 5
const ownerBase = `${resolvePromptManagerConnectBase()}/embedded/agent-manager`
export const effortBoardClient = createClient(AgentManagerService, createScenarioConnectTransport({ baseUrl: ownerBase }))

export interface EffortObservations {
  boards: EffortBoard[]
  unavailable: Array<{ effortRef: string; reason: string }>
}

/** Read-only runtime roster entry for an Agent Manager child run. */
export interface ContractorMember {
  id: string
  runId: string
  displayName: string
  status: string
  role: string
  assignment: string
  effortRef: string
  model?: string
  runner?: string
}

/**
 * Project exact Agent Manager child subjects into Prompt Manager's visual
 * roster. Contractors are deliberately not team-member relations: they are
 * runtime assignments and disappear when the owner no longer reports them.
 */
export function contractorMembersFromObservations(observations: EffortObservations, effortRefByBoard?: string[]): ContractorMember[] {
  const byRun = new Map<string, ContractorMember>()
  observations.boards.forEach((board, boardIndex) => {
    const fallbackEffortRef = effortRefByBoard?.[boardIndex] ?? ''
    board.rows.forEach((row) => {
      const effortRef = row.enrollment?.effortRef ?? fallbackEffortRef
      row.assignments.forEach((assignment) => {
        const subject = assignment.subject
        const runId = subject?.runId?.trim()
        if (!runId || subject?.role === 'orchestrator' || subject?.role === 'supervisor') return
        if (!subject) return
        const role = subject.role?.trim() || 'worker'
        const assignmentLabel = subject.assignment?.trim() || subject.reference?.trim() || role
        byRun.set(runId, {
          id: `contractor:${runId}`,
          runId,
          displayName: assignmentLabel,
          status: assignment.runtimeState?.trim() || row.runtimeState || 'observed',
          role,
          assignment: assignmentLabel,
          effortRef,
          model: assignment.effectiveModel?.trim() || assignment.requestedModel?.trim() || undefined,
          runner: assignment.effectiveRunner?.trim() || assignment.requestedRunner?.trim() || undefined,
        })
      })
    })
  })
  return [...byRun.values()].sort((a, b) => a.displayName.localeCompare(b.displayName) || a.runId.localeCompare(b.runId))
}

export function useTeamContractors(effortRefs?: string[]) {
  const observations = useEffortObservations('', effortRefs?.length ? effortRefs : [])
  const contractors = contractorMembersFromObservations(observations.data ?? { boards: [], unavailable: [] }, effortRefs)
  return { ...observations, contractors }
}

// These are owner reads only. Neither opening this view nor refreshing it
// performs discovery, enrolls work or dispatches a supervisor.
export async function readEffortObservations(pageToken: string, effortRefs?: string[], signal?: AbortSignal): Promise<EffortObservations> {
  if (effortRefs === undefined) {
    const board = await effortBoardClient.getEffortBoard({ pageSize: EFFORT_PAGE_SIZE, pageToken }, { signal, timeoutMs: 30_000 })
    return { boards: [board], unavailable: [] }
  }
  if (effortRefs.length > LINKED_EFFORT_PAGE_SIZE) throw new Error('Linked effort request exceeds the page limit')
  const results = await Promise.allSettled(effortRefs.map(effortRef =>
    effortBoardClient.getEffortBoard({ effortRef, pageSize: 1 }, { signal, timeoutMs: 30_000 }),
  ))
  const observations: EffortObservations = { boards: [], unavailable: [] }
  results.forEach((result, index) => {
    const effortRef = effortRefs[index]
    if (effortRef === undefined) throw new Error('Owner observation lost its requested identity')
    if (result.status === 'fulfilled' && result.value.rows.some(row => row.enrollment?.effortRef === effortRef)) {
      // A malformed owner response must never substitute another effort.
      observations.boards.push({ ...result.value, rows: result.value.rows.filter(row => row.enrollment?.effortRef === effortRef) })
    } else {
      observations.unavailable.push({ effortRef, reason: result.status === 'rejected'
        ? (result.reason instanceof Error ? result.reason.message : 'Owner read failed')
        : 'Exact effort was not returned by the owner' })
    }
  })
  return observations
}

export function useEffortObservations(pageToken: string, effortRefs?: string[]) {
  return useQuery({
    queryKey: ['pm-effort-observations', pageToken, effortRefs ?? null],
    queryFn: ({ signal }) => readEffortObservations(pageToken, effortRefs, signal),
    enabled: effortRefs === undefined || effortRefs.length > 0,
    staleTime: 30_000,
    retry: false,
    refetchOnWindowFocus: false,
  })
}

export function effortOwnerPath(base: string, effortRef: string): string {
  return `${base.replace(/\/$/, '')}/efforts?${new URLSearchParams({ effortRef })}`
}

export function useAgentManagerUrl(enabled = true) {
  return useQuery({
    queryKey: ['pm-agent-manager-external-url'],
    enabled,
    queryFn: async ({ signal }) => {
      const controller = new AbortController()
      const abort = () => controller.abort()
      signal.addEventListener('abort', abort, { once: true })
      if (signal.aborted) controller.abort()
      const timeout = setTimeout(abort, 10_000)
      try {
        const response = await fetch(`${ownerBase}/external-url`, { signal: controller.signal })
        if (!response.ok) throw new Error('Agent Manager link unavailable')
        const data: unknown = await response.json()
        if (!data || typeof data !== 'object' || !('url' in data) || typeof data.url !== 'string') throw new Error('Agent Manager link unavailable')
        const url = new URL(data.url)
        if (!['http:', 'https:'].includes(url.protocol)) throw new Error('Invalid Agent Manager link')
        return url.toString().replace(/\/$/, '')
      } finally {
        clearTimeout(timeout)
        signal.removeEventListener('abort', abort)
      }
    },
    staleTime: 60_000,
    retry: false,
    refetchOnWindowFocus: false,
  })
}
