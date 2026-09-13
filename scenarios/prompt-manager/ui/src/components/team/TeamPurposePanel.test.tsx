import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor, within } from '@/test-utils/renderWithProviders'
import { buildDefaultCreateTeamRequest } from '@/lib/schemas'
import type { TeamDetails } from '@/types/team'
import { TeamPurposePanel } from './TeamPurposePanel'

vi.mock('@/components/markdown/MarkdownRenderer', () => ({
  MarkdownRenderer: ({ content }: { content: string }) => <div>{content}</div>,
}))

function makeTeam(overrides: Partial<TeamDetails> = {}): TeamDetails {
  return {
    ...buildDefaultCreateTeamRequest('Quality team'),
    id: 'quality', displayName: 'Quality team', enabled: true,
    purpose: 'domain-stewardship', lifetime: 'standing', effortRefs: ['effort:quality'],
    memberCount: 0, members: [], roles: [],
    createdAt: '2026-09-12T00:00:00Z', updatedAt: '2026-09-12T00:00:00Z',
    ...overrides,
  }
}

describe('TeamPurposePanel', () => {
  afterEach(() => { vi.unstubAllGlobals() })

  it.each([
    { field: 'Team purpose', value: 'delivery', expected: { purpose: 'delivery', lifetime: 'standing' } },
    { field: 'Team lifetime', value: 'finite', expected: { purpose: 'domain-stewardship', lifetime: 'finite' } },
  ])('saves $field independently without changing execution or authority', async ({ field, value, expected }) => {
    const onUpdate = vi.fn().mockResolvedValue(undefined)
    render(<TeamPurposePanel team={makeTeam()} onUpdate={onUpdate} />)
    expect(screen.getByRole('button', { name: 'Save team details' })).toBeDisabled()

    fireEvent.change(screen.getByRole('combobox', { name: field }), { target: { value } })
    expect(screen.getByRole('combobox', { name: 'Team purpose' })).toHaveValue(expected.purpose)
    expect(screen.getByRole('combobox', { name: 'Team lifetime' })).toHaveValue(expected.lifetime)
    fireEvent.click(screen.getByRole('button', { name: 'Save team details' }))

    await waitFor(() => expect(onUpdate).toHaveBeenCalledTimes(1))
    expect(onUpdate).toHaveBeenCalledWith({ ...expected, effortRefs: ['effort:quality'] })
    expect(await screen.findByRole('status')).toHaveTextContent('Team details saved.')
    expect(screen.getByRole('combobox', { name: field })).toHaveValue(value)
  })

  it('clears classification and effort references explicitly', async () => {
    const onUpdate = vi.fn().mockResolvedValue(undefined)
    render(<TeamPurposePanel team={makeTeam()} onUpdate={onUpdate} />)
    fireEvent.change(screen.getByRole('combobox', { name: 'Team purpose' }), { target: { value: '' } })
    fireEvent.change(screen.getByRole('combobox', { name: 'Team lifetime' }), { target: { value: '' } })
    fireEvent.click(screen.getByText('Linked effort references (1)'))
    fireEvent.change(screen.getByRole('textbox', { name: 'Linked effort references' }), { target: { value: '' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save team details' }))

    await waitFor(() => expect(onUpdate).toHaveBeenCalledTimes(1))
    expect(onUpdate).toHaveBeenCalledWith({ purpose: '', lifetime: '', effortRefs: [] })
    expect(await screen.findByRole('status')).toHaveTextContent('Team details saved.')
    expect(screen.getByRole('combobox', { name: 'Team purpose' })).toHaveValue('')
    expect(screen.getByRole('combobox', { name: 'Team lifetime' })).toHaveValue('')
    expect(screen.getByRole('textbox', { name: 'Linked effort references' })).toHaveValue('')
  })

  it('retains unsaved values after refusal and a background team refresh, then retries the same edit', async () => {
    const team = makeTeam()
    const onUpdate = vi.fn().mockRejectedValueOnce(new Error('Team revision changed')).mockResolvedValueOnce(undefined)
    const view = render(<TeamPurposePanel team={team} onUpdate={onUpdate} />)
    fireEvent.change(screen.getByRole('combobox', { name: 'Team purpose' }), { target: { value: 'supervision' } })
    fireEvent.change(screen.getByRole('combobox', { name: 'Team lifetime' }), { target: { value: 'finite' } })
    fireEvent.click(screen.getByText('Linked effort references (1)'))
    fireEvent.change(screen.getByRole('textbox', { name: 'Linked effort references' }), { target: { value: ' effort:new\n\neffort:new\neffort:second ' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save team details' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Team revision changed')
    view.rerender(<TeamPurposePanel team={{ ...team, effortRefs: ['effort:external-refresh'] }} onUpdate={onUpdate} />)
    expect(screen.getByRole('combobox', { name: 'Team purpose' })).toHaveValue('supervision')
    expect(screen.getByRole('combobox', { name: 'Team lifetime' })).toHaveValue('finite')
    expect(screen.getByRole('textbox', { name: 'Linked effort references' })).toHaveValue(' effort:new\n\neffort:new\neffort:second ')
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Save team details' }))

    await waitFor(() => expect(onUpdate).toHaveBeenCalledTimes(2))
    expect(onUpdate.mock.calls[1]?.[0]).toEqual({ purpose: 'supervision', lifetime: 'finite', effortRefs: ['effort:new', 'effort:second'] })
    expect(await screen.findByRole('status')).toHaveTextContent('Team details saved.')
  })

  it('shows objectives and declared contract boundaries without inferring grants from labels', () => {
    const team = makeTeam({
      purpose: 'delivery', lifetime: 'finite',
      objectivesServed: [{ id: 'objective:quality', role: 'contributor', coverage: 'partial', note: 'Measure release quality.', acknowledgedRevision: 'revision-7' }],
      members: [{ agentId: 'reviewer', displayName: 'Release reviewer', status: 'active', roles: [] }], memberCount: 1,
      operatingContract: {
        schemaVersion: 1,
        documents: { sharedState: [], planOfRecord: [{ id: 'charter', paths: [{ base: 'repo-root', path: 'docs/quality.md' }], writePolicy: 'operator-only' }] },
        knowledgeTopics: {},
        members: { reviewer: {
          lane: 'Propose release improvements', allowedWrites: [{ kind: 'knowledge', base: 'team-shared', path: 'findings' }],
          forbiddenWrites: [{ base: 'repo-root', path: 'scenarios/**' }], safetyCriticalRules: ['Implementation requires an accepted assignment.'],
          readOnlyModeBehavior: { stillWriteKnowledge: true, stillWriteHandoff: true },
        } },
      },
    })
    render(<TeamPurposePanel team={team} onUpdate={vi.fn()} />)
    expect(screen.getByText('objective:quality').closest('li')).toHaveTextContent('contributor · partial coverage')
    expect(screen.getByText('Measure release quality.')).toBeInTheDocument()
    expect(screen.getByText('Acknowledged revision: revision-7')).toBeInTheDocument()
    fireEvent.click(screen.getByText('Declared authority and source contracts'))
    expect(screen.getByText('Release reviewer')).toBeInTheDocument()
    expect(screen.getByText('Propose release improvements')).toBeInTheDocument()
    expect(screen.getByText('Allowed writes: knowledge: team-shared: findings')).toBeInTheDocument()
    expect(screen.getByText('Prohibited writes: repo-root: scenarios/**')).toBeInTheDocument()
    expect(screen.getByText('Implementation requires an accepted assignment.')).toBeInTheDocument()
    expect(screen.getByText('charter · operator-only')).toBeInTheDocument()
    expect(screen.getByText('repo-root: docs/quality.md')).toBeInTheDocument()
    expect(screen.getByText(/Current effort action grants appear with the effort observation below/)).toBeInTheDocument()
  })

  it('preserves unspecified labels and resolves the owner link only after opening the model guide', async () => {
    const fetchOwner = vi.fn(async (input: RequestInfo | URL) => {
      expect(String(input)).toMatch(/\/embedded\/agent-manager\/external-url$/)
      return new Response(JSON.stringify({ url: 'https://agents.example.test/apps/agent-manager/proxy/' }), { status: 200 })
    })
    vi.stubGlobal('fetch', fetchOwner)
    render(<TeamPurposePanel team={makeTeam({ purpose: undefined, lifetime: undefined })} onUpdate={vi.fn()} />)
    expect(screen.getByText('Lifetime unspecified')).toBeInTheDocument()
    expect(screen.getByText('Purpose unspecified')).toBeInTheDocument()
    expect(screen.getByText('No objective relationships declared.')).toBeInTheDocument()
    expect(fetchOwner).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: 'How teams and efforts work' }))
    const dialog = screen.getByRole('dialog', { name: 'Teams and efforts' })
    expect(within(dialog).getByText(/Effort Supervision is a standing team/)).toBeInTheDocument()
    expect(within(dialog).getByText(/Purpose, lifetime, and an enabled schedule do not grant additional permission/)).toBeInTheDocument()
    expect(await within(dialog).findByRole('link', { name: 'Open the effort board' })).toHaveAttribute('href', 'https://agents.example.test/apps/agent-manager/proxy/efforts')
    fireEvent.click(within(dialog).getByRole('button', { name: 'Close dialog' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('keeps the model guide usable when the owner address is unavailable', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{}', { status: 503 })))
    render(<TeamPurposePanel team={makeTeam()} onUpdate={vi.fn()} />)
    fireEvent.click(screen.getByRole('button', { name: 'How teams and efforts work' }))
    const dialog = screen.getByRole('dialog', { name: 'Teams and efforts' })
    expect(await within(dialog).findByText('The effort board address is unavailable.')).toBeInTheDocument()
    expect(within(dialog).queryByRole('link', { name: 'Open the effort board' })).not.toBeInTheDocument()
    expect(within(dialog).getByText(/Effort Supervision is a standing team/)).toBeInTheDocument()
  })
})
