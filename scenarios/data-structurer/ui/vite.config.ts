import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'node:path';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    host: true,
    port: Number(process.env.UI_PORT) || 5173,
  },
  preview: {
    host: true,
    port: Number(process.env.UI_PORT) || 4173,
  },
  build: {
    sourcemap: true,
  },
});
