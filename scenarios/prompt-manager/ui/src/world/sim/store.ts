/**
 * A minimal store around the world state: dispatch signals, advance time,
 * read state or view, subscribe to discrete changes. Renderer-free so the
 * data layer can drive it from a clock and tests from a script.
 */
import type { WorldTuning } from '../config'
import type { AgentInput, TeamInput, CreateWorldInput, GeneratedWorld, LayoutOverride, Signal, WorldState } from './model'
import { resizePathCache, step } from './tick'
import { createViewSelector, type WorldView } from './view/select'
import { createLiveWorld, createWorld, reconcileGeneratedWorld, rebuildLayout, presentationPlaces, DEFAULT_COLORS } from './world'

export interface WorldStore {
  setVisitorConversation(visitor: WorldState["visitorConversation"]): void
  getState(): WorldState
  getView(): WorldView
  /** Queue signals; they apply on the next tick in order. */
  dispatch(signals: readonly Signal[]): void
  /** Advance by dt seconds, running as many fixed ticks as fit; leftover time carries over. */
  advance(dt: number): void
  /** Listener runs after any tick that changed the revision. */
  subscribe(listener: () => void): () => void
  /** Full snapshots, including motion between discrete view revisions. */
  subscribeState(listener: () => void): () => void
  /** Apply live tuning and bounded-history limits; publish only changed view inputs. */
  setTuning(tuning: WorldTuning): void
  tuning(): WorldTuning
  /** Regenerate the layout with new overrides, keeping every actor where it stands. */
  applyOverrides(overrides: LayoutOverride[]): void
  adoptGenerated(generated: GeneratedWorld, input: CreateWorldInput): void
  overrides(): LayoutOverride[]
  updatePresentation(teams: readonly TeamInput[], agents: readonly AgentInput[]): void
}

export function createWorldStore(input: CreateWorldInput, tuning: WorldTuning, treeVariants = 0, generated?: GeneratedWorld): WorldStore {
  let current = generated ? createLiveWorld(generated, input, tuning) : createWorld(input, tuning, treeVariants)
  let active = tuning
  let currentOverrides: LayoutOverride[] = input.overrides ?? []
  let presentation = { teams: input.teams, agents: input.agents }
  let pending: Signal[] = []
  let carry = 0
  let select = createViewSelector(active.actor)
  const listeners = new Set<() => void>()
  const stateListeners = new Set<() => void>()

  const notify = (before: number, stateChanged = true) => {
    if (stateChanged) for (const listener of stateListeners) listener()
    if (current.revision === before) return
    for (const listener of listeners) listener()
  }

  return {
    setVisitorConversation: (visitor) => {
      const before = current.revision
      const previous = current.visitorConversation
      current = { ...current, visitorConversation: visitor && previous?.agentId === visitor.agentId && previous.stationary === visitor.stationary ? { ...previous, ...visitor } : visitor }
      if (previous && previous.agentId !== visitor?.agentId) {
        const actor = current.actors[previous.agentId]
        if (actor) {
          const occupancy = Object.fromEntries(Object.entries(current.occupancy).filter(([seat, occupant]) => seat !== actor.seatId || occupant !== actor.id))
          current = { ...current, occupancy, actors: { ...current.actors, [actor.id]: {
            ...actor, path: [], destination: undefined, seatId: undefined,
            state: actor.runId || actor.state === 'failed' ? actor.state : 'idle',
            stateSince: actor.runId || actor.state === 'failed' ? actor.stateSince : current.time,
            idle: { activity: 'rest', until: current.time }, anim: { ...actor.anim, seated: false },
          } } }
        }
      }
      if (previous?.agentId !== visitor?.agentId) {
        current = { ...current, revision: current.revision + 1 }
      }
      notify(before)
    },
    getState: () => current,
    getView: () => select(current),
    dispatch: (signals) => {
      pending.push(...signals)
    },
    advance: (dt) => {
      carry += dt
      const tickSeconds = active.sim.tickSeconds
      const before = current.revision
      const previousState = current
      while (carry >= tickSeconds - 1e-9) {
        const signals = pending
        pending = []
        current = step(current, tickSeconds, signals, active)
        carry -= tickSeconds
      }
      notify(before, current !== previousState)
    },
    subscribe: (listener) => {
      listeners.add(listener)
      return () => listeners.delete(listener)
    },
    subscribeState: (listener) => {
      stateListeners.add(listener)
      return () => stateListeners.delete(listener)
    },
    setTuning: (next) => {
      const before = current.revision
      const previousCacheSize = active.sim.pathCacheSize
      const previousTiers = active.actor.equipmentTiers
      const nextTiers = next.actor.equipmentTiers
      active = next
      if (previousCacheSize !== next.sim.pathCacheSize) resizePathCache(current, next.sim.pathCacheSize)
      const tiersChanged = previousTiers.length !== nextTiers.length || previousTiers.some((value, index) => value !== nextTiers[index])
      const trimEvents = current.events.length > next.sim.eventsRing
      if (!tiersChanged && !trimEvents) return
      if (tiersChanged) select = createViewSelector(next.actor)
      current = { ...current, events: trimEvents ? current.events.slice(-next.sim.eventsRing) : current.events, revision: current.revision + 1 }
      notify(before)
    },
    tuning: () => active,
    applyOverrides: (overrides) => {
      const before = current.revision
      currentOverrides = overrides
      current = rebuildLayout(current, { ...input, ...presentation, overrides }, active, treeVariants)
      notify(before)
    },
    adoptGenerated: (generated, nextInput) => {
      const before = current.revision
      const fresh = createLiveWorld(generated, { ...nextInput, now: current.time }, active)
      current = reconcileGeneratedWorld(current, fresh)
      input = nextInput
      currentOverrides = nextInput.overrides ?? []
      presentation = { teams: nextInput.teams, agents: nextInput.agents }
      notify(before)
    },
    overrides: () => currentOverrides,
    updatePresentation: (teams, agents) => {
      const before = current.revision
      presentation = { teams: [...teams], agents: [...agents] }
      const actors = { ...current.actors }
      const places = presentationPlaces(current.places, teams, agents)
      let changed = places !== current.places
      for (const input of agents) {
        const actor = actors[input.id]
        if (!actor) continue
        const colors = { ...DEFAULT_COLORS, ...input.colors }
        const skillCount = input.skillCount ?? 0
        if (actor.name === input.name && actor.skillCount === skillCount && actor.colors.body === colors.body && actor.colors.head === colors.head && actor.colors.accent === colors.accent) continue
        actors[input.id] = { ...actor, name: input.name, colors, skillCount }
        changed = true
      }
      if (!changed) return
      current = { ...current, actors, places, revision: current.revision + 1 }
      notify(before)
    },
  }
}
