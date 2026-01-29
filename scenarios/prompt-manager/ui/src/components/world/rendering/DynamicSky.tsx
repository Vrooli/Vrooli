/**
 * DynamicSky - Renders a sky dome that changes based on time of day.
 * Uses procedural gradients for morning, noon, sunset, and night.
 */

import { useMemo, useRef } from 'react'
import { useFrame } from '@react-three/fiber'
import * as THREE from 'three'
import { useEnvironmentStore } from '@/stores/environmentStore'
import { SKYBOX_PRESETS } from '@/config/environments'
import type { TimeOfDay } from '@/types/environment'
import { SKY_VERTEX_SHADER, SKY_FRAGMENT_SHADER } from '@/lib/shaders/glsl/sky.glsl'

interface DynamicSkyProps {
  /** Override the time of day */
  timeOfDay?: TimeOfDay
  /** Radius of the sky dome */
  radius?: number
  /** Enable stars at night */
  enableStars?: boolean
}

/**
 * Convert hex color to THREE.Color
 */
function hexToColor(hex: string): THREE.Color {
  return new THREE.Color(hex)
}

/**
 * Create a gradient shader material for the sky dome
 */
function createGradientMaterial(colors: string[]): THREE.ShaderMaterial {
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
 * Dynamic sky dome that changes based on time of day.
 */
export function DynamicSky({
  timeOfDay: timeOfDayProp,
  radius = 50,
}: DynamicSkyProps) {
  const meshRef = useRef<THREE.Mesh>(null)

  // Get current environment config
  const currentEnv = useEnvironmentStore((state) => state.current)
  const timeOfDay = timeOfDayProp ?? currentEnv.timeOfDay
  const skyboxConfig = currentEnv.skybox

  // Create the appropriate material based on skybox type
  const material = useMemo(() => {
    switch (skyboxConfig.type) {
      case 'gradient':
        if (Array.isArray(skyboxConfig.source)) {
          return createGradientMaterial(skyboxConfig.source)
        }
        return createSolidMaterial(skyboxConfig.source ?? '#87CEEB')

      case 'solid':
        return createSolidMaterial(
          typeof skyboxConfig.source === 'string' ? skyboxConfig.source : '#87CEEB'
        )

      case 'procedural': {
        // For procedural, use gradient with preset colors
        const presetColors = SKYBOX_PRESETS[timeOfDay]
        if (Array.isArray(presetColors.source)) {
          return createGradientMaterial(presetColors.source)
        }
        return createSolidMaterial('#87CEEB')
      }

      default:
        return createSolidMaterial('#87CEEB')
    }
  }, [skyboxConfig, timeOfDay])

  // Slowly rotate the sky dome for a subtle effect
  useFrame((_, delta) => {
    if (meshRef.current) {
      meshRef.current.rotation.y += delta * 0.01
    }
  })

  return (
    <mesh ref={meshRef} material={material}>
      <sphereGeometry args={[radius, 32, 32]} />
    </mesh>
  )
}

/**
 * Sun/Moon component that positions based on time of day
 */
export function CelestialBody({ timeOfDay: timeOfDayProp }: { timeOfDay?: TimeOfDay }) {
  const currentEnv = useEnvironmentStore((state) => state.current)
  const timeOfDay = timeOfDayProp ?? currentEnv.timeOfDay

  // Position based on time of day
  const position = useMemo<[number, number, number]>(() => {
    switch (timeOfDay) {
      case 'morning':
        return [30, 15, 30] // Rising sun
      case 'noon':
        return [0, 40, 0] // High sun
      case 'sunset':
        return [-30, 10, 30] // Setting sun
      case 'night':
        return [20, 35, -20] // Moon position
      default:
        return [0, 40, 0]
    }
  }, [timeOfDay])

  // Color based on time of day
  const color = useMemo(() => {
    switch (timeOfDay) {
      case 'morning':
        return '#FFE4B5' // Warm yellow
      case 'noon':
        return '#FFFAF0' // Bright white-yellow
      case 'sunset':
        return '#FF6B35' // Orange-red
      case 'night':
        return '#E8E8E8' // Moon white
      default:
        return '#FFFAF0'
    }
  }, [timeOfDay])

  const size = timeOfDay === 'night' ? 1.5 : 2

  return (
    <mesh position={position}>
      <sphereGeometry args={[size, 32, 32]} />
      <meshBasicMaterial
        color={color}
        toneMapped={false}
      />
    </mesh>
  )
}
