import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

const UI_PORT = Number.parseInt(process.env.UI_PORT || '5173', 10);

export default defineConfig({
  plugins: [react()],
  server: {
    host: true,
    port: UI_PORT
  },
  preview: {
    host: true,
    port: UI_PORT
  },
  build: {
    outDir: 'dist',
    sourcemap: true
  }
});
