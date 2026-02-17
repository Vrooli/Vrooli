/**
 * FrameRateController - caps Canvas rendering FPS using demand invalidation.
 * Uses performance store config and tier defaults.
 */
// AI_CHECK: R3F_FRAMELOOP_FPS_CAP=1 | LAST: 2026-02-17

import { useEffect, useRef } from 'react'
import { useThree } from '@react-three/fiber'
import { usePerformanceStore } from '@/stores/performanceStore'
import { useGraphicsStore } from '@/stores/graphicsStore'

export function FrameRateController() {
  const maxFpsSetting = usePerformanceStore((state) => state.config.maxFps)
  const graphicsTier = useGraphicsStore((state) => state.tier)
  const maxFps = maxFpsSetting === 'auto'
    ? (graphicsTier === 'low' || graphicsTier === 'medium' ? 30 : 60)
    : maxFpsSetting
  const setFrameloop = useThree((state) => state.setFrameloop)
  const invalidate = useThree((state) => state.invalidate)
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    const fpsCap = Number.isFinite(maxFps) ? Math.max(1, Math.round(maxFps)) : 60

    if (fpsCap >= 60) {
      setFrameloop('always')
      return
    }

    const frameDelay = Math.max(1, Math.round(1000 / fpsCap))
    let active = true
    setFrameloop('demand')

    const tick = () => {
      if (!active) return
      invalidate()
      timeoutRef.current = setTimeout(tick, frameDelay)
    }

    tick()

    return () => {
      active = false
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current)
        timeoutRef.current = null
      }
      setFrameloop('always')
    }
  }, [invalidate, maxFps, setFrameloop])

  return null
}
