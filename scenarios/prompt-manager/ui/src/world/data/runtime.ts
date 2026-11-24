/**
 * The live world: a sim store fed by the feed and stepped on a wall clock.
 * Owns the interval and the feed handle; React only sees the store.
 */
import { useCallback, useEffect, useRef, useState } from 'react'
import { listRunningAgents } from '@/services/heartbeatService'
import { tuning as shippedTuning, type SceneId, type WeatherId, type WorldTuning } from '../config'
import { checkWorldInvariants, createWorldStore, type AgentInput, type GeneratedWorld, type LayoutOverride, type TeamInput, type Violation, type WorldStore } from '../sim'
import { parseSeconds, type KnownRun } from './adapters'
import { startFeed, type FeedStatus } from './feed'
import { createWorldApi, type WorldApi } from './worldClient'
import { GeneratedWorldCache } from './generationCache'
import { ScenePreparationCoordinator, type PreparationSnapshot } from './scenePreparation'
import { createGenerationClient } from './generationClient'
import type { GenerationProgress } from '../sim/world'
import { canonicalRoster, worldRecipe } from '../sim/recipe'
import { HABITAT_IDS } from '../config/habitats'

export type PreparationProgress = GenerationProgress | { stage: 'presentation' | 'assets'; completed: number; total: number }

export interface RuntimeOptions {
  prepareAssets?: (scene: SceneId, signal: AbortSignal) => Promise<void>
  prepareDestination?: (world: GeneratedWorld, signal: AbortSignal, weather: WeatherId) => Promise<void>
  ready?: boolean
  seed: number
  scene: SceneId
  teams: TeamInput[]
  agents: AgentInput[]
  treeVariants: number
  clearPoints?: Array<readonly [number, number]>
  tuning?: WorldTuning
  api?: WorldApi
  /** When false the runtime steps the sim but opens no feed (tests, smoke goldens). */
  live?: boolean
  /** When false, freeze the demo unless a visitor explicitly invites an agent. */
  step?: boolean
  /** Persisted layout overrides applied at creation; later edits go through store.applyOverrides. */
  overrides?: LayoutOverride[]
}

export async function pollRunningAgents(): Promise<KnownRun[]> {
  const response = await listRunningAgents()
  const now = Date.now() / 1000
  return response.agents.map((a) => ({ agentId: a.agentId, teamId: a.teamId, runId: a.runId, startedAt: parseSeconds(a.startedAt, now) }))
}

export interface WorldRuntime extends Pick<WorldSimProbe, 'preparation' | 'generation'> {
  initialPresentation: boolean
  epoch: number
  store: WorldStore | null
  preparing: boolean
  error: string | null
  progress: PreparationProgress | null
  retry: () => void
  feed: FeedStatus
}

/** Handle the smoke tool reads from `window.__worldSim`: invariants of the live world, on demand. */
export interface WorldSimProbe {
  habitats(): { samples: number; bytes: number; counts: Record<string, number>; regions: { count: number; members: number; bytes: number; counts: Record<string, number> } }
  violations(): Violation[]
  vegetationDry(): { checked: number; violations: Violation[] }
  revision(): number
  actorCount(): number
  scene(): SceneId
  preparation(): PreparationSnapshot
  generation(): { mode: 'worker' | 'fallback'; fallbackReason?: string; source: 'worker' | 'fallback' | 'cache'; cache: ReturnType<GeneratedWorldCache['stats']>; count: number; generatorVersion: string; reason: string }
}

declare global {
  interface Window {
    __worldSim?: WorldSimProbe
  }
}

const IDLE_TICK_MULTIPLIER = 1
let nextCommitEpoch = 0

