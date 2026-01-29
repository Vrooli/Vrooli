/**
 * Moon component with glow effect.
 * Positioned opposite to the sun, visible at night.
 */

import { useMemo, useRef } from 'react'
import { useFrame } from '@react-three/fiber'
import * as THREE from 'three'
import { useEnvironmentStore } from '@/stores/environmentStore'
import { calculateMoonPosition, isNightTime } from '@/lib/sky/sunPosition'

interface MoonProps {
  /** Override the time value (0-24 hours) */
  timeValue?: number
  /** Size of the moon */
  size?: number
}

/**
 * Moon with subtle glow effect.
 * Only rendered when the sun is below the horizon.
 */
export function Moon({ timeValue: timeValueProp, size = 1.5 }: MoonProps) {
  const meshRef = useRef<THREE.Mesh>(null)
  const glowRef = useRef<THREE.Mesh>(null)

  const storeTimeValue = useEnvironmentStore((state) => state.timeValue)
  const timeValue = timeValueProp ?? storeTimeValue

  // Calculate moon position
  const position = useMemo(() => calculateMoonPosition(timeValue), [timeValue])

  // Check if moon should be visible
  const isVisible = useMemo(() => {
    // Moon is visible when it's above the horizon (opposite side of sun)
    return position[1] > 0
  }, [position])

  // Moon is brighter at night
  const opacity = useMemo(() => {
    if (!isNightTime(timeValue)) {
      // Daytime - very faint if visible at all
      return 0.2
    }
    // Nighttime - full brightness
    return 1
  }, [timeValue])

  // Subtle rotation for visual interest
  useFrame((_, delta) => {
    if (meshRef.current) {
      meshRef.current.rotation.y += delta * 0.02
    }
  })

  if (!isVisible) {
    return null
  }

  return (
    <group position={position}>
      {/* Main moon sphere */}
      <mesh ref={meshRef}>
        <sphereGeometry args={[size, 32, 32]} />
        <meshBasicMaterial
          color="#E8E8E8"
          toneMapped={false}
          transparent
          opacity={opacity}
        />
      </mesh>

      {/* Glow effect */}
      <mesh ref={glowRef} scale={1.3}>
        <sphereGeometry args={[size, 32, 32]} />
        <meshBasicMaterial
          color="#B8C4D8"
          toneMapped={false}
          transparent
          opacity={opacity * 0.3}
          side={THREE.BackSide}
        />
      </mesh>
    </group>
  )
}
