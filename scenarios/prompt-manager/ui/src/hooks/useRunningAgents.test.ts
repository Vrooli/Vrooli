import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { listRunningAgents } from '@/services/heartbeatService'
import { useRunningAgents, resetRunningAgentsPollingForTests } from './useRunningAgents'

vi.mock('@/services/heartbeatService', () => ({
  listRunningAgents: vi.fn(),
  stopRunningAgent: vi.fn(),
}))

describe('useRunningAgents', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    resetRunningAgentsPollingForTests()
    Object.defineProperty(document, 'hidden', {
      configurable: true,
      value: false,
    })
  })

  afterEach(() => {
    resetRunningAgentsPollingForTests()
    Object.defineProperty(document, 'hidden', {
      configurable: true,
      value: false,
    })
  })

  it('shares a single network poll across multiple consumers', async () => {
    vi.mocked(listRunningAgents).mockResolvedValue({
      count: 1,
      agents: [{
        teamId: 'team-1',
        teamName: 'Team One',
        agentId: 'agent-1',
        agentName: 'Agent One',
        runId: 'run-1',
        startedAt: '2026-02-17T10:00:00Z',
        duration: '3m',
      }],
    })

    const hookA = renderHook(() => useRunningAgents())
    const hookB = renderHook(() => useRunningAgents())

    await waitFor(() => {
      expect(hookA.result.current.count).toBe(1)
      expect(hookB.result.current.count).toBe(1)
    })

    expect(listRunningAgents).toHaveBeenCalledTimes(1)
  })

  it('does not poll when disabled', async () => {
    renderHook(() => useRunningAgents({ enabled: false }))

    await waitFor(() => {
      expect(listRunningAgents).not.toHaveBeenCalled()
    })
  })
})
