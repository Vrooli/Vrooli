/**
 * SkillTreeScene - Composes the 3D scene with avatars, lights, and controls.
 *
 * Note: 3D skill nodes have been removed in favor of a 2D overlay (SkillSelectionOverlay).
 * This scene now focuses on avatar display with ambient environment.
 */

import { useRef, useEffect } from 'react'
import { OrbitControls, Stars } from '@react-three/drei'
import { useThree } from '@react-three/fiber'
import { useAvatarComponent } from './AvatarProvider'
import type { Avatar } from '@/types/avatar'

interface CameraState {
  position: [number, number, number]
  target: [number, number, number]
  zoom: number
}

/** Avatar with its computed position in the scene */
export interface AvatarWithPosition {
  avatar: Avatar
  position: [number, number, number]
}

interface SkillTreeSceneProps {
  cameraState: CameraState
  selectedNodeIds: string[]
  cursorPosition: { x: number; y: number } | null
  /** All avatars with their positions */
  avatarsWithPositions: AvatarWithPosition[]
  /** Called when an avatar is clicked, with avatar ID and position */
  onAvatarClick?: (avatarId: string, position: [number, number, number]) => void
  isDarkMode?: boolean
}

export function SkillTreeScene({
  cameraState,
  selectedNodeIds,
  cursorPosition,
  avatarsWithPositions,
  onAvatarClick,
  isDarkMode = true,
}: SkillTreeSceneProps) {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const controlsRef = useRef<any>(null)
  const { camera } = useThree()

  // Get avatar component from DI
  const AvatarComponent = useAvatarComponent()

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
        ref={controlsRef}
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

      {/* Render all avatars */}
      {avatarsWithPositions.map(({ avatar, position }) => (
        <AvatarComponent
          key={avatar.id}
          avatarId={avatar.id}
          position={position}
          cursorPosition={cursorPosition}
          selectedNodes={selectedNodeIds}
          isAnimating={false}
          onAvatarClick={() => onAvatarClick?.(avatar.id, position)}
          colors={{
            body: avatar.bodyColor,
            head: avatar.headColor,
            accent: avatar.accentColor,
          }}
        />
      ))}
    </>
  )
}
