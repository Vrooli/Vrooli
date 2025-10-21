import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Get ports from environment variables
const UI_PORT = parseInt(process.env.UI_PORT);
const API_PORT = parseInt(process.env.API_PORT);

const API_BASE_URL = `http://localhost:${API_PORT}`;
const WS_BASE_URL = `ws://localhost:${API_PORT}`;

const createHealthMiddleware = (serviceName = 'graph-studio-ui') => (req: any, res: any, next: any) => {
  if (!req?.url?.startsWith('/health')) {
    return next();
  }

  res.setHeader('Content-Type', 'application/json');
  res.end(
    JSON.stringify({
      status: 'healthy',
      service: serviceName,
      timestamp: new Date().toISOString()
    })
  );
};

const healthPlugin = {
  name: 'graph-studio-health-endpoint',
  configureServer(server: any) {
    server.middlewares.use(createHealthMiddleware());
  },
  configurePreviewServer(server: any) {
    server.middlewares.use(createHealthMiddleware());
  }
};

export default defineConfig({
  plugins: [react(), healthPlugin],
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
