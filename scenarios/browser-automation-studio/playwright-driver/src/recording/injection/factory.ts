import { InitScriptInjectionStrategy } from './strategies/init-script-injection';

/** Environment switch for verbose injection diagnostics. */
export const INJECTION_DIAGNOSTICS_ENV_VAR = 'INJECTION_DIAGNOSTICS';

export function isDiagnosticsEnabled(): boolean {
  const value = process.env[INJECTION_DIAGNOSTICS_ENV_VAR];
  return value === 'true' || value === '1';
}

/** Construct the one supported recording injection implementation. */
export function createInitScriptInjectionStrategy(): InitScriptInjectionStrategy {
  return new InitScriptInjectionStrategy();
}
