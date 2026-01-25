/**
 * useDeviceCapability - Hook for detecting device capabilities.
 * Determines recommended graphics tier based on device hardware.
 */

import { useState, useEffect } from 'react'
import type { PerformanceTier, DeviceCapability } from '@/types/graphics'

/**
 * Detects device capabilities and returns recommended graphics tier.
 */
function detectCapabilities(): DeviceCapability {
  if (typeof window === 'undefined') {
    return {
      gpuTier: 1,
      isMobile: false,
      recommendedTier: 'medium',
    }
  }

  // Check if mobile
  const isMobile = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(
    navigator.userAgent
  )

  // Get device pixel ratio as rough GPU indicator
  const dpr = window.devicePixelRatio ?? 1

  // Try to detect GPU info (limited browser support)
  let gpuTier = 1
  try {
    const canvas = document.createElement('canvas')
    const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl')
    if (gl && 'getExtension' in gl) {
      const debugInfo = (gl as WebGLRenderingContext).getExtension('WEBGL_debug_renderer_info')
      if (debugInfo) {
        const renderer = (gl as WebGLRenderingContext).getParameter(
          debugInfo.UNMASKED_RENDERER_WEBGL
        )
        // Basic GPU tier detection based on renderer string
        if (renderer) {
          const rendererLower = renderer.toLowerCase()
          if (
            rendererLower.includes('nvidia') ||
            rendererLower.includes('radeon') ||
            rendererLower.includes('geforce')
          ) {
            gpuTier = 3
          } else if (
            rendererLower.includes('intel') ||
            rendererLower.includes('hd graphics')
          ) {
            gpuTier = 2
          } else if (
            rendererLower.includes('mali') ||
            rendererLower.includes('adreno')
          ) {
            gpuTier = 1
          }
        }
      }
    }
  } catch {
    // GPU detection failed, use defaults
  }

  // Determine recommended tier
  let recommendedTier: PerformanceTier = 'medium'

  if (isMobile) {
    recommendedTier = gpuTier >= 2 ? 'medium' : 'low'
  } else {
    if (gpuTier >= 3 && dpr >= 2) {
      recommendedTier = 'ultra'
    } else if (gpuTier >= 2 || dpr >= 1.5) {
      recommendedTier = 'high'
    } else {
      recommendedTier = 'medium'
    }
  }

  return {
    gpuTier,
    isMobile,
    recommendedTier,
  }
}

/**
 * Hook to detect device capabilities.
 *
 * @example
 * ```tsx
 * function App() {
 *   const { isMobile, recommendedTier } = useDeviceCapability()
 *
 *   if (isMobile) {
 *     return <MobileWarning />
 *   }
 *
 *   return <World initialTier={recommendedTier} />
 * }
 * ```
 */
export function useDeviceCapability(): DeviceCapability {
  const [capability, setCapability] = useState<DeviceCapability>(() => ({
    gpuTier: 1,
    isMobile: false,
    recommendedTier: 'medium',
  }))

  useEffect(() => {
    setCapability(detectCapabilities())
  }, [])

  return capability
}

/**
 * Hook to check if device is mobile.
 */
export function useIsMobile(): boolean {
  const [isMobile, setIsMobile] = useState(false)

  useEffect(() => {
    setIsMobile(
      /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(
        navigator.userAgent
      )
    )
  }, [])

  return isMobile
}

/**
 * Hook to get recommended tier.
 */
export function useRecommendedTier(): PerformanceTier {
  const { recommendedTier } = useDeviceCapability()
  return recommendedTier
}
