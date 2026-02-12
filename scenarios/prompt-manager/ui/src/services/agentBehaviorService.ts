/**
 * Agent Behavior Service - Manages autonomous agent behaviors in the 3D world.
 *
 * Pure module-level state (no React dependencies). Each agent's `useFrame` in
 * AgentWithAccessories reads/writes from the maps here. Behavior evaluation
 * runs every ~60 frames per agent, naturally staggered by per-agent offsets.
 *
 * Behavior types (weighted random):
 * - idle (40%): Stay put, face a random direction
 * - walk-to-furniture (25%): Walk to an available seat, sit visually
 * - socialize (20%): Walk to another idle agent, face each other
 * - wander (15%): Walk to a random point within bounds
 */

import { useFurnitureStore } from '@/stores/furnitureStore'

// ===== Types =====

export type BehaviorType = 'idle' | 'walk-to-furniture' | 'socialize' | 'wander'

export interface AgentBehaviorState {
  agentId: string
  behavior: BehaviorType
  targetPosition: [number, number, number] | null
  targetFacingYaw: number | null
  behaviorStartTime: number
  behaviorDuration: number
  socialPartner: string | null
  locked: boolean
  initialFacingYaw: number
  desiredYaw: number
  faceCamera: boolean
  faceCameraStartTime: number
  voluntarySeat: { furnitureId: string; seatIndex: number } | null
  /** Per-agent frame offset for staggered evaluation (0-59) */
  frameOffset: number
  /** Random initial delay before first behavior evaluation (seconds) */
  initialDelay: number
}

interface TickResult {
  targetPosition: [number, number, number] | null
  desiredYaw: number
}

// ===== Module-level state =====

const behaviorMap = new Map<string, AgentBehaviorState>()
const positionMap = new Map<string, [number, number, number]>()

// ===== Constants =====

const TWO_PI = Math.PI * 2
const FACE_CAMERA_DURATION = 5 // seconds
const WANDER_RADIUS = 3 // max distance from home position
const SOCIALIZE_DISTANCE = 1.2 // distance between socializing agents
const MAX_MOVER_RATIO = 0.5 // if >50% of agents are walking, bias toward idle

// Behavior weights and durations
const BEHAVIOR_CONFIG: Record<BehaviorType, { weight: number; minDuration: number; maxDuration: number }> = {
  'idle': { weight: 40, minDuration: 3, maxDuration: 8 },
  'walk-to-furniture': { weight: 25, minDuration: 5, maxDuration: 15 },
  'socialize': { weight: 20, minDuration: 4, maxDuration: 10 },
  'wander': { weight: 15, minDuration: 2, maxDuration: 6 },
}

// ===== Utility functions =====

/** Deterministic hash from string to a positive integer. */
function hashString(str: string): number {
  let hash = 0
  for (let i = 0; i < str.length; i++) {
    hash = ((hash << 5) - hash) + str.charCodeAt(i)
    hash |= 0
  }
  return Math.abs(hash)
}

/** Deterministic yaw from agent ID, in [0, 2*PI). */
export function computeInitialYaw(agentId: string): number {
  const h = hashString(agentId)
  return (h % 10000) / 10000 * TWO_PI
}

/** Normalize an angle to [-PI, PI]. */
function normalizeAngle(angle: number): number {
  while (angle > Math.PI) angle -= TWO_PI
  while (angle < -Math.PI) angle += TWO_PI
  return angle
}

/** Simple seeded pseudo-random from a seed integer. Returns [0, 1). */
function seededRandom(seed: number): number {
  const x = Math.sin(seed * 12.9898 + 78.233) * 43758.5453
  return x - Math.floor(x)
}

/** Pick a random float in [min, max) using a seed. */
function randomRange(min: number, max: number, seed: number): number {
  return min + seededRandom(seed) * (max - min)
}

// ===== Core API =====

/**
 * Register an agent in the behavior system.
 * Call on mount from AgentWithAccessories.
 */
export function initAgent(agentId: string, homePosition: [number, number, number]): void {
  if (behaviorMap.has(agentId)) return

  const h = hashString(agentId)
  const initialYaw = computeInitialYaw(agentId)

  behaviorMap.set(agentId, {
    agentId,
    behavior: 'idle',
    targetPosition: null,
    targetFacingYaw: null,
    behaviorStartTime: 0,
    behaviorDuration: randomRange(3, 8, h + 1), // initial idle duration
    socialPartner: null,
    locked: false,
    initialFacingYaw: initialYaw,
    desiredYaw: initialYaw,
    faceCamera: false,
    faceCameraStartTime: 0,
    voluntarySeat: null,
    frameOffset: h % 60,
    initialDelay: randomRange(0, 5, h + 2),
  })

  positionMap.set(agentId, [...homePosition])
}

