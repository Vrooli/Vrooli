import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Get ports from environment variables. The default lives here rather than in
// the npm script: package scripts run through cmd.exe on Windows, which cannot
// expand ${UI_PORT:-3252}.
const UI_PORT = parseInt(process.env.UI_PORT) || 3252;
const API_PORT = parseInt(process.env.API_PORT);

const API_BASE_URL = `http://localhost:${API_PORT}`;
const WS_BASE_URL = `ws://localhost:${API_PORT}`;

export default defineConfig({
  plugins: [react()],
  preview: {
    port: UI_PORT,
    host: true,
  },
  server: {
    port: UI_PORT,
    host: true,
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
