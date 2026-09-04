/**
 * createWorld: build the initial state from the team graph.
 */
import { biomeSets, resolveTerrain, scenes, type LayoutTuning, type SimTuning, type TerrainTuning } from '../config'
import { layoutStrategies } from './layout/strategy'
import { pathMask as buildPathMask } from './layout/terrace'
import { buildNavGrid, nearestWalkable } from './nav/grid'
import type { Actor, ActorColors, ActorVariant, CreateWorldInput, Seat, WorldState } from './model'
import { Rng, hashString, seedRng } from './rng'
import { biomeGrid, buildTerrain } from './terrain'
import { initialWeather } from './weather'

const DEFAULT_COLORS: ActorColors = { body: '#6366f1', head: '#818cf8', accent: '#fbbf24' }
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

export function createWorld(input: CreateWorldInput, tuning: WorldTuningSlice, treeVariants = 0): WorldState {
  const scene = scenes[input.scene]
  const terrainTuning = resolveTerrain(scene, tuning)
  const terrain = buildTerrain({ seed: input.seed, tuning: terrainTuning })
  const biomeSet = biomeSets[scene.biomeSet]
  const biomes = biomeGrid(terrain, terrainTuning, biomeSet)
  const strategy = layoutStrategies[scene.layoutStrategy]
  if (!strategy) throw new Error(`layout strategy ${scene.layoutStrategy} is not registered`)
  const layout = strategy.generate({ teams: input.teams, agents: input.agents, tuning: tuning.layout, options: {
    seed: input.seed,
    scatterDecor: scene.environment === 'outdoor',
    treeVariants,
    clearPoints: input.clearPoints,
    overrides: input.overrides,
    terrain,
    terrainTuning,
    biomes,
    biomeSet,
    treePropIds: [...new Set(biomeSet.biomes.flatMap((biome) => Object.keys(biome.vegetation)))],
    gatheringLabel: scene.gatheringLabel,
    fillerIds: scene.props.filler,
  } })
  const places: WorldState['places'] = {}
  const seats: Record<string, Seat> = {}
  for (const place of layout.places) {
    places[place.id] = place
    for (const seat of place.seats) seats[seat.id] = seat
  }
  const commonsPlace = layout.places.find((place) => place.kind === 'gathering')
  const roomSites = layout.places.filter((place) => place.kind === 'room').map((place) => {
    const exitDistance = place.size[1] / 2 + tuning.layout.cellSize
    return {
      position: [place.position[0] + Math.sin(place.rotation) * exitDistance, place.position[1] + Math.cos(place.rotation) * exitDistance] as const,
      rotation: place.rotation,
      size: place.size,
      height: 0,
    }
  })
  // Route paths over dry ground before the terrace kerbs apply the normal
  // walking-slope gate. The final nav admits those explicit paths, then stamps
  // walls and props over them so a path never cuts through a place.
  const routingNav = buildNavGrid(layout.bounds, layout.places, [], tuning.layout.cellSize, tuning.layout.cellSize, tuning.actor.bodyRadius, terrain, { ...terrainTuning, maxWalkSlope: Math.PI / 2 })
  const commonsCenter = commonsPlace?.position ?? [0, 0]
  const commonsPathTarget = [commonsCenter[0], commonsCenter[1] + tuning.layout.commonsSeatRadius] as const
  const paintedPaths = buildPathMask(terrain, terrainTuning, routingNav, roomSites, commonsPathTarget)
  const nav = buildNavGrid(layout.bounds, layout.places, layout.decor, tuning.layout.cellSize, tuning.layout.cellSize, tuning.actor.bodyRadius, terrain, terrainTuning, paintedPaths)
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
    const deskSeatId = layout.deskSeatByAgent[agent.id]
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
    bounds: layout.bounds,
    terrain,
    biomes,
    biomeSetId: biomeSet.id,
    pathMask: paintedPaths,
    weather,
    places,
    placeOrder: layout.places.map((p) => p.id),
    seats,
    occupancy,
    decor: layout.decor,
    actors,
    actorOrder,
    gatherings: {},
    events: [],
    nextSeq: 1,
    nav,
    revision: 1,
  }
}

/**
 * Regenerate places, seats, nav and decor for a new override set while
 * keeping every actor: positions and states survive, paths reset, seats
 * survive only where they did not move, desk assignments refresh. Used by
 * the editor; never respawns anyone.
 */
export function rebuildLayout(state: WorldState, input: CreateWorldInput, tuning: WorldTuningSlice, treeVariants = 0): WorldState {
  const fresh = createWorld({ ...input, now: state.time }, tuning, treeVariants)
  const actors: Record<string, Actor> = {}
  const occupancy: Record<string, string> = {}
  for (const id of state.actorOrder) {
    const actor = state.actors[id]
    if (!actor) continue
    // A seat that survived at the same spot is kept (the room did not move);
    // any other seat is released and the actor re-paths on its next roll.
    const held = actor.seatId ? fresh.seats[actor.seatId] : undefined
    const previous = actor.seatId ? state.seats[actor.seatId] : undefined
    const keeps = Boolean(held && previous && held.position[0] === previous.position[0] && held.position[1] === previous.position[1] && actor.path.length === 0)
    if (keeps && actor.seatId) occupancy[actor.seatId] = id
    actors[id] = {
      ...actor,
      deskSeatId: fresh.actors[id]?.deskSeatId,
      path: [],
      seatId: keeps ? actor.seatId : undefined,
      destination: undefined,
      speed: 0,
      anim: { ...actor.anim, seated: keeps ? actor.anim.seated : false },
      idle: { activity: 'rest', until: state.time },
    }
  }
  return {
    ...fresh,
    terrain: input.seed === state.seed ? state.terrain : fresh.terrain,
    biomes: input.seed === state.seed ? state.biomes : fresh.biomes,
    biomeSetId: input.seed === state.seed ? state.biomeSetId : fresh.biomeSetId,
    pathMask: fresh.pathMask,
    weather: state.weather,
    rngState: state.rngState,
    tick: state.tick,
    time: state.time,
    actors,
    actorOrder: [...state.actorOrder],
    gatherings: { ...state.gatherings },
    events: [...state.events],
    nextSeq: state.nextSeq,
    occupancy,
    revision: state.revision + 1,
  }
}
