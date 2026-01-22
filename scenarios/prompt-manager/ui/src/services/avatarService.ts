/**
 * Avatar Service - Avatar state machine, animations, and API functions.
 *
 * Provides:
 * - Avatar state machine for behavior management
 * - Animation calculations (look rotation, idle sway, wave, celebration)
 * - Easing functions
 * - API wrapper with caching
 */

import { api } from '@/lib/api'
import type { Avatar, CreateAvatarRequest, UpdateAvatarRequest } from '@/types/avatar'
import type { AvatarState } from '@/types/skilltree'

// ============================================================================
// State Machine
// ============================================================================

/**
 * State machine configuration for avatar behaviors.
 */
interface StateConfig {
  name: AvatarState
  duration: number
  canInterrupt: boolean
  nextStates: AvatarState[]
}

const STATE_CONFIG: Record<AvatarState, StateConfig> = {
  idle: {
    name: 'idle',
    duration: Infinity,
    canInterrupt: true,
    nextStates: ['looking', 'waving', 'thinking'],
  },
  looking: {
    name: 'looking',
    duration: Infinity,
    canInterrupt: true,
    nextStates: ['idle', 'waving', 'thinking'],
  },
  waving: {
    name: 'waving',
    duration: 1500,
    canInterrupt: false,
    nextStates: ['idle', 'looking'],
  },
  celebrating: {
    name: 'celebrating',
    duration: 2000,
    canInterrupt: false,
    nextStates: ['idle'],
  },
  thinking: {
    name: 'thinking',
    duration: 3000,
    canInterrupt: true,
    nextStates: ['idle', 'looking'],
  },
}

/**
 * Avatar state machine class.
 */
export class AvatarStateMachine {
  private currentState: AvatarState = 'idle'
  private stateStartTime: number = Date.now()
  private listeners: Set<(state: AvatarState) => void> = new Set()

  constructor(initialState: AvatarState = 'idle') {
    this.currentState = initialState
    this.stateStartTime = Date.now()
  }

  getState(): AvatarState {
    return this.currentState
  }

  getStateTime(): number {
    return Date.now() - this.stateStartTime
  }

  isStateComplete(): boolean {
    const config = STATE_CONFIG[this.currentState]
    return this.getStateTime() >= config.duration
  }

  transition(newState: AvatarState): boolean {
    const currentConfig = STATE_CONFIG[this.currentState]

    if (!currentConfig.canInterrupt && !this.isStateComplete()) {
      return false
    }

    if (!currentConfig.nextStates.includes(newState)) {
      return false
    }

    this.currentState = newState
    this.stateStartTime = Date.now()
    this.notifyListeners()
    return true
  }

  forceTransition(newState: AvatarState): void {
    this.currentState = newState
    this.stateStartTime = Date.now()
    this.notifyListeners()
  }

  subscribe(listener: (state: AvatarState) => void): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  private notifyListeners(): void {
    for (const listener of this.listeners) {
      listener(this.currentState)
    }
  }
}

// ============================================================================
// Animation Calculations
// ============================================================================

/**
 * Calculate head rotation to look at cursor position.
 */
export function calculateLookRotation(
  headPos: [number, number, number],
  target: { x: number; y: number; z?: number },
  maxRotation: number = Math.PI / 3
): [number, number] {
  const dx = target.x - headPos[0]
  const dy = target.y - headPos[1]
  const dz = (target.z ?? 5) - headPos[2]

  const horizontalDist = Math.sqrt(dx * dx + dz * dz)

  let rotY = Math.atan2(dx, dz)
  let rotX = -Math.atan2(dy, horizontalDist)

  // Clamp rotations
  rotY = Math.max(-maxRotation, Math.min(maxRotation, rotY))
  rotX = Math.max(-maxRotation / 2, Math.min(maxRotation / 2, rotX))

  return [rotX, rotY]
}

/**
 * Calculate idle sway animation offsets.
 */
export function calculateIdleSway(time: number): {
  positionOffset: [number, number, number]
  rotationOffset: [number, number, number]
} {
  const swayAmplitude = 0.02
  const swayFrequency = 0.5

  return {
    positionOffset: [
      Math.sin(time * swayFrequency) * swayAmplitude,
      Math.sin(time * swayFrequency * 1.3) * swayAmplitude * 0.5,
      Math.cos(time * swayFrequency * 0.7) * swayAmplitude * 0.3,
    ],
    rotationOffset: [
      Math.sin(time * swayFrequency * 0.8) * swayAmplitude * 2,
      Math.sin(time * swayFrequency * 1.1) * swayAmplitude,
      Math.cos(time * swayFrequency * 0.9) * swayAmplitude * 0.5,
    ],
  }
}

/**
 * Calculate wave animation rotation values.
 */
