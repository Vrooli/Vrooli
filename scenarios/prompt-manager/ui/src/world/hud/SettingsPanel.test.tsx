import { useState } from 'react'
import { fireEvent, renderWithProviders as render, screen } from '@/test-utils/renderWithProviders'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { WorldSettingsContent, type PeriodMode } from './SettingsPanel'
import { WorldClock } from '../config/clock'
import {
  useAcknowledgeObjective,
  useAddObjectiveRelation,
  useAttachObjective,
  useDeleteObjective,
  useDeleteObjectiveRelation,
  useDetachObjective,
  useObjectiveRelations,
  useObjectiveValidation,
  useObjectives,
  useReorderObjectives,
  useReorderTeamAttachments,
  useTeamAttachments,
  useUpdateAttachment,
  useUpsertObjective,
} from '@/services/objectiveService'

vi.mock('@/services/objectiveService', () => ({
  useObjectives: vi.fn(),
  useObjectiveRelations: vi.fn(),
  useObjectiveValidation: vi.fn(),
  useTeamAttachments: vi.fn(),
  useUpsertObjective: vi.fn(),
  useDeleteObjective: vi.fn(),
  useReorderObjectives: vi.fn(),
  useAttachObjective: vi.fn(),
  useUpdateAttachment: vi.fn(),
  useDetachObjective: vi.fn(),
  useReorderTeamAttachments: vi.fn(),
  useAcknowledgeObjective: vi.fn(),
  useAddObjectiveRelation: vi.fn(),
  useDeleteObjectiveRelation: vi.fn(),
}))

function query<T>(data: T) {
  return { data, isLoading: false, isError: false, refetch: vi.fn() }
}
const mutation = () => ({ mutateAsync: vi.fn().mockResolvedValue(undefined) })

beforeEach(() => {
  vi.mocked(useObjectives).mockReturnValue(query([]) as never)
  vi.mocked(useObjectiveRelations).mockReturnValue(query([]) as never)
  vi.mocked(useObjectiveValidation).mockReturnValue(query(undefined) as never)
  vi.mocked(useTeamAttachments).mockReturnValue(query({ attachments: [], attachmentRevision: '' }) as never)
  vi.mocked(useUpsertObjective).mockReturnValue(mutation() as never)
  vi.mocked(useDeleteObjective).mockReturnValue(mutation() as never)
  vi.mocked(useReorderObjectives).mockReturnValue(mutation() as never)
  vi.mocked(useAttachObjective).mockReturnValue(mutation() as never)
  vi.mocked(useUpdateAttachment).mockReturnValue(mutation() as never)
  vi.mocked(useDetachObjective).mockReturnValue(mutation() as never)
  vi.mocked(useReorderTeamAttachments).mockReturnValue(mutation() as never)
  vi.mocked(useAcknowledgeObjective).mockReturnValue(mutation() as never)
  vi.mocked(useAddObjectiveRelation).mockReturnValue(mutation() as never)
  vi.mocked(useDeleteObjectiveRelation).mockReturnValue(mutation() as never)
})


function mount(clock?: WorldClock) {
  const changed = vi.fn()
  function Harness() {
    const [seed, setSeed] = useState(1)
    const [periodMode, setPeriodMode] = useState<PeriodMode>({ kind: 'fixed', period: 'night' })
    return <WorldSettingsContent worldClock={clock} seed={seed} onSeedChange={next => { changed(next); setSeed(next) }} weather="auto" onWeatherChange={() => {}}
      sceneId="park" onSceneChange={() => {}} quality={{ auto: false, profileId: 'high' }}
      onPickProfile={() => {}} onAutoChange={() => {}} periodMode={periodMode}
      onPeriodModeChange={setPeriodMode} showDiagnostics={false} onShowDiagnosticsChange={() => {}}
      onCameraHome={() => {}} zoomTarget="cursor" onZoomTargetChange={() => {}} />
  }
  render(<Harness />)
  return { changed, field: screen.getByRole('spinbutton', { name: 'World seed' }) }
}

