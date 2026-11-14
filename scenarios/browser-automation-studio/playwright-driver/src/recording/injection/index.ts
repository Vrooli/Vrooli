export {
  type InjectionStrategy,
  type InjectionStrategyName,
  type InjectionResult,
  type InjectionStrategyStats,
  type InjectionStrategyOptions,
  createInitialStats,
  cloneStats,
  updateStats,
  resetStats,
} from './types';

export {
  InitScriptInjectionStrategy,
  createInitScriptInjectionStrategy,
} from './strategies/init-script-injection';

export {
  isDiagnosticsEnabled,
  INJECTION_DIAGNOSTICS_ENV_VAR,
} from './factory';
