import { useProgress } from '@react-three/drei'
import { useFrame, useThree } from '@react-three/fiber'
import { useEffect, useRef } from 'react'
import type { QualityProfile, Scene } from '../../config'
import { updateDiagnostics } from '../diagnostics/store'

export interface ShadowWorldStore {
  getState(): {
    revision: number
    actorOrder: string[]
    actors: Record<string, { speed: number } | undefined>
  }
}

interface ShadowMapState {
  autoUpdate: boolean
  needsUpdate: boolean
}

export interface ShadowRefreshController {
  request(): void
  dispose(): void
}

/** Accumulate wall-clock seconds, independent of display refresh rate. */
export function advanceShadowClock(elapsed: number, delta: number, hz: number): { elapsed: number; refresh: boolean } {
  if (hz <= 0) return { elapsed: 0, refresh: false }
  const total = elapsed + Math.max(0, delta)
  const interval = 1 / hz
  return total + Number.EPSILON >= interval
    ? { elapsed: Math.max(0, total - interval * Math.floor((total + Number.EPSILON) / interval)), refresh: true }
    : { elapsed: total, refresh: false }
}

export function installShadowRefresh(shadowMap: ShadowMapState, onRefresh: () => void = () => undefined): ShadowRefreshController {
  const previousAutoUpdate = shadowMap.autoUpdate
  shadowMap.autoUpdate = false
  return {
    request() {
      shadowMap.needsUpdate = true
      onRefresh()
    },
    dispose() {
      shadowMap.autoUpdate = previousAutoUpdate
    },
  }
}

/** Own explicit shadow-map invalidation for static world geometry. */
export function useShadowRefresh(store: ShadowWorldStore, scene: Scene, profile: QualityProfile, periodKey: string): void {
  const gl = useThree((state) => state.gl)
  const { active, progress } = useProgress()
  const controller = useRef<ShadowRefreshController | null>(null)
  const refreshes = useRef(0)
  const lastRevision = useRef(store.getState().revision)
  const movingSeconds = useRef(0)

  useEffect(() => {
    refreshes.current = 0
    updateDiagnostics({ shadowRefreshes: 0 })
    const installed = installShadowRefresh(gl.shadowMap, () => {
      refreshes.current += 1
      updateDiagnostics({ shadowRefreshes: refreshes.current })
    })
    controller.current = installed
    installed.request()
    return () => {
      installed.dispose()
      controller.current = null
    }
  }, [gl])

  useEffect(() => {
    controller.current?.request()
  }, [periodKey, profile.shadowMapSize, profile.shadows, scene.id])

  useEffect(() => {
    if (!active && progress >= 100) controller.current?.request()
  }, [active, progress])

  useFrame((_, delta) => {
    const state = store.getState()
    if (state.revision !== lastRevision.current) {
      lastRevision.current = state.revision
      controller.current?.request()
    }
    if (!profile.shadows || profile.shadowRefreshHz <= 0) return
    const moving = state.actorOrder.some((id) => (state.actors[id]?.speed ?? 0) > 0)
    if (!moving) {
      movingSeconds.current = 0
      return
    }
    const next = advanceShadowClock(movingSeconds.current, delta, profile.shadowRefreshHz)
    movingSeconds.current = next.elapsed
    if (next.refresh) controller.current?.request()
  })
}
