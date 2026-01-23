// Member types aligned with the Go API (api/members/models.go)
// These match the exact response shapes from the member API

/**
 * Member represents a visual character in the skill tree.
 * Members can have skills assigned to them.
 */
export interface Member {
  id: string
  name: string
  bodyColor: string // hex color
  headColor: string // hex color
  accentColor: string // hex color
  skills: string[] // Skill IDs assigned to this member
  createdAt: string
  updatedAt: string
}

/**
 * CreateMemberRequest matches the API's CreateRequest type
 */
export interface CreateMemberRequest {
  id?: string
  name: string
  bodyColor: string
  headColor: string
  accentColor: string
  skills?: string[]
}

/**
 * UpdateMemberRequest matches the API's UpdateRequest type
 */
export interface UpdateMemberRequest {
  name?: string
  bodyColor?: string
  headColor?: string
  accentColor?: string
  skills?: string[]
}

/**
 * Default member colors for new members
 */
export const DEFAULT_MEMBER_COLORS = {
  bodyColor: '#4f46e5', // indigo-600
  headColor: '#818cf8', // indigo-400
  accentColor: '#c7d2fe', // indigo-200
} as const
