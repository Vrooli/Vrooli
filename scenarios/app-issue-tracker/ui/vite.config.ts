import { defineConfig } from 'vite';
import type { ServerOptions } from 'vite';
import react from '@vitejs/plugin-react';

const PROXY_TARGET_ENV_KEYS = [
  'APP_MONITOR_PROXY_API_TARGET',
  'APP_ISSUE_TRACKER_PROXY_API_TARGET',
  'VITE_PROXY_API_TARGET',
  'APP_MONITOR_PROXY_API_ORIGIN',
  'APP_ISSUE_TRACKER_PROXY_API_ORIGIN',
  'VITE_PROXY_API_ORIGIN',
  'APP_MONITOR_PROXY_API_URL',
  'APP_ISSUE_TRACKER_PROXY_API_URL',
  'VITE_PROXY_API_URL',
];

const PROXY_PORT_ENV_KEYS = [
  'APP_MONITOR_PROXY_API_PORT',
  'APP_ISSUE_TRACKER_PROXY_API_PORT',
  'VITE_PROXY_API_PORT',
];

const PROXY_WS_TARGET_ENV_KEYS = [
  'APP_MONITOR_PROXY_WS_TARGET',
  'APP_ISSUE_TRACKER_PROXY_WS_TARGET',
  'VITE_PROXY_WS_TARGET',
];

const FALLBACK_API_HOST_ENV_KEYS = [
  'VITE_API_HOST',
  'APP_ISSUE_TRACKER_API_HOST',
  'API_HOST',
];

const FALLBACK_API_PROTOCOL_ENV_KEYS = [
  'VITE_API_PROTOCOL',
  'APP_ISSUE_TRACKER_API_PROTOCOL',
  'API_PROTOCOL',
];

function readFirstEnv(keys: string[]): string | undefined {
  for (const key of keys) {
    const value = process.env[key];
    if (typeof value === 'string') {
      const trimmed = value.trim();
      if (trimmed.length > 0) {
        return trimmed;
      }
    }
  }
  return undefined;
}

function stripTrailingSlash(value?: string | null): string | undefined {
  if (!value) {
    return undefined;
  }

  const trimmed = value.trim();
  if (trimmed.length === 0) {
    return undefined;
  }

  return trimmed.replace(/\/+$/u, '');
}

function resolveApiProtocol(): 'http' | 'https' {
  const candidate = readFirstEnv(FALLBACK_API_PROTOCOL_ENV_KEYS);
  if (candidate && candidate.toLowerCase() === 'https') {
    return 'https';
  }
  return 'http';
}

function resolveApiHost(): string {
  return readFirstEnv(FALLBACK_API_HOST_ENV_KEYS) ?? '127.0.0.1';
}

function extractPort(candidate: string | undefined): string | undefined {
  if (!candidate) {
    return undefined;
  }

  try {
    const location = new URL(candidate);
    if (location.port) {
      return location.port;
    }

    if (location.protocol === 'http:' || location.protocol === 'ws:') {
      return '80';
    }

    if (location.protocol === 'https:' || location.protocol === 'wss:') {
      return '443';
    }
  } catch {
    const match = candidate.match(/:(\d+)(?:[^0-9]|$)/u);
    if (match) {
      return match[1];
    }
  }

  return undefined;
}

function ensureAbsolute(target: string, protocol: 'http' | 'https' | 'ws' | 'wss'): string {
  if (/^[a-zA-Z][a-zA-Z0-9+-.]*:\/\//u.test(target)) {
    return stripTrailingSlash(target) ?? target;
  }

  if (target.startsWith('//')) {
    return stripTrailingSlash(`${protocol}:${target}`) ?? `${protocol}:${target}`;
  }

  if (target.startsWith('/')) {
    return stripTrailingSlash(`${protocol}://${resolveApiHost()}${target}`) ?? `${protocol}://${resolveApiHost()}${target}`;
  }

  return stripTrailingSlash(`${protocol}://${target}`) ?? `${protocol}://${target}`;
}

function resolveProxyTarget(defaultOrigin: string): string {
  const candidate = stripTrailingSlash(readFirstEnv(PROXY_TARGET_ENV_KEYS));
  if (candidate) {
    if (/^[a-zA-Z][a-zA-Z0-9+-.]*:\/\//u.test(candidate)) {
      return candidate;
    }
    return ensureAbsolute(candidate, resolveApiProtocol());
  }
  return defaultOrigin;
}

