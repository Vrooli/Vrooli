/**
 * WorldScene - Composes the 3D scene with agents, lights, and controls.
 *
 * Note: 3D skill nodes have been removed; this scene focuses on agent display
 * with an ambient environment.
 *
 * Includes:
 * - Performance monitoring with FPS tracking and auto-adjustment
 * - LOD system for optimizing cursor tracking at scale
 * - Memory cleanup for proper asset disposal
 * - Dynamic sky with continuous time-based sun/moon positioning
 * - Procedural clouds that respond to time of day
 */
// DOC: docs/concepts/3D-WORLD-ARCHITECTURE.md#component-hierarchy

import { useRef, useEffect, useMemo, useCallback } from 'react'
import { OrbitControls } from '@react-three/drei'
import { useThree } from '@react-three/fiber'
import * as THREE from 'three'
import { AgentWithAccessories } from './agents/AgentWithAccessories'
import { WorldErrorBoundary } from './WorldErrorBoundary'
import { DragPlane, DraggableObject, PlacementPlane } from './interaction'
import { FurnitureManager } from './furniture'
import { DecorationManager } from './decorations'
import { PerformanceMonitor, FPSOverlay } from './performance'
import { DynamicLighting, DynamicFog, DynamicSky, CelestialBody, Moon, ProceduralClouds, GroundSurface } from './rendering'
import { BoundaryOutline } from './rendering/BoundaryOutline'
import { useInteractionStore } from '@/stores/interactionStore'
import { useEnvironmentStore } from '@/stores/environmentStore'
import { useIsPlacing, useIsEditMode } from '@/stores/worldEditorStore'
import { calculateStarOpacity } from '@/lib/sky/sunPosition'
import { applyPlacementConstraints } from '@/lib/world'
import type { Agent } from '@/types/agent'
import type { FurnitureInstance } from '@/types/furniture'
import type { DecorationInstance } from '@/types/decoration'

/**
 * Custom star field with fog disabled and rotation tied to timeValue.
 * Rotates along the same orbital plane as the sun/moon so all celestial
 * objects move together as time progresses.
 */
interface StarFieldProps {
  count: number
  radius: number
  depth: number
  opacity: number
  timeValue: number
}

function StarField({ count, radius, depth, opacity, timeValue }: StarFieldProps) {
  const geometry = useMemo(() => {
    const g = new THREE.BufferGeometry()
    const positions = new Float32Array(count * 3)
    const colors = new Float32Array(count * 3)
    for (let i = 0; i < count; i++) {
      const r = radius + Math.random() * depth
      const theta = Math.random() * Math.PI * 2
      const phi = Math.acos(2 * Math.random() - 1)
      positions[i * 3] = r * Math.sin(phi) * Math.cos(theta)
      positions[i * 3 + 1] = r * Math.sin(phi) * Math.sin(theta)
      positions[i * 3 + 2] = r * Math.cos(phi)
      // Vary brightness so some stars pop more than others
      const brightness = 0.4 + Math.random() * 0.6
      colors[i * 3] = brightness
      colors[i * 3 + 1] = brightness
      colors[i * 3 + 2] = brightness
    }
    g.setAttribute('position', new THREE.BufferAttribute(positions, 3))
    g.setAttribute('color', new THREE.BufferAttribute(colors, 3))
    return g
  }, [count, radius, depth])

  // Rotate in sync with the sun/moon arc.
  // Uses the same hourAngle formula as calculateSunPosition so stars
  // sweep east-to-west at the same rate as the celestial bodies.
  const rotation = useMemo<[number, number, number]>(() => {
    const h = ((timeValue % 24) + 24) % 24
    const hourAngle = ((h - 12) / 24) * Math.PI * 2
    // Tilt to match the sun's orbital plane (y*0.8, z*0.3 in calculateSunPosition)
    const tiltAngle = Math.atan2(0.3, 0.8)
    return [tiltAngle, 0, -hourAngle]
  }, [timeValue])

  return (
    <points geometry={geometry} rotation={rotation}>
      <pointsMaterial
        size={2.5}
        sizeAttenuation={false}
        transparent
        opacity={opacity}
        depthWrite={false}
        fog={false}
        vertexColors
        blending={THREE.AdditiveBlending}
      />
    </points>
  )
}

/** Type for OrbitControls ref - drei doesn't export proper types */
type OrbitControlsRef = {
  target: { set: (x: number, y: number, z: number) => void }
  update: () => void
}

interface CameraState {
  position: [number, number, number]
  target: [number, number, number]
  zoom: number
}

/** Agent with its computed position in the scene */
export interface AgentWithPosition {
  agent: Agent
  position: [number, number, number]
  isSeated?: boolean
  seatRotation?: number
}

interface WorldSceneProps {
  cameraState: CameraState
  selectedNodeIds: string[]
  /** All agents with their positions */
  agentsWithPositions: AgentWithPosition[]
  /** Called when an agent is clicked, with agent ID and position */
  onAgentClick?: (agentId: string, position: [number, number, number]) => void
  /** Called when furniture is clicked */
  onFurnitureClick?: (furniture: FurnitureInstance) => void
  /** Called when a decoration is clicked */
  onDecorationClick?: (decoration: DecorationInstance) => void
  /** Called when an agent is repositioned via drag */
  onAgentPositionChange?: (agentId: string, newPosition: [number, number, number]) => void
  isDarkMode?: boolean
  /** Whether to show FPS overlay */
  showFpsOverlay?: boolean
  /** Whether to enable automatic performance adjustment */
  autoAdjustPerformance?: boolean
}

