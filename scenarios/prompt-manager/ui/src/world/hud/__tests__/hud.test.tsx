import { fireEvent, render, screen, waitFor } from '@/test-utils/renderWithProviders'
import { expectNoA11yViolations } from '@/test-utils/a11y'
import { describe, expect, it, vi } from 'vitest'
import { tuning } from '../../config'
import type { WorldActions } from '../../data'
import { createWorldStore } from '../../sim'
import { AgentCard } from '../AgentCard'
import { EMPTY_FILTERS, matchesFilters } from '../filterState'
import { WorldHud } from '../Hud'
import { SummaryStrip } from '../SummaryStrip'
import { TwoDMode } from '../TwoDMode'
import { formatCountdown, formatDuration } from '../format'

const NOW = 1_700_000_000

function makeStore(actors = 4) {
  const teams = [{ id: 'team-a', name: 'Alpha', memberIds: ['a1', 'a2'] }, { id: 'team-b', name: 'Beta', memberIds: ['b1', 'b2'] }]
  const agents = ['a1', 'a2', 'b1', 'b2'].slice(0, actors).map((id) => ({ id, name: id.toUpperCase(), skillCount: 3 }))
  return createWorldStore({ seed: 1, now: NOW, teams, agents, scene: 'office' }, tuning, 0)
}

type FakeActions = WorldActions & {
  runNow: ReturnType<typeof vi.fn<[string, string], Promise<{ runId?: string; message: string }>>>
  stop: ReturnType<typeof vi.fn<[string, string], Promise<void>>>
  acknowledgeFailure: ReturnType<typeof vi.fn<[string], undefined>>
}

function fakeActions(): FakeActions {
  return {
    runNow: vi.fn<[string, string], Promise<{ runId?: string; message: string }>>(() => Promise.resolve({ runId: 'run-1', message: 'accepted' })),
    stop: vi.fn<[string, string], Promise<void>>(() => Promise.resolve()),
    acknowledgeFailure: vi.fn<[string], undefined>(),
  }
}

function hudProps(store = makeStore(), overrides: Partial<Parameters<typeof WorldHud>[0]> = {}) {
  return {
    store,
    actions: fakeActions(),
    feed: { mode: 'stream' as const, lastEventAt: NOW, reconnects: 0, lastError: null },
    focusedId: null,
    onFocus: vi.fn(),
    onFocusTeam: vi.fn(),
    onHome: vi.fn(),
    following: false,
    onFollowChange: vi.fn(),
    filters: EMPTY_FILTERS,
    onFiltersChange: vi.fn(),
    summaryFilter: null,
    onSummaryFilterChange: vi.fn(),
    highlightedTeamId: null,
    onHighlightTeam: vi.fn(),
    twoD: false,
    onTwoDChange: vi.fn(),
    tickerLimit: 10,
    ...overrides,
  }
}

describe('format', () => {
  it('formats durations and countdowns', () => {
    expect(formatDuration(42)).toBe('42s')
    expect(formatDuration(185)).toBe('3m 05s')
    expect(formatDuration(4321)).toBe('1h 12m')
    expect(formatCountdown(NOW + 125, NOW)).toBe('T-02:05')
    expect(formatCountdown(NOW - 1, NOW)).toBe('now')
  })
})

