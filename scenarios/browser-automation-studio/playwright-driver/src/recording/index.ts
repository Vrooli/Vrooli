/**
 * Recording Module
 *
 * STABILITY: VOLATILE (by design)
 *
 * Public API for Record Mode functionality.
 *
 * CHANGE AXIS: Recording Action Types
 * Primary extension points:
 * - action-types.ts: Type normalization, kind mapping, confidence calculation
 * - action-executor.ts: Replay execution for each action type
 *
 * When adding a new recorded action type:
 * 1. Add to ACTION_TYPES in action-types.ts
 * 2. Add alias mapping if needed (e.g., 'input' -> 'type')
 * 3. Add proto kind mapping (ACTION_KIND_MAP)
 * 4. Add to SELECTOR_OPTIONAL_ACTIONS if no selector required
 * 5. Update buildTypedActionPayload() for typed payloads
 * 6. Register executor in action-executor.ts using registerActionExecutor()
 * 7. Add tests in tests/unit/recording/action-executor.test.ts
 *
 * CHANGE AXIS: Selector Strategies
 * Primary extension point: selector-config.ts (config) + selectors.ts (reference) + injector.ts (runtime)
 *
 * When adding a new selector strategy:
 * 1. Add to SELECTOR_STRATEGIES in selector-config.ts
 * 2. Update confidence/specificity scores in selector-config.ts
 * 3. Implement strategy in selectors.ts (for documentation/testing)
 * 4. Update injector.ts if the algorithm changes (config is auto-injected)
 *
 * Note: selector-config.ts is the SINGLE SOURCE OF TRUTH for configuration.
 * The injector uses serializeConfigForBrowser() to inject config into pages.
 */

export * from './types';
export * from './action-types';
export * from './selector-config';
// Note: action-executor re-exports ActionType from types, so we selectively export
export {
  registerActionExecutor,
  getActionExecutor,
  hasActionExecutor,
  getRegisteredActionTypes,
  type ActionExecutor,
  type ActionExecutorContext,
} from './action-executor';
export * from './selectors';
export * from './injector';
export * from './controller';
export * from './buffer';
