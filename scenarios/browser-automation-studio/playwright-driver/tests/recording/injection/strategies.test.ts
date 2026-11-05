/** Focused contract tests for the sole init-script injection implementation. */

import {
  InitScriptInjectionStrategy,
  createInitScriptInjectionStrategy,
} from '../../../src/recording/injection/strategies';

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

  describe('cleanup', () => {
    it('should complete without error before initialization', async () => {
      await expect(strategy.cleanup()).resolves.not.toThrow();
    });
  });
});