/**
 * Remove an agent from the behavior system.
 * Call on unmount from AgentWithAccessories.
 */
export function removeAgent(agentId: string): void {
  const state = behaviorMap.get(agentId)
  if (state) {
    // Clean up social partner reference
    if (state.socialPartner) {
      const partner = behaviorMap.get(state.socialPartner)
      if (partner && partner.socialPartner === agentId) {
        partner.socialPartner = null
        partner.behavior = 'idle'
        partner.behaviorDuration = 0 // trigger re-evaluation
      }
    }
    // Clean up voluntary seat
    if (state.voluntarySeat) {
      useFurnitureStore.getState().unseatAgent(agentId)
    }
  }
  behaviorMap.delete(agentId)
  positionMap.delete(agentId)
}

/**
 * Lock an agent (team seating or dragging). Skips all behavior evaluation.
 */
export function lockAgent(agentId: string): void {
  const state = behaviorMap.get(agentId)
  if (!state) return

  // End social behavior if active
  if (state.socialPartner) {
    const partner = behaviorMap.get(state.socialPartner)
    if (partner && partner.socialPartner === agentId) {
      partner.socialPartner = null
      partner.behavior = 'idle'
      partner.behaviorDuration = 0
    }
    state.socialPartner = null
  }

  // Release voluntary seat if any
  if (state.voluntarySeat) {
    useFurnitureStore.getState().unseatAgent(agentId)
    state.voluntarySeat = null
  }

  state.locked = true
}

/**
 * Unlock an agent after team/drag release. Resumes behavior evaluation.
 */
export function unlockAgent(agentId: string): void {
  const state = behaviorMap.get(agentId)
  if (!state) return
  state.locked = false
  state.behavior = 'idle'
  state.behaviorDuration = 0 // trigger re-evaluation next tick
  state.targetPosition = null
}

/**
 * Write current world position (called every frame from useFrame).
 */
export function updatePosition(agentId: string, pos: [number, number, number]): void {
  positionMap.set(agentId, pos)
}

/**
 * Trigger face-camera behavior on click. Computes yaw from agent to camera.
 * Auto-resets after FACE_CAMERA_DURATION seconds.
 */
export function triggerFaceCamera(
  agentId: string,
  clockTime: number,
  cameraPos: [number, number, number],
  agentPos: [number, number, number],
): void {
  const state = behaviorMap.get(agentId)
  if (!state) return

  const dx = cameraPos[0] - agentPos[0]
  const dz = cameraPos[2] - agentPos[2]
  const yaw = Math.atan2(dx, dz)

  state.faceCamera = true
  state.faceCameraStartTime = clockTime
  state.desiredYaw = yaw
}

/**
 * Get the current desired yaw for an agent (for rotation lerp).
 */
export function getDesiredYaw(agentId: string): number {
  return behaviorMap.get(agentId)?.desiredYaw ?? 0
}

/**
 * Check if an agent is facing the camera within a 90-degree forward cone.
 * The agent's "forward" is along +Z in local space, which maps to the yaw direction.
 */
export function isFacingCamera(
  agentYaw: number,
  agentPos: [number, number, number],
  cameraPos: [number, number, number],
): boolean {
  // Direction from agent to camera
  const dx = cameraPos[0] - agentPos[0]
  const dz = cameraPos[2] - agentPos[2]
  const toCameraYaw = Math.atan2(dx, dz)

  // Angle difference between agent facing and direction to camera
  const diff = Math.abs(normalizeAngle(agentYaw - toCameraYaw))

  // 90-degree cone = 45 degrees each side = PI/4
  return diff < Math.PI / 4
}

/**
 * Evaluate and potentially switch to a new behavior.
 * Called every ~60 frames (staggered per agent).
 */
