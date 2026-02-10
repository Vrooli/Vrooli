/**
 * SlimeAgent - A cute blob creature built with Three.js MeshPhysicalMaterial.
 * Features cursor tracking, idle wobble/breathing, hop locomotion with
 * squash-and-stretch, reaction animations, and per-agent cosmetic variation.
 *
 * LOD System Integration:
 * - high: Full wobble, cursor tracking (pupil + body lean), all reactions
 * - medium: Half wobble, pupil-only cursor tracking, no body lean
 * - low: No wobble, minimal breathing, no cursor tracking
 * - culled: Skip all, set position only
 */

import { useRef, useMemo, useCallback, useEffect } from 'react'
import { useFrame, useThree } from '@react-three/fiber'
import { MeshWobbleMaterial } from '@react-three/drei'
import type { Group, Mesh } from 'three'
import * as THREE from 'three'
import type { AgentProps } from '@/types/world'
import { cursorRef } from '../cursorRef'
import { useHoverHighlight } from '@/hooks/useHoverHighlight'
import { useLODStore } from '@/stores/lodStore'
import { useGraphicsStore } from '@/stores/graphicsStore'
import type { LODLevel } from '@/types/lod'
import { bindSlimeShader, syncSlimeShader } from '@/lib/shaders/slimeShader'

// Body sphere radius - used for ground offset so bottom of sphere sits on ground
const BODY_RADIUS = 0.4

// Reusable unit scale vector — avoids allocating a new Vector3 every frame
const UNIT_SCALE = new THREE.Vector3(1, 1, 1)

// Default agent colors
const DEFAULT_COLORS = {
  body: '#6366f1', // Indigo
  head: '#818cf8', // Light indigo (belly highlight)
  eye: '#ffffff',
  pupil: '#1e1b4b',
  accent: '#a5b4fc',
  glow: '#c7d2fe',
}

// Deterministic hash from string (for per-agent variation)
function hashString(str: string): number {
  let hash = 0
  for (let i = 0; i < str.length; i++) {
    const char = str.charCodeAt(i)
    hash = ((hash << 5) - hash) + char
    hash |= 0
  }
  return Math.abs(hash)
}

// Mouth style variants
type MouthStyle = 'smile' | 'cat' | 'chevron' | 'none'
const MOUTH_STYLES: MouthStyle[] = ['smile', 'cat', 'chevron', 'none']

interface AgentVariation {
  hasEarNubs: boolean
  hasBlush: boolean
  mouthStyle: MouthStyle
  wobbleSpeed: number
  bodyAspectY: number
}

function getAgentVariation(agentId: string): AgentVariation {
  const h = hashString(agentId)
  return {
    hasEarNubs: (h % 2) === 0,
    hasBlush: ((h >> 1) % 2) === 0,
    mouthStyle: MOUTH_STYLES[(h >> 2) % 4] as MouthStyle,
    wobbleSpeed: 1.3 + ((h >> 4) % 5) * 0.1, // 1.3-1.7
    bodyAspectY: 0.82 + ((h >> 7) % 7) * 0.01, // 0.82-0.88
  }
}

