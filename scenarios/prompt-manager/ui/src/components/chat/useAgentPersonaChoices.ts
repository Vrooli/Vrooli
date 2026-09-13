/**
 * Shared persona-context choices for agent-page chat and prompt preview.
 *
 * Both the conversation surface and the prompt preview must offer the same
 * base-agent and team-member contexts, so the roster-derived agent/membership
 * list is computed once here. The page agent is always present even before the
 * roster loads, which keeps the identity resolvable on a cold open.
 */

import { useMemo } from 'react'
import { useWorldRoster } from '@/world/data/roster'
import type { ChatAgentChoice, ChatMembershipChoice } from './MemberChatContextSelector'

export interface AgentPersonaChoices {
  agents: ChatAgentChoice[]
  memberships: ChatMembershipChoice[]
}

export function useAgentPersonaChoices(agentId: string): AgentPersonaChoices {
  const roster = useWorldRoster()

  const agents = useMemo<ChatAgentChoice[]>(() => {
    const list = roster.agents.map((agent) => ({ id: agent.id, name: agent.name }))
    return list.some((agent) => agent.id === agentId)
      ? list
      : [{ id: agentId, name: agentId }, ...list]
  }, [roster.agents, agentId])

  const memberships = useMemo<ChatMembershipChoice[]>(
    () =>
      roster.teams.flatMap((team) =>
        team.memberIds.map((memberId) => ({
          teamId: team.id,
          teamName: team.name,
          agentId: memberId,
        })),
      ),
    [roster.teams],
  )

  return { agents, memberships }
}
