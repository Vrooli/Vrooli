/**
 * DynamicSky - Renders a sky dome that changes based on continuous time.
 *
 * Uses a custom gradient shader that smoothly transitions colors based on time of day.
 * The gradient is rendered on BackSide, allowing stars and celestial bodies to render inside.
 */

import { useMemo, useEffect } from 'react'
import * as THREE from 'three'
import { useEnvironmentStore } from '@/stores/environmentStore'
import {
  calculateSunPosition,
  calculateSkyColors,
} from '@/lib/sky/sunPosition'
import { SKY_VERTEX_SHADER, SKY_FRAGMENT_SHADER } from '@/lib/shaders/glsl/sky.glsl'

interface DynamicSkyProps {
  /** Override the time value (0-24 hours) */
  timeValue?: number
  /** Radius of the sky dome */
  radius?: number
}

/**
 * Dynamic sky that changes based on continuous time.
 *
 * Uses a custom gradient shader for all scene types to ensure:
 * 1. Smooth color transitions based on time
 * 2. Stars and celestial bodies can render inside the dome (BackSide rendering)
 * 3. Consistent behavior across all scene types
 */
export function DynamicSky({
  timeValue: timeValueProp,
  radius = 80,
}: DynamicSkyProps) {
  const storeTimeValue = useEnvironmentStore((state) => state.timeValue)

  // Use prop time value, or convert legacy timeOfDay prop, or use store value
  const timeValue = timeValueProp ?? storeTimeValue

  // Calculate sky colors based on continuous time
  const skyColors = useMemo(() => calculateSkyColors(timeValue), [timeValue])

  // Stable Color objects — passed into the shader as uniform values and
  // mutated directly when skyColors changes.  Avoids re-creating the
  // ShaderMaterial (which triggers a ~2-5 ms GPU shader recompilation).
  const topColor = useMemo(() => new THREE.Color(), [])
  const middleColor = useMemo(() => new THREE.Color(), [])
  const bottomColor = useMemo(() => new THREE.Color(), [])

  const gradientMaterial = useMemo(() => {
    return new THREE.ShaderMaterial({
      uniforms: {
        topColor: { value: topColor },
        middleColor: { value: middleColor },
        bottomColor: { value: bottomColor },
        offset: { value: 0.5 },
        exponent: { value: 0.6 },
      },
      vertexShader: SKY_VERTEX_SHADER,
      fragmentShader: SKY_FRAGMENT_SHADER,
      side: THREE.BackSide,
      depthWrite: false,
    })
  }, [topColor, middleColor, bottomColor])

  // Update Color objects when sky colors change (no shader recompilation)
  useEffect(() => {
    topColor.set(skyColors.top)
    middleColor.set(skyColors.middle)
    bottomColor.set(skyColors.bottom)
  }, [skyColors, topColor, middleColor, bottomColor])

  // Dispose material on unmount
  useEffect(() => {
    return () => { gradientMaterial.dispose() }
  }, [gradientMaterial])

  // Perf: Removed useFrame rotation (0.01 rad/s is imperceptible on a gradient with BackSide rendering)

  // Perf: 16x16 segments (not 32x32) — distant sky dome, polygon difference invisible
  return (
    <mesh material={gradientMaterial}>
      <sphereGeometry args={[radius, 16, 16]} />
    </mesh>
  )
}

/**
 * Sun component that positions based on continuous time.
 * Visible when sun is above the horizon.
 */
export function CelestialBody({
  timeValue: timeValueProp,
}: {
  timeValue?: number
}) {
  const storeTimeValue = useEnvironmentStore((state) => state.timeValue)

  // Use prop time value or store value
  const timeValue = timeValueProp ?? storeTimeValue

  // Calculate position from continuous time - recalculates when timeValue changes
  const sunPosition = calculateSunPosition(timeValue)

  // Only show sun when it's above the horizon (y > 0)
  const isAboveHorizon = sunPosition[1] > 0

  // Color based on time - warmer at sunrise/sunset
  const color = useMemo(() => {
    const h = ((timeValue % 24) + 24) % 24

    if (h >= 6 && h < 8) {
      // Sunrise - warm yellow/orange
      return '#FFE4B5'
    } else if (h >= 8 && h < 16) {
      // Midday - bright white-yellow
      return '#FFFAF0'
    } else if (h >= 16 && h < 18) {
      // Afternoon - slightly warm
      return '#FFF5E0'
    } else if (h >= 18 && h < 20) {
      // Sunset - orange-red
      return '#FF6B35'
    }
    // Night (shouldn't be visible anyway)
    return '#FFFAF0'
  }, [timeValue])

  // Don't render if below horizon
  if (!isAboveHorizon) {
    return null
  }

  // Perf: 16x16 segments — distant object, polygon difference invisible
  return (
    <mesh position={sunPosition}>
      <sphereGeometry args={[1.5, 16, 16]} />
      <meshBasicMaterial
        color={color}
        toneMapped={false}
        fog={false}
      />
    </mesh>
  )
}
