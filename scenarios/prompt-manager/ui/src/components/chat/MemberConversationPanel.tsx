/**
 * MemberConversationPanel - reusable persona conversation surface.
 *
 * Ties the shared context selector to the durable conversation hook so the
 * world HUD and the agent-detail view can mount one component and get the same
 * identity, reload, retry and recovery behavior. The selector is controlled by
 * the caller; the panel owns only the conversation for the selected identity.
 */

import { cn } from '@/lib/utils'
import { ChatPanel } from './ChatPanel'
import {
  MemberChatContextSelector,
  type ChatAgentChoice,
  type ChatMembershipChoice,
} from './MemberChatContextSelector'
import type { ConversationIdentity, SessionStorage } from './conversationSession'
import { useMemberConversation } from './useMemberConversation'

export interface MemberConversationPanelProps {
  agents: ChatAgentChoice[]
  memberships: ChatMembershipChoice[]
  identity: ConversationIdentity
  onIdentityChange: (identity: ConversationIdentity) => void
  /** Hide the inline selector when a shared selector owns the context. */
  showContextSelector?: boolean
  storage?: SessionStorage
  pollIntervalMs?: number
  className?: string
}

const RECOVERY_PREFIX: Record<'start' | 'continue', string> = {
  start: 'Could not start the conversation: ',
  continue: 'Could not send the turn: ',
}

export function MemberConversationPanel({
  agents,
  memberships,
  identity,
  onIdentityChange,
  showContextSelector = true,
  storage,
  pollIntervalMs,
  className,
}: MemberConversationPanelProps) {
  const { run, events, loading, sending, recovery, send } = useMemberConversation({
    identity,
    storage,
    pollIntervalMs,
  })

  const membership = identity.teamId
    ? memberships.find(
        (item) => item.teamId === identity.teamId && item.agentId === identity.agentId,
      )
    : undefined
  const contextLabel = membership?.teamName ?? (identity.teamId ? identity.teamId : 'Base agent')
  const runOrigin = run?.teamId ?? run?.agentId

  return (
    <section
      aria-label="Member conversation"
      data-testid="member-conversation"
      className={cn('flex h-full flex-col gap-3', className)}
    >
      {showContextSelector && (
        <MemberChatContextSelector
          agents={agents}
          memberships={memberships}
          value={identity}
          onChange={onIdentityChange}
          disabled={sending}
        />
      )}

      <p className="text-xs text-muted-foreground" data-testid="conversation-context">
        Context: {contextLabel}
        {runOrigin ? ` · run origin ${runOrigin}` : ''}
      </p>

      {recovery && (
        <p
          role="alert"
          data-testid="conversation-recovery"
          data-operation={recovery.operation}
          className="rounded-lg bg-destructive/20 px-3 py-2 text-sm text-destructive"
        >
          {RECOVERY_PREFIX[recovery.operation]}
          {recovery.message}
        </p>
      )}

      <div className="min-h-0 flex-1 overflow-hidden">
        <ChatPanel
          run={run}
          events={events}
          eventsLoading={loading}
          onContinue={send}
          canStart
          onStart={send}
          startError={recovery?.operation === 'start' ? recovery.message : null}
        />
      </div>
    </section>
  )
}
