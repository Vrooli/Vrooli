import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { TeamDashboardTab } from './TeamDashboardTab'
import type { TeamDetails } from '@/types/team'
import {
  buildBoundedParallelExecution,
  buildDefaultCreateTeamRequest,
  buildIndependentCoordination,
  buildLeaderLedCoordination,
} from '@/lib/schemas'
import * as heartbeatService from '@/services/heartbeatService'

vi.mock('@/components/shared/ExpandableDescription', () => ({
  ExpandableDescription: ({ value }: { value: string }) => <div>{value}</div>,
}))

vi.mock('@/components/shared/AgentColorBadge', () => ({
  AgentColorBadge: () => <div data-testid="agent-color-badge" />,
}))

vi.mock('@/services/heartbeatService', async () => {
  const actual = await vi.importActual<typeof import('@/services/heartbeatService')>('@/services/heartbeatService')
  return {
    ...actual,
    listHeartbeats: vi.fn(),
    listTeamLogs: vi.fn(),
  }
})

const baseTeam: TeamDetails = {
  id: 'scenario-qa',
  displayName: 'Scenario QA',
  mission: 'Validate important scenarios.',
  enabled: true,
  runtime: { mode: 'multi-process' },
  coordination: buildIndependentCoordination(),
  execution: buildBoundedParallelExecution(2),
  decisionMode: 'yolo',
  operatingContract: buildDefaultCreateTeamRequest('Scenario QA').operatingContract,
  memberCount: 2,
  roles: [],
  members: [
    { agentId: 'lead', displayName: 'Lead Agent', roles: [], status: 'active' },
    { agentId: 'worker', displayName: 'Worker Agent', roles: [], status: 'active' },
  ],
  createdAt: '2026-04-09T00:00:00Z',
  updatedAt: '2026-04-09T00:00:00Z',
}

describe('TeamDashboardTab', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(heartbeatService.listHeartbeats).mockResolvedValue([])
    vi.mocked(heartbeatService.listTeamLogs).mockResolvedValue({
      teamId: baseTeam.id,
      logs: [],
      total: 0,
      hasMore: false,
    })
  })

  function renderDashboard(team: TeamDetails, onUpdate: (updates: unknown) => Promise<void>) {
    return render(
      <MemoryRouter>
        <TeamDashboardTab team={team} onUpdate={onUpdate} />
      </MemoryRouter>
    )
  }

  it('promotes multi-process teams to leader-led serialized execution when switching to single-process runtime', () => {
    const onUpdate = vi.fn().mockResolvedValue(undefined)
    renderDashboard(baseTeam, onUpdate)

    fireEvent.click(screen.getByRole('button', { name: 'Single-Process' }))

    expect(onUpdate).toHaveBeenCalledWith({
      runtime: { mode: 'single-process' },
      coordination: buildLeaderLedCoordination('lead', 'single-process'),
      execution: { queuePolicy: 'serialized', maxConcurrentRuns: 1 },
    })
  })

  it('applies the peer preset from the coordination controls', () => {
    const onUpdate = vi.fn().mockResolvedValue(undefined)
    renderDashboard(baseTeam, onUpdate)

    fireEvent.click(screen.getByRole('button', { name: 'Peer' }))

    expect(onUpdate).toHaveBeenCalledWith({
      coordination: {
        pattern: 'peer',
        reportingMode: 'org-chart',
        messagingMode: 'async-inbox',
        capabilities: {
          showOrgContext: true,
          injectInbox: true,
          allowPeerTriggers: true,
          showTaskBoardGuidance: true,
          showDecisionLogGuidance: true,
          showKnowledgeLogGuidance: true,
          requireHandoff: true,
        },
      },
      execution: baseTeam.execution,
    })
  })

  it('updates the queue policy from the execution controls', () => {
    const onUpdate = vi.fn().mockResolvedValue(undefined)
    const team = {
      ...baseTeam,
      execution: { queuePolicy: 'bounded-parallel', maxConcurrentRuns: 4 },
    } satisfies TeamDetails

    renderDashboard(team, onUpdate)

    fireEvent.click(screen.getByRole('button', { name: 'Serialized' }))

    expect(onUpdate).toHaveBeenCalledWith({
      execution: { queuePolicy: 'serialized', maxConcurrentRuns: 1 },
    })
  })

  it('updates bounded parallel concurrency from the numeric control', () => {
    const onUpdate = vi.fn().mockResolvedValue(undefined)
    renderDashboard(baseTeam, onUpdate)

    fireEvent.change(screen.getByLabelText('Max Concurrent Runs'), { target: { value: '5' } })

    expect(onUpdate).toHaveBeenCalledWith({
      execution: {
        queuePolicy: 'bounded-parallel',
        maxConcurrentRuns: 5,
      },
    })
  })
})
