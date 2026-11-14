import { describe, expect, it } from 'vitest'
import {
  clearConversationSession,
  clearPendingStart,
  conversationIdentityKey,
  createConversationSession,
  isSessionCompatible,
  readConversationSession,
  readPendingStart,
  updateConversationSession,
  writeConversationSession,
  writePendingStart,
  type SessionStorage,
} from './conversationSession'

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

describe('conversationIdentityKey', () => {
  it('separates the base context from each team membership', () => {
    const base = conversationIdentityKey({ agentId: 'ada' })
    const teamA = conversationIdentityKey({ agentId: 'ada', teamId: 'team-a' })
    const teamB = conversationIdentityKey({ agentId: 'ada', teamId: 'team-b' })
    expect(new Set([base, teamA, teamB]).size).toBe(3)
  })

  it('treats a blank team as the base context and trims', () => {
    expect(conversationIdentityKey({ agentId: ' ada ', teamId: '  ' })).toBe(
      conversationIdentityKey({ agentId: 'ada' }),
    )
  })
})

describe('conversation sessions', () => {
  it('round-trips an accepted session', () => {
    const storage = memoryStorage()
    const identity = { agentId: 'ada', teamId: 'team-a' }
    writeConversationSession(createConversationSession(identity, 'run-1', 'req-1'), storage)
    const session = readConversationSession(identity, storage)
    expect(session).toMatchObject({ runId: 'run-1', firstRequestId: 'req-1', teamId: 'team-a' })
  })

  it('never returns another identity session', () => {
    const storage = memoryStorage()
    writeConversationSession(
      createConversationSession({ agentId: 'ada', teamId: 'team-a' }, 'run-1', 'req-1'),
      storage,
    )
    expect(readConversationSession({ agentId: 'ada', teamId: 'team-b' }, storage)).toBeNull()
    expect(readConversationSession({ agentId: 'ada' }, storage)).toBeNull()
  })

  it('treats malformed records as absent', () => {
    const storage = memoryStorage()
    storage.setItem('prompt-manager.conversation.ada::base', '{not json')
    expect(readConversationSession({ agentId: 'ada' }, storage)).toBeNull()
  })

  it('records a pending turn durably and preserves the session key', () => {
    const storage = memoryStorage()
    const identity = { agentId: 'ada', teamId: 'team-a' }
    const session = createConversationSession(identity, 'run-1', 'req-1')
    writeConversationSession(session, storage)
    const updated = updateConversationSession(
      identity,
      { pendingTurn: { requestId: 'turn-2', message: 'second turn' } },
      storage,
    )
    expect(updated?.key).toBe(session.key)
    expect(readConversationSession(identity, storage)?.pendingTurn).toEqual({
      requestId: 'turn-2',
      message: 'second turn',
    })
  })

  it('keeps a dispatched first-turn identity until it is confirmed', () => {
    const storage = memoryStorage()
    const identity = { agentId: 'ada', teamId: 'team-a' }
    writePendingStart(identity, { requestId: 'req-1', message: 'hello' }, storage)
    expect(readPendingStart(identity, storage)).toEqual({ requestId: 'req-1', message: 'hello' })
    expect(readPendingStart({ agentId: 'ada' }, storage)).toBeNull()
    clearPendingStart(identity, storage)
    expect(readPendingStart(identity, storage)).toBeNull()
  })

  it('treats a malformed pending start as absent', () => {
    const storage = memoryStorage()
    storage.setItem('prompt-manager.conversation.ada::base.pending-start', '{"requestId":1}')
    expect(readPendingStart({ agentId: 'ada' }, storage)).toBeNull()
  })

  it('reports incompatibility and clears a session', () => {
    const storage = memoryStorage()
    const identity = { agentId: 'ada' }
    const session = createConversationSession(identity, 'run-1', 'req-1')
    expect(isSessionCompatible(session, identity)).toBe(true)
    expect(isSessionCompatible(session, { agentId: 'ada', teamId: 'team-a' })).toBe(false)
    writeConversationSession(session, storage)
    clearConversationSession(identity, storage)
    expect(readConversationSession(identity, storage)).toBeNull()
  })
})
