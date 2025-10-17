import { defineConfig, PluginOption } from 'vite';
import react from '@vitejs/plugin-react';

// Health check middleware plugin
const healthCheckPlugin = (): PluginOption => ({
  name: 'health-check',
  configureServer(server) {
    server.middlewares.use((req, res, next) => {
      if (req.url === '/health') {
        const apiPort = process.env.VITE_API_PORT || process.env.API_PORT || '30400';
        const apiUrl = `http://localhost:${apiPort}/api/v1`;

        // Perform a simple connectivity check to the API
        fetch(`${apiUrl}/health`)
          .then(response => {
            const connected = response.ok;
            const healthResponse = {
              status: connected ? 'healthy' : 'degraded',
              service: 'elo-swipe-ui',
              timestamp: new Date().toISOString(),
              readiness: true,
              api_connectivity: {
                connected,
                api_url: apiUrl,
                last_check: new Date().toISOString(),
                error: connected ? null : {
                  code: 'API_UNAVAILABLE',
                  message: `API returned status ${response.status}`,
                  category: 'network',
                  retryable: true
                },
                latency_ms: null
              }
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
                api_url: apiUrl,
                last_check: new Date().toISOString(),
                error: {
                  code: 'CONNECTION_REFUSED',
                  message: error.message || 'Failed to connect to API',
                  category: 'network',
                  retryable: true
                },
                latency_ms: null
              }
            };

            res.setHeader('Content-Type', 'application/json');
            res.statusCode = 200;
            res.end(JSON.stringify(healthResponse));
          });
      } else {
        next();
      }
    });
  }
});

export default defineConfig(({ mode }) => {
  const apiPort = process.env.VITE_API_PORT || process.env.API_PORT || '30400';

  return {
    plugins: [react(), healthCheckPlugin()],
    envPrefix: ['VITE_', 'API_'],
    resolve: {
      preserveSymlinks: true,
    },
    define: {
      __API_URL__: JSON.stringify(process.env.VITE_API_URL || `http://localhost:${apiPort}/api/v1`),
    },
    server: {
      proxy: {
        '/api': {
          target: `http://localhost:${apiPort}`,
          changeOrigin: true,
        },
      },
    },
    preview: {
      proxy: {
        '/api': {
          target: `http://localhost:${apiPort}`,
          changeOrigin: true,
        },
      },
    },
    build: {
      outDir: 'dist',
      sourcemap: mode !== 'production',
    },
  };
});
