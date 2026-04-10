import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import type React from 'react'
import { MemberDetailPanel } from './MemberDetailPanel'
import type { TeamDetails, TeamMember } from '@/types/team'
import * as heartbeatService from '@/services/heartbeatService'
import { useSelectionStore } from '@/stores/selectionStore'

const toastMock = vi.fn()

vi.mock('@/hooks/use-toast', () => ({
  toast: (...args: unknown[]) => toastMock(...args),
}))

vi.mock('./MemberScheduleSection', () => ({
  MemberScheduleSection: ({ onTriggerHeartbeat }: { onTriggerHeartbeat: () => void }) => (
    <button type="button" onClick={onTriggerHeartbeat}>Mock Run Now</button>
  ),
}))

vi.mock('./MemberPromptPipelineSection', () => ({
  MemberPromptPipelineSection: () => <div>pipeline</div>,
}))

vi.mock('@/services/heartbeatService', async () => {
  const actual = await vi.importActual<typeof import('@/services/heartbeatService')>('@/services/heartbeatService')
  return {
    ...actual,
    getResponsibilities: vi.fn(),
    getHeartbeatInstructions: vi.fn(),
    getHeartbeat: vi.fn(),
    listLogs: vi.fn(),
    triggerHeartbeat: vi.fn(),
  }
})

describe('MemberDetailPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useSelectionStore.getState().clearAllSelection()

    vi.mocked(heartbeatService.getResponsibilities).mockResolvedValue('')
    vi.mocked(heartbeatService.getHeartbeatInstructions).mockResolvedValue('')
    vi.mocked(heartbeatService.getHeartbeat).mockResolvedValue({
      teamId: 'team-1',
      agentId: 'agent-1',
      enabled: true,
      schedule: '0 * * * *',
      createdAt: '2026-02-18T00:00:00Z',
      updatedAt: '2026-02-18T00:00:00Z',
    })
    vi.mocked(heartbeatService.listLogs).mockResolvedValue([])
    vi.mocked(heartbeatService.triggerHeartbeat).mockResolvedValue({
      teamId: 'team-1',
      agentId: 'agent-1',
      runId: 'run-new-1',
      status: 'running',
    })
  })

  it('shows Open Run CTA on heartbeat triggered toast', async () => {
    render(
      <MemberDetailPanel
        team={{ id: 'team-1', displayName: 'Team', roles: [] } as unknown as TeamDetails}
        member={{ agentId: 'agent-1', displayName: 'Agent', status: 'active', roles: [] } as unknown as TeamMember}
        onUpdateMember={vi.fn().mockResolvedValue({})}
        onRemoveMember={vi.fn().mockResolvedValue(undefined)}
        onClose={vi.fn()}
      />
    )

    const runNow = await screen.findByRole('button', { name: 'Mock Run Now' })
    fireEvent.click(runNow)

    await waitFor(() => {
      expect(heartbeatService.triggerHeartbeat).toHaveBeenCalledWith('team-1', 'agent-1')
    })
    expect(toastMock).toHaveBeenCalledWith(expect.objectContaining({
      title: 'Heartbeat triggered',
      variant: 'success',
      action: expect.anything(),
    }))

    const toastArg = toastMock.mock.calls[0]?.[0] as { action?: React.ReactElement }
    expect(toastArg.action).toBeTruthy()

    if (!toastArg.action) {
      throw new Error('Expected toast action to exist')
    }

    render(toastArg.action)
    fireEvent.click(screen.getByRole('button', { name: 'Open Run' }))

    expect(useSelectionStore.getState().selectedRunId).toBe('run-new-1')
  })
})
