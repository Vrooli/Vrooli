import type { ComponentProps } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@/test-utils/renderWithProviders'
import { OrgChartPanel } from './OrgChartPanel'
import { OrgChartNode } from './OrgChartNode'
import type { TeamDetails } from '@/types/team'
import type { OrgChartNode as OrgChartFlowNode } from '@/types/orgChart'
import * as heartbeatService from '@/services/heartbeatService'

const state = vi.hoisted(() => ({
  flowProps: undefined as Record<string, unknown> | undefined,
  runningAgents: [] as Array<{ agentId: string; teamId: string; runId: string }>,
}))

vi.mock('@xyflow/react', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@xyflow/react')>()
  return { ...actual, Handle: () => null }
})

vi.mock('@/components/graph/FlowShell', () => ({
  FlowShell: (props: Record<string, unknown>) => {
    state.flowProps = props
    return <div data-testid="org-chart-flow" />
  },
  layoutFlowDagre: <Node, Edge>(nodes: Node[], edges: Edge[]) => ({ nodes, edges }),
}))

vi.mock('@/hooks/useRunningAgents', () => ({
  useRunningAgents: () => ({ runningAgents: state.runningAgents, isLoading: false }),
}))

vi.mock('@/services/heartbeatService', async () => {
  const actual = await vi.importActual<typeof import('@/services/heartbeatService')>('@/services/heartbeatService')
  return {
    ...actual,
    listHeartbeats: vi.fn(),
    getTeamExecutionStatus: vi.fn(),
  }
})

const team = (id: string): TeamDetails => ({
  id,
  displayName: id,
  mission: '',
  enabled: true,
  runtime: { mode: 'multi-process' },
  coordination: { reportingMode: 'org-chart' } as TeamDetails['coordination'],
  execution: { maxConcurrentRuns: 1 } as TeamDetails['execution'],
  operatingContract: {} as TeamDetails['operatingContract'],
  memberCount: 1,
  roles: [],
  members: [{ agentId: 'worker', displayName: 'Worker', roles: [], status: 'active' }],
  createdAt: '2026-10-16T00:00:00Z',
  updatedAt: '2026-10-16T00:00:00Z',
})

function latestWorkerNode() {
  const nodes = state.flowProps?.nodes as OrgChartFlowNode[] | undefined
  return nodes?.find((node) => node.id === 'worker')
}

const emptyExecutionStatus = (teamId: string) => ({
  teamId,
  state: 'idle',
  runningAgentIds: [],
  uncertainAgentIds: [],
  unreachableAgentIds: [],
  pausedAgentIds: [],
  approvalPendingAgentIds: [],
  needsReviewAgentIds: [],
  watcherUnavailableAgentIds: [],
  preparationInterruptedAgentIds: [],
  admissionDeferredAgentIds: [],
})

