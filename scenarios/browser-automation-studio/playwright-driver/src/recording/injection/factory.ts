/**
 * Injection Strategy Factory
 *
 * Factory for creating injection strategies based on environment and configuration.
 *
 * ## Strategy Selection
 *
 * The factory selects strategies based on:
 * 1. `INJECTION_STRATEGY` environment variable (highest priority)
 * 2. Explicit `strategyName` option
 * 3. Provider capabilities (auto-select for rebrowser-playwright)
 *
 * ## Environment Variable
 *
 * Set `INJECTION_STRATEGY` to force a specific strategy:
 * - `init-script` - RECOMMENDED for rebrowser-playwright
 * - `cdp-injection` - Fallback with full CDP control
 * - `route-injection` - Legacy, standard Playwright only
 * - `auto` - Auto-detect based on provider
 *
 * ## Usage
 *
 * ```typescript
 * // Auto-select based on provider
 * const strategy = createInjectionStrategy({
 *   providerName: 'rebrowser-playwright',
 *   logger: createLogger(),
 * });
 *
 * // Force a specific strategy
 * const strategy = createInjectionStrategy({
 *   strategyName: 'init-script',
 *   logger: createLogger(),
 * });
 * ```
 *
 * @module recording/injection/factory
 */

import type winston from 'winston';
import {
  type InjectionStrategy,
  type InjectionStrategyName,
  type InjectionStrategyFactoryOptions,
} from './types';
import {
  InitScriptInjectionStrategy,
  CDPInjectionStrategy,
  RouteInjectionStrategy,
} from './strategies';
import { logger as defaultLogger, LogContext, scopedLog } from '../../utils';

// =============================================================================
// Environment Variable
// =============================================================================

/**
 * Environment variable name for strategy selection.
 */
export const INJECTION_STRATEGY_ENV_VAR = 'INJECTION_STRATEGY';

/**
 * Environment variable for enabling diagnostics.
 */
export const INJECTION_DIAGNOSTICS_ENV_VAR = 'INJECTION_DIAGNOSTICS';

/**
 * Get strategy name from environment variable.
 */
export function getStrategyFromEnv(): InjectionStrategyName | 'auto' | null {
  const envValue = process.env[INJECTION_STRATEGY_ENV_VAR];
  if (!envValue) return null;

  const normalizedValue = envValue.toLowerCase().trim();

  if (normalizedValue === 'auto') return 'auto';
  if (normalizedValue === 'init-script' || normalizedValue === 'initscript') return 'init-script';
  if (normalizedValue === 'cdp-injection' || normalizedValue === 'cdp') return 'cdp-injection';
  if (normalizedValue === 'route-injection' || normalizedValue === 'route') return 'route-injection';

  return null;
}

/**
 * Check if diagnostics are enabled via environment variable.
 */
export function isDiagnosticsEnabled(): boolean {
  const envValue = process.env[INJECTION_DIAGNOSTICS_ENV_VAR];
  return envValue === 'true' || envValue === '1';
}

// =============================================================================
// Strategy Selection
// =============================================================================

/**
 * Default strategy order for auto-detection.
 *
 * Order is based on reliability and performance:
 * 1. init-script - Most reliable for rebrowser-playwright
 * 2. cdp-injection - Fallback with full control
 * 3. route-injection - Legacy, may not work with rebrowser
 */
export const DEFAULT_STRATEGY_ORDER: InjectionStrategyName[] = [
  'init-script',
  'cdp-injection',
  'route-injection',
];

/**
 * Select the best strategy for a given provider.
 *
 * @param providerName - Name of the Playwright provider
 * @returns Recommended strategy name
 */
export function selectStrategyForProvider(providerName: string): InjectionStrategyName {
  const lowerName = providerName.toLowerCase();

  // For rebrowser-playwright, init-script is REQUIRED
  // Route interception is broken with anti-detection patches
  if (lowerName.includes('rebrowser')) {
    return 'init-script';
  }

  // For standard Playwright, init-script still works best
  // but route-injection is also supported
  return 'init-script';
}

// =============================================================================
// Factory
// =============================================================================

/**
 * Injection Strategy Factory
 *
 * Creates and manages injection strategies based on configuration.
 */
export class InjectionStrategyFactory {
  private readonly logger: winston.Logger;

  constructor(logger?: winston.Logger) {
    this.logger = logger ?? defaultLogger;
  }

  /**
   * Create an injection strategy based on options.
   *
   * Selection priority:
   * 1. INJECTION_STRATEGY environment variable
   * 2. Explicit strategyName option
   * 3. Provider-based auto-selection
   *
   * @param options - Factory options
   * @returns Created strategy instance
   */
  create(options: InjectionStrategyFactoryOptions = {}): InjectionStrategy {
    // Priority 1: Environment variable
    const envStrategy = getStrategyFromEnv();
    if (envStrategy && envStrategy !== 'auto') {
      this.logger.info(scopedLog(LogContext.INJECTION, 'using strategy from environment variable'), {
        strategy: envStrategy,
        envVar: INJECTION_STRATEGY_ENV_VAR,
      });
      return this.createByName(envStrategy);
    }

    // Priority 2: Explicit option
    const requestedStrategy = options.strategyName;
    if (requestedStrategy && requestedStrategy !== 'auto') {
      this.logger.info(scopedLog(LogContext.INJECTION, 'using explicitly requested strategy'), {
        strategy: requestedStrategy,
      });
      return this.createByName(requestedStrategy);
    }

    // Priority 3: Provider-based selection
    const providerName = options.providerName ?? 'rebrowser-playwright';
    const selectedStrategy = selectStrategyForProvider(providerName);

    this.logger.info(scopedLog(LogContext.INJECTION, 'auto-selected strategy for provider'), {
      strategy: selectedStrategy,
      provider: providerName,
    });

    return this.createByName(selectedStrategy);
  }

  /**
   * Create a strategy by name.
   *
   * @param name - Strategy name
   * @returns Strategy instance
   * @throws Error if strategy name is unknown
   */
  createByName(name: InjectionStrategyName): InjectionStrategy {
    switch (name) {
      case 'init-script':
        return new InitScriptInjectionStrategy();
      case 'cdp-injection':
        return new CDPInjectionStrategy();
      case 'route-injection':
        return new RouteInjectionStrategy();
      default:
        throw new Error(`Unknown injection strategy: ${name}`);
    }
  }

  /**
   * Get all available strategy names.
   */
  getAvailableStrategies(): InjectionStrategyName[] {
    return ['init-script', 'cdp-injection', 'route-injection'];
  }

  /**
   * Check if a strategy supports a given provider.
   *
   * @param strategyName - Strategy to check
   * @param providerName - Provider to check against
   * @returns True if strategy supports the provider
   */
  strategySupportsProvider(strategyName: InjectionStrategyName, providerName: string): boolean {
    const strategy = this.createByName(strategyName);
    return strategy.supportsProvider(providerName);
  }
}

/**
 * Create an injection strategy using the default factory.
 *
 * @param options - Factory options
 * @returns Created strategy instance
 */
export function createInjectionStrategy(options: InjectionStrategyFactoryOptions = {}): InjectionStrategy {
  const factory = new InjectionStrategyFactory(options.logger);
  return factory.create(options);
}

/**
 * Create an injection strategy by name.
 *
 * @param name - Strategy name
 * @returns Strategy instance
 */
export function createInjectionStrategyByName(name: InjectionStrategyName): InjectionStrategy {
  const factory = new InjectionStrategyFactory();
  return factory.createByName(name);
}
