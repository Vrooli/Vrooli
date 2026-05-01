import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useActionsData } from './useActionsData'
import * as actionService from '@/services/actionService'
import type { Action, CreateActionRequest, UpdateActionRequest } from '@/types'

vi.mock('@/services/actionService', () => ({
  getActions: vi.fn(),
  createAction: vi.fn(),
  updateAction: vi.fn(),
  deleteAction: vi.fn(),
  validateAction: vi.fn(),
}))

const fetchHealthScores = vi.fn()
vi.mock('@/stores/graphStore', () => ({
  useGraphStore: {
    getState: () => ({ fetchHealthScores }),
  },
}))

function createTestAction(overrides: Partial<Action> = {}): Action {
  return {
    kind: 'action',
    schemaVersion: 1,
    id: 'team.decisions.list',
    name: 'List Team Decisions',
    description: 'List recent decisions.',
    status: 'draft',
    owner: { type: 'scenario', id: 'prompt-manager' },
    command: { argv: ['prompt-manager', 'team', 'decision-list', '{{team}}', '--json'] },
    inputs: {},
    outputs: {},
    permissions: {
      filesystemRead: false,
      filesystemWrite: false,
      localhostNetwork: false,
      externalNetwork: false,
      apiRead: false,
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

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  })

  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

describe('useActionsData', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.resetAllMocks()
  })

  it('fetches actions on mount', async () => {
    const actions = [createTestAction()]
    vi.mocked(actionService.getActions).mockResolvedValue(actions)

    const { result } = renderHook(() => useActionsData(), {
      wrapper: createWrapper(),
    })

    expect(result.current.isLoading).toBe(true)

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    expect(result.current.actions).toEqual(actions)
    expect(actionService.getActions).toHaveBeenCalledWith(undefined)
  })

  it('passes filters into the query function', async () => {
    vi.mocked(actionService.getActions).mockResolvedValue([])

    renderHook(() => useActionsData({ status: 'draft', tag: 'teams' }), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(actionService.getActions).toHaveBeenCalledWith({ status: 'draft', tag: 'teams' })
    })
  })

  it('creates, updates, deletes, and validates through the service seam', async () => {
    const action = createTestAction()
    const validation = {
      actionId: action.id,
      valid: true,
      runnable: false,
      status: action.status,
      checks: [],
      action,
    }
    vi.mocked(actionService.getActions).mockResolvedValue([action])
    vi.mocked(actionService.createAction).mockResolvedValue({ action, validation })
    vi.mocked(actionService.updateAction).mockResolvedValue({ action, validation })
    vi.mocked(actionService.deleteAction).mockResolvedValue(undefined)
    vi.mocked(actionService.validateAction).mockResolvedValue(validation)

    const { result } = renderHook(() => useActionsData(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    const createRequest = action as CreateActionRequest
    const updateRequest = { description: 'Updated' } as UpdateActionRequest
    await result.current.createAction(createRequest)
    await result.current.updateAction(action.id, updateRequest)
    await result.current.deleteAction(action.id, true)
    await expect(result.current.validateAction(action.id)).resolves.toEqual(validation)

    expect(actionService.createAction).toHaveBeenCalledWith(createRequest)
    expect(actionService.updateAction).toHaveBeenCalledWith(action.id, updateRequest)
    expect(actionService.deleteAction).toHaveBeenCalledWith(action.id, true)
    expect(actionService.validateAction).toHaveBeenCalledWith(action.id)
  })

  it('exposes validation pending state separately from mutations', async () => {
    vi.mocked(actionService.getActions).mockResolvedValue([])
    let resolveValidation: (value: Awaited<ReturnType<typeof actionService.validateAction>>) => void = () => {}
    vi.mocked(actionService.validateAction).mockImplementation(
      () => new Promise((resolve) => { resolveValidation = resolve })
    )

    const { result } = renderHook(() => useActionsData(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    const validationPromise = result.current.validateAction('team.decisions.list')

    await waitFor(() => {
      expect(result.current.isValidating).toBe(true)
    })

    resolveValidation({
      actionId: 'team.decisions.list',
      valid: true,
      runnable: false,
      status: 'draft',
      checks: [],
    })
    await validationPromise

    await waitFor(() => {
      expect(result.current.isValidating).toBe(false)
    })
  })
})