describe('SummaryStrip', () => {
  it('renders counts and toggles filters', () => {
    const onToggle = vi.fn()
    const store = makeStore()
    store.dispatch([{ kind: 'run.started', agentId: 'a1', runId: 'r', at: NOW }])
    store.advance(tuning.sim.tickSeconds)
    render(<SummaryStrip summary={store.getView().summary} now={NOW} activeFilter={null} onToggleFilter={onToggle} teamNames={{}} />)
    expect(screen.getByTestId('world-hud-summary-running')).toHaveTextContent('1')
    expect(screen.getByTestId('world-hud-summary-idle')).toHaveTextContent('3')
    fireEvent.click(screen.getByTestId('world-hud-summary-failed'))
    expect(onToggle).toHaveBeenCalledWith('failed')
    expect(screen.getByTestId('world-hud-next-heartbeat')).toHaveTextContent('No heartbeat scheduled')
  })

  it('shows the next heartbeat countdown with the team name', () => {
    const store = makeStore()
    store.dispatch([{ kind: 'heartbeat.upcoming', teamId: 'team-b', scheduledAt: NOW + 600, at: NOW }])
    store.advance(tuning.sim.tickSeconds)
    render(<SummaryStrip summary={store.getView().summary} now={NOW} activeFilter="idle" onToggleFilter={vi.fn()} teamNames={{ 'team-b': 'Beta' }} />)
    expect(screen.getByTestId('world-hud-next-heartbeat')).toHaveTextContent('Beta')
    expect(screen.getByTestId('world-hud-next-heartbeat')).toHaveTextContent('T-10:00')
    expect(screen.getByTestId('world-hud-summary-idle')).toHaveAttribute('aria-pressed', 'true')
  })
})

describe('AgentCard', () => {
  it('runs, stops and acknowledges through the actions', async () => {
    const store = makeStore()
    const actions = fakeActions()
    const view = store.getView()
    const actor = view.actors[0]
    if (!actor) throw new Error('missing actor')
    const { rerender } = render(<AgentCard actor={actor} teamName="Alpha" now={NOW} actions={actions} following={false} onFollowChange={vi.fn()} onClose={vi.fn()} />)
    fireEvent.click(screen.getByTestId('world-hud-run-now'))
    await waitFor(() => expect(actions.runNow).toHaveBeenCalledWith('team-a', 'a1'))
    await screen.findByText('Run run-1 requested')
    expect(screen.getByTestId('world-hud-stop-run')).toBeDisabled()
    store.dispatch([{ kind: 'run.started', agentId: 'a1', runId: 'r', at: NOW }])
    store.advance(tuning.sim.tickSeconds)
    const running = store.getView().actors[0]
    if (!running) throw new Error('missing')
    rerender(<AgentCard actor={running} teamName="Alpha" now={NOW} actions={actions} following={false} onFollowChange={vi.fn()} onClose={vi.fn()} />)
    // Home is the desk: a member already at its desk starts working at once.
    expect(screen.getByTestId('world-hud-agent-state')).toHaveTextContent('Working')
    fireEvent.click(screen.getByTestId('world-hud-stop-run'))
    await waitFor(() => expect(actions.stop).toHaveBeenCalledWith('team-a', 'a1'))
    store.dispatch([{ kind: 'run.failed', agentId: 'a1', runId: 'r', error: 'exit 1', at: NOW }])
    store.advance(tuning.sim.tickSeconds)
    const failed = store.getView().actors[0]
    if (!failed) throw new Error('missing')
    rerender(<AgentCard actor={failed} teamName="Alpha" now={NOW} actions={actions} following={false} onFollowChange={vi.fn()} onClose={vi.fn()} />)
    expect(screen.getByText('exit 1')).toBeInTheDocument()
    fireEvent.click(screen.getByTestId('world-hud-acknowledge'))
    expect(actions.acknowledgeFailure).toHaveBeenCalledWith('a1')
  })

  it('reports a failed request instead of swallowing it', async () => {
    const store = makeStore()
    const actions = fakeActions()
    actions.runNow.mockImplementation(() => Promise.reject(new Error('agent-manager unreachable')))
    const actor = store.getView().actors[0]
    if (!actor) throw new Error('missing actor')
    render(<AgentCard actor={actor} now={NOW} actions={actions} following={false} onFollowChange={vi.fn()} onClose={vi.fn()} />)
    fireEvent.click(screen.getByTestId('world-hud-run-now'))
    await screen.findByText('agent-manager unreachable')
  })
})

