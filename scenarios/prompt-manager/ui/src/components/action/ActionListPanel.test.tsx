import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { ActionListPanel } from './ActionListPanel'
import { useActionsData } from '@/hooks/useActionsData'
import type { Action } from '@/types'

vi.mock('@/hooks/useActionsData', () => ({
  useActionsData: vi.fn(),
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

const createAction = vi.fn()

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(useActionsData).mockReturnValue({
    actions: [makeAction()],
    isLoading: false,
    isError: false,
    error: null,
    createAction,
    updateAction: vi.fn(),
    deleteAction: vi.fn(),
    validateAction: vi.fn(),
    runAction: vi.fn(),
    isCreating: false,
    isUpdating: false,
    isDeleting: false,
    isValidating: false,
    isRunning: false,
    refetch: vi.fn(),
  })
})

describe('ActionListPanel', () => {
  it('renders searchable Action rows and selects an Action', () => {
    const onSelect = vi.fn()

    render(
      <ActionListPanel
        selectedActionId={null}
        onSelectAction={onSelect}
        searchQuery="decisions"
      />
    )

    fireEvent.click(screen.getByText('List Team Decisions'))

    expect(screen.getByText('team.decisions.list')).toBeDefined()
    expect(screen.getByText('prompt-manager team decisions list')).toBeDefined()
    expect(screen.getByRole('button', { name: 'List Team Decisions, active Action owned by scenario:prompt-manager' })).toBeDefined()
    expect(onSelect).toHaveBeenCalledWith('team.decisions.list')
  })

  it('creates a draft Action with a controlled prompt-manager command', async () => {
    createAction.mockResolvedValue({ action: makeAction({ id: 'action.draft.1' }), validation: {} })
    const onSelect = vi.fn()

    render(<ActionListPanel selectedActionId={null} onSelectAction={onSelect} />)
    fireEvent.click(screen.getByRole('button', { name: 'New Action' }))

    await waitFor(() => expect(createAction).toHaveBeenCalled())
    expect(createAction.mock.calls[0]?.[0]).toMatchObject({
      id: 'action.draft.1',
      status: 'draft',
      command: { argv: ['prompt-manager', 'action', 'list'] },
      permissions: { apiRead: true },
      pack: 'drafts',
    })
    expect(onSelect).toHaveBeenCalledWith('action.draft.1')
  })
})
