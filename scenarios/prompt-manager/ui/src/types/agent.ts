// Agent types aligned with the Go API (api/agents/models.go)
// These match the exact response shapes from the agent API

/**
 * Agent represents an entity in the skill tree that can be assigned skills.
 * This is the new name for what was previously called "Member".
 */
export interface Agent {
  id: string
  displayName: string
  status: AgentStatus
  appearance?: AgentAppearance
  skills: string[] // Skill IDs assigned through relations
  createdAt: string
  updatedAt: string
}

/**
 * Agent status values
 */
export type AgentStatus = 'active' | 'inactive' | 'suspended'

/**
 * AgentAppearance represents visual appearance for 3D UI
 */
export interface AgentAppearance {
  body: string // hex color
  head: string // hex color
  accent: string // hex color
}

/**
 * CreateAgentRequest matches the API's CreateRequest type
 */
export interface CreateAgentRequest {
  id?: string
  displayName: string
  appearance?: AgentAppearance
  skills?: string[]
}

/**
 * UpdateAgentRequest matches the API's UpdateRequest type
 */
export interface UpdateAgentRequest {
  displayName?: string
  status?: AgentStatus
  appearance?: AgentAppearance
  skills?: string[]
}

/**
 * EffectiveSkillsResponse from /agents/{id}/effective-skills
 */
export interface EffectiveSkillsResponse {
  agentId: string
  teamId?: string
  skills: string[]
}

/**
 * Default agent colors for new agents
 */
export const DEFAULT_AGENT_COLORS: AgentAppearance = {
  body: '#4f46e5', // indigo-600
  head: '#818cf8', // indigo-400
  accent: '#c7d2fe', // indigo-200
}

/**
 * Convert Agent to legacy Member format for backward compatibility
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
 * Convert legacy Member to Agent format
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
