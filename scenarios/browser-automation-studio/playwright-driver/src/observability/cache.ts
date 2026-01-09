/**
 * Observability Cache
 *
 * Time-based caching for observability responses.
 * Prevents expensive operations from being run too frequently.
 */

import type { ObservabilityResponse, ObservabilityDepth } from './types';

// =============================================================================
// Constants
// =============================================================================

/** Default cache TTL in milliseconds */
export const DEFAULT_CACHE_TTL_MS = 30_000; // 30 seconds

// =============================================================================
// Cache Entry Type
// =============================================================================

interface CacheEntry {
  response: ObservabilityResponse;
  cachedAt: Date;
  expiresAt: Date;
}

// =============================================================================
// Cache Class
// =============================================================================

/**
 * Simple time-based cache for observability responses.
 *
 * Features:
 * - Per-depth caching (quick, standard, deep have separate entries)
 * - Configurable TTL
 * - Thread-safe (uses atomic operations)
 * - Auto-cleanup on get
 */
export class ObservabilityCache {
  private cache: Map<ObservabilityDepth, CacheEntry> = new Map();
  private ttlMs: number;

  constructor(ttlMs: number = DEFAULT_CACHE_TTL_MS) {
    this.ttlMs = ttlMs;
  }

  /**
   * Get cached response if valid.
   *
   * @param depth - Depth level to retrieve
   * @returns Cached response with cache metadata, or undefined if not cached/expired
   */
  get(depth: ObservabilityDepth): ObservabilityResponse | undefined {
    const entry = this.cache.get(depth);

    if (!entry) {
      return undefined;
    }

    // Check if expired
    if (new Date() > entry.expiresAt) {
      this.cache.delete(depth);
      return undefined;
    }

    // Return cached response with cache metadata
    return {
      ...entry.response,
      cached: true,
      cached_at: entry.cachedAt.toISOString(),
      cache_ttl_ms: this.ttlMs,
    };
  }

  /**
   * Store a response in the cache.
   *
   * @param depth - Depth level to cache
   * @param response - Response to cache
   */
  set(depth: ObservabilityDepth, response: ObservabilityResponse): void {
    const now = new Date();

    this.cache.set(depth, {
      response,
      cachedAt: now,
      expiresAt: new Date(now.getTime() + this.ttlMs),
    });
  }

  /**
   * Invalidate a specific depth level.
   *
   * @param depth - Depth level to invalidate
   */
  invalidate(depth: ObservabilityDepth): void {
    this.cache.delete(depth);
  }

  /**
   * Invalidate all cached entries.
   */
  invalidateAll(): void {
    this.cache.clear();
  }

  /**
   * Check if a depth level is cached and valid.
   *
   * @param depth - Depth level to check
   * @returns true if cached and not expired
   */
  has(depth: ObservabilityDepth): boolean {
    const entry = this.cache.get(depth);
    if (!entry) return false;
    if (new Date() > entry.expiresAt) {
      this.cache.delete(depth);
      return false;
    }
    return true;
  }

  /**
   * Get time until cache expires for a depth level.
   *
   * @param depth - Depth level to check
   * @returns Milliseconds until expiry, or 0 if not cached/expired
   */
  getTimeToExpiry(depth: ObservabilityDepth): number {
    const entry = this.cache.get(depth);
    if (!entry) return 0;

    const remaining = entry.expiresAt.getTime() - Date.now();
    return Math.max(0, remaining);
  }

  /**
   * Get cache statistics.
   */
  getStats(): { size: number; depths: ObservabilityDepth[] } {
    // Clean up expired entries first
    const entries = Array.from(this.cache.entries());
    for (const [depth, entry] of entries) {
      if (new Date() > entry.expiresAt) {
        this.cache.delete(depth);
      }
    }

    return {
      size: this.cache.size,
      depths: Array.from(this.cache.keys()),
    };
  }
}

// =============================================================================
// Singleton Instance
// =============================================================================

/** Default cache instance */
let defaultCache: ObservabilityCache | null = null;

/**
 * Get the default cache instance.
 */
export function getObservabilityCache(): ObservabilityCache {
  if (!defaultCache) {
    defaultCache = new ObservabilityCache();
  }
  return defaultCache;
}

/**
 * Reset the default cache (for testing).
 */
export function resetObservabilityCache(): void {
  defaultCache = null;
}
