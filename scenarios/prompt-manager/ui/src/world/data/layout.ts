/**
 * Per-scene layout persistence through WorldService, with a debounced save.
 */
import { useCallback, useEffect, useRef, useState } from 'react'
import type { SceneId } from '../config'
import type { LayoutOverride } from '../sim'
import { createWorldApi, type WorldApi } from './worldClient'

export interface LayoutPersistence {
  overrides: LayoutOverride[]
  loaded: boolean
  saving: boolean
  error: string | null
  retry: () => void
  save: (overrides: LayoutOverride[]) => void
}

export function overridesFromWire(wire: { overrides: Array<{ placeId: string; position?: { x: number; z: number }; rotation?: number; removed: boolean }> }): LayoutOverride[] {
  return wire.overrides.map((o) => ({
    placeId: o.placeId === 'commons' ? 'gathering' : o.placeId === 'campfire' ? 'hearth' : o.placeId,
    position: o.position ? ([o.position.x, o.position.z] as const) : undefined,
    rotation: o.rotation,
    removed: o.removed || undefined,
  }))
}

export function overridesToWire(scene: SceneId, overrides: readonly LayoutOverride[]) {
  return {
    scene,
    overrides: overrides.map((o) => ({
      placeId: o.placeId,
      position: o.position ? { x: o.position[0], z: o.position[1] } : undefined,
      rotation: o.rotation,
      removed: o.removed ?? false,
    })),
    decor: [],
  }
}

export function useLayoutPersistence(scene: SceneId, enabled: boolean, debounceMs: number, api?: WorldApi, initial?: { scene: SceneId; overrides: LayoutOverride[] }): LayoutPersistence {
  const imported = useRef(initial)
  const initialOverrides = !enabled && imported.current?.scene === scene ? imported.current.overrides : []
  const [snapshot, setSnapshot] = useState<{ scene: SceneId; enabled: boolean; overrides: LayoutOverride[]; loaded: boolean; error?: string }>({ scene, enabled, overrides: initialOverrides, loaded: !enabled })
  const apiRef = useRef<WorldApi | null>(api ?? null)
  const pending = useRef<{ scene: SceneId; version: number; overrides: LayoutOverride[]; timer: ReturnType<typeof setTimeout> } | null>(null)
  const writes = useRef(Promise.resolve())
  const editVersion = useRef(0)
  const drafts = useRef(new Map<SceneId, { version: number; overrides: LayoutOverride[]; saving: boolean; error: string | null }>())
  const [, refresh] = useState(0)
  const [loadAttempt, retryLoad] = useState(0)

  const flush = useCallback(() => {
    const save = pending.current
    if (!save) return
    pending.current = null
    clearTimeout(save.timer)
    apiRef.current ??= createWorldApi()
    const client = apiRef.current
    // Serialize writes so a slow earlier request cannot overwrite the latest edit.
    const complete = (error: string | null) => {
      const draft = drafts.current.get(save.scene)
      if (draft?.version !== save.version) return
      drafts.current.set(save.scene, { ...draft, saving: false, error })
      refresh(value => value + 1)
    }
    writes.current = writes.current.then(() => client.setLayout(overridesToWire(save.scene, save.overrides))).then(() => complete(null)).catch((error: unknown) => complete(error instanceof Error ? error.message : String(error)))
  }, [])

  useEffect(() => {
    if (!enabled) {
      setSnapshot({ scene, enabled, overrides: imported.current?.scene === scene ? imported.current.overrides : [], loaded: true })
      return flush
    }
    let cancelled = false
    const version = editVersion.current
    const draft = drafts.current.get(scene)
    if (draft && (draft.error || draft.saving)) {
      setSnapshot({ scene, enabled, overrides: draft.overrides, loaded: true })
      return flush
    }
    setSnapshot({ scene, enabled, overrides: [], loaded: false })
    apiRef.current ??= createWorldApi()
    const client = apiRef.current
    // Returning to a scene waits for its pending writes before loading it.
    void writes.current.then(() => client.getLayout(scene)).then(wire => {
      if (!cancelled && version === editVersion.current) setSnapshot({ scene, enabled, overrides: overridesFromWire(wire), loaded: true })
    }).catch((error: unknown) => {
      if (!cancelled && version === editVersion.current) setSnapshot({ scene, enabled, overrides: [], loaded: false, error: error instanceof Error ? error.message : String(error) })
    })
    return () => { cancelled = true; flush() }
  }, [scene, enabled, flush, loadAttempt])

  const save = useCallback((next: LayoutOverride[]) => {
    editVersion.current++
    setSnapshot({ scene, enabled, overrides: next, loaded: true })
    if (!enabled) return
    drafts.current.set(scene, { version: editVersion.current, overrides: next, saving: true, error: null })
    if (pending.current) {
      if (pending.current.scene === scene) clearTimeout(pending.current.timer)
      else flush()
    }
    pending.current = { scene, version: editVersion.current, overrides: next, timer: setTimeout(flush, debounceMs) }
  }, [scene, enabled, debounceMs, flush])

  const current = snapshot.scene === scene && snapshot.enabled === enabled
  const draft = enabled ? drafts.current.get(scene) : undefined
  return {
    overrides: current ? snapshot.overrides : [], loaded: current && snapshot.loaded, save,
    saving: draft?.saving ?? false,
    error: draft?.error ?? (current ? snapshot.error ?? null : null),
    retry: () => {
      if (draft?.error) save(draft.overrides)
      else retryLoad(value => value + 1)
    },
  }
}
