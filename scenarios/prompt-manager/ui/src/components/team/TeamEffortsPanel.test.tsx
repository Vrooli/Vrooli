import { create } from '@bufbuild/protobuf'
import { timestampFromDate } from '@bufbuild/protobuf/wkt'
import { EffortBoardSchema } from '@vrooli/proto-types/agent-manager/v1/domain/effort_pb'
import { WatchActionKind } from '@vrooli/proto-types/agent-manager/v1/domain/watch_pb'
import { act, fireEvent, render, screen, within } from '@/test-utils/renderWithProviders'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { TeamEffortsPanel } from './TeamEffortsPanel'
import type { EffortObservations } from '@/services/effortService'

const owner = vi.hoisted(() => ({ data: undefined as EffortObservations | undefined, isPending: false, isFetching: false, error: null as Error | null, refetch: vi.fn(), calls: [] as Array<{ pageToken: string; effortRefs?: string[] }> }))
vi.mock('@/services/effortService', async () => ({
  ...await vi.importActual<typeof import('@/services/effortService')>('@/services/effortService'),
  useEffortObservations: (pageToken: string, effortRefs?: string[]) => { owner.calls.push({ pageToken, effortRefs }); return owner },
  useAgentManagerUrl: () => ({ data: 'https://example.test/apps/agent-manager/proxy' }),
}))
const board = () => create(EffortBoardSchema, { partial: true, rows: [{
  enrollment: { effortRef: 'effort:delivery/one', displayName: 'Finite delivery' }, runtimeState: 'completed',
  outcomeStanding: { state: 'unverified', attribution: 'workspace self-report' },
  nextAction: 'Wait for acceptance review',
  assignments: [{ subject: { role: 'orchestrator', runId: 'run-one' }, requestedModel: 'requested-model', effectiveModel: 'actual-model' }],
}] })
beforeEach(() => { owner.data = undefined; owner.isPending = false; owner.isFetching = false; owner.error = null; owner.refetch.mockReset(); owner.calls = [] })
afterEach(() => { vi.useRealTimers() })

test('a complete authored team list does not establish whether a runtime leader binding exists', () => {
  owner.data = { boards: [board()], unavailable: [] }
  const view = render(<TeamEffortsPanel teams={[]} />)
  expect(screen.getByText(/Authored team relationships unknown/)).toBeInTheDocument()
  expect(screen.queryByText(/No registered team binding/)).toBeNull()
  view.rerender(<TeamEffortsPanel teams={[]} teamRegistryComplete />)
  expect(screen.getByText(/No authored team relationship in the registered team list/)).toBeInTheDocument()
  expect(screen.getByText(/Runtime leader binding coverage unknown/)).toBeInTheDocument()
  expect(screen.queryByText(/No registered team binding/)).toBeNull()
  const link = screen.getByRole('link', { name: 'Open effort, acceptance evidence and all assignments' })
  expect(new URL(link.getAttribute('href')!).searchParams.get('effortRef')).toBe('effort:delivery/one')
})

test('a known finite leader binding is shown independently from authored links and supervisor observation', () => {
  owner.data = { boards: [board()], unavailable: [] }
  const view = render(<TeamEffortsPanel team={{ id: 'delivery', displayName: 'Delivery' }} leaderEffortRefs={['effort:delivery/one']} />)
  expect(owner.calls[owner.calls.length - 1]?.effortRefs).toEqual(['effort:delivery/one'])
  expect(screen.getByText(/Finite leader binding for this team/)).toBeInTheDocument()
  expect(screen.queryByText(/Observed by this team/)).toBeNull()
  expect(screen.queryByText(/Runtime leader binding coverage unknown/)).toBeNull()
  view.rerender(<TeamEffortsPanel team={{ id: 'delivery', displayName: 'Delivery', effortRefs: ['effort:delivery/one'] }} leaderEffortRefs={['effort:delivery/one']} observedEffortRefs={['effort:delivery/one']} />)
  expect(screen.getByText(/Authored team relationship:/)).toHaveTextContent('Delivery')
  expect(screen.getByText(/Finite leader binding for this team/)).toBeInTheDocument()
  expect(screen.getByText(/Observed by this team/)).toBeInTheDocument()
  expect(owner.calls[owner.calls.length - 1]?.effortRefs).toEqual(['effort:delivery/one'])
})

