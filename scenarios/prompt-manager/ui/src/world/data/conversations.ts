import { z } from 'zod'
import { continueRun, createMemberConversation, getRunDetails, getRunEvents, type RunDetails, type RunEvent } from '@/services/heartbeatService'

export interface ConversationMember { id: string; teamId?: string; name: string }
export interface ConversationMessage { id: string; sequence: number; role: 'user' | 'assistant'; content: string }
export interface Conversation {
  member: ConversationMember
  runId?: string
  run?: RunDetails
  firstMessage: string
  requestId: string
  lastRead: number
  after: number
  messages: ConversationMessage[]
  sending: boolean
  error?: string
  settled?: boolean
}
export type ConversationSnapshot = Readonly<Record<string, Conversation>>
export const conversationKey = (member: Pick<ConversationMember, 'id' | 'teamId'>) => JSON.stringify([member.teamId ?? '', member.id])
const STORAGE_KEY = 'prompt-manager.world.conversations.v1'
const MAX_CONVERSATIONS = 24
const savedSchema = z.array(z.object({
  member: z.object({ id: z.string().min(1), teamId: z.string().optional(), name: z.string() }),
  runId: z.string().optional(), requestId: z.uuid(), firstMessage: z.string().max(16000), lastRead: z.number().int().nonnegative(),
})).max(MAX_CONVERSATIONS)

export function conversationMessages(events: RunEvent[]): ConversationMessage[] {
  return events.flatMap(event => {
    const role = typeof event.data.role === 'string' ? event.data.role.toLowerCase() : ''
    return event.eventType === 'message' && (role === 'assistant' || role === 'user') && typeof event.data.content === 'string' && event.data.content.trim()
      ? [{ id: event.id, sequence: event.sequence, role, content: event.data.content } as ConversationMessage] : []
  })
}
export const hasUnread = (conversation: Conversation | undefined) => conversation?.messages.some(message => message.role === 'assistant' && message.sequence > conversation.lastRead) ?? false
const api = { start: createMemberConversation, continue: continueRun, details: getRunDetails, events: getRunEvents }

/** Browser-local pointers/read receipts; Agent Manager remains transcript authority.
 * Only an explicit send starts work. Polling is bounded, non-overlapping, and
 * stops for drained terminal runs. No model invocation on selection or reload. */
export function createWorldConversations(deps = api, storage: () => Storage = () => window.localStorage) {
  let snapshot: ConversationSnapshot = {}
  try {
    const saved = savedSchema.parse(JSON.parse(storage().getItem(STORAGE_KEY) ?? '[]'))
    snapshot = Object.fromEntries(saved.map(item => [conversationKey(item.member), { ...item, after: 0, messages: [], sending: false }]))
  } catch { /* Unavailable/invalid storage must not break the world. */ }
  const listeners = new Set<() => void>()
  const polling = new Set<string>()
  let cursor = 0
  const publish = (key: string, value: Conversation) => {
    snapshot = { ...snapshot, [key]: value }
    try {
      storage().setItem(STORAGE_KEY, JSON.stringify(Object.values(snapshot).map(({ member, runId, firstMessage, requestId, lastRead }) => ({ member, runId, firstMessage, requestId, lastRead }))))
    } catch { /* Conversation still works for this session. */ }
    listeners.forEach(listener => listener())
  }
  const refresh = async (key: string) => {
    const current = snapshot[key]
    if (!current?.runId || current.sending || polling.has(key)) return
    polling.add(key)
    try {
      const [run, events] = await Promise.all([deps.details(current.runId), deps.events(current.runId, { afterSequence: current.after, limit: 100 })])
      const live = snapshot[key]
      if (live !== current) return // A send/read receipt wins over an older response; next tick retries.
      const messages = [...current.messages, ...conversationMessages(events)].filter((message, index, all) => all.findIndex(other => other.id === message.id) === index).slice(-200)
      // Initial run event may include the full assembled prompt. Display only
      // the operator's first message; reference files are not chat bubbles.
      const clean = messages.filter(message => !(message.role === 'user' && message.content.includes('Operator message:\n')))
      publish(key, { ...live, run, after: Math.max(current.after, ...events.map(event => event.sequence)), messages: clean,
        error: undefined, settled: !['pending', 'starting', 'running'].includes(run.status) && events.length < 100 })
    } catch (error) {
      const live = snapshot[key]
      if (live === current) publish(key, { ...live, error: error instanceof Error ? error.message : 'Could not load replies' })
    } finally { polling.delete(key) }
  }
  return {
    getSnapshot: () => snapshot,
    subscribe: (listener: () => void) => { listeners.add(listener); return () => { listeners.delete(listener) } },
    refresh,
    async tick(selected?: string) {
      const keys = Object.keys(snapshot).filter(key => !snapshot[key]?.settled && snapshot[key]?.runId)
      const batch = keys.length ? Array.from({ length: Math.min(3, keys.length) }, (_, index) => keys[(cursor + index) % keys.length]).filter((key): key is string => key !== undefined) : []
      cursor += 3
      if (selected && snapshot[selected]?.runId) batch.push(selected)
      await Promise.all([...new Set(batch)].map(refresh))
    },
    read(key: string) {
      const current = snapshot[key]
      if (current && current.after > current.lastRead) publish(key, { ...current, lastRead: current.after })
    },
    async send(member: ConversationMember, message: string) {
      const key = conversationKey(member)
      const prior = snapshot[key]
      if (prior?.sending) return
      if (!message.trim() || new TextEncoder().encode(message).length > 16000) throw new Error('Enter a message of at most 16,000 bytes.')
      if (!prior && Object.keys(snapshot).length >= MAX_CONVERSATIONS) throw new Error('The world has 24 saved conversations. Continue an existing conversation in Agent Manager.')
      if (prior?.runId && !prior.run?.actions?.canContinue) throw new Error('Wait for the agent to finish before sending another message.')
      if (prior && !prior.runId && prior.firstMessage !== message) throw new Error('Retry the original message first; its delivery has not been confirmed.')
      const current: Conversation = prior ?? { member, requestId: crypto.randomUUID(), firstMessage: message, lastRead: 0, after: 0, messages: [], sending: false }
      publish(key, { ...current, sending: true, error: undefined })
      try {
        if (current.runId) {
          await deps.continue(current.runId, message)
          publish(key, { ...(snapshot[key] ?? current), sending: false, settled: false, run: current.run ? { ...current.run, status: 'pending', actions: undefined } : undefined })
        } else {
          const run = await deps.start({ agentId: member.id, teamId: member.teamId, message, requestId: current.requestId })
          publish(key, { ...(snapshot[key] ?? current), runId: run.id, run, sending: false, settled: false })
        }
      } catch (error) {
        publish(key, { ...(snapshot[key] ?? current), sending: false, error: error instanceof Error ? error.message : 'Could not send message' })
        throw error
      }
    },
  }
}
export type WorldConversations = ReturnType<typeof createWorldConversations>
