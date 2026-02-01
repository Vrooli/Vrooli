/**
 * GeometricAgent - A 3D geometric agent built with Three.js primitives.
 * Features cursor tracking, idle animations, reaction animations, and hover effects.
 *
 * LOD System Integration:
 * - Cursor tracking disabled for distant agents (LOD low/culled)
 * - Animations simplified for medium LOD
 * - Hover disabled for low/culled LOD
 */
// DOC: docs/concepts/3D-WORLD-ARCHITECTURE.md#geometricagent-anatomy

import { useRef, useMemo, useCallback, useEffect } from 'react'
import { useFrame, useThree } from '@react-three/fiber'
import { MeshWobbleMaterial } from '@react-three/drei'
import type { Group, Mesh } from 'three'
import * as THREE from 'three'
import type { AgentProps } from '@/types/world'
import { useHoverHighlight } from '@/hooks/useHoverHighlight'
import { useLODStore } from '@/stores/lodStore'
import type { LODLevel } from '@/types/lod'

// Default agent colors
const DEFAULT_COLORS = {
  body: '#6366f1', // Indigo
  head: '#818cf8', // Light indigo
  eye: '#ffffff',
  pupil: '#1e1b4b',
  accent: '#a5b4fc',
  glow: '#c7d2fe',
}

