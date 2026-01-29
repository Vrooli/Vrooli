/**
 * Zustand + R3F Store Testing Utilities
 *
 * Utilities for testing Zustand stores with React Three Fiber patterns:
 * - Store subscription tracking to verify granular selectors
 * - getState() usage validation for useFrame patterns
 * - State snapshot comparisons for action testing
 * - Mutation tracking for ref vs state patterns
 *
 * @example
 * import {
 *   takeStoreSnapshot,
 *   diffSnapshots,
 *   StoreSubscriptionTracker,
 * } from '@/test/r3f-store-test-utils'
 *
 * const before = takeStoreSnapshot(useMyStore)
 * useMyStore.getState().someAction()
 * const after = takeStoreSnapshot(useMyStore)
 * const diff = diffSnapshots(before, after)
 */

import { vi } from 'vitest'
import type { StoreApi, UseBoundStore } from 'zustand'

// =============================================================================
// TYPES
// =============================================================================

/** Generic Zustand store type */
export type AnyStore<T extends object = object> = UseBoundStore<StoreApi<T>>

/** Store snapshot - deep clone of state at a point in time */
export type StoreSnapshot<T> = {
  timestamp: number
  state: T
}

/** Diff result between two snapshots */
export interface SnapshotDiff<T> {
  changed: Partial<T>
  added: Partial<T>
  removed: Partial<T>
  unchanged: Partial<T>
}

/** Subscription event from a tracked store */
export interface SubscriptionEvent<T> {
  timestamp: number
  selector?: (state: T) => unknown
  previousState: Partial<T>
  newState: Partial<T>
  changedKeys: string[]
}

// =============================================================================
// STORE SNAPSHOTS
// =============================================================================

/**
 * Takes a deep clone snapshot of the current store state.
 *
 * @param store - Zustand store
 * @returns Timestamped snapshot of the state
 *
 * @example
 * const before = takeStoreSnapshot(useGameStore)
 * gameStore.getState().movePlayer(10, 20)
 * const after = takeStoreSnapshot(useGameStore)
 */
export function takeStoreSnapshot<T extends object>(
  store: AnyStore<T>
): StoreSnapshot<T> {
  const state = store.getState()
  return {
    timestamp: performance.now(),
    state: deepClone(state),
  }
}

/**
 * Compares two snapshots and returns the differences.
 *
 * @param before - Earlier snapshot
 * @param after - Later snapshot
 * @returns Object containing changed, added, removed, and unchanged keys
 *
 * @example
 * const diff = diffSnapshots(before, after)
 * expect(diff.changed).toHaveProperty('playerX')
 * expect(diff.unchanged).toHaveProperty('playerY')
 */
export function diffSnapshots<T extends object>(
  before: StoreSnapshot<T>,
  after: StoreSnapshot<T>
): SnapshotDiff<T> {
  const changed: Partial<T> = {}
  const added: Partial<T> = {}
  const removed: Partial<T> = {}
  const unchanged: Partial<T> = {}

  const beforeKeys = new Set(Object.keys(before.state))
  const afterKeys = new Set(Object.keys(after.state))

  // Check all keys in after state
  for (const key of afterKeys) {
    const beforeVal = (before.state as Record<string, unknown>)[key]
    const afterVal = (after.state as Record<string, unknown>)[key]

    if (!beforeKeys.has(key)) {
      // Key was added
      (added as Record<string, unknown>)[key] = afterVal
    } else if (!deepEqual(beforeVal, afterVal)) {
      // Key value changed
      (changed as Record<string, unknown>)[key] = afterVal
    } else {
      // Key unchanged
      (unchanged as Record<string, unknown>)[key] = afterVal
    }
  }

  // Check for removed keys
  for (const key of beforeKeys) {
    if (!afterKeys.has(key)) {
      (removed as Record<string, unknown>)[key] = (before.state as Record<string, unknown>)[key]
    }
  }

  return { changed, added, removed, unchanged }
}

