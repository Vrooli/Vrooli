/**
 * Injection Strategy Unit Tests
 *
 * Tests for individual injection strategy implementations.
 * These tests focus on interface compliance and basic functionality
 * without requiring a real browser.
 */

import {
  InitScriptInjectionStrategy,
  createInitScriptInjectionStrategy,
  CDPInjectionStrategy,
  createCDPInjectionStrategy,
  RouteInjectionStrategy,
  createRouteInjectionStrategy,
} from '../../../src/recording/injection/strategies';
import type { InjectionStrategy, InjectionStrategyName } from '../../../src/recording/injection/types';

describe('InitScriptInjectionStrategy', () => {
  let strategy: InitScriptInjectionStrategy;

  beforeEach(() => {
    strategy = new InitScriptInjectionStrategy();
  });

  describe('interface compliance', () => {
    it('should have correct name', () => {
      expect(strategy.name).toBe('init-script');
    });

    it('should implement all required methods', () => {
      expect(typeof strategy.initialize).toBe('function');
      expect(typeof strategy.injectScript).toBe('function');
      expect(typeof strategy.verify).toBe('function');
      expect(typeof strategy.getStats).toBe('function');
      expect(typeof strategy.resetStats).toBe('function');
      expect(typeof strategy.cleanup).toBe('function');
      expect(typeof strategy.supportsProvider).toBe('function');
    });
  });

  describe('stats management', () => {
    it('should return initial stats', () => {
      const stats = strategy.getStats();

      expect(stats.attempted).toBe(0);
      expect(stats.successful).toBe(0);
      expect(stats.failed).toBe(0);
      expect(stats.avgInjectionTimeMs).toBe(0);
      expect(stats.lastInjectionAt).toBeNull();
    });

    it('should reset stats', () => {
      // Get stats and modify through internal mechanism
      const initialStats = strategy.getStats();
      expect(initialStats.attempted).toBe(0);

      strategy.resetStats();

      const afterReset = strategy.getStats();
      expect(afterReset.attempted).toBe(0);
    });
  });

  describe('supportsProvider', () => {
    it('should support rebrowser-playwright', () => {
      expect(strategy.supportsProvider('rebrowser-playwright')).toBe(true);
    });

    it('should support standard playwright', () => {
      expect(strategy.supportsProvider('playwright')).toBe(true);
    });

    it('should support any provider (universal strategy)', () => {
      expect(strategy.supportsProvider('firefox')).toBe(true);
      expect(strategy.supportsProvider('webkit')).toBe(true);
    });
  });

  describe('cleanup', () => {
    it('should complete without error before initialization', async () => {
      await expect(strategy.cleanup()).resolves.not.toThrow();
    });
  });
});

describe('CDPInjectionStrategy', () => {
  let strategy: CDPInjectionStrategy;

  beforeEach(() => {
    strategy = new CDPInjectionStrategy();
  });

  describe('interface compliance', () => {
    it('should have correct name', () => {
      expect(strategy.name).toBe('cdp-injection');
    });

    it('should implement all required methods', () => {
      expect(typeof strategy.initialize).toBe('function');
      expect(typeof strategy.injectScript).toBe('function');
      expect(typeof strategy.verify).toBe('function');
      expect(typeof strategy.getStats).toBe('function');
      expect(typeof strategy.resetStats).toBe('function');
      expect(typeof strategy.cleanup).toBe('function');
      expect(typeof strategy.supportsProvider).toBe('function');
    });
  });

  describe('stats management', () => {
    it('should return initial stats', () => {
      const stats = strategy.getStats();

      expect(stats.attempted).toBe(0);
      expect(stats.successful).toBe(0);
      expect(stats.failed).toBe(0);
    });

    it('should reset stats', () => {
      strategy.resetStats();
      const stats = strategy.getStats();
      expect(stats.attempted).toBe(0);
    });
  });

  describe('supportsProvider', () => {
    it('should support rebrowser-playwright', () => {
      expect(strategy.supportsProvider('rebrowser-playwright')).toBe(true);
    });

    it('should support chromium', () => {
      expect(strategy.supportsProvider('chromium')).toBe(true);
    });

    it('should NOT support firefox (CDP is Chromium-only)', () => {
      expect(strategy.supportsProvider('firefox')).toBe(false);
    });

    it('should NOT support webkit (CDP is Chromium-only)', () => {
      expect(strategy.supportsProvider('webkit')).toBe(false);
    });
  });

  describe('cleanup', () => {
    it('should complete without error before initialization', async () => {
      await expect(strategy.cleanup()).resolves.not.toThrow();
    });
  });
});