export function evaluateBehavior(
  agentId: string,
  clockTime: number,
  homePosition: [number, number, number],
): void {
  const state = behaviorMap.get(agentId)
  if (!state) return
  if (state.locked) return

  // Respect face-camera window
  if (state.faceCamera) {
    if (clockTime - state.faceCameraStartTime < FACE_CAMERA_DURATION) return
    state.faceCamera = false
  }

  // Apply initial delay (only for first evaluation)
  if (state.behaviorStartTime === 0 && state.initialDelay > 0) {
    state.behaviorStartTime = clockTime
    state.behaviorDuration = state.initialDelay
    state.initialDelay = 0
    return
  }

  // Check if current behavior has expired
  if (clockTime - state.behaviorStartTime < state.behaviorDuration) return

  // Clean up previous behavior
  cleanupBehavior(agentId, state)

  // Count how many agents are currently walking
  let totalAgents = 0
  let walkingAgents = 0
  for (const [, s] of behaviorMap) {
    if (s.locked) continue
    totalAgents++
    if (s.behavior === 'walk-to-furniture' || s.behavior === 'wander' || s.behavior === 'socialize') {
      if (s.targetPosition) walkingAgents++
    }
  }

  // Max-mover cap: if too many are walking, strongly bias toward idle
  const tooManyMoving = totalAgents > 0 && (walkingAgents / totalAgents) > MAX_MOVER_RATIO

  // Roll for next behavior
  const seed = hashString(agentId) + Math.floor(clockTime * 100)
  const newBehavior = pickBehavior(seed, tooManyMoving, agentId)

  state.behavior = newBehavior
  state.behaviorStartTime = clockTime
  state.targetPosition = null
  state.targetFacingYaw = null

  const config = BEHAVIOR_CONFIG[newBehavior]
  state.behaviorDuration = randomRange(config.minDuration, config.maxDuration, seed + 10)

  switch (newBehavior) {
    case 'idle':
      setupIdle(state, seed)
      break
    case 'walk-to-furniture':
      if (!setupFurniture(state, agentId, seed)) {
        // Fall back to idle if no furniture available
        state.behavior = 'idle'
        setupIdle(state, seed)
      }
      break
    case 'socialize':
      if (!setupSocialize(state, agentId, seed)) {
        // Fall back to idle if no partner available
        state.behavior = 'idle'
        setupIdle(state, seed)
      }
      break
    case 'wander':
      setupWander(state, homePosition, seed)
      break
  }
}

/**
 * Tick an agent's behavior and return target position and yaw.
 * Returns null if locked (external code handles position).
 */
export function tickAgent(agentId: string, clockTime: number): TickResult | null {
  const state = behaviorMap.get(agentId)
  if (!state || state.locked) return null

  // Check if face-camera has expired
  if (state.faceCamera && clockTime - state.faceCameraStartTime >= FACE_CAMERA_DURATION) {
    state.faceCamera = false
  }

  // Check if social partner got locked mid-socialize
  if (state.socialPartner) {
    const partner = behaviorMap.get(state.socialPartner)
    if (!partner || partner.locked || partner.socialPartner !== agentId) {
      state.socialPartner = null
      state.behavior = 'idle'
      state.behaviorDuration = 0
      state.targetPosition = null
    }
  }

  return {
    targetPosition: state.targetPosition,
    desiredYaw: state.desiredYaw,
  }
}

/**
 * Check if a specific agent should have behavior evaluation run this frame.
 * Returns true every 60 frames, staggered per agent.
 */
export function shouldEvaluateThisFrame(agentId: string, frameCount: number): boolean {
  const state = behaviorMap.get(agentId)
  if (!state) return false
  return frameCount % 60 === state.frameOffset
}

/**
 * Check if a specific agent is locked (team-seated or dragged).
 */
export function isLocked(agentId: string): boolean {
  return behaviorMap.get(agentId)?.locked ?? false
}

/**
 * Get the voluntary seat info for an agent, if any.
 */
export function getVoluntarySeat(agentId: string): { furnitureId: string; seatIndex: number } | null {
  return behaviorMap.get(agentId)?.voluntarySeat ?? null
}

// ===== Behavior setup helpers =====

function cleanupBehavior(agentId: string, state: AgentBehaviorState): void {
  // Clean up voluntary seat
  if (state.voluntarySeat) {
    useFurnitureStore.getState().unseatAgent(agentId)
    state.voluntarySeat = null
  }

  // Clean up social partner
  if (state.socialPartner) {
    const partner = behaviorMap.get(state.socialPartner)
    if (partner && partner.socialPartner === agentId) {
      partner.socialPartner = null
      partner.behavior = 'idle'
      partner.behaviorDuration = 0
    }
    state.socialPartner = null
  }
}

function setupIdle(state: AgentBehaviorState, seed: number): void {
  // Face a random direction
  state.desiredYaw = seededRandom(seed + 20) * TWO_PI
  state.targetPosition = null
}

function setupFurniture(state: AgentBehaviorState, agentId: string, seed: number): boolean {
  const store = useFurnitureStore.getState()

  // Get all furniture with available seats in the active scene
  const sceneKey = Object.keys(store.scenes).find((key) => {
    const items = store.scenes[key as keyof typeof store.scenes]
    return items && items.length > 0
  })
  if (!sceneKey) return false

  const furnitureList = store.scenes[sceneKey as keyof typeof store.scenes] ?? []
  const available = furnitureList.filter((f) => store.hasAvailableSeats(f.id))
  if (available.length === 0) return false

  // Pick random furniture
  const furnitureIndex = Math.floor(seededRandom(seed + 30) * available.length)
  const furniture = available[furnitureIndex]
  if (!furniture) return false

  // Try to seat the agent
  const seated = store.seatAgent(agentId, furniture.id)
  if (!seated) return false

  // Get the seat position for navigation
  const seatPos = store.getAgentSeatPosition(agentId)
  if (!seatPos) {
    store.unseatAgent(agentId)
    return false
  }

  state.targetPosition = seatPos.position
  state.desiredYaw = seatPos.rotation
  state.voluntarySeat = {
    furnitureId: furniture.id,
    seatIndex: 0, // seatAgent picks the first available
  }

  return true
}

