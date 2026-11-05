/**
 * useMemberConversation - durable, idempotent persona conversation state.
 *
 * The hook is the consuming half of the conversation contract delivered in
 * conversationSession.ts. It binds one immutable identity to one server-owned
 * run and keeps the operator's turns idempotent across retry and reload:
 *
 * - the first turn persists `requestId` before dispatch, so a lost response
 *   resumes the same conversation instead of starting a second one;
 * - each later turn persists a `pendingTurn` request id and reuses it while
 *   retrying that same message;
 * - a reload reads the durable session and reloads the recorded run rather than
 *   sending anything;
 * - a failure records which operation failed (start vs continue).
 *
 * The world HUD and the agent-detail view are expected to share this hook.
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  clearConversationSession,
  clearPendingStart,
  conversationIdentityKey,
  createConversationSession,
  readConversationSession,
  readPendingStart,
  updateConversationSession,
  writeConversationSession,
  writePendingStart,
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
  type RunEvent,
} from '@/services/heartbeatService'

export type ConversationOperation = 'start' | 'continue'

export interface ConversationRecovery {
  operation: ConversationOperation
  message: string
}

export interface UseMemberConversationOptions {
  identity: ConversationIdentity
  storage?: SessionStorage
  pollIntervalMs?: number
}

export interface UseMemberConversationResult {
  identityKey: string
  run: RunDetails | null
  events: RunEvent[]
  loading: boolean
  sending: boolean
  recovery: ConversationRecovery | null
  send: (message: string) => Promise<void>
  end: () => void
}

const ACTIVE_STATUSES = new Set(['pending', 'starting', 'running'])

function eventOptions(afterSequence: number): { afterSequence?: number } | undefined {
  return afterSequence >= 0 ? { afterSequence } : undefined
}

export function useMemberConversation({
  identity,
  storage,
  pollIntervalMs = 2000,
}: UseMemberConversationOptions): UseMemberConversationResult {
  const agentId = identity.agentId.trim()
  const teamId = identity.teamId?.trim() || undefined
  const resolvedIdentity = useMemo<ConversationIdentity>(
    () => ({ agentId, teamId }),
    [agentId, teamId],
  )
  const identityKey = conversationIdentityKey(resolvedIdentity)

  const [run, setRun] = useState<RunDetails | null>(null)
  const [events, setEvents] = useState<RunEvent[]>([])
  const [loading, setLoading] = useState(true)
  const [sending, setSending] = useState(false)
  const [recovery, setRecovery] = useState<ConversationRecovery | null>(null)
  const afterSequence = useRef(-1)
  const runId = run?.id ?? null

  // Reload the durable session whenever the identity changes.
  useEffect(() => {
    const flag = { cancelled: false }
    afterSequence.current = -1
    setRun(null)
    setEvents([])
    setRecovery(null)
    const session = readConversationSession(resolvedIdentity, storage)
    if (!session) {
      setLoading(false)
      return
    }
    setLoading(true)
    void (async () => {
      try {
        const [details, initialEvents] = await Promise.all([
          getRunDetails(session.runId),
          getRunEvents(session.runId),
        ])
        if (flag.cancelled) return
        setRun(details)
        setEvents(initialEvents)
        afterSequence.current = initialEvents.reduce(
          (max, event) => Math.max(max, event.sequence),
          -1,
        )
      } catch (error) {
        if (flag.cancelled) return
        setRecovery({
          operation: 'continue',
          message: error instanceof Error ? error.message : 'Could not reload the conversation',
        })
      } finally {
        if (!flag.cancelled) setLoading(false)
      }
    })()
    return () => {
      flag.cancelled = true
    }
  }, [resolvedIdentity, storage])

  const refresh = useCallback(async (targetRunId?: string) => {
    const id = targetRunId ?? runId
    if (!id) return
    try {
      const [details, newEvents] = await Promise.all([
        getRunDetails(id),
        getRunEvents(id, eventOptions(afterSequence.current)),
      ])
      setRun(details)
      if (newEvents.length > 0) {
        afterSequence.current = newEvents.reduce(
          (max, event) => Math.max(max, event.sequence),
          afterSequence.current,
        )
        setEvents((previous) => {
          const known = new Set(previous.map((event) => event.id))
          return [...previous, ...newEvents.filter((event) => !known.has(event.id))]
        })
      }
    } catch {
      // A transient poll failure is not a recovery state; the next tick retries.
    }
  }, [runId])

  // Poll while the accepted run is still working.
  useEffect(() => {
    if (!runId || !run || !ACTIVE_STATUSES.has(run.status)) return
    const interval = setInterval(() => void refresh(), pollIntervalMs)
    return () => clearInterval(interval)
  }, [runId, run, refresh, pollIntervalMs])

  const send = useCallback(
    async (rawMessage: string) => {
      const message = rawMessage.trim()
      if (!message || sending) return
      setRecovery(null)
      const session = readConversationSession(resolvedIdentity, storage)
      if (!session) {
        const pendingStart = readPendingStart(resolvedIdentity, storage)
        if (pendingStart && pendingStart.message !== message) {
          setRecovery({
            operation: 'start',
            message: 'Retry the original message first; its delivery has not been confirmed.',
          })
          return
        }
        const requestId = pendingStart?.requestId ?? newConversationRequestId()
        writePendingStart(resolvedIdentity, { requestId, message }, storage)
        setSending(true)
        try {
          const started = await createMemberConversation({
            teamId: resolvedIdentity.teamId,
            agentId: resolvedIdentity.agentId,
            message,
            requestId,
          })
          writeConversationSession(
            createConversationSession(resolvedIdentity, started.id, requestId),
            storage,
          )
          clearPendingStart(resolvedIdentity, storage)
          afterSequence.current = -1
          setEvents([])
          setRun(started)
        } catch (error) {
          setRecovery({
            operation: 'start',
            message: error instanceof Error ? error.message : 'Could not start the conversation',
          })
        } finally {
          setSending(false)
        }
        return
      }
      const pendingTurn = session.pendingTurn
      if (pendingTurn && pendingTurn.message !== message) {
        setRecovery({
          operation: 'continue',
          message: 'Retry the previous message first; its delivery has not been confirmed.',
        })
        return
      }
      const requestId = pendingTurn?.requestId ?? newConversationRequestId()
      updateConversationSession(resolvedIdentity, { pendingTurn: { requestId, message } }, storage)
      setSending(true)
      try {
        await continueRun(session.runId, message, { requestId })
        updateConversationSession(resolvedIdentity, { pendingTurn: undefined }, storage)
        await refresh(session.runId)
      } catch (error) {
        setRecovery({
          operation: 'continue',
          message: error instanceof Error ? error.message : 'Could not send the turn',
        })
      } finally {
        setSending(false)
      }
    },
    [resolvedIdentity, storage, sending, refresh],
  )

  const end = useCallback(() => {
    clearConversationSession(resolvedIdentity, storage)
    clearPendingStart(resolvedIdentity, storage)
    afterSequence.current = -1
    setRun(null)
    setEvents([])
    setRecovery(null)
  }, [resolvedIdentity, storage])

  return { identityKey, run, events, loading, sending, recovery, send, end }
}
