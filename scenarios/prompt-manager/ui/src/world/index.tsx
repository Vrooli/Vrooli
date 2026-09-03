/**
 * /world route component. The only module that composes scene and hud.
 *
 * URL levers (all optional, used by the smoke tool and deep links):
 *   ?scene=park|office   ?profile=low|medium|high|ultra (manual, disables auto)
 *   ?period=dawn|day|dusk|night   ?intro=0 (skip the dolly)   ?diag=1 (overlay)
 *   ?seed=<int> (sim seed)   ?actors=<n> (synthetic roster, no feed: goldens and demos)
 *   ?ao=0 ?bloom=0 ?shadows=0 (diagnostic overrides of the active profile's effects)
 *   ?view=3d|2d (deep-link intent: outranks the stored 2D preference and the narrow-screen default, never a missing WebGL)
 */
import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { Box3, Vector3 } from 'three'
import { useSearchParams } from 'react-router-dom'
import { ViewOverlay } from '@/components/shared/ViewOverlay'
import { useMediaQuery } from '@/hooks/useMediaQuery'
import {
  isPeriodId,
  isQualityProfileId,
  isSceneId,
  periodForHour,
  resolvePeriod,
  scenes,
  tuning as shippedTuning,
  withTuningOverride,
  type PeriodId,
  type QualityProfileId,
  type QualityState,
  type SceneId,
  type TuningOverride,
} from './config'
import {
  CameraRig,
  DiagnosticsOverlay,
  DiagnosticsProbe,
  LightingRig,
  PostChain,
  QualityGovernor,
  WorldCanvas,
  applyVerdict,
  pickProfile,
  setAuto,
  type CameraRigHandle,
  type WorldBounds,
} from './engine'
import { Actors, Labels, Places, Props, RoomHandles, SceneEnvironment, Stage, Trees, WorldStoreContext } from './scene'
import { EMPTY_FILTERS, EditorToolbar, WorldHelpContent, WorldHud, WorldSettingsContent, type FilterState, type SummaryFilter } from './hud'
import { createWorldActions, syntheticRoster, useLayoutPersistence, useWorldPreferences, useWorldRoster, useWorldRuntime } from './data'
import { canRedo, canUndo, commit, emptyHistory, redo, undo, upsertOverride, type OverrideHistory } from './sim'
import { poseToPosition } from './engine'

export interface WorldViewProps {
  onOpenMobileSidebar?: () => void
  pendingWorkCount?: number
  runningAgentCount?: number
  homeView?: 'world' | 'graph'
  onHomeViewChange?: (view: 'world' | 'graph') => void
  leftPanelContent?: ReactNode
}

export type PeriodMode = { kind: 'clock' } | { kind: 'fixed'; period: PeriodId }

const DEFAULT_SEED = 1
/** The ground disc reaches this share of the far plane; fog hides its edge before the clip does. */
const HORIZON_SHARE = 0.8
const SYNTHETIC_PER_TEAM = 5
const TICKER_LIMIT = 12

function hasWebGL(): boolean {
  try {
    const canvas = document.createElement('canvas')
    return canvas.getContext('webgl2') !== null
  } catch {
    return false
  }
}

function parseInt10(value: string | null, fallback: number): number {
  const n = value === null ? NaN : Number.parseInt(value, 10)
  return Number.isFinite(n) && n >= 0 ? n : fallback
}

