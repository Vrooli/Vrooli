/**
 * createWorld: build the initial state from the team graph.
 */
import { biomeSets, resolveTerrain, overrideTerrain, scenes, type LayoutTuning, type SimTuning, type TerrainTuning } from '../config'
import { layoutStrategies } from './layout/strategy'
import { terrainSurfaceSteps } from './terrain/surface'
import { waterGeometrySteps } from './terrain/waterSurface'
import type { Site } from './layout/sites'
import { pathMaskSteps } from './layout/terrace'
import { buildNavGridSteps, isWalkable, nearestWalkable } from './nav/grid'
import type { AgentInput, TeamInput, Actor, ActorColors, ActorVariant, CreateWorldInput, GeneratedWorld, Seat, WorldState } from './model'
import { Rng, hashString, seedRng } from './rng'
import { buildTerrainSteps, rebuildTerrainRegionSteps } from './terrain/field'
import { biomeGridSteps } from './terrain/biomes'
import { habitatFieldSteps, habitatRegionSteps } from './terrain/habitats'
import { initialWeather } from './weather'
import { sceneBiomes } from '../config/biomes'
import { centreLevelSteps, centreWeight, levelCentreSteps, regionForBounds, terrainForBounds } from './layout/centre'
import { scatterDecorSteps } from './layout/scatter'
import { canonical, canonicalRosterSteps, canonicalTopologySteps, structuralRoster } from './recipe'

export const DEFAULT_COLORS: ActorColors = { body: '#6366f1', head: '#818cf8', accent: '#fbbf24' }
const HALF = 0.5

export interface WorldTuningSlice {
  sim: SimTuning
  layout: LayoutTuning
  terrain: TerrainTuning
  actor: { blinkIntervalSeconds: { min: number; max: number }; bodyRadius: number }
  weather: import('../config').WeatherTuning
}

export function variantFor(id: string): ActorVariant {
  const rng = new Rng(hashString(`variant:${id}`))
  const ears = rng.int(3)
  const mouth = rng.int(3)
  return {
    ears: ears === 0 ? 0 : ears === 1 ? 1 : 2,
    mouth: mouth === 0 ? 0 : mouth === 1 ? 1 : 2,
    aspect: rng.range(-1, 1),
  }
}

export interface GenerationProgress {
  stage: 'terrain' | 'biomes' | 'habitats' | 'layout' | 'dressing' | 'routing' | 'navigation' | 'paths' | 'water-geometry' | 'terrain-surface' | 'indexing' | 'roster'
  completed: number
  total: number
}

function* stageRows<T>(stage: GenerationProgress['stage'], steps: Generator<{ completed: number; total: number }, T>): Generator<GenerationProgress, T> {
  let step = steps.next()
  try {
    while (!step.done) {
      yield { stage, ...step.value }
      step = steps.next()
    }
    return step.value
  } finally {
    steps.return(undefined as T)
  }
}

export function generateWorld(input: Omit<CreateWorldInput, 'now'>, tuning: WorldTuningSlice, treeVariants = 0): GeneratedWorld {
  const steps = generateWorldSteps(input, tuning, treeVariants)
  let step = steps.next()
  while (!step.done) step = steps.next()
  return step.value
}

