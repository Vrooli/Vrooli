/**
 * /world route component. The only module that composes scene and hud.
 *
 * URL levers (all optional, used by the smoke tool and deep links):
 *   ?scene=park|office   ?profile=low|medium|high|ultra (manual, disables auto)
 *   ?period=dawn|day|dusk|night   ?intro=0 (skip the dolly)   ?diag=1 (overlay)
 *   ?seed=<int> (sim seed)   ?actors=<n> (synthetic roster, no feed: goldens and demos)
 *   ?ao=0 ?bloom=0 ?shadows=0 ?dpr=<0.5..3> ?msaa=<0..8>
 *     (diagnostic overrides of the active profile's rendering cost)
 *   ?view=3d|2d (deep-link intent: outranks stored preferences and narrow screens)
 *   ?forceWebglFail=1 (permanent fallback/retry test lever)
 */
import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { Box3, Vector3 } from 'three'
import { useSearchParams } from 'react-router-dom'
import { ViewOverlay } from '@/components/shared/ViewOverlay'
import { useMediaQuery } from '@/hooks/useMediaQuery'
import {
  biomeSets,
  isPeriodId,
  isQualityProfileId,
  isSceneId,
  periodForHour,
  resolvePeriod,
  resolveTerrain,
  scenes,
  tuning as shippedTuning,
  withTuningOverride,
  type PeriodId,
  type QualityProfileId,
  type QualityState,
  type SceneId,
  type TuningOverride,
  type WeatherId,
} from './config'
import {
  CameraRig,
  DiagnosticsOverlay,
  DiagnosticsProbe,
  FrameDriver,
  LightingRig,
  PostChain,
  QualityGovernor,
  WorldCanvas,
  applyWeather,
  applyVerdict,
  chooseInitialProfile,
  pickProfile,
  probeWebGL,
  resolveTwoD,
  retryWebGL,
  readDiagnostics,
  setAuto,
  updateDiagnostics,
  type CameraRigHandle,
  type QualityVerdictRecord,
  type WorldBounds,
} from './engine'
import { ActorPoseProvider, Actors, Labels, Places, Props, RoomHandles, SceneEnvironment, Terrain, Vegetation, Water, Weather, WorldStoreContext } from './scene'
import { EMPTY_FILTERS, EditorToolbar, WorldHelpContent, WorldHud, WorldSettingsContent, type FilterState, type SummaryFilter } from './hud'
import { createWorldActions, syntheticRoster, useLayoutPersistence, useWorldPreferences, useWorldRoster, useWorldRuntime } from './data'
import { canRedo, canUndo, commit, emptyHistory, heightAt, redo, terrainDigest, undo, upsertOverride, type OverrideHistory } from './sim'
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
const SYNTHETIC_PER_TEAM = 5
const SYNTHETIC_MAX_TEAMS = 5
const SYNTHETIC_SCALE_MAX_TEAMS = 4
const TICKER_LIMIT = 12

function parseInt10(value: string | null, fallback: number): number {
  const n = value === null ? NaN : Number.parseInt(value, 10)
  return Number.isFinite(n) && n >= 0 ? n : fallback
}

function clampedNumber(value: string | null, fallback: number, min: number, max: number): number {
  const parsed = value === null ? NaN : Number(value)
  return Number.isFinite(parsed) ? Math.max(min, Math.min(max, parsed)) : fallback
}

