/**
 * useWorldDefaults - Hook to seed each scene with defaults on first visit.
 *
 * Distinguishes between `undefined` (never visited — seed) and `[]`
 * (explicitly cleared — skip seeding).
 */

import { useEffect, useRef } from 'react'
import { useFurnitureStore } from '@/stores/furnitureStore'
import { useDecorationStore } from '@/stores/decorationStore'
import { useEnvironmentStore } from '@/stores/environmentStore'
import type { SceneType } from '@/types/environment'
import { getSceneDefaults } from '@/config/sceneDefaults'
import type { SceneGeneratorContext } from '@/config/sceneDefaults'

/**
 * Hook to seed the current scene with defaults on first visit.
 * Runs whenever the active scene type changes.
 *
 * Reads store data imperatively via getState() inside the effect to avoid
 * subscribing to the entire `scenes` objects. Subscribing to `scenes` would
 * cause the effect to re-fire on every unrelated store update, which—combined
 * with the `?? []` selector fallbacks—could trigger an infinite re-render loop
 * (React error #185).
 */
export function useWorldDefaults(numAgents: number) {
  const sceneType = useEnvironmentStore((s) => s.current.type)
  const ctx: SceneGeneratorContext = { numAgents }

  // Track which scenes we've already attempted to seed this session,
  // so we don't re-seed if the user clears and then switches away/back.
  const seededRef = useRef<Set<SceneType>>(new Set())

  useEffect(() => {
    if (seededRef.current.has(sceneType)) return

    seededRef.current.add(sceneType)

    // Read store state imperatively — no reactive subscription needed here.
    const decoData = useDecorationStore.getState().scenes[sceneType]
    const furnData = useFurnitureStore.getState().scenes[sceneType]

    // `undefined` means never visited — seed from generator.
    // An explicit `[]` means the user cleared it — don't re-seed.
    if (decoData !== undefined && furnData !== undefined) return

    const defaults = getSceneDefaults(sceneType, ctx)

    if (decoData === undefined) {
      useDecorationStore.getState().seedScene(sceneType, defaults.decorations)
    }
    if (furnData === undefined) {
      useFurnitureStore.getState().seedScene(sceneType, defaults.furniture)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- ctx is rebuilt each render but we only seed once per scene
  }, [sceneType])
}

/**
 * Returns a function that resets the current scene to its generator defaults.
 */
export function useResetWorldToDefaults(numAgents: number) {
  const resetDecorations = useDecorationStore((s) => s.resetToDefaults)
  const resetFurniture = useFurnitureStore((s) => s.resetToDefaults)

  return () => {
    const ctx: SceneGeneratorContext = { numAgents }
    resetDecorations(undefined, ctx)
    resetFurniture(undefined, ctx)
  }
}
