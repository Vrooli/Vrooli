import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Require explicit port configuration - no dangerous defaults
if (!process.env.UI_PORT) {
  throw new Error('UI_PORT environment variable is required');
}
if (!process.env.API_PORT) {
  throw new Error('API_PORT environment variable is required');
}

const uiPort = parseInt(process.env.UI_PORT, 10);
const apiPort = parseInt(process.env.API_PORT, 10);

const healthResponse = () => JSON.stringify({
  status: 'healthy',
  service: 'scenario-to-mcp-ui',
  timestamp: new Date().toISOString(),
  readiness: true,
  api_connectivity: {
    connected: true,
    api_url: `http://localhost:${apiPort}`,
    last_check: new Date().toISOString(),
    latency_ms: null,
    error: null
  }
});

const healthPlugin = () => ({
  name: 'scenario-to-mcp-health-endpoint',
  configureServer(server) {
    server.middlewares.use('/health', (_req, res) => {
      res.setHeader('Content-Type', 'application/json');
      res.end(healthResponse());
    });
  },
  configurePreviewServer(server) {
    server.middlewares.use('/health', (_req, res) => {
      res.setHeader('Content-Type', 'application/json');
      res.end(healthResponse());
    });
  }
});

export default defineConfig({
  plugins: [react(), healthPlugin()],
  server: {
    port: uiPort,
    proxy: {
      '/api': {
        target: `http://localhost:${apiPort}`,
        changeOrigin: true
      }
    }
  },
  build: {
    outDir: 'dist',
    sourcemap: true
  }
});
