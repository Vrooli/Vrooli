/**
 * AgentWithAccessories - Wrapper component that combines an agent with accessories and overlays.
 * Provides a complete agent representation with all visual enhancements.
 *
 * Uses the AgentProvider DI system to resolve the agent component at runtime,
 * making the agent type truly pluggable.
 */

import { useEffect, useMemo, useRef } from 'react'
import { useFrame, useThree } from '@react-three/fiber'
import type { Group } from 'three'
import { useAgentComponent } from '../AgentProvider'
import * as behavior from '@/services/agentBehaviorService'
import { BackpackAccessory } from '../accessories/BackpackAccessory'
import { HeadAccessory } from '../accessories/HeadAccessory'
import { HeldItemAccessory } from '../accessories/HeldItemAccessory'
import { AgentOverlayGroup } from '../overlays/AgentOverlayGroup'
import { HoverGlow } from '../effects'
import { useAccessoryStore } from '@/stores/accessoryStore'
import { useWorldScaleStore } from '@/stores/worldScaleStore'
import { useHoverHighlight } from '@/hooks/useHoverHighlight'
import type { AgentProps } from '@/types/world'
import type { Agent } from '@/types/agent'
import { DEFAULT_AGENT_COLORS } from '@/types/agent'
import { usePerformanceStore } from '@/stores/performanceStore'

const NONE_ACCESSORY = { type: 'none' as const }

interface AgentWithAccessoriesProps extends Omit<AgentProps, 'agentId'> {
  /** Full agent data */
  agent: Agent
  /** Enable overlays (name tags, status icons) */
  showOverlays?: boolean
  /** Enable accessories rendering */
  showAccessories?: boolean
  /** Enable hover highlighting */
  enableHover?: boolean
  /** Whether agent is seated on furniture */
  isSeated?: boolean
  /** Rotation when seated (radians) */
  seatRotation?: number
}

/**
 * Complete agent representation with accessories and overlays.
 *
 * @example
 * ```tsx
 * <AgentWithAccessories
 *   agent={agent}
 *   position={[0, 0, 0]}
 *   selectedNodes={[]}
 *   isAnimating={false}
 *   onAgentClick={() => handleClick(agent.id)}
 * />
 * ```
 */
