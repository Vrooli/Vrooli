import { defineConfig } from 'vite';

// Get ports from environment variables
const UI_PORT = parseInt(process.env.UI_PORT);
const API_PORT = parseInt(process.env.API_PORT);

const API_BASE_URL = `http://localhost:${API_PORT}`;
const WS_BASE_URL = `ws://localhost:${API_PORT}`;

export default defineConfig({
  plugins: [
    {
      name: 'health-endpoint',
      configureServer(server) {
        server.middlewares.use('/health', async (req, res) => {
          const healthResponse = {
            status: 'healthy',
            service: 'ecosystem-manager-ui',
            timestamp: new Date().toISOString(),
            readiness: true,
            api_connectivity: {
              connected: false,
              api_url: `http://localhost:${API_PORT}`,
              last_check: new Date().toISOString(),
              error: null,
              latency_ms: null
            }
          };

          // Check API connectivity
          const startTime = Date.now();
          try {
            const response = await fetch(`http://localhost:${API_PORT}/health`);
            if (response.ok) {
              healthResponse.api_connectivity.connected = true;
              healthResponse.api_connectivity.latency_ms = Date.now() - startTime;
            } else {
              healthResponse.api_connectivity.error = {
                code: `HTTP_${response.status}`,
                message: `API returned status ${response.status}`,
                category: 'network',
                retryable: response.status >= 500
              };
            }
          } catch (error) {
            healthResponse.api_connectivity.connected = false;
            healthResponse.api_connectivity.latency_ms = Date.now() - startTime;
            healthResponse.api_connectivity.error = {
              code: 'CONNECTION_FAILED',
              message: error.message,
              category: 'network',
              retryable: true
            };
            healthResponse.status = 'degraded';
          }

          res.setHeader('Content-Type', 'application/json');
          res.end(JSON.stringify(healthResponse));
        });
      }
    }
  ],
  server: {
    port: UI_PORT,
    host: true,
    allowedHosts: ['ecosystem-manager.itsagitime.com'],
    proxy: {
      '/api': {
        target: API_BASE_URL,
        changeOrigin: true,
        secure: false,
      },
      '/health': {
        target: API_BASE_URL,
        changeOrigin: true,
        secure: false,
      },
      '/ws': {
        target: WS_BASE_URL,
        ws: true,
        changeOrigin: true,
        secure: false,
      }
    }
  }
});
