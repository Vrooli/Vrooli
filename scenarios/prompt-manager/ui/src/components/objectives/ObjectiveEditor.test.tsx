import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@/test-utils/renderWithProviders'
import { ObjectiveEditor, type ObjectiveEditorCallbacks } from './ObjectiveEditor'
import type { AttachmentViewModel, ObjectiveViewModel } from './objectiveViewModel'

function objective(overrides: Partial<ObjectiveViewModel> = {}): ObjectiveViewModel {
  return {
    id: 'win-market',
    title: 'Win the market',
    class: 'terminal',
    classLabel: 'Terminal end',
    hasEvidence: true,
    evidenceSource: 'revenue',
    globalOrder: 0,
    meaningRevision: 'rev-1',
    ...overrides,
  }
}

function attachment(overrides: Partial<AttachmentViewModel> = {}): AttachmentViewModel {
  const objectiveModel = overrides.objective ?? objective({ id: 'ship-launch', title: 'Ship the launch', class: 'instrumental', classLabel: 'Instrumental means' })
  return {
    objectiveId: objectiveModel.id,
    teamId: 'marketing-crew',
    role: 'primary',
    roleLabel: 'Primary',
    coverage: 'partial',
    coverageLabel: 'Partial coverage',
    priority: 0,
    acknowledgedRevision: 'rev-1',
    attachmentRevision: 'att-1',
    restatementPending: false,
    objective: objectiveModel,
    ...overrides,
  }
}

const noop: ObjectiveEditorCallbacks = {
  onCreateObjective: vi.fn().mockResolvedValue(undefined),
  onUpdateObjective: vi.fn().mockResolvedValue(undefined),
  onDeleteObjective: vi.fn().mockResolvedValue(undefined),
  onReorderObjectives: vi.fn().mockResolvedValue(undefined),
}