/** Pure spatial generation with cooperative checkpoints; published inputs and buffers are never mutated. */
export function* generateWorldSteps(input: Omit<CreateWorldInput, 'now'>, tuning: WorldTuningSlice, treeVariants = 0): Generator<GenerationProgress, GeneratedWorld> {
  const roster = yield* stageRows('roster', canonicalRosterSteps(input))
  input = { ...input, ...roster }
  const topologyIdentity = yield* stageRows('roster', canonicalTopologySteps(roster))
  const scene = scenes[input.scene]
  let terrainTuning = resolveTerrain(scene, tuning)
  const terrain = yield* stageRows('terrain', buildTerrainSteps({ seed: input.seed, tuning: terrainTuning }))
  const biomeSet = sceneBiomes(scene)
  // Floorplan placement does not consume biomes; resolve them once after its centre is known.
  let biomes = scene.centre ? undefined : yield* stageRows('biomes', biomeGridSteps(terrain, terrainTuning, biomeSets[scene.biomeSet]))
  const strategy = layoutStrategies[scene.layoutStrategy]
  if (!strategy) throw new Error(`layout strategy ${scene.layoutStrategy} is not registered`)
  yield { stage: 'layout', completed: 0, total: 1 }
  const layoutInput = { teams: input.teams, agents: input.agents, tuning: tuning.layout, options: {
    seed: input.seed,
    // World generation owns dressing so it can yield independently of layout.
    scatterDecor: false,
    treeVariants,
    clearPoints: input.clearPoints,
    overrides: input.overrides,
    terrain,
    terrainTuning,
    biomes,
    biomeSet,
    gatheringLabel: scene.gatheringLabel,
    fillerIds: scene.props.filler,
  } }
  const layout = yield* stageRows('layout', strategy.generateSteps(layoutInput))
  yield { stage: 'layout', completed: 1, total: 1 }
  if (scene.environment === 'outdoor') {
    biomes ??= yield* stageRows('biomes', biomeGridSteps(terrain, terrainTuning, biomeSets[scene.biomeSet]))
    const dressing = yield* stageRows('dressing', scatterDecorSteps({ field: terrain, tuning: terrainTuning, biomes, biomeSet, places: layout.places, bounds: layout.bounds, layout: tuning.layout, seed: input.seed, clearPoints: input.clearPoints ?? [] }))
    for (const [index, item] of layout.decor.entries()) {
      if (index % 128 === 0) yield { stage: 'dressing', completed: index, total: layout.decor.length }
      dressing.push(item)
    }
    layout.decor = dressing
  }
  const region = regionForBounds(scene, layout.bounds)
  if (region) {
    const level = yield* stageRows('terrain', centreLevelSteps(terrain, scene, layout.bounds, tuning.terrain))
    terrainTuning = terrainForBounds(scene, tuning.terrain, layout.bounds)
    yield* stageRows('terrain', rebuildTerrainRegionSteps(terrain, { seed: input.seed, tuning: terrainTuning }, (x, z) => centreWeight(region, x, z) > 0))
    if (level !== undefined) yield* stageRows('terrain', levelCentreSteps(terrain, region, level))
    biomes = yield* stageRows('biomes', biomeGridSteps(terrain, terrainTuning, biomeSet, (x, z) => biomeSets[
      scene.centre?.biomeSet && centreWeight(region, x, z) === 1 ? scene.centre.biomeSet : scene.biomeSet
    ]))
    const exterior = yield* stageRows('dressing', scatterDecorSteps({
      field: terrain, tuning: terrainTuning, biomes, biomeSet,
      places: layout.places, bounds: layout.bounds, layout: tuning.layout,
      seed: input.seed, clearPoints: input.clearPoints ?? [],
      exclude: (x, z) => centreWeight(region, x, z) === 1,
    }))
    for (const [index, item] of exterior.entries()) {
      if (index % 128 === 0) yield { stage: 'dressing', completed: index, total: exterior.length }
      layout.decor.push(item)
    }
  }
  if (!biomes) throw new Error('World generation did not resolve biome data')
  const places: WorldState['places'] = {}
  const seats: Record<string, Seat> = {}
  const placeOrder: string[] = []
  const roomSites: Site[] = []
  let commonsPlace: WorldState['places'][string] | undefined
  for (const [index, place] of layout.places.entries()) {
    if (index % 128 === 0) yield { stage: 'indexing', completed: index, total: layout.places.length }
    places[place.id] = place
    placeOrder.push(place.id)
    for (const [seatIndex, seat] of place.seats.entries()) {
      if (seatIndex % 128 === 0) yield { stage: 'indexing', completed: seatIndex, total: place.seats.length }
      seats[seat.id] = seat
    }
    if (!commonsPlace && place.kind === 'gathering') commonsPlace = place
    if (place.kind === 'room') {
      const exitDistance = place.size[1] / 2 + tuning.layout.cellSize
      roomSites.push({
        position: [place.position[0] + Math.sin(place.rotation) * exitDistance, place.position[1] + Math.cos(place.rotation) * exitDistance],
        rotation: place.rotation,
        size: place.size,
        height: 0,
      })
    }
  }
  // Route paths over dry ground before the terrace kerbs apply the normal
  // walking-slope gate. The final nav admits those explicit paths, then stamps
  // walls and props over them so a path never cuts through a place.
  const routingNav = yield* stageRows('routing', buildNavGridSteps(layout.bounds, layout.places, [], tuning.layout.cellSize, tuning.layout.cellSize, tuning.actor.bodyRadius, terrain, overrideTerrain(terrainTuning, { maxWalkSlope: Math.PI / 2 })))
  const commonsCenter = commonsPlace?.position ?? [0, 0]
  const commonsPathTarget = [commonsCenter[0], commonsCenter[1] + tuning.layout.commonsSeatRadius] as const
  const paintedPaths = yield* stageRows('paths', pathMaskSteps(terrain, terrainTuning, routingNav, roomSites, commonsPathTarget))
  const nav = yield* stageRows('navigation', buildNavGridSteps(layout.bounds, layout.places, layout.decor, tuning.layout.cellSize, tuning.layout.cellSize, tuning.actor.bodyRadius, terrain, terrainTuning, paintedPaths))
  const terrainSurface = yield* stageRows('terrain-surface', terrainSurfaceSteps(terrain, terrainTuning, biomes, paintedPaths, biomeSet))
  const waterGeometry = yield* stageRows('water-geometry', waterGeometrySteps(terrain, terrainTuning))
  const habitats = yield* stageRows('habitats', habitatFieldSteps(terrain, biomes, biomeSet, paintedPaths, Object.values(places)))
  const habitatRegions = yield* stageRows('habitats', habitatRegionSteps(terrain, habitats))
  return {
    scene: input.scene, seed: input.seed, bounds: layout.bounds, waterGeometry, terrainSurface,
    terrain, biomes, habitats, habitatRegions, biomeSetId: biomeSet.id, pathMask: paintedPaths,
    places, placeOrder, seats,
    decor: layout.decor, nav, deskSeatByAgent: layout.deskSeatByAgent,
    topologyIdentity,
  }
}

