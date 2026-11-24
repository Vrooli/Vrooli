import type { Browser, Page } from 'rebrowser-playwright';

export type IdentityEvidenceStatus = 'effective' | 'mismatch' | 'unverified' | 'unavailable';

export interface IdentityDimensionEvidence<T = string | number> {
  configured?: T;
  observed?: T;
  /** Populated only when the configured/default value agrees with observation. */
  effective?: T;
  status: IdentityEvidenceStatus;
  source: string;
  reason?: string;
}

export interface EffectiveIdentityEvidence {
  engine: string;
  engine_version: string;
  engine_source: 'playwright.browser';
  user_agent: IdentityDimensionEvidence<string>;
  locale: IdentityDimensionEvidence<string>;
  timezone: IdentityDimensionEvidence<string>;
  color_scheme: IdentityDimensionEvidence<string>;
  device_scale_factor: IdentityDimensionEvidence<number>;
  viewport: IdentityDimensionEvidence<{ width: number; height: number }>;
}

export interface IdentityContextConfiguration {
  userAgent?: string;
  locale?: string;
  timezoneId?: string;
  colorScheme?: string;
  deviceScaleFactor: number;
  viewport: { width: number; height: number };
}

interface PageIdentityObservation {
  userAgent?: string;
  locale?: string;
  localeCanonical?: string;
  configuredLocaleCanonical?: string;
  timezone?: string;
  timezoneCanonical?: string;
  configuredTimezoneCanonical?: string;
  colorScheme?: string;
  deviceScaleFactor?: number;
  viewport?: { width: number; height: number };
}

const identityScript = (configured: Pick<IdentityContextConfiguration, 'locale' | 'timezoneId'>): PageIdentityObservation => {
  const canonicalLocale = (value?: string): string | undefined => {
    if (!value) return undefined;
    try {
      return Intl.getCanonicalLocales(value)[0];
    } catch {
      return undefined;
    }
  };
  const canonicalTimezone = (value?: string): string | undefined => {
    if (!value) return undefined;
    try {
      return new Intl.DateTimeFormat('en-US', { timeZone: value }).resolvedOptions().timeZone;
    } catch {
      return undefined;
    }
  };
  return {
    userAgent: navigator.userAgent,
    locale: navigator.language,
    localeCanonical: canonicalLocale(navigator.language),
    configuredLocaleCanonical: canonicalLocale(configured.locale),
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    timezoneCanonical: canonicalTimezone(Intl.DateTimeFormat().resolvedOptions().timeZone),
    configuredTimezoneCanonical: canonicalTimezone(configured.timezoneId),
    // CSS Media Queries exposes only light/dark. A light match can also mean
    // no active preference, so page-visible evidence cannot verify that
    // distinction on its own.
    colorScheme: matchMedia('(prefers-color-scheme: dark)').matches
      ? 'dark'
      : matchMedia('(prefers-color-scheme: light)').matches
        ? 'light'
        : undefined,
    deviceScaleFactor: window.devicePixelRatio,
    viewport: { width: window.innerWidth, height: window.innerHeight },
  };
};

const equals = (configured: unknown, observed: unknown): boolean => {
  if (typeof configured === 'number' && typeof observed === 'number') {
    return Math.abs(configured - observed) < 0.001;
  }
  if (
    typeof configured === 'object' && configured !== null &&
    typeof observed === 'object' && observed !== null &&
    'width' in configured && 'height' in configured &&
    'width' in observed && 'height' in observed
  ) {
    return configured.width === observed.width && configured.height === observed.height;
  }
  return configured === observed;
};

const uaMatchesEngineVersion = (engine: string, version: string, userAgent: string): boolean | undefined => {
  const major = version.match(/^\s*(\d+)/)?.[1];
  if (!major) return undefined;
  const pattern = engine === 'chromium'
    ? /(?:Chrome|Chromium)\/(\d+)/
    : engine === 'firefox'
      ? /Firefox\/(\d+)/
      : undefined;
  if (!pattern) return undefined;
  return pattern.exec(userAgent)?.[1] === major;
};

