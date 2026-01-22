// Avatar types aligned with the Go API (api/avatars/models.go)
// These match the exact response shapes from the avatar API

/**
 * Avatar represents a visual character in the skill tree.
 * Avatars can have prompts (skills) assigned to them.
 */
export interface Avatar {
  id: string
  name: string
  bodyColor: string // hex color
  headColor: string // hex color
  accentColor: string // hex color
  skills: string[] // Prompt IDs assigned to this avatar
  createdAt: string
  updatedAt: string
}

/**
 * CreateAvatarRequest matches the API's CreateRequest type
 */
export interface CreateAvatarRequest {
  id?: string
  name: string
  bodyColor: string
  headColor: string
  accentColor: string
  skills?: string[]
}

/**
 * UpdateAvatarRequest matches the API's UpdateRequest type
 */
export interface UpdateAvatarRequest {
  name?: string
  bodyColor?: string
  headColor?: string
  accentColor?: string
  skills?: string[]
}

/**
 * Default avatar colors for new avatars
 */
export const DEFAULT_AVATAR_COLORS = {
  bodyColor: '#4f46e5', // indigo-600
  headColor: '#818cf8', // indigo-400
  accentColor: '#c7d2fe', // indigo-200
} as const
