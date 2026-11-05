import { beforeEach, describe, expect, it, vi } from 'vitest'
import { act, renderHook, waitFor } from '@testing-library/react'
import { useMemberConversation } from './useMemberConversation'
import {
  createConversationSession,
  readConversationSession,
  readPendingStart,
  writeConversationSession,
  type ConversationIdentity,
  type SessionStorage,
} from './conversationSession'
import {
  continueRun,
  createMemberConversation,
  getRunDetails,
  getRunEvents,
  newConversationRequestId,
  type RunDetails,
} from '@/services/heartbeatService'

vi.mock('@/services/heartbeatService', () => ({
  continueRun: vi.fn(),
  createMemberConversation: vi.fn(),
  getRunDetails: vi.fn(),
  getRunEvents: vi.fn(),
  newConversationRequestId: vi.fn(),
}))

function memoryStorage(): SessionStorage {
  const map = new Map<string, string>()
  return {
    getItem: (key) => (map.has(key) ? (map.get(key) as string) : null),
    setItem: (key, value) => {
      map.set(key, value)
    },
    removeItem: (key) => {
      map.delete(key)
    },
  }
}

function makeRun(overrides: Partial<RunDetails> = {}): RunDetails {
  return { id: 'run-1', taskId: 'task-1', status: 'completed', ...overrides }
}

const teamIdentity: ConversationIdentity = { agentId: 'ada', teamId: 'team-a' }

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(getRunDetails).mockResolvedValue(makeRun())
  vi.mocked(getRunEvents).mockResolvedValue([])
})

describe('useMemberConversation', () => {
  it('starts a conversation with a durable first-turn request id', async () => {
    vi.mocked(newConversationRequestId).mockReturnValue('req-start')
    vi.mocked(createMemberConversation).mockResolvedValue(makeRun({ id: 'run-9', status: 'running' }))
    const storage = memoryStorage()

    const { result } = renderHook(() =>
      useMemberConversation({ identity: teamIdentity, storage }),
    )

    expect(result.current.loading).toBe(false)

    await act(async () => {
      await result.current.send('hello there')
    })

    expect(createMemberConversation).toHaveBeenCalledWith({
      teamId: 'team-a',
      agentId: 'ada',
      message: 'hello there',
      requestId: 'req-start',
    })
    expect(readConversationSession(teamIdentity, storage)?.runId).toBe('run-9')
    expect(readPendingStart(teamIdentity, storage)).toBeNull()
    expect(result.current.run?.id).toBe('run-9')
  })

  it('keeps the failed start operation and reuses the same request id on retry', async () => {
    vi.mocked(newConversationRequestId).mockReturnValue('req-start')
    vi.mocked(createMemberConversation)
      .mockRejectedValueOnce(new Error('response lost'))
      .mockResolvedValueOnce(makeRun({ id: 'run-9', status: 'running' }))
    const storage = memoryStorage()

    const { result } = renderHook(() =>
      useMemberConversation({ identity: teamIdentity, storage }),
    )

    await act(async () => {
      await result.current.send('first message')
    })
    expect(result.current.recovery).toEqual({ operation: 'start', message: 'response lost' })
    expect(readPendingStart(teamIdentity, storage)).toEqual({
      requestId: 'req-start',
      message: 'first message',
    })

    // A different message must not silently abandon the unconfirmed first turn.
    await act(async () => {
      await result.current.send('different message')
    })
    expect(createMemberConversation).toHaveBeenCalledTimes(1)

    await act(async () => {
      await result.current.send('first message')
    })
    expect(createMemberConversation).toHaveBeenLastCalledWith({
      teamId: 'team-a',
      agentId: 'ada',
      message: 'first message',
      requestId: 'req-start',
    })
    expect(newConversationRequestId).toHaveBeenCalledTimes(1)
  })

  it('reuses a pending turn request id when a continue response is lost', async () => {
    const storage = memoryStorage()
    writeConversationSession(createConversationSession(teamIdentity, 'run-1', 'req-first'), storage)
    vi.mocked(newConversationRequestId).mockReturnValue('turn-1')
    vi.mocked(continueRun)
      .mockRejectedValueOnce(new Error('network'))
      .mockResolvedValueOnce(undefined)
    vi.mocked(getRunDetails).mockResolvedValue(makeRun({ status: 'completed' }))

    const { result } = renderHook(() =>
      useMemberConversation({ identity: teamIdentity, storage }),
    )
    await waitFor(() => expect(result.current.run?.id).toBe('run-1'))

    await act(async () => {
      await result.current.send('second turn')
    })
    expect(continueRun).toHaveBeenCalledWith('run-1', 'second turn', { requestId: 'turn-1' })
    expect(result.current.recovery).toEqual({ operation: 'continue', message: 'network' })
    expect(readConversationSession(teamIdentity, storage)?.pendingTurn).toEqual({
      requestId: 'turn-1',
      message: 'second turn',
    })

    await act(async () => {
      await result.current.send('second turn')
    })
    expect(continueRun).toHaveBeenLastCalledWith('run-1', 'second turn', { requestId: 'turn-1' })
    expect(newConversationRequestId).toHaveBeenCalledTimes(1)
    expect(readConversationSession(teamIdentity, storage)?.pendingTurn).toBeUndefined()
  })

  it('reloads the recorded run instead of starting a new conversation', async () => {
    const storage = memoryStorage()
    writeConversationSession(createConversationSession(teamIdentity, 'run-7', 'req-first'), storage)
    vi.mocked(getRunDetails).mockResolvedValue(makeRun({ id: 'run-7' }))

    const { result } = renderHook(() =>
      useMemberConversation({ identity: teamIdentity, storage }),
    )

    await waitFor(() => expect(result.current.run?.id).toBe('run-7'))
    expect(createMemberConversation).not.toHaveBeenCalled()
    expect(getRunDetails).toHaveBeenCalledWith('run-7')
  })

  it('never reuses another identity session', () => {
    const storage = memoryStorage()
    writeConversationSession(createConversationSession(teamIdentity, 'run-1', 'req-first'), storage)

    const { result } = renderHook(() =>
      useMemberConversation({ identity: { agentId: 'ada', teamId: 'team-b' }, storage }),
    )

    expect(result.current.run).toBeNull()
    expect(createMemberConversation).not.toHaveBeenCalled()
    expect(readConversationSession(teamIdentity, storage)?.runId).toBe('run-1')
  })
})
