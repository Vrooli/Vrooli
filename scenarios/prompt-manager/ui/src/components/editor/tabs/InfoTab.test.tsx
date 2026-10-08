import { beforeEach, describe, expect, it, vi } from 'vitest'
import { renderWithProviders, screen, waitFor, within } from '@/test-utils/renderWithProviders'
import { InfoTab } from './InfoTab'
import { getAgentTeams } from '@/services/agentService'
import { getHeartbeat, listRuns } from '@/services/heartbeatService'
import type { Agent } from '@/types/agent'

vi.mock('@/services/agentService', () => ({
  getAgentTeams: vi.fn(),
}))

vi.mock('@/services/heartbeatService', () => ({
  getHeartbeat: vi.fn(),
  listRuns: vi.fn(),
}))

const agent: Agent = {
  id: 'agent-coordinator',
  displayName: 'Coordinator',
  description: 'Campaign coordinator',
  status: 'active',
  appearance: null,
  capabilities: null,
  connectors: [],
  tags: [],
  fileOrder: [],
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
}

const membership = (teamId: string, teamDisplayName = teamId) => ({
  teamId,
  teamDisplayName,
  status: 'active' as const,
  roles: [],
})

const heartbeat = (teamId: string, status: 'completed' | 'failed' = 'completed', profileKey = `${teamId}/profile`) => ({
  teamId,
  agentId: agent.id,
  enabled: true,
  schedule: '*/5 * * * *',
  profileKey,
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
  lastExecution: {
    startedAt: '2026-01-02T00:00:00Z',
    endedAt: '2026-01-02T00:01:00Z',
    status,
    runId: `${teamId}-run`,
  },
})

