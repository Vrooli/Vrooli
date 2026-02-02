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

import { useRef, useEffect, useMemo } from 'react'
import { OrbitControls, Stars } from '@react-three/drei'
import { useThree } from '@react-three/fiber'
import { AgentWithAccessories } from './agents/AgentWithAccessories'
import { WorldErrorBoundary } from './WorldErrorBoundary'
import { DragPlane, PlacementPlane } from './interaction'
import { FurnitureManager } from './furniture'
import { DecorationManager } from './decorations'
import { PerformanceMonitor, FPSOverlay } from './performance'
import { DynamicLighting, DynamicFog, DynamicSky, CelestialBody, Moon, ProceduralClouds, GroundSurface } from './rendering'
import { BoundaryOutline } from './rendering/BoundaryOutline'
import { useInteractionStore } from '@/stores/interactionStore'
import { useEnvironmentStore } from '@/stores/environmentStore'
import { useIsPlacing } from '@/stores/worldEditorStore'
import { calculateStarOpacity } from '@/lib/sky/sunPosition'
import type { Agent } from '@/types/agent'
import type { FurnitureInstance } from '@/types/furniture'

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
  cursorPosition: { x: number; y: number } | null
  /** All agents with their positions */
  agentsWithPositions: AgentWithPosition[]
  /** Called when an agent is clicked, with agent ID and position */
  onAgentClick?: (agentId: string, position: [number, number, number]) => void
  /** Called when furniture is clicked */
  onFurnitureClick?: (furniture: FurnitureInstance) => void
  isDarkMode?: boolean
  /** Whether to show FPS overlay */
  showFpsOverlay?: boolean
  /** Whether to enable automatic performance adjustment */
  autoAdjustPerformance?: boolean
}

export function WorldScene({
  cameraState,
  selectedNodeIds,
  cursorPosition,
  agentsWithPositions,
  onAgentClick,
  onFurnitureClick,
  isDarkMode = true,
  showFpsOverlay = false,
  autoAdjustPerformance = true,
}: WorldSceneProps) {
  const controlsRef = useRef<OrbitControlsRef>(null)
  const { camera } = useThree()

  // Disable orbit controls during drag
  const isDragging = useInteractionStore((state) => state.isDragging)
  const isPlacing = useIsPlacing()

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
          <Stars
            radius={80}
            depth={40}
            count={2000}
            factor={4 * starOpacity}
            fade
            speed={1}
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
        maxPolarAngle={Math.PI * 0.85}
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
        draggable={!isPlacing}
        onFurnitureClick={onFurnitureClick}
      />
      <DecorationManager
        interactive={!isPlacing}
        draggable={!isPlacing}
      />

      {/* Render all agents with accessories and overlays */}
      {agentsWithPositions.map(({ agent, position, isSeated = false, seatRotation = 0 }) => (
        <AgentWithAccessories
          key={agent.id}
          agent={agent}
          position={position}
          cursorPosition={cursorPosition}
          selectedNodes={selectedNodeIds}
          isAnimating={false}
          isSeated={isSeated}
          seatRotation={seatRotation}
          onAgentClick={() => onAgentClick?.(agent.id, position)}
          showOverlays
          showAccessories
        />
      ))}
    </>
  )
}
