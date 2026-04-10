/**
 * DynamicFog - Renders fog based on environment configuration.
 * Fog color is synced with the sky's horizon color so the horizon
 * blends seamlessly into the sky at any time of day.
 */

import { useThree } from '@react-three/fiber'
import { useEffect, useMemo } from 'react'
import * as THREE from 'three'
import { useEnvironmentStore, selectCurrentFog } from '@/stores/environmentStore'
import { calculateSkyColors } from '@/lib/sky/sunPosition'
import type { FogConfig } from '@/types/environment'

interface DynamicFogProps {
  /** Override fog config (uses its color as-is, bypassing sky sync) */
  config?: FogConfig | null
}

/**
 * Sets up scene fog based on environment config.
 * When using the store's fog config, the fog color is derived from the
 * sky's horizon (middle) color so the horizon never fades to a mismatched
 * static color like white.
 */
export function DynamicFog({ config }: DynamicFogProps) {
  const { scene } = useThree()
  const storeFog = useEnvironmentStore(selectCurrentFog)
  const timeValue = useEnvironmentStore((state) => state.timeValue)
  const fog = config !== undefined ? config : storeFog

  // Compute sky horizon color so fog blends with the sky
  const skyColors = useMemo(() => calculateSkyColors(timeValue), [timeValue])

  // When an explicit override config is provided, use its color.
  // Otherwise, derive fog color from the sky's horizon band.
  const fogColor = config !== undefined ? fog?.color : skyColors.middle

  useEffect(() => {
    if (fog) {
      scene.fog = new THREE.Fog(fogColor ?? fog.color, fog.near, fog.far)
    } else {
      scene.fog = null
    }

    return () => {
      scene.fog = null
    }
  }, [scene, fog, fogColor])

  // This component doesn't render anything - it just modifies the scene
  return null
}
