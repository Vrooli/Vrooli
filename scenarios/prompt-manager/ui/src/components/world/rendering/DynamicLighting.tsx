/**
 * DynamicLighting - Renders lights based on environment configuration.
 * Responds to environment store changes for time of day and scene type.
 */

import { useEnvironmentStore, selectCurrentLighting } from '@/stores/environmentStore'
import type { LightingPreset } from '@/types/environment'

interface DynamicLightingProps {
  /** Override lighting preset */
  preset?: LightingPreset
  /** Enable shadow casting for directional lights */
  enableShadows?: boolean
  /** Shadow map size */
  shadowMapSize?: number
}

/**
 * Renders ambient, directional, and point lights based on environment config.
 * Automatically updates when the environment store changes.
 */
export function DynamicLighting({
  preset,
  enableShadows = true,
  shadowMapSize = 2048,
}: DynamicLightingProps) {
  const storeLighting = useEnvironmentStore(selectCurrentLighting)
  const lighting = preset ?? storeLighting

  return (
    <>
      {/* Ambient light */}
      <ambientLight
        intensity={lighting.ambient.intensity}
        color={lighting.ambient.color}
      />

      {/* Directional lights */}
      {lighting.directional.map((light, idx) => (
        <directionalLight
          key={`dir-${idx}`}
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

      {/* Point lights */}
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
