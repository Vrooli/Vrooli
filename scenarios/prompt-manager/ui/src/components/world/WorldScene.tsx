/**
 * WorldScene - Composes the 3D scene with members, lights, and controls.
 *
 * Note: 3D skill nodes have been removed in favor of a 2D overlay (SkillSelectionOverlay).
 * This scene now focuses on member display with ambient environment.
 *
 * Includes:
 * - Performance monitoring with FPS tracking and auto-adjustment
 * - LOD system for optimizing cursor tracking at scale
 * - Memory cleanup for proper asset disposal
 */
// DOC: docs/concepts/3D-WORLD-ARCHITECTURE.md#component-hierarchy

import { useRef, useEffect } from 'react'
import { OrbitControls, Stars } from '@react-three/drei'
import { useThree } from '@react-three/fiber'
import { MemberWithAccessories } from './members/MemberWithAccessories'
import { WorldErrorBoundary } from './WorldErrorBoundary'
import { DragPlane, PlacementPlane } from './interaction'
import { FurnitureManager } from './furniture'
import { DecorationManager } from './decorations'
import { PerformanceMonitor, FPSOverlay } from './performance'
import { DynamicLighting, DynamicFog, DynamicSky, CelestialBody } from './rendering'
import { BoundaryOutline } from './rendering/BoundaryOutline'
import { useInteractionStore } from '@/stores/interactionStore'
import { useEnvironmentStore } from '@/stores/environmentStore'
import { useIsPlacing } from '@/stores/worldEditorStore'
import type { Member } from '@/types/member'
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

/** Member with its computed position in the scene */
export interface MemberWithPosition {
  member: Member
  position: [number, number, number]
  isSeated?: boolean
  seatRotation?: number
}

interface WorldSceneProps {
  cameraState: CameraState
  selectedNodeIds: string[]
  cursorPosition: { x: number; y: number } | null
  /** All members with their positions */
  membersWithPositions: MemberWithPosition[]
  /** Called when a member is clicked, with member ID and position */
  onMemberClick?: (memberId: string, position: [number, number, number]) => void
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
  membersWithPositions,
  onMemberClick,
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
  const groundConfig = currentEnv.ground
  const boundaryConfig = currentEnv.boundary
  const isNightTime = currentEnv.timeOfDay === 'night'
  const groundSize = groundConfig.size ?? 30
  const boundarySize = boundaryConfig.visible
    ? (boundaryConfig.size ?? groundSize * 2)
    : groundSize
  const dragPlaneSize = Math.max(groundSize, boundarySize, 10)
  const groundY = groundConfig.position ?? 0


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

      {/* Dynamic Sky dome - renders gradient based on time of day */}
      <WorldErrorBoundary componentName="DynamicSky" minimal>
        <DynamicSky />
        <CelestialBody />
      </WorldErrorBoundary>

      {/* Stars - only show in dark mode or night time */}
      {(isDarkMode || isNightTime) && (
        <WorldErrorBoundary componentName="Stars" minimal>
          <Stars radius={80} depth={40} count={2000} factor={4} fade speed={1} />
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

      {/* Grid helper - respects environment ground config */}
      {groundConfig.visible && groundConfig.type === 'grid' && (
        <gridHelper
          args={[
            groundConfig.size ?? 30,
            groundConfig.divisions ?? 30,
            groundConfig.color ?? (isDarkMode ? '#1e293b' : '#e2e8f0'),
            groundConfig.color ?? (isDarkMode ? '#1e293b' : '#e2e8f0'),
          ]}
          position={[0, groundY, 0]}
        />
      )}
      {/* Ground plane for non-grid environments */}
      {groundConfig.visible && groundConfig.type === 'plane' && (
        <mesh
          rotation={[-Math.PI / 2, 0, 0]}
          position={[0, groundY, 0]}
          receiveShadow
        >
          <planeGeometry args={[groundConfig.size ?? 100, groundConfig.size ?? 100]} />
          <meshStandardMaterial color={groundConfig.color ?? '#228B22'} />
        </mesh>
      )}

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

      {/* Render all members with accessories and overlays */}
      {membersWithPositions.map(({ member, position, isSeated = false, seatRotation = 0 }) => (
        <MemberWithAccessories
          key={member.id}
          member={member}
          position={position}
          cursorPosition={cursorPosition}
          selectedNodes={selectedNodeIds}
          isAnimating={false}
          isSeated={isSeated}
          seatRotation={seatRotation}
          onMemberClick={() => onMemberClick?.(member.id, position)}
          showOverlays
          showAccessories
        />
      ))}
    </>
  )
}
