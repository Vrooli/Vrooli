import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor, within } from '@/test-utils/renderWithProviders'
import { buildDefaultCreateTeamRequest } from '@/lib/schemas'
import type { Team, TeamDetails } from '@/types/team'
import * as teamService from '@/services/teamService'
import { TeamListPanel } from './TeamListPanel'

vi.mock('@/services/teamService', () => ({ getTeams: vi.fn(), createTeam: vi.fn(), exportClaudeCodeTeam: vi.fn() }))
vi.mock('@/stores/graphStore', () => ({ useGraphStore: { getState: () => ({ fetchHealthScores: vi.fn() }) } }))
vi.mock('@/components/markdown/MarkdownRenderer', () => ({ MarkdownRenderer: () => null }))
vi.mock('./CCTeamImportModal', () => ({ CCTeamImportModal: () => null }))
vi.mock('./TeamEffortsPanel', () => ({
  TeamEffortsPanel: ({ teams, teamRegistryComplete, onOpenTeam }: {
    teams: Team[]; teamRegistryComplete: boolean; onOpenTeam: (id: string) => void
  }) => <section aria-label="Observed efforts">
    <p>{teams.length} registered teams · registry {teamRegistryComplete ? 'available' : 'unavailable'}</p>
    <button onClick={() => onOpenTeam('launch')}>Open linked delivery team</button>
  </section>,
}))

function makeTeam(id: string, displayName: string, overrides: Partial<TeamDetails> = {}): TeamDetails {
  return {
    ...buildDefaultCreateTeamRequest(displayName),
    id, displayName, enabled: true, memberCount: 0, members: [], roles: [],
    createdAt: '2026-09-12T00:00:00Z', updatedAt: '2026-09-12T00:00:00Z',
    ...overrides,
  }
}

const teams = [
  makeTeam('marketing', 'Marketing', { purpose: 'domain-stewardship', lifetime: 'standing' }),
  makeTeam('launch', 'Launch delivery', { purpose: 'delivery', lifetime: 'finite' }),
  makeTeam('supervisor', 'Effort supervisor', { purpose: 'supervision', lifetime: 'standing' }),
  makeTeam('legacy', 'Legacy committee'),
  makeTeam('continuing-delivery', 'Continuing delivery', { purpose: 'delivery', lifetime: 'standing' }),
]

