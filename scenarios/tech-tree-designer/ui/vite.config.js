import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import http from 'node:http';

const UI_PORT = process.env.UI_PORT || '3000';
const API_HOST = process.env.API_HOST || 'localhost';
const API_PORT = process.env.API_PORT || '';
const API_HEALTH_URL = API_PORT ? `http://${API_HOST}:${API_PORT}/health` : `http://${API_HOST}/health`;

const healthEndpointPlugin = () => ({
  name: 'tech-tree-designer-health-endpoint',
  async configureServer(server) {
    server.middlewares.use(async (req, res, next) => {
      if (req.url !== '/health') {
        next();
        return;
      }

      const now = () => new Date().toISOString();
      const healthResponse = {
        status: 'healthy',
        service: 'tech-tree-designer-ui',
        timestamp: now(),
        readiness: true,
        version: process.env.npm_package_version || '1.0.0',
        api_connectivity: {
          connected: false,
          api_url: API_HEALTH_URL,
          last_check: now(),
          error: null,
          latency_ms: null
        }
      };

      if (API_PORT) {
        const startTime = Date.now();
        await new Promise((resolve) => {
          const request = http.get(API_HEALTH_URL, { timeout: 2000 }, (apiRes) => {
            const { statusCode = 0 } = apiRes;

            apiRes.resume();
            apiRes.on('end', () => {
              healthResponse.api_connectivity.latency_ms = Date.now() - startTime;
              healthResponse.api_connectivity.last_check = now();
              healthResponse.api_connectivity.connected = statusCode >= 200 && statusCode < 300;

              if (!healthResponse.api_connectivity.connected) {
                healthResponse.status = 'degraded';
                healthResponse.readiness = false;
                healthResponse.api_connectivity.error = {
                  code: `HTTP_${statusCode || 'UNKNOWN'}`,
                  message: `API health endpoint returned status ${statusCode || 'unknown'}`,
                  category: 'network',
                  retryable: true
                };
              } else {
                healthResponse.api_connectivity.error = null;
              }

              resolve();
            });
          });

          request.on('error', (error) => {
            healthResponse.status = 'degraded';
            healthResponse.readiness = false;
            healthResponse.api_connectivity.latency_ms = Date.now() - startTime;
            healthResponse.api_connectivity.connected = false;
            healthResponse.api_connectivity.last_check = now();
            healthResponse.api_connectivity.error = {
              code: 'CONNECTION_ERROR',
              message: `Failed to reach API health endpoint: ${error.message}`,
              category: 'network',
              retryable: true
            };
            resolve();
          });

          request.on('timeout', () => {
            request.destroy(new Error('timeout'));
          });
        });
      } else {
        healthResponse.status = 'degraded';
        healthResponse.readiness = false;
        healthResponse.api_connectivity.error = {
          code: 'MISSING_CONFIGURATION',
          message: 'API_PORT environment variable is not configured',
          category: 'configuration',
          retryable: false
        };
      }

      res.setHeader('Content-Type', 'application/json');
      res.end(JSON.stringify(healthResponse));
    });
  }
});

export default defineConfig({
  base: './',
  plugins: [react(), healthEndpointPlugin()],
  server: {
    port: parseInt(UI_PORT, 10),
    host: true,
    open: false
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['react', 'react-dom'],
          visualization: ['react-flow-renderer', 'd3'],
          ui: ['framer-motion', 'lucide-react', 'recharts']
        }
      }
    }
  },
  define: {
    __APP_VERSION__: JSON.stringify(process.env.npm_package_version || '1.0.0'),
    'import.meta.env.VITE_API_PORT': JSON.stringify(API_PORT),
    'import.meta.env.VITE_API_HOST': JSON.stringify(API_HOST),
    'import.meta.env.VITE_APP_VERSION': JSON.stringify(process.env.npm_package_version || '1.0.0')
  }
});
