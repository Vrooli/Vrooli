/**
 * Injection Strategy Factory Tests
 *
 * Tests for the injection strategy factory that creates and selects
 * appropriate injection strategies based on configuration.
 */

import {
  InjectionStrategyFactory,
  createInjectionStrategy,
  createInjectionStrategyByName,
  selectStrategyForProvider,
  getStrategyFromEnv,
  INJECTION_STRATEGY_ENV_VAR,
  DEFAULT_STRATEGY_ORDER,
} from '../../../src/recording/injection/factory';
import {
  InitScriptInjectionStrategy,
  CDPInjectionStrategy,
  RouteInjectionStrategy,
} from '../../../src/recording/injection/strategies';

describe('InjectionStrategyFactory', () => {
  let originalEnv: string | undefined;

  beforeEach(() => {
    // Save original env value
    originalEnv = process.env[INJECTION_STRATEGY_ENV_VAR];
    delete process.env[INJECTION_STRATEGY_ENV_VAR];
  });

  afterEach(() => {
    // Restore original env value
    if (originalEnv !== undefined) {
      process.env[INJECTION_STRATEGY_ENV_VAR] = originalEnv;
    } else {
      delete process.env[INJECTION_STRATEGY_ENV_VAR];
    }
  });

  describe('createByName', () => {
    it('should create init-script strategy', () => {
      const factory = new InjectionStrategyFactory();
      const strategy = factory.createByName('init-script');

      expect(strategy).toBeInstanceOf(InitScriptInjectionStrategy);
      expect(strategy.name).toBe('init-script');
    });

    it('should create cdp-injection strategy', () => {
      const factory = new InjectionStrategyFactory();
      const strategy = factory.createByName('cdp-injection');

      expect(strategy).toBeInstanceOf(CDPInjectionStrategy);
      expect(strategy.name).toBe('cdp-injection');
    });

    it('should create route-injection strategy', () => {
      const factory = new InjectionStrategyFactory();
      const strategy = factory.createByName('route-injection');

      expect(strategy).toBeInstanceOf(RouteInjectionStrategy);
      expect(strategy.name).toBe('route-injection');
    });

    it('should throw for unknown strategy name', () => {
      const factory = new InjectionStrategyFactory();

      expect(() => {
        factory.createByName('unknown' as any);
      }).toThrow('Unknown injection strategy: unknown');
    });
  });

  describe('create', () => {
    it('should use env variable when set to init-script', () => {
      process.env[INJECTION_STRATEGY_ENV_VAR] = 'init-script';

      const factory = new InjectionStrategyFactory();
      const strategy = factory.create();

      expect(strategy.name).toBe('init-script');
    });

    it('should use env variable when set to cdp', () => {
      process.env[INJECTION_STRATEGY_ENV_VAR] = 'cdp';

      const factory = new InjectionStrategyFactory();
      const strategy = factory.create();

      expect(strategy.name).toBe('cdp-injection');
    });

    it('should use explicit strategyName option', () => {
      const factory = new InjectionStrategyFactory();
      const strategy = factory.create({ strategyName: 'cdp-injection' });

      expect(strategy.name).toBe('cdp-injection');
    });

    it('should select init-script for rebrowser-playwright provider', () => {
      const factory = new InjectionStrategyFactory();
      const strategy = factory.create({ providerName: 'rebrowser-playwright' });

      expect(strategy.name).toBe('init-script');
    });

    it('should env variable takes priority over strategyName option', () => {
      process.env[INJECTION_STRATEGY_ENV_VAR] = 'cdp-injection';

      const factory = new InjectionStrategyFactory();
      const strategy = factory.create({ strategyName: 'init-script' });

      // Env var should win
      expect(strategy.name).toBe('cdp-injection');
    });

    it('should default to init-script for auto on unknown provider', () => {
      const factory = new InjectionStrategyFactory();
      const strategy = factory.create({ strategyName: 'auto' });

      // Default provider is rebrowser-playwright, which selects init-script
      expect(strategy.name).toBe('init-script');
    });
  });

  describe('getAvailableStrategies', () => {
    it('should return all available strategy names', () => {
      const factory = new InjectionStrategyFactory();
      const strategies = factory.getAvailableStrategies();

      expect(strategies).toContain('init-script');
      expect(strategies).toContain('cdp-injection');
      expect(strategies).toContain('route-injection');
      expect(strategies).toHaveLength(3);
    });
  });

  describe('strategySupportsProvider', () => {
    it('should return true for init-script on rebrowser-playwright', () => {
      const factory = new InjectionStrategyFactory();
      const supports = factory.strategySupportsProvider('init-script', 'rebrowser-playwright');

      expect(supports).toBe(true);
    });

    it('should return false for route-injection on rebrowser-playwright', () => {
      const factory = new InjectionStrategyFactory();
      const supports = factory.strategySupportsProvider('route-injection', 'rebrowser-playwright');

      expect(supports).toBe(false);
    });

    it('should return false for cdp-injection on firefox', () => {
      const factory = new InjectionStrategyFactory();
      const supports = factory.strategySupportsProvider('cdp-injection', 'firefox');

      expect(supports).toBe(false);
    });
  });
});

