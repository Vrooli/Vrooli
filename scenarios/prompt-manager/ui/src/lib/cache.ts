/**
 * Generic cache utility for service layers.
 *
 * Provides a reusable cache implementation with:
 * - Configurable TTL (time-to-live)
 * - Type-safe cache entries
 * - Cache invalidation
 * - Force refresh support
 */

/**
 * Cache entry with data and timestamp.
 */
export interface CacheEntry<T> {
  data: T
  timestamp: number
}

/**
 * Default cache TTL in milliseconds.
 */
export const DEFAULT_CACHE_TTL_MS = 5000 // 5 seconds

/**
 * Check if a cache entry is still valid.
 *
 * @param entry - The cache entry to check
 * @param ttlMs - TTL in milliseconds (default: 5000)
 * @returns True if the entry exists and hasn't expired
 */
export function isCacheValid<T>(
  entry: CacheEntry<T> | null,
  ttlMs: number = DEFAULT_CACHE_TTL_MS
): entry is CacheEntry<T> {
  if (!entry) return false
  return Date.now() - entry.timestamp < ttlMs
}

/**
 * Create a cache entry with current timestamp.
 *
 * @param data - The data to cache
 * @returns A cache entry with the data and current timestamp
 */
export function createCacheEntry<T>(data: T): CacheEntry<T> {
  return {
    data,
    timestamp: Date.now(),
  }
}

/**
 * Creates a cache manager for a specific data type.
 *
 * @example
 * const skillsCache = createCacheManager<Skill[]>()
 *
 * // Set cache
 * skillsCache.set(skills)
 *
 * // Get if valid
 * const cached = skillsCache.getIfValid()
 *
 * // Invalidate
 * skillsCache.invalidate()
 *
 * @param ttlMs - TTL in milliseconds (default: 5000)
 * @returns A cache manager object
 */
export function createCacheManager<T>(ttlMs: number = DEFAULT_CACHE_TTL_MS) {
  let cache: CacheEntry<T> | null = null

  return {
    /**
     * Get cached data if still valid.
     *
     * @param forceRefresh - If true, always returns null
     * @returns The cached data or null if expired/missing
     */
    getIfValid(forceRefresh = false): T | null {
      if (forceRefresh) return null
      return isCacheValid(cache, ttlMs) ? cache.data : null
    },

    /**
     * Set cache with current timestamp.
     *
     * @param data - The data to cache
     */
    set(data: T): void {
      cache = createCacheEntry(data)
    },

    /**
     * Invalidate the cache.
     */
    invalidate(): void {
      cache = null
    },

    /**
     * Check if cache has valid data.
     *
     * @returns True if cache is valid
     */
    isValid(): boolean {
      return isCacheValid(cache, ttlMs)
    },
  }
}
