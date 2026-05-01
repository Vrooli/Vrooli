import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { ActionEditorPanel } from './ActionEditorPanel'
import { useActionsData } from '@/hooks/useActionsData'
import type { Action, ActionValidationResponse } from '@/types'

vi.mock('@/hooks/useActionsData', () => ({
  useActionsData: vi.fn(),
}))

vi.mock('@/hooks/use-toast', () => ({
  toast: vi.fn(),
}))

vi.mock('@/lib/clipboard', () => ({
  copyToClipboard: vi.fn().mockResolvedValue(undefined),
}))

function makeAction(overrides: Partial<Action> = {}): Action {
  return {
    kind: 'action',
    schemaVersion: 1,
    id: 'team.decisions.list',
    name: 'List Team Decisions',
    description: 'List decisions for a team.',
    status: 'active',
    owner: { type: 'scenario', id: 'prompt-manager' },
    command: { argv: ['prompt-manager', 'team', 'decisions', 'list'] },
    inputs: {},
    outputs: {},
    permissions: {
      filesystemRead: false,
      filesystemWrite: false,
      localhostNetwork: false,
      externalNetwork: false,
      apiRead: true,
      apiWrite: false,
      processStart: false,
      processStop: false,
      hostConfigure: false,
      secretRead: false,
      secretWrite: false,
      destructive: false,
    },
    examples: [],
    tags: ['teams'],
    revision: 1,
    createdAt: '2026-04-30T00:00:00Z',
    updatedAt: '2026-04-30T00:00:00Z',
    ...overrides,
  }
}

const validation: ActionValidationResponse = {
  actionId: 'team.decisions.list',
  valid: true,
  runnable: false,
  status: 'active',
  checks: [
    {
      code: 'command.controlled',
      status: 'passed',
      message: 'Command is controlled.',
      path: 'command.argv',
    },
  ],
}

const updateAction = vi.fn()
const deleteAction = vi.fn()
const validateAction = vi.fn()

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(useActionsData).mockReturnValue({
    actions: [makeAction()],
    isLoading: false,
    isError: false,
    error: null,
    createAction: vi.fn(),
    updateAction,
    deleteAction,
    validateAction,
    isCreating: false,
    isUpdating: false,
    isDeleting: false,
    isValidating: false,
    refetch: vi.fn(),
  })
})

describe('ActionEditorPanel', () => {
  it('renders Action contract details and keeps run unavailable', () => {
    render(<ActionEditorPanel actionId="team.decisions.list" onClose={vi.fn()} />)

    expect(screen.getByText('List Team Decisions')).toBeDefined()
    expect(screen.getByText('team.decisions.list')).toBeDefined()
    expect(screen.getAllByText('prompt-manager team decisions list').length).toBeGreaterThan(0)
    expect(screen.getByRole('button', { name: 'Run unavailable' })).toBeDisabled()
  })

  it('shows validation results from the API', async () => {
    validateAction.mockResolvedValue(validation)
    render(<ActionEditorPanel actionId="team.decisions.list" onClose={vi.fn()} />)

    fireEvent.click(screen.getByRole('button', { name: 'Validate' }))

    await waitFor(() => expect(validateAction).toHaveBeenCalledWith('team.decisions.list'))
    expect(await screen.findByText('command.controlled')).toBeDefined()
    expect(screen.getByText('Command is controlled.')).toBeDefined()
  })

  it('saves edited contract JSON through the Action API', async () => {
    updateAction.mockResolvedValue({
      action: makeAction({ name: 'Updated Action' }),
      validation,
    })

    render(<ActionEditorPanel actionId="team.decisions.list" onClose={vi.fn()} />)
    const editor = screen.getByLabelText('Action contract JSON')
    const nextJson = JSON.stringify({
      ...makeAction({ name: 'Updated Action' }),
      revision: undefined,
      createdAt: undefined,
      updatedAt: undefined,
    })

    fireEvent.change(editor, { target: { value: nextJson } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(updateAction).toHaveBeenCalled())
    expect(updateAction.mock.calls[0]?.[0]).toBe('team.decisions.list')
    expect(updateAction.mock.calls[0]?.[1]).toMatchObject({ name: 'Updated Action' })
  })

  it('updates the JSON draft from typed contract fields', async () => {
    updateAction.mockResolvedValue({
      action: makeAction({
        name: 'Typed Action',
        command: { argv: ['prompt-manager', 'team', 'decisions', 'show', '{{team}}'] },
        inputs: {
          team: {
            type: 'team',
            description: '',
            required: true,
            enum: [],
            pattern: '',
            allowMultiline: false,
          },
        },
      }),
      validation,
    })

    render(<ActionEditorPanel actionId="team.decisions.list" onClose={vi.fn()} />)

    fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'Typed Action' } })
    fireEvent.change(screen.getByLabelText(/Argv tokens/), {
      target: { value: 'prompt-manager\nteam\ndecisions\nshow\n{{team}}' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Add input' }))
    const nameFields = screen.getAllByLabelText('Name')
    expect(nameFields.length).toBeGreaterThan(1)
    fireEvent.change(nameFields[1]!, { target: { value: 'team' } })
    fireEvent.change(screen.getByLabelText(/Type/), { target: { value: 'team' } })
    fireEvent.click(screen.getByLabelText('filesystemWrite'))
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(updateAction).toHaveBeenCalled())
    expect(updateAction.mock.calls[0]?.[1]).toMatchObject({
      name: 'Typed Action',
      command: { argv: ['prompt-manager', 'team', 'decisions', 'show', '{{team}}'] },
      inputs: { team: { type: 'team', required: true } },
      permissions: { filesystemWrite: true },
    })
  })

  it('archives instead of hard deleting from the primary lifecycle action', async () => {
    const onClose = vi.fn()
    deleteAction.mockResolvedValue(undefined)

    render(<ActionEditorPanel actionId="team.decisions.list" onClose={onClose} />)
    fireEvent.click(screen.getByRole('button', { name: 'Archive' }))

    await waitFor(() => expect(deleteAction).toHaveBeenCalledWith('team.decisions.list', false))
    expect(onClose).toHaveBeenCalled()
  })

  it('requires a second click before hard deleting', async () => {
    const onClose = vi.fn()
    deleteAction.mockResolvedValue(undefined)

    render(<ActionEditorPanel actionId="team.decisions.list" onClose={onClose} />)
    fireEvent.click(screen.getByRole('button', { name: 'Hard delete' }))

    expect(deleteAction).not.toHaveBeenCalled()
    expect(screen.getByRole('button', { name: 'Confirm hard delete' })).toBeDefined()

    fireEvent.click(screen.getByRole('button', { name: 'Confirm hard delete' }))

    await waitFor(() => expect(deleteAction).toHaveBeenCalledWith('team.decisions.list', true))
    expect(onClose).toHaveBeenCalled()
  })
})