describe('ObjectiveEditor global scope', () => {
  it('lists objectives in persisted order with class and evidence context', () => {
    render(<ObjectiveEditor scope="global" objectives={[
      objective({ id: 'b', title: 'Second', globalOrder: 1 }),
      objective({ id: 'a', title: 'First', globalOrder: 0 }),
    ]} />)
    const items = screen.getAllByRole('listitem')
    expect(items[0]).toHaveTextContent('First')
    expect(items[1]).toHaveTextContent('Second')
    expect(items[0]).toHaveTextContent('Terminal end')
    expect(items[0]).toHaveTextContent('evidence: revenue')
  })

  it('creates an objective from trimmed form input', async () => {
    const callbacks: ObjectiveEditorCallbacks = { ...noop, onCreateObjective: vi.fn().mockResolvedValue(undefined) }
    render(<ObjectiveEditor scope="global" objectives={[]} callbacks={callbacks} />)
    fireEvent.click(screen.getByRole('button', { name: 'New objective' }))
    fireEvent.change(screen.getByLabelText('Objective ID'), { target: { value: ' new-gap ' } })
    fireEvent.change(screen.getByLabelText('Objective title'), { target: { value: ' Close the gap ' } })
    fireEvent.change(screen.getByLabelText('Objective class'), { target: { value: 'instrumental' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create objective' }))
    await waitFor(() => expect(callbacks.onCreateObjective).toHaveBeenCalledWith({
      id: 'new-gap', title: 'Close the gap', class: 'instrumental', evidenceSource: undefined, gapMarker: undefined,
    }))
    await screen.findByText('Objective created.')
  })

  it('sends the current meaning revision when editing', async () => {
    const callbacks: ObjectiveEditorCallbacks = { ...noop, onUpdateObjective: vi.fn().mockResolvedValue(undefined) }
    render(<ObjectiveEditor scope="global" objectives={[objective()]} callbacks={callbacks} />)
    fireEvent.click(screen.getByRole('button', { name: 'Edit' }))
    fireEvent.change(screen.getByLabelText('Objective title'), { target: { value: 'Win the category' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save objective' }))
    await waitFor(() => expect(callbacks.onUpdateObjective).toHaveBeenCalledWith(
      expect.objectContaining({ id: 'win-market', title: 'Win the category' }),
      'rev-1',
    ))
  })

  it('persists an accessible move and disables the boundary controls', async () => {
    const callbacks: ObjectiveEditorCallbacks = { ...noop, onReorderObjectives: vi.fn().mockResolvedValue(undefined) }
    render(<ObjectiveEditor scope="global" objectives={[
      objective({ id: 'a', title: 'First', globalOrder: 0 }),
      objective({ id: 'b', title: 'Second', globalOrder: 1 }),
    ]} callbacks={callbacks} />)
    expect(screen.getByRole('button', { name: 'Move First up' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Move Second down' })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: 'Move First down' }))
    await waitFor(() => expect(callbacks.onReorderObjectives).toHaveBeenCalledWith(['b', 'a']))
  })

  it('exposes a keyboard-reachable drag handle for every objective', () => {
    render(<ObjectiveEditor scope="global" objectives={[objective()]} />)
    expect(screen.getByRole('button', { name: 'Reorder Win the market' })).toBeInTheDocument()
  })

  it('preserves the draft and offers recovery on a revision conflict', async () => {
    const onReload = vi.fn()
    const callbacks: ObjectiveEditorCallbacks = {
      ...noop,
      onUpdateObjective: vi.fn().mockRejectedValue(new Error('objectives: revision conflict: expected "rev-0" but current is "rev-1"')),
    }
    render(<ObjectiveEditor scope="global" objectives={[objective()]} callbacks={callbacks} onReload={onReload} />)
    fireEvent.click(screen.getByRole('button', { name: 'Edit' }))
    fireEvent.change(screen.getByLabelText('Objective title'), { target: { value: 'Unsaved title' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save objective' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('revision conflict')
    expect(screen.getByLabelText('Objective title')).toHaveValue('Unsaved title')
    fireEvent.click(screen.getByRole('button', { name: 'Reload authority' }))
    expect(onReload).toHaveBeenCalledTimes(1)
  })
})

describe('ObjectiveEditor team scope', () => {
  it('shows role, coverage and restatement context from the authority read', () => {
    render(<ObjectiveEditor scope="team" teamId="marketing-crew" objectives={[objective()]} attachments={[
      attachment({ restatementPending: true }),
    ]} attachmentRevision="att-1" />)
    expect(screen.getByText('Ship the launch')).toBeInTheDocument()
    expect(screen.getByText(/Primary · Partial coverage/)).toBeInTheDocument()
    expect(screen.getByText('Meaning changed; acknowledgement pending.')).toBeInTheDocument()
  })

  it('acknowledges with the objective current meaning revision', async () => {
    const callbacks: ObjectiveEditorCallbacks = { ...noop, onAcknowledge: vi.fn().mockResolvedValue(undefined) }
    render(<ObjectiveEditor scope="team" teamId="marketing-crew" objectives={[objective()]} attachments={[
      attachment({ restatementPending: true }),
    ]} callbacks={callbacks} />)
    fireEvent.click(screen.getByRole('button', { name: 'Acknowledge' }))
    await waitFor(() => expect(callbacks.onAcknowledge).toHaveBeenCalledWith('ship-launch', 'rev-1'))
  })

  it('reorders team priority with the attachment revision', async () => {
    const callbacks: ObjectiveEditorCallbacks = { ...noop, onReorderTeamAttachments: vi.fn().mockResolvedValue(undefined) }
    render(<ObjectiveEditor scope="team" teamId="marketing-crew" objectives={[objective()]} attachments={[
      attachment({ objective: objective({ id: 'a', title: 'First', globalOrder: 0 }) }),
      attachment({ objective: objective({ id: 'b', title: 'Second', globalOrder: 1 }) }),
    ]} attachmentRevision="att-7" callbacks={callbacks} />)
    fireEvent.click(screen.getByRole('button', { name: 'Move First down' }))
    await waitFor(() => expect(callbacks.onReorderTeamAttachments).toHaveBeenCalledWith(['b', 'a'], 'att-7'))
  })
})
