/**
 * A minimal store around the world state: dispatch signals, advance time,
 * read state or view, subscribe to discrete changes. Renderer-free so the
 * data layer can drive it from a clock and tests from a script.
 */
import type { WorldTuning } from '../config'
import type { CreateWorldInput, LayoutOverride, Signal, WorldState } from './model'
import { step } from './tick'
import { createViewSelector, type WorldView } from './view/select'
import { createWorld, rebuildLayout } from './world'

export interface WorldStore {
  getState(): WorldState
  getView(): WorldView
  /** Queue signals; they apply on the next tick in order. */
  dispatch(signals: readonly Signal[]): void
  /** Advance by dt seconds, running as many fixed ticks as fit; leftover time carries over. */
  advance(dt: number): void
  /** Listener runs after any tick that changed the revision. */
  subscribe(listener: () => void): () => void
  /** Replace the tuning at runtime (dev levers). Takes effect on the next tick. */
  setTuning(tuning: WorldTuning): void
  tuning(): WorldTuning
  /** Regenerate the layout with new overrides, keeping every actor where it stands. */
  applyOverrides(overrides: LayoutOverride[]): void
  overrides(): LayoutOverride[]
}

export function createWorldStore(input: CreateWorldInput, tuning: WorldTuning, treeVariants = 0): WorldStore {
  let current = createWorld(input, tuning, treeVariants)
  let active = tuning
  let currentOverrides: LayoutOverride[] = input.overrides ?? []
  let pending: Signal[] = []
  let carry = 0
  let select = createViewSelector(active.actor)
  const listeners = new Set<() => void>()

  const notify = (before: number) => {
    if (current.revision === before) return
    for (const listener of listeners) listener()
  }

  return {
    getState: () => current,
    getView: () => select(current),
    dispatch: (signals) => {
      pending.push(...signals)
    },
    advance: (dt) => {
      carry += dt
      const tickSeconds = active.sim.tickSeconds
      const before = current.revision
      while (carry >= tickSeconds - 1e-9) {
        const signals = pending
        pending = []
        current = step(current, tickSeconds, signals, active)
        carry -= tickSeconds
      }
      notify(before)
    },
    subscribe: (listener) => {
      listeners.add(listener)
      return () => listeners.delete(listener)
    },
    setTuning: (next) => {
      active = next
      select = createViewSelector(next.actor)
      current = { ...current, revision: current.revision + 1 }
      for (const listener of listeners) listener()
    },
    tuning: () => active,
    applyOverrides: (overrides) => {
      currentOverrides = overrides
      current = rebuildLayout(current, { ...input, overrides }, active, treeVariants)
      for (const listener of listeners) listener()
    },
    overrides: () => currentOverrides,
  }
}
