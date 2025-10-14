import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Get ports from environment variables with validation
const UI_PORT = parseInt(process.env.UI_PORT);
const API_PORT = parseInt(process.env.API_PORT);

// Validate required environment variables (fail fast)
if (!process.env.UI_PORT || isNaN(UI_PORT)) {
  console.error('ERROR: UI_PORT environment variable is required and must be a valid port number');
  process.exit(1);
}

if (!process.env.API_PORT || isNaN(API_PORT)) {
  console.error('ERROR: API_PORT environment variable is required and must be a valid port number');
  process.exit(1);
}

const API_BASE_URL = `http://localhost:${API_PORT}`;
const WS_BASE_URL = `ws://localhost:${API_PORT}`;

// Health check middleware for interoperability with Vrooli lifecycle orchestration
// The /health endpoint is standardized across all scenarios for monitoring and orchestration
function healthCheckPlugin() {
  return {
    name: 'health-check',
    configureServer(server) {
      server.middlewares.stack.unshift({
        route: '',
        handle: async (req, res, next) => {
          if (req.url === '/health') {
            // Test API connectivity
            let apiConnected = false;
            let apiError = null;
            let latencyMs = null;
            const checkTimestamp = new Date().toISOString();

            try {
              const apiUrl = `http://localhost:${API_PORT}/health`;
              const startTime = Date.now();
              const response = await fetch(apiUrl, {
                signal: AbortSignal.timeout(3000)
              });

              if (response.ok) {
                apiConnected = true;
                latencyMs = Date.now() - startTime;
              } else {
                apiError = {
                  code: `HTTP_${response.status}`,
                  message: `API returned status ${response.status}`,
                  category: 'network',
                  retryable: true
                };
              }
            } catch (error) {
              const errorMessage = error instanceof Error ? error.message : 'unknown error';
              apiError = {
                code: errorMessage.includes('timeout') ? 'TIMEOUT' : 'CONNECTION_REFUSED',
                message: `Failed to connect to API: ${errorMessage}`,
                category: 'network',
                retryable: true
              };
            }

            const overallStatus = apiConnected ? 'healthy' : 'degraded';
            res.statusCode = apiConnected ? 200 : 503;
            res.setHeader('Content-Type', 'application/json');
            res.end(JSON.stringify({
              status: overallStatus,
              service: 'financial-calculators-hub-ui',
              timestamp: new Date().toISOString(),
              readiness: true,
              api_connectivity: {
                connected: apiConnected,
                api_url: `http://localhost:${API_PORT}`,
                last_check: checkTimestamp,
                error: apiError,
                latency_ms: latencyMs
              }
            }));
            return;
          }
          next();
        }
      });
    }
  };
}

export default defineConfig({
  plugins: [react(), healthCheckPlugin()],
  server: {
    port: UI_PORT,
    host: true,
    proxy: {
      '/api': {
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
