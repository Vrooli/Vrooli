/**
 * DynamicSky - Renders a sky dome that changes based on continuous time.
 *
 * Uses a custom gradient shader that smoothly transitions colors based on time of day.
 * The gradient is rendered on BackSide, allowing stars and celestial bodies to render inside.
 */

import { useMemo, useRef } from 'react'
import { useFrame } from '@react-three/fiber'
import * as THREE from 'three'
import { useEnvironmentStore } from '@/stores/environmentStore'
import {
  calculateSunPosition,
  calculateSkyColors,
} from '@/lib/sky/sunPosition'
import type { SceneType } from '@/types/environment'
import { SKY_VERTEX_SHADER, SKY_FRAGMENT_SHADER } from '@/lib/shaders/glsl/sky.glsl'

interface DynamicSkyProps {
  /** Override the time value (0-24 hours) */
  timeValue?: number
  /** Radius of the sky dome */
  radius?: number
  /** Override scene type */
  sceneType?: SceneType
}

/**
 * Convert hex color to THREE.Color
 */
function hexToColor(hex: string): THREE.Color {
  return new THREE.Color(hex)
}

/**
 * Create a gradient shader material for the sky dome (used for abstract-space)
 */
function createGradientMaterial(colors: { top: string; middle: string; bottom: string }): THREE.ShaderMaterial {
  return new THREE.ShaderMaterial({
    uniforms: {
      topColor: { value: hexToColor(colors.top) },
      middleColor: { value: hexToColor(colors.middle) },
      bottomColor: { value: hexToColor(colors.bottom) },
      offset: { value: 0.5 },
      exponent: { value: 0.6 },
    },
    vertexShader: SKY_VERTEX_SHADER,
    fragmentShader: SKY_FRAGMENT_SHADER,
    side: THREE.BackSide,
    depthWrite: false,
  })
}

/**
 * Create a gradient material from color array (legacy support)
 */
function createGradientMaterialFromArray(colors: string[]): THREE.ShaderMaterial {
  const colorArray = colors.map(hexToColor)

  return new THREE.ShaderMaterial({
    uniforms: {
      topColor: { value: colorArray[0] ?? new THREE.Color('#87CEEB') },
      middleColor: { value: colorArray[1] ?? colorArray[0] ?? new THREE.Color('#ADD8E6') },
      bottomColor: { value: colorArray[2] ?? colorArray[1] ?? colorArray[0] ?? new THREE.Color('#FFF8DC') },
      offset: { value: 0.5 },
      exponent: { value: 0.6 },
    },
    vertexShader: SKY_VERTEX_SHADER,
    fragmentShader: SKY_FRAGMENT_SHADER,
    side: THREE.BackSide,
    depthWrite: false,
  })
}

/**
 * Create a solid color material for the sky
 */
function createSolidMaterial(color: string): THREE.MeshBasicMaterial {
  return new THREE.MeshBasicMaterial({
    color: new THREE.Color(color),
    side: THREE.BackSide,
    depthWrite: false,
  })
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
  sceneType: _sceneTypeProp, // Currently unused - gradient shader used for all scene types
}: DynamicSkyProps) {
  const meshRef = useRef<THREE.Mesh>(null)

  // Get current environment config
  const currentEnv = useEnvironmentStore((state) => state.current)
  const storeTimeValue = useEnvironmentStore((state) => state.timeValue)
  const skyboxConfig = currentEnv.skybox

  // Use prop time value, or convert legacy timeOfDay prop, or use store value
  const timeValue = timeValueProp ?? storeTimeValue

  // Calculate sky colors based on continuous time
  const skyColors = useMemo(() => calculateSkyColors(timeValue), [timeValue])

  // Create the gradient material - always uses time-based colors for procedural types
  const gradientMaterial = useMemo(() => {
    // For solid skybox type with explicit source, use that
    if (skyboxConfig.type === 'solid' && typeof skyboxConfig.source === 'string') {
      return createSolidMaterial(skyboxConfig.source)
    }

    // For gradient type with explicit color array, use that
    if (skyboxConfig.type === 'gradient' && Array.isArray(skyboxConfig.source)) {
      return createGradientMaterialFromArray(skyboxConfig.source)
    }

    // For all other cases (procedural, hdri without source, or default), use time-based colors
    return createGradientMaterial(skyColors)
  }, [skyboxConfig, skyColors])

  // Slowly rotate the sky dome for a subtle effect
  useFrame((_, delta) => {
    if (meshRef.current) {
      meshRef.current.rotation.y += delta * 0.01
    }
  })

  // Render the gradient sky dome
  return (
    <mesh ref={meshRef} material={gradientMaterial}>
      <sphereGeometry args={[radius, 32, 32]} />
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

  return (
    <mesh position={sunPosition}>
      <sphereGeometry args={[1.5, 32, 32]} />
      <meshBasicMaterial
        color={color}
        toneMapped={false}
        fog={false}
      />
    </mesh>
  )
}
