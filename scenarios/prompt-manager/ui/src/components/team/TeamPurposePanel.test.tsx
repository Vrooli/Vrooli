import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor, within } from '@/test-utils/renderWithProviders'
import { buildDefaultCreateTeamRequest } from '@/lib/schemas'
import type { TeamDetails } from '@/types/team'
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
import { TeamPurposePanel } from './TeamPurposePanel'

vi.mock('@/components/markdown/MarkdownRenderer', () => ({
  MarkdownRenderer: ({ content }: { content: string }) => <div>{content}</div>,
}))

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

  it('mounts the shared objective editor with the team authority attachments', () => {
    // The display must come from the objective authority, never from the team
    // declaration, so the team carries no objectivesServed at all here.
    vi.mocked(useObjectives).mockReturnValue(query([
      { id: 'objective:quality', title: 'Release quality', class: 'terminal', evidenceSource: 'release board', hasEvidence: true, gapMarker: '', globalOrder: 0, meaningRevision: 'rev-1' },
    ]) as never)
    vi.mocked(useTeamAttachments).mockReturnValue(query({
      attachments: [{
        objectiveId: 'objective:quality', teamId: 'quality', role: 'supporting', coverage: 'partial',
        note: 'Measure release quality.', priority: 0, acknowledgedRevision: 'revision-7',
        attachmentRevision: 'arev-1', restatementPending: true,
      }],
      attachmentRevision: 'trev-1',
    }) as never)
    const team = makeTeam({
      purpose: 'delivery', lifetime: 'finite',
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

    // The team container reuses the canonical editor rather than a private list.
    const editor = screen.getByRole('region', { name: 'Objective authority' })
    expect(within(editor).getByText('Objective commitments')).toBeInTheDocument()
    expect(within(editor).getByText('Release quality')).toBeInTheDocument()
    expect(within(editor).getByText(/objective:quality · Terminal end · Supporting · Partial coverage/)).toBeInTheDocument()
    expect(within(editor).getByText('Measure release quality.')).toBeInTheDocument()
    expect(within(editor).getByText('Meaning changed; acknowledgement pending.')).toBeInTheDocument()

    fireEvent.click(screen.getByText('Advanced operating settings'))
    expect(screen.getByText('Release reviewer')).toBeInTheDocument()
    expect(screen.getByText('Propose release improvements')).toBeInTheDocument()
    expect(screen.getByText('Allowed writes: knowledge: team-shared: findings')).toBeInTheDocument()
    expect(screen.getByText('Prohibited writes: repo-root: scenarios/**')).toBeInTheDocument()
    expect(screen.getByText('Implementation requires an accepted assignment.')).toBeInTheDocument()
    expect(screen.getByText('charter · operator-only')).toBeInTheDocument()
    expect(screen.getByText('repo-root: docs/quality.md')).toBeInTheDocument()
    expect(screen.getByText(/Current effort action grants appear with the effort observation below/)).toBeInTheDocument()
  })

  it('persists team objective ordering through the canonical service', async () => {
    const reorder = vi.fn().mockResolvedValue(undefined)
    vi.mocked(useReorderTeamAttachments).mockReturnValue({ mutateAsync: reorder } as never)
    vi.mocked(useObjectives).mockReturnValue(query([
      { id: 'objective:one', title: 'First end', class: 'terminal', evidenceSource: '', hasEvidence: true, gapMarker: '', globalOrder: 0, meaningRevision: 'r1' },
      { id: 'objective:two', title: 'Second end', class: 'instrumental', evidenceSource: '', hasEvidence: true, gapMarker: '', globalOrder: 1, meaningRevision: 'r1' },
    ]) as never)
    vi.mocked(useTeamAttachments).mockReturnValue(query({
      attachments: [
        { objectiveId: 'objective:one', teamId: 'quality', role: 'primary', coverage: 'full', priority: 0, attachmentRevision: 'a1', restatementPending: false },
        { objectiveId: 'objective:two', teamId: 'quality', role: 'supporting', coverage: 'partial', priority: 1, attachmentRevision: 'a1', restatementPending: false },
      ],
      attachmentRevision: 'trev-9',
    }) as never)
    render(<TeamPurposePanel team={makeTeam()} onUpdate={vi.fn()} />)

    fireEvent.click(screen.getByRole('button', { name: 'Move Second end up' }))
    await waitFor(() => expect(reorder).toHaveBeenCalledWith({
      teamId: 'quality', objectiveIds: ['objective:two', 'objective:one'], expectedTeamRevision: 'trev-9',
    }))
  })

  it('preserves unspecified labels and resolves the owner link only after opening the model guide', async () => {
    const fetchOwner = vi.fn((input: RequestInfo | URL) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
      expect(url).toMatch(/\/embedded\/agent-manager\/external-url$/)
      return Promise.resolve(new Response(JSON.stringify({ url: 'https://agents.example.test/apps/agent-manager/proxy/' }), { status: 200 }))
    })
    vi.stubGlobal('fetch', fetchOwner)
    render(<TeamPurposePanel team={makeTeam({ purpose: undefined, lifetime: undefined })} onUpdate={vi.fn()} />)
    expect(screen.getByText('Lifetime unspecified')).toBeInTheDocument()
    expect(screen.getByText('Purpose unspecified')).toBeInTheDocument()
    expect(screen.getByText('This team has no objective commitments yet. Link a terminal end or instrumental means.')).toBeInTheDocument()
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