export function WorldView(props: WorldViewProps) {
  const [params] = useSearchParams()
  const sceneParam = params.get('scene')
  const profileParam = params.get('profile')
  const periodParam = params.get('period')
  const reducedMotion = useMediaQuery('(prefers-reduced-motion: reduce)')
  const intro = params.get('intro') !== '0'
  const showDiagnosticsParam = params.get('diag') === '1'

  const [sceneId, setSceneId] = useState<SceneId>(() => (isSceneId(sceneParam) ? sceneParam : 'park'))
  const [quality, setQuality] = useState<QualityState>(() =>
    isQualityProfileId(profileParam)
      ? { auto: false, profileId: profileParam }
      : { auto: true, profileId: shippedTuning.quality.defaultProfile },
  )
  const [periodMode, setPeriodMode] = useState<PeriodMode>(() =>
    isPeriodId(periodParam) ? { kind: 'fixed', period: periodParam } : { kind: 'clock' },
  )
  const [showDiagnostics, setShowDiagnostics] = useState(showDiagnosticsParam)
  const cameraRig = useRef<CameraRigHandle | null>(null)
  const seed = parseInt10(params.get('seed'), DEFAULT_SEED)
  const syntheticActors = parseInt10(params.get('actors'), 0)
  const [focusedId, setFocusedId] = useState<string | null>(params.get('focus'))
  const [hoveredId, setHoveredId] = useState<string | null>(null)
  const [following, setFollowing] = useState(false)
  const [filters, setFilters] = useState<FilterState>(EMPTY_FILTERS)
  const [summaryFilter, setSummaryFilter] = useState<SummaryFilter | null>(null)
  const [highlightedTeamId, setHighlightedTeamId] = useState<string | null>(null)
  const narrow = useMediaQuery('(max-width: 767px)')
  const webglAvailable = useMemo(() => hasWebGL(), [])
  const preferences = useWorldPreferences(undefined, syntheticActors === 0)
  const [twoDChoice, setTwoDChoice] = useState<boolean | null>(null)
  // Dev levers: an override merged over the shipped tuning, re-validated on every edit.
  const [tuningOverride, setTuningOverride] = useState<TuningOverride>({})
  const tuning = useMemo(() => (Object.keys(tuningOverride).length > 0 ? withTuningOverride(tuningOverride, shippedTuning) : shippedTuning), [tuningOverride])
  const viewParam = params.get('view')
  const requestedTwoD = viewParam === '2d' ? true : viewParam === '3d' ? false : null
  const twoD = !webglAvailable || (twoDChoice ?? requestedTwoD ?? (preferences.preferences.twoDMode || narrow))

  const scene = scenes[sceneId]
  const baseProfile = tuning.quality.profiles[quality.profileId]
  // Diagnostic overrides let the smoke tool isolate one effect without a new profile.
  const profile = useMemo(() => {
    const off = (key: string) => params.get(key) === '0'
    if (!off('ao') && !off('bloom') && !off('shadows')) return baseProfile
    return { ...baseProfile, ao: baseProfile.ao && !off('ao'), bloom: baseProfile.bloom && !off('bloom'), shadows: baseProfile.shadows && !off('shadows') }
  }, [baseProfile, params])
  const periodId: PeriodId =
    periodMode.kind === 'fixed' ? periodMode.period : periodForHour(new Date().getHours(), tuning.lighting)
  const period = useMemo(() => resolvePeriod(scene, periodId), [scene, periodId])

  // Layout editing: persisted overrides applied over the generated layout by id.
  const layoutStore = useLayoutPersistence(sceneId, syntheticActors === 0, tuning.editor.saveDebounceMs)
  const [editing, setEditing] = useState(false)
  const [history, setHistory] = useState<OverrideHistory>(emptyHistory)
  const [selectedRoomId, setSelectedRoomId] = useState<string | null>(null)

  // Roster: the live team graph, or a synthetic one for goldens and demos.
  const liveRoster = useWorldRoster()
  const roster = useMemo(
    () => (syntheticActors > 0 ? { ...syntheticRoster(syntheticActors, SYNTHETIC_PER_TEAM, seed), ready: true } : liveRoster),
    [syntheticActors, seed, liveRoster],
  )
  // Trees keep clear of the ground point under the hero camera.
  const clearPoints = useMemo(() => {
    const { position } = poseToPosition(scene.camera.hero, [0, 0], 1)
    return [[position[0], position[2]] as const]
  }, [scene])
  const runtime = useWorldRuntime({
    seed,
    scene: sceneId,
    teams: roster.teams,
    agents: roster.agents,
    treeVariants: scene.props.trees.length,
    clearPoints,
    live: syntheticActors === 0,
    tuning,
    overrides: layoutStore.loaded ? layoutStore.overrides : undefined,
  })

  // When persisted overrides arrive, seed the history from them.
  useEffect(() => {
    if (!layoutStore.loaded) return
    setHistory({ current: layoutStore.overrides, past: [], future: [] })
    // eslint-disable-next-line react-hooks/exhaustive-deps -- seed once per load
  }, [layoutStore.loaded, sceneId])

  const applyHistory = useCallback(
    (next: OverrideHistory) => {
      setHistory(next)
      runtime.store.applyOverrides(next.current)
      layoutStore.save(next.current)
    },
    [runtime.store, layoutStore],
  )
  const moveRoom = useCallback(
    (roomId: string, position: readonly [number, number], commitMove: boolean) => {
      const next = upsertOverride(history.current, { placeId: roomId, position })
      if (commitMove) applyHistory(commit(history, next, tuning.editor.maxHistory))
      else runtime.store.applyOverrides(next)
    },
    [history, applyHistory, runtime.store, tuning.editor.maxHistory],
  )
  const selectedRoomLabel = selectedRoomId ? runtime.store.getState().places[selectedRoomId]?.label ?? null : null
  const actions = useMemo(() => createWorldActions((signals) => runtime.store.dispatch(signals)), [runtime.store])
  const simBounds = runtime.store.getState().bounds
  const bounds = useMemo<WorldBounds>(
    () => ({ width: simBounds.width, depth: simBounds.depth, center: simBounds.center, footprint: simBounds.footprint, outline: simBounds.outline }),
    [simBounds],
  )

  // Focus: frame the actor's rest bounds; follow keeps the target on it while it walks.
  useEffect(() => {
    const rig = cameraRig.current
    if (!rig) return
    if (!focusedId) {
      rig.follow(null)
      return
    }
    const actor = runtime.store.getState().actors[focusedId]
    if (!actor) return
    const radius = tuning.actor.bodyRadius
    const box = new Box3(
      new Vector3(actor.position[0] - radius, 0, actor.position[1] - radius),
      new Vector3(actor.position[0] + radius, radius * 2, actor.position[1] + radius),
    )
    rig.focus(box, true)
    rig.follow(
      following
        ? () => {
            const live = runtime.store.getState().actors[focusedId]
            return live ? [live.position[0], radius, live.position[1]] : [0, 0, 0]
          }
        : null,
    )
  }, [focusedId, following, runtime.store, tuning.actor.bodyRadius])

  const focusTeam = useCallback(
    (teamId: string) => {
      const state = runtime.store.getState()
      const room = Object.values(state.places).find((p) => p.kind === 'room' && p.teamId === teamId)
      if (!room || !cameraRig.current) return
      const [w, d] = room.size
      const box = new Box3(
        new Vector3(room.position[0] - w / 2, 0, room.position[1] - d / 2),
        new Vector3(room.position[0] + w / 2, tuning.layout.wallHeight, room.position[1] + d / 2),
      )
      setFocusedId(null)
      cameraRig.current.focus(box, true)
    },
    [runtime.store, tuning.layout.wallHeight],
  )

  const goHome = useCallback(() => {
    setFocusedId(null)
    setFollowing(false)
    cameraRig.current?.home(true)
  }, [])

  // Escape closes the card and returns home; typing targets are left alone.
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return
      const target = event.target
      if (target instanceof HTMLElement && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.tagName === 'SELECT')) return
      goHome()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [goHome])

  // Persisted preferences: adopt the saved choices once loaded, unless the URL pinned them.
  useEffect(() => {
    if (!preferences.loaded || syntheticActors > 0) return
    const saved = preferences.preferences
    if (!isSceneId(sceneParam)) setSceneId(saved.scene)
    if (!isQualityProfileId(profileParam)) setQuality({ auto: saved.qualityAuto, profileId: saved.qualityProfile })
    if (!isPeriodId(periodParam)) setPeriodMode(saved.periodMode === 'clock' ? { kind: 'clock' } : { kind: 'fixed', period: saved.periodMode })
    if (!showDiagnosticsParam) setShowDiagnostics(saved.showDiagnostics)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- adopt once when the saved preferences arrive
  }, [preferences.loaded])

  const onVerdict = useCallback((verdict: 'decline' | 'incline') => {
    setQuality((state) => applyVerdict(state, verdict))
  }, [])
  const getTarget = useCallback(
    (): [number, number, number] => cameraRig.current?.target() ?? [bounds.center[0], 0, bounds.center[1]],
    [bounds],
  )

  return (
    <div className="relative h-full w-full overflow-hidden" data-testid="world-view">
      {!twoD && (
      <WorldCanvas profile={profile} camera={tuning.camera}>
        <LightingRig scene={scene} period={period} lighting={tuning.lighting} profile={profile} bounds={bounds} fovDeg={tuning.camera.fov} />
        <CameraRig
          ref={cameraRig}
          scene={scene}
          camera={tuning.camera}
          bounds={bounds}
          intro={intro}
          reducedMotion={reducedMotion}
        />
        <WorldStoreContext.Provider value={runtime.store}>
          <Stage scene={scene} bounds={bounds} horizon={tuning.camera.far * HORIZON_SHARE} />
          <Places scene={scene} layout={tuning.layout} />
          <Props scene={scene} period={period} />
          <Trees scene={scene} />
          <Actors profile={profile} onSelect={editing ? undefined : setFocusedId} onHover={setHoveredId} />
          {editing && (
            <RoomHandles
              editor={tuning.editor}
              selectedRoomId={selectedRoomId}
              onSelectRoom={setSelectedRoomId}
              onMove={moveRoom}
              onDragging={(dragging) => cameraRig.current?.setEnabled(!dragging)}
            />
          )}
          <Labels labels={tuning.labels} profile={profile} fovDeg={tuning.camera.fov} focusedId={focusedId} hoveredId={hoveredId} />
          <SceneEnvironment scene={scene} profile={profile} period={period} bounds={bounds} />
        </WorldStoreContext.Provider>
        <PostChain profile={profile} />
        <QualityGovernor auto={quality.auto} profile={profile} quality={tuning.quality} onVerdict={onVerdict} />
        <DiagnosticsProbe
          scene={sceneId}
          profileId={quality.profileId}
          profile={profile}
          auto={quality.auto}
          period={periodId}
          getTarget={getTarget}
          bounds={bounds}
        />
      </WorldCanvas>
      )}
      <WorldHud
        store={runtime.store}
        actions={actions}
        feed={runtime.feed}
        focusedId={focusedId}
        onFocus={setFocusedId}
        onFocusTeam={focusTeam}
        onHome={goHome}
        following={following}
        onFollowChange={setFollowing}
        filters={filters}
        onFiltersChange={setFilters}
        summaryFilter={summaryFilter}
        onSummaryFilterChange={setSummaryFilter}
        highlightedTeamId={highlightedTeamId}
        onHighlightTeam={setHighlightedTeamId}
        twoD={twoD}
        onTwoDChange={(next) => {
          setTwoDChoice(next)
          preferences.update({ twoDMode: next })
        }}
        tickerLimit={TICKER_LIMIT}
      />
      {showDiagnostics && !twoD && <DiagnosticsOverlay />}
      {!twoD && (
        <div className="pointer-events-none absolute right-3 top-14 z-20">
          <EditorToolbar
            editing={editing}
            onEditingChange={(next) => {
              setEditing(next)
              setSelectedRoomId(null)
              if (next) {
                setFocusedId(null)
                cameraRig.current?.setPose({ ...scene.camera.hero, polarDeg: tuning.editor.aerialPolarDeg }, true)
              } else {
                cameraRig.current?.home(true)
              }
            }}
            canUndo={canUndo(history)}
            canRedo={canRedo(history)}
            onUndo={() => applyHistory(undo(history))}
            onRedo={() => applyHistory(redo(history))}
            onReset={() => applyHistory(commit(history, [], tuning.editor.maxHistory))}
            selectedRoomLabel={selectedRoomLabel}
            onRemoveSelected={() => {
              if (!selectedRoomId) return
              applyHistory(commit(history, upsertOverride(history.current, { placeId: selectedRoomId, removed: true }), tuning.editor.maxHistory))
              setSelectedRoomId(null)
            }}
            overrideCount={history.current.length}
            saving={false}
          />
        </div>
      )}
      <ViewOverlay
        onOpenMobileSidebar={props.onOpenMobileSidebar}
        pendingWorkCount={props.pendingWorkCount}
        runningAgentCount={props.runningAgentCount}
        homeView={props.homeView}
        onHomeViewChange={props.onHomeViewChange}
        leftPanelContent={props.leftPanelContent}
        settingsTitle="World Settings"
        settingsContent={
          <WorldSettingsContent
            sceneId={sceneId}
            onSceneChange={(next) => {
              setSceneId(next)
              preferences.update({ scene: next })
            }}
            quality={quality}
            onPickProfile={(id: QualityProfileId) => {
              setQuality(pickProfile(id))
              preferences.update({ qualityProfile: id, qualityAuto: false })
            }}
            onAutoChange={(auto) => {
              setQuality((s) => setAuto(s, auto))
              preferences.update({ qualityAuto: auto })
            }}
            periodMode={periodMode}
            onPeriodModeChange={(mode) => {
              setPeriodMode(mode)
              preferences.update({ periodMode: mode.kind === 'clock' ? 'clock' : mode.period })
            }}
            showDiagnostics={showDiagnostics}
            onShowDiagnosticsChange={(show) => {
              setShowDiagnostics(show)
              preferences.update({ showDiagnostics: show })
            }}
            onCameraHome={goHome}
            levers={import.meta.env.DEV ? { tuning, override: tuningOverride, onChange: setTuningOverride, onReset: () => setTuningOverride({}) } : undefined}
          />
        }
        helpTitle="World"
        helpContent={<WorldHelpContent />}
      />
    </div>
  )
}