/**
 * Asserts that only specific fields changed between snapshots.
 *
 * @param before - Earlier snapshot
 * @param after - Later snapshot
 * @param expectedChangedFields - Fields that should have changed
 * @throws If unexpected fields changed
 *
 * @example
 * // After calling setPlayerX(10), only playerX should change
 * assertOnlyFieldsChanged(before, after, ['playerX'])
 */
export function assertOnlyFieldsChanged<T extends object>(
  before: StoreSnapshot<T>,
  after: StoreSnapshot<T>,
  expectedChangedFields: (keyof T)[]
): void {
  const diff = diffSnapshots(before, after)
  const actualChangedFields = Object.keys(diff.changed) as (keyof T)[]
  const expectedSet = new Set(expectedChangedFields)

  // Check for unexpected changes
  const unexpectedChanges = actualChangedFields.filter(field => !expectedSet.has(field))
  if (unexpectedChanges.length > 0) {
    throw new Error(
      `Unexpected fields changed: ${unexpectedChanges.join(', ')}. ` +
      `Expected only: ${expectedChangedFields.join(', ')}`
    )
  }

  // Check for missing expected changes
  const missingChanges = expectedChangedFields.filter(
    field => !actualChangedFields.includes(field)
  )
  if (missingChanges.length > 0) {
    throw new Error(
      `Expected fields to change but they didn't: ${missingChanges.join(', ')}`
    )
  }
}

// =============================================================================
// SUBSCRIPTION TRACKING
// =============================================================================

/**
 * Tracks store subscriptions to verify granular selector usage.
 *
 * @example
 * const tracker = new StoreSubscriptionTracker(useGameStore)
 *
 * // In component: const playerX = useGameStore(s => s.playerX)
 *
 * // Trigger update that doesn't affect playerX
 * useGameStore.getState().setScore(100)
 *
 * // Verify component didn't re-subscribe (would indicate bad selector)
 * expect(tracker.getSubscriptionEvents()).toHaveLength(0)
 */
export class StoreSubscriptionTracker<T extends object> {
  private events: SubscriptionEvent<T>[] = []
  private unsubscribe: (() => void) | null = null
  private previousState: T

  constructor(private store: AnyStore<T>) {
    this.previousState = deepClone(store.getState())
    this.startTracking()
  }

  private startTracking(): void {
    this.unsubscribe = this.store.subscribe((newState) => {
      const changedKeys: string[] = []
      const previousClone = this.previousState
      const newClone = deepClone(newState)

      for (const key of Object.keys(newState)) {
        if (!deepEqual(
          (previousClone as Record<string, unknown>)[key],
          (newClone as Record<string, unknown>)[key]
        )) {
          changedKeys.push(key)
        }
      }

      if (changedKeys.length > 0) {
        this.events.push({
          timestamp: performance.now(),
          previousState: previousClone as Partial<T>,
          newState: newClone as Partial<T>,
          changedKeys,
        })
      }

      this.previousState = newClone
    })
  }

  /**
   * Get all subscription events.
   */
  getSubscriptionEvents(): SubscriptionEvent<T>[] {
    return [...this.events]
  }

  /**
   * Get events where a specific field changed.
   */
  getEventsForField(field: keyof T): SubscriptionEvent<T>[] {
    return this.events.filter(e => e.changedKeys.includes(field as string))
  }

  /**
   * Get count of subscription events.
   */
  getEventCount(): number {
    return this.events.length
  }

  /**
   * Assert that a specific field triggered a specific number of updates.
   */
  assertFieldUpdates(field: keyof T, expectedCount: number): void {
    const actualCount = this.getEventsForField(field).length
    if (actualCount !== expectedCount) {
      throw new Error(
        `Expected ${expectedCount} updates for field '${String(field)}', but got ${actualCount}`
      )
    }
  }

