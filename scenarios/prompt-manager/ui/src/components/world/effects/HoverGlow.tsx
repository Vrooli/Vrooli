/**
 * HoverGlow - Visual effect for highlighting hovered objects.
 * Renders an animated glow effect around the object.
 */

import { useRef } from 'react'
import { useFrame } from '@react-three/fiber'
import type { Group, Mesh } from 'three'
import * as THREE from 'three'
import { usePerformanceStore } from '@/stores/performanceStore'
import { requestWorldRender } from '../performance/worldRenderLoop'

interface HoverGlowProps {
  /** Whether glow is active */
  isActive: boolean
  /** Position to center the glow */
  position: [number, number, number]
  /** Size of the glow (radius) */
  size?: number
  /** Glow color */
  color?: string
  /** Intensity of the glow (0-1) */
  intensity?: number
  /** Whether to animate the glow */
  animated?: boolean
}

// Stable references for materials
const GLOW_MATERIAL_PARAMS = {
  transparent: true,
  side: THREE.DoubleSide,
  depthWrite: false,
}

/**
 * Renders an animated glow effect for highlighting objects.
 * Uses multiple layered rings that pulse for a nice effect.
 */
export function HoverGlow({
  isActive,
  position,
  size = 0.8,
  color = '#6366f1',
  intensity = 0.6,
  animated = true,
}: HoverGlowProps) {
  const groupRef = useRef<Group>(null)
  const innerRingRef = useRef<Mesh>(null)
  const outerRingRef = useRef<Mesh>(null)
  const glowSphereRef = useRef<Mesh>(null)
  const timeRef = useRef(0)
  const perfWindowMsRef = useRef(0)
  const perfWindowCallbacksRef = useRef(0)

  useFrame((_, delta) => {
    const t0 = performance.now()
    if (!isActive || !animated) return
    if (!groupRef.current) return

    timeRef.current += delta

    // Pulsing animation
    const pulse = Math.sin(timeRef.current * 3) * 0.1 + 1
    const pulseOpacity = (Math.sin(timeRef.current * 2) * 0.15 + 0.85) * intensity

    // Animate inner ring
    if (innerRingRef.current) {
      innerRingRef.current.scale.setScalar(pulse * 0.9)
      const mat = innerRingRef.current.material as THREE.MeshBasicMaterial
      mat.opacity = pulseOpacity * 0.4
    }

    // Animate outer ring (opposite phase)
    if (outerRingRef.current) {
      outerRingRef.current.scale.setScalar(2 - pulse * 0.1)
      const mat = outerRingRef.current.material as THREE.MeshBasicMaterial
      mat.opacity = pulseOpacity * 0.2
    }

    // Animate glow sphere
    if (glowSphereRef.current) {
      glowSphereRef.current.scale.setScalar(pulse)
      const mat = glowSphereRef.current.material as THREE.MeshBasicMaterial
      mat.opacity = pulseOpacity * 0.15
    }

    // Gentle rotation
    groupRef.current.rotation.y += delta * 0.5

    perfWindowMsRef.current += performance.now() - t0
    perfWindowCallbacksRef.current += 1
    requestWorldRender('hover-active', 1)
    if (perfWindowCallbacksRef.current >= 60) {
      usePerformanceStore.getState().recordFrameLoopAggregate(
        perfWindowMsRef.current,
        perfWindowCallbacksRef.current
      )
      perfWindowMsRef.current = 0
      perfWindowCallbacksRef.current = 0
    }
  })

  if (!isActive) {
    return null
  }

  return (
    <group ref={groupRef} position={position}>
      {/* Inner glow ring - horizontal */}
      <mesh ref={innerRingRef} rotation={[-Math.PI / 2, 0, 0]} position={[0, 0, 0]}>
        <ringGeometry args={[size * 0.5, size * 0.7, 32]} />
        <meshBasicMaterial
          color={color}
          {...GLOW_MATERIAL_PARAMS}
          opacity={intensity * 0.4}
        />
      </mesh>

      {/* Outer glow ring - horizontal */}
      <mesh ref={outerRingRef} rotation={[-Math.PI / 2, 0, 0]} position={[0, 0, 0]}>
        <ringGeometry args={[size * 0.8, size * 1.0, 32]} />
        <meshBasicMaterial
          color={color}
          {...GLOW_MATERIAL_PARAMS}
          opacity={intensity * 0.2}
        />
      </mesh>

      {/* Ambient glow sphere */}
      <mesh ref={glowSphereRef}>
        <sphereGeometry args={[size * 0.65, 16, 16]} />
        <meshBasicMaterial
          color={color}
          {...GLOW_MATERIAL_PARAMS}
          opacity={intensity * 0.15}
        />
      </mesh>

      {/* Vertical accent ring */}
      <mesh rotation={[0, 0, 0]} position={[0, 0, 0]}>
        <ringGeometry args={[size * 0.55, size * 0.6, 32]} />
        <meshBasicMaterial
          color={color}
          {...GLOW_MATERIAL_PARAMS}
          opacity={intensity * 0.25}
        />
      </mesh>
    </group>
  )
}
