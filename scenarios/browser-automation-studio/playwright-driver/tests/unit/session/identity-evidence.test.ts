import type { Browser, Page } from 'rebrowser-playwright';
import { observeEffectiveIdentity, type IdentityContextConfiguration } from '../../../src/session/identity-evidence';

const configuration: IdentityContextConfiguration = {
  userAgent: 'Mozilla/5.0 Chrome/140.0.0.0 Safari/537.36',
  locale: 'en-GB',
  timezoneId: 'Europe/London',
  colorScheme: 'dark',
  deviceScaleFactor: 1.25,
  viewport: { width: 1440, height: 900 },
};

function fakeOwners(observation: Record<string, unknown>): { page: Page; browser: Browser } {
  const page = {
    evaluate: jest.fn().mockResolvedValue(observation),
  } as unknown as Page;
  const browser = {
    browserType: () => ({ name: () => 'chromium' }),
    version: () => '140.0.7339.80',
  } as unknown as Browser;
  return { page, browser };
}

const matchingObservation = {
  userAgent: configuration.userAgent,
  locale: configuration.locale,
  localeCanonical: Intl.getCanonicalLocales(configuration.locale)[0],
  configuredLocaleCanonical: Intl.getCanonicalLocales(configuration.locale)[0],
  timezone: configuration.timezoneId,
  timezoneCanonical: new Intl.DateTimeFormat('en-US', { timeZone: configuration.timezoneId }).resolvedOptions().timeZone,
  configuredTimezoneCanonical: new Intl.DateTimeFormat('en-US', { timeZone: configuration.timezoneId }).resolvedOptions().timeZone,
  colorScheme: configuration.colorScheme,
  deviceScaleFactor: configuration.deviceScaleFactor,
  viewport: configuration.viewport,
};