  /**
   * Clear recorded events.
   */
  clearEvents(): void {
    this.events = []
    this.previousState = deepClone(this.store.getState())
  }

  /**
   * Stop tracking and cleanup.
   */
  dispose(): void {
    this.unsubscribe?.()
    this.unsubscribe = null
  }
}

// =============================================================================
// GETSTATE USAGE VALIDATION
// =============================================================================

/**
 * Validates that useFrame callbacks use getState() instead of subscriptions.
 *
 * This is critical for R3F performance - useFrame should never trigger React
 * re-renders, so it must access store state via getState() not hooks.
 *
 * @param code - Source code string to analyze
 * @returns Validation result with issues found
 *
 * @example
 * const result = validateGetStateUsage(`
 *   useFrame(() => {
 *     const playerX = useGameStore(s => s.playerX) // BAD
 *   })
 * `)
 * expect(result.issues).toContain('useFrame hook subscription')
 */
export function validateGetStateUsage(code: string): {
  isValid: boolean
  issues: string[]
  suggestions: string[]
} {
  const issues: string[] = []
  const suggestions: string[] = []

  // Pattern: useFrame with hook subscription inside
  const useFrameWithHook = /useFrame\s*\(\s*(?:\([^)]*\)|[^)]+)\s*=>\s*\{[^}]*use\w+Store\s*\(/g
  if (useFrameWithHook.test(code)) {
    issues.push('Found store hook subscription inside useFrame - this causes 60 re-renders/sec')
    suggestions.push('Use useStore.getState() instead of useStore(selector) in useFrame')
  }

  // Pattern: setState inside useFrame
  const setStateInUseFrame = /useFrame\s*\(\s*(?:\([^)]*\)|[^)]+)\s*=>\s*\{[^}]*setState\s*\(/g
  if (setStateInUseFrame.test(code)) {
    issues.push('Found setState call inside useFrame - this causes 60 re-renders/sec')
    suggestions.push('Mutate refs directly in useFrame instead of using setState')
  }

  // Pattern: Missing getState for animation values
  const hasUseFrame = /useFrame/.test(code)
  const hasGetState = /\.getState\(\)/.test(code)
  if (hasUseFrame && !hasGetState) {
    // This might be fine if it's just animating refs
    // But we should note it for review
    suggestions.push(
      'Consider using store.getState() in useFrame for values that change infrequently'
    )
  }

  return {
    isValid: issues.length === 0,
    issues,
    suggestions,
  }
}

/**
 * Creates a spy wrapper around a store to track getState() vs subscription usage.
 */
export function createStoreUsageSpy<T extends object>(
  store: AnyStore<T>
): {
  store: AnyStore<T>
  getStateCallCount: () => number
  subscriptionCallCount: () => number
  reset: () => void
} {
  let getStateCalls = 0
  let subscriptionCalls = 0

  const originalGetState = store.getState.bind(store)
  const originalSubscribe = store.subscribe.bind(store)

  // Spy on getState
  store.getState = (() => {
    getStateCalls++
    return originalGetState()
  }) as typeof store.getState

  // Spy on subscribe
  store.subscribe = ((listener: (state: T, prevState: T) => void) => {
    subscriptionCalls++
    return originalSubscribe(listener)
  }) as typeof store.subscribe

  return {
    store,
    getStateCallCount: () => getStateCalls,
    subscriptionCallCount: () => subscriptionCalls,
    reset: () => {
      getStateCalls = 0
      subscriptionCalls = 0
    },
  }
}

// =============================================================================
// MUTATION TRACKING (REF VS STATE)
// =============================================================================

