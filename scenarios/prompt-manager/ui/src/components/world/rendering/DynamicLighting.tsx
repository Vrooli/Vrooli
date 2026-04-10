/**
 * DynamicLighting - Renders lights synced with continuous time.
 * Directional light follows the sun position and adjusts color/intensity.
 */

import { useMemo } from 'react'
import { useEnvironmentStore, selectCurrentLighting } from '@/stores/environmentStore'
import { calculateLighting, calculateSunPosition } from '@/lib/sky/sunPosition'
import type { LightingPreset } from '@/types/environment'

interface DynamicLightingProps {
  /** Override lighting preset */
  preset?: LightingPreset
  /** Enable shadow casting for directional lights */
  enableShadows?: boolean
  /** Shadow map size */
  shadowMapSize?: number
  /** Whether to use continuous time-based lighting (default: true) */
  useContinuousTime?: boolean
}

/**
 * Renders lights synced with continuous time.
 * Main directional light follows sun position with time-based color/intensity.
 */
export function DynamicLighting({
  preset,
  enableShadows = true,
  shadowMapSize = 2048,
  useContinuousTime = true,
}: DynamicLightingProps) {
  const storeLighting = useEnvironmentStore(selectCurrentLighting)
  const timeValue = useEnvironmentStore((state) => state.timeValue)
  const lighting = preset ?? storeLighting

  // Calculate time-based lighting
  const timeLighting = useMemo(
    () => calculateLighting(timeValue),
    [timeValue]
  )

  // Calculate sun position for directional light
  const sunPosition = useMemo(
    () => calculateSunPosition(timeValue),
    [timeValue]
  )

  // Use time-based lighting if enabled, otherwise use preset
  const ambientColor = useContinuousTime ? timeLighting.ambientColor : lighting.ambient.color
  const ambientIntensity = useContinuousTime ? timeLighting.ambientIntensity : lighting.ambient.intensity
  const directionalColor = useContinuousTime ? timeLighting.color : (lighting.directional[0]?.color ?? '#ffffff')
  const directionalIntensity = useContinuousTime ? timeLighting.intensity : (lighting.directional[0]?.intensity ?? 1)
  const directionalPosition: [number, number, number] = useContinuousTime
    ? sunPosition
    : (lighting.directional[0]?.position ?? [10, 10, 5])

  return (
    <>
      {/* Ambient light - time-based color and intensity */}
      <ambientLight
        intensity={ambientIntensity}
        color={ambientColor}
      />

      {/* Main directional light - follows sun position */}
      <directionalLight
        position={directionalPosition}
        intensity={directionalIntensity}
        color={directionalColor}
        castShadow={enableShadows}
        shadow-mapSize={[shadowMapSize, shadowMapSize]}
        shadow-camera-near={0.1}
        shadow-camera-far={100}
        shadow-camera-left={-30}
        shadow-camera-right={30}
        shadow-camera-top={30}
        shadow-camera-bottom={-30}
      />

      {/* Additional directional lights from preset (if not using continuous time) */}
      {!useContinuousTime && lighting.directional.slice(1).map((light, idx) => (
        <directionalLight
          key={`dir-${idx + 1}`}
          position={light.position}
          intensity={light.intensity}
          color={light.color}
          castShadow={enableShadows && (light.castShadow ?? true)}
          shadow-mapSize={[
            light.shadowMapSize ?? shadowMapSize,
            light.shadowMapSize ?? shadowMapSize,
          ]}
          shadow-camera-near={0.1}
          shadow-camera-far={50}
          shadow-camera-left={-20}
          shadow-camera-right={20}
          shadow-camera-top={20}
          shadow-camera-bottom={-20}
        />
      ))}

      {/* Point lights from preset */}
      {Array.isArray(lighting.point) && lighting.point.map((light, idx) => (
        <pointLight
          key={`point-${idx}`}
          position={light.position}
          intensity={light.intensity}
          color={light.color}
          distance={light.distance ?? 0}
          decay={light.decay ?? 2}
        />
      ))}
    </>
  )
}
