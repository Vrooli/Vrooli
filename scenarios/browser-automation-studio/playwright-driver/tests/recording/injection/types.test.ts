/**
 * Injection Strategy Types Tests
 *
 * Tests for the helper functions in the types module.
 */

import {
  createInitialStats,
  cloneStats,
  updateStats,
  resetStats,
  type InjectionStrategyStats,
} from '../../../src/recording/injection/types';

describe('createInitialStats', () => {
  it('should create stats with all values at zero', () => {
    const stats = createInitialStats();

    expect(stats.attempted).toBe(0);
    expect(stats.successful).toBe(0);
    expect(stats.failed).toBe(0);
    expect(stats.avgInjectionTimeMs).toBe(0);
    expect(stats.lastInjectionAt).toBeNull();
  });
});

describe('cloneStats', () => {
  it('should create an independent copy', () => {
    const original = createInitialStats();
    original.attempted = 5;
    original.successful = 3;

    const clone = cloneStats(original);

    // Modify original
    original.attempted = 10;

    // Clone should not be affected
    expect(clone.attempted).toBe(5);
  });

  it('should copy all properties', () => {
    const original: InjectionStrategyStats = {
      attempted: 10,
      successful: 7,
      failed: 3,
      avgInjectionTimeMs: 150.5,
      lastInjectionAt: '2024-01-15T10:30:00.000Z',
    };

    const clone = cloneStats(original);

    expect(clone.attempted).toBe(10);
    expect(clone.successful).toBe(7);
    expect(clone.failed).toBe(3);
    expect(clone.avgInjectionTimeMs).toBe(150.5);
    expect(clone.lastInjectionAt).toBe('2024-01-15T10:30:00.000Z');
  });
});

describe('updateStats', () => {
  it('should increment attempted on success', () => {
    const stats = createInitialStats();

    updateStats(stats, true, 100);

    expect(stats.attempted).toBe(1);
    expect(stats.successful).toBe(1);
    expect(stats.failed).toBe(0);
  });

  it('should increment attempted and failed on failure', () => {
    const stats = createInitialStats();

    updateStats(stats, false, 100);

    expect(stats.attempted).toBe(1);
    expect(stats.successful).toBe(0);
    expect(stats.failed).toBe(1);
  });

  it('should update avgInjectionTimeMs', () => {
    const stats = createInitialStats();

    updateStats(stats, true, 100);
    expect(stats.avgInjectionTimeMs).toBe(100);

    updateStats(stats, true, 200);
    expect(stats.avgInjectionTimeMs).toBe(150); // (100 + 200) / 2
  });

  it('should set lastInjectionAt to ISO timestamp', () => {
    const stats = createInitialStats();
    const before = new Date().toISOString();

    updateStats(stats, true, 100);

    const after = new Date().toISOString();

    expect(stats.lastInjectionAt).not.toBeNull();
    const lastInjectionAt = stats.lastInjectionAt;
    if (!lastInjectionAt) {
      throw new Error('Expected lastInjectionAt to be set');
    }
    expect(lastInjectionAt >= before).toBe(true);
    expect(lastInjectionAt <= after).toBe(true);
  });

  it('should handle mixed success and failure', () => {
    const stats = createInitialStats();

    updateStats(stats, true, 100);
    updateStats(stats, false, 200);
    updateStats(stats, true, 150);

    expect(stats.attempted).toBe(3);
    expect(stats.successful).toBe(2);
    expect(stats.failed).toBe(1);
    // Average: (100 + 200 + 150) / 3 = 150
    expect(stats.avgInjectionTimeMs).toBe(150);
  });
});

describe('resetStats', () => {
  it('should reset all values to initial state', () => {
    const stats: InjectionStrategyStats = {
      attempted: 10,
      successful: 7,
      failed: 3,
      avgInjectionTimeMs: 150.5,
      lastInjectionAt: '2024-01-15T10:30:00.000Z',
    };

    resetStats(stats);

    expect(stats.attempted).toBe(0);
    expect(stats.successful).toBe(0);
    expect(stats.failed).toBe(0);
    expect(stats.avgInjectionTimeMs).toBe(0);
    expect(stats.lastInjectionAt).toBeNull();
  });

  it('should work on already-empty stats', () => {
    const stats = createInitialStats();

    resetStats(stats);

    expect(stats.attempted).toBe(0);
    expect(stats.successful).toBe(0);
    expect(stats.failed).toBe(0);
    expect(stats.avgInjectionTimeMs).toBe(0);
    expect(stats.lastInjectionAt).toBeNull();
  });
});

describe('InjectionStrategyStats average calculation', () => {
  it('should calculate rolling average correctly over many updates', () => {
    const stats = createInitialStats();

    // Add 10 successful injections with varying times
    const times = [100, 200, 150, 180, 120, 160, 140, 190, 110, 130];
    const expectedAvg = times.reduce((a, b) => a + b, 0) / times.length;

    for (const time of times) {
      updateStats(stats, true, time);
    }

    expect(stats.avgInjectionTimeMs).toBeCloseTo(expectedAvg, 1);
  });
});