/**
 * Tracks whether updates happen via direct mutation (refs) or state updates.
 *
 * R3F best practice: Animations should use ref mutation, not state updates.
 *
 * @example
 * const tracker = new MutationTracker()
 *
 * // Track ref mutations
 * tracker.trackRef(meshRef.current, 'mesh')
 *
 * // Track state updates
 * tracker.trackStore(useGameStore, 'game')
 *
 * // After animation...
 * expect(tracker.getRefMutationCount('mesh')).toBeGreaterThan(0)
 * expect(tracker.getStateUpdateCount('game')).toBe(0)
 */
export class MutationTracker {
  private refMutations: Map<string, number> = new Map()
  private stateUpdates: Map<string, number> = new Map()
  private unsubscribes: (() => void)[] = []

  /**
   * Track mutations on a ref object (via Proxy).
   */
  trackRef<T extends object>(ref: T, name: string): T {
    this.refMutations.set(name, 0)

    return new Proxy(ref, {
      set: (target, prop, value) => {
        (target as Record<string | symbol, unknown>)[prop] = value
        this.refMutations.set(name, (this.refMutations.get(name) ?? 0) + 1)
        return true
      },
      get: (target, prop) => {
        const value = (target as Record<string | symbol, unknown>)[prop]
        if (typeof value === 'object' && value !== null) {
          return this.trackRef(value as object, `${name}.${String(prop)}`)
        }
        return value
      },
    })
  }

  /**
   * Track state updates on a Zustand store.
   */
  trackStore<T extends object>(store: AnyStore<T>, name: string): void {
    this.stateUpdates.set(name, 0)

    const unsubscribe = store.subscribe(() => {
      this.stateUpdates.set(name, (this.stateUpdates.get(name) ?? 0) + 1)
    })

    this.unsubscribes.push(unsubscribe)
  }

  /**
   * Get mutation count for a tracked ref.
   */
  getRefMutationCount(name: string): number {
    return this.refMutations.get(name) ?? 0
  }

  /**
   * Get update count for a tracked store.
   */
  getStateUpdateCount(name: string): number {
    return this.stateUpdates.get(name) ?? 0
  }

  /**
   * Get all ref mutations.
   */
  getAllRefMutations(): Map<string, number> {
    return new Map(this.refMutations)
  }

  /**
   * Get all state updates.
   */
  getAllStateUpdates(): Map<string, number> {
    return new Map(this.stateUpdates)
  }

  /**
   * Assert that animations used refs, not state.
   */
  assertRefBasedAnimation(refName: string, storeName: string): void {
    const refMutations = this.getRefMutationCount(refName)
    const stateUpdates = this.getStateUpdateCount(storeName)

    if (refMutations === 0 && stateUpdates > 0) {
      throw new Error(
        `Animation used state updates (${stateUpdates}) instead of ref mutations. ` +
        'This will cause 60 re-renders per second.'
      )
    }

    if (refMutations === 0 && stateUpdates === 0) {
      throw new Error('No mutations detected. Animation may not be working.')
    }
  }

  /**
   * Reset all counters.
   */
  reset(): void {
    this.refMutations.clear()
    this.stateUpdates.clear()
  }

  /**
   * Cleanup subscriptions.
   */
  dispose(): void {
    for (const unsubscribe of this.unsubscribes) {
      unsubscribe()
    }
    this.unsubscribes = []
  }
}

// =============================================================================
// STORE TEST HELPERS
// =============================================================================

/**
 * Creates a test-friendly version of a store with reset capability.
 *
 * @example
 * const { store, reset } = createTestStore(useGameStore, { score: 0 })
 *
 * store.getState().setScore(100)
 * expect(store.getState().score).toBe(100)
 *
 * reset() // Back to { score: 0 }
 * expect(store.getState().score).toBe(0)
 */
export function createTestStore<T extends object>(
  store: AnyStore<T>,
  initialState: T
): { store: AnyStore<T>; reset: () => void } {
  // Set initial state
  store.setState(initialState, true)

  return {
    store,
    reset: () => {
      store.setState(initialState, true)
    },
  }
}