describe('RouteInjectionStrategy', () => {
  let strategy: RouteInjectionStrategy;

  beforeEach(() => {
    strategy = new RouteInjectionStrategy();
  });

  describe('interface compliance', () => {
    it('should have correct name', () => {
      expect(strategy.name).toBe('route-injection');
    });

    it('should implement all required methods', () => {
      expect(typeof strategy.initialize).toBe('function');
      expect(typeof strategy.injectScript).toBe('function');
      expect(typeof strategy.verify).toBe('function');
      expect(typeof strategy.getStats).toBe('function');
      expect(typeof strategy.resetStats).toBe('function');
      expect(typeof strategy.cleanup).toBe('function');
      expect(typeof strategy.supportsProvider).toBe('function');
    });
  });

  describe('stats management', () => {
    it('should return initial stats', () => {
      const stats = strategy.getStats();

      expect(stats.attempted).toBe(0);
      expect(stats.successful).toBe(0);
      expect(stats.failed).toBe(0);
    });

    it('should reset stats', () => {
      strategy.resetStats();
      const stats = strategy.getStats();
      expect(stats.attempted).toBe(0);
    });
  });

  describe('supportsProvider', () => {
    it('should support standard playwright', () => {
      expect(strategy.supportsProvider('playwright')).toBe(true);
    });

    it('should NOT support rebrowser-playwright (route interception broken)', () => {
      expect(strategy.supportsProvider('rebrowser-playwright')).toBe(false);
    });
  });

  describe('cleanup', () => {
    it('should complete without error before initialization', async () => {
      await expect(strategy.cleanup()).resolves.not.toThrow();
    });
  });
});

describe('Factory Functions', () => {
  it('createInitScriptInjectionStrategy should create correct type', () => {
    const strategy = createInitScriptInjectionStrategy();
    expect(strategy).toBeInstanceOf(InitScriptInjectionStrategy);
    expect(strategy.name).toBe('init-script');
  });

  it('createCDPInjectionStrategy should create correct type', () => {
    const strategy = createCDPInjectionStrategy();
    expect(strategy).toBeInstanceOf(CDPInjectionStrategy);
    expect(strategy.name).toBe('cdp-injection');
  });

  it('createRouteInjectionStrategy should create correct type', () => {
    const strategy = createRouteInjectionStrategy();
    expect(strategy).toBeInstanceOf(RouteInjectionStrategy);
    expect(strategy.name).toBe('route-injection');
  });
});

describe('Strategy Interface Conformance', () => {
  const strategies: Array<[InjectionStrategyName, () => InjectionStrategy]> = [
    ['init-script', () => new InitScriptInjectionStrategy()],
    ['cdp-injection', () => new CDPInjectionStrategy()],
    ['route-injection', () => new RouteInjectionStrategy()],
  ];

  describe.each(strategies)('%s strategy', (name, createStrategy) => {
    let strategy: InjectionStrategy;

    beforeEach(() => {
      strategy = createStrategy();
    });

    it('should have the correct name', () => {
      expect(strategy.name).toBe(name);
    });

    it('should return stats object with required fields', () => {
      const stats = strategy.getStats();

      expect(typeof stats.attempted).toBe('number');
      expect(typeof stats.successful).toBe('number');
      expect(typeof stats.failed).toBe('number');
      expect(typeof stats.avgInjectionTimeMs).toBe('number');
      expect(stats.lastInjectionAt === null || typeof stats.lastInjectionAt === 'string').toBe(true);
    });

    it('should return independent stats on each call', () => {
      const stats1 = strategy.getStats();
      const stats2 = strategy.getStats();

      // Modifying one should not affect the other
      (stats1 as any).attempted = 999;
      expect(stats2.attempted).toBe(0);
    });

    it('should have cleanup that resolves', async () => {
      await expect(strategy.cleanup()).resolves.not.toThrow();
    });

    it('should return boolean from supportsProvider', () => {
      expect(typeof strategy.supportsProvider('test-provider')).toBe('boolean');
    });
  });
});
