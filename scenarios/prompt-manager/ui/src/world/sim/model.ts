/**
 * The world model. Plain data, no classes, no renderer types.
 * Units: metres on the XZ plane, seconds of wall-clock time, radians.
 */

export type Vec2 = readonly [x: number, z: number]

export type PlaceKind = 'room' | 'desk' | 'table' | 'gathering' | 'hearth' | 'board' | 'corridor' | 'door' | 'filler'

/** A spot an actor can occupy: standing at a desk, sitting at a table or around the campfire. */
export interface Seat {
  id: string
  placeId: string
  position: Vec2
  /** Direction the occupant faces (radians, 0 = +z). */
  facing: number
  /** Sitting seats squash the actor; standing seats do not. */
  sitting: boolean
}

export interface Place {
  id: string
  kind: PlaceKind
  teamId?: string
  ownerAgentId?: string
  /** Room that contains this place, for desks and tables. */
  parentId?: string
  position: Vec2
  /** Yaw in radians (0 = facing +z). */
  rotation: number
  /** Footprint width (x) and depth (z) before rotation. */
  size: Vec2
  seats: Seat[]
  label: string
}

export type DecorKind = 'tree' | 'lamp' | 'decor'

export interface DecorSpot {
  id: string
  kind: DecorKind
  /** Index into the scene's prop list for this kind. */
  variant: number
  /** Stable asset id for biome-driven vegetation and decor. */
  propId?: string
  position: Vec2
  rotation: number
  scale: number
  /** Per-instance RGB multiplier used by vegetation materials. */
  tint?: readonly [number, number, number]
  roomId?: string
}

/** An axis-aligned extent on the ground plane. */
export interface Extent {
  width: number
  depth: number
  center: Vec2
}

export interface WorldBounds extends Extent {
  /**
   * Extent of what was actually placed (rooms, commons, board) without the
   * slab margin.
   */
  footprint: Extent
  /**
   * Ground points on the edge of everything placed: place corners and the
   * commons rim. The camera frames these, so empty corners of the footprint
   * box never push the world away.
   */
  outline: Vec2[]
}

export type ActorState =
  | 'idle'
  | 'walkingToDesk'
  | 'working'
  | 'failed'
  | 'walkingToTable'
  | 'gathered'
  | 'socializing'

export type IdleActivity = 'rest' | 'wander' | 'sit' | 'socialize'

export interface IdleLayer {
  activity: IdleActivity
  /** Sim time at which the current activity ends and a new roll happens. */
  until: number
  partnerId?: string
  seatId?: string
}

export type EmoteKind = 'start' | 'done' | 'fail' | 'message' | 'gather'

export interface ActorAnimation {
  /** Hop phase in [0, 1) while walking; 0 at rest. */
  hopPhase: number
  /** Vertical scale factor; below 1 right after landing, relaxes toward 1. */
  squash: number
  /** Breathing phase in [0, 1). */
  breathPhase: number
  /** Seconds until the next blink starts (or ends while blinking). */
  blinkTimer: number
  blinking: boolean
  /** Whether the actor is in a sitting pose. */
  seated: boolean
  emote?: { kind: EmoteKind; remaining: number }
}

export interface ActorColors {
  body: string
  head: string
  accent: string
}

/** Deterministic cosmetic variation derived from the actor id. */
export interface ActorVariant {
  ears: 0 | 1 | 2
  mouth: 0 | 1 | 2
  /** Body width/height ratio nudge in [-1, 1]. */
  aspect: number
}

export interface LastRun {
  runId: string
  status: 'completed' | 'failed' | 'running'
  startedAt: number
  endedAt?: number
  error?: string
}

export interface Actor {
  id: string
  name: string
  teamId?: string
  /** The desk this actor works at, when it has a team; undefined for unassigned agents. */
  deskSeatId?: string
  state: ActorState
  /** Sim time at which the state was entered. */
  stateSince: number
  position: Vec2
  facing: number
  /** Remaining waypoints; empty when idle in place. */
  path: Vec2[]
  /** Seat the actor is heading to or occupying. */
  seatId?: string
  /** Where the actor is going, for the HUD. */
  destination?: Vec2
  /** Current ground speed (metres per second), ramps toward the target speed. */
  speed: number
  hurrying: boolean
  runId?: string
  lastRun?: LastRun
  failedError?: string
  skillCount: number
  colors: ActorColors
  variant: ActorVariant
  idle: IdleLayer
  anim: ActorAnimation
  /** Last text message from the agent (speech bubble), with its sim time. */
  message?: { text: string; at: number }
}

export type Signal =
  | { kind: 'run.started'; agentId: string; teamId?: string; runId: string; at: number }
  | { kind: 'run.finished'; agentId: string; runId: string; at: number }
  | { kind: 'run.failed'; agentId: string; runId: string; error: string; at: number }
  | { kind: 'heartbeat.upcoming'; teamId: string; scheduledAt: number; at: number }
  | { kind: 'heartbeat.cancelled'; teamId: string; at: number }
  | { kind: 'agent.message'; agentId: string; message: string; at: number }
  | { kind: 'failed.acknowledged'; agentId: string; at: number }

export type SignalKind = Signal['kind']

export interface WorldEvent {
  seq: number
  at: number
  kind: SignalKind | 'actor.arrived' | 'actor.state'
  agentId?: string
  teamId?: string
  runId?: string
  message?: string
  /** For actor.state events: the new state. */
  state?: ActorState
}

export interface Gathering {
  teamId: string
  scheduledAt: number
  /** Sim time after which the gathering is abandoned. */
  until: number
}

export interface NavGrid {
  cellSize: number
  cols: number
  rows: number
  originX: number
  originZ: number
  /** 1 = walkable, 0 = blocked. */
  walkable: Uint8Array
}

export interface WorldState {
  scene: import('../config').SceneId
  seed: number
  rngState: number
  tick: number
  /** Sim wall-clock time in seconds. */
  time: number
  bounds: WorldBounds
  terrain: import('./terrain').TerrainField
  biomes: Uint8Array
  biomeSetId: string
  pathMask: Float32Array
  weather: import('./weather').WeatherState
  places: Record<string, Place>
  placeOrder: string[]
  seats: Record<string, Seat>
  /** seatId -> actorId */
  occupancy: Record<string, string>
  decor: DecorSpot[]
  actors: Record<string, Actor>
  actorOrder: string[]
  gatherings: Record<string, Gathering>
  events: WorldEvent[]
  nextSeq: number
  nav: NavGrid
  /** Increments on every discrete change (state transition, event, layout). Continuous motion does not bump it. */
  revision: number
}

export interface TeamInput {
  id: string
  name: string
  memberIds: string[]
}

export interface AgentInput {
  id: string
  name: string
  colors?: Partial<ActorColors>
  skillCount?: number
}

export interface LayoutOverride {
  placeId: string
  position?: Vec2
  rotation?: number
  removed?: boolean
}

export interface CreateWorldInput {
  seed: number
  /** Sim start time in seconds (wall clock). */
  now: number
  teams: TeamInput[]
  agents: AgentInput[]
  overrides?: LayoutOverride[]
  /** Points the layout keeps clear of trees (the hero camera ground point). */
  clearPoints?: Vec2[]
  scene: import('../config').SceneId
}