describe('OrgChartPanel heartbeat owner projection', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    state.flowProps = undefined
    state.runningAgents = []
  })

  afterEach(() => { vi.restoreAllMocks() })

  it('retains watcher-unavailable owner status when the live-run hint also exists', async () => {
    vi.mocked(heartbeatService.listHeartbeats).mockResolvedValue([{
      teamId: 'team-a', agentId: 'worker', enabled: true, schedule: '*/5 * * * *',
      createdAt: '2026-10-16T00:00:00Z', updatedAt: '2026-10-16T00:00:00Z',
      lastExecution: { status: 'completed', runId: 'older-success', startedAt: '2026-10-15T00:00:00Z' },
    }])
    vi.mocked(heartbeatService.getTeamExecutionStatus).mockResolvedValue({
      ...emptyExecutionStatus('team-a'), runningAgentIds: ['worker'], watcherUnavailableAgentIds: ['worker'],
    })
    state.runningAgents = [{ agentId: 'worker', teamId: 'team-a', runId: 'live-run' }]

    render(<OrgChartPanel
      team={team('team-a')} edges={[]} allAgents={[]} selectedMemberId={null}
      onSelectMember={vi.fn()} onEdgeUpdate={vi.fn()} onAddMember={vi.fn()}
    />)

    await waitFor(() => expect(latestWorkerNode()?.data.heartbeatStatus).toBe('observer_unavailable'))
  })

  it('shows an interrupted preparation distinctly without claiming it is running', async () => {
    vi.mocked(heartbeatService.listHeartbeats).mockResolvedValue([{
      teamId: 'team-a', agentId: 'worker', enabled: true, schedule: '*/5 * * * *',
      createdAt: '2026-10-16T00:00:00Z', updatedAt: '2026-10-16T00:00:00Z',
      lastExecution: { status: 'completed', runId: 'older-success', startedAt: '2026-10-15T00:00:00Z' },
    }])
    vi.mocked(heartbeatService.getTeamExecutionStatus).mockResolvedValue({
      ...emptyExecutionStatus('team-a'), preparationInterruptedAgentIds: ['worker'],
    })

    render(<OrgChartPanel
      team={team('team-a')} edges={[]} allAgents={[]} selectedMemberId={null}
      onSelectMember={vi.fn()} onEdgeUpdate={vi.fn()} onAddMember={vi.fn()}
    />)

    await waitFor(() => expect(latestWorkerNode()?.data.heartbeatStatus).toBe('preparation_interrupted'))
  })

  it('shows capacity-deferred admission as waiting, not running after an older success', async () => {
    vi.mocked(heartbeatService.listHeartbeats).mockResolvedValue([{
      teamId: 'team-a', agentId: 'worker', enabled: true, schedule: '*/5 * * * *',
      createdAt: '2026-10-16T00:00:00Z', updatedAt: '2026-10-16T00:00:00Z',
      lastExecution: { status: 'completed', runId: 'older-success', startedAt: '2026-10-15T00:00:00Z' },
    }])
    vi.mocked(heartbeatService.getTeamExecutionStatus).mockResolvedValue({
      ...emptyExecutionStatus('team-a'), admissionDeferredAgentIds: ['worker'],
    })

    render(<OrgChartPanel
      team={team('team-a')} edges={[]} allAgents={[]} selectedMemberId={null}
      onSelectMember={vi.fn()} onEdgeUpdate={vi.fn()} onAddMember={vi.fn()}
    />)

    await waitFor(() => expect(latestWorkerNode()?.data.heartbeatStatus).toBe('admission_deferred'))
    const deferredNode = latestWorkerNode()
    if (!deferredNode) throw new Error('worker node missing from OrgChartPanel')
    const deferredNodeProps = {
      id: deferredNode.id,
      data: deferredNode.data,
      type: deferredNode.type ?? 'orgMember',
      dragging: deferredNode.dragging ?? false,
      zIndex: deferredNode.zIndex ?? 0,
      selectable: deferredNode.selectable ?? true,
      deletable: deferredNode.deletable ?? true,
      selected: deferredNode.selected ?? false,
      draggable: deferredNode.draggable ?? true,
      isConnectable: true,
      positionAbsoluteX: deferredNode.position.x,
      positionAbsoluteY: deferredNode.position.y,
    } satisfies ComponentProps<typeof OrgChartNode>
    render(<OrgChartNode {...deferredNodeProps} />)
    expect(screen.getByTitle(/waiting for capacity; scheduled retry/i)).toBeInTheDocument()
    expect(screen.getByTitle(/waiting for capacity; scheduled retry/i).querySelector('svg')).toHaveClass('text-muted-foreground/50')
  })

  it('does not reuse a prior team heartbeat config after the next config read fails', async () => {
    vi.mocked(heartbeatService.listHeartbeats)
      .mockResolvedValueOnce([{
        teamId: 'team-a', agentId: 'worker', enabled: true, schedule: '*/5 * * * *',
        createdAt: '2026-10-16T00:00:00Z', updatedAt: '2026-10-16T00:00:00Z',
        lastExecution: { status: 'completed', runId: 'older-success', startedAt: '2026-10-15T00:00:00Z' },
      }])
      .mockRejectedValueOnce(new Error('owner read unavailable'))
    vi.mocked(heartbeatService.getTeamExecutionStatus)
      .mockResolvedValueOnce(emptyExecutionStatus('team-a'))
      .mockResolvedValueOnce(emptyExecutionStatus('team-b'))

    const props = {
      edges: [], allAgents: [], selectedMemberId: null,
      onSelectMember: vi.fn(), onEdgeUpdate: vi.fn(), onAddMember: vi.fn(),
    }
    const view = render(<OrgChartPanel team={team('team-a')} {...props} />)
    await waitFor(() => expect(latestWorkerNode()?.data.heartbeatStatus).toBe('completed'))

    view.rerender(<OrgChartPanel team={team('team-b')} {...props} />)

    await waitFor(() => expect(heartbeatService.listHeartbeats).toHaveBeenCalledTimes(2))
    await waitFor(() => expect(latestWorkerNode()?.data.heartbeatStatus).toBe('unknown'))
    const unknownNode = latestWorkerNode()
    if (!unknownNode) throw new Error('worker node missing from OrgChartPanel')
    const unknownNodeProps = {
      id: unknownNode.id,
      data: unknownNode.data,
      type: unknownNode.type ?? 'orgMember',
      dragging: unknownNode.dragging ?? false,
      zIndex: unknownNode.zIndex ?? 0,
      selectable: unknownNode.selectable ?? true,
      deletable: unknownNode.deletable ?? true,
      selected: unknownNode.selected ?? false,
      draggable: unknownNode.draggable ?? true,
      isConnectable: true,
      positionAbsoluteX: unknownNode.position.x,
      positionAbsoluteY: unknownNode.position.y,
    } satisfies ComponentProps<typeof OrgChartNode>
    render(<OrgChartNode {...unknownNodeProps} />)
    expect(screen.getByTitle('Heartbeat status unavailable')).toBeInTheDocument()
  })
})
