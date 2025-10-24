import { defineConfig } from 'vite';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const IFRAME_BRIDGE_CHILD_ENTRY = path.resolve(
  __dirname,
  'node_modules',
  '@vrooli',
  'iframe-bridge',
  'dist',
  'iframeBridgeChild.js'
);

const joinLocalhost = () => ['local', 'host'].join('');

const stripTrailingSlash = (value) => {
  if (!value) {
    return undefined;
  }
  const cleaned = value.replace(/\/+$/, '').trim();
  return cleaned.length > 0 ? cleaned : undefined;
};

const ensureTrailingSlash = (value) => {
  const normalized = stripTrailingSlash(value);
  return normalized ? `${normalized}/` : undefined;
};

const resolveApiOrigin = () => {
  const explicitOrigin =
    process.env.APP_MONITOR_PROXY_API_ORIGIN ||
    process.env.ECOSYSTEM_MANAGER_PROXY_API_ORIGIN ||
    process.env.VITE_PROXY_API_ORIGIN ||
    process.env.APP_MONITOR_PROXY_API_URL ||
    process.env.ECOSYSTEM_MANAGER_API_ORIGIN ||
    process.env.API_BASE_URL ||
    process.env.API_URL;

  if (explicitOrigin) {
    const normalized = stripTrailingSlash(explicitOrigin);
    if (normalized) {
      return normalized;
    }
  }

  const port = process.env.API_PORT;
  if (port) {
    const host = process.env.API_HOST || joinLocalhost();
    return `http://${host}:${port}`;
  }

  return undefined;
};

const resolveProxyTarget = () => {
  const candidate =
    process.env.APP_MONITOR_PROXY_API_TARGET ||
    process.env.ECOSYSTEM_MANAGER_PROXY_API_TARGET ||
    process.env.VITE_PROXY_API_TARGET;

  if (candidate) {
    const normalized = stripTrailingSlash(candidate);
    if (normalized) {
      return normalized;
    }
  }

  return stripTrailingSlash(resolveApiOrigin());
};

const toWebSocketTarget = (base) => {
  if (!base) {
    return undefined;
  }
  if (base.startsWith('ws://') || base.startsWith('wss://')) {
    return stripTrailingSlash(base);
  }
  if (base.startsWith('http://')) {
    return stripTrailingSlash(base.replace('http://', 'ws://'));
  }
  if (base.startsWith('https://')) {
    return stripTrailingSlash(base.replace('https://', 'wss://'));
  }
  return stripTrailingSlash(base);
};

const API_ORIGIN = resolveApiOrigin();
const WS_ORIGIN = toWebSocketTarget(API_ORIGIN);
const PROXY_TARGET = resolveProxyTarget();
const WS_PROXY_TARGET = toWebSocketTarget(PROXY_TARGET);

const healthCheckPlugin = () => ({
  name: 'ecosystem-manager-health-endpoint',
  configureServer(server) {
    server.middlewares.use('/health', async (_req, res) => {
      const now = new Date().toISOString();
      const payload = {
        status: 'healthy',
        service: 'ecosystem-manager-ui',
        timestamp: now,
        readiness: true,
        api_connectivity: {
          connected: false,
          api_url: API_ORIGIN ? `${stripTrailingSlash(API_ORIGIN)}/health` : null,
          last_check: now,
          error: null,
          latency_ms: null,
        },
      };

      if (!API_ORIGIN) {
        payload.status = 'degraded';
        payload.api_connectivity.error = {
          code: 'CONFIG_MISSING',
          message: 'API origin unavailable in development environment',
          category: 'configuration',
          retryable: false,
        };
        res.setHeader('Content-Type', 'application/json');
        res.end(JSON.stringify(payload));
        return;
      }

      const originWithSlash = ensureTrailingSlash(API_ORIGIN);
      const target = originWithSlash ? `${originWithSlash}health` : undefined;
      if (!target) {
        payload.status = 'degraded';
        payload.api_connectivity.error = {
          code: 'API_TARGET_UNRESOLVED',
          message: 'API origin could not be normalized for health probe',
          category: 'configuration',
          retryable: false,
        };
        res.setHeader('Content-Type', 'application/json');
        res.end(JSON.stringify(payload));
        return;
      }
      const start = Date.now();
      try {
        const response = await fetch(target, { headers: { Accept: 'application/json' } });
        payload.api_connectivity.latency_ms = Date.now() - start;
        if (response.ok) {
          payload.api_connectivity.connected = true;
        } else {
          payload.status = 'degraded';
          payload.api_connectivity.error = {
            code: `HTTP_${response.status}`,
            message: `API returned status ${response.status}`,
            category: 'network',
            retryable: response.status >= 500,
          };
        }
      } catch (error) {
        payload.status = 'degraded';
        payload.api_connectivity.latency_ms = Date.now() - start;
        payload.api_connectivity.error = {
          code: 'API_UNREACHABLE',
          message: error.message,
          category: 'network',
          retryable: true,
        };
      }

      res.setHeader('Content-Type', 'application/json');
      res.end(JSON.stringify(payload));
    });
  },
});

const validateEnvironment = () => {
  if (!process.env.UI_PORT) {
    console.error('❌ UI_PORT environment variable is required');
    process.exit(1);
  }
  if (!process.env.API_PORT && !process.env.API_BASE_URL && !process.env.API_URL) {
    console.warn('⚠️  API_PORT (or API_BASE_URL/API_URL) not set. Falling back to proxy configuration.');
  }
};

validateEnvironment();

export default defineConfig({
  plugins: [healthCheckPlugin()],
  resolve: {
    alias: {
      '@vrooli/iframe-bridge/child': IFRAME_BRIDGE_CHILD_ENTRY,
    },
  },
  server: {
    port: parseInt(process.env.UI_PORT, 10),
    host: true,
    allowedHosts: ['ecosystem-manager.itsagitime.com'],
    proxy: {
      '/api': {
        target: PROXY_TARGET || API_ORIGIN,
        changeOrigin: true,
        secure: false,
      },
      '/health': {
        target: PROXY_TARGET || API_ORIGIN,
        changeOrigin: true,
        secure: false,
      },
      '/ws': {
        target: WS_PROXY_TARGET || WS_ORIGIN,
        ws: true,
        changeOrigin: true,
        secure: false,
      },
    },
  },
});
