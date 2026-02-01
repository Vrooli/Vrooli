/**
 * Agent types for the prompt-manager UI.
 *
 * API types are re-exported from @/lib/schemas (single source of truth).
 * Conversion utilities between Agent and Member formats are defined here.
 */

// Re-export API types from schemas (these include runtime validation)
export type {
  Agent,
  AgentStatus,
  AgentAppearance,
  CreateAgentRequest,
  UpdateAgentRequest,
  EffectiveSkillsResponse,
} from '@/lib/schemas'

// Re-export default colors constant
export { DEFAULT_AGENT_COLORS } from '@/lib/schemas'

// Import types for use in conversion functions
import type { Agent } from '@/lib/schemas'
import { DEFAULT_AGENT_COLORS } from '@/lib/schemas'

/**
 * Convert Agent to Member format.
 */
export function agentToMember(agent: Agent): import('./member').Member {
  return {
    id: agent.id,
    name: agent.displayName,
    bodyColor: agent.appearance?.body ?? DEFAULT_AGENT_COLORS.body,
    headColor: agent.appearance?.head ?? DEFAULT_AGENT_COLORS.head,
    accentColor: agent.appearance?.accent ?? DEFAULT_AGENT_COLORS.accent,
    skills: agent.skills,
    createdAt: agent.createdAt,
    updatedAt: agent.updatedAt,
  }
}

/**
 * Convert Member to Agent format.
 */
export function memberToAgent(member: import('./member').Member): Agent {
  return {
    id: member.id,
    displayName: member.name,
    status: 'active',
    appearance: {
      body: member.bodyColor,
      head: member.headColor,
      accent: member.accentColor,
    },
    skills: member.skills,
    createdAt: member.createdAt,
    updatedAt: member.updatedAt,
  }
}
