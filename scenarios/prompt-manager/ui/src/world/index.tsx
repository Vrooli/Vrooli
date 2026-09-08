import { numericOverrides, parseNumericOverride, choiceSettings, parseChoiceSetting, integerSettings, parseIntegerSetting } from './config/settings'
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
 *   ?workbench=1 (opt-in session-only development controls in the built UI)
 */
import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import type { GeneratedWorld } from './sim/model'
import { presentationManifest } from './data/presentationManifest'
import { preloadProps, propRecord } from './engine/assets'
import { TerrainMeshCache } from './data/terrainMesh'
import { Box3, Color, Vector3 } from 'three'
import { useSearchParams } from 'react-router-dom'
import { ViewOverlay } from '@/components/shared/ViewOverlay'
import { useMediaQuery } from '@/hooks/useMediaQuery'
import {
  biomeSets,
  isPeriodId,
  isQualityProfileId,
  isSceneId,
  resolvePeriod,
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
  bindCameraHome,
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
import { useLightingSample } from './engine/lighting/clock'
import { continuousPeriod } from './engine/lighting/interpolate'
import { CelestialSky } from './scene/CelestialSky'
import { GridOverlay } from './scene/GridOverlay'
import { CameraCollisionOverlay } from './scene/CameraCollisionOverlay'
import { SkyEvents } from './scene/ambient/SkyEvents'
import { Fireflies } from './scene/ambient/Fireflies'
import { Birds, Butterflies } from './scene/ambient/Butterflies'
import { Rabbits } from './scene/ambient/Rabbits'
import { Fish } from './scene/ambient/Fish'
import { prepareWildlifePreview } from './data/wildlifePreview'
import type { WildlifePreviewId } from './config/ambient'
import { AnimationLeases } from './engine/animationLeases'
import { WorldClock } from './config/clock'
import type { SkyEvent } from './sim/ambient/schedule'
import { biomeLegend, habitatLegend } from './scene/classificationOverlay'
import { createDiagnosticRecipe, parseRecipeFile, recipeLocation, RECIPE_SESSION_KEY, type DiagnosticRecipe } from './data/diagnosticRecipe'
import { CameraToolbar } from './hud/CameraToolbar'
import { createNavigationTelemetry, useNavigationPreferences } from './hud/navigationState'
import type { NavigationTool } from './config/navigation'
import { isWalkable } from './sim/nav/grid'
import { EMPTY_FILTERS, EditorToolbar, WorldHelpContent, WorldHud, WorldSettingsContent, type FilterState, type SummaryFilter } from './hud'
import { createWorldActions, syntheticRoster, useLayoutPersistence, useWorldPreferences, useWorldRoster, useWorldRuntime } from './data'
import { canRedo, canUndo, commit, emptyHistory, heightAt, maximumHeightInRegion, redo, terrainDigest, undo, upsertOverride, type OverrideHistory } from './sim'
import { terrainForBounds } from './sim/layout/centre'
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

/** The ground disc reaches this share of the far plane; fog hides its edge before the clip does. */
const SYNTHETIC_PER_TEAM = 5
const SYNTHETIC_MAX_TEAMS = 5
const SYNTHETIC_SCALE_MAX_TEAMS = 4
const TICKER_LIMIT = 12

export function WorldView(props: WorldViewProps) {
  const terrainMeshCache = useMemo(() => new TerrainMeshCache({ onStats: terrainMeshCache => updateDiagnostics({ terrainMeshCache }) }), [])
  const [params, setParams] = useSearchParams()
  const [recipeSession] = useState<{ recipe?: DiagnosticRecipe; error?: string }>(() => {
    if (params.get('recipe') !== 'session') return {}
    try {
      const text = sessionStorage.getItem(RECIPE_SESSION_KEY)
      if (!text) throw new Error('The imported recipe is missing from this tab. Import the file again.')
      return { recipe: parseRecipeFile(text) }
    } catch (error) { return { error: error instanceof Error ? error.message : 'Could not restore the imported recipe.' } }
  })
  const importedRecipe = recipeSession.recipe
  const sceneSetting = parseChoiceSetting(choiceSettings.scene, params.get('scene'))
  const profileSetting = parseChoiceSetting(choiceSettings.profile, params.get('profile'))
  const periodSetting = parseChoiceSetting(choiceSettings.period, params.get('period'))
  const weatherSetting = parseChoiceSetting(choiceSettings.weather, params.get('weather'))
  const sceneParam = params.has('scene') ? sceneSetting.value : null
  const profileParam = params.has('profile') ? profileSetting.value : null
  const periodParam = params.has('period') ? periodSetting.value : null
  const reducedMotion = useMediaQuery('(prefers-reduced-motion: reduce)')
  const intro = params.get('intro') !== '0'
  const showDiagnosticsParam = params.get('diag') === '1'
  const capture = params.get('capture') === '1'
  const workbench = import.meta.env.DEV || params.get('workbench') === '1'

  const [sceneId, setSceneId] = useState<SceneId>(() => (isSceneId(sceneParam) ? sceneParam : 'park'))
  const [quality, setQuality] = useState<QualityState>(() =>
    isQualityProfileId(profileParam)
      ? { auto: false, profileId: profileParam }
      : { auto: true, profileId: shippedTuning.quality.defaultProfile },
  )
  const [periodMode, setPeriodMode] = useState<PeriodMode>(() =>
    importedRecipe?.view.lightingMode === 'clock' ? { kind: 'clock' } : isPeriodId(periodParam) ? { kind: 'fixed', period: periodParam } : { kind: 'clock' },
  )
  const [showDiagnostics, setShowDiagnostics] = useState(showDiagnosticsParam)
  const [showNavigationOverlay, setShowNavigationOverlay] = useState(importedRecipe?.view.navigationOverlay ?? false)
  const [showBiomeOverlay, setShowBiomeOverlay] = useState(importedRecipe?.view.biomeOverlay ?? false)
  const [showHabitatOverlay, setShowHabitatOverlay] = useState(importedRecipe?.view.habitatOverlay ?? false)
  const [showCameraCollisionOverlay, setShowCameraCollisionOverlay] = useState(importedRecipe?.view.cameraCollisionOverlay ?? false)
  const [cameraCollisionStatus, setCameraCollisionStatus] = useState('Preparing camera outlines…')
  const [ambientEnabled, setAmbientEnabled] = useState(importedRecipe?.view.ambientLife ?? true)
  const [skyPreview, setSkyPreview] = useState<SkyEvent | undefined>()
  const [wildlifePreviewStatus, setWildlifePreviewStatus] = useState('Choose an animal or event to inspect.')
  const wildlifePreviewRequest = useRef<AbortController | null>(null)
  useEffect(() => () => wildlifePreviewRequest.current?.abort(), [])
  const [skyStatus, setSkyStatus] = useState('No active sky events')
  const animationLeases = useMemo(() => new AnimationLeases(), [])
  const [worldClock] = useState(() => {
    const clock = new WorldClock(Date.now, importedRecipe?.view.clock?.timeZone)
    if (importedRecipe?.view.clock) clock.fix(importedRecipe.view.clock.utcMilliseconds)
    return clock
  })
  const [navigationOverlayStatus, setNavigationOverlayStatus] = useState('Preparing navigation markers…')
  const [qualityNotice, setQualityNotice] = useState<string | null>(null)
  const calibrated = useRef(false)
  const cameraRig = useRef<CameraRigHandle | null>(null)
  const [navigation, setNavigation] = useNavigationPreferences()
  const navigationTelemetry = useMemo(createNavigationTelemetry, [])
  const [navigationTool, setNavigationTool] = useState<NavigationTool>('orbit')
  const seedSetting = parseIntegerSetting(integerSettings.seed, params.get('seed'))
  const actorSetting = parseIntegerSetting(integerSettings.actors, params.get('actors'))
  const seed = seedSetting.value
  const syntheticActors = actorSetting.value
  const urlSettingErrors = [
    ...([[choiceSettings.scene, sceneSetting], [choiceSettings.profile, profileSetting], [choiceSettings.period, periodSetting], [choiceSettings.weather, weatherSetting]] as const).map(([descriptor, result]) => result.error && `${descriptor.label}: ${result.error} Using ${result.value}.`),
    seedSetting.error && `${integerSettings.seed.label}: ${seedSetting.error} Using ${seed}.`,
    actorSetting.error && `${integerSettings.actors.label}: ${actorSetting.error} Using ${syntheticActors}.`,
  ].filter(Boolean)
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
  const [tuningState, setTuningState] = useState(() => {
    const imported: TuningOverride = importedRecipe?.tuning ?? {}
    const override: TuningOverride = importedRecipe?.view.zoomTarget ? { ...imported, camera: { ...imported.camera, dollyToCursor: importedRecipe.view.zoomTarget === 'cursor' } } : imported
    return { override, value: withTuningOverride(override, shippedTuning) }
  })
  const tuningOverride = tuningState.override, tuning = tuningState.value
  const setTuningOverride = useCallback((override: TuningOverride) => {
    setTuningState(previous => ({ override, value: withTuningOverride(override, shippedTuning, previous.value) }))
  }, [])
  const zoomTarget = tuning.camera.dollyToCursor ? 'cursor' : 'center'
  const cameraTuning = tuning.camera
  const setZoomTarget = useCallback((target: 'cursor' | 'center') => {
    setTuningState(previous => {
      const override = { ...previous.override, camera: { ...previous.override.camera, dollyToCursor: target === 'cursor' } }
      return { override, value: withTuningOverride(override, shippedTuning, previous.value) }
    })
  }, [])
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

  const requestedScene = scenes[sceneId]
  const sceneBiomeSet = biomeSets[requestedScene.biomeSet]
  const baseProfile = tuning.quality.profiles[quality.profileId]
  const dprSetting = parseNumericOverride(numericOverrides.dpr, params.get('dpr'), baseProfile.dpr)
  const msaaSetting = parseNumericOverride(numericOverrides.msaa, params.get('msaa'), baseProfile.msaa)
  const lampSetting = parseNumericOverride(numericOverrides.lampLights, params.get('lampLights'), baseProfile.lampLights)
  const pressureSetting = parseNumericOverride(numericOverrides.pressure, params.get('pressure'), null)
  for (const [descriptor, result] of [[numericOverrides.dpr, dprSetting], [numericOverrides.msaa, msaaSetting], [numericOverrides.lampLights, lampSetting], [numericOverrides.pressure, pressureSetting]] as const) {
    if (result.error) urlSettingErrors.push(`${descriptor.label}: ${result.error} Using ${result.value ?? 'Automatic'}.`)
  }
  // Diagnostic overrides let the smoke tool isolate one effect without a new profile.
  const profile = useMemo(() => {
    const off = (key: string) => params.get(key) === '0'
    const dpr = dprSetting.value
    const msaa = msaaSetting.value
    const lampLights = lampSetting.value
    if (!off('ao') && !off('bloom') && !off('shadows') && dpr === baseProfile.dpr && msaa === baseProfile.msaa && lampLights === baseProfile.lampLights) return baseProfile
    return {
      ...baseProfile,
      dpr,
      msaa,
      lampLights,
      ao: baseProfile.ao && !off('ao'),
      bloom: baseProfile.bloom && !off('bloom'),
      shadows: baseProfile.shadows && !off('shadows'),
    }
  }, [baseProfile, params, dprSetting.value, msaaSetting.value, lampSetting.value])
  const { periodId, localMinutes } = useLightingSample(periodMode, tuning.lighting, worldClock)

  // Layout editing: persisted overrides applied over the generated layout by id.
  const layoutStore = useLayoutPersistence(sceneId, syntheticActors === 0, tuning.editor.saveDebounceMs, undefined,
    importedRecipe ? { scene: importedRecipe.scene, overrides: importedRecipe.layout } : undefined)
  const [editing, setEditing] = useState(false)
  const [history, setHistory] = useState<OverrideHistory>(emptyHistory)
  const [selectedRoomId, setSelectedRoomId] = useState<string | null>(null)

  // Roster: the live team graph, or a synthetic one for goldens and demos.
  const liveRoster = useWorldRoster()
  const roster = useMemo(
    () => {
      const maxTeams = syntheticActors > 25 ? SYNTHETIC_SCALE_MAX_TEAMS : SYNTHETIC_MAX_TEAMS
      const perTeam = importedRecipe?.roster.perTeam ?? Math.max(SYNTHETIC_PER_TEAM, Math.ceil(syntheticActors / maxTeams))
      return syntheticActors > 0 ? { ...syntheticRoster(syntheticActors, perTeam, seed), ready: true } : liveRoster
    },
    [syntheticActors, seed, liveRoster, importedRecipe],
  )
  // Trees keep clear of the ground point under the hero camera.
  const clearPoints = useMemo(() => {
    if (importedRecipe && requestedScene.id === importedRecipe.scene) return importedRecipe.clearPoints
    const { position } = poseToPosition(requestedScene.camera.hero, [0, 0], 1)
    return [[position[0], position[2]] as const]
  }, [requestedScene, importedRecipe])
  const pinnedWeather: WeatherId | null = weatherSetting.value === 'auto' ? null : weatherSetting.value
  const pinnedPressure = pressureSetting.value
  const prepareAssets = useCallback((scene: SceneId, signal: AbortSignal) => {
    const records = presentationManifest({ scene, decor: [] }).map(asset => {
      const record = propRecord(asset.assetSet, asset.id)
      if (!record) throw new Error(`Missing core world prop: ${asset.assetSet}/${asset.id}`)
      return record
    })
    return preloadProps(records, signal)
  }, [])
  const prepareDestination = useCallback(async (world: GeneratedWorld, signal: AbortSignal, liveWeather: WeatherId) => {
    const destination = scenes[world.scene]
    if (destination.environment === 'indoor' && !destination.centre) return
    const weatherId = pinnedWeather ?? (pinnedPressure !== null && pinnedPressure >= 0.75 ? 'rain' : liveWeather)
    const weather = tuning.weather.states[weatherId]
    const tint = new Color(weather.terrainTint)
    const shadow = new Color(weather.terrainShadowTint)
    const records = presentationManifest(world).map(asset => {
      const record = propRecord(asset.assetSet, asset.id)
      if (!record) throw new Error(`Missing destination ${asset.role} prop: ${asset.assetSet}/${asset.id}`)
      return record
    })
    const propsReady = preloadProps(records, signal)
    await Promise.all([propsReady, terrainMeshCache.prepare({
      field: world.terrain, surface: world.terrainSurface,
      innerRadiusSetting: terrainForBounds(destination, tuning.terrain, world.bounds).base().innerRadius,
      profile: { terrainInnerRadius: profile.terrainInnerRadius, terrainCellScale: profile.terrainCellScale },
      visual: tuning.visual.terrain,
      weather: { terrainTintMix: weather.terrainTintMix, terrainTintVariation: weather.terrainTintVariation },
      tint: [tint.r, tint.g, tint.b], shadowTint: [shadow.r, shadow.g, shadow.b],
    }, { signal })])
    signal.throwIfAborted()
  }, [pinnedWeather, pinnedPressure, profile.terrainInnerRadius, profile.terrainCellScale, tuning, terrainMeshCache])
  const runtime = useWorldRuntime({
    prepareDestination,
    prepareAssets,
    ready: roster.ready && layoutStore.loaded,
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
  const scene = scenes[runtime.store?.getState().scene ?? sceneId]
  const basePeriod = useMemo(() => periodMode.kind === 'fixed' ? resolvePeriod(scene, periodId, tuning) : continuousPeriod(scene, localMinutes, tuning), [scene, periodId, periodMode.kind, localMinutes, tuning])
  // Store identity survives adoption; the commit epoch invalidates spatial memoization.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const seedDigest = useMemo(() => runtime.store ? terrainDigest(runtime.store.getState()) : '', [runtime.store, runtime.epoch])
  const weatherId = pinnedWeather ?? (pinnedPressure !== null && pinnedPressure >= 0.75 ? 'rain' : (runtime.store?.getState().weather.state ?? 'clear'))
  // eslint-disable-next-line react-hooks/exhaustive-deps -- adopted bounds change without replacing the store
  const terrainTuning = useMemo(() => runtime.store ? terrainForBounds(scene, tuning.terrain, runtime.store.getState().bounds) : null, [scene, tuning.terrain, runtime.store, runtime.epoch])
  const weatherPreset = tuning.weather.states[weatherId]
  const period = useMemo(() => applyWeather(basePeriod, weatherId, tuning.weather), [basePeriod, weatherId, tuning.weather])
  useEffect(() => updateDiagnostics({ weather: weatherId, weatherPressure: pinnedPressure ?? (runtime.store?.getState().weather.pressure ?? 0) }), [pinnedPressure, runtime.store, weatherId])

  // When persisted overrides arrive, seed the history from them.
  useEffect(() => {
    if (!layoutStore.loaded) return
    setHistory({ current: layoutStore.overrides, past: [], future: [] })
    // eslint-disable-next-line react-hooks/exhaustive-deps -- seed once per load
  }, [layoutStore.loaded, sceneId])

  const applyHistory = useCallback(
    (next: OverrideHistory) => {
      if (runtime.preparing) return
      setHistory(next)
      layoutStore.save(next.current)
    },
    [runtime.preparing, layoutStore],
  )
  const moveRoom = useCallback(
    (roomId: string, position: readonly [number, number]) => {
      if (runtime.preparing) return
      const next = upsertOverride(history.current, { placeId: roomId, position })
      applyHistory(commit(history, next, tuning.editor.maxHistory))
    },
    [history, applyHistory, runtime.preparing, tuning.editor.maxHistory],
  )
  const selectedRoomLabel = selectedRoomId ? runtime.store?.getState().places[selectedRoomId]?.label ?? null : null
  const actions = useMemo(() => createWorldActions((signals) => runtime.store?.dispatch(signals)), [runtime.store])
  const simBounds = runtime.store?.getState().bounds
  const bounds = useMemo<WorldBounds>(
    () => simBounds ?? { width: 0, depth: 0, center: [0, 0], footprint: { width: 0, depth: 0, center: [0, 0] }, outline: [] },
    [simBounds],
  )

  const groundCeiling = useCallback((minX: number, minZ: number, maxX: number, maxZ: number) => {
    const terrain = runtime.store?.getState().terrain
    return terrain ? maximumHeightInRegion(terrain, minX, minZ, maxX, maxZ) : 0
  }, [runtime.store])

  const walkSurface = useCallback((x: number, z: number, radius: number) => {
    const state = runtime.store?.getState()
    if (!state) return { height: 0, walkable: false }
    let walkable = isWalkable(state.nav, [x, z])
    const height = heightAt(state.terrain, x, z)
    for (let i = 0; i < 8 && walkable; i++) {
      const angle = i * Math.PI / 4
      const sx = x + Math.cos(angle) * radius, sz = z + Math.sin(angle) * radius
      walkable = isWalkable(state.nav, [sx, sz]) && Math.abs(heightAt(state.terrain, sx, sz) - height) <= Math.tan(.45) * radius + .05
    }
    return { height, walkable }
  }, [runtime.store])

  const framedSelection = useRef<{ instance: symbol; actorId: string } | null>(null)
  // Frame once per selection and mounted rig. Follow and appearance updates must
  // not overwrite the distance and angles chosen after that framing command.
  const applyFocus = useCallback(() => {
    const rig = cameraRig.current
    if (!rig || !runtime.store) return
    if (!focusedId) {
      framedSelection.current = null
      rig.follow(null)
      return
    }
    const actor = runtime.store.getState().actors[focusedId]
    if (!actor) { rig.follow(null); return }
    const radius = tuning.actor.bodyRadius
    const ground = heightAt(runtime.store.getState().terrain, actor.position[0], actor.position[1])
    const box = new Box3(
      new Vector3(actor.position[0] - radius, ground, actor.position[1] - radius),
      new Vector3(actor.position[0] + radius, ground + radius * 2, actor.position[1] + radius),
    )
    if (framedSelection.current?.instance !== rig.instance || framedSelection.current.actorId !== focusedId) {
      rig.focus(box, true)
      framedSelection.current = { instance: rig.instance, actorId: focusedId }
    }
    rig.follow(
      following
        ? () => {
            const live = runtime.store?.getState().actors[focusedId]
            return live && runtime.store ? [live.position[0], heightAt(runtime.store.getState().terrain, live.position[0], live.position[1]) + radius, live.position[1]] : null
          }
        : null,
    )
  }, [focusedId, following, runtime.store, tuning.actor.bodyRadius])

  const previewHabitat = runtime.store?.getState().habitats
  useEffect(() => {
    if (wildlifePreviewRequest.current) {
      wildlifePreviewRequest.current.abort(); wildlifePreviewRequest.current = null
      setWildlifePreviewStatus('Preview context changed. Choose a preview again.')
    }
  }, [runtime.store, previewHabitat, ambientEnabled, reducedMotion])
  const previewWildlife = useCallback(async (kind: WildlifePreviewId) => {
    wildlifePreviewRequest.current?.abort()
    const controller = new AbortController()
    wildlifePreviewRequest.current = controller
    if (!ambientEnabled || reducedMotion && kind !== 'fireflies') {
      wildlifePreviewRequest.current = null
      setWildlifePreviewStatus(!ambientEnabled ? 'Ambient life is disabled.' : 'Reduced motion suppresses this animal or event.')
      return
    }
    const store = runtime.store
    if (!store) { wildlifePreviewRequest.current = null; setWildlifePreviewStatus('The world is not ready. Choose a preview again.'); return }
    const world = store.getState()
    const habitat = world.habitats
    setWildlifePreviewStatus('Finding eligible habitat…')
    try {
      const preview = await prepareWildlifePreview(world, kind, worldClock.snapshot().utcMilliseconds, controller.signal, reducedMotion)
      if (controller.signal.aborted) return
      if (store.getState().habitats !== habitat) { setWildlifePreviewStatus('The world changed. Choose a preview again.'); return }
      if (!preview) { setWildlifePreviewStatus('No eligible habitat in this world.'); return }
      const rig = cameraRig.current
      if (!rig) { setWildlifePreviewStatus('The camera is not ready. Choose a preview again.'); return }
      setFocusedId(null); setFollowing(false); rig.follow(null)
      worldClock.fix(preview.utcMilliseconds)
      setPeriodMode({ kind: 'fixed', period: preview.period })
      setParams(previous => { const next = new URLSearchParams(previous); next.set('weather', 'clear'); next.set('period', preview.period); return next }, { replace: true })
      const [x, y, z] = preview.position, r = preview.radius
      rig.focus(new Box3(new Vector3(x - r, y - r, z - r), new Vector3(x + r, y + r, z + r)), true)
      setWildlifePreviewStatus(`Preview ready: ${preview.description}.`)
    } catch (error) {
      if (!controller.signal.aborted) setWildlifePreviewStatus(error instanceof Error ? error.message : 'Could not prepare wildlife preview.')
    } finally {
      if (wildlifePreviewRequest.current === controller) wildlifePreviewRequest.current = null
    }
  }, [ambientEnabled, reducedMotion, runtime.store, worldClock, setParams])

  const focusTeam = useCallback(
    (teamId: string) => {
      if (!runtime.store) return
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

  // Editor and modal handlers own Escape before the world home fallback.
  useEffect(() => bindCameraHome(goHome), [goHome])

  // Persisted preferences: adopt the saved choices once loaded, unless the URL pinned them.
  useEffect(() => {
    if (!preferences.loaded || syntheticActors > 0) return
    const saved = preferences.preferences
    setZoomTarget(saved.zoomTarget)
    setAmbientEnabled(saved.ambientLife)
    if (!isSceneId(sceneParam)) setSceneId(saved.scene)
    if (!isQualityProfileId(profileParam)) setQuality({ auto: saved.qualityAuto, profileId: saved.qualityProfile })
    if (periodParam === null) setPeriodMode(saved.periodMode === 'clock' ? { kind: 'clock' } : { kind: 'fixed', period: saved.periodMode })
    if (!showDiagnosticsParam) setShowDiagnostics(saved.showDiagnostics)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- adopt once when the saved preferences arrive
  }, [preferences.loaded])

  const onVerdict = useCallback((record: QualityVerdictRecord) => {
    const history = [...readDiagnostics().qualityHistory, record].slice(-12)
    updateDiagnostics({ qualityHistory: history })
    if (!calibrated.current && record.verdict === 'incline' && quality.auto && !isQualityProfileId(profileParam) && preferences.preferences.qualityProfile === shippedTuning.quality.defaultProfile) {
      calibrated.current = true
      const selected = chooseInitialProfile(record.measuredFps, record.boundFps / tuning.quality.recoverRatio, tuning.quality)
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
  }, [preferences, profileParam, quality.auto, tuning.quality])
  const getTarget = useCallback(
    (): [number, number, number] => cameraRig.current?.target() ?? [bounds.center[0], 0, bounds.center[1]],
    [bounds],
  )
  const { generation: readWorldGeneration, preparation: readWorldPreparation } = runtime
  const readWorkbenchStatus = useCallback(() => {
    const generation = readWorldGeneration()
    const terrain = terrainMeshCache.stats()
    const assets = readDiagnostics().preparedAssets
    return {
      preparation: readWorldPreparation(),
      generation: { count: generation.count, source: generation.source, reason: generation.reason },
      caches: [
        { name: 'Generated worlds', entries: generation.cache.entries, bytes: generation.cache.estimatedBytes, budget: generation.cache.budgetBytes, hits: generation.cache.hits, misses: generation.cache.misses, evictions: generation.cache.evictions },
        { name: 'Terrain meshes', entries: terrain.entries, bytes: terrain.estimatedBytes, budget: terrain.budgetBytes, hits: terrain.hits, misses: terrain.misses, evictions: terrain.evictions },
        { name: 'Prepared geometry', entries: assets.residentAssets, bytes: assets.cpuBytes, budget: assets.budgetBytes, hits: assets.hits, misses: assets.misses, evictions: assets.evictions },
      ],
    }
  }, [readWorldGeneration, readWorldPreparation, terrainMeshCache])

  if (!runtime.store || !terrainTuning) return (
    <div className="flex h-full items-center justify-center" role={runtime.error ? 'alert' : 'status'}>
      {layoutStore.error ? <div role="alert"><p>Could not load layout: {layoutStore.error}</p><button onClick={layoutStore.retry}>Retry layout</button></div> : runtime.error ? <div><p>World preparation failed: {runtime.error}</p><button onClick={runtime.retry}>Retry</button></div> : <p>Preparing world{runtime.progress ? `: ${runtime.progress.stage}` : '…'}</p>}
    </div>
  )

  return (
    <div className="relative h-full w-full overflow-hidden" data-testid="world-view">
      {urlSettingErrors.length > 0 && <div role="alert" className="absolute left-3 top-28 z-50 rounded bg-background p-3 text-sm">{urlSettingErrors.join(' ')}</div>}
      {recipeSession.error && <div role="alert" className="absolute left-3 top-16 z-50 rounded bg-background p-3">{recipeSession.error}</div>}
      {layoutStore.error && <div className="absolute bottom-3 right-3 z-50 rounded bg-background/95 p-3" role="alert">
        <span>Layout {layoutStore.loaded ? 'save' : 'load'} failed: {layoutStore.error}</span>
        <button className="ml-3" onClick={layoutStore.retry}>Retry layout</button>
      </div>}
      {(runtime.preparing || runtime.error) && <div className="absolute left-3 top-3 z-50 rounded bg-background/95 p-3" role={runtime.error ? 'alert' : 'status'}>
        {runtime.error ? <><span>World preparation failed: {runtime.error}</span><button onClick={runtime.retry}>Retry</button></> : `Preparing next world${runtime.progress ? `: ${runtime.progress.stage}` : '…'}`}
      </div>}
      {!twoD && (
      <WorldCanvas profile={profile} camera={tuning.camera} capture={capture}>
        {workbench && showCameraCollisionOverlay && <CameraCollisionOverlay epoch={runtime.epoch} onStatus={setCameraCollisionStatus} />}
        {workbench && (showNavigationOverlay || showBiomeOverlay || showHabitatOverlay) && <GridOverlay kind={showHabitatOverlay ? 'habitats' : showBiomeOverlay ? 'biomes' : 'navigation'} nav={runtime.store.getState().nav} terrain={runtime.store.getState().terrain}
          biomes={runtime.store.getState().biomes} habitats={runtime.store.getState().habitats} biomeSet={biomeSets[scene.biomeSet]} onStatus={setNavigationOverlayStatus} />}
        <CameraRig
          key={scene.id}
          epoch={runtime.epoch}
          ref={cameraRig}
          scene={scene}
          camera={cameraTuning}
          bounds={bounds}
          groundCeiling={groundCeiling}
          walkSurface={walkSurface}
          navigation={navigation}
          tool={navigationTool}
          onNavigationState={navigationTelemetry.publish}
          onFollowDetached={() => setFollowing(false)}
          intro={intro}
          initialPresentation={runtime.initialPresentation}
          reducedMotion={reducedMotion}
          onReady={applyFocus}
          initialPose={importedRecipe?.scene === sceneId ? importedRecipe.view.camera : undefined}
        />
        <WorldStoreContext.Provider value={runtime.store}>
          <FrameDriver animationLeases={animationLeases} settings={tuning.quality.frameDriver} store={runtime.store} weatherActive={(weatherId === 'rain' || weatherId === 'snow') && Math.floor(tuning.weather.particleBaseCount * weatherPreset.particleRate * profile.weatherParticleScale) > 0} diagnosticsOpen={showDiagnostics} continuous={params.get('capture') === '1'} intro={intro && runtime.initialPresentation} settleSeconds={tuning.camera.smoothTime} />
          <SkyEvents clock={worldClock} seed={seed} leases={animationLeases} eligibility={{ ambientEnabled, reducedMotion, night: periodId === 'night', clearSky: weatherId === 'clear' }} preview={workbench ? skyPreview : undefined} onStatus={setSkyStatus} />
          <Fireflies clock={worldClock} leases={animationLeases} enabled={ambientEnabled && periodId === 'night' && weatherId === 'clear'} reducedMotion={reducedMotion} profileId={quality.profileId} />
          <Butterflies clock={worldClock} leases={animationLeases} enabled={ambientEnabled && periodId === 'day' && weatherId === 'clear'} reducedMotion={reducedMotion} profileId={quality.profileId} />
          <Birds clock={worldClock} leases={animationLeases} enabled={ambientEnabled && periodId !== 'night' && weatherId === 'clear'} reducedMotion={reducedMotion} profileId={quality.profileId} />
          <Rabbits clock={worldClock} leases={animationLeases} enabled={ambientEnabled && weatherId === 'clear'} reducedMotion={reducedMotion} profileId={quality.profileId} />
          <Fish clock={worldClock} leases={animationLeases} enabled={ambientEnabled && weatherId === 'clear'} reducedMotion={reducedMotion} profileId={quality.profileId} />
          <LightingRig scene={scene} period={period} lighting={tuning.lighting} profile={profile} bounds={bounds} fovDeg={tuning.camera.fov} store={runtime.store} />
          <CelestialSky seed={runtime.store.getState().seed} profileId={quality.profileId} clock={worldClock} mode={periodMode.kind === 'clock' ? 'clock' : periodMode.period} period={basePeriod} cloudCoverage={weatherPreset.cloudCoverage} />
          <Terrain prepareMesh={terrainMeshCache.prepare} scene={scene} tuning={terrainTuning} profile={profile} weather={weatherPreset} visual={tuning.visual.terrain} />
          <Water scene={scene} tuning={terrainTuning} profile={profile} visual={tuning.visual.water} />
          <Places scene={scene} layout={tuning.layout} />
          <Props scene={scene} period={period} tuning={tuning.layout} lighting={tuning.lighting} profile={profile} camera={tuning.camera} />
          <Vegetation scene={scene} profile={profile} camera={tuning.camera} />
          <ActorPoseProvider focusedId={focusedId}>
            <Actors tuning={tuning.actor} profile={profile} onSelect={editing ? undefined : setFocusedId} onHover={setHoveredId} />
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
          <SceneEnvironment tuning={tuning.weather} scene={scene} profile={profile} period={period} bounds={bounds} weather={weatherPreset} altitude={tuning.weather.cloudAltitude} />
          <Weather id={weatherId} preset={weatherPreset} tuning={tuning.weather} profile={profile} getTarget={getTarget} />
        </WorldStoreContext.Provider>
        <PostChain profile={profile} settings={tuning.visual.post} diagnosticsEnabled={showDiagnostics} />
        <QualityGovernor auto={quality.auto} profile={profile} profileId={quality.profileId} quality={tuning.quality} onVerdict={onVerdict} />
        <DiagnosticsProbe
          epoch={runtime.epoch}
          settings={tuning.quality.diagnostics}
          frameHeight={tuning.camera.frameHeight}
          scene={runtime.store.getState().scene}
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
      {!twoD && <CameraToolbar telemetry={navigationTelemetry} preferences={navigation} onPreferences={setNavigation}
        tool={navigationTool} onTool={setNavigationTool} onMode={mode => { setFollowing(false); cameraRig.current?.setMode(mode) }}
        onCommand={command => cameraRig.current?.command(command)} onPreset={preset => cameraRig.current?.preset(preset)}
        onHome={goHome} canFrame={Boolean(focusedId)} onFrame={() => { framedSelection.current = null; applyFocus() }}
        onLock={() => cameraRig.current?.lockLook()} zoomTarget={zoomTarget}
        onZoomTarget={next => { setZoomTarget(next); preferences.update({ zoomTarget: next }) }} />}
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
      {showDiagnostics && !twoD && <DiagnosticsOverlay seed={seed} seedDigest={seedDigest} refreshMs={tuning.quality.diagnostics.overlayRefreshMs} />}
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
            saving={layoutStore.saving}
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
            worldClock={workbench && !twoD ? worldClock : undefined}
            wildlifePreview={workbench && !twoD ? { status: wildlifePreviewStatus, onPreview: kind => { void previewWildlife(kind) } } : undefined}
            ambientLife={!twoD ? { enabled: ambientEnabled, onChange: enabled => { setAmbientEnabled(enabled); preferences.update({ ambientLife: enabled }) } } : undefined}
            skyPreview={!twoD ? { status: skyStatus, onPreview: variant => {
              if (variant === 'off') { setSkyPreview(undefined); return }
              const snapshot = worldClock.snapshot()
              const duration = variant === 'comet' ? 30 : variant === 'great-fireball' ? 4 : 1.8
              const start = snapshot.utcMilliseconds / 1000 - (snapshot.timeScale === 0 ? duration * .35 : 0)
              const base = { id: `preview:${variant}:${start}`, seed, start, end: start + duration }
              setSkyPreview(variant === 'comet' ? { ...base, family: 'comet' } : { ...base, family: 'meteor', variant })
            } } : undefined}
            cameraCollisionOverlay={!twoD ? { enabled: showCameraCollisionOverlay, onChange: setShowCameraCollisionOverlay, status: cameraCollisionStatus } : undefined}
            navigationOverlay={!twoD ? { enabled: showNavigationOverlay, onChange: enabled => { setShowNavigationOverlay(enabled); if (enabled) { setShowBiomeOverlay(false); setShowHabitatOverlay(false) } }, cells: runtime.store.getState().nav.walkable.length, status: navigationOverlayStatus } : undefined}
            biomeOverlay={!twoD ? { enabled: showBiomeOverlay, onChange: enabled => { setShowBiomeOverlay(enabled); if (enabled) { setShowNavigationOverlay(false); setShowHabitatOverlay(false) } }, legend: biomeLegend(biomeSets[scene.biomeSet]), status: navigationOverlayStatus } : undefined}
            habitatOverlay={!twoD ? { enabled: showHabitatOverlay, onChange: enabled => { setShowHabitatOverlay(enabled); if (enabled) { setShowNavigationOverlay(false); setShowBiomeOverlay(false) } }, legend: habitatLegend(), status: navigationOverlayStatus } : undefined}
            weather={pinnedWeather ?? 'auto'}
            onWeatherChange={(next) => setParams(previous => {
              const updated = new URLSearchParams(previous)
              updated.delete('pressure')
              if (next === 'auto') updated.delete('weather')
              else updated.set('weather', next)
              return updated
            }, { replace: true })}
            seed={seed}
            onSeedChange={(next) => setParams(previous => {
              const updated = new URLSearchParams(previous)
              updated.set('seed', String(next))
              return updated
            }, { replace: true })}
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
            zoomTarget={zoomTarget}
            onZoomTargetChange={(next) => { setZoomTarget(next); preferences.update({ zoomTarget: next }) }}
            levers={workbench ? {
              tuning, override: tuningOverride, onChange: setTuningOverride, onReset: () => setTuningOverride({}),
              readStatus: readWorkbenchStatus,
              importRecipe: text => {
                const recipe = parseRecipeFile(text)
                sessionStorage.setItem(RECIPE_SESSION_KEY, JSON.stringify(recipe))
                window.location.assign(recipeLocation(recipe))
              },
              exportRecipe: syntheticActors > 0 && !runtime.preparing && !twoD ? () => {
                const diagnostics = readDiagnostics()
                if (!diagnostics.ready) throw new Error('Wait for the world to finish presenting before exporting.')
                const surface = document.querySelector('[data-testid="world-canvas"]')
                const canvas = surface instanceof HTMLCanvasElement ? surface : surface?.querySelector('canvas')
                const size = canvas?.getBoundingClientRect()
                if (!size?.width || !size.height) throw new Error('The world canvas is unavailable.')
                const maxTeams = syntheticActors > 25 ? SYNTHETIC_SCALE_MAX_TEAMS : SYNTHETIC_MAX_TEAMS
                const recipe = createDiagnosticRecipe({
                  seed, scene: sceneId,
                  roster: { kind: 'synthetic', actors: syntheticActors, perTeam: importedRecipe?.roster.perTeam ?? Math.max(SYNTHETIC_PER_TEAM, Math.ceil(syntheticActors / maxTeams)) },
                  tuning: { ...tuning, quality: { ...tuning.quality, profiles: { ...tuning.quality.profiles, [quality.profileId]: profile } } },
                  layout: history.current.map(item => ({ ...item, position: item.position ? [item.position[0], item.position[1]] : undefined })), clearPoints: clearPoints.map(point => [...point]),
                  view: { period: periodId, weather: weatherId, profile: quality.profileId, zoomTarget, ambientLife: ambientEnabled, lightingMode: periodMode.kind === 'clock' ? 'clock' : 'preset', clock: { mode: 'fixed', utcMilliseconds: worldClock.snapshot().utcMilliseconds, timeZone: worldClock.snapshot().timeZone }, navigationOverlay: showNavigationOverlay, biomeOverlay: showBiomeOverlay, habitatOverlay: showHabitatOverlay, cameraCollisionOverlay: showCameraCollisionOverlay,
                    camera: { position: diagnostics.cameraPosition, target: diagnostics.cameraTarget } },
                  renderer: { vendor: null, renderer: diagnostics.gpu || null, width: Math.round(size.width), height: Math.round(size.height), dpr: diagnostics.dpr },
                })
                const url = URL.createObjectURL(new Blob([JSON.stringify(recipe, null, 2)], { type: 'application/json' }))
                const link = document.createElement('a')
                link.href = url
                link.download = `world-${sceneId}-${seed}.json`
                link.click()
                window.setTimeout(() => URL.revokeObjectURL(url), 1000)
              } : undefined,
            } : undefined}
          />
        }
        helpTitle="World"
        helpContent={<WorldHelpContent camera={cameraTuning} />}
      />
    </div>
  )
}