describe('InfoTab evidence coverage states', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(getHeartbeat).mockResolvedValue(heartbeat('team-a'))
    vi.mocked(listRuns).mockResolvedValue({ runs: [], total: 0, hasMore: false })
  })

  it('keeps loading evidence visibly distinct from zero memberships', () => {
    vi.mocked(getAgentTeams).mockReturnValue(new Promise(() => {}))
    vi.mocked(getHeartbeat).mockReturnValue(new Promise(() => {}))
    vi.mocked(listRuns).mockReturnValue(new Promise(() => {}))

    renderWithProviders(<InfoTab agent={agent} />, { withRouter: true })

    expect(screen.getByText('Reading evidence')).toBeInTheDocument()
    expect(screen.getAllByText('Loading').length).toBeGreaterThan(0)
    expect(screen.getByText('…')).toBeInTheDocument()
  })

  it('labels an observed empty membership result as empty rather than unavailable', async () => {
    vi.mocked(getAgentTeams).mockResolvedValue([])

    renderWithProviders(<InfoTab agent={agent} />, { withRouter: true })

    expect(await screen.findByText('No memberships')).toBeInTheDocument()
    expect(screen.getAllByText('Empty')).toHaveLength(2)
    expect(within(screen.getByLabelText('Agent profile summary')).getAllByText('0')).toHaveLength(3)
  })

  it('labels an owner failure as unavailable and never fabricates zero coverage', async () => {
    vi.mocked(getAgentTeams).mockRejectedValue(new Error('owner unavailable'))

    renderWithProviders(<InfoTab agent={agent} />, { withRouter: true })

    expect(await screen.findByText('Coverage unavailable')).toBeInTheDocument()
    expect(screen.getAllByText('Unavailable').length).toBeGreaterThan(0)
    expect(screen.queryByText('No memberships')).not.toBeInTheDocument()
  })

  it('distinguishes partial signal coverage and failed latest-run attention', async () => {
    vi.mocked(getAgentTeams).mockResolvedValue([membership('team-a'), membership('team-b')])
    vi.mocked(getHeartbeat).mockImplementation(async (teamId) => {
      if (teamId === 'team-b') throw new Error('heartbeat unavailable')
      return heartbeat('team-a', 'failed')
    })

    renderWithProviders(<InfoTab agent={agent} />, { withRouter: true })

    expect(await screen.findByText('Partial coverage')).toBeInTheDocument()
    expect(screen.getAllByText('Attention')).toHaveLength(2)
    expect(within(screen.getByLabelText('Agent profile summary')).getByText('1')).toBeInTheDocument()
    await waitFor(() => expect(getHeartbeat).toHaveBeenCalledTimes(2))
  })

  it('labels fully observed current health as observed, not partial', async () => {
    vi.mocked(getAgentTeams).mockResolvedValue([membership('team-a')])
    vi.mocked(getHeartbeat).mockResolvedValue({
      ...heartbeat('team-a'),
      lastExecution: {
        ...heartbeat('team-a').lastExecution!,
        endedAt: new Date().toISOString(),
      },
    })

    renderWithProviders(<InfoTab agent={agent} />, { withRouter: true })

    expect(await screen.findByText('Observed')).toBeInTheDocument()
    expect(screen.getByText(/All current membership signals are available and fresh/)).toBeInTheDocument()
  })

  it('keeps activity loading distinct from empty history', async () => {
    vi.mocked(getAgentTeams).mockResolvedValue([])

    renderWithProviders(<InfoTab agent={agent} />, { withRouter: true })

    expect((await screen.findAllByText('Empty history')).length).toBe(2)
    expect(listRuns).not.toHaveBeenCalled()
  })

  it('labels old execution evidence as stale instead of current health', async () => {
    vi.mocked(getAgentTeams).mockResolvedValue([membership('team-a')])
    vi.mocked(getHeartbeat).mockResolvedValue(heartbeat('team-a'))

    renderWithProviders(<InfoTab agent={agent} />, { withRouter: true })

    expect(await screen.findByText('Stale evidence')).toBeInTheDocument()
    expect(screen.getAllByText('Stale', { selector: 'span' })).toHaveLength(1)
    expect(screen.getByText(/evidence older than the freshness window/)).toBeInTheDocument()
  })

  it('renders owner-observed agent run history without turning it into current health', async () => {
    vi.mocked(getAgentTeams).mockResolvedValue([membership('team-a')])
    vi.mocked(listRuns).mockResolvedValue({
      runs: [
        { id: 'agent-run-1', taskId: 'task-1', status: 'completed', startedAt: '2026-09-16T10:00:00Z', endedAt: '2026-09-16T10:01:00Z', agentId: agent.id },
        { id: 'agent-run-2', taskId: 'task-2', status: 'failed', startedAt: '2026-09-16T09:00:00Z', endedAt: '2026-09-16T09:01:00Z', agentId: agent.id },
      ],
      total: 2,
      hasMore: false,
    })

    renderWithProviders(<InfoTab agent={agent} />, { withRouter: true })

    expect(await screen.findByText('Agent-scoped run evidence observed')).toBeInTheDocument()
    expect(screen.getByText('Owner-observed runs')).toBeInTheDocument()
    expect(screen.getByText(/completed successfully/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'agent-run-1' })).toBeInTheDocument()
    expect(screen.getByText('Current health')).toBeInTheDocument()
    expect(listRuns).toHaveBeenCalledWith({ profileKey: 'team-a/profile', limit: 100 })
  })

  it('scopes owner history through heartbeat profile keys instead of the Prompt Manager agent slug', async () => {
    vi.mocked(getAgentTeams).mockResolvedValue([membership('team-a'), membership('team-b')])
    vi.mocked(getHeartbeat).mockImplementation(async (teamId) => heartbeat(teamId, 'completed', `${teamId}/profile`))
    vi.mocked(listRuns).mockImplementation(async (opts = {}) => ({
      runs: [{ id: `${opts.profileKey}-run`, taskId: 'task-1', status: 'completed', startedAt: '2026-09-16T10:00:00Z', agentId: agent.id }],
      total: 1,
      hasMore: false,
    }))

    renderWithProviders(<InfoTab agent={agent} />, { withRouter: true })

    expect(await screen.findByText('Agent-scoped run evidence observed')).toBeInTheDocument()
    expect(listRuns).not.toHaveBeenCalledWith({ agentId: agent.id, limit: 100 })
    expect(listRuns).toHaveBeenCalledTimes(2)
  })

  it('keeps mixed owner profile reads visibly partial', async () => {
    vi.mocked(getAgentTeams).mockResolvedValue([membership('team-a'), membership('team-b')])
    vi.mocked(getHeartbeat).mockImplementation(async (teamId) => heartbeat(teamId, 'completed', `${teamId}/profile`))
    vi.mocked(listRuns).mockImplementation(async (opts = {}) => {
      if (opts.profileKey === 'team-b/profile') throw new Error('owner profile unavailable')
      return { runs: [{ id: 'team-a-run', taskId: 'task-1', status: 'completed', startedAt: '2026-09-16T10:00:00Z', agentId: agent.id }], total: 1, hasMore: false }
    })

    renderWithProviders(<InfoTab agent={agent} />, { withRouter: true })

    expect(await screen.findByText('Agent-scoped run evidence partial')).toBeInTheDocument()
    expect(screen.getAllByText('Partial').length).toBeGreaterThan(0)
    expect(screen.getByText(/one or more owned profile reads were unavailable/)).toBeInTheDocument()
  })
})
