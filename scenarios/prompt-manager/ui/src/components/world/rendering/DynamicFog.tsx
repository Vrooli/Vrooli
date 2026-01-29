/**
 * DynamicFog - Renders fog based on environment configuration.
 */

import { useThree } from '@react-three/fiber'
import { useEffect } from 'react'
import * as THREE from 'three'
import { useEnvironmentStore, selectCurrentFog } from '@/stores/environmentStore'
import type { FogConfig } from '@/types/environment'

interface DynamicFogProps {
  /** Override fog config */
  config?: FogConfig | null
}

/**
 * Sets up scene fog based on environment config.
 * Automatically updates when the environment store changes.
 */
export function DynamicFog({ config }: DynamicFogProps) {
  const { scene } = useThree()
  const storeFog = useEnvironmentStore(selectCurrentFog)
  const fog = config !== undefined ? config : storeFog

  useEffect(() => {
    if (fog) {
      scene.fog = new THREE.Fog(fog.color, fog.near, fog.far)
    } else {
      scene.fog = null
    }

    return () => {
      scene.fog = null
    }
  }, [scene, fog])

  // This component doesn't render anything - it just modifies the scene
  return null
}

/**
 * Hook to get the current fog color (useful for background matching)
 */