export function calculateWaveAnimation(progress: number): [number, number, number] {
  const wavePhase = progress * Math.PI * 4
  const armRaise = Math.sin(progress * Math.PI) * 0.8
  const waveMotion = Math.sin(wavePhase) * 0.3 * Math.sin(progress * Math.PI)

  return [
    armRaise,
    waveMotion,
    Math.sin(progress * Math.PI * 2) * 0.1,
  ]
}

/**
 * Calculate celebration animation values.
 */
export function calculateCelebrationAnimation(progress: number): {
  scale: number
  rotation: number
  particleBurst: boolean
} {
  const scale = 1 + Math.sin(progress * Math.PI * 2) * 0.1
  const rotation = progress * Math.PI * 2

  // Trigger particle bursts at start and near end
  const particleBurst = progress < 0.1 || progress > 0.9

  return { scale, rotation, particleBurst }
}

// ============================================================================
// Easing Functions
// ============================================================================

export const easing = {
  linear: (t: number): number => t,

  easeInOut: (t: number): number => {
    return t < 0.5
      ? 2 * t * t
      : 1 - Math.pow(-2 * t + 2, 2) / 2
  },

  easeOut: (t: number): number => {
    return 1 - Math.pow(1 - t, 2)
  },

  easeIn: (t: number): number => {
    return t * t
  },

  bounce: (t: number): number => {
    const n1 = 7.5625
    const d1 = 2.75

    if (t < 1 / d1) {
      return n1 * t * t
    } else if (t < 2 / d1) {
      return n1 * (t -= 1.5 / d1) * t + 0.75
    } else if (t < 2.5 / d1) {
      return n1 * (t -= 2.25 / d1) * t + 0.9375
    } else {
      return n1 * (t -= 2.625 / d1) * t + 0.984375
    }
  },

  elastic: (t: number): number => {
    if (t === 0) return 0
    if (t === 1) return 1
    const p = 0.3
    const a = 1
    const s = p / 4
    return a * Math.pow(2, -10 * t) * Math.sin((t - s) * (2 * Math.PI) / p) + 1
  },
}

// ============================================================================
// Interpolation Helpers
// ============================================================================

/**
 * Linear interpolation between two values.
 */
export function lerp(start: number, end: number, t: number): number {
  return start + (end - start) * t
}

/**
 * Linear interpolation between two 3D positions.
 */
export function lerpPosition(
  start: [number, number, number],
  end: [number, number, number],
  t: number
): [number, number, number] {
  return [
    lerp(start[0], end[0], t),
    lerp(start[1], end[1], t),
    lerp(start[2], end[2], t),
  ]
}

// Cache configuration
const CACHE_TTL_MS = 5000 // 5 seconds

// Cache state
interface CacheEntry<T> {
  data: T
  timestamp: number
}

let avatarsCache: CacheEntry<Avatar[]> | null = null

/**
 * Check if cache entry is still valid.
 */
function isCacheValid<T>(entry: CacheEntry<T> | null): entry is CacheEntry<T> {
  if (!entry) return false
  return Date.now() - entry.timestamp < CACHE_TTL_MS
}

/**
 * Invalidate all caches. Call after mutations.
 */
export function invalidateCache(): void {
  avatarsCache = null
}

/**
 * Get all avatars with caching.
 *
 * @param forceRefresh - Skip cache and fetch fresh data
 * @returns Array of all avatars
 */
export async function getAvatars(forceRefresh = false): Promise<Avatar[]> {
  if (!forceRefresh && isCacheValid(avatarsCache)) {
    return avatarsCache.data
  }

  const data = await api.getAvatars()
  avatarsCache = { data, timestamp: Date.now() }
  return data
}

/**
 * Get a single avatar by ID.
 * Uses cached data if available, otherwise fetches.
 *
 * @param id - Avatar ID
 * @returns The avatar, or undefined if not found
 */
export async function getAvatar(id: string): Promise<Avatar | undefined> {
  // Try cache first
  if (isCacheValid(avatarsCache)) {
    const cached = avatarsCache.data.find((a) => a.id === id)
    if (cached) return cached
  }

  // Fetch from API
  try {
    return await api.getAvatar(id)
  } catch (error) {
    console.error(`[avatarService] Failed to get avatar ${id}:`, error)
    return undefined
  }
}

/**
 * Create a new avatar.
 *
 * @param request - Create request data
 * @returns The created avatar
 */
export async function createAvatar(request: CreateAvatarRequest): Promise<Avatar> {
  const avatar = await api.createAvatar(request)
  invalidateCache()
  return avatar
}

/**
 * Update an avatar.
 *
 * @param id - Avatar ID to update
 * @param updates - Fields to update
 * @returns The updated avatar
 */
export async function updateAvatar(id: string, updates: UpdateAvatarRequest): Promise<Avatar> {
  const avatar = await api.updateAvatar(id, updates)
  invalidateCache()
  return avatar
}

/**
 * Delete an avatar.
 *
 * @param id - Avatar ID to delete
 */
export async function deleteAvatar(id: string): Promise<void> {
  await api.deleteAvatar(id)
  invalidateCache()
}
