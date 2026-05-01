import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  createAction,
  deleteAction,
  getAction,
  getActions,
  invalidateCache,
  searchActions,
  updateAction,
  validateAction,
  runAction,
} from './actionService'
import { api } from '@/lib/api'
import type { Action, CreateActionRequest, UpdateActionRequest } from '@/types'

vi.mock('@/lib/api', () => ({
  api: {
    getActions: vi.fn(),
    getAction: vi.fn(),
    createAction: vi.fn(),
    updateAction: vi.fn(),
    deleteAction: vi.fn(),
    validateAction: vi.fn(),
    runAction: vi.fn(),
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

describe('actionService', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    invalidateCache()
    vi.useRealTimers()
  })

  it('caches unfiltered action lists', async () => {
    const actions = [createTestAction()]
    vi.mocked(api.getActions).mockResolvedValue(actions)

    await getActions()
    const second = await getActions()

    expect(api.getActions).toHaveBeenCalledTimes(1)
    expect(second).toEqual(actions)
  })

  it('does not cache filtered action lists', async () => {
    const actions = [createTestAction()]
    vi.mocked(api.getActions).mockResolvedValue(actions)

    await getActions({ status: 'draft' })
    await getActions({ status: 'draft' })

    expect(api.getActions).toHaveBeenCalledTimes(2)
    expect(api.getActions).toHaveBeenCalledWith({ status: 'draft' })
  })

  it('invalidates list cache after create, update, and delete', async () => {
    const action = createTestAction()
    const validation = {
      actionId: action.id,
      valid: true,
      runnable: false,
      status: action.status,
      checks: [],
      action,
    }
    vi.mocked(api.getActions).mockResolvedValue([action])
    vi.mocked(api.createAction).mockResolvedValue({ action, validation })
    vi.mocked(api.updateAction).mockResolvedValue({ action, validation })
    vi.mocked(api.deleteAction).mockResolvedValue(undefined)

    await getActions()
    await createAction(action as CreateActionRequest)
    await getActions()
    await updateAction(action.id, { description: 'Updated' } as UpdateActionRequest)
    await getActions()
    await deleteAction(action.id, true)
    await getActions()

    expect(api.getActions).toHaveBeenCalledTimes(4)
    expect(api.deleteAction).toHaveBeenCalledWith(action.id, true)
  })

  it('fetches and validates single actions through the API seam', async () => {
    const action = createTestAction()
    vi.mocked(api.getAction).mockResolvedValue(action)
    vi.mocked(api.validateAction).mockResolvedValue({
      actionId: action.id,
      valid: true,
      runnable: false,
      status: action.status,
      checks: [],
      action,
    })

    await expect(getAction(action.id)).resolves.toEqual(action)
    await expect(validateAction(action.id)).resolves.toMatchObject({ valid: true })
    expect(api.getAction).toHaveBeenCalledWith(action.id)
    expect(api.validateAction).toHaveBeenCalledWith(action.id)
  })

  it('runs actions through the API seam without invalidating list cache', async () => {
    const action = createTestAction()
    vi.mocked(api.runAction).mockResolvedValue({
      actionId: action.id,
      status: 'dry-run',
      durationMs: 1,
      argv: ['prompt-manager', 'team', 'decision-list', 'meta-optimization', '--json'],
      stdout: '',
      stderr: '',
      stdoutTruncated: false,
      stderrTruncated: false,
      error: '',
      validation: {
        actionId: action.id,
        valid: true,
        runnable: true,
        status: action.status,
        checks: [],
      },
    })

    await expect(runAction(action.id, { input: { team: 'meta-optimization' }, dryRun: true }))
      .resolves.toMatchObject({ status: 'dry-run' })
    expect(api.runAction).toHaveBeenCalledWith(action.id, {
      input: { team: 'meta-optimization' },
      dryRun: true,
    })
  })

  it('searches cached actions by id, owner, command, and tags', async () => {
    const actions = [
      createTestAction(),
      createTestAction({
        id: 'scenario.ui.screenshot',
        name: 'Capture Screenshot',
        owner: { type: 'scenario', id: 'ui-runner' },
        command: { argv: ['vrooli', 'scenario', 'screenshot', '{{scenario}}'] },
        tags: ['ui'],
      }),
    ]
    vi.mocked(api.getActions).mockResolvedValue(actions)
    await getActions()

    expect(await searchActions('ui-runner')).toHaveLength(1)
    expect(await searchActions('screenshot')).toHaveLength(1)
    expect(await searchActions('teams')).toHaveLength(1)
    expect(api.getActions).toHaveBeenCalledTimes(1)
  })
})
