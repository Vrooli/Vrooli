/**
 * Injection Strategy DI Architecture
 *
 * This module provides a dependency injection system with swappable injection
 * strategies for the recording system.
 *
 * ## Quick Start
 *
 * ```typescript
 * import { createInjectionStrategy } from './injection';
 *
 * // Create strategy (auto-selects based on provider)
 * const strategy = createInjectionStrategy({
 *   providerName: 'rebrowser-playwright',
 * });
 *
 * // Initialize on context
 * await strategy.initialize(context, {
 *   bindingName: '__vrooli_recordAction',
 *   logger: createLogger(),
 * });
 *
 * // Script is injected automatically on page creation
 * const page = await context.newPage();
 * await page.goto('https://example.com');
 *
 * // Verify it worked
 * const verified = await strategy.verify(page);
 * ```
 *
 * ## Available Strategies
 *
 * | Strategy | When to Use |
 * |----------|-------------|
 * | `init-script` | RECOMMENDED for rebrowser-playwright |
 * | `cdp-injection` | Fallback with full CDP control |
 * | `route-injection` | Legacy, standard Playwright only |
 *
 * ## Environment Variables
 *
 * | Variable | Values | Description |
 * |----------|--------|-------------|
 * | `INJECTION_STRATEGY` | `auto`, `init-script`, `cdp-injection`, `route-injection` | Force specific strategy |
 * | `INJECTION_DIAGNOSTICS` | `true`, `false` | Verbose injection logging |
 *
 * ## Strategy Selection Priority
 *
 * 1. `INJECTION_STRATEGY` environment variable
 * 2. Explicit `strategyName` option
 * 3. Provider capabilities (auto-select for rebrowser-playwright)
 *
 * @module recording/injection
 */

// Types
export {
  type InjectionStrategy,
  type InjectionStrategyName,
  type InjectionResult,
  type InjectionStrategyStats,
  type InjectionStrategyOptions,
  type InjectionStrategyFactoryOptions,
  type AutoDetectorOptions,
  type AutoDetectionResult,
  createInitialStats,
  cloneStats,
  updateStats,
  resetStats,
} from './types';

// Strategies
export {
  InitScriptInjectionStrategy,
  createInitScriptInjectionStrategy,
  CDPInjectionStrategy,
  createCDPInjectionStrategy,
  RouteInjectionStrategy,
  createRouteInjectionStrategy,
} from './strategies';

// Factory
export {
  InjectionStrategyFactory,
  createInjectionStrategy,
  createInjectionStrategyByName,
  getStrategyFromEnv,
  isDiagnosticsEnabled,
  selectStrategyForProvider,
  DEFAULT_STRATEGY_ORDER,
  INJECTION_STRATEGY_ENV_VAR,
  INJECTION_DIAGNOSTICS_ENV_VAR,
} from './factory';

// Auto-Detector
export {
  InjectionAutoDetector,
  createInjectionAutoDetector,
  detectWorkingStrategy,
} from './auto-detector';