export function WorldScene({
  cameraState,
  selectedNodeIds,
  agentsWithPositions,
  onAgentClick,
  onFurnitureClick,
  onDecorationClick,
  onAgentPositionChange,
  isDarkMode = true,
  showFpsOverlay = false,
  autoAdjustPerformance = true,
}: WorldSceneProps) {
  const controlsRef = useRef<OrbitControlsRef>(null)
  const { camera } = useThree()

  // Disable orbit controls during drag
  const isDragging = useInteractionStore((state) => state.isDragging)
  const isPlacing = useIsPlacing()
  const isEditMode = useIsEditMode()

  // Environment config for ground styling
  const currentEnv = useEnvironmentStore((state) => state.current)
  const timeValue = useEnvironmentStore((state) => state.timeValue)
  const groundConfig = currentEnv.ground
  const boundaryConfig = currentEnv.boundary
  const sceneType = currentEnv.type
  const groundSize = groundConfig.size ?? 30
  const boundarySize = boundaryConfig.visible
    ? (boundaryConfig.size ?? groundSize * 2)
    : groundSize
  const dragPlaneSize = Math.max(groundSize, boundarySize, 10)
  const groundY = groundConfig.position ?? 0

  // Placement constraints for agent drag (same as furniture/decorations)
  const placementConfig = currentEnv.placement
  const agentDraggable = isEditMode && !isPlacing
  const constrainAgentPosition = useMemo(() => {
    return (position: [number, number, number]): [number, number, number] =>
      applyPlacementConstraints(position, {
        placement: placementConfig,
        boundary: boundaryConfig,
        groundSize,
      })
  }, [placementConfig, boundaryConfig, groundSize])

  const handleAgentPositionChange = useCallback(
    (agentId: string, newPosition: [number, number, number]) => {
      onAgentPositionChange?.(agentId, newPosition)
    },
    [onAgentPositionChange]
  )

  // Calculate star opacity based on continuous time
  const starOpacity = useMemo(() => calculateStarOpacity(timeValue), [timeValue])
  const showStars = starOpacity > 0

  // Update camera position when state changes
  useEffect(() => {
    camera.position.set(...cameraState.position)
    if (controlsRef.current) {
      controlsRef.current.target.set(...cameraState.target)
      controlsRef.current.update()
    }
  }, [camera, cameraState])

  return (
    <>
      {/* Performance Monitoring */}
      <PerformanceMonitor
        enabled
        autoAdjust={autoAdjustPerformance}
        enableMemoryCleanup
        enableLOD
      />
      {showFpsOverlay && <FPSOverlay position={[-6, 4, 0]} detailed />}

      {/* Dynamic Lighting from environment config */}
      <DynamicLighting enableShadows shadowMapSize={2048} />

      {/* Dynamic Fog from environment config */}
      <DynamicFog />

      {/* Dynamic Sky dome - renders based on continuous time */}
      <WorldErrorBoundary componentName="DynamicSky" minimal>
        <DynamicSky />
        <CelestialBody />
        <Moon />
      </WorldErrorBoundary>

      {/* Procedural Clouds - time-based coloring, disabled for abstract-space */}
      {sceneType !== 'abstract-space' && (
        <WorldErrorBoundary componentName="ProceduralClouds" minimal>
          <ProceduralClouds />
        </WorldErrorBoundary>
      )}

      {/* Stars - fade in at dusk, fade out at dawn */}
      {showStars && (
        <WorldErrorBoundary componentName="Stars" minimal>
          <StarField
            count={3000}
            radius={50}
            depth={25}
            opacity={starOpacity}
            timeValue={timeValue}
          />
        </WorldErrorBoundary>
      )}

      {/* Controls - disabled during drag */}
      <OrbitControls
        ref={controlsRef as React.Ref<never>}
        enabled={!isDragging && !isPlacing}
        enableDamping
        dampingFactor={0.05}
        minDistance={3}
        maxDistance={30}
        maxPolarAngle={Math.PI * 0.45}
        minPolarAngle={Math.PI * 0.15}
      />

      {/* Drag plane - catches pointer events during drag */}
      <DragPlane y={groundY} size={dragPlaneSize} />
      <PlacementPlane y={groundY} size={dragPlaneSize} />

      {/* Ground surface (grid or textured plane) */}
      <GroundSurface
        ground={groundConfig}
        groundY={groundY}
        isDarkMode={isDarkMode}
      />

      {/* World boundary outline */}
      <BoundaryOutline groundSize={groundSize} />

      {/* Furniture and Decorations */}
      <FurnitureManager
        interactive={!isPlacing}
        draggable={isEditMode && !isPlacing}
        onFurnitureClick={onFurnitureClick}
      />
      <DecorationManager
        interactive={!isPlacing}
        draggable={isEditMode && !isPlacing}
        onDecorationClick={onDecorationClick}
      />

      {/* Render all agents with accessories and overlays */}
      {agentsWithPositions.map(({ agent, position, isSeated = false, seatRotation = 0 }) => {
        const agentComponent = (
          <AgentWithAccessories
            key={agent.id}
            agent={agent}
            position={agentDraggable ? [0, 0, 0] : position}
            selectedNodes={selectedNodeIds}
            isAnimating={false}
            isSeated={isSeated}
            seatRotation={seatRotation}
            onAgentClick={() => onAgentClick?.(agent.id, position)}
            showOverlays
            showAccessories
          />
        )

        if (agentDraggable && !isSeated) {
          return (
            <DraggableObject
              key={agent.id}
              objectId={`agent-${agent.id}`}
              position={position}
              onPositionChange={(pos) => handleAgentPositionChange(agent.id, pos)}
              constrainPosition={constrainAgentPosition}
            >
              {agentComponent}
            </DraggableObject>
          )
        }

        return agentComponent
      })}
    </>
  )
}
