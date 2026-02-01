/**
 * Agent Animation Service - State machine and animation utilities for 3D agents.
 *
 * Provides:
 * - Agent state machine for behavior management
 * - Animation calculations (look rotation, idle sway, wave, celebration)
 * - Easing functions
 * - Interpolation helpers
 *
 * Note: For API operations, use agentService.ts instead.
 */

import type { AgentState } from '@/types/world'

// ============================================================================
// State Machine
// ============================================================================

/**
 * State machine configuration for agent behaviors.
 */
interface StateConfig {
  name: AgentState
  duration: number
  canInterrupt: boolean
  nextStates: AgentState[]
}

const STATE_CONFIG: Record<AgentState, StateConfig> = {
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
 * Agent state machine class.
 */
export class AgentStateMachine {
  private currentState: AgentState = 'idle'
  private stateStartTime: number = Date.now()
  private listeners: Set<(state: AgentState) => void> = new Set()

  constructor(initialState: AgentState = 'idle') {
    this.currentState = initialState
    this.stateStartTime = Date.now()
  }

  getState(): AgentState {
    return this.currentState
  }

  getStateTime(): number {
    return Date.now() - this.stateStartTime
  }

  isStateComplete(): boolean {
    const config = STATE_CONFIG[this.currentState]
    return this.getStateTime() >= config.duration
  }

  transition(newState: AgentState): boolean {
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

  forceTransition(newState: AgentState): void {
    this.currentState = newState
    this.stateStartTime = Date.now()
    this.notifyListeners()
  }

  subscribe(listener: (state: AgentState) => void): () => void {
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
