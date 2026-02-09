const ABSOLUTE_URL_PATTERN = /^[a-z][a-z0-9+.-]*:/i;
const DOMAIN_LIKE_PATTERN = /^[^\s/]+\.[^\s/]+(?:[/?#].*)?$/i;
const LOCALHOST_LIKE_PATTERN = /^(localhost|127(?:\.\d{1,3}){3}|0\.0\.0\.0)(:\d+)?(?:[/?#].*)?$/i;
const HOST_PORT_PATTERN = /^[a-z0-9.-]+:\d+(?:[/?#].*)?$/i;
const SCENARIO_PROXY_PATH_PATTERN = /^\/apps\/([^/]+)\/proxy\/?$/i;
const APPS_SCENARIO_PATH_PATTERN = /^\/?apps\/([^/?#]+)(?:\/(preview))?\/?$/i;
const SCENARIO_IDENTIFIER_PATTERN = /^[a-z0-9][a-z0-9._-]*$/i;
const RESERVED_SHELL_ROUTE_IDS = new Set(['workspace', 'tabs', 'resources', 'logs']);

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

    const base = resolveReferenceBase(reference);
    if (trimmed.startsWith('/')) {
      if (base) {
        return new URL(trimmed, base).href;
      }
      const browserOrigin = getBrowserOrigin();
      if (browserOrigin) {
        return new URL(trimmed, browserOrigin).href;
      }
    }

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

export const isAppMonitorProxyPreviewTarget = (value: string): boolean => {
  const scenario = parseScenarioProxyPreviewTarget(value);
  return scenario?.scenarioIdentifier.trim().toLowerCase() === 'app-monitor';
};

export const formatPreviewUrlForDisplay = (value: string): string => {
  const scenario = parseScenarioProxyPreviewTarget(value);
  if (!scenario) {
    return value;
  }
  return scenario.displayLabel;
};

const buildScenarioProxyPath = (scenarioIdentifier: string): string => (
  `/apps/${encodeURIComponent(scenarioIdentifier)}/proxy/`
);

const isLikelyScenarioIdentifier = (value: string): boolean => (
  value.includes('-') || value.includes('_')
);

export const normalizeScenarioNavigationInput = (value: string): string | null => {
  const trimmed = value.trim();
  if (!trimmed) {
    return null;
  }

  if (SCENARIO_IDENTIFIER_PATTERN.test(trimmed) && isLikelyScenarioIdentifier(trimmed)) {
    return buildScenarioProxyPath(trimmed);
  }

  try {
    const parsed = new URL(trimmed, 'http://preview.local');
    const pathMatch = parsed.pathname.match(APPS_SCENARIO_PATH_PATTERN);
    if (!pathMatch) {
      return null;
    }

    const scenarioIdentifier = decodeURIComponent(pathMatch[1] ?? '').trim();
    if (!scenarioIdentifier) {
      return null;
    }
    if (RESERVED_SHELL_ROUTE_IDS.has(scenarioIdentifier.toLowerCase())) {
      return null;
    }

    const proxyPath = `${buildScenarioProxyPath(scenarioIdentifier)}${parsed.search}${parsed.hash}`;
    if (ABSOLUTE_URL_PATTERN.test(trimmed)) {
      return `${parsed.origin}${proxyPath}`;
    }
    return proxyPath;
  } catch {
    return null;
  }
};

export const isBlockedHostEmbedPreviewTarget = (
  targetUrl: string,
  hostOrigin: string | null,
): boolean => {
  if (!hostOrigin) {
    return false;
  }

  try {
    const parsed = new URL(targetUrl, 'http://preview.local');
    if (parsed.origin !== hostOrigin) {
      return false;
    }

    const path = parsed.pathname.endsWith('/') && parsed.pathname !== '/'
      ? parsed.pathname.slice(0, -1)
      : parsed.pathname;
    if (path.match(SCENARIO_PROXY_PATH_PATTERN)) {
      return false;
    }

    return (
      path === ''
      || path === '/'
      || path === '/apps'
      || path === '/apps/workspace'
      || path === '/tabs'
      || path === '/resources'
      || path === '/logs'
      || /^\/apps\/[^/]+(?:\/preview)?$/i.test(path)
    );
  } catch {
    return false;
  }
};