/** Overlay current display labels without mutating reusable generated places. */
export function presentationPlaces(places: WorldState['places'], teams: readonly TeamInput[], agents: readonly AgentInput[]): WorldState['places'] {
  const names = new Map(agents.map(agent => [agent.id, agent.name]))
  const teamNames = new Map(teams.map(team => [team.id, team.name]))
  let result = places
  for (const place of Object.values(places)) {
    const teamName = place.teamId ? teamNames.get(place.teamId) : undefined
    const label = place.ownerAgentId ? names.get(place.ownerAgentId) : teamName && place.kind === 'room' ? teamName : teamName && place.kind === 'table' ? `${teamName} table` : undefined
    if (label === undefined || label === place.label) continue
    if (result === places) result = { ...places }
    result[place.id] = { ...place, label }
  }
  return result
}

/** Attach independent mutable simulation state to a reusable spatial result.
 * Generated buffers remain borrowed and must not be mutated or transferred.
 */
export function createLiveWorld(generated: GeneratedWorld, input: CreateWorldInput, tuning: WorldTuningSlice): WorldState {
  if (input.scene !== generated.scene || input.seed !== generated.seed) throw new Error('Generated world does not match live scene and seed')
  if (canonical(structuralRoster(input)) !== generated.topologyIdentity) throw new Error('Generated world does not match live roster topology')
  const { seats, nav } = generated
  const places = presentationPlaces(generated.places, input.teams, input.agents)
  const rng = new Rng(seedRng(input.seed))
  const weather = initialWeather(input.now, rng, tuning.weather)
  const teamOf = new Map<string, string>()
  for (const team of [...input.teams].sort((a, b) => a.id.localeCompare(b.id))) {
    for (const memberId of team.memberIds) if (!teamOf.has(memberId)) teamOf.set(memberId, team.id)
  }
  const actors: Record<string, Actor> = {}
  const actorOrder: string[] = []
  const commons = places.gathering
  const spawnRadius = tuning.layout.commonsRadius - tuning.layout.clearingRadius * HALF
  const occupancy: Record<string, string> = {}
  for (const agent of [...input.agents].sort((a, b) => a.id.localeCompare(b.id))) {
    // Home is the desk: a member spawns at its desk seat so the first frame
    // shows every room staffed. Unassigned agents live in the commons.
    const deskSeatId = generated.deskSeatByAgent[agent.id]
    const home = deskSeatId ? seats[deskSeatId] : undefined
    const angle = rng.next() * Math.PI * 2
    const radius = tuning.layout.commonsSeatRadius + (spawnRadius - tuning.layout.commonsSeatRadius) * Math.sqrt(rng.next())
    const center = commons ? commons.position : ([0, 0] as const)
    const facing = rng.range(-Math.PI, Math.PI)
    if (home) occupancy[home.id] = agent.id
    const sampledPosition: readonly [number, number] = [center[0] + Math.sin(angle) * radius, center[1] + Math.cos(angle) * radius]
    const actor: Actor = {
      id: agent.id,
      name: agent.name,
      teamId: teamOf.get(agent.id),
      deskSeatId,
      state: 'idle',
      stateSince: input.now,
      position: home ? home.position : nearestWalkable(nav, sampledPosition, 8) ?? sampledPosition,
      facing: home ? home.facing : facing,
      path: [],
      seatId: home?.id,
      speed: 0,
      hurrying: false,
      skillCount: agent.skillCount ?? 0,
      colors: { ...DEFAULT_COLORS, ...agent.colors },
      variant: variantFor(agent.id),
      idle: { activity: 'rest', until: input.now + rng.range(0, tuning.sim.idle.rollIntervalSeconds) },
      anim: {
        hopPhase: 0,
        squash: 1,
        breathPhase: rng.next(),
        blinkTimer: rng.range(tuning.actor.blinkIntervalSeconds.min, tuning.actor.blinkIntervalSeconds.max),
        blinking: false,
        seated: home?.sitting ?? false,
      },
    }
    actors[actor.id] = actor
    actorOrder.push(actor.id)
  }
  return {
    scene: input.scene,
    seed: input.seed,
    rngState: rng.state,
    tick: 0,
    time: input.now,
    bounds: generated.bounds,
    terrain: generated.terrain,
    waterGeometry: generated.waterGeometry,
    terrainSurface: generated.terrainSurface,
    biomes: generated.biomes,
    habitats: generated.habitats,
    habitatRegions: generated.habitatRegions,
    biomeSetId: generated.biomeSetId,
    pathMask: generated.pathMask,
    weather,
    places,
    placeOrder: generated.placeOrder,
    seats,
    occupancy,
    decor: generated.decor,
    actors,
    actorOrder,
    gatherings: {},
    events: [],
    nextSeq: 1,
    nav,
    revision: 1,
  }
}