describe('effective identity evidence', () => {
  it('reports effective values only when the owner readback matches configuration and engine version', async () => {
    const owners = fakeOwners(matchingObservation);
    const evidence = await observeEffectiveIdentity(owners.page, owners.browser, configuration);

    expect(owners.page.evaluate).toHaveBeenCalledWith(expect.any(Function), {
      locale: configuration.locale,
      timezoneId: configuration.timezoneId,
    });
    expect(evidence.engine).toBe('chromium');
    expect(evidence.engine_version).toBe('140.0.7339.80');
    expect(evidence.user_agent).toEqual(expect.objectContaining({
      configured: configuration.userAgent,
      observed: configuration.userAgent,
      effective: configuration.userAgent,
      status: 'effective',
    }));
    expect(evidence.locale.effective).toBe('en-GB');
    expect(evidence.timezone.effective).toBe('Europe/London');
    expect(evidence.color_scheme.effective).toBe('dark');
    expect(evidence.device_scale_factor.effective).toBe(1.25);
    expect(evidence.viewport.effective).toEqual({ width: 1440, height: 900 });
  });

  it('keeps mismatched settings observed but withholds their effective value', async () => {
    const owners = fakeOwners({
      ...matchingObservation,
      locale: 'en-US',
      localeCanonical: 'en-US',
    });
    const evidence = await observeEffectiveIdentity(owners.page, owners.browser, configuration);

    expect(evidence.locale.status).toBe('mismatch');
    expect(evidence.locale.observed).toBe('en-US');
    expect(evidence.locale.effective).toBeUndefined();
  });

  it('accepts locale casing and timezone aliases with equal engine canonical forms', async () => {
    const aliasConfiguration = {
      ...configuration,
      locale: 'EN-us',
      timezoneId: 'US/Eastern',
    };
    const owners = fakeOwners({
      ...matchingObservation,
      locale: 'en-US',
      localeCanonical: 'en-US',
      configuredLocaleCanonical: 'en-US',
      timezone: 'America/New_York',
      timezoneCanonical: 'America/New_York',
      configuredTimezoneCanonical: 'America/New_York',
    });
    const evidence = await observeEffectiveIdentity(owners.page, owners.browser, aliasConfiguration);

    expect(evidence.locale.configured).toBe('EN-us');
    expect(evidence.locale.observed).toBe('en-US');
    expect(evidence.locale.effective).toBe('en-US');
    expect(evidence.timezone.configured).toBe('US/Eastern');
    expect(evidence.timezone.observed).toBe('America/New_York');
    expect(evidence.timezone.effective).toBe('America/New_York');
  });

  it('accepts UTC aliases when the engine canonicalizes both to the same zone', async () => {
    const owners = fakeOwners({
      ...matchingObservation,
      timezone: 'UTC',
      timezoneCanonical: 'UTC',
      configuredTimezoneCanonical: 'UTC',
    });
    const evidence = await observeEffectiveIdentity(owners.page, owners.browser, {
      ...configuration,
      timezoneId: 'Etc/UTC',
    });

    expect(evidence.timezone.configured).toBe('Etc/UTC');
    expect(evidence.timezone.observed).toBe('UTC');
    expect(evidence.timezone.effective).toBe('UTC');
  });

  it('marks canonicalization failures unverified and retains original values', async () => {
    const owners = fakeOwners({
      ...matchingObservation,
      timezone: 'UTC',
      timezoneCanonical: 'UTC',
      configuredTimezoneCanonical: undefined,
    });
    const invalidConfiguration = { ...configuration, timezoneId: 'invalid-zone' };
    const evidence = await observeEffectiveIdentity(owners.page, owners.browser, invalidConfiguration);

    expect(evidence.timezone.configured).toBe('invalid-zone');
    expect(evidence.timezone.observed).toBe('UTC');
    expect(evidence.timezone.status).toBe('unverified');
    expect(evidence.timezone.effective).toBeUndefined();
  });

  it('marks a timezone with different canonical semantics as a mismatch', async () => {
    const owners = fakeOwners({
      ...matchingObservation,
      timezone: 'America/New_York',
      timezoneCanonical: 'America/New_York',
    });
    const evidence = await observeEffectiveIdentity(owners.page, owners.browser, configuration);

    expect(evidence.timezone.status).toBe('mismatch');
    expect(evidence.timezone.observed).toBe('America/New_York');
    expect(evidence.timezone.effective).toBeUndefined();
  });

  it('reports explicit light and dark only when the page media result matches', async () => {
    const lightOwners = fakeOwners({ ...matchingObservation, colorScheme: 'light' });
    const lightEvidence = await observeEffectiveIdentity(lightOwners.page, lightOwners.browser, {
      ...configuration,
      colorScheme: 'light',
    });
    expect(lightEvidence.color_scheme).toEqual(expect.objectContaining({
      configured: 'light', observed: 'light', effective: 'light', status: 'effective',
    }));

    const darkOwners = fakeOwners({ ...matchingObservation, colorScheme: 'dark' });
    const darkEvidence = await observeEffectiveIdentity(darkOwners.page, darkOwners.browser, configuration);
    expect(darkEvidence.color_scheme).toEqual(expect.objectContaining({
      configured: 'dark', observed: 'dark', effective: 'dark', status: 'effective',
    }));
  });

  it('keeps no-preference configured with a light observation unverified', async () => {
    const owners = fakeOwners({ ...matchingObservation, colorScheme: 'light' });
    const evidence = await observeEffectiveIdentity(owners.page, owners.browser, {
      ...configuration,
      colorScheme: 'no-preference',
    });

    expect(evidence.color_scheme.configured).toBe('no-preference');
    expect(evidence.color_scheme.observed).toBe('light');
    expect(evidence.color_scheme.status).toBe('unverified');
    expect(evidence.color_scheme.effective).toBeUndefined();
  });

  it('marks no-preference with a dark observation as a mismatch', async () => {
    const owners = fakeOwners({ ...matchingObservation, colorScheme: 'dark' });
    const evidence = await observeEffectiveIdentity(owners.page, owners.browser, {
      ...configuration,
      colorScheme: 'no-preference',
    });

    expect(evidence.color_scheme.status).toBe('mismatch');
    expect(evidence.color_scheme.observed).toBe('dark');
    expect(evidence.color_scheme.effective).toBeUndefined();
  });

  it('marks a missing color media observation unavailable', async () => {
    const owners = fakeOwners({ ...matchingObservation, colorScheme: undefined });
    const evidence = await observeEffectiveIdentity(owners.page, owners.browser, configuration);

    expect(evidence.color_scheme.status).toBe('unavailable');
    expect(evidence.color_scheme.effective).toBeUndefined();
  });

  it('withholds an apparent UA match when its major disagrees with the serving engine version', async () => {
    const staleUA = 'Mozilla/5.0 Chrome/120.0.0.0 Safari/537.36';
    const owners = fakeOwners({ ...matchingObservation, userAgent: staleUA });
    const evidence = await observeEffectiveIdentity(owners.page, owners.browser, {
      ...configuration,
      userAgent: staleUA,
    });

    expect(evidence.user_agent.status).toBe('mismatch');
    expect(evidence.user_agent.observed).toBe(staleUA);
    expect(evidence.user_agent.effective).toBeUndefined();
    expect(evidence.user_agent.reason).toContain('does not match');
  });

  it('marks unobservable properties unavailable without fabricating effective values', async () => {
    const { browser } = fakeOwners({});
    const page = { evaluate: jest.fn().mockRejectedValue(new Error('page unavailable')) } as unknown as Page;
    const evidence = await observeEffectiveIdentity(page, browser, configuration);

    for (const dimension of [evidence.user_agent, evidence.locale, evidence.timezone, evidence.color_scheme, evidence.device_scale_factor, evidence.viewport]) {
      expect(dimension.status).toBe('unavailable');
      expect(dimension.effective).toBeUndefined();
    }
  });
});