describe('getStrategyFromEnv', () => {
  let originalEnv: string | undefined;

  beforeEach(() => {
    originalEnv = process.env[INJECTION_STRATEGY_ENV_VAR];
    delete process.env[INJECTION_STRATEGY_ENV_VAR];
  });

  afterEach(() => {
    if (originalEnv !== undefined) {
      process.env[INJECTION_STRATEGY_ENV_VAR] = originalEnv;
    } else {
      delete process.env[INJECTION_STRATEGY_ENV_VAR];
    }
  });

  it('should return null when env is not set', () => {
    expect(getStrategyFromEnv()).toBeNull();
  });

  it('should return auto for "auto"', () => {
    process.env[INJECTION_STRATEGY_ENV_VAR] = 'auto';
    expect(getStrategyFromEnv()).toBe('auto');
  });

  it('should return init-script for "init-script"', () => {
    process.env[INJECTION_STRATEGY_ENV_VAR] = 'init-script';
    expect(getStrategyFromEnv()).toBe('init-script');
  });

  it('should return init-script for "initscript" (alternate spelling)', () => {
    process.env[INJECTION_STRATEGY_ENV_VAR] = 'initscript';
    expect(getStrategyFromEnv()).toBe('init-script');
  });

  it('should return cdp-injection for "cdp"', () => {
    process.env[INJECTION_STRATEGY_ENV_VAR] = 'cdp';
    expect(getStrategyFromEnv()).toBe('cdp-injection');
  });

  it('should return cdp-injection for "cdp-injection"', () => {
    process.env[INJECTION_STRATEGY_ENV_VAR] = 'cdp-injection';
    expect(getStrategyFromEnv()).toBe('cdp-injection');
  });

  it('should return route-injection for "route"', () => {
    process.env[INJECTION_STRATEGY_ENV_VAR] = 'route';
    expect(getStrategyFromEnv()).toBe('route-injection');
  });

  it('should return null for unknown value', () => {
    process.env[INJECTION_STRATEGY_ENV_VAR] = 'unknown';
    expect(getStrategyFromEnv()).toBeNull();
  });

  it('should handle case-insensitive values', () => {
    process.env[INJECTION_STRATEGY_ENV_VAR] = 'INIT-SCRIPT';
    expect(getStrategyFromEnv()).toBe('init-script');
  });

  it('should trim whitespace', () => {
    process.env[INJECTION_STRATEGY_ENV_VAR] = '  init-script  ';
    expect(getStrategyFromEnv()).toBe('init-script');
  });
});

describe('selectStrategyForProvider', () => {
  it('should select init-script for rebrowser-playwright', () => {
    expect(selectStrategyForProvider('rebrowser-playwright')).toBe('init-script');
  });

  it('should select init-script for Rebrowser-Playwright (case insensitive)', () => {
    expect(selectStrategyForProvider('Rebrowser-Playwright')).toBe('init-script');
  });

  it('should select init-script for standard playwright', () => {
    expect(selectStrategyForProvider('playwright')).toBe('init-script');
  });
});

describe('DEFAULT_STRATEGY_ORDER', () => {
  it('should have init-script first (most reliable)', () => {
    expect(DEFAULT_STRATEGY_ORDER[0]).toBe('init-script');
  });

  it('should have cdp-injection second (fallback)', () => {
    expect(DEFAULT_STRATEGY_ORDER[1]).toBe('cdp-injection');
  });

  it('should have route-injection last (legacy)', () => {
    expect(DEFAULT_STRATEGY_ORDER[2]).toBe('route-injection');
  });
});

describe('convenience functions', () => {
  let originalEnv: string | undefined;

  beforeEach(() => {
    originalEnv = process.env[INJECTION_STRATEGY_ENV_VAR];
    delete process.env[INJECTION_STRATEGY_ENV_VAR];
  });

  afterEach(() => {
    if (originalEnv !== undefined) {
      process.env[INJECTION_STRATEGY_ENV_VAR] = originalEnv;
    } else {
      delete process.env[INJECTION_STRATEGY_ENV_VAR];
    }
  });

  describe('createInjectionStrategy', () => {
    it('should create strategy with default options', () => {
      const strategy = createInjectionStrategy();
      expect(strategy.name).toBe('init-script');
    });

    it('should create strategy with explicit name', () => {
      const strategy = createInjectionStrategy({ strategyName: 'cdp-injection' });
      expect(strategy.name).toBe('cdp-injection');
    });
  });

  describe('createInjectionStrategyByName', () => {
    it('should create init-script strategy', () => {
      const strategy = createInjectionStrategyByName('init-script');
      expect(strategy).toBeInstanceOf(InitScriptInjectionStrategy);
    });

    it('should create cdp-injection strategy', () => {
      const strategy = createInjectionStrategyByName('cdp-injection');
      expect(strategy).toBeInstanceOf(CDPInjectionStrategy);
    });
  });
});
