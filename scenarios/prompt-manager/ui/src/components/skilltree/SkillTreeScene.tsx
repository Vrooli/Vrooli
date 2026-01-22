/**
 * SkillTreeScene - Composes the 3D scene with lights, camera, and controls.
 */

import { useRef, useEffect } from 'react'
import { OrbitControls, Stars } from '@react-three/drei'
import { useThree } from '@react-three/fiber'
import type { SkillTreeData, CameraState } from '@/types/skilltree'
import { SkillTreeNodes } from './SkillTreeNodes'
import { SkillTreeConnections } from './SkillTreeConnections'
import { useAvatarComponent } from './AvatarProvider'

interface SkillTreeSceneProps {
  treeData: SkillTreeData
  cameraState: CameraState
  selectedNodeIds: string[]
  hoveredNodeId: string | null
  cursorPosition: { x: number; y: number } | null
  onNodeClick: (nodeId: string, event: MouseEvent) => void
  onNodeHover: (nodeId: string | null) => void
  onAvatarClick?: () => void
  avatarColors?: {
    body: string
    head: string
    accent: string
  }
  isDarkMode?: boolean
}

export function SkillTreeScene({
  treeData,
  cameraState,
  selectedNodeIds,
  hoveredNodeId,
  cursorPosition,
  onNodeClick,
  onNodeHover,
  onAvatarClick,
  avatarColors,
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

      {/* Connections */}
      <SkillTreeConnections
        connections={treeData.connections}
        selectedNodeIds={selectedNodeIds}
      />

      {/* Nodes */}
      <SkillTreeNodes
        nodes={treeData.nodes}
        hoveredNodeId={hoveredNodeId}
        onNodeClick={onNodeClick}
        onNodeHover={onNodeHover}
      />

      {/* Avatar */}
      <AvatarComponent
        position={[0, 0, 0]}
        cursorPosition={cursorPosition}
        selectedNodes={selectedNodeIds}
        isAnimating={false}
        onAvatarClick={onAvatarClick}
        colors={avatarColors}
      />
    </>
  )
}
