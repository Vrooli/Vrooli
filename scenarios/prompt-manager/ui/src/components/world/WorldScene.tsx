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
import { useMemberComponent } from './MemberProvider'
import { WorldErrorBoundary } from './WorldErrorBoundary'
import { DragPlane } from './interaction'
import { FurnitureManager } from './furniture'
import { DecorationManager } from './decorations'
import { PerformanceMonitor, FPSOverlay } from './performance'
import { useInteractionStore } from '@/stores/interactionStore'
import { useFurnitureStore } from '@/stores/furnitureStore'
import type { Member } from '@/types/member'

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
}

interface WorldSceneProps {
  cameraState: CameraState
  selectedNodeIds: string[]
  cursorPosition: { x: number; y: number } | null
  /** All members with their positions */
  membersWithPositions: MemberWithPosition[]
  /** Called when a member is clicked, with member ID and position */
  onMemberClick?: (memberId: string, position: [number, number, number]) => void
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
  isDarkMode = true,
  showFpsOverlay = false,
  autoAdjustPerformance = true,
}: WorldSceneProps) {
  const controlsRef = useRef<OrbitControlsRef>(null)
  const { camera } = useThree()

  // Get member component from DI
  const MemberComponent = useMemberComponent()

  // Disable orbit controls during drag
  const isDragging = useInteractionStore((state) => state.isDragging)

  // Furniture seat position lookup
  const getMemberSeatPosition = useFurnitureStore((state) => state.getMemberSeatPosition)

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

      {/* Lighting */}
      <ambientLight intensity={0.4} />
      <directionalLight
        position={[10, 10, 5]}
        intensity={1}
        castShadow
        shadow-mapSize={[2048, 2048]}
      />
      <pointLight position={[-10, 5, -10]} intensity={0.5} color="#6366f1" />
      <pointLight position={[10, -5, 10]} intensity={0.3} color="#22d3ee" />

      {/* Environment */}
      {isDarkMode && (
        <WorldErrorBoundary componentName="Stars" minimal>
          <Stars radius={100} depth={50} count={2000} factor={4} fade speed={1} />
        </WorldErrorBoundary>
      )}
      <fog attach="fog" args={[isDarkMode ? '#0f172a' : '#f8fafc', 10, 50]} />

      {/* Controls - disabled during drag */}
      <OrbitControls
        ref={controlsRef as React.Ref<never>}
        enabled={!isDragging}
        enableDamping
        dampingFactor={0.05}
        minDistance={3}
        maxDistance={30}
        maxPolarAngle={Math.PI * 0.85}
        minPolarAngle={Math.PI * 0.15}
      />

      {/* Drag plane - catches pointer events during drag */}
      <DragPlane y={0} />

      {/* Grid helper */}
      <gridHelper
        args={[30, 30, isDarkMode ? '#1e293b' : '#e2e8f0', isDarkMode ? '#1e293b' : '#e2e8f0']}
        position={[0, -2, 0]}
      />

      {/* Furniture and Decorations */}
      <FurnitureManager interactive draggable />
      <DecorationManager interactive draggable />

      {/* Render all members */}
      {membersWithPositions.map(({ member, position }) => {
        const seatInfo = getMemberSeatPosition(member.id)
        const finalPosition = seatInfo?.position ?? position
        const isSeated = !!seatInfo
        const seatRotation = seatInfo?.rotation ?? 0

        return (
          <MemberComponent
            key={member.id}
            memberId={member.id}
            position={finalPosition}
            cursorPosition={cursorPosition}
            selectedNodes={selectedNodeIds}
            isAnimating={false}
            isSeated={isSeated}
            seatRotation={seatRotation}
            onMemberClick={() => onMemberClick?.(member.id, finalPosition)}
            colors={{
              body: member.bodyColor,
              head: member.headColor,
              accent: member.accentColor,
            }}
          />
        )
      })}
    </>
  )
}