export function AgentWithAccessories({
  agent,
  position,
  selectedNodes: selectedNodesProp,
  isAnimating,
  onAnimationComplete,
  onAgentClick,
  colors,
  showOverlays = true,
  showAccessories = true,
  enableHover = true,
  isSeated = false,
  seatRotation = 0,
}: AgentWithAccessoriesProps) {
  // Defensive: ensure selectedNodes is always an array
  const selectedNodes = selectedNodesProp ?? []

  // Resolve agent component via DI
  const AgentComponent = useAgentComponent()

  // World scale multipliers
  const agentScale = useWorldScaleStore((state) => state.agent)
  const overlayScale = useWorldScaleStore((state) => state.overlay)

  // Subscribe only to this agent's accessory state (avoids all-agent rerenders).
  const agentAccessoryState = useAccessoryStore((state) => state.agentAccessories[agent.id])
  const accessoryDefaults = useAccessoryStore((state) => state.defaults)

  // AI_CHECK: R3F_AGENT_ACCESSORY_SUBSCRIPTION=1 | LAST: 2026-02-17
  const storedAccessories = useMemo(() => {
    return {
      head: agentAccessoryState?.accessories.head ?? accessoryDefaults.head ?? NONE_ACCESSORY,
      back: agentAccessoryState?.accessories.back ?? accessoryDefaults.back ?? NONE_ACCESSORY,
      held: agentAccessoryState?.accessories.held ?? accessoryDefaults.held ?? NONE_ACCESSORY,
    }
  }, [agentAccessoryState, accessoryDefaults])

  // Hover highlighting
  const { isHovered, hoverProps } = useHoverHighlight(agent.id, {
    enabled: enableHover,
  })

  // Merge provided colors with agent colors
  const agentColors = useMemo(
    () =>
      colors ?? {
        body: agent.appearance?.body ?? DEFAULT_AGENT_COLORS.body,
        head: agent.appearance?.head ?? DEFAULT_AGENT_COLORS.head,
        accent: agent.appearance?.accent ?? DEFAULT_AGENT_COLORS.accent,
      },
    [colors, agent.appearance]
  )

  // ===== BEHAVIOR SERVICE =====
  // Register/unregister agent with the behavior system
  useEffect(() => {
    behavior.initAgent(agent.id, position)
    return () => behavior.removeAgent(agent.id)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- only run on mount/unmount
  }, [agent.id])

  // Lock/unlock agent when seated by team (external seating)
  const prevSeatedRef = useRef(isSeated)
  useEffect(() => {
    if (isSeated && !prevSeatedRef.current) {
      behavior.lockAgent(agent.id)
    } else if (!isSeated && prevSeatedRef.current) {
      behavior.unlockAgent(agent.id)
    }
    prevSeatedRef.current = isSeated
  }, [isSeated, agent.id])

  // ===== LOCOMOTION =====
  // Smoothly interpolate the wrapper group toward the target position.
  // Children use local coordinates [0,0,0] so they move with the group.
  //
  // IMPORTANT: We store the target in a ref and drive all movement from
  // useFrame.  The <group> receives NO position prop after mount so that
  // R3F's reconciler doesn't overwrite the in-progress interpolation.
  const locomotionRef = useRef<Group>(null)
  const targetRef = useRef<[number, number, number]>(position)
  targetRef.current = position // always latest target (from props - team seating, drag, etc.)

  const { camera } = useThree()
  const facingCameraRef = useRef(true)
  const frameCountRef = useRef(0)
  const initializedRef = useRef(false)

  const LOCOMOTION_SPEED = 3 // world units per second
  const ARRIVAL_THRESHOLD = 0.05

  useFrame(({ clock }, delta) => {
    if (!locomotionRef.current) return
    const timingStart = performance.now()

    const agentId = agent.id
    frameCountRef.current++

    // Set initial facing direction on first frame
    if (!initializedRef.current) {
      locomotionRef.current.rotation.y = behavior.getDesiredYaw(agentId)
      initializedRef.current = true
    }

    // 1. Report current position to behavior service
    const pos = locomotionRef.current.position
    behavior.updatePosition(agentId, [pos.x, pos.y, pos.z])

    // 2. Evaluate behavior (staggered per agent, ~every 60 frames)
    if (behavior.shouldEvaluateThisFrame(agentId, frameCountRef.current)) {
      behavior.evaluateBehavior(agentId, clock.getElapsedTime(), position)
    }

    // 3. Tick behavior to get target and yaw
    const result = behavior.tickAgent(agentId, clock.getElapsedTime())
    if (result && !isSeated) {
      // Behavior service wants the agent to move somewhere
      if (result.targetPosition) {
        targetRef.current = result.targetPosition
      }
    }

    // 4. Locomotion: interpolate toward target
    const [tx, ty, tz] = targetRef.current
    const dx = tx - pos.x
    const dy = ty - pos.y
    const dz = tz - pos.z
    const dist = Math.sqrt(dx * dx + dy * dy + dz * dz)

    if (dist < ARRIVAL_THRESHOLD) {
      pos.set(tx, ty, tz)
      // Smoothly lerp rotation toward desired yaw after arriving
      const desired = behavior.getDesiredYaw(agentId)
      let yawDiff = desired - locomotionRef.current.rotation.y
      while (yawDiff > Math.PI) yawDiff -= Math.PI * 2
      while (yawDiff < -Math.PI) yawDiff += Math.PI * 2
      if (Math.abs(yawDiff) > 0.01) {
        locomotionRef.current.rotation.y += yawDiff * Math.min(1, 4 * delta)
      }
    } else {
      const step = Math.min(LOCOMOTION_SPEED * delta, dist)
      const ratio = step / dist
      pos.x += dx * ratio
      pos.y += dy * ratio
      pos.z += dz * ratio

      // Face direction of movement while walking
      if (dist > ARRIVAL_THRESHOLD * 2) {
        const targetYaw = Math.atan2(dx, dz)
        const currentYaw = locomotionRef.current.rotation.y
        let yawDiff = targetYaw - currentYaw
        while (yawDiff > Math.PI) yawDiff -= Math.PI * 2
        while (yawDiff < -Math.PI) yawDiff += Math.PI * 2
        locomotionRef.current.rotation.y += yawDiff * Math.min(1, 5 * delta)
      }
    }

    // 5. Compute isFacingCamera (every 5 frames, matching LOD cadence)
    if (frameCountRef.current % 5 === 0) {
      facingCameraRef.current = behavior.isFacingCamera(
        locomotionRef.current.rotation.y,
        [pos.x, pos.y, pos.z],
        [camera.position.x, camera.position.y, camera.position.z],
      )
    }

    if (frameCountRef.current % 30 === 0) {
      usePerformanceStore.getState().recordSubsystemSample(
        'agent.locomotion',
        performance.now() - timingStart
      )
    }
  })

  // Local origin for children (parent group handles world position)
  const LOCAL_ORIGIN: [number, number, number] = [0, 0, 0]

  // Initial position only — useFrame drives all subsequent movement
  const initialPosition = useRef(position)

  return (
    <group ref={locomotionRef} position={initialPosition.current} {...(enableHover ? hoverProps : {})}>
      <group scale={[agentScale, agentScale, agentScale]}>
        {/* Base agent model (resolved via DI) */}
        <AgentComponent
          agentId={agent.id}
          position={LOCAL_ORIGIN}
          selectedNodes={selectedNodes}
          isAnimating={isAnimating}
          onAnimationComplete={onAnimationComplete}
          onAgentClick={() => {
            // Trigger face-camera on click
            if (locomotionRef.current) {
              const pos = locomotionRef.current.position
              behavior.triggerFaceCamera(
                agent.id,
                performance.now() / 1000,
                [camera.position.x, camera.position.y, camera.position.z],
                [pos.x, pos.y, pos.z],
              )
            }
            onAgentClick?.()
          }}
          colors={agentColors}
          isSeated={isSeated}
          seatRotation={seatRotation}
          isFacingCamera={facingCameraRef.current}
        />

        {/* Accessories */}
        {showAccessories && (
          <group>
            {/* Back accessory from store */}
            {storedAccessories.back.type !== 'none' && (
              <BackpackAccessory
                type={storedAccessories.back.type}
                scale={storedAccessories.back.scale}
              />
            )}

            {/* Head accessory from store */}
            {storedAccessories.head.type !== 'none' && (
              <HeadAccessory
                type={storedAccessories.head.type}
                color={storedAccessories.head.color}
              />
            )}

            {/* Held item from store */}
            {storedAccessories.held.type !== 'none' && (
              <HeldItemAccessory
                type={storedAccessories.held.type}
                hand={storedAccessories.held.hand}
                color={storedAccessories.held.color}
              />
            )}
          </group>
        )}

        {/* Hover glow effect - centered on body (0.4 = body radius offset) */}
        <HoverGlow
          isActive={isHovered}
          position={[0, 0.4, 0]}
          size={0.8}
          color={agentColors.accent || '#6366f1'}
          intensity={0.6}
        />
      </group>

      {/* Overlays (name tag, status, speech) - outside agent scale, uses own overlay scale */}
      {showOverlays && (
        <AgentOverlayGroup
          agentId={agent.id}
          name={agent.displayName}
          position={LOCAL_ORIGIN}
          isHovered={isHovered}
          overlayScale={overlayScale}
        />
      )}
    </group>
  )
}
