import { defineConfig, PluginOption } from 'vite';
import react from '@vitejs/plugin-react';

const API_BASE_PATH = '/api/v1';

const stripTrailingSlash = (value: string): string => {
  let result = value;
  while (result.endsWith('/') && !/^https?:\/\/$/iu.test(result)) {
    result = result.slice(0, -1);
  }
  return result;
};

const ensureLeadingSlash = (value: string): string => (value.startsWith('/') ? value : `/${value}`);

const joinUrlSegments = (base: string, segment: string): string => {
  const normalizedBase = stripTrailingSlash(base);
  const normalizedSegment = ensureLeadingSlash(segment);
  if (!normalizedBase) {
    return normalizedSegment;
  }
  return `${normalizedBase}${normalizedSegment}`;
};

const resolveApiProtocol = (): 'http' | 'https' => {
  const protocol = process.env.VITE_API_PROTOCOL ?? process.env.API_PROTOCOL;
  if (typeof protocol === 'string' && protocol.toLowerCase() === 'https') {
    return 'https';
  }
  return 'http';
};

const resolveApiHost = (): string => {
  const host = process.env.VITE_API_HOST ?? process.env.API_HOST;
  if (typeof host === 'string' && host.trim().length > 0) {
    return host.trim();
  }
  return '127.0.0.1';
};

const resolveApiPort = (): string => {
  const port = process.env.VITE_API_PORT ?? process.env.API_PORT;
  if (typeof port === 'string' && port.trim().length > 0) {
    return port.trim();
  }
  return '30400';
};

const resolveApiOrigin = (): string => {
  const protocol = resolveApiProtocol();
  const host = resolveApiHost();
  const port = resolveApiPort();
  return `${protocol}://${host}:${port}`;
};

const resolveApiUrl = (): string => {
  const explicit = process.env.VITE_API_URL ?? process.env.API_URL;
  if (typeof explicit === 'string' && explicit.trim().length > 0) {
    return stripTrailingSlash(explicit.trim());
  }
  return stripTrailingSlash(joinUrlSegments(resolveApiOrigin(), API_BASE_PATH));
};

const resolveProxyTarget = (): string => {
  const explicit = process.env.VITE_API_URL ?? process.env.API_URL;
  if (typeof explicit === 'string' && explicit.trim().length > 0) {
    try {
      return new URL(explicit).origin;
    } catch (error) {
      if (explicit.startsWith('/')) {
        return resolveApiOrigin();
      }
    }
  }
  return resolveApiOrigin();
};

// Health check middleware plugin
const healthCheckPlugin = (): PluginOption => ({
  name: 'health-check',
  configureServer(server) {
    server.middlewares.use((req, res, next) => {
      if (req.url === '/health') {
        const apiBase = resolveApiUrl();

        // Perform a simple connectivity check to the API
        fetch(`${apiBase}/health`)
          .then(response => {
            const connected = response.ok;
            const healthResponse = {
              status: connected ? 'healthy' : 'degraded',
              service: 'elo-swipe-ui',
              timestamp: new Date().toISOString(),
              readiness: true,
              api_connectivity: {
                connected,
                api_url: apiBase,
                last_check: new Date().toISOString(),
                error: connected
                  ? null
                  : {
                      code: 'API_UNAVAILABLE',
                      message: `API returned status ${response.status}`,
                      category: 'network',
                      retryable: true,
                    },
                latency_ms: null,
              },
            };

            res.setHeader('Content-Type', 'application/json');
            res.statusCode = 200;
            res.end(JSON.stringify(healthResponse));
          })
          .catch(error => {
            const healthResponse = {
              status: 'degraded',
              service: 'elo-swipe-ui',
              timestamp: new Date().toISOString(),
              readiness: true,
              api_connectivity: {
                connected: false,
                api_url: apiBase,
                last_check: new Date().toISOString(),
                error: {
                  code: 'CONNECTION_REFUSED',
                  message: error instanceof Error && error.message ? error.message : 'Failed to connect to API',
                  category: 'network',
                  retryable: true,
                },
                latency_ms: null,
              },
            };

            res.setHeader('Content-Type', 'application/json');
            res.statusCode = 200;
            res.end(JSON.stringify(healthResponse));
          });
      } else {
        next();
      }
    });
  },
});

export default defineConfig(({ mode }) => ({
  plugins: [react(), healthCheckPlugin()],
  envPrefix: ['VITE_', 'API_'],
  resolve: {
    preserveSymlinks: true,
  },
  define: {
    __API_URL__: JSON.stringify(process.env.VITE_API_URL ?? process.env.API_URL ?? ''),
  },
  server: {
    proxy: {
      '/api': {
        target: resolveProxyTarget(),
        changeOrigin: true,
      },
    },
  },
  preview: {
    proxy: {
      '/api': {
        target: resolveProxyTarget(),
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: mode !== 'production',
  },
}));