export function SlimeAgent({
  position,
  selectedNodes: selectedNodesProp,
  isAnimating: _isAnimating,
  onAgentClick,
  colors,
  agentId,
  isSeated = false,
  seatRotation = 0,
}: AgentProps) {
  void _isAnimating

  const { camera } = useThree()

  const selectedNodes = selectedNodesProp ?? []

  // LOD state refs
  const lodLevelRef = useRef<LODLevel>('high')
  const lodFrameCountRef = useRef<number>(0)
  const objectIdRef = useRef(agentId ?? `agent-${Math.random().toString(36).slice(2)}`)

  // Hover highlighting
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

  // Per-agent variation (deterministic)
  const variation = useMemo(
    () => getAgentVariation(objectIdRef.current),
    []
  )

  const groupRef = useRef<Group>(null)
  const bodyRef = useRef<Mesh>(null)
  const leftPupilRef = useRef<Mesh>(null)
  const rightPupilRef = useRef<Mesh>(null)

  // Reusable vectors for world position and movement detection
  const worldPosVec = useRef(new THREE.Vector3())
  const prevWorldPos = useRef(new THREE.Vector3())
  const isMovingRef = useRef(false)
  const prevYVelocity = useRef(0)

  // Animation state
  const animationState = useRef({
    time: 0,
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

  // Create body material with slime shader
  const bodyMaterial = useMemo(() => {
    const mat = new THREE.MeshPhysicalMaterial({
      color: COLORS.body,
      clearcoat: 1.0,
      clearcoatRoughness: 0.05,
      transmission: 0.15,
      thickness: 1.5,
      ior: 1.4,
      iridescence: 0.3,
      iridescenceIOR: 1.2,
      roughness: 0.2,
      metalness: 0.0,
    })
    bindSlimeShader(mat, {
      wobbleIntensity: 0.02,
      wobbleSpeed: variation.wobbleSpeed,
    })
    return mat
  }, [COLORS.body, variation.wobbleSpeed])

  // Eye/pupil materials
  const eyeMaterial = useMemo(() => new THREE.MeshBasicMaterial({ color: COLORS.eye }), [COLORS.eye])
  const pupilMaterial = useMemo(() => new THREE.MeshBasicMaterial({ color: COLORS.pupil }), [COLORS.pupil])
  const accentMaterial = useMemo(
    () => new THREE.MeshStandardMaterial({
      color: COLORS.accent,
      metalness: 0.4,
      roughness: 0.4,
      emissive: COLORS.glow,
      emissiveIntensity: 0.2,
    }),
    [COLORS.accent, COLORS.glow]
  )
  const blushMaterial = useMemo(
    () => new THREE.MeshBasicMaterial({
      color: '#ff8fa3',
      transparent: true,
      opacity: 0.3,
    }),
    []
  )

  const handleClick = useCallback(
    (event: { stopPropagation: () => void }) => {
      event.stopPropagation()
      onAgentClick?.()
    },
    [onAgentClick]
  )

  // Trigger wave/celebration when selection changes
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

  // Animation loop with LOD-based optimization
  useFrame((_, delta) => {
    if (!groupRef.current) return

    const state = animationState.current
    state.time += delta

    // ===== LOD CALCULATION & MOVEMENT DETECTION (every 5 frames) =====
    lodFrameCountRef.current++
    if (lodFrameCountRef.current % 5 === 0) {
      groupRef.current.getWorldPosition(worldPosVec.current)
      const dx = worldPosVec.current.x - camera.position.x
      const dy = worldPosVec.current.y - camera.position.y
      const dz = worldPosVec.current.z - camera.position.z
      const distance = Math.sqrt(dx * dx + dy * dy + dz * dz)

      const moveDx = worldPosVec.current.x - prevWorldPos.current.x
      const moveDz = worldPosVec.current.z - prevWorldPos.current.z
      const moveDist = Math.sqrt(moveDx * moveDx + moveDz * moveDz)
      isMovingRef.current = moveDist > 0.01
      prevWorldPos.current.copy(worldPosVec.current)

      const lodLevel = useLODStore.getState().calculateLODLevel(distance)
      lodLevelRef.current = lodLevel
      useLODStore.getState().updateObjectLOD(objectIdRef.current, distance)
    }

    const lodLevel = lodLevelRef.current

    // ===== CULLED: Skip all rendering logic =====
    if (lodLevel === 'culled') {
      groupRef.current.position.set(position[0], position[1], position[2])
      return
    }

    // ===== SHADER SYNC =====
    const wobbleEnabled = useGraphicsStore.getState().config.agentWobble
    const wobbleIntensity = !wobbleEnabled ? 0
      : lodLevel === 'high' ? 0.02
        : lodLevel === 'medium' ? 0.01
          : 0
    syncSlimeShader(bodyMaterial, state.time * variation.wobbleSpeed, 1.0, wobbleIntensity)

    // ===== HOP LOCOMOTION (when moving) =====
    if (isMovingRef.current) {
      const hopPhase = state.time * 6
      const hopY = Math.abs(Math.sin(hopPhase)) * 0.15
      groupRef.current.position.y = position[1] + hopY
      groupRef.current.position.x = position[0]

      // Squash-and-stretch
      if (lodLevel === 'high' || lodLevel === 'medium') {
        const sinVal = Math.sin(hopPhase)
        const yVel = Math.cos(hopPhase)
        prevYVelocity.current = yVel

        if (yVel > 0) {
          // Going up - stretch
          const stretchY = 1.0 + sinVal * 0.15
          const stretchXZ = 1.0 - sinVal * 0.08
          groupRef.current.scale.set(stretchXZ, stretchY, stretchXZ)
          syncSlimeShader(bodyMaterial, state.time * variation.wobbleSpeed, stretchY, wobbleIntensity)
        } else {
          // Landing - squash
          const squashY = 1.0 - Math.abs(sinVal) * 0.12
          const squashXZ = 1.0 + Math.abs(sinVal) * 0.06
          groupRef.current.scale.set(squashXZ, squashY, squashXZ)
          syncSlimeShader(bodyMaterial, state.time * variation.wobbleSpeed, squashY, wobbleIntensity)
        }
      }

      return
    }

    // Reset scale when stopping
    if (groupRef.current.scale.x !== 1 || groupRef.current.scale.y !== 1) {
      groupRef.current.scale.lerp(UNIT_SCALE, 0.15)
      if (Math.abs(groupRef.current.scale.x - 1) < 0.001) {
        groupRef.current.scale.set(1, 1, 1)
      }
    }

    // ===== IDLE BREATHING (all LOD levels except culled) =====
    const seatedOffset = isSeated ? -0.2 : 0
    const breathMult = isSeated ? 0.1 : 1

    if (lodLevel === 'low') {
      // No sway or scale breathing at low LOD
      groupRef.current.position.y = position[1] + seatedOffset
      groupRef.current.position.x = position[0]
    } else {
      // Lateral sway only (no Y bobbing - agent sits on ground)
      const swayX = Math.sin(state.time * 0.8) * 0.01 * breathMult
      groupRef.current.position.y = position[1] + seatedOffset
      groupRef.current.position.x = position[0] + swayX

      // Breathing via scale only (bottom stays grounded because inner offset = body radius)
      const breathScale = 1.0 + Math.sin(state.time * 2) * 0.03 * breathMult
      groupRef.current.scale.set(1, breathScale, 1)
      syncSlimeShader(bodyMaterial, state.time * variation.wobbleSpeed, breathScale, wobbleIntensity)
    }

    if (isSeated) {
      groupRef.current.rotation.y = seatRotation
    }

    // ===== CURSOR TRACKING (high and medium LOD only) =====
    const cursorPosition = cursorRef.current
    if ((lodLevel === 'high' || lodLevel === 'medium') && cursorPosition) {
      const trackingStrength = lodLevel === 'high' ? 0.03 : 0.02
      const dx = cursorPosition.x * trackingStrength
      const dy = cursorPosition.y * trackingStrength * 0.67

      // Pupil offset
      const clampedDx = Math.max(-0.03, Math.min(0.03, dx))
      const clampedDy = Math.max(-0.02, Math.min(0.02, dy))

      if (leftPupilRef.current) {
        leftPupilRef.current.position.x = clampedDx
        leftPupilRef.current.position.y = clampedDy
        leftPupilRef.current.position.z = 0.06
      }
      if (rightPupilRef.current) {
        rightPupilRef.current.position.x = clampedDx
        rightPupilRef.current.position.y = clampedDy
        rightPupilRef.current.position.z = 0.06
      }

      // Body lean (high LOD only)
      if (lodLevel === 'high') {
        const leanTarget = cursorPosition.x * 0.1
        groupRef.current.rotation.y += (leanTarget - groupRef.current.rotation.y) * 0.05
      }
    }

    // ===== REACTION ANIMATIONS (high LOD only, skip when seated) =====
    if (lodLevel === 'high' && !isSeated) {
      // Wave animation - body tilts side to side
      if (state.isWaving) {
        state.waveProgress += delta * 2
        const tilt = Math.sin(state.waveProgress * 8) * 0.15
        groupRef.current.rotation.z = tilt

        if (state.waveProgress > 1.5) {
          state.isWaving = false
          state.waveProgress = 0
          groupRef.current.rotation.z = 0
        }
      }

      // Celebration animation - spin + bounce + increased wobble
      if (state.isCelebrating) {
        state.celebrationProgress += delta
        const spin = state.celebrationProgress * Math.PI * 4
        groupRef.current.rotation.y = spin

        const bounce = Math.sin(state.celebrationProgress * 15) * 0.1
        groupRef.current.scale.setScalar(1 + bounce)

        // Increased wobble during celebration
        syncSlimeShader(bodyMaterial, state.time * variation.wobbleSpeed, 1.0, 0.05)

        if (state.celebrationProgress > 1.5) {
          state.isCelebrating = false
          state.celebrationProgress = 0
          groupRef.current.rotation.y = 0
          groupRef.current.scale.setScalar(1)
        }
      }
    }
  })

  return (
    <group ref={groupRef} position={position} onClick={handleClick} {...hoverProps}>
      {/* Inner offset group - lifts body so sphere bottom sits at ground level */}
      <group position={[0, BODY_RADIUS, 0]}>
        {/* Body - egg-shaped sphere with slime shader */}
        <mesh
          ref={bodyRef}
          scale={[1, variation.bodyAspectY / 0.85, 1]}
          material={bodyMaterial}
          castShadow
        >
          <sphereGeometry args={[0.4, 32, 32]} />
        </mesh>

        {/* Eyes */}
        {/* Left eye */}
        <mesh position={[-0.12, 0.1, 0.3]} material={eyeMaterial}>
          <sphereGeometry args={[0.1, 16, 16]} />
          {/* Left pupil - z=0.06 protrudes past eye surface (r=0.1) */}
          <mesh ref={leftPupilRef} position={[0, 0, 0.06]} material={pupilMaterial}>
            <sphereGeometry args={[0.055, 12, 12]} />
          </mesh>
        </mesh>

        {/* Right eye */}
        <mesh position={[0.12, 0.1, 0.3]} material={eyeMaterial}>
          <sphereGeometry args={[0.1, 16, 16]} />
          {/* Right pupil - z=0.06 protrudes past eye surface (r=0.1) */}
          <mesh ref={rightPupilRef} position={[0, 0, 0.06]} material={pupilMaterial}>
            <sphereGeometry args={[0.055, 12, 12]} />
          </mesh>
        </mesh>

        {/* Mouth */}
        <SlimeMouth style={variation.mouthStyle} material={accentMaterial} />

        {/* Ear nubs (50% of agents) */}
        {variation.hasEarNubs && (
          <>
            <mesh position={[-0.2, 0.3, 0]} material={accentMaterial}>
              <sphereGeometry args={[0.08, 12, 12]} />
            </mesh>
            <mesh position={[0.2, 0.3, 0]} material={accentMaterial}>
              <sphereGeometry args={[0.08, 12, 12]} />
            </mesh>
          </>
        )}

        {/* Blush marks (50% of agents) */}
        {variation.hasBlush && (
          <>
            <mesh position={[-0.22, 0.0, 0.25]} material={blushMaterial}>
              <circleGeometry args={[0.06, 16]} />
            </mesh>
            <mesh position={[0.22, 0.0, 0.25]} material={blushMaterial}>
              <circleGeometry args={[0.06, 16]} />
            </mesh>
          </>
        )}

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
    </group>
  )
}

/**
 * Slime mouth component - renders different mouth styles.
 */
function SlimeMouth({ style, material }: { style: MouthStyle; material: THREE.Material }) {
  switch (style) {
    case 'smile':
      // Half-torus arc (0→π) sweeps right→top→left (∩ frown). Rotate π around Z to flip to ∪ smile.
      return (
        <mesh position={[0, -0.05, 0.42]} rotation={[0, 0, Math.PI]} material={material}>
          <torusGeometry args={[0.04, 0.01, 8, 16, Math.PI]} />
        </mesh>
      )
    case 'cat':
      // :3 cat mouth - two small arcs, each flipped to ∪ (ω shape)
      return (
        <group position={[0, -0.05, 0.42]}>
          <mesh position={[-0.025, 0, 0]} rotation={[0, 0, Math.PI - 0.2]} material={material}>
            <torusGeometry args={[0.025, 0.008, 8, 12, Math.PI]} />
          </mesh>
          <mesh position={[0.025, 0, 0]} rotation={[0, 0, Math.PI + 0.2]} material={material}>
            <torusGeometry args={[0.025, 0.008, 8, 12, Math.PI]} />
          </mesh>
        </group>
      )
    case 'chevron':
      // ^ mouth
      return (
        <group position={[0, -0.06, 0.42]}>
          <mesh position={[-0.015, 0.01, 0]} rotation={[0, 0, 0.4]} material={material}>
            <boxGeometry args={[0.04, 0.008, 0.008]} />
          </mesh>
          <mesh position={[0.015, 0.01, 0]} rotation={[0, 0, -0.4]} material={material}>
            <boxGeometry args={[0.04, 0.008, 0.008]} />
          </mesh>
        </group>
      )
    case 'none':
    default:
      return null
  }
}

/**
 * Floating orb particle that orbits the slime agent.
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