export function useWorldRuntime(options: RuntimeOptions): WorldRuntime {
  const tuning = options.tuning ?? shippedTuning
  const recipe = worldRecipe(options, tuning)
  const rosterKey = recipe.dressingIdentity
  const execution = useRef<{ mode: 'worker' | 'fallback'; fallbackReason?: string }>({ mode: 'worker' })
  const generation = useRef({ count: 0, generatorVersion: recipe.generatorVersion, reason: 'initial', terrainIdentity: '', layoutIdentity: '', identity: '' })
  const cache = useRef(new GeneratedWorldCache())
  const preparation = useRef(new ScenePreparationCoordinator())
  const [prepared, setPrepared] = useState<{ initial: boolean; source: 'worker' | 'fallback' | 'cache'; epoch: number; identity: string; store: WorldStore } | null>(null)
  const store = prepared?.store ?? null
  const readGeneration = useCallback<WorldSimProbe['generation']>(() => ({
    ...execution.current, source: prepared?.source ?? 'worker', cache: cache.current.stats(),
    count: generation.current.count, generatorVersion: generation.current.generatorVersion, reason: generation.current.reason,
  }), [prepared?.source])
  const [error, setError] = useState<string | null>(null)
  const [progress, setProgress] = useState<PreparationProgress | null>(null)
  const [attempt, setAttempt] = useState(0)
  const committedStore = useRef<WorldStore | null>(null)
  committedStore.current = store
  const latest = useRef(options)
  latest.current = options
  useEffect(() => {
    if (options.ready === false) return
    let request: ReturnType<ScenePreparationCoordinator['start']> | undefined
    let client: ReturnType<typeof createGenerationClient> | undefined
    setError(null)
    setProgress(null)
    const input = {
      seed: options.seed, ...canonicalRoster(options), scene: options.scene,
      clearPoints: options.clearPoints, overrides: options.overrides,
    }
    try {
      const cached = cache.current.get(rosterKey)
      if (!cached) {
        client = createGenerationClient()
        execution.current = { mode: client.mode, fallbackReason: client.fallbackReason }
      }
      request = preparation.current.start(
        async (signal, runStage) => {
          let assetsReady = false
          const assets = runStage('assets', async () => {
            signal.throwIfAborted()
            await latest.current.prepareAssets?.(input.scene, signal)
          }).then(() => { assetsReady = true })
          const generationResult = runStage('generation', () => cached ? Promise.resolve(cached) : client ? client.generate({ input, tuning }, { signal, onProgress: progress => { if (!signal.aborted) setProgress(progress) } }) : Promise.reject(new Error('Generation worker is unavailable'))).then(world => {
            if (!assetsReady && !signal.aborted) setProgress({ stage: 'assets', completed: 0, total: 1 })
            return world
          })
          const [generated] = await Promise.all([generationResult, assets])
          signal.throwIfAborted()
          const current = latest.current
          const liveInput = { ...input, ...canonicalRoster(current), now: Date.now() / 1000 }
          const previous = committedStore.current
          const next = previous?.getState().scene === input.scene && previous.getState().seed === input.seed
            ? previous
            : createWorldStore(liveInput, current.tuning ?? shippedTuning, current.treeVariants, generated)
          let prepareDestination = latest.current.prepareDestination
          while (prepareDestination) {
            setProgress({ stage: 'presentation', completed: 0, total: 1 })
            const destination = prepareDestination
            await runStage('presentation', () => destination(generated, signal, next.getState().weather.state))
            signal.throwIfAborted()
            if (prepareDestination === latest.current.prepareDestination) break
            prepareDestination = latest.current.prepareDestination
          }
          return { generated, next }
        },
        ({ generated, next }) => {
        const current = latest.current
        const liveInput = { ...input, ...canonicalRoster(current), now: Date.now() / 1000 }
        if (next === committedStore.current) next.adoptGenerated(generated, liveInput)
        if (!cached) cache.current.put(rosterKey, generated)
        setPrepared({ initial: committedStore.current === null, source: cached ? 'cache' : execution.current.mode, epoch: ++nextCommitEpoch, identity: rosterKey, store: next })
      })
      void request.promise.catch((failure: unknown) => { setError(failure instanceof Error ? failure.message : String(failure)) })
    } catch (failure) {
      setError(failure instanceof Error ? failure.message : String(failure))
    }
    return () => { request?.cancel(); client?.dispose() }
    // Identity contains generation inputs; cosmetic changes apply at publication.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rosterKey, options.ready, attempt])
  const [feed, setFeed] = useState<FeedStatus>({ mode: 'stopped', lastEventAt: null, reconnects: 0, lastError: null })
  const apiRef = useRef<WorldApi | null>(options.api ?? null)

  useEffect(() => {
    store?.setTuning(tuning)
  }, [store, tuning])

  useEffect(() => {
    const roster = canonicalRoster({ teams: options.teams, agents: options.agents })
    store?.updatePresentation(roster.teams, roster.agents)
  }, [store, options.teams, options.agents])

  useEffect(() => {
    if (!store) return
    if (prepared?.identity === rosterKey && generation.current.identity !== rosterKey) {
      const previous = generation.current
      generation.current = {
        count: previous.count + 1,
        generatorVersion: recipe.generatorVersion,
        reason: previous.count === 0 ? 'initial' : previous.terrainIdentity !== recipe.terrainIdentity ? 'terrain-input' : previous.layoutIdentity !== recipe.layoutIdentity ? 'layout-input' : 'dressing-input',
        terrainIdentity: recipe.terrainIdentity, layoutIdentity: recipe.layoutIdentity, identity: rosterKey,
      }
    }
    const probe: WorldSimProbe = {
      habitats: () => {
        const field = store.getState().habitats
        const counts: Record<string, number> = Object.fromEntries(HABITAT_IDS.map(id => [id, 0]))
        for (const index of field) {
          const id = HABITAT_IDS[index] ?? 'none'
          counts[id] = (counts[id] ?? 0) + 1
        }
        const regions = store.getState().habitatRegions
        const regionCounts: Record<string, number> = Object.fromEntries(HABITAT_IDS.map(id => [id, 0]))
        for (const region of regions.regions) regionCounts[region.habitat] = (regionCounts[region.habitat] ?? 0) + 1
        return { samples: field.length, bytes: field.byteLength, counts, regions: {
          count: regions.regions.length, members: regions.members.length,
          bytes: regions.labels.byteLength + regions.members.byteLength, counts: regionCounts,
        } }
      },
      violations: () => checkWorldInvariants(store.getState(), store.tuning()),
      vegetationDry: () => ({ checked: store.getState().decor.length, violations: checkWorldInvariants(store.getState(), store.tuning()).filter((violation) => violation.rule === 'vegetation-dry') }),
      revision: () => store.getState().revision,
      actorCount: () => store.getState().actorOrder.length,
      scene: () => store.getState().scene,
      preparation: () => preparation.current.read(),
      generation: readGeneration,
    }
    window.__worldSim = probe
    return () => {
      if (window.__worldSim === probe) delete window.__worldSim
    }
  }, [store, prepared?.identity, prepared?.source, rosterKey, recipe.generatorVersion, recipe.terrainIdentity, recipe.layoutIdentity, readGeneration])

  useEffect(() => {
    if (!store) return
    let last = performance.now()
    const interval = window.setInterval(() => {
      const now = performance.now()
      if (options.step !== false || store.getState().visitorConversation) store.advance(((now - last) / 1000) * IDLE_TICK_MULTIPLIER)
      last = now
    }, tuning.sim.tickSeconds * 1000)
    return () => window.clearInterval(interval)
  }, [store, tuning.sim.tickSeconds, options.step])

  useEffect(() => {
    if (!store || options.live === false) return
    apiRef.current ??= createWorldApi()
    const handle = startFeed({
      api: apiRef.current,
      pollRunning: pollRunningAgents,
      onSignals: (signals) => store.dispatch(signals),
      onStatus: setFeed,
      tuning: tuning.data,
    })
    return () => handle.stop()
  }, [store, tuning.data, options.live])

  return { preparation: preparation.current.read, generation: readGeneration, initialPresentation: prepared?.initial ?? true, epoch: prepared?.epoch ?? 0, store, feed, error, progress, preparing: options.ready === false || prepared?.identity !== rosterKey, retry: () => setAttempt(value => value + 1) }
}