test('linked delivery and observation relationships keep schedule, execution, acceptance and models separate', () => {
  const data = board()
  data.rows[0]!.lastAssessment = create(EffortBoardSchema, { rows: [{ lastAssessment: { assessmentId: 'receipt-one', disposition: 'quiet', rationale: 'Waiting is justified', supervisorRunId: 'supervisor-one' } }] }).rows[0]!.lastAssessment
  owner.data = { boards: [data], unavailable: [] }
  const open = vi.fn()
  render(<TeamEffortsPanel team={{ id: 'supervision', displayName: 'Supervision', purpose: 'supervision' }} teams={[{ id: 'delivery', displayName: 'Delivery team', effortRefs: ['effort:delivery/one'] }]} observedEffortRefs={['effort:delivery/one']} observationAvailable scheduled={{ enabled: true, summary: 'hourly' }} onOpenTeam={open} />)
  expect(owner.calls[owner.calls.length - 1]?.effortRefs).toEqual(['effort:delivery/one'])
  expect(screen.getByText(/Enabled · hourly/)).toBeInTheDocument()
  expect(screen.getByText('completed')).toBeInTheDocument()
  expect(screen.getByText('unverified · workspace self-report')).toBeInTheDocument()
  expect(screen.getByText(/Requested model: requested-model · effective model: actual-model/)).toBeInTheDocument()
  expect(screen.getByText('Receipt receipt-one')).toBeInTheDocument()
  expect(screen.getByText(/Observed by this team/)).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'Delivery team' }))
  expect(open).toHaveBeenCalledWith('delivery')
  expect(screen.getByRole('link', { name: 'Open orchestrator run' })).toHaveAttribute('href', 'https://example.test/apps/agent-manager/proxy/runs/run-one')
})

test('a supervisor purpose alone never infers supervision of every discovered effort', () => {
  render(<TeamEffortsPanel team={{ id: 'supervision', displayName: 'Supervision', purpose: 'supervision' }} />)
  expect(owner.calls[owner.calls.length - 1]?.effortRefs).toEqual([])
  expect(screen.getByText(/Supervisor observation mapping unavailable/)).toBeInTheDocument()
  expect(screen.getByText(/This does not establish that the team has no work/)).toBeInTheDocument()
})

test('an owner supervision read failure remains visible for an unclassified legacy team', () => {
  render(<TeamEffortsPanel team={{ id: 'legacy', displayName: 'Legacy' }} observationAvailable={false} observationError="lead: cannot read supervision reservation" />)
  expect(screen.getByText(/Supervisor observation mapping unavailable/)).toHaveTextContent('lead: cannot read supervision reservation')
})

test('linked observations are paged before issuing requests and global pages retain owner cursors', () => {
  owner.data = { boards: [board()], unavailable: [] }
  const view = render(<TeamEffortsPanel team={{ id: 'delivery', displayName: 'Delivery', effortRefs: ['1', '2', '3', '4', '5', '6'] }} />)
  expect(owner.calls[owner.calls.length - 1]?.effortRefs).toEqual(['1', '2', '3', '4', '5'])
  fireEvent.click(screen.getByRole('button', { name: 'Next efforts' }))
  expect(owner.calls[owner.calls.length - 1]?.effortRefs).toEqual(['6'])
  const data = board(); data.nextPageToken = 'opaque-owner-cursor'; owner.data = { boards: [data], unavailable: [] }
  view.rerender(<TeamEffortsPanel />)
  fireEvent.click(screen.getByRole('button', { name: 'Next efforts' }))
  expect(owner.calls[owner.calls.length - 1]?.pageToken).toBe('opaque-owner-cursor')
})

test('owner outage and partial failures remain unknown, while successful empty pages retain discovery caveats', () => {
  owner.error = new Error('offline')
  const view = render(<TeamEffortsPanel />)
  expect(screen.getByRole('alert')).toHaveTextContent('Effort count and state are unknown')
  expect(screen.queryByText(/No efforts returned/)).toBeNull()
  owner.error = null; owner.data = { boards: [], unavailable: [{ effortRef: 'missing', reason: 'owner unavailable' }] }
  view.rerender(<TeamEffortsPanel />)
  expect(screen.getByRole('alert')).toHaveTextContent('their state is unknown')
  owner.data = { boards: [create(EffortBoardSchema, { partial: true })], unavailable: [] }
  view.rerender(<TeamEffortsPanel />)
  expect(screen.getByText(/No efforts returned on this page. Check discovery coverage/)).toBeInTheDocument()
})

test('a cached grant changes to expired at its expiry without another owner request', () => {
  vi.useFakeTimers(); vi.setSystemTime(new Date('2026-09-12T15:00:00Z'))
  const data = board()
  Object.assign(data.rows[0]!.enrollment!, { authorizedBy: 'operator', permittedActions: [WatchActionKind.CONTINUE], authorityExpiresAt: timestampFromDate(new Date('2026-09-12T15:00:01Z')) })
  owner.data = { boards: [data], unavailable: [] }
  render(<TeamEffortsPanel />)
  const article = within(screen.getByRole('article'))
  expect(article.getByText(/Recorded grant: continue/)).toBeInTheDocument()
  act(() => { vi.advanceTimersByTime(1002) })
  expect(article.queryByText(/Recorded grant:/)).toBeNull()
  expect(article.getByText(/Expired.*steering unavailable/)).toBeInTheDocument()
})
