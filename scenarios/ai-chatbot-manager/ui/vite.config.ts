import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const apiPort = env.API_PORT || process.env.API_PORT || '8090';

  return {
    plugins: [react()],
    server: {
      host: '0.0.0.0',
      port: Number(process.env.UI_PORT) || 5173,
      proxy: {
        '/api': {
          target: `http://localhost:${apiPort}`,
          changeOrigin: true,
        },
        '/ws': {
          target: `http://localhost:${apiPort}`,
          changeOrigin: true,
          ws: true,
        },
      },
    },
  };
});