describe('matchesFilters', () => {
  it('applies search, team, failed and summary filters', () => {
    const idle = { name: 'Alpha One', teamId: 'team-a', state: 'idle' }
    const failed = { name: 'Beta Two', teamId: 'team-b', state: 'failed' }
    expect(matchesFilters(idle, { ...EMPTY_FILTERS, search: 'alpha' }, null)).toBe(true)
    expect(matchesFilters(idle, { ...EMPTY_FILTERS, search: 'beta' }, null)).toBe(false)
    expect(matchesFilters(idle, { ...EMPTY_FILTERS, teamId: 'team-b' }, null)).toBe(false)
    expect(matchesFilters(failed, { ...EMPTY_FILTERS, onlyFailed: true }, null)).toBe(true)
    expect(matchesFilters(idle, EMPTY_FILTERS, 'failed')).toBe(false)
    expect(matchesFilters(idle, EMPTY_FILTERS, 'idle')).toBe(true)
  })
})

describe('WorldHud', () => {
  it('renders the strip, panel and ticker, and focuses from the team panel and ticker', () => {
    const store = makeStore()
    store.dispatch([{ kind: 'run.started', agentId: 'b1', runId: 'r9', at: NOW }])
    store.advance(tuning.sim.tickSeconds)
    const props = hudProps(store)
    render(<WorldHud {...props} />)
    expect(screen.getByTestId('world-hud-summary')).toBeInTheDocument()
    fireEvent.click(screen.getByTestId('world-hud-team-panel-team-b'))
    expect(props.onFocusTeam).toHaveBeenCalledWith('team-b')
    fireEvent.click(screen.getByText(/B1 started a run/))
    expect(props.onFocus).toHaveBeenCalledWith('b1')
    fireEvent.click(screen.getByTestId('world-hud-home'))
    expect(props.onHome).toHaveBeenCalled()
    expect(screen.getByTestId('world-hud-feed-status')).toHaveTextContent('feed: stream')
  })

  it('shows the agent card for the focused actor and closes it', () => {
    const props = hudProps(makeStore(), { focusedId: 'a2' })
    render(<WorldHud {...props} />)
    expect(screen.getByTestId('world-hud-agent-card')).toHaveTextContent('A2')
    fireEvent.click(screen.getByTestId('world-hud-agent-card-close'))
    expect(props.onFocus).toHaveBeenCalledWith(null)
  })

  it('2D mode lists every actor by team and offers the same actions', async () => {
    const store = makeStore()
    const props = hudProps(store, { twoD: true, focusedId: 'b2' })
    const { container } = render(<WorldHud {...props} />)
    expect(screen.getByTestId('world-hud-2d-mode')).toBeInTheDocument()
    expect(screen.getByTestId('world-hud-actor-list-a1')).toBeInTheDocument()
    fireEvent.click(screen.getByTestId('world-hud-actor-list-a1'))
    expect(props.onFocus).toHaveBeenCalledWith('a1')
    fireEvent.click(screen.getByTestId('world-hud-run-now'))
    await waitFor(() => expect(props.actions.runNow).toHaveBeenCalledWith('team-b', 'b2'))
    await expectNoA11yViolations(container)
  })

  it('filters the 2D list by search and summary filter', () => {
    const store = makeStore()
    render(<WorldHud {...hudProps(store, { twoD: true, filters: { ...EMPTY_FILTERS, search: 'b' } })} />)
    expect(screen.queryByTestId('world-hud-actor-list-a1')).not.toBeInTheDocument()
    expect(screen.getByTestId('world-hud-actor-list-b1')).toBeInTheDocument()
  })

  it('passes axe at a narrow width', async () => {
    const { container } = render(<WorldHud {...hudProps(makeStore(), { focusedId: 'a1' })} />)
    await expectNoA11yViolations(container)
  })
})

describe('TwoDMode', () => {
  it('groups unassigned agents under the commons', () => {
    const store = createWorldStore({ seed: 1, now: NOW, teams: [], agents: [{ id: 'solo', name: 'Solo' }], scene: 'office' }, tuning, 0)
    render(<TwoDMode actors={store.getView().actors} teams={[]} now={NOW} focusedId={null} onFocus={vi.fn()} />)
    expect(screen.getByText('Commons')).toBeInTheDocument()
    expect(screen.getByTestId('world-hud-actor-list-solo')).toBeInTheDocument()
  })
})
