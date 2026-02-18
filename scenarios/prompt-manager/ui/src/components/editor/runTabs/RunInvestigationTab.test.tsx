import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { RunInvestigationTab } from './RunInvestigationTab'
import {
  listRuns,
  continueRun,
  getRunDetails,
  createInvestigationRun,
  createInvestigationApplyRun,
} from '@/services/heartbeatService'

vi.mock('@/components/shared/EventsDisplay', () => ({
  EventsDisplay: ({ runId }: { runId: string }) => <div data-testid="events-display">events:{runId}</div>,
}))

vi.mock('@/services/heartbeatService', () => ({
  listRuns: vi.fn(),
  continueRun: vi.fn(),
  getRunDetails: vi.fn(),
  createInvestigationRun: vi.fn(),
  createInvestigationApplyRun: vi.fn(),
}))

function makeRun(id: string, status: string) {
  return {
    id,
    taskId: `task-${id}`,
    status,
    startedAt: '2026-02-17T00:00:00Z',
  }
}

describe('RunInvestigationTab', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(listRuns).mockResolvedValue({
      runs: [makeRun('inv-1', 'completed')],
      total: 1,
      hasMore: false,
    })
    vi.mocked(getRunDetails).mockResolvedValue(makeRun('inv-1', 'completed'))
    vi.mocked(continueRun).mockResolvedValue()
    vi.mocked(createInvestigationRun).mockResolvedValue(makeRun('inv-2', 'running'))
    vi.mocked(createInvestigationApplyRun).mockResolvedValue(makeRun('apply-1', 'running'))
  })

  it('restores linked investigations on mount via investigates_run_id filter', async () => {
    const { unmount } = render(<RunInvestigationTab runId="source-1" />)

    await screen.findByText(/Selected investigation status:/i)
    expect(listRuns).toHaveBeenCalledWith({ investigatesRunId: 'source-1', limit: 50 })
    expect(screen.getByRole('button', { name: /Refresh investigations/i })).toBeInTheDocument()

    unmount()

    render(<RunInvestigationTab runId="source-1" />)
    await screen.findByText(/Selected investigation status:/i)

    expect(listRuns).toHaveBeenCalledWith({ investigatesRunId: 'source-1', limit: 50 })
  })

  it('uses continueRun for Follow Up action', async () => {
    render(<RunInvestigationTab runId="source-1" />)

    fireEvent.click(await screen.findByRole('button', { name: /Follow Up/i }))

    const input = await screen.findByPlaceholderText(/Add a message for follow up/i)
    fireEvent.change(input, { target: { value: 'Please explain root cause in one paragraph.' } })

    fireEvent.click(screen.getByRole('button', { name: /Send follow up/i }))

    await waitFor(() => {
      expect(continueRun).toHaveBeenCalledWith('inv-1', 'Please explain root cause in one paragraph.')
    })
    expect(createInvestigationRun).not.toHaveBeenCalled()
  })

  it('starts a new investigation from follow-up message when requested', async () => {
    render(<RunInvestigationTab runId="source-1" />)

    fireEvent.click(await screen.findByRole('button', { name: /New Investigation/i }))

    const input = await screen.findByPlaceholderText(/Add a message for new investigation/i)
    fireEvent.change(input, { target: { value: 'Try a deeper investigation focused on tool-call ordering.' } })

    fireEvent.click(screen.getByRole('button', { name: /Send new investigation/i }))

    await waitFor(() => {
      expect(createInvestigationRun).toHaveBeenCalledWith(
        ['source-1'],
        expect.objectContaining({
          depth: 'standard',
          customContext: 'Try a deeper investigation focused on tool-call ordering.',
        })
      )
    })
    expect(continueRun).not.toHaveBeenCalled()
  })

  it('supports apply recommendations with an optional message', async () => {
    render(<RunInvestigationTab runId="source-1" />)

    fireEvent.click(await screen.findByRole('button', { name: /Apply Recommendations/i }))
    fireEvent.click(screen.getByRole('button', { name: /Send apply recommendations/i }))

    await waitFor(() => {
      expect(createInvestigationApplyRun).toHaveBeenCalledWith('inv-1', undefined)
    })
  })
})