export function WorldView(props: WorldViewProps) {
  const [params] = useSearchParams()
  const sceneParam = params.get('scene')
  const profileParam = params.get('profile')
  const periodParam = params.get('period')
  const reducedMotion = useMediaQuery('(prefers-reduced-motion: reduce)')
  const intro = params.get('intro') !== '0'
  const showDiagnosticsParam = params.get('diag') === '1'
  const capture = showDiagnosticsParam || params.get('capture') === '1'

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
  const [qualityNotice, setQualityNotice] = useState<string | null>(null)
  const calibrated = useRef(false)
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
  const forceWebglFail = params.get('forceWebglFail') === '1'
  const [webgl, setWebgl] = useState(() => probeWebGL(forceWebglFail))
  const preferences = useWorldPreferences(undefined, syntheticActors === 0)
  const [twoDChoice, setTwoDChoice] = useState<boolean | null>(null)
  // Dev levers: an override merged over the shipped tuning, re-validated on every edit.
  const [tuningOverride, setTuningOverride] = useState<TuningOverride>({})
  const tuning = useMemo(() => (Object.keys(tuningOverride).length > 0 ? withTuningOverride(tuningOverride, shippedTuning) : shippedTuning), [tuningOverride])
  const viewParam = params.get('view')
  const requestedTwoD = viewParam === '2d' ? true : viewParam === '3d' ? false : null
  const twoD = resolveTwoD({
    webglAvailable: webgl.ok,
    userChoice: twoDChoice,
    requestedTwoD,
    storedTwoD: preferences.preferences.twoDMode,
    narrow,
  })
  const askedFor3D = requestedTwoD === false || twoDChoice === false

  useEffect(() => updateDiagnostics({ webgl }), [webgl])

  const retry3D = useCallback(() => {
    const result = retryWebGL(forceWebglFail)
    setWebgl(result)
    if (result.ok) setTwoDChoice(false)
  }, [forceWebglFail])

  const scene = scenes[sceneId]
  const sceneBiomeSet = biomeSets[scene.biomeSet]
  const baseProfile = tuning.quality.profiles[quality.profileId]
  // Diagnostic overrides let the smoke tool isolate one effect without a new profile.
  const profile = useMemo(() => {
    const off = (key: string) => params.get(key) === '0'
    const dpr = clampedNumber(params.get('dpr'), baseProfile.dpr, 0.5, 3)
    const msaa = Math.round(clampedNumber(params.get('msaa'), baseProfile.msaa, 0, 8))
    if (!off('ao') && !off('bloom') && !off('shadows') && dpr === baseProfile.dpr && msaa === baseProfile.msaa) return baseProfile
    return {
      ...baseProfile,
      dpr,
      msaa,
      ao: baseProfile.ao && !off('ao'),
      bloom: baseProfile.bloom && !off('bloom'),
      shadows: baseProfile.shadows && !off('shadows'),
    }
  }, [baseProfile, params])
  const periodId: PeriodId =
    periodMode.kind === 'fixed' ? periodMode.period : periodForHour(new Date().getHours(), tuning.lighting)
  const basePeriod = useMemo(() => resolvePeriod(scene, periodId), [scene, periodId])

  // Layout editing: persisted overrides applied over the generated layout by id.
  const layoutStore = useLayoutPersistence(sceneId, syntheticActors === 0, tuning.editor.saveDebounceMs)
  const [editing, setEditing] = useState(false)
  const [history, setHistory] = useState<OverrideHistory>(emptyHistory)
  const [selectedRoomId, setSelectedRoomId] = useState<string | null>(null)

  // Roster: the live team graph, or a synthetic one for goldens and demos.
  const liveRoster = useWorldRoster()
  const roster = useMemo(
    () => {
      const maxTeams = syntheticActors > 25 ? SYNTHETIC_SCALE_MAX_TEAMS : SYNTHETIC_MAX_TEAMS
      const perTeam = Math.max(SYNTHETIC_PER_TEAM, Math.ceil(syntheticActors / maxTeams))
      return syntheticActors > 0 ? { ...syntheticRoster(syntheticActors, perTeam, seed), ready: true } : liveRoster
    },
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
    treeVariants: new Set(sceneBiomeSet.biomes.flatMap((biome) => Object.keys(biome.vegetation))).size,
    clearPoints,
    live: syntheticActors === 0,
    step: syntheticActors === 0,
    tuning,
    overrides: layoutStore.loaded ? layoutStore.overrides : undefined,
  })
  const seedDigest = useMemo(() => terrainDigest(runtime.store.getState()), [runtime.store])
  const weatherParam = params.get('weather')
  const pinnedWeather: WeatherId | null = weatherParam === 'clear' || weatherParam === 'cloudy' || weatherParam === 'rain' || weatherParam === 'snow' ? weatherParam : null
  const pressureRaw = Number(params.get('pressure'))
  const pinnedPressure = Number.isFinite(pressureRaw) && params.has('pressure') ? Math.max(0, Math.min(1, pressureRaw)) : null
  const weatherId = pinnedWeather ?? (pinnedPressure !== null && pinnedPressure >= 0.75 ? 'rain' : runtime.store.getState().weather.state)
  const terrainTuning = useMemo(() => resolveTerrain(scene, tuning), [scene, tuning])
  const weatherPreset = tuning.weather.states[weatherId]
  const period = useMemo(() => applyWeather(basePeriod, weatherId, tuning.weather), [basePeriod, weatherId, tuning.weather])
  useEffect(() => updateDiagnostics({ weather: weatherId, weatherPressure: pinnedPressure ?? runtime.store.getState().weather.pressure }), [pinnedPressure, runtime.store, weatherId])

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
    const ground = heightAt(runtime.store.getState().terrain, actor.position[0], actor.position[1])
    const box = new Box3(
      new Vector3(actor.position[0] - radius, ground, actor.position[1] - radius),
      new Vector3(actor.position[0] + radius, ground + radius * 2, actor.position[1] + radius),
    )
    rig.focus(box, true)
    rig.follow(
      following
        ? () => {
            const live = runtime.store.getState().actors[focusedId]
            return live ? [live.position[0], heightAt(runtime.store.getState().terrain, live.position[0], live.position[1]) + radius, live.position[1]] : [0, 0, 0]
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

  const onVerdict = useCallback((record: QualityVerdictRecord) => {
    const history = [...readDiagnostics().qualityHistory, record].slice(-12)
    updateDiagnostics({ qualityHistory: history })
    if (!calibrated.current && record.verdict === 'incline' && quality.auto && !isQualityProfileId(profileParam) && preferences.preferences.qualityProfile === shippedTuning.quality.defaultProfile) {
      calibrated.current = true
      const selected = chooseInitialProfile(record.measuredFps, record.boundFps / tuning.quality.recoverRatio)
      setQuality({ auto: true, profileId: selected })
      preferences.update({ qualityProfile: selected, qualityAuto: true })
      if (selected !== record.from) {
        setQualityNotice(`Quality calibrated to ${selected}: ${record.reason}`)
        window.setTimeout(() => setQualityNotice(null), 5000)
      }
      return
    }
    if (record.to === record.from) return
    setQuality((state) => applyVerdict(state, record.verdict))
    setQualityNotice(`Quality adjusted to ${record.to}: ${record.reason}`)
    window.setTimeout(() => setQualityNotice(null), 5000)
    preferences.update({ qualityProfile: record.to, qualityAuto: true })
  }, [preferences, profileParam, quality.auto, tuning.quality.recoverRatio])
  const getTarget = useCallback(
    (): [number, number, number] => cameraRig.current?.target() ?? [bounds.center[0], 0, bounds.center[1]],
    [bounds],
  )

  return (
    <div className="relative h-full w-full overflow-hidden" data-testid="world-view">
      {!twoD && (
      <WorldCanvas profile={profile} camera={tuning.camera} capture={capture}>
        <CameraRig
          ref={cameraRig}
          scene={scene}
          camera={tuning.camera}
          bounds={bounds}
          intro={intro}
          reducedMotion={reducedMotion}
        />
        <WorldStoreContext.Provider value={runtime.store}>
          <FrameDriver store={runtime.store} weatherActive={weatherId === 'rain' || weatherId === 'snow'} diagnosticsOpen={showDiagnostics} continuous={params.get('capture') === '1'} intro={intro} settleSeconds={tuning.camera.smoothTime} />
          <LightingRig scene={scene} period={period} lighting={tuning.lighting} profile={profile} bounds={bounds} fovDeg={tuning.camera.fov} store={runtime.store} />
          <Terrain scene={scene} tuning={terrainTuning} profile={profile} weather={weatherPreset} />
          <Water scene={scene} tuning={terrainTuning} profile={profile} />
          <Places scene={scene} layout={tuning.layout} />
          <Props scene={scene} period={period} tuning={tuning.layout} />
          <Vegetation scene={scene} profile={profile} />
          <ActorPoseProvider>
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
          </ActorPoseProvider>
          <SceneEnvironment scene={scene} profile={profile} period={period} bounds={bounds} weather={weatherPreset} altitude={tuning.weather.cloudAltitude} />
          <Weather id={weatherId} preset={weatherPreset} tuning={tuning.weather} profile={profile} getTarget={getTarget} />
        </WorldStoreContext.Provider>
        <PostChain profile={profile} diagnosticsEnabled={showDiagnostics} />
        <QualityGovernor auto={quality.auto} profile={profile} profileId={quality.profileId} quality={tuning.quality} onVerdict={onVerdict} />
        <DiagnosticsProbe
          scene={sceneId}
          profileId={quality.profileId}
          profile={profile}
          auto={quality.auto}
          period={periodId}
          getTarget={getTarget}
          bounds={bounds}
          measureEnabled={showDiagnostics}
        />
      </WorldCanvas>
      )}
      {!webgl.ok && askedFor3D && (
        <div
          data-testid="world-webgl-banner"
          className="absolute left-1/2 top-4 z-40 w-[min(92vw,36rem)] -translate-x-1/2 rounded-md border border-amber-500/60 bg-background/95 px-4 py-3 text-sm text-foreground shadow-lg"
          role="alert"
        >
          <p className="font-medium">3D world unavailable: {webgl.reason}</p>
          <p className="mt-1 text-muted-foreground">{webgl.detail}</p>
          <button
            type="button"
            data-testid="world-webgl-retry"
            className="mt-3 rounded-md bg-primary px-3 py-1.5 font-medium text-primary-foreground"
            onClick={retry3D}
          >
            Retry 3D
          </button>
        </div>
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
        weather={pinnedWeather || pinnedPressure !== null ? { state: weatherId, pressure: pinnedPressure ?? runtime.store.getState().weather.pressure } : undefined}
      />
      {showDiagnostics && !twoD && <DiagnosticsOverlay seed={seed} seedDigest={seedDigest} />}
      {qualityNotice && (
        <div className="pointer-events-auto absolute right-3 top-3 z-40 max-w-sm rounded-md border border-border bg-background/95 px-3 py-2 text-sm shadow-lg" role="status" data-testid="world-quality-notice">
          <span>{qualityNotice}</span>
          <button type="button" className="ml-3 text-muted-foreground" onClick={() => setQualityNotice(null)} aria-label="Dismiss quality notice">×</button>
        </div>
      )}
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