function resolveWebSocketProxyTarget(httpTarget: string): string {
  const explicit = stripTrailingSlash(readFirstEnv(PROXY_WS_TARGET_ENV_KEYS));
  if (explicit) {
    if (/^wss?:\/\//iu.test(explicit) || explicit.startsWith('//')) {
      return ensureAbsolute(explicit, explicit.startsWith('wss') ? 'wss' : 'ws');
    }
    return ensureAbsolute(explicit, 'ws');
  }

  if (/^wss?:\/\//iu.test(httpTarget)) {
    return httpTarget;
  }

  if (/^https?:\/\//iu.test(httpTarget)) {
    try {
      const url = new URL(httpTarget);
      url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
      return stripTrailingSlash(url.toString()) ?? url.toString();
    } catch {
      return httpTarget;
    }
  }

  return ensureAbsolute(httpTarget, 'ws');
}

function optionalNumberEnv(name: string): number | undefined {
  const raw = process.env[name];
  if (!raw) {
    return undefined;
  }

  const parsed = Number.parseInt(raw, 10);
  if (Number.isNaN(parsed)) {
    throw new Error(`${name} must be a valid integer (received: ${raw})`);
  }
  return parsed;
}

// API_PORT is optional for build - runtime resolution handles this
const apiPort = process.env.API_PORT || '15000';

const hmrHost = process.env.VITE_HMR_HOST;
const hmrPort = optionalNumberEnv('VITE_HMR_PORT');
const hmrProtocol = process.env.VITE_HMR_PROTOCOL as 'ws' | 'wss' | undefined;

const hmrConfig = hmrHost || hmrPort || hmrProtocol
  ? {
      ...(hmrHost ? { host: hmrHost } : {}),
      ...(hmrPort !== undefined ? { clientPort: hmrPort } : {}),
      ...(hmrProtocol ? { protocol: hmrProtocol } : {}),
    }
  : undefined;

const apiProtocol = resolveApiProtocol();
const apiHost = resolveApiHost();
const defaultApiOrigin = stripTrailingSlash(`${apiProtocol}://${apiHost}:${apiPort}`) ?? `${apiProtocol}://${apiHost}:${apiPort}`;
const proxyTarget = resolveProxyTarget(defaultApiOrigin);
const wsProxyTarget = resolveWebSocketProxyTarget(proxyTarget);
const resolvedApiPort =
  readFirstEnv(PROXY_PORT_ENV_KEYS) ??
  extractPort(proxyTarget) ??
  extractPort(defaultApiOrigin) ??
  apiPort;

function resolveJsdomUrl(): string {
  const candidates = [stripTrailingSlash(proxyTarget), stripTrailingSlash(defaultApiOrigin)].filter(
    (value): value is string => Boolean(value),
  );

  for (const candidate of candidates) {
    try {
      const parsed = new URL(candidate);
      return `${parsed.protocol}//${parsed.host}/`;
    } catch {
      if (/^https?:/iu.test(candidate)) {
        return candidate.endsWith('/') ? candidate : `${candidate}/`;
      }
    }
  }

  return 'https://app-issue-tracker.test/';
}

const jsdomUrl = resolveJsdomUrl();

const server: ServerOptions = {
  hmr: hmrConfig,
  proxy: {
    '/api': {
      target: proxyTarget,
      changeOrigin: true,
      ws: true, // Enable WebSocket proxying
    },
    '/health': {
      target: proxyTarget,
      changeOrigin: true,
    },
    '/ws': {
      target: wsProxyTarget,
      changeOrigin: true,
      ws: true,
    },
  },
};

const uiPort = optionalNumberEnv('UI_PORT');
if (uiPort !== undefined) {
  server.port = uiPort;
}

const serverHost = process.env.VITE_SERVER_HOST;
server.host = serverHost ?? true;

const allowedHosts = new Set<string>();

const allowedHostsRaw = process.env.VITE_ALLOWED_HOSTS;
if (allowedHostsRaw) {
  allowedHostsRaw
    .split(',')
    .map((host) => host.trim())
    .filter(Boolean)
    .forEach((host) => allowedHosts.add(host));
}

const allowedOriginsRaw = process.env.ALLOWED_ORIGINS;
if (allowedOriginsRaw) {
  allowedOriginsRaw
    .split(',')
    .map((origin) => origin.trim())
    .filter(Boolean)
    .forEach((origin) => {
      try {
        const parsed = new URL(origin);
        if (parsed.hostname) {
          allowedHosts.add(parsed.hostname);
        }
      } catch {
        // ignore malformed origins
      }
    });
}

if (allowedHosts.size > 0) {
  server.allowedHosts = Array.from(allowedHosts);
} else {
  // Allow lifecycle-managed domains (Vite will validate origin headers separately)
  server.allowedHosts = true;
}

export default defineConfig({
  plugins: [react()],
  define: {
    // Expose API port to the frontend for WebSocket connections
    __API_PORT__: JSON.stringify(resolvedApiPort),
  },
  server,
  test: {
    environment: 'jsdom',
    setupFiles: './src/test-utils/setupTests.ts',
    coverage: {
      reporter: ['text', 'html'],
    },
    environmentOptions: {
      jsdom: {
        url: jsdomUrl,
      },
    },
  },
});
