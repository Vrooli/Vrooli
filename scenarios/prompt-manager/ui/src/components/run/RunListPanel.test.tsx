import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { RunListPanel } from './RunListPanel'
import { listRuns } from '@/services/heartbeatService'

vi.mock('@/services/heartbeatService', () => ({
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
    })

    fireEvent.click(screen.getByRole('button', { name: 'Running' }))

    await waitFor(() => {
      expect(listRuns).toHaveBeenCalledWith({
        status: 'running',
        tagPrefix: undefined,
        profileKey: 'prompt-manager-heartbeat',
        limit: 100,
      })
    })
  })
})
