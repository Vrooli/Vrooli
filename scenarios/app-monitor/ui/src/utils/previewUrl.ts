const ABSOLUTE_URL_PATTERN = /^[a-z][a-z0-9+.-]*:/i;
const DOMAIN_LIKE_PATTERN = /^[^\s/]+\.[^\s/]+(?:[/?#].*)?$/i;
const LOCALHOST_LIKE_PATTERN = /^(localhost|127(?:\.\d{1,3}){3}|0\.0\.0\.0)(:\d+)?(?:[/?#].*)?$/i;
const HOST_PORT_PATTERN = /^[a-z0-9.-]+:\d+(?:[/?#].*)?$/i;
const SCENARIO_PROXY_PATH_PATTERN = /^\/apps\/([^/]+)\/proxy\/?$/i;

const getBrowserLocationHref = (): string | null => {
  if (typeof window === 'undefined') {
    return null;
  }
  return window.location.href;
};

const getBrowserOrigin = (): string | null => {
  if (typeof window === 'undefined') {
    return null;
  }
  return window.location.origin;
};

const resolveReferenceBase = (reference: string | null | undefined): string | null => {
  const trimmed = typeof reference === 'string' ? reference.trim() : '';
  if (trimmed.length > 0) {
    try {
      return new URL(trimmed).href;
    } catch {
      const browserHref = getBrowserLocationHref();
      if (browserHref) {
        try {
          return new URL(trimmed, browserHref).href;
        } catch {
          return null;
        }
      }
      return null;
    }
  }

  return getBrowserLocationHref();
};

export const resolvePreviewUrlCandidate = (
  value: string,
  reference: string | null = null,
): string | null => {
  const trimmed = value.trim();
  if (!trimmed) {
    return null;
  }

  try {
    if (LOCALHOST_LIKE_PATTERN.test(trimmed) || HOST_PORT_PATTERN.test(trimmed)) {
      return new URL(`http://${trimmed}`).href;
    }

    if (ABSOLUTE_URL_PATTERN.test(trimmed)) {
      return new URL(trimmed).href;
    }

    if (trimmed.startsWith('//')) {
      return new URL(`https:${trimmed}`).href;
    }

    if (DOMAIN_LIKE_PATTERN.test(trimmed)) {
      return new URL(`https://${trimmed}`).href;
    }

    if (trimmed.startsWith('/')) {
      const browserOrigin = getBrowserOrigin();
      if (browserOrigin) {
        return new URL(trimmed, browserOrigin).href;
      }
    }

    const base = resolveReferenceBase(reference);
    if (!base) {
      return null;
    }

    return new URL(trimmed, base).href;
  } catch {
    return null;
  }
};

export interface ScenarioProxyPreviewTarget {
  scenarioIdentifier: string;
  query: string;
  displayLabel: string;
}

export const parseScenarioProxyPreviewTarget = (
  value: string,
): ScenarioProxyPreviewTarget | null => {
  const resolved = resolvePreviewUrlCandidate(value, null);
  const fallback = value.trim();
  const candidate = resolved ?? fallback;

  if (!candidate) {
    return null;
  }

  try {
    const parsed = new URL(candidate, 'http://preview.local');
    const match = parsed.pathname.match(SCENARIO_PROXY_PATH_PATTERN);
    if (!match) {
      return null;
    }

    const scenarioIdentifier = decodeURIComponent(match[1] ?? '').trim();
    if (!scenarioIdentifier) {
      return null;
    }

    const query = parsed.search;
    return {
      scenarioIdentifier,
      query,
      displayLabel: `${scenarioIdentifier}${query ? `:${query}` : ''}`,
    };
  } catch {
    return null;
  }
};

export const formatPreviewUrlForDisplay = (value: string): string => {
  const scenario = parseScenarioProxyPreviewTarget(value);
  if (!scenario) {
    return value;
  }
  return scenario.displayLabel;
};
