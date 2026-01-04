/**
 * Unit Tests for Observability Cache
 *
 * Tests the time-based caching for observability responses.
 */

import { describe, beforeEach, it, expect, jest } from '@jest/globals';
import {
  ObservabilityCache,
  getObservabilityCache,
  resetObservabilityCache,
  DEFAULT_CACHE_TTL_MS,
} from '../../../src/observability/cache';
import type { ObservabilityResponse } from '../../../src/observability/types';

// Helper to create a mock response
function createMockResponse(overrides: Partial<ObservabilityResponse> = {}): ObservabilityResponse {
  return {
    status: 'ok',
    ready: true,
    timestamp: new Date().toISOString(),
    version: '2.0.0',
    uptime_ms: 123456,
    depth: 'quick',
    summary: {
      sessions: 5,
      recordings: 1,
      browser_connected: true,
    },
    ...overrides,
  };
}

describe('ObservabilityCache', () => {
  let cache: ObservabilityCache;

  beforeEach(() => {
    cache = new ObservabilityCache(1000); // 1 second TTL for tests
  });

  describe('get', () => {
    it('should return undefined for empty cache', () => {
      expect(cache.get('quick')).toBeUndefined();
    });

    it('should return cached response within TTL', () => {
      const response = createMockResponse();
      cache.set('quick', response);

      const cached = cache.get('quick');

      expect(cached).toBeDefined();
      expect(cached?.status).toBe('ok');
      expect(cached?.cached).toBe(true);
      expect(cached?.cached_at).toBeDefined();
      expect(cached?.cache_ttl_ms).toBe(1000);
    });

    it('should return undefined after TTL expires', async () => {
      const response = createMockResponse();
      const shortCache = new ObservabilityCache(50); // 50ms TTL
      shortCache.set('quick', response);

      // Wait for TTL to expire
      await new Promise((resolve) => setTimeout(resolve, 60));

      expect(shortCache.get('quick')).toBeUndefined();
    });

    it('should cache different depths independently', () => {
      const quickResponse = createMockResponse({ depth: 'quick' });
      const standardResponse = createMockResponse({ depth: 'standard' });

      cache.set('quick', quickResponse);
      cache.set('standard', standardResponse);

      const cachedQuick = cache.get('quick');
      const cachedStandard = cache.get('standard');

      expect(cachedQuick?.depth).toBe('quick');
      expect(cachedStandard?.depth).toBe('standard');
    });
  });

  describe('set', () => {
    it('should store response in cache', () => {
      const response = createMockResponse();
      cache.set('quick', response);

      expect(cache.has('quick')).toBe(true);
    });

    it('should overwrite existing entry', () => {
      const response1 = createMockResponse({ status: 'ok' });
      const response2 = createMockResponse({ status: 'degraded' });

      cache.set('quick', response1);
      cache.set('quick', response2);

      const cached = cache.get('quick');
      expect(cached?.status).toBe('degraded');
    });
  });

  describe('invalidate', () => {
    it('should remove specific depth from cache', () => {
      cache.set('quick', createMockResponse());
      cache.set('standard', createMockResponse());

      cache.invalidate('quick');

      expect(cache.has('quick')).toBe(false);
      expect(cache.has('standard')).toBe(true);
    });

    it('should not throw for non-existent depth', () => {
      expect(() => cache.invalidate('deep')).not.toThrow();
    });
  });

  describe('invalidateAll', () => {
    it('should clear all cache entries', () => {
      cache.set('quick', createMockResponse());
      cache.set('standard', createMockResponse());
      cache.set('deep', createMockResponse());

      cache.invalidateAll();

      expect(cache.has('quick')).toBe(false);
      expect(cache.has('standard')).toBe(false);
      expect(cache.has('deep')).toBe(false);
    });
  });

  describe('has', () => {
    it('should return true for valid cached entry', () => {
      cache.set('quick', createMockResponse());
      expect(cache.has('quick')).toBe(true);
    });

    it('should return false for non-existent entry', () => {
      expect(cache.has('quick')).toBe(false);
    });

    it('should return false and clean up expired entry', async () => {
      const shortCache = new ObservabilityCache(50);
      shortCache.set('quick', createMockResponse());

      await new Promise((resolve) => setTimeout(resolve, 60));

      expect(shortCache.has('quick')).toBe(false);
    });
  });

  describe('getTimeToExpiry', () => {
    it('should return 0 for non-existent entry', () => {
      expect(cache.getTimeToExpiry('quick')).toBe(0);
    });

    it('should return positive value for valid entry', () => {
      cache.set('quick', createMockResponse());
      expect(cache.getTimeToExpiry('quick')).toBeGreaterThan(0);
    });

    it('should return 0 for expired entry', async () => {
      const shortCache = new ObservabilityCache(50);
      shortCache.set('quick', createMockResponse());

      await new Promise((resolve) => setTimeout(resolve, 60));

      expect(shortCache.getTimeToExpiry('quick')).toBe(0);
    });
  });

  describe('getStats', () => {
    it('should return empty stats for new cache', () => {
      const stats = cache.getStats();
      expect(stats.size).toBe(0);
      expect(stats.depths).toEqual([]);
    });

    it('should return correct stats with entries', () => {
      cache.set('quick', createMockResponse());
      cache.set('standard', createMockResponse());

      const stats = cache.getStats();
      expect(stats.size).toBe(2);
      expect(stats.depths).toContain('quick');
      expect(stats.depths).toContain('standard');
    });

    it('should not count expired entries', async () => {
      const shortCache = new ObservabilityCache(50);
      shortCache.set('quick', createMockResponse());
      shortCache.set('standard', createMockResponse());

      await new Promise((resolve) => setTimeout(resolve, 60));

      const stats = shortCache.getStats();
      expect(stats.size).toBe(0);
    });
  });
});

describe('Default cache instance', () => {
  beforeEach(() => {
    resetObservabilityCache();
  });

  it('should return singleton instance', () => {
    const cache1 = getObservabilityCache();
    const cache2 = getObservabilityCache();

    expect(cache1).toBe(cache2);
  });

  it('should reset singleton on resetObservabilityCache', () => {
    const cache1 = getObservabilityCache();
    cache1.set('quick', createMockResponse());

    resetObservabilityCache();

    const cache2 = getObservabilityCache();
    expect(cache2.has('quick')).toBe(false);
  });
});

describe('DEFAULT_CACHE_TTL_MS', () => {
  it('should be 30 seconds', () => {
    expect(DEFAULT_CACHE_TTL_MS).toBe(30_000);
  });
});
