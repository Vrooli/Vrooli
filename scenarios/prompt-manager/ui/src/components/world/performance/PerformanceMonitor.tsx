/**
 * PerformanceMonitor - Invisible component that monitors FPS and adjusts graphics tier.
 * Place this component inside your R3F Canvas to enable automatic performance optimization.
 */

import { useEffect } from 'react'
import { useFPSMonitor } from '@/hooks/useFPSMonitor'
import { usePeriodicCleanup } from '@/hooks/useAssetDisposal'
import { useLODStore } from '@/stores/lodStore'
import type { PerformanceTier } from '@/types/graphics'

interface PerformanceMonitorProps {
  /** Whether to monitor FPS */
  enabled?: boolean
  /** Whether to automatically adjust graphics tier */
  autoAdjust?: boolean
  /** Callback when tier is adjusted */
  onTierAdjust?: (tier: PerformanceTier, reason: string) => void
  /** Whether to enable periodic memory cleanup */
  enableMemoryCleanup?: boolean
  /** Whether to enable LOD system */
  enableLOD?: boolean
  /** LOD configuration overrides */
  lodConfig?: {
    enableCursorLOD?: boolean
    enableAnimationLOD?: boolean
    enableHoverLOD?: boolean
  }
}

/**
 * Component that monitors performance and handles automatic optimization.
 *
 * @example
 * ```tsx
 * function App() {
 *   return (
 *     <Canvas>
 *       <PerformanceMonitor
 *         enabled
 *         autoAdjust
 *         enableMemoryCleanup
 *         enableLOD
 *         onTierAdjust={(tier, reason) => {
 *           console.log(`Graphics tier changed to ${tier}: ${reason}`)
 *         }}
 *       />
 *       <Scene />
 *     </Canvas>
 *   )
 * }
 * ```
 */
export function PerformanceMonitor({
  enabled = true,
  autoAdjust = true,
  onTierAdjust,
  enableMemoryCleanup = true,
  enableLOD = true,
  lodConfig,
}: PerformanceMonitorProps) {
  // FPS monitoring hook
  const { isDegraded, metrics } = useFPSMonitor({
    enabled,
    autoAdjust,
    onTierAdjust,
  })

  // Periodic memory cleanup
  usePeriodicCleanup(enableMemoryCleanup)

  // Configure LOD system
  useEffect(() => {
    if (!enableLOD) return

    const store = useLODStore.getState()
    store.setConfig({
      enableCursorLOD: lodConfig?.enableCursorLOD ?? true,
      enableAnimationLOD: lodConfig?.enableAnimationLOD ?? true,
      enableHoverLOD: lodConfig?.enableHoverLOD ?? true,
    })
  }, [enableLOD, lodConfig])

  // Log performance warnings in development
  useEffect(() => {
    if (isDegraded && process.env.NODE_ENV === 'development') {
      console.warn(
        `[PerformanceMonitor] Performance degraded: ${metrics.averageFps} FPS (target: 60)`
      )
    }
  }, [isDegraded, metrics.averageFps])

  // This component renders nothing
  return null
}

/**
 * Hook for checking if performance monitoring is degraded.
 */
export function useIsPerformanceDegraded(): boolean {
  const { isDegraded } = useFPSMonitor({ enabled: true, autoAdjust: false })
  return isDegraded
}
