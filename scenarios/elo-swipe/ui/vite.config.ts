import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig(({ mode }) => {
  const apiPort = process.env.VITE_API_PORT || process.env.API_PORT || '30400';

  return {
    plugins: [react()],
    envPrefix: ['VITE_', 'API_'],
    define: {
      __API_URL__: JSON.stringify(process.env.VITE_API_URL || `http://localhost:${apiPort}/api/v1`),
    },
    server: {
      proxy: {
        '/api': {
          target: `http://localhost:${apiPort}`,
          changeOrigin: true,
        },
      },
    },
    preview: {
      proxy: {
        '/api': {
          target: `http://localhost:${apiPort}`,
          changeOrigin: true,
        },
      },
    },
    build: {
      outDir: 'dist',
      sourcemap: mode !== 'production',
    },
  };
});