describe('TeamListPanel purpose and lifetime controls', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(teamService.getTeams).mockResolvedValue(teams)
    vi.mocked(teamService.createTeam).mockResolvedValue(makeTeam('created', 'Created team'))
  })

  it('combines independent purpose/lifetime filters and preserves unspecified teams', async () => {
    render(<TeamListPanel selectedTeamId={null} onSelectTeam={vi.fn()} />)
    expect(await screen.findByRole('button', { name: /Legacy committee/ })).toBeInTheDocument()
    const legacy = screen.getByRole('button', { name: /Legacy committee/ })
    expect(within(legacy).getByText('Lifetime unspecified')).toBeInTheDocument()
    expect(within(legacy).getByText('Purpose unspecified')).toBeInTheDocument()

    fireEvent.change(screen.getByRole('combobox', { name: 'Filter teams by purpose' }), { target: { value: 'delivery' } })
    expect(screen.getByRole('button', { name: /Launch delivery/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Continuing delivery/ })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Marketing/ })).not.toBeInTheDocument()
    fireEvent.change(screen.getByRole('combobox', { name: 'Filter teams by lifetime' }), { target: { value: 'standing' } })
    expect(screen.queryByRole('button', { name: /Launch delivery/ })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Continuing delivery/ })).toBeInTheDocument()

    fireEvent.change(screen.getByRole('combobox', { name: 'Filter teams by purpose' }), { target: { value: 'unspecified' } })
    fireEvent.change(screen.getByRole('combobox', { name: 'Filter teams by lifetime' }), { target: { value: 'unspecified' } })
    expect(screen.getByRole('button', { name: /Legacy committee/ })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Continuing delivery/ })).not.toBeInTheDocument()
  })

  it('searches the displayed classification without assigning missing classification', async () => {
    const onSelectTeam = vi.fn()
    const view = render(<TeamListPanel selectedTeamId={null} onSelectTeam={onSelectTeam} searchQuery="standing" />)
    expect(await screen.findByRole('button', { name: /Marketing/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Effort supervisor/ })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Legacy committee/ })).not.toBeInTheDocument()
    view.rerender(<TeamListPanel selectedTeamId={null} onSelectTeam={onSelectTeam} searchQuery="unspecified" />)
    fireEvent.click(screen.getByRole('button', { name: /Legacy committee/ }))
    expect(onSelectTeam).toHaveBeenCalledWith('legacy')
    expect(teamService.createTeam).not.toHaveBeenCalled()
  })

  it.each([
    { preset: 'domain', purpose: 'domain-stewardship', lifetime: 'standing' },
    { preset: 'delivery', purpose: 'delivery', lifetime: 'finite' },
    { preset: 'supervision', purpose: 'supervision', lifetime: 'standing' },
    { preset: 'custom', purpose: '', lifetime: '' },
  ])('creates the $preset preset through the team mutation', async ({ preset, purpose, lifetime }) => {
    const onSelectTeam = vi.fn()
    render(<TeamListPanel selectedTeamId={null} onSelectTeam={onSelectTeam} />)
    const selector = await screen.findByRole('combobox', { name: 'New team preset' })
    fireEvent.change(selector, { target: { value: preset } })
    fireEvent.click(screen.getByRole('button', { name: 'New Team' }))

    await waitFor(() => expect(teamService.createTeam).toHaveBeenCalledTimes(1))
    expect(teamService.createTeam).toHaveBeenCalledWith(expect.objectContaining({ purpose, lifetime }))
    const request = vi.mocked(teamService.createTeam).mock.calls[0]?.[0]
    expect(request).not.toHaveProperty('enabled')
    await waitFor(() => expect(onSelectTeam).toHaveBeenCalledWith('created'))
  })

  it('shows creation failure, retains the chosen preset, and permits retry', async () => {
    vi.mocked(teamService.createTeam).mockRejectedValueOnce(new Error('Team creation unavailable'))
    const onSelectTeam = vi.fn()
    render(<TeamListPanel selectedTeamId={null} onSelectTeam={onSelectTeam} />)
    fireEvent.change(await screen.findByRole('combobox', { name: 'New team preset' }), { target: { value: 'supervision' } })
    fireEvent.click(screen.getByRole('button', { name: 'New Team' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Team creation unavailable')
    expect(screen.getByRole('combobox', { name: 'New team preset' })).toHaveValue('supervision')
    expect(onSelectTeam).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: 'New Team' }))
    await waitFor(() => expect(onSelectTeam).toHaveBeenCalledWith('created'))
    expect(teamService.createTeam).toHaveBeenCalledTimes(2)
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('opens all observed efforts using the full registry even while filtering, then opens a linked team', async () => {
    const onSelectTeam = vi.fn()
    render(<TeamListPanel selectedTeamId={null} onSelectTeam={onSelectTeam} />)
    fireEvent.change(await screen.findByRole('combobox', { name: 'Filter teams by purpose' }), { target: { value: 'supervision' } })
    expect(screen.queryByRole('region', { name: 'Observed efforts' })).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Efforts and orchestration' }))
    const dialog = screen.getByRole('dialog', { name: 'Efforts and orchestration' })
    expect(within(dialog).getByText('5 registered teams · registry available')).toBeInTheDocument()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Open linked delivery team' }))
    expect(onSelectTeam).toHaveBeenCalledWith('launch')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(teamService.createTeam).not.toHaveBeenCalled()
  })
})
