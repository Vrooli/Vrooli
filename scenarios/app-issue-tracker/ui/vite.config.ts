import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

const apiPort = process.env.API_PORT || '8090';

export default defineConfig({
  plugins: [react()],
  define: {
    // Expose API port to the frontend for WebSocket connections
    __API_PORT__: JSON.stringify(apiPort),
  },
  server: {
    host: '0.0.0.0',
    port: Number(process.env.UI_PORT) || 5173,
    allowedHosts: ['app-issue-tracker.itsagitime.com'],
    proxy: {
      '/api': {
        target: `http://localhost:${apiPort}`,
        changeOrigin: true,
        ws: true, // Enable WebSocket proxying
      },
      '/health': {
        target: `http://localhost:${apiPort}`,
        changeOrigin: true,
      },
    },
  },
});
