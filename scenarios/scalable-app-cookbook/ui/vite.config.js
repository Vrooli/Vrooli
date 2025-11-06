import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

const DEFAULT_UI_PORT = 3301;
const DEFAULT_API_PORT = 17515;
const HEALTH_TIMEOUT_MS = 3000;
const SERVICE_NAME = 'scalable-app-cookbook-ui';

const PROXY_TARGET_ENV_KEYS = [
  'APP_MONITOR_PROXY_API_TARGET',
  'SCALABLE_APP_COOKBOOK_PROXY_API_TARGET',
  'VITE_PROXY_API_TARGET',
  'APP_MONITOR_PROXY_API_ORIGIN',
  'SCALABLE_APP_COOKBOOK_PROXY_API_ORIGIN',
  'VITE_PROXY_API_ORIGIN',
  'APP_MONITOR_PROXY_API_URL',
  'SCALABLE_APP_COOKBOOK_PROXY_API_URL',
  'VITE_PROXY_API_URL',
];

const PROXY_WS_TARGET_ENV_KEYS = [
  'APP_MONITOR_PROXY_WS_TARGET',
  'SCALABLE_APP_COOKBOOK_PROXY_WS_TARGET',
  'VITE_PROXY_WS_TARGET',
];

const FALLBACK_API_HOST_ENV_KEYS = [
  'VITE_API_HOST',
  'SCALABLE_APP_COOKBOOK_API_HOST',
  'API_HOST',
];

const FALLBACK_API_PROTOCOL_ENV_KEYS = [
  'VITE_API_PROTOCOL',
  'SCALABLE_APP_COOKBOOK_API_PROTOCOL',
  'API_PROTOCOL',
];

const readFirstEnv = (keys) => {
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
};

const stripTrailingSlash = (value) => {
  if (!value) {
    return undefined;
  }

  const trimmed = value.trim();
  if (trimmed.length === 0) {
    return undefined;
  }

  return trimmed.replace(/\/+$/u, '');
};

const joinUrl = (base, path) => {
  const normalizedBase = stripTrailingSlash(base) ?? base;
  const normalizedPath = path.startsWith('/') ? path.slice(1) : path;

  if (!normalizedBase) {
    return `/${normalizedPath}`;
  }

  return `${normalizedBase}/${normalizedPath}`;
};

const toPort = (value, fallback) => {
  const parsed = Number.parseInt(value ?? '', 10);
  return Number.isFinite(parsed) ? parsed : fallback;
};

const resolveApiProtocol = () => {
  const candidate = readFirstEnv(FALLBACK_API_PROTOCOL_ENV_KEYS);
  if (candidate && candidate.toLowerCase() === 'https') {
    return 'https';
  }
  return 'http';
};

const resolveApiHost = () => readFirstEnv(FALLBACK_API_HOST_ENV_KEYS) ?? '127.0.0.1';

const ensureAbsolute = (target, protocol, host) => {
  if (!target) {
    return undefined;
  }

  const trimmed = target.trim();
  if (trimmed.length === 0) {
    return undefined;
  }

  if (/^[a-zA-Z][a-zA-Z0-9+-.]*:\/\//u.test(trimmed)) {
    return stripTrailingSlash(trimmed) ?? trimmed;
  }

  if (trimmed.startsWith('//')) {
    return stripTrailingSlash(`${protocol}:${trimmed}`) ?? `${protocol}:${trimmed}`;
  }

  if (trimmed.startsWith('/')) {
    return stripTrailingSlash(`${protocol}://${host}${trimmed}`) ?? `${protocol}://${host}${trimmed}`;
  }

  return stripTrailingSlash(`${protocol}://${trimmed}`) ?? `${protocol}://${trimmed}`;
};

const resolveProxyTarget = (defaultOrigin, protocol, host) => {
  const candidate = readFirstEnv(PROXY_TARGET_ENV_KEYS);
  if (candidate) {
    return ensureAbsolute(candidate, protocol, host) ?? defaultOrigin;
  }
  return defaultOrigin;
};

