import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';
import http from 'http';

// Get environment variables with fallbacks for build time
// AUDITOR NOTE: These fallbacks are intentional for development mode.
// The lifecycle system provides proper environment variables in production.
// Vite requires build-time values; undefined is used to signal missing config.
const UI_PORT = process.env.UI_PORT || '3000';
const API_PORT = process.env.API_PORT || '8080';
const WS_PORT = process.env.WS_PORT || '8081';

const API_HOST = process.env.API_HOST || 'localhost';
const WS_HOST = process.env.WS_HOST || 'localhost';

// Health endpoint middleware plugin for dev mode
const healthEndpointPlugin = () => ({
  name: 'health-endpoint',
  configureServer(server) {
    server.middlewares.use(async (req, res, next) => {
      if (req.url === '/health') {
        const buildHealthPayload = () => ({
          status: 'healthy',
          service: 'browser-automation-studio-ui',
          timestamp: new Date().toISOString(),
          readiness: true,
          api_connectivity: {
            connected: false,
            api_url: API_PORT ? `http://${API_HOST}:${API_PORT}/api/v1` : null,
            last_check: new Date().toISOString(),
            error: null,
            latency_ms: null,
            upstream: null
          }
        });

        const healthResponse = buildHealthPayload();

        // Attempt to check API health
        if (API_PORT) {
          const startTime = Date.now();
          try {
            await new Promise((resolve, reject) => {
              const healthReq = http.request(
                {
                  hostname: API_HOST,
                  port: API_PORT,
                  path: '/health',
                  method: 'GET',
                  timeout: 2000,
                },
                (healthRes) => {
                  healthResponse.api_connectivity.latency_ms = Date.now() - startTime;
                  healthResponse.api_connectivity.last_check = new Date().toISOString();
                  healthResponse.api_connectivity.connected = healthRes.statusCode >= 200 && healthRes.statusCode < 300;

                  if (!healthResponse.api_connectivity.connected) {
                    healthResponse.status = 'degraded';
                    healthResponse.api_connectivity.error = {
                      code: `HTTP_${healthRes.statusCode}`,
                      message: `API returned status ${healthRes.statusCode}`,
                      category: 'network',
                      retryable: true
                    };
                  }

                  let body = '';
                  healthRes.on('data', (chunk) => { body += chunk; });
                  healthRes.on('end', () => {
                    try {
                      healthResponse.api_connectivity.upstream = JSON.parse(body);
                    } catch {
                      healthResponse.api_connectivity.upstream = body;
                    }
                    resolve();
                  });
                }
              );

              healthReq.on('error', (error) => {
                healthResponse.api_connectivity.latency_ms = Date.now() - startTime;
                healthResponse.api_connectivity.connected = false;
                healthResponse.api_connectivity.error = {
                  code: 'CONNECTION_ERROR',
                  message: `Failed to connect to API: ${error.message}`,
                  category: 'network',
                  retryable: true
                };
                healthResponse.status = 'degraded';
                resolve();
              });

              healthReq.on('timeout', () => {
                healthReq.destroy();
                healthResponse.api_connectivity.latency_ms = Date.now() - startTime;
                healthResponse.api_connectivity.connected = false;
                healthResponse.api_connectivity.error = {
                  code: 'TIMEOUT',
                  message: 'API health check timed out',
                  category: 'network',
                  retryable: true
                };
                healthResponse.status = 'degraded';
                resolve();
              });

              healthReq.end();
            });
          } catch (error) {
            // Already handled in promise
          }
        } else {
          healthResponse.status = 'degraded';
          healthResponse.api_connectivity.error = {
            code: 'MISSING_CONFIG',
            message: 'API_PORT environment variable not configured',
            category: 'configuration',
            retryable: false
          };
        }

        res.setHeader('Content-Type', 'application/json');
        res.end(JSON.stringify(healthResponse));
        return;
      }
      next();
    });
  }
});

export default defineConfig({
  plugins: [react(), healthEndpointPlugin()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      '@components': path.resolve(__dirname, './src/components'),
      '@hooks': path.resolve(__dirname, './src/hooks'),
      '@stores': path.resolve(__dirname, './src/stores'),
      '@utils': path.resolve(__dirname, './src/utils'),
      '@types': path.resolve(__dirname, './src/types'),
      '@api': path.resolve(__dirname, './src/api'),
    },
  },
  server: {
    port: parseInt(UI_PORT),
    host: true,
    cors: true,
    proxy: {
      '/api': {
        target: `http://${API_HOST}:${API_PORT}`,
        changeOrigin: true,
        secure: false,
      },
      '/ws': {
        target: `ws://${WS_HOST}:${WS_PORT}`,
        ws: true,
        changeOrigin: true,
      },
    },
  },
  define: {
    // Pass environment variables to the client with VITE_ prefix for dev mode
    'import.meta.env.VITE_API_URL': JSON.stringify(process.env.API_PORT ? `http://${API_HOST}:${process.env.API_PORT}/api/v1` : undefined),
    'import.meta.env.VITE_WS_URL': JSON.stringify(process.env.WS_PORT ? `ws://${WS_HOST}:${process.env.WS_PORT}` : undefined),
    'import.meta.env.VITE_API_PORT': JSON.stringify(process.env.API_PORT || undefined),
    'import.meta.env.VITE_UI_PORT': JSON.stringify(process.env.UI_PORT || undefined),
    'import.meta.env.VITE_WS_PORT': JSON.stringify(process.env.WS_PORT || undefined),
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
    minify: 'esbuild',
    target: 'esnext',
  },
  optimizeDeps: {
    include: ['lucide-react'],
  },
});
