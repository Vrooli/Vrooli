import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { waitFor } from '@testing-library/react'
import { renderHookWithProviders } from '@/test'
import { useActionsData } from './useActionsData'
import * as actionService from '@/services/actionService'
import type { Action, CreateActionRequest, UpdateActionRequest } from '@/types'

vi.mock('@/services/actionService', () => ({
  getActions: vi.fn(),
  createAction: vi.fn(),
  updateAction: vi.fn(),
  deleteAction: vi.fn(),
  validateAction: vi.fn(),
  runAction: vi.fn(),
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
    id: 'team.swarm.work.list',
    name: 'List Team Work',
    description: 'List recent work items.',
    status: 'draft',
    owner: { type: 'scenario', id: 'prompt-manager' },
    command: { argv: ['swarm-manager', 'backlog', 'list', '--json'] },
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

    const { result } = renderHookWithProviders(() => useActionsData())

    expect(result.current.isLoading).toBe(true)

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    expect(result.current.actions).toEqual(actions)
    expect(actionService.getActions).toHaveBeenCalledWith(undefined)
  })

  it('passes filters into the query function', async () => {
    vi.mocked(actionService.getActions).mockResolvedValue([])

    renderHookWithProviders(() => useActionsData({ status: 'draft', tag: 'teams' }))

    await waitFor(() => {
      expect(actionService.getActions).toHaveBeenCalledWith({ status: 'draft', tag: 'teams' })
    })
  })

  it('creates, updates, deletes, validates, and runs through the service seam', async () => {
    const action = createTestAction()
    const validation = {
      actionId: action.id,
      valid: true,
      runnable: false,
      unvalidated: false,
      requiresConfirmation: false,
      status: action.status,
      checks: [],
      action,
    }
    vi.mocked(actionService.getActions).mockResolvedValue([action])
    vi.mocked(actionService.createAction).mockResolvedValue({ action, validation })
    vi.mocked(actionService.updateAction).mockResolvedValue({ action, validation })
    vi.mocked(actionService.deleteAction).mockResolvedValue(undefined)
    vi.mocked(actionService.validateAction).mockResolvedValue(validation)
    vi.mocked(actionService.runAction).mockResolvedValue({
      actionId: action.id,
      status: 'dry-run',
      durationMs: 1,
      argv: ['swarm-manager', 'backlog', 'list', '--json'],
      stdout: '',
      stderr: '',
      stdoutTruncated: false,
      stderrTruncated: false,
      error: '',
      validation,
    })

    const { result } = renderHookWithProviders(() => useActionsData())

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    const createRequest = action as CreateActionRequest
    const updateRequest = { description: 'Updated' } as UpdateActionRequest
    await result.current.createAction(createRequest)
    await result.current.updateAction(action.id, updateRequest)
    await result.current.deleteAction(action.id, true)
    await expect(result.current.validateAction(action.id)).resolves.toEqual(validation)
    await expect(result.current.runAction(action.id, { input: { team: 'meta-optimization' }, dryRun: true }))
      .resolves.toMatchObject({ status: 'dry-run' })

    expect(actionService.createAction).toHaveBeenCalledWith(createRequest)
    expect(actionService.updateAction).toHaveBeenCalledWith(action.id, updateRequest)
    expect(actionService.deleteAction).toHaveBeenCalledWith(action.id, true)
    expect(actionService.validateAction).toHaveBeenCalledWith(action.id)
    expect(actionService.runAction).toHaveBeenCalledWith(action.id, {
      input: { team: 'meta-optimization' },
      dryRun: true,
    })
  })

  it('exposes validation pending state separately from mutations', async () => {
    vi.mocked(actionService.getActions).mockResolvedValue([])
    let resolveValidation: (value: Awaited<ReturnType<typeof actionService.validateAction>>) => void = () => {}
    vi.mocked(actionService.validateAction).mockImplementation(
      () => new Promise((resolve) => { resolveValidation = resolve })
    )

    const { result } = renderHookWithProviders(() => useActionsData())

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    const validationPromise = result.current.validateAction('team.swarm.work.list')

    await waitFor(() => {
      expect(result.current.isValidating).toBe(true)
    })

    resolveValidation({
      actionId: 'team.swarm.work.list',
      valid: true,
      runnable: false,
      unvalidated: false,
      requiresConfirmation: false,
      status: 'draft',
      checks: [],
    })
    await validationPromise

    await waitFor(() => {
      expect(result.current.isValidating).toBe(false)
    })
  })

  it('exposes run pending state separately from mutations', async () => {
    vi.mocked(actionService.getActions).mockResolvedValue([])
    let resolveRun: (value: Awaited<ReturnType<typeof actionService.runAction>>) => void = () => {}
    vi.mocked(actionService.runAction).mockImplementation(
      () => new Promise((resolve) => { resolveRun = resolve })
    )

    const { result } = renderHookWithProviders(() => useActionsData())

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    const runPromise = result.current.runAction('team.swarm.work.list', { input: {}, dryRun: true })

    await waitFor(() => {
      expect(result.current.isRunning).toBe(true)
    })

    resolveRun({
      actionId: 'team.swarm.work.list',
      status: 'dry-run',
      durationMs: 1,
      argv: ['swarm-manager', 'backlog', 'list', '--json'],
      stdout: '',
      stderr: '',
      stdoutTruncated: false,
      stderrTruncated: false,
      error: '',
      validation: {
        actionId: 'team.swarm.work.list',
        valid: true,
        runnable: true,
        unvalidated: false,
        requiresConfirmation: false,
        status: 'draft',
        checks: [],
      },
    })
    await runPromise

    await waitFor(() => {
      expect(result.current.isRunning).toBe(false)
    })
  })
})
