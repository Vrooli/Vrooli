/**
 * AgentChatTab - agent-page chat with explicit persona context selection.
 *
 * Mounts the shared persona conversation surface on the agent detail workflow.
 * The owning agent is the default base-agent context; every team the agent
 * belongs to is offered as a team-member context. The selection can be owned by
 * the caller (the agent editor lifts it so the prompt preview stays on the same
 * context), or by this tab when it is mounted standalone.
 *
 * The context list comes from the world roster (the same team/agent graph the
 * 3-D world uses), which keeps base-agent and member identities consistent
 * across both surfaces.
 */

import { useEffect, useState } from 'react'
import { MemberConversationPanel } from './MemberConversationPanel'
import type { ConversationIdentity, SessionStorage } from './conversationSession'
import { useAgentPersonaChoices } from './useAgentPersonaChoices'

export interface AgentChatTabProps {
  /** Agent whose page hosts the chat. */
  agentId: string
  /** Optional context to open on first render (a team the agent belongs to). */
  initialTeamId?: string
  /** Controlled identity; when provided the caller owns context selection. */
  identity?: ConversationIdentity
  /** Notified when the selection changes in controlled mode. */
  onIdentityChange?: (identity: ConversationIdentity) => void
  /** Hide the inline selector when a shared selector owns the context. */
  showContextSelector?: boolean
  storage?: SessionStorage
  pollIntervalMs?: number
  className?: string
}

export function AgentChatTab({
  agentId,
  initialTeamId,
  identity: controlledIdentity,
  onIdentityChange,
  showContextSelector = true,
  storage,
  pollIntervalMs,
  className,
}: AgentChatTabProps) {
  const { agents, memberships } = useAgentPersonaChoices(agentId)

  const [uncontrolledIdentity, setUncontrolledIdentity] = useState<ConversationIdentity>(() => ({
    agentId,
    teamId: initialTeamId,
  }))

  // A different agent page resets to that agent's base context; a stale team
  // selection from the previous agent must never leak into the new identity.
  useEffect(() => {
    setUncontrolledIdentity({ agentId, teamId: initialTeamId })
  }, [agentId, initialTeamId])

  const identity = controlledIdentity ?? uncontrolledIdentity
  const handleIdentityChange = onIdentityChange ?? setUncontrolledIdentity

  return (
    <MemberConversationPanel
      agents={agents}
      memberships={memberships}
      identity={identity}
      onIdentityChange={handleIdentityChange}
      showContextSelector={showContextSelector}
      storage={storage}
      pollIntervalMs={pollIntervalMs}
      className={className}
    />
  )
}
