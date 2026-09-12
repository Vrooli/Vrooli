import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    host: '0.0.0.0',
    port: Number.parseInt(process.env.UI_PORT || '5173', 10),
  },
  // Resolved here rather than in the npm script: package scripts run through
  // cmd.exe on Windows, which cannot expand ${UI_PORT:-4173}.
  preview: {
    host: '0.0.0.0',
    port: Number.parseInt(process.env.UI_PORT || '4173', 10),
  },
});