/** Synchronous composition retained for deterministic tools and non-browser callers. */
export function createWorld(input: CreateWorldInput, tuning: WorldTuningSlice, treeVariants = 0): WorldState {
  return createLiveWorld(generateWorld(input, tuning, treeVariants), input, tuning)
}

/**
 * Regenerate places, seats, nav and decor for a new override set while
 * keeping actor identity and activity. Paths reset and desk assignments
 * refresh; resting occupants move with regenerated homes while active actors
 * retain walkable positions. Used by the editor and roster adoption.
 */
export function rebuildLayout(state: WorldState, input: CreateWorldInput, tuning: WorldTuningSlice, treeVariants = 0): WorldState {
  const fresh = createWorld({ ...input, now: state.time }, tuning, treeVariants)
  return reconcileGeneratedWorld(state, fresh)
}

/** Keep live activity while replacing spatial data and reconciling roster membership. */
export function reconcileGeneratedWorld(state: WorldState, fresh: WorldState): WorldState {
  const actors: Record<string, Actor> = {}
  const occupancy: Record<string, string> = {}
  for (const id of fresh.actorOrder) {
    const next = fresh.actors[id]
    if (!next) continue
    const actor = state.actors[id]
    if (!actor) {
      actors[id] = next
      if (next.seatId) occupancy[next.seatId] = id
      continue
    }
    // A seat that survived at the same spot is kept (the room did not move).
    // Active actors release moved seats and re-path on the next simulation tick.
    const held = actor.seatId ? fresh.seats[actor.seatId] : undefined
    const previous = actor.seatId ? state.seats[actor.seatId] : undefined
    const keeps = Boolean(actor.teamId === next.teamId && held && previous && held.position[0] === previous.position[0] && held.position[1] === previous.position[1] && actor.path.length === 0)
    // A resting occupant moves with its regenerated home instead of becoming
    // stranded at the removed chair. Preserve its live identity and clocks.
    const relocated = !keeps && actor.state === 'idle' && actor.path.length === 0 && actor.seatId !== undefined
    if (keeps && actor.seatId) occupancy[actor.seatId] = id
    if (relocated && next.seatId) occupancy[next.seatId] = id
    actors[id] = {
      ...actor,
      name: next.name, teamId: next.teamId, colors: next.colors, skillCount: next.skillCount,
      deskSeatId: next.deskSeatId,
      position: relocated ? next.position : isWalkable(fresh.nav, actor.position) ? actor.position : nearestWalkable(fresh.nav, actor.position, 8) ?? next.position,
      facing: relocated ? next.facing : actor.facing,
      path: [],
      seatId: keeps ? actor.seatId : relocated ? next.seatId : undefined,
      destination: undefined,
      speed: 0,
      anim: { ...actor.anim, seated: keeps ? actor.anim.seated : relocated ? next.anim.seated : false },
      idle: { activity: 'rest', until: state.time },
    }
  }
  const teamIds = new Set(Object.values(fresh.places).flatMap(place => place.teamId ? [place.teamId] : []))
  return {
    ...fresh,
    terrain: fresh.terrain,
    biomes: fresh.biomes,
    biomeSetId: fresh.biomeSetId,
    pathMask: fresh.pathMask,
    weather: state.weather,
    rngState: state.rngState,
    tick: state.tick,
    time: state.time,
    actors,
    actorOrder: [...fresh.actorOrder],
    gatherings: Object.fromEntries(Object.entries(state.gatherings).filter(([teamId]) => teamIds.has(teamId))),
    events: [...state.events],
    nextSeq: state.nextSeq,
    occupancy,
    revision: state.revision + 1,
  }
}
