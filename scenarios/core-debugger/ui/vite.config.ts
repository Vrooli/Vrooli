import path from 'node:path';
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig(() => {
  const apiPort = process.env.VITE_API_PORT || process.env.API_PORT;
  const proxyTarget = process.env.VITE_API_PROXY || (apiPort ? `http://localhost:${apiPort}` : undefined);

  return {
    plugins: [react()],
    resolve: {
      alias: {
        '@': path.resolve(__dirname, 'src')
      }
    },
    server: {
      host: '0.0.0.0',
      port: Number(process.env.VITE_PORT) || 5173,
      strictPort: false,
      proxy: proxyTarget
        ? {
            '/api': {
              target: proxyTarget,
              changeOrigin: true,
              secure: false
            }
          }
        : undefined
    },
    preview: {
      host: '0.0.0.0',
      port: Number(process.env.UI_PORT) || 4173
    },
    build: {
      outDir: 'dist',
      sourcemap: true
    }
  };
});
