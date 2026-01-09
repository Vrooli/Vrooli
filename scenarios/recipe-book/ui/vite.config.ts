import { defineConfig } from 'vite';
import path from 'node:path';

export default defineConfig({
  root: '.',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  resolve: {
    preserveSymlinks: true,
    alias: {
      '@vrooli/iframe-bridge/child': path.resolve(__dirname, '../../../packages/iframe-bridge/dist/iframeBridgeChild.js'),
      '@vrooli/iframe-bridge': path.resolve(__dirname, '../../../packages/iframe-bridge/dist/index.js'),
    },
  },
  server: {
    host: '0.0.0.0',
    port: Number(process.env.UI_PORT) || 5173,
    strictPort: false,
  },
});
