import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@/test-utils/renderWithProviders'
import { TeamDashboardTab } from './TeamDashboardTab'
import type { TeamDetails } from '@/types/team'
import {
  buildBoundedParallelExecution,
  buildDefaultCreateTeamRequest,
  buildIndependentCoordination,
  buildLeaderLedCoordination,
} from '@/lib/schemas'
import * as heartbeatService from '@/services/heartbeatService'

const effortPanel = vi.hoisted(() => ({ render: vi.fn() }))
vi.mock('@/components/team/TeamEffortsPanel', () => ({
  TeamEffortsPanel: (props: unknown) => { effortPanel.render(props); return null },
}))

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
    getTeamRunAccounting: vi.fn(),
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

const pendingBackgroundRequest = new Promise<never>(() => {})

describe('TeamDashboardTab', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  beforeEach(() => {
    vi.clearAllMocks()
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ items: [] }), { status: 200 })))
    vi.mocked(heartbeatService.listHeartbeats).mockReturnValue(pendingBackgroundRequest)
    vi.mocked(heartbeatService.listTeamLogs).mockReturnValue(pendingBackgroundRequest)
    vi.mocked(heartbeatService.getTeamRunAccounting).mockReturnValue(pendingBackgroundRequest)
  })

  function renderDashboard(team: TeamDetails, onUpdate: (updates: unknown) => Promise<void>) {
    return render(<TeamDashboardTab team={team} onUpdate={onUpdate} />)
  }

  it('reports unknown execution health when there are no execution records', async () => {
    vi.mocked(heartbeatService.listHeartbeats).mockResolvedValue([{
      teamId: baseTeam.id, agentId: 'lead', enabled: true, schedule: '*/5 * * * *',
      createdAt: baseTeam.createdAt, updatedAt: baseTeam.updatedAt,
    }])
    const onHealthChange = vi.fn()
    render(<TeamDashboardTab team={baseTeam} onUpdate={vi.fn()} onHealthChange={onHealthChange} />)
    await waitFor(() => expect(effortPanel.render).toHaveBeenLastCalledWith(expect.objectContaining({ observationAvailable: true })))
    expect(onHealthChange).toHaveBeenLastCalledWith('gray')
    expect(onHealthChange).not.toHaveBeenCalledWith('green')
  })

  it('passes active supervision references and a finite binding separately from scheduling eligibility', async () => {
    vi.mocked(heartbeatService.listHeartbeats).mockResolvedValue([{
      teamId: baseTeam.id, agentId: 'lead', enabled: false, schedule: '*/5 * * * *',
      createdAt: baseTeam.createdAt, updatedAt: baseTeam.updatedAt,
      supervisionState: { efforts: {
        'effort:observed': { observationOnly: true },
        'effort:retired': { retired: true },
      } },
      finiteLeader: { effortRef: 'effort:delivery' },
    }, {
      teamId: baseTeam.id, agentId: 'worker', enabled: false, schedule: '*/5 * * * *',
      createdAt: baseTeam.createdAt, updatedAt: baseTeam.updatedAt,
      finiteLeader: { effortRef: 'effort:retired-leader', retired: true },
    }])
    renderDashboard(baseTeam, vi.fn())
    await waitFor(() => expect(effortPanel.render).toHaveBeenLastCalledWith(expect.objectContaining({
      team: baseTeam,
      observationAvailable: true,
      observedEffortRefs: ['effort:observed'],
      leaderEffortRefs: ['effort:delivery'],
      scheduled: { enabled: false, summary: 'lead: disabled; worker: disabled' },
    })))
  })

  it.each([
    { source: 'owner read failed', extra: { supervision: {}, supervisionError: 'cannot read supervision reservation' } },
    { source: 'state source missing', extra: { supervision: {} } },
    { source: 'supervision configuration missing', extra: {} },
  ])('keeps supervisor coverage unknown when $source despite successful heartbeat configuration reads', async ({ extra }) => {
    vi.mocked(heartbeatService.listHeartbeats).mockResolvedValue([{
      teamId: baseTeam.id, agentId: 'lead', enabled: true, schedule: '*/5 * * * *',
      createdAt: baseTeam.createdAt, updatedAt: baseTeam.updatedAt, ...extra,
    }])
    renderDashboard({ ...baseTeam, purpose: 'supervision' }, vi.fn())
    await waitFor(() => expect(effortPanel.render).toHaveBeenLastCalledWith(expect.objectContaining({
      observationAvailable: false,
      observedEffortRefs: [],
      scheduled: { enabled: true, summary: 'lead: schedule enabled' },
    })))
  })

  it('renders joined owner accounting through the real Connect wire boundary with no local logs', async () => {
    const actual = await vi.importActual<typeof import('@/services/heartbeatService')>('@/services/heartbeatService')
    vi.mocked(heartbeatService.getTeamRunAccounting).mockImplementation(actual.getTeamRunAccounting)
    vi.mocked(heartbeatService.listHeartbeats).mockResolvedValue([])
    vi.mocked(heartbeatService.listTeamLogs).mockResolvedValue({ teamId: baseTeam.id, logs: [], total: 0, hasMore: false })
    const wire = {
      teamId: baseTeam.id, windowStart: '2026-09-12T00:00:00Z', windowEnd: '2026-09-13T00:00:00Z', observedAt: '2026-09-12T15:00:00Z',
      knownRuns: 3, observedRuns: 2, unavailableRuns: 1, unqueriedRuns: 0, observedExecutions: 1, duplicateExecutions: 1,
      actualModels: { 'actual-model': 1 }, unknownModelExecutions: 0, runtimeStates: { complete: 1 }, terminalReasons: { blocked: 1 },
      usage: { tokens: null, costUSD: null, qualifiedRuns: 0, reportedTokenRuns: 2, reportedCostRuns: 0, partial: true },
      coverage: { partial: true, declarationsRead: 4, declarationLimit: 500, ownerReadLimit: 50, invalidTimestamps: 0,
        limitations: ['Retained declarations are a bounded sample.'] },
      runs: [{ runId: 'resumed', agentIds: ['lead'], declaredAt: '2026-09-12T10:00:00Z', availability: 'available' },
        { runId: 'imported', agentIds: ['worker'], declaredAt: '2026-09-12T10:00:00Z', availability: 'available' },
        { runId: 'missing', agentIds: ['worker'], declaredAt: '2026-09-12T10:00:00Z', availability: 'unavailable' }],
    }
    vi.stubGlobal('fetch', vi.fn().mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = input instanceof Request ? input.url : String(input)
      if (url.endsWith('.HeartbeatService/ListRuns')) {
        expect(JSON.parse(await new Response(init?.body).text()).query).toMatchObject({ accounting: 'true', team_id: baseTeam.id })
        return Promise.resolve(new Response(JSON.stringify({ data: wire }), { status: 200, headers: { 'content-type': 'application/json' } }))
      }
      if (url.includes('/backlog?')) return Promise.resolve(new Response(JSON.stringify({ items: [] }), { status: 200 }))
      throw new Error(`Unexpected request: ${url}`)
    }))
    renderDashboard(baseTeam, vi.fn())
    expect(await screen.findByText('3 known owner run IDs from PM declarations in 24h')).toBeInTheDocument()
    expect(screen.getByText('2 observed · 1 unavailable · 0 not queried')).toBeInTheDocument()
    expect(screen.getByText('1 distinct observed execution · 1 resumed/imported duplicate excluded')).toBeInTheDocument()
    expect(screen.getByText('Actual models: actual-model: 1 · 0 unknown')).toBeInTheDocument()
    expect(screen.getByText('Terminal reasons: blocked: 1')).toBeInTheDocument()
    expect(screen.getByText('Token usage: unknown · Actual charge: unknown')).toBeInTheDocument()
    expect(screen.getByText(/Usage coverage: 0\/2 owner runs qualified/)).toBeInTheDocument()
    expect(screen.getByText(/Partial history coverage/)).toBeInTheDocument()
    expect(screen.getByText(/Runtime completion does not establish outcome acceptance/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'missing', hidden: true })).toHaveAttribute('href', '/runs/missing')
    expect(screen.queryByText('100% success')).not.toBeInTheDocument()
  })

  it('shows owner unavailability without replacing it with zero run or usage totals', async () => {
    vi.mocked(heartbeatService.getTeamRunAccounting).mockRejectedValue(new Error('owner unavailable'))
    renderDashboard(baseTeam, vi.fn())
    expect(await screen.findByText('Owner run accounting unavailable.')).toBeInTheDocument()
    expect(screen.queryByText(/0 known owner run IDs/)).not.toBeInTheDocument()
    expect(screen.queryByText(/Token usage: 0/)).not.toBeInTheDocument()
  })

  it.each([
    { roster: 'populated', members: baseTeam.members },
    { roster: 'empty', members: [] },
  ])('opens the dashboard with no log files and a $roster roster', async ({ members }) => {
    const actual = await vi.importActual<typeof import('@/services/heartbeatService')>('@/services/heartbeatService')
    vi.mocked(heartbeatService.listTeamLogs).mockImplementation(actual.listTeamLogs)
    vi.mocked(heartbeatService.listHeartbeats).mockResolvedValue([])
    vi.stubGlobal('fetch', vi.fn().mockImplementation((input: RequestInfo | URL) => {
      const url = input instanceof Request ? input.url : String(input)
      if (url.endsWith('.HeartbeatService/ListTeamLogs')) {
        // Observed Connect response for a team with no member log files.
        return Promise.resolve(new Response(JSON.stringify({
          data: { teamId: 'effort-supervision', logs: null, total: 0, hasMore: false },
        }), { status: 200, headers: { 'content-type': 'application/json' } }))
      }
      if (url === '/embedded/swarm-manager/api/v1/backlog?archived=false') {
        return Promise.resolve(new Response(JSON.stringify({ items: [] }), { status: 200 }))
      }
      throw new Error(`Unexpected request: ${url}`)
    }))
    const onUpdate = vi.fn().mockResolvedValue(undefined)
    renderDashboard({ ...baseTeam, id: 'effort-supervision', members, memberCount: members.length }, onUpdate)

    expect(await screen.findByText('No local logs.')).toBeInTheDocument()
    expect(screen.getByText('0 local logs in 24h')).toBeInTheDocument()
    expect(await screen.findByText('No upcoming heartbeats in the next 24 hours.')).toBeInTheDocument()
    if (members.length === 0) expect(screen.getByText('No members yet.')).toBeInTheDocument()
    expect(await vi.mocked(heartbeatService.listTeamLogs).mock.results[0]?.value).toEqual({
      teamId: 'effort-supervision', logs: [], total: 0, hasMore: false,
    })
    expect(onUpdate).not.toHaveBeenCalled()
  })

  it('does not present empty local logs and one successful heartbeat as complete AM run metrics', async () => {
    vi.mocked(heartbeatService.listTeamLogs).mockResolvedValue({
      teamId: 'effort-supervision', logs: [], total: 0, hasMore: false,
    })
    vi.mocked(heartbeatService.listHeartbeats).mockResolvedValue([{
      teamId: 'effort-supervision', agentId: 'effort-supervisor', enabled: true,
      schedule: '*/5 * * * *', createdAt: baseTeam.createdAt, updatedAt: baseTeam.updatedAt,
      lastExecution: { status: 'completed', runId: 'latest-owner-run', startedAt: '2026-09-12T10:00:00Z' },
    }])
    renderDashboard({ ...baseTeam, id: 'effort-supervision' }, vi.fn())
    expect(await screen.findByText('0 local logs in 24h')).toBeInTheDocument()
    expect(screen.queryByText('0 runs in 24h')).not.toBeInTheDocument()
    expect(screen.queryByText('100% success')).not.toBeInTheDocument()
    expect(screen.getByText(/Run totals and success rate are unavailable/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Agent Manager run history' })).toHaveAttribute('href', '/embedded/agent-manager/')
  })

  it('marks counts from a partial local log page as a lower bound', async () => {
    vi.mocked(heartbeatService.listTeamLogs).mockResolvedValue({
      teamId: baseTeam.id, logs: [{ agentId: 'lead', agentDisplayName: 'Lead', filename: 'log.txt', timestamp: new Date().toISOString() }],
      total: 30, hasMore: true,
    })
    renderDashboard(baseTeam, vi.fn())
    expect(await screen.findByText('At least 1 local log in 24h')).toBeInTheDocument()
  })

  it('does not report zero evidence when local logs are unavailable', async () => {
    vi.mocked(heartbeatService.listTeamLogs).mockRejectedValue(new Error('owner unavailable'))
    const warning = vi.spyOn(console, 'warn').mockImplementation(() => {})
    try {
      renderDashboard(baseTeam, vi.fn())
      expect(await screen.findByText('Local logs unavailable.')).toBeInTheDocument()
      expect(screen.queryByText('0 local logs in 24h')).not.toBeInTheDocument()
      expect(screen.queryByText('No local logs.')).not.toBeInTheDocument()
    } finally {
      warning.mockRestore()
    }
  })

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

  it('shows the team open-work count and keeps the feed as the disposition link', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((input: RequestInfo | URL) => {
        const url = input instanceof Request ? input.url : String(input)
        if (url !== '/embedded/swarm-manager/api/v1/backlog?archived=false') throw new Error(`Unexpected request: ${url}`)
        return Promise.resolve(new Response(
          JSON.stringify({
            items: [
              { status: 'backlog', tags: ['scenario-qa'] },
              { status: 'done', tags: ['scenario-qa'] },
              { status: 'backlog', tags: ['other-team'] },
            ],
          }),
          { status: 200 },
        ))
      }),
    )
    const onUpdate = vi.fn().mockResolvedValue(undefined)
    renderDashboard(baseTeam, onUpdate)

    expect(await screen.findByText(/1 open work item\./)).toBeDefined()
    expect(screen.getByRole('link', { name: /Open work feed/ })).toHaveAttribute('href', '/swarm-manager')
  })
})
