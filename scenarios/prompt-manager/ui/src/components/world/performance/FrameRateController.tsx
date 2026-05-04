/**
 * FrameRateController - owns the world Canvas frameloop.
 * Default mode is demand rendering; forceAlwaysFrameloop is only for A/B diagnostics.
 */
// AI_CHECK: R3F_FRAMELOOP_FPS_CAP=1 | LAST: 2026-02-17

import { useCallback, useEffect, useRef } from 'react'
import { useThree } from '@react-three/fiber'
import { usePerformanceStore } from '@/stores/performanceStore'
import { useGraphicsStore } from '@/stores/graphicsStore'
import { registerWorldRenderRequester } from './worldRenderLoop'

export function FrameRateController() {
  const maxFpsSetting = usePerformanceStore((state) => state.config.maxFps)
  const forceAlwaysFrameloop = usePerformanceStore((state) => state.config.forceAlwaysFrameloop)
  const showOverlay = usePerformanceStore((state) => state.config.showOverlay)
  const graphicsTier = useGraphicsStore((state) => state.tier)
  const maxFps = maxFpsSetting === 'auto'
    ? (graphicsTier === 'low' || graphicsTier === 'medium' ? 30 : 60)
    : maxFpsSetting
  const setFrameloop = useThree((state) => state.setFrameloop)
  const invalidate = useThree((state) => state.invalidate)
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const rafRef = useRef<number | null>(null)
  const pendingFramesRef = useRef(0)
  const lastInvalidateAtRef = useRef(0)

  const requestRender = useCallback((reason: string, frames = 1) => {
    const fpsCap = Number.isFinite(maxFps) ? Math.max(1, Math.round(maxFps)) : 60
    const frameDelay = Math.max(1, Math.round(1000 / fpsCap))
    const requestedFrames = Math.max(1, Math.min(120, Math.ceil(frames)))

    usePerformanceStore.getState().recordRenderRequest(reason)

    if (forceAlwaysFrameloop) {
      return
    }

    pendingFramesRef.current = Math.max(pendingFramesRef.current, requestedFrames)

    const flush = () => {
      timeoutRef.current = null
      rafRef.current = null
      if (pendingFramesRef.current <= 0) return

      const now = performance.now()
      const elapsed = now - lastInvalidateAtRef.current
      if (elapsed < frameDelay) {
        timeoutRef.current = setTimeout(flush, frameDelay - elapsed)
        return
      }

      lastInvalidateAtRef.current = now
      pendingFramesRef.current -= 1
      invalidate()
      usePerformanceStore.getState().recordInvalidate()
      if (pendingFramesRef.current > 0) {
        rafRef.current = window.requestAnimationFrame(flush)
      }
    }

    if (timeoutRef.current === null && rafRef.current === null) {
      rafRef.current = window.requestAnimationFrame(flush)
    }
  }, [forceAlwaysFrameloop, invalidate, maxFps])

  useEffect(() => {
    const fpsCap = Number.isFinite(maxFps) ? Math.max(1, Math.round(maxFps)) : 60
    usePerformanceStore.getState().setSceneSnapshot({
      effectiveMaxFps: fpsCap,
      frameloopMode: forceAlwaysFrameloop ? 'always' : 'demand',
      forceAlwaysFrameloop,
    })

    setFrameloop(forceAlwaysFrameloop ? 'always' : 'demand')
    pendingFramesRef.current = 0

    const unregister = registerWorldRenderRequester(requestRender)
    requestRender('scene-mount', 2)

    return () => {
      unregister()
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current)
        timeoutRef.current = null
      }
      if (rafRef.current !== null) {
        window.cancelAnimationFrame(rafRef.current)
        rafRef.current = null
      }
      setFrameloop('always')
    }
  }, [forceAlwaysFrameloop, maxFps, requestRender, setFrameloop])

  useEffect(() => {
    if (!showOverlay || forceAlwaysFrameloop) return
    const interval = window.setInterval(() => requestRender('diagnostics', 1), 1000)
    return () => window.clearInterval(interval)
  }, [forceAlwaysFrameloop, requestRender, showOverlay])

  return null
}
