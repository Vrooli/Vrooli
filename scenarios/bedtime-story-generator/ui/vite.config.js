import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Get ports from environment variables
const UI_PORT = parseInt(process.env.UI_PORT) || 38899;
const API_PORT = parseInt(process.env.API_PORT) || 16902;

const API_BASE_URL = `http://localhost:${API_PORT}`;
const WS_BASE_URL = `ws://localhost:${API_PORT}`;

export default defineConfig({
  plugins: [react()],
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
