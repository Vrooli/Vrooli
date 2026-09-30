import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const apiPort = env.API_PORT || process.env.API_PORT || '8090';
  // Resolved here rather than in the npm script: package scripts run through
  // cmd.exe on Windows, which cannot expand ${UI_PORT:-4173}.
  const uiPort = Number(process.env.UI_PORT) || 4173;

  return {
    plugins: [react()],
    preview: {
      host: '0.0.0.0',
      port: uiPort,
    },
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