/**
 * Waits for a store state to match a condition.
 *
 * @param store - Zustand store
 * @param predicate - Condition to match
 * @param timeout - Maximum wait time in ms
 *
 * @example
 * // Trigger async action
 * store.getState().loadData()
 *
 * // Wait for loading to complete
 * await waitForStoreState(store, s => !s.isLoading, 5000)
 */
export async function waitForStoreState<T extends object>(
  store: AnyStore<T>,
  predicate: (state: T) => boolean,
  timeout = 5000
): Promise<void> {
  // Check immediately
  if (predicate(store.getState())) {
    return
  }

  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      unsubscribe()
      reject(new Error(`Timeout waiting for store state after ${timeout}ms`))
    }, timeout)

    const unsubscribe = store.subscribe((state) => {
      if (predicate(state)) {
        clearTimeout(timer)
        unsubscribe()
        resolve()
      }
    })
  })
}

/**
 * Mocks store actions for isolated testing.
 *
 * @example
 * const mocked = mockStoreActions(useGameStore, ['setScore', 'movePlayer'])
 *
 * // Actions are now vi.fn()
 * store.getState().setScore(100)
 *
 * expect(mocked.setScore).toHaveBeenCalledWith(100)
 */
export function mockStoreActions<T extends object>(
  store: AnyStore<T>,
  actionNames: (keyof T)[]
): Record<string, ReturnType<typeof vi.fn>> {
  const mocks: Record<string, ReturnType<typeof vi.fn>> = {}
  const state = store.getState()

  for (const name of actionNames) {
    if (typeof state[name] === 'function') {
      mocks[name as string] = vi.fn()
      store.setState({ [name]: mocks[name as string] } as Partial<T>, false)
    }
  }

  return mocks
}

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

/**
 * Deep clone an object, handling common Three.js/Zustand patterns.
 */
function deepClone<T>(obj: T): T {
  if (obj === null || typeof obj !== 'object') {
    return obj
  }

  if (Array.isArray(obj)) {
    return obj.map(item => deepClone(item)) as unknown as T
  }

  if (obj instanceof Map) {
    return new Map(Array.from(obj.entries()).map(([k, v]) => [k, deepClone(v)])) as unknown as T
  }

  if (obj instanceof Set) {
    return new Set(Array.from(obj).map(v => deepClone(v))) as unknown as T
  }

  // Skip functions (actions in Zustand stores)
  const cloned: Record<string, unknown> = {}
  for (const key of Object.keys(obj as object)) {
    const value = (obj as Record<string, unknown>)[key]
    if (typeof value !== 'function') {
      cloned[key] = deepClone(value)
    }
  }

  return cloned as T
}

/**
 * Deep equality check, handling common Three.js/Zustand patterns.
 */
function deepEqual(a: unknown, b: unknown): boolean {
  if (a === b) return true

  if (a === null || b === null) return a === b
  if (typeof a !== typeof b) return false

  if (typeof a !== 'object') return a === b

  if (Array.isArray(a) && Array.isArray(b)) {
    if (a.length !== b.length) return false
    return a.every((val, i) => deepEqual(val, b[i]))
  }

  if (a instanceof Map && b instanceof Map) {
    if (a.size !== b.size) return false
    for (const [key, value] of a) {
      if (!b.has(key) || !deepEqual(value, b.get(key))) return false
    }
    return true
  }

  if (a instanceof Set && b instanceof Set) {
    if (a.size !== b.size) return false
    for (const value of a) {
      if (!b.has(value)) return false
    }
    return true
  }

  const aObj = a as Record<string, unknown>
  const bObj = b as Record<string, unknown>
  const aKeys = Object.keys(aObj).filter(k => typeof aObj[k] !== 'function')
  const bKeys = Object.keys(bObj).filter(k => typeof bObj[k] !== 'function')

  if (aKeys.length !== bKeys.length) return false

  return aKeys.every(key => deepEqual(aObj[key], bObj[key]))
}
