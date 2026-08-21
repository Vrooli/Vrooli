import { describe, it, expect, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@/test-utils/renderWithProviders'
import { MemoryRouter } from 'react-router-dom'
import { RunInfoTab } from './RunInfoTab'
import { getRunDetails, retryRun } from '@/services/heartbeatService'

const toastMock = vi.fn()

vi.mock('@/services/heartbeatService', () => ({
  getRunDetails: vi.fn(),
  retryRun: vi.fn(),
}))

vi.mock('@/hooks/use-toast', () => ({
  toast: (...args: unknown[]) => toastMock(...args),
}))

vi.mock('@/components/shared/EventsDisplay', () => ({
  CopyButton: () => <button type="button">Copy</button>,
}))

describe('RunInfoTab', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(getRunDetails).mockResolvedValue({
      id: 'run-failed-1',
      taskId: 'task-1',
      status: 'failed',
      error: 'intermittent upstream error',
      startedAt: '2026-02-18T00:00:00Z',
      endedAt: '2026-02-18T00:01:00Z',
      actions: {
        canInvestigate: true,
        canApplyInvestigation: false,
        canDelete: true,
        canStop: false,
        canRetry: true,
        canContinue: false,
      },
    })
    vi.mocked(retryRun).mockResolvedValue({
      teamId: 'team-1',
      agentId: 'agent-1',
      runId: 'run-retry-1',
      status: 'running',
    })
  })

  it('shows Retry in error section and triggers retry endpoint', async () => {
    render(
      <MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
        <RunInfoTab runId="run-failed-1" />
      </MemoryRouter>
    )

    await screen.findByText('intermittent upstream error')

    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))

    await waitFor(() => {
      expect(retryRun).toHaveBeenCalledWith('run-failed-1')
    })
    expect(toastMock).toHaveBeenCalledWith(expect.objectContaining({
      title: 'Retry triggered',
      variant: 'success',
    }))
  })
})
