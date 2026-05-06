import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { RunListPanel } from './RunListPanel'
import { listHeartbeatAttempts, listRuns } from '@/services/heartbeatService'

vi.mock('@/services/heartbeatService', () => ({
  listHeartbeatAttempts: vi.fn(),
  listRuns: vi.fn(),
}))

describe('RunListPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.defineProperty(document, 'hidden', {
      configurable: true,
      value: false,
    })
    vi.mocked(listRuns).mockResolvedValue({
      runs: [],
      total: 0,
      hasMore: false,
    })
    vi.mocked(listHeartbeatAttempts).mockResolvedValue({
      attempts: [],
      total: 0,
      hasMore: false,
    })
  })

  afterEach(() => {
    Object.defineProperty(document, 'hidden', {
      configurable: true,
      value: false,
    })
  })

  it('requests prompt-manager scoped runs and applies status filter from UI', async () => {
    render(
      <RunListPanel
        selectedRunId={null}
        onSelectRun={vi.fn()}
      />
    )

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

    fireEvent.click(screen.getByRole('button', { name: 'Running' }))

    await waitFor(() => {
      expect(listRuns).toHaveBeenCalledWith({
        status: 'running',
        tagPrefix: undefined,
        profileKey: 'prompt-manager-heartbeat',
        limit: 100,
      })
      expect(listHeartbeatAttempts).toHaveBeenCalledWith({
        status: 'running',
        profileKey: 'prompt-manager-heartbeat',
        limit: 100,
      })
    })
  })

  it('surfaces pre-run heartbeat attempts with their error text', async () => {
    vi.mocked(listHeartbeatAttempts).mockResolvedValue({
      attempts: [{
        id: 'attempt-1',
        teamId: 'scenario-qa',
        agentId: 'bug-investigator',
        status: 'failed',
        phase: 'creating_run',
        startedAt: '2026-05-06T12:00:00Z',
        error: 'creating run: validation error',
      }],
      total: 1,
      hasMore: false,
    })

    render(
      <RunListPanel
        selectedRunId={null}
        onSelectRun={vi.fn()}
      />
    )

    expect(await screen.findByText('creating run: validation error')).toBeInTheDocument()
    expect(screen.getByText('attempt')).toBeInTheDocument()
  })
})
