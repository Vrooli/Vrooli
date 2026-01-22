/**
 * Avatar animation state machine and behavior service.
 * Manages avatar states, transitions, and animations.
 */

import type { AvatarState } from '@/types/skilltree'

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

  /**
   * Get the current state.
   */
  getState(): AvatarState {
    return this.currentState
  }

  /**
   * Get time elapsed in current state (ms).
   */
  getStateTime(): number {
    return Date.now() - this.stateStartTime
  }

  /**
   * Check if state duration has elapsed.
   */
  isStateComplete(): boolean {
    const config = STATE_CONFIG[this.currentState]
    return this.getStateTime() >= config.duration
  }

  /**
   * Attempt to transition to a new state.
   */
  transition(newState: AvatarState): boolean {
    const currentConfig = STATE_CONFIG[this.currentState]

    // Check if transition is allowed
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

  /**
   * Force transition (bypasses rules).
   */
  forceTransition(newState: AvatarState): void {
    this.currentState = newState
    this.stateStartTime = Date.now()
    this.notifyListeners()
  }

  /**
   * Subscribe to state changes.
   */
  subscribe(listener: (state: AvatarState) => void): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  private notifyListeners(): void {
    this.listeners.forEach((listener) => listener(this.currentState))
  }
}

/**
 * Calculate head rotation to look at a target position.
 * Returns [rotationX, rotationY] in radians.
 */
export function calculateLookRotation(
  headPosition: [number, number, number],
  targetPosition: { x: number; y: number; z?: number },
  maxRotation: number = Math.PI / 3
): [number, number] {
  const dx = targetPosition.x - headPosition[0]
  const dy = (targetPosition.y || 0) - headPosition[1]
  const dz = (targetPosition.z || 5) - headPosition[2]

  // Calculate angles
  const horizontalAngle = Math.atan2(dx, dz)
  const verticalAngle = Math.atan2(dy, Math.sqrt(dx * dx + dz * dz))

  // Clamp to max rotation
  const clampedHorizontal = Math.max(
    -maxRotation,
    Math.min(maxRotation, horizontalAngle)
  )
  const clampedVertical = Math.max(
    -maxRotation / 2,
    Math.min(maxRotation / 2, verticalAngle)
  )

  return [clampedVertical, clampedHorizontal]
}

/**
 * Calculate body sway based on time.
 * Returns offset values for idle animation.
 */
export function calculateIdleSway(time: number): {
  positionOffset: [number, number, number]
  rotationOffset: [number, number, number]
} {
  const slowCycle = time * 0.5
  const fastCycle = time * 1.2

  return {
    positionOffset: [
      Math.sin(slowCycle) * 0.02,
      Math.sin(fastCycle) * 0.03 + Math.sin(slowCycle * 0.7) * 0.02,
      Math.cos(slowCycle * 0.8) * 0.01,
    ],
    rotationOffset: [
      Math.sin(slowCycle * 0.6) * 0.02,
      Math.sin(slowCycle * 0.4) * 0.01,
      Math.sin(slowCycle * 0.9) * 0.015,
    ],
  }
}

/**
 * Calculate wave animation.
 * Returns arm rotation in radians.
 */
export function calculateWaveAnimation(
  progress: number
): [number, number, number] {
  // progress is 0-1 over the wave duration
  const wavePhase = progress * Math.PI * 4 // Two full waves

  return [
    Math.sin(wavePhase) * 0.5, // Upper arm rotation
    Math.sin(wavePhase * 2) * 0.3 + 0.5, // Forearm rotation
    Math.sin(wavePhase) * 0.2, // Wrist rotation
  ]
}

/**
 * Calculate celebration animation.
 * Returns various animation parameters.
 */
export function calculateCelebrationAnimation(progress: number): {
  scale: number
  rotation: number
  particleBurst: boolean
} {
  const bouncePhase = progress * Math.PI * 6

  return {
    scale: 1 + Math.sin(bouncePhase) * 0.1,
    rotation: progress * Math.PI * 2, // Full spin
    particleBurst: progress < 0.1 || progress > 0.9,
  }
}

/**
 * Easing functions for smooth animations.
 */
export const easing = {
  linear: (t: number) => t,
  easeInOut: (t: number) => t < 0.5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2,
  easeOut: (t: number) => 1 - Math.pow(1 - t, 3),
  easeIn: (t: number) => t * t * t,
  bounce: (t: number) => {
    const n1 = 7.5625
    const d1 = 2.75
    if (t < 1 / d1) return n1 * t * t
    if (t < 2 / d1) return n1 * (t -= 1.5 / d1) * t + 0.75
    if (t < 2.5 / d1) return n1 * (t -= 2.25 / d1) * t + 0.9375
    return n1 * (t -= 2.625 / d1) * t + 0.984375
  },
  elastic: (t: number) => {
    const c4 = (2 * Math.PI) / 3
    return t === 0
      ? 0
      : t === 1
      ? 1
      : Math.pow(2, -10 * t) * Math.sin((t * 10 - 0.75) * c4) + 1
  },
}

/**
 * Interpolate between two values.
 */
export function lerp(start: number, end: number, t: number): number {
  return start + (end - start) * t
}

/**
 * Interpolate between two 3D positions.
 */
export function lerpPosition(
  start: [number, number, number],
  end: [number, number, number],
  t: number
): [number, number, number] {
  return [lerp(start[0], end[0], t), lerp(start[1], end[1], t), lerp(start[2], end[2], t)]
}
