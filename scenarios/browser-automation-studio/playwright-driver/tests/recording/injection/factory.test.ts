import {
  createInitScriptInjectionStrategy,
  INJECTION_DIAGNOSTICS_ENV_VAR,
  isDiagnosticsEnabled,
} from '../../../src/recording/injection';
import { InitScriptInjectionStrategy } from '../../../src/recording/injection/strategies';

describe('single init-script injection construction', () => {
  it('constructs the init-script implementation directly', () => {
    expect(createInitScriptInjectionStrategy()).toBeInstanceOf(InitScriptInjectionStrategy);
  });
});

describe('injection diagnostics configuration', () => {
  const original = process.env[INJECTION_DIAGNOSTICS_ENV_VAR];

  afterEach(() => {
    if (original === undefined) delete process.env[INJECTION_DIAGNOSTICS_ENV_VAR];
    else process.env[INJECTION_DIAGNOSTICS_ENV_VAR] = original;
  });

  it.each(['true', '1'])('enables diagnostics for %s', (value) => {
    process.env[INJECTION_DIAGNOSTICS_ENV_VAR] = value;
    expect(isDiagnosticsEnabled()).toBe(true);
  });

  it.each(['false', '0', 'yes', ''])('keeps diagnostics disabled for %s', (value) => {
    process.env[INJECTION_DIAGNOSTICS_ENV_VAR] = value;
    expect(isDiagnosticsEnabled()).toBe(false);
  });
});
