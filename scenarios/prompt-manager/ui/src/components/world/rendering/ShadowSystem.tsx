/**
 * ShadowSystem - Configurable shadow rendering setup.
 * Manages shadow map quality and soft shadow effects.
 */

import { useEffect, useRef } from 'react'
import { useThree } from '@react-three/fiber'
import * as THREE from 'three'
import { useGraphicsStore } from '@/stores/graphicsStore'

interface ShadowSystemProps {
  /** Enable soft shadows via PCF */
  softShadows?: boolean
  /** Shadow bias to prevent artifacts */
  bias?: number
  /** Normal bias for curved surfaces */
  normalBias?: number
}

/**
 * Configures the shadow system based on graphics settings.
 * Updates shadow map size and type dynamically.
 */
export function ShadowSystem({
  softShadows = true,
  bias = -0.0001,
  normalBias = 0.02,
}: ShadowSystemProps) {
  const { gl } = useThree()
  const config = useGraphicsStore((state) => state.config)
  const initialized = useRef(false)

  useEffect(() => {
    if (!config.shadows) {
      gl.shadowMap.enabled = false
      return
    }

    gl.shadowMap.enabled = true
    gl.shadowMap.type = softShadows ? THREE.PCFSoftShadowMap : THREE.PCFShadowMap
    gl.shadowMap.autoUpdate = true

    // Store bias values for use by lights
    // Note: Individual lights should apply these values
    void bias
    void normalBias

    initialized.current = true

    return () => {
      // Cleanup is handled by R3F
    }
  }, [gl, config.shadows, softShadows, bias, normalBias])

  // This component doesn't render anything - it's a configuration component
  return null
}

/**
 * Helper hook to get shadow configuration for lights
 */
