// Simplified configuration for browser-automation-studio
import { resolveProxyPathForPort } from './utils/proxyResolver';
import type { Config, ConfigResponse } from './types/config';

const LOOPBACK_HOSTS = new Set(['localhost', '127.0.0.1', '0.0.0.0', '::1', '[::1]']);
const API_SUFFIX = '/api/v1';

let cachedConfig: Config | null = null;

function isLoopbackHost(value?: string | null): boolean {
  return value ? LOOPBACK_HOSTS.has(value.toLowerCase()) : false;
}

function coerceString(value: unknown, fallback: string): string {
  if (typeof value !== 'string') {
    return fallback;
  }
  const trimmed = value.trim();
  if (!trimmed || trimmed.toLowerCase() === 'undefined' || trimmed.toLowerCase() === 'null') {
    return fallback;
  }
  return trimmed;
}

function extractPort(urlString: string, fallback: string): string {
  if (!urlString) {
    return fallback;
  }
  try {
    const parsed = new URL(urlString);
    return parsed.port || fallback;
  } catch {
    return fallback;
  }
}

function createUrlFromString(raw?: string): URL | null {
  if (!raw) {
    return null;
  }

  const trimmed = raw.trim();
  if (!trimmed) {
    return null;
  }

  // Try with protocol
  const attempts = [trimmed];
  if (!/^[a-zA-Z][a-zA-Z0-9+-.]*:/u.test(trimmed)) {
    attempts.push(`http://${trimmed}`);
  }

  for (const attempt of attempts) {
    try {
      return new URL(attempt);
    } catch {
      if (typeof window !== 'undefined' && window.location) {
        try {
          return new URL(attempt, window.location.origin);
        } catch {
          // Try next
        }
      }
    }
  }

  return null;
}

function stripApiSuffix(url: URL): void {
  const trimmed = (url.pathname || '').replace(/\/+$/u, '');
  if (trimmed.toLowerCase().endsWith(API_SUFFIX)) {
    url.pathname = trimmed.slice(0, -API_SUFFIX.length) || '/';
  } else {
    url.pathname = trimmed || '/';
  }
}

function ensureTrailingSlash(url: URL): void {
  if (!url.pathname || !url.pathname.endsWith('/')) {
    url.pathname = (url.pathname || '') + '/';
  }
}

function resolveWebSocketUrl(explicitWs?: string, apiUrl?: string): string {
  const candidates = [explicitWs, apiUrl].filter((v): v is string => typeof v === 'string' && v.trim().length > 0);

  for (const candidate of candidates) {
    const url = createUrlFromString(candidate);
    if (!url) {
      continue;
    }

    stripApiSuffix(url);

    // Try proxy path resolution
    let proxyPath: string | null = null;
    if (typeof window !== 'undefined' && window.location) {
      proxyPath = resolveProxyPathForPort(url.port || undefined);
      if (proxyPath) {
        url.hostname = window.location.hostname || url.hostname;
        url.port = window.location.port ?? '';
        url.pathname = proxyPath;
        url.search = '';
      }
    }

    ensureTrailingSlash(url);

    // Set WebSocket protocol
    if (typeof window !== 'undefined' && window.location) {
      url.protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    } else {
      url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
    }

    // Skip if remote host but url is loopback and no proxy
    if (typeof window !== 'undefined' && window.location) {
      const remoteHost = !isLoopbackHost(window.location.hostname);
      if (remoteHost && isLoopbackHost(url.hostname) && !proxyPath) {
        continue;
      }
    }

    return url.toString();
  }

  // Fallback to current location
  if (typeof window !== 'undefined' && window.location) {
    const url = new URL(window.location.origin);
    url.protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    ensureTrailingSlash(url);
    return url.toString();
  }

  return '';
}

/**
 * Builds config object from environment variables (shared between sync and async)
 */
function buildConfigFromEnv(): Config | null {
  if (!import.meta.env.VITE_API_URL) {
    return null;
  }

  const apiUrl = import.meta.env.VITE_API_URL;
  const wsBase = resolveWebSocketUrl(import.meta.env.VITE_WS_URL, apiUrl);
  const apiPortFallback = coerceString(import.meta.env.VITE_API_PORT, '24785');
  const uiPortFallback = coerceString(import.meta.env.VITE_UI_PORT, '42969');
  const wsPortFallback = coerceString(import.meta.env.VITE_WS_PORT, extractPort(wsBase, '29031'));

  return {
    API_URL: apiUrl,
    WS_URL: wsBase,
    API_PORT: extractPort(apiUrl, apiPortFallback),
    UI_PORT: uiPortFallback,
    WS_PORT: extractPort(wsBase, wsPortFallback),
  };
}

async function getConfig(): Promise<Config> {
  if (cachedConfig) {
    return cachedConfig;
  }

  // Dev mode - use build-time env variables
  const envConfig = buildConfigFromEnv();
  if (envConfig) {
    cachedConfig = envConfig;
    return cachedConfig;
  }

  // Production mode - fetch runtime config
  const basePath = import.meta.env.BASE_URL || '/';
  const normalizedBase = basePath.endsWith('/') ? basePath : `${basePath}/`;
  const configPath = `${normalizedBase}config`;

  try {
    const response = await fetch(configPath, {
      headers: { accept: 'application/json' },
    });
    if (!response.ok) {
      throw new Error(`Config fetch failed: ${response.status}`);
    }

    const raw = await response.text();
    let data: ConfigResponse;

    try {
      data = JSON.parse(raw) as ConfigResponse;
    } catch (parseError) {
      const isHtml = raw.trim().startsWith('<');
      const hint = isHtml
        ? 'Received HTML instead of JSON. Run the scenario via "make start" or ensure the production server (node server.js) is serving /config.'
        : `Response body: ${raw.slice(0, 120)}`;
      throw new Error(`Config endpoint did not return JSON. ${hint}`);
    }

    if (!data.apiUrl) {
      throw new Error('API URL not configured');
    }

    const apiUrl = String(data.apiUrl);
    const wsBase = resolveWebSocketUrl(data.wsUrl, apiUrl);
    const apiPort = extractPort(apiUrl, coerceString(data.apiPort, '24785'));
    const wsPort = extractPort(wsBase, coerceString(data.wsPort, '29031'));
    const uiPort = typeof window !== 'undefined' && window.location && window.location.port ? window.location.port : '42969';

    cachedConfig = {
      API_URL: apiUrl,
      WS_URL: wsBase,
      API_PORT: apiPort,
      UI_PORT: uiPort,
      WS_PORT: wsPort,
    };
    return cachedConfig;
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Unknown error';
    throw new Error(`Failed to load configuration: ${message}`);
  }
}

function getConfigSync(): Config {
  if (cachedConfig) {
    return cachedConfig;
  }

  const envConfig = buildConfigFromEnv();
  if (envConfig) {
    return envConfig;
  }

  throw new Error('Configuration not loaded. Call getConfig() first.');
}

export { getConfig };
export const getCachedConfig = () => cachedConfig;
export const config = getConfigSync;
export const getApiBase = () => config().API_URL;
export const getWsBase = () => config().WS_URL;

export const API_BASE = (() => {
  try {
    return getApiBase();
  } catch {
    return '';
  }
})();

export const WS_BASE = (() => {
  try {
    return getWsBase();
  } catch {
    return '';
  }
})();