describe('world seed settings', () => {
  it('plays from the selected instant, freezes there, and explicitly returns to wall time', () => {
    let wall = 1000000
    const clock = new WorldClock(() => wall, 'UTC'); clock.fix(10000)
    mount(clock)
    fireEvent.click(screen.getByRole('button', { name: 'Play from here' }))
    expect(screen.getByText('Presentation time: Playing from selected time')).toBeInTheDocument()
    expect(screen.getByRole('radio', { name: 'Night' })).toHaveAttribute('aria-checked', 'true')
    expect(clock.snapshot().utcMilliseconds).toBe(10000)
    wall += 2000
    expect(clock.snapshot().utcMilliseconds).toBe(12000)
    fireEvent.click(screen.getByRole('button', { name: 'Freeze time' }))
    wall += 10000
    expect(clock.snapshot().utcMilliseconds).toBe(12000)
    expect(screen.getByText('Presentation time: Frozen')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Resume live time' }))
    expect(clock.snapshot().utcMilliseconds).toBe(wall)
    expect(screen.getByText('Presentation time: Live')).toBeInTheDocument()
  })
  it('applies explicitly, retains keyboard focus, and avoids duplicate generation requests', () => {
    const { changed, field } = mount()
    field.focus()
    fireEvent.change(field, { target: { value: '42' } })
    fireEvent.blur(field)
    expect(changed).not.toHaveBeenCalled()
    field.focus()
    fireEvent.keyDown(field, { key: 'Enter' })
    expect(changed).toHaveBeenCalledWith(42)
    expect(field).toHaveFocus()
    expect(field).toHaveValue(42)
    fireEvent.click(screen.getByRole('button', { name: 'Apply seed' }))
    expect(changed).toHaveBeenCalledTimes(1)
  })

  it.each(['', '-1', '1.5', '4294967296'])('rejects invalid seed %j without applying it', raw => {
    const { changed, field } = mount()
    fireEvent.change(field, { target: { value: raw } })
    fireEvent.click(screen.getByRole('button', { name: 'Apply seed' }))
    expect(changed).not.toHaveBeenCalled()
    expect(field).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByRole('alert')).toHaveTextContent('Enter a whole number')
    fireEvent.change(field, { target: { value: '4294967295' } })
    fireEvent.keyDown(field, { key: 'Enter' })
    expect(changed).toHaveBeenCalledWith(4294967295)
    expect(field).toHaveAttribute('aria-invalid', 'false')
  })
})

it('renders one group at a time without duplicating the form', () => {
  function GroupHarness() {
    const [seed, setSeed] = useState(1)
    return <WorldSettingsContent
      group="advanced"
      seed={seed} onSeedChange={setSeed} weather="auto" onWeatherChange={() => {}}
      sceneId="park" onSceneChange={() => {}} quality={{ auto: false, profileId: 'high' }}
      onPickProfile={() => {}} onAutoChange={() => {}} periodMode={{ kind: 'fixed', period: 'night' }}
      onPeriodModeChange={() => {}} showDiagnostics={false} onShowDiagnosticsChange={() => {}}
      onCameraHome={() => {}} zoomTarget="cursor" onZoomTargetChange={() => {}} />
  }
  render(<GroupHarness />)
  expect(screen.queryByRole('spinbutton', { name: 'World seed' })).not.toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Reset camera' })).not.toBeInTheDocument()
  expect(screen.getByText('Show diagnostics')).toBeInTheDocument()
})

it('mounts the shared objective editor in the management group with a team context selector', () => {
  render(<WorldSettingsContent
    group="management"
    teams={[{ id: 'quality', label: 'Quality team' }, { id: 'delivery', label: 'Delivery team' }]}
    seed={1} onSeedChange={() => {}} weather="auto" onWeatherChange={() => {}}
    sceneId="park" onSceneChange={() => {}} quality={{ auto: false, profileId: 'high' }}
    onPickProfile={() => {}} onAutoChange={() => {}} periodMode={{ kind: 'fixed', period: 'night' }}
    onPeriodModeChange={() => {}} showDiagnostics={false} onShowDiagnosticsChange={() => {}}
    onCameraHome={() => {}} zoomTarget="cursor" onZoomTargetChange={() => {}} />)

  expect(screen.queryByText(/appears here once the shared objective editor is available/)).not.toBeInTheDocument()
  expect(screen.getByRole('heading', { name: 'Objectives' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'New objective' })).toBeInTheDocument()
  expect(useTeamAttachments).toHaveBeenCalledWith('')

  fireEvent.change(screen.getByLabelText('Objective management scope'), { target: { value: 'quality' } })
  expect(screen.getByRole('heading', { name: 'Objective commitments' })).toBeInTheDocument()
  expect(useTeamAttachments).toHaveBeenLastCalledWith('quality')
})

it('offers deep night outside the workbench and freezes the selected local time', () => {
  const clock = new WorldClock(() => Date.parse('2026-09-08T16:00:00Z'), 'America/New_York')
  mount(clock)
  expect(screen.queryByText('World workbench', { exact: true })).not.toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'Deep night' }))
  expect(clock.snapshot().localMinutes).toBe(150)
  expect(screen.getByRole('radio', { name: 'Clock' })).toHaveAttribute('aria-checked', 'true')
  expect(clock.snapshot().timeScale).toBe(0)
  fireEvent.click(screen.getByRole('button', { name: 'Midday' }))
  expect(clock.snapshot().localMinutes).toBe(720)
})
