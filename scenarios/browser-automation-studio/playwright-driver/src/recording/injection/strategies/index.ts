/**
 * Injection Strategies Index
 *
 * Exports all available injection strategies for the recording system.
 *
 * @module recording/injection/strategies
 */

// Init Script Strategy - RECOMMENDED for rebrowser-playwright
export {
  InitScriptInjectionStrategy,
  createInitScriptInjectionStrategy,
} from './init-script-injection';

// CDP Strategy - Fallback with full control
export {
  CDPInjectionStrategy,
  createCDPInjectionStrategy,
} from './cdp-injection';

// Route Strategy - Legacy, standard Playwright only
export {
  RouteInjectionStrategy,
  createRouteInjectionStrategy,
} from './route-injection';
