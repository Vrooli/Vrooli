/**
 * WorldScene - Composes the 3D scene with members, lights, and controls.
 *
 * Note: 3D skill nodes have been removed in favor of a 2D overlay (SkillSelectionOverlay).
 * This scene now focuses on member display with ambient environment.
 */

import { useRef, useEffect } from 'react'
import { OrbitControls, Stars } from '@react-three/drei'
import { useThree } from '@react-three/fiber'
import { useMemberComponent } from './MemberProvider'
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
}

export function WorldScene({
  cameraState,
  selectedNodeIds,
  cursorPosition,
  membersWithPositions,
  onMemberClick,
  isDarkMode = true,
}: WorldSceneProps) {
  const controlsRef = useRef<OrbitControlsRef>(null)
  const { camera } = useThree()

  // Get member component from DI
  const MemberComponent = useMemberComponent()

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
      {isDarkMode && <Stars radius={100} depth={50} count={2000} factor={4} fade speed={1} />}
      <fog attach="fog" args={[isDarkMode ? '#0f172a' : '#f8fafc', 10, 50]} />

      {/* Controls */}
      <OrbitControls
        ref={controlsRef as React.Ref<never>}
        enableDamping
        dampingFactor={0.05}
        minDistance={3}
        maxDistance={30}
        maxPolarAngle={Math.PI * 0.85}
        minPolarAngle={Math.PI * 0.15}
      />

      {/* Grid helper */}
      <gridHelper
        args={[30, 30, isDarkMode ? '#1e293b' : '#e2e8f0', isDarkMode ? '#1e293b' : '#e2e8f0']}
        position={[0, -2, 0]}
      />

      {/* Render all members */}
      {membersWithPositions.map(({ member, position }) => (
        <MemberComponent
          key={member.id}
          memberId={member.id}
          position={position}
          cursorPosition={cursorPosition}
          selectedNodes={selectedNodeIds}
          isAnimating={false}
          onMemberClick={() => onMemberClick?.(member.id, position)}
          colors={{
            body: member.bodyColor,
            head: member.headColor,
            accent: member.accentColor,
          }}
        />
      ))}
    </>
  )
}
