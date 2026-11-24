/**
 * Operator world preferences persisted through WorldService, shared across
 * browsers. Loaded once on mount; saved with a debounce on change.
 */
import { useCallback, useEffect, useRef, useState } from 'react'
import { Code } from '@connectrpc/connect'
import type { PeriodId, QualityProfileId, SceneId } from '../config'
import { isPeriodId, isQualityProfileId, isSceneId, tuning } from '../config'
import { createWorldApi, type WorldApi } from './worldClient'

export interface WorldPreferences {
  scene: SceneId
  qualityProfile: QualityProfileId
  qualityAuto: boolean
  periodMode: 'clock' | PeriodId
  twoDMode: boolean
  showDiagnostics: boolean
  scale: number
  zoomTarget: 'cursor' | 'center'
  ambientLife: boolean
}

export const DEFAULT_PREFERENCES: WorldPreferences = {
  scene: 'park',
  qualityProfile: tuning.quality.defaultProfile,
  qualityAuto: true,
  periodMode: 'clock',
  twoDMode: false,
  showDiagnostics: false,
  scale: 1,
  zoomTarget: 'cursor',
  ambientLife: true,
}

const SAVE_DEBOUNCE_MS = 600

/**
 * Observable persistence state for operator world preferences. Edits are held
 * locally, so a failed or conflicting save never discards them; the caller can
 * surface the state and offer `retry`.
 */
export type WorldPreferencesSaveStatus =
  | 'idle'
  | 'dirty'
  | 'saving'
  | 'saved'
  | 'error'
  | 'unavailable'
  | 'conflict'

export interface WorldPreferencesSaveState {
  status: WorldPreferencesSaveStatus
  message: string | null
  at: number | null
}

const SAVE_MESSAGES: Record<WorldPreferencesSaveStatus, string | null> = {
  idle: null,
  dirty: 'Unsaved changes.',
  saving: 'Saving changes…',
  saved: 'Changes saved.',
  error: 'Could not save changes. Your edits are kept; retry when ready.',
  unavailable: 'The settings service is unavailable. Your edits are kept; retry when the connection returns.',
  conflict: 'These settings changed elsewhere. Your edits are kept; retry to save your version.',
}

/** Human-readable status text, preferring any detail recorded with the state. */
export function describeSaveState(state: WorldPreferencesSaveState): string | null {
  return state.message ?? SAVE_MESSAGES[state.status]
}

/** Map a save failure to the state the operator should see. */
export function classifySaveError(error: unknown): WorldPreferencesSaveStatus {
  const code = (error as { code?: unknown } | null)?.code
  if (code === Code.Unavailable || code === Code.DeadlineExceeded) return 'unavailable'
  if (code === Code.AlreadyExists || code === Code.FailedPrecondition || code === Code.Aborted) return 'conflict'
  const text = error instanceof Error ? error.message : typeof error === 'string' ? error : ''
  if (/failed to fetch|network|load failed|timed? ?out|unavailable|offline/i.test(text)) return 'unavailable'
  if (/conflict|already exists|precondition|stale/i.test(text)) return 'conflict'
  return 'error'
}

export function preferencesFromWire(wire: { scene: string; qualityProfile: string; qualityAuto: boolean; periodMode: string; twoDMode: boolean; showDiagnostics: boolean; scale: number; zoomTarget?: string; ambientLife?: boolean }): WorldPreferences {
  return {
    scene: isSceneId(wire.scene) ? wire.scene : DEFAULT_PREFERENCES.scene,
    qualityProfile: isQualityProfileId(wire.qualityProfile) ? wire.qualityProfile : DEFAULT_PREFERENCES.qualityProfile,
    qualityAuto: wire.qualityAuto,
    periodMode: wire.periodMode === 'clock' ? 'clock' : isPeriodId(wire.periodMode) ? wire.periodMode : 'clock',
    twoDMode: wire.twoDMode,
    showDiagnostics: wire.showDiagnostics,
    scale: wire.scale > 0 ? wire.scale : 1,
    zoomTarget: wire.zoomTarget === 'center' ? 'center' : 'cursor',
    ambientLife: wire.ambientLife ?? true,
  }
}

export interface UseWorldPreferences {
  preferences: WorldPreferences
  loaded: boolean
  update: (patch: Partial<WorldPreferences>) => void
  /** Observable persistence state; `retry` re-attempts the last failed save. */
  saveState: WorldPreferencesSaveState
  retry: () => void
}

export function useWorldPreferences(api?: WorldApi, enabled = true): UseWorldPreferences {
  const [preferences, setPreferences] = useState<WorldPreferences>(DEFAULT_PREFERENCES)
  const [loaded, setLoaded] = useState(!enabled)
  const [saveState, setSaveState] = useState<WorldPreferencesSaveState>({ status: 'idle', message: null, at: null })
  const apiRef = useRef<WorldApi | null>(api ?? null)
  const enabledRef = useRef(enabled)
  enabledRef.current = enabled
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const dirty = useRef(false)
  // The newest preferences, tracked outside React state so a debounced or
  // retried save always sends the operator's latest edits.
  const latest = useRef(DEFAULT_PREFERENCES)

  useEffect(() => {
    if (!enabled) return
    let cancelled = false
    apiRef.current ??= createWorldApi()
    apiRef.current
      .getConfig()
      .then((wire) => {
        if (cancelled || dirty.current) return
        const mapped = preferencesFromWire(wire)
        latest.current = mapped
        setPreferences(mapped)
      })
      .catch(() => undefined)
      .finally(() => {
        if (!cancelled) setLoaded(true)
      })
    return () => {
      cancelled = true
    }
  }, [enabled])

  const saveLatest = useCallback(() => {
    if (!enabledRef.current) return
    if (timer.current) {
      clearTimeout(timer.current)
      timer.current = null
    }
    const snapshot = latest.current
    apiRef.current ??= createWorldApi()
    setSaveState({ status: 'saving', message: null, at: Date.now() })
    return apiRef.current
      .setConfig(snapshot)
      .then(() => {
        // A newer edit arrived while this save was in flight; its own debounce
        // owns the final status, so do not report saved yet.
        if (latest.current !== snapshot) return
        dirty.current = false
        setSaveState({ status: 'saved', message: null, at: Date.now() })
      })
      .catch((error: unknown) => {
        const status = classifySaveError(error)
        setSaveState({ status, message: SAVE_MESSAGES[status], at: Date.now() })
      })
  }, [])

  const update = useCallback(
    (patch: Partial<WorldPreferences>) => {
      dirty.current = true
      setPreferences((prev) => {
        const next = { ...prev, ...patch }
        latest.current = next
        return next
      })
      if (!enabledRef.current) return
      setSaveState({ status: 'dirty', message: null, at: null })
      if (timer.current) clearTimeout(timer.current)
      timer.current = setTimeout(() => {
        timer.current = null
        void saveLatest()
      }, SAVE_DEBOUNCE_MS)
    },
    [saveLatest],
  )

  const retry = useCallback(() => {
    void saveLatest()
  }, [saveLatest])

  useEffect(() => () => {
    if (timer.current) clearTimeout(timer.current)
  }, [])

  return { preferences, loaded, update, saveState, retry }
}
