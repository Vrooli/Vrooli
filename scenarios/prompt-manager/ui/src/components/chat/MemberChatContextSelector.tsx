/**
 * MemberChatContextSelector - reusable agent/team context control.
 *
 * Lets the operator choose the base-agent context or one of the agent's
 * team-member contexts. The choice maps to a single ConversationIdentity and is
 * owned by the caller, so the 3-D world and the agent-detail view can share the
 * same control and pass the result to the durable session contract in
 * conversationSession.ts.
 */

import { useId, useMemo } from 'react'
import { cn } from '@/lib/utils'
import type { ConversationIdentity } from './conversationSession'

export interface ChatAgentChoice {
  id: string
  name: string
}

export interface ChatMembershipChoice {
  teamId: string
  teamName: string
  agentId: string
}

export interface MemberChatContextSelectorProps {
  agents: ChatAgentChoice[]
  memberships: ChatMembershipChoice[]
  value: ConversationIdentity
  onChange: (identity: ConversationIdentity) => void
  disabled?: boolean
  className?: string
}

const BASE_CONTEXT = 'base'

export function MemberChatContextSelector({
  agents,
  memberships,
  value,
  onChange,
  disabled = false,
  className,
}: MemberChatContextSelectorProps) {
  const agentSelectId = useId()
  const contextSelectId = useId()

  const agentMemberships = useMemo(
    () => memberships.filter((membership) => membership.agentId === value.agentId),
    [memberships, value.agentId],
  )

  const selectedAgentId = agents.some((agent) => agent.id === value.agentId) ? value.agentId : ''
  const selectedContext =
    value.teamId && agentMemberships.some((membership) => membership.teamId === value.teamId)
      ? value.teamId
      : BASE_CONTEXT

  const selectClass = cn(
    'w-full rounded-lg border border-border bg-background px-3 py-2 text-sm',
    'focus:outline-none focus:ring-2 focus:ring-ring',
    disabled && 'opacity-50 cursor-not-allowed',
  )

  return (
    <div className={cn('grid grid-cols-1 gap-3 sm:grid-cols-2', className)}>
      <div>
        <label htmlFor={agentSelectId} className="block text-sm font-medium text-muted-foreground mb-1">
          Agent
        </label>
        <select
          id={agentSelectId}
          value={selectedAgentId}
          onChange={(event) => onChange({ agentId: event.target.value, teamId: undefined })}
          disabled={disabled}
          className={selectClass}
        >
          <option value="" disabled>
            Select an agent
          </option>
          {agents.map((agent) => (
            <option key={agent.id} value={agent.id}>
              {agent.name}
            </option>
          ))}
        </select>
      </div>

      <div>
        <label htmlFor={contextSelectId} className="block text-sm font-medium text-muted-foreground mb-1">
          Context
        </label>
        <select
          id={contextSelectId}
          value={selectedContext}
          onChange={(event) =>
            onChange({
              agentId: value.agentId,
              teamId: event.target.value === BASE_CONTEXT ? undefined : event.target.value,
            })
          }
          disabled={disabled || !selectedAgentId}
          className={selectClass}
        >
          <option value={BASE_CONTEXT}>Base agent</option>
          {agentMemberships.map((membership) => (
            <option key={membership.teamId} value={membership.teamId}>
              {membership.teamName}
            </option>
          ))}
        </select>
      </div>
    </div>
  )
}