function setupSocialize(state: AgentBehaviorState, agentId: string, seed: number): boolean {
  // Find eligible partners: unlocked, not already socializing, not this agent
  const candidates: string[] = []
  for (const [id, s] of behaviorMap) {
    if (id === agentId) continue
    if (s.locked) continue
    if (s.socialPartner) continue
    if (s.behavior === 'walk-to-furniture' && s.voluntarySeat) continue
    candidates.push(id)
  }
  if (candidates.length === 0) return false

  // Pick random partner
  const partnerIndex = Math.floor(seededRandom(seed + 40) * candidates.length)
  const partnerId = candidates[partnerIndex]
  if (!partnerId) return false
  const partnerState = behaviorMap.get(partnerId)
  if (!partnerState) return false

  // Get current positions
  const myPos = positionMap.get(agentId)
  const partnerPos = positionMap.get(partnerId)
  if (!myPos || !partnerPos) return false

  // Compute midpoint between the two agents
  const midX = (myPos[0] + partnerPos[0]) / 2
  const midZ = (myPos[2] + partnerPos[2]) / 2

  // Position agents SOCIALIZE_DISTANCE apart, centered on midpoint
  const angle = seededRandom(seed + 41) * TWO_PI
  const halfDist = SOCIALIZE_DISTANCE / 2
  const myTargetX = midX + Math.cos(angle) * halfDist
  const myTargetZ = midZ + Math.sin(angle) * halfDist
  const partnerTargetX = midX - Math.cos(angle) * halfDist
  const partnerTargetZ = midZ - Math.sin(angle) * halfDist

  // Set up this agent
  state.socialPartner = partnerId
  state.targetPosition = [myTargetX, myPos[1], myTargetZ]
  state.desiredYaw = Math.atan2(partnerTargetX - myTargetX, partnerTargetZ - myTargetZ)

  // Set up partner
  partnerState.behavior = 'socialize'
  partnerState.socialPartner = agentId
  partnerState.targetPosition = [partnerTargetX, partnerPos[1], partnerTargetZ]
  partnerState.desiredYaw = Math.atan2(myTargetX - partnerTargetX, myTargetZ - partnerTargetZ)
  partnerState.behaviorStartTime = state.behaviorStartTime
  partnerState.behaviorDuration = state.behaviorDuration

  return true
}

function setupWander(
  state: AgentBehaviorState,
  homePosition: [number, number, number],
  seed: number,
): void {
  // Random point within WANDER_RADIUS of home position
  const angle = seededRandom(seed + 50) * TWO_PI
  const radius = seededRandom(seed + 51) * WANDER_RADIUS
  const x = homePosition[0] + Math.cos(angle) * radius
  const z = homePosition[2] + Math.sin(angle) * radius

  state.targetPosition = [x, homePosition[1], z]
  // Face direction of travel (will be overridden by locomotion while moving)
  state.desiredYaw = Math.atan2(
    x - (positionMap.get(state.agentId)?.[0] ?? homePosition[0]),
    z - (positionMap.get(state.agentId)?.[2] ?? homePosition[2]),
  )
}

function pickBehavior(seed: number, tooManyMoving: boolean, agentId: string): BehaviorType {
  // If too many agents are already moving, heavily favor idle
  if (tooManyMoving) {
    const roll = seededRandom(seed)
    if (roll < 0.75) return 'idle'
    if (roll < 0.90) return 'wander'
    return 'idle'
  }

  const totalWeight = Object.values(BEHAVIOR_CONFIG).reduce((sum, c) => sum + c.weight, 0)
  let roll = seededRandom(seed) * totalWeight
  void agentId // reserved for future per-agent weighting

  for (const [behavior, config] of Object.entries(BEHAVIOR_CONFIG)) {
    roll -= config.weight
    if (roll <= 0) return behavior as BehaviorType
  }

  return 'idle'
}

// ===== Test helpers =====

/**
 * Reset all module-level state. Only for use in tests.
 * @internal
 */
export function _resetForTesting(): void {
  behaviorMap.clear()
  positionMap.clear()
}
