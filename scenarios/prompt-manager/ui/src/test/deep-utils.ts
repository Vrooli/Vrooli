/**
 * Deep Clone and Equality Utilities
 *
 * Shared utilities for deep cloning and equality comparison, optimized for
 * Three.js objects and Zustand store patterns.
 *
 * Features:
 * - Handles Map, Set, and Array types
 * - Skips functions (preserves Zustand store actions)
 * - Handles null/undefined safely
 *
 * @example
 * import { deepClone, deepEqual } from '@/test/deep-utils'
 *
 * const cloned = deepClone(storeState)
 * const isEqual = deepEqual(before, after)
 */

/**
 * Deep clone an object, handling common Three.js/Zustand patterns.
 *
 * @param obj - Object to clone
 * @returns Deep clone of the object (functions are omitted)
 *
 * @example
 * const before = deepClone(useMyStore.getState())
 * // ... make changes ...
 * const after = deepClone(useMyStore.getState())
 */
export function deepClone<T>(obj: T): T {
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
 *
 * @param a - First value to compare
 * @param b - Second value to compare
 * @returns True if values are deeply equal (ignoring functions)
 *
 * @example
 * if (!deepEqual(before.position, after.position)) {
 *   console.log('Position changed')
 * }
 */
export function deepEqual(a: unknown, b: unknown): boolean {
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