export function GeometricAgent({
  position,
  cursorPosition,
  selectedNodes: selectedNodesProp,
  isAnimating: _isAnimating,
  onAgentClick,
  colors,
  agentId,
  isSeated = false,
  seatRotation = 0,
}: AgentProps) {
  void _isAnimating // Reserved for future animation triggers

  const { camera } = useThree()

  // Defensive: ensure selectedNodes is always an array
  const selectedNodes = selectedNodesProp ?? []

  // LOD state refs (updated in useFrame, no React updates)
  const lodLevelRef = useRef<LODLevel>('high')
  const lodDistanceRef = useRef<number>(0)
  const lodFrameCountRef = useRef<number>(0)
  const objectIdRef = useRef(agentId ?? `agent-${Math.random().toString(36).slice(2)}`)

  // Hover highlighting - only if agentId is provided and LOD allows
  // Note: isHovered is used by AgentWithAccessories which wraps this component
  const { hoverProps } = useHoverHighlight(agentId ?? 'unknown', {
    enabled: !!agentId && (lodLevelRef.current === 'high' || lodLevelRef.current === 'medium'),
  })

  // Merge custom colors with defaults
  const COLORS = useMemo(
    () => ({
      body: colors?.body ?? DEFAULT_COLORS.body,
      head: colors?.head ?? DEFAULT_COLORS.head,
      eye: DEFAULT_COLORS.eye,
      pupil: DEFAULT_COLORS.pupil,
      accent: colors?.accent ?? DEFAULT_COLORS.accent,
      glow: DEFAULT_COLORS.glow,
    }),
    [colors]
  )
  const groupRef = useRef<Group>(null)
  const headRef = useRef<Mesh>(null)
  const bodyRef = useRef<Mesh>(null)
  const leftArmRef = useRef<Group>(null)
  const rightArmRef = useRef<Group>(null)

  // Animation state
  const animationState = useRef({
    time: 0,
    lookTarget: new THREE.Vector3(0, 0, 1),
    waveProgress: 0,
    isWaving: false,
    celebrationProgress: 0,
    isCelebrating: false,
  })

  // Cleanup LOD tracking on unmount
  useEffect(() => {
    const objectId = objectIdRef.current
    return () => {
      useLODStore.getState().removeObject(objectId)
    }
  }, [])

  // Trigger wave when selection changes
  const prevSelectionCount = useRef(selectedNodes.length)
  if (selectedNodes.length > prevSelectionCount.current) {
    if (selectedNodes.length >= 3) {
      animationState.current.isCelebrating = true
      animationState.current.celebrationProgress = 0
    } else {
      animationState.current.isWaving = true
      animationState.current.waveProgress = 0
    }
  }
  prevSelectionCount.current = selectedNodes.length

  // Create materials - depend on COLORS to update when colors change
  const materials = useMemo(
    () => ({
      body: new THREE.MeshStandardMaterial({
        color: COLORS.body,
        metalness: 0.3,
        roughness: 0.6,
      }),
      head: new THREE.MeshStandardMaterial({
        color: COLORS.head,
        metalness: 0.2,
        roughness: 0.5,
      }),
      eye: new THREE.MeshBasicMaterial({ color: COLORS.eye }),
      pupil: new THREE.MeshBasicMaterial({ color: COLORS.pupil }),
      accent: new THREE.MeshStandardMaterial({
        color: COLORS.accent,
        metalness: 0.4,
        roughness: 0.4,
        emissive: COLORS.glow,
        emissiveIntensity: 0.2,
      }),
    }),
    [COLORS]
  )

  // Handle click on agent
  const handleClick = useCallback(
    (event: { stopPropagation: () => void }) => {
      event.stopPropagation()
      onAgentClick?.()
    },
    [onAgentClick]
  )

  // Animation loop with LOD-based optimization
  useFrame((_, delta) => {
    if (!groupRef.current || !headRef.current) return

    const state = animationState.current
    state.time += delta

    // ===== LOD CALCULATION (every 5 frames) =====
    lodFrameCountRef.current++
    if (lodFrameCountRef.current % 5 === 0) {
      // Calculate distance from camera
      const dx = position[0] - camera.position.x
      const dy = position[1] - camera.position.y
      const dz = position[2] - camera.position.z
      const distance = Math.sqrt(dx * dx + dy * dy + dz * dz)
      lodDistanceRef.current = distance

      // Calculate LOD level (uses store's thresholds)
      const lodLevel = useLODStore.getState().calculateLODLevel(distance)
      lodLevelRef.current = lodLevel

      // Update store for aggregate statistics
      useLODStore.getState().updateObjectLOD(objectIdRef.current, distance)
    }

    const lodLevel = lodLevelRef.current

    // ===== CULLED: Skip all rendering logic =====
    if (lodLevel === 'culled') {
      // Just update position for potential re-entry
      groupRef.current.position.set(position[0], position[1], position[2])
      return
    }

    // ===== IDLE FLOATING ANIMATION (all LOD levels except culled) =====
    // When seated, reduce floating significantly and lower position
    const seatedOffset = isSeated ? -0.2 : 0
    const floatMultiplier = isSeated ? 0.1 : 1

    // Simplified for low LOD
    if (lodLevel === 'low') {
      // Minimal floating for low LOD
      const floatY = Math.sin(state.time * 0.5) * 0.02 * floatMultiplier
      groupRef.current.position.y = position[1] + floatY + seatedOffset
      groupRef.current.position.x = position[0]
    } else {
      // Full floating animation for medium/high
      const floatY = Math.sin(state.time * 1.5) * 0.05 * floatMultiplier
      const floatX = Math.sin(state.time * 0.8) * 0.02 * floatMultiplier
      groupRef.current.position.y = position[1] + floatY + seatedOffset
      groupRef.current.position.x = position[0] + floatX
    }

    // Apply seat rotation when seated
    if (isSeated) {
      groupRef.current.rotation.y = seatRotation
    }

    // ===== CURSOR TRACKING (high and medium LOD only) =====
    if (lodLevel === 'high' || lodLevel === 'medium') {
      if (cursorPosition) {
        // Full tracking for high LOD
        const trackingStrength = lodLevel === 'high' ? 0.5 : 0.25
        state.lookTarget.set(
          cursorPosition.x * trackingStrength,
          cursorPosition.y * (trackingStrength * 0.6),
          5
        )
      } else {
        state.lookTarget.lerp(new THREE.Vector3(0, 0, 5), 0.02)
      }

      // Head rotation toward target
      const headLookDirection = state.lookTarget.clone().normalize()
      const targetRotationY = Math.atan2(headLookDirection.x, headLookDirection.z)
      const targetRotationX = -Math.atan2(
        headLookDirection.y,
        Math.sqrt(headLookDirection.x ** 2 + headLookDirection.z ** 2)
      )

      const rotationSpeed = lodLevel === 'high' ? 0.1 : 0.05
      headRef.current.rotation.y += (targetRotationY - headRef.current.rotation.y) * rotationSpeed
      headRef.current.rotation.x +=
        (Math.max(-0.3, Math.min(0.3, targetRotationX)) - headRef.current.rotation.x) * rotationSpeed
    }

    // ===== REACTION ANIMATIONS (high LOD only, skip when seated) =====
    if (lodLevel === 'high' && !isSeated) {
      // Wave animation
      if (state.isWaving && rightArmRef.current) {
        state.waveProgress += delta * 2
        const wave = Math.sin(state.waveProgress * 8) * 0.5
        rightArmRef.current.rotation.z = -Math.PI / 4 + wave
        rightArmRef.current.rotation.x = -Math.PI / 3

        if (state.waveProgress > 1.5) {
          state.isWaving = false
          state.waveProgress = 0
          rightArmRef.current.rotation.z = 0
          rightArmRef.current.rotation.x = 0
        }
      }

      // Celebration animation
      if (state.isCelebrating) {
        state.celebrationProgress += delta
        const spin = state.celebrationProgress * Math.PI * 4
        groupRef.current.rotation.y = spin

        const bounce = Math.sin(state.celebrationProgress * 15) * 0.1
        groupRef.current.scale.setScalar(1 + bounce)

        if (state.celebrationProgress > 1.5) {
          state.isCelebrating = false
          state.celebrationProgress = 0
          groupRef.current.rotation.y = 0
          groupRef.current.scale.setScalar(1)
        }
      }
    }

    // ===== BODY SWAY (high and medium LOD) =====
    if ((lodLevel === 'high' || lodLevel === 'medium') && bodyRef.current) {
      const swayAmount = lodLevel === 'high' ? 0.03 : 0.015
      bodyRef.current.rotation.z = Math.sin(state.time * 0.7) * swayAmount
    }

    // ===== ARM IDLE ANIMATION (high LOD only) =====
    if (lodLevel === 'high') {
      if (leftArmRef.current && !state.isWaving) {
        leftArmRef.current.rotation.z = Math.sin(state.time * 1.2) * 0.1
      }
      if (rightArmRef.current && !state.isWaving) {
        rightArmRef.current.rotation.z = -Math.sin(state.time * 1.2 + 0.5) * 0.1
      }
    }
  })

  return (
    <group ref={groupRef} position={position} onClick={handleClick} {...hoverProps}>
      {/* Body - capsule shape */}
      <mesh ref={bodyRef} position={[0, -0.3, 0]} material={materials.body}>
        <capsuleGeometry args={[0.25, 0.5, 8, 16]} />
      </mesh>

      {/* Head */}
      <mesh ref={headRef} position={[0, 0.4, 0]} material={materials.head}>
        <sphereGeometry args={[0.3, 32, 32]} />

        {/* Eyes */}
        <group position={[0, 0.05, 0.25]}>
          {/* Left eye */}
          <mesh position={[-0.1, 0, 0]} material={materials.eye}>
            <sphereGeometry args={[0.06, 16, 16]} />
            <mesh position={[0, 0, 0.04]} material={materials.pupil}>
              <sphereGeometry args={[0.03, 8, 8]} />
            </mesh>
          </mesh>

          {/* Right eye */}
          <mesh position={[0.1, 0, 0]} material={materials.eye}>
            <sphereGeometry args={[0.06, 16, 16]} />
            <mesh position={[0, 0, 0.04]} material={materials.pupil}>
              <sphereGeometry args={[0.03, 8, 8]} />
            </mesh>
          </mesh>
        </group>

        {/* Antenna/accent */}
        <mesh position={[0, 0.35, 0]} material={materials.accent}>
          <sphereGeometry args={[0.05, 16, 16]} />
        </mesh>
        <mesh position={[0, 0.28, 0]} material={materials.accent}>
          <cylinderGeometry args={[0.015, 0.015, 0.1, 8]} />
        </mesh>
      </mesh>

      {/* Left arm */}
      <group ref={leftArmRef} position={[-0.35, -0.1, 0]}>
        <mesh material={materials.body}>
          <capsuleGeometry args={[0.06, 0.25, 4, 8]} />
        </mesh>
      </group>

      {/* Right arm */}
      <group ref={rightArmRef} position={[0.35, -0.1, 0]}>
        <mesh material={materials.body}>
          <capsuleGeometry args={[0.06, 0.25, 4, 8]} />
        </mesh>
      </group>

      {/* Floating orbs around agent when selected */}
      {selectedNodes.length > 0 && (
        <group>
          {selectedNodes.slice(0, 5).map((_, i) => (
            <FloatingOrb
              key={i}
              index={i}
              total={Math.min(selectedNodes.length, 5)}
            />
          ))}
        </group>
      )}
    </group>
  )
}

/**
 * Floating orb particle that orbits the agent.
 */
function FloatingOrb({ index, total }: { index: number; total: number }) {
  const meshRef = useRef<Mesh>(null)
  const angle = (index / total) * Math.PI * 2

  useFrame(({ clock }) => {
    if (!meshRef.current) return

    const t = clock.getElapsedTime() + index * 0.5
    const radius = 0.6 + Math.sin(t * 2) * 0.1
    const height = Math.sin(t * 3 + index) * 0.2

    meshRef.current.position.x = Math.cos(t + angle) * radius
    meshRef.current.position.z = Math.sin(t + angle) * radius
    meshRef.current.position.y = height
  })

  return (
    <mesh ref={meshRef}>
      <sphereGeometry args={[0.04, 8, 8]} />
      <MeshWobbleMaterial
        color={DEFAULT_COLORS.glow}
        emissive={DEFAULT_COLORS.accent}
        emissiveIntensity={0.5}
        factor={0.5}
        speed={2}
      />
    </mesh>
  )
}