function dimension<T extends string | number | { width: number; height: number }>(
  configured: T | undefined,
  observed: T | undefined,
  source: string,
  reason?: string,
  equivalent?: () => boolean | undefined
): IdentityDimensionEvidence<T> {
  if (observed === undefined || observed === null) {
    return { ...(configured === undefined ? {} : { configured }), status: 'unavailable', source, reason: reason ?? 'owner could not read this value' };
  }
  if (configured !== undefined) {
    const matches = equivalent ? equivalent() : equals(configured, observed);
    if (matches === undefined) {
      return { configured, observed, status: 'unverified', source, reason: reason ?? 'engine could not establish semantic equivalence' };
    }
    if (!matches) {
      return { configured, observed, status: 'mismatch', source, reason: reason ?? 'owner readback differs from configured value' };
    }
  }
  return { ...(configured === undefined ? {} : { configured }), observed, effective: observed, status: 'effective', source };
}

/**
 * Reads page-visible identity in the fresh, pre-navigation context and keeps
 * configured, observed, and effective values separate. Engine identity and
 * version come from Playwright's browser owner, never page JavaScript.
 */
export async function observeEffectiveIdentity(
  page: Page,
  browser: Browser,
  configured: IdentityContextConfiguration
): Promise<EffectiveIdentityEvidence> {
  const source = 'playwright.page.about_blank';
  let engine = 'unavailable';
  let engineVersion = '';
  try {
    engine = browser.browserType().name();
    engineVersion = browser.version();
  } catch {
    // A missing owner API is explicit in the returned facts and cannot create
    // an effective UA claim.
  }
  let observed: PageIdentityObservation = {};
  try {
    observed = await page.evaluate(identityScript, {
      locale: configured.locale,
      timezoneId: configured.timezoneId,
    });
  } catch {
    // Session admission must not fail because the optional evidence read fails.
  }

  const userAgent = dimension(configured.userAgent, observed.userAgent, source);
  if (userAgent.status === 'effective') {
    const matches = uaMatchesEngineVersion(engine, engineVersion, observed.userAgent ?? '');
    if (matches === false) {
      userAgent.status = 'mismatch';
      delete userAgent.effective;
      userAgent.reason = `user-agent major does not match the owner-reported ${engine} engine version ${engineVersion || '(unavailable)'}`;
    } else if (matches === undefined) {
      userAgent.status = 'unverified';
      delete userAgent.effective;
      userAgent.reason = `UA-to-version consistency is not verified for engine ${engine || '(unavailable)'}`;
    }
  }

  return {
    engine,
    engine_version: engineVersion,
    engine_source: 'playwright.browser',
    user_agent: userAgent,
    locale: dimension(configured.locale, observed.locale, source, 'engine could not canonicalize configured and observed locales', () => {
      if (!observed.configuredLocaleCanonical || !observed.localeCanonical) return undefined;
      return observed.configuredLocaleCanonical === observed.localeCanonical;
    }),
    timezone: dimension(configured.timezoneId, observed.timezone, source, 'engine could not canonicalize configured and observed timezones', () => {
      if (!observed.configuredTimezoneCanonical || !observed.timezoneCanonical) return undefined;
      return observed.configuredTimezoneCanonical === observed.timezoneCanonical;
    }),
    color_scheme: dimension(
      configured.colorScheme,
      observed.colorScheme,
      source,
      'page-visible color scheme cannot distinguish no-preference from light',
      () => {
        if (configured.colorScheme === 'no-preference') {
          if (observed.colorScheme === 'dark') return false;
          if (observed.colorScheme === 'light') return undefined;
          return undefined;
        }
        return configured.colorScheme === observed.colorScheme;
      }
    ),
    device_scale_factor: dimension(configured.deviceScaleFactor, observed.deviceScaleFactor, source),
    viewport: dimension(configured.viewport, observed.viewport, source),
  };
}
