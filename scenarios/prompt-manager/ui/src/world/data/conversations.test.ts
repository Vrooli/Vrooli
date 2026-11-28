import { describe, expect, it, vi } from 'vitest'
import { conversationKey, createWorldConversations, hasUnread } from './conversations'
import type { RunDetails, RunEvent } from '@/services/heartbeatService'

const member = { id: 'ada', teamId: 'garden', name: 'Ada' }
const run: RunDetails = { id: 'run-1', taskId: 'task-1', status: 'completed', actions: { canContinue: true, canStop: false, canDelete: false, canRetry: false, canInvestigate: false, canApplyInvestigation: false } }
const reply: RunEvent = { id: 'reply-1', runId: run.id, sequence: 3, timestamp: '', eventType: 'message', data: { role: 'assistant', content: 'The garden is doing well.' } }
function fixture() {
  const values = new Map<string, string>()
  const storage = { getItem: (key: string) => values.get(key) ?? null, setItem: (key: string, value: string) => { values.set(key, value) } } as Storage
  const api = { start: vi.fn().mockResolvedValue(run), continue: vi.fn().mockResolvedValue(undefined), details: vi.fn().mockResolvedValue(run), events: vi.fn().mockResolvedValue([reply]) }
  return { api, storage: () => storage, store: createWorldConversations(api, () => storage) }
}

describe('world conversations', () => {
  it('starts only on send and binds to the selected member, preserving retry identity', async () => {
    const { api, store, storage } = fixture()
    await store.tick()
    expect(api.start).not.toHaveBeenCalled()
    api.start.mockRejectedValueOnce(new Error('connection lost'))
    await expect(store.send(member, 'Hello')).rejects.toThrow('connection lost')
    const restored = createWorldConversations(api, storage)
    await restored.send(member, 'Hello')
    expect(api.start.mock.calls[1]?.[0]).toEqual(api.start.mock.calls[0]?.[0])
    expect(api.start.mock.calls[0]?.[0]).toMatchObject({ teamId: 'garden', agentId: 'ada', message: 'Hello' })
    expect(api.start.mock.calls[0]?.[0].requestId).toMatch(/^[a-f0-9-]{36}$/)
  })
  it('marks unseen replies unread, retains receipts across reloads, and continues the same run', async () => {
    const { api, store, storage } = fixture()
    const key = conversationKey(member)
    await store.send(member, 'Hello')
    await store.refresh(key)
    expect(hasUnread(store.getSnapshot()[key])).toBe(true)
    store.read(key)
    expect(hasUnread(store.getSnapshot()[key])).toBe(false)
    const restored = createWorldConversations(api, storage)
    await restored.refresh(key)
    expect(hasUnread(restored.getSnapshot()[key])).toBe(false)
    await restored.send(member, 'Tell me more')
    expect(api.continue).toHaveBeenCalledWith('run-1', 'Tell me more')
    expect(api.start).toHaveBeenCalledTimes(1)
    expect(restored.getSnapshot()[key]?.run?.status).toBe('pending')
  })
  it('does not let a late poll overwrite a sent continuation', async () => {
    const { api, store } = fixture()
    const key = conversationKey(member)
    await store.send(member, 'Hello')
    let resolve!: (events: RunEvent[]) => void
    api.events.mockReturnValueOnce(new Promise<RunEvent[]>(done => { resolve = done }))
    const poll = store.refresh(key)
    await store.send(member, 'Tell me more')
    resolve([reply]); await poll
    expect(store.getSnapshot()[key]?.run?.status).toBe('pending')
    expect(store.getSnapshot()[key]?.after).toBe(0)
  })
  it('isolates member identities and hides assembled context from chat', async () => {
    const { api, store } = fixture()
    expect(conversationKey(member)).not.toBe(conversationKey({ ...member, teamId: 'other' }))
    api.events.mockResolvedValue([ { ...reply, id: 'prompt', sequence: 1, data: { role: 'user', content: '<context>Private reference files</context>\nOperator message:\nHello' } }, reply ])
    await store.send(member, 'Hello')
    await store.refresh(conversationKey(member))
    expect(store.getSnapshot()[conversationKey(member)]?.messages).toEqual([expect.objectContaining({ content: 'The garden is doing well.' })])
  })
  it('handles unavailable storage and prevents simultaneous duplicate sends', async () => {
    const { api } = fixture()
    const store = createWorldConversations(api, () => { throw new Error('disabled') })
    await Promise.all([store.send(member, 'Hello'), store.send(member, 'Hello')])
    expect(api.start).toHaveBeenCalledTimes(1)
  })
})