const resolveWebSocketProxyTarget = (httpTarget, host) => {
  const explicit = readFirstEnv(PROXY_WS_TARGET_ENV_KEYS);
  if (explicit) {
    const normalized = explicit.trim();
    if (/^wss?:\/\//iu.test(normalized) || normalized.startsWith('//')) {
      const wsProtocol = normalized.startsWith('wss') ? 'wss' : 'ws';
      return ensureAbsolute(normalized, wsProtocol, host) ?? ensureAbsolute(httpTarget, 'ws', host) ?? httpTarget;
    }
    return ensureAbsolute(normalized, 'ws', host) ?? ensureAbsolute(httpTarget, 'ws', host) ?? httpTarget;
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

  return ensureAbsolute(httpTarget, 'ws', host) ?? httpTarget;
};

const UI_PORT = toPort(process.env.UI_PORT, DEFAULT_UI_PORT);
const apiPort = toPort(process.env.API_PORT, DEFAULT_API_PORT);
const apiProtocol = resolveApiProtocol();
const apiHost = resolveApiHost();
const defaultApiOrigin = stripTrailingSlash(`${apiProtocol}://${apiHost}:${apiPort}`) ?? `${apiProtocol}://${apiHost}:${apiPort}`;
const proxyTarget = resolveProxyTarget(defaultApiOrigin, apiProtocol, apiHost);
const wsProxyTarget = resolveWebSocketProxyTarget(proxyTarget, apiHost);
const apiHealthEndpoint = joinUrl(proxyTarget, 'health');

const healthPlugin = {
  name: 'scalable-app-cookbook-ui-health-endpoint',
  configureServer(server) {
    server.middlewares.use('/health', (req, res) => {
      const requestStartedAt = Date.now();
      const timestamp = new Date().toISOString();
      const controller = new AbortController();
      const timeout = setTimeout(() => controller.abort(), HEALTH_TIMEOUT_MS);

      const respond = (body) => {
        if (!res.headersSent) {
          res.statusCode = 200;
          res.setHeader('Content-Type', 'application/json');
          res.end(JSON.stringify(body));
        }
      };

      const buildPayload = ({ connected, latencyMs = null, error = null }) => ({
        status: connected ? 'healthy' : 'degraded',
        service: SERVICE_NAME,
        timestamp,
        readiness: connected,
        api_connectivity: {
          connected,
          api_url: apiHealthEndpoint,
          last_check: timestamp,
          latency_ms: connected ? latencyMs : null,
          error,
        },
      });

      fetch(apiHealthEndpoint, { signal: controller.signal })
        .then(async (apiResponse) => {
          const latencyMs = Date.now() - requestStartedAt;

          if (apiResponse.ok) {
            respond(buildPayload({ connected: true, latencyMs }));
            return;
          }

          const errorText = await apiResponse.text().catch(() => '');
          respond(
            buildPayload({
              connected: false,
              error: {
                code: 'API_UNHEALTHY_RESPONSE',
                message: `API health endpoint returned status ${apiResponse.status}`,
                category: 'network',
                retryable: true,
                details: {
                  status: apiResponse.status,
                  body: errorText.slice(0, 1024),
                },
              },
            }),
          );
        })
        .catch((error) => {
          const isAbortError = error?.name === 'AbortError';
          respond(
            buildPayload({
              connected: false,
              error: {
                code: isAbortError ? 'API_HEALTH_TIMEOUT' : 'API_HEALTH_UNREACHABLE',
                message: isAbortError ? 'Timed out waiting for API health response' : error.message,
                category: 'network',
                retryable: true,
                details: {
                  timeout_ms: HEALTH_TIMEOUT_MS,
                },
              },
            }),
          );
        })
        .finally(() => {
          clearTimeout(timeout);
        });
    });
  },
};

export default defineConfig({
  plugins: [react(), healthPlugin],
  server: {
    port: UI_PORT,
    host: process.env.VITE_SERVER_HOST ?? process.env.VITE_HOST ?? process.env.UI_HOST ?? true,
    proxy: {
      '/api': {
        target: proxyTarget,
        changeOrigin: true,
        secure: false,
      },
      '/ws': {
        target: wsProxyTarget,
        ws: true,
        changeOrigin: true,
        secure: false,
      },
    },
  },
});
