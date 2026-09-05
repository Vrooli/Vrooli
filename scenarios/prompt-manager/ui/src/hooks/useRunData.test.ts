import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { areRunsEqual, useRunData } from './useRunData'
import type { RunDetails } from '@/services/heartbeatService'
import { listHeartbeatAttempts, listRuns } from '@/services/heartbeatService'

vi.mock('@/services/heartbeatService', () => ({
  listHeartbeatAttempts: vi.fn(),
  listRuns: vi.fn(),
}))

function makeRun(overrides: Partial<RunDetails> = {}): RunDetails {
  return {
    id: 'run-1',
    taskId: 'task-1',
    status: 'running',
    startedAt: '2026-02-17T09:00:00Z',
    ...overrides,
  }
}

describe('useRunData', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.defineProperty(document, 'hidden', {
      configurable: true,
      value: false,
    })
    vi.mocked(listHeartbeatAttempts).mockResolvedValue({ attempts: [], total: 0, hasMore: false })
  })

  afterEach(() => {
    Object.defineProperty(document, 'hidden', {
      configurable: true,
      value: false,
    })
  })

  it('areRunsEqual returns true for identical runs', () => {
    const left = [makeRun()]
    const right = [makeRun()]
    expect(areRunsEqual(left, right)).toBe(true)
  })

  it('areRunsEqual returns false when status changes', () => {
    const left = [makeRun({ status: 'running' })]
    const right = [makeRun({ status: 'failed' })]
    expect(areRunsEqual(left, right)).toBe(false)
  })

  it('skips fetch when document is hidden', async () => {
    Object.defineProperty(document, 'hidden', {
      configurable: true,
      value: true,
    })

    vi.mocked(listRuns).mockResolvedValue({ runs: [], total: 0, hasMore: false })

    renderHook(() => useRunData())

    await waitFor(() => {
      expect(listRuns).not.toHaveBeenCalled()
    })
  })

  it('fetches runs on mount when visible', async () => {
    vi.mocked(listRuns).mockResolvedValue({
      runs: [makeRun()],
      total: 1,
      hasMore: false,
    })

    const { result } = renderHook(() => useRunData())

    await waitFor(() => {
      expect(listRuns).toHaveBeenCalledTimes(1)
      expect(result.current.loading).toBe(false)
      expect(result.current.runs).toHaveLength(1)
    })
  })

  it('scopes run fetches to prompt-manager profile by default', async () => {
    vi.mocked(listRuns).mockResolvedValue({
      runs: [],
      total: 0,
      hasMore: false,
    })

    renderHook(() => useRunData())

    await waitFor(() => {
      expect(listRuns).toHaveBeenCalledWith({
        status: undefined,
        tagPrefix: undefined,
        profileKey: 'prompt-manager-heartbeat',
        limit: 100,
      })
      expect(listHeartbeatAttempts).toHaveBeenCalledWith({
        status: undefined,
        profileKey: 'prompt-manager-heartbeat',
        limit: 100,
      })
    })
  })

  it('includes heartbeat attempts that failed before run creation', async () => {
    vi.mocked(listRuns).mockResolvedValue({
      runs: [],
      total: 0,
      hasMore: false,
    })
    vi.mocked(listHeartbeatAttempts).mockResolvedValue({
      attempts: [{
        id: 'attempt-1',
        teamId: 'scenario-qa',
        agentId: 'bug-investigator',
        profileKey: 'prompt-manager-heartbeat',
        status: 'failed',
        phase: 'creating_run',
        startedAt: '2026-05-06T12:00:00Z',
        endedAt: '2026-05-06T12:00:01Z',
        errorCategory: 'contract_validation',
        error: 'creating run: validation error',
        recovery: 'fix_integration_contract',
      }],
      total: 1,
      hasMore: false,
    })

    const { result } = renderHook(() => useRunData())

    await waitFor(() => {
      expect(result.current.runs).toHaveLength(1)
      expect(result.current.runs[0]?.id).toBe('attempt:attempt-1')
      expect(result.current.runs[0]?.source).toBe('heartbeat-attempt')
      expect(result.current.runs[0]?.errorCategory).toBe('contract_validation')
    })
  })
})
