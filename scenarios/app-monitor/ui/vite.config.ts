import react from '@vitejs/plugin-react';
import path from 'path';
import { defineConfig, loadEnv } from 'vite';

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  // Load env file based on `mode` in the current working directory.
  const env = loadEnv(mode, process.cwd(), '');

  return {
    base: './',  // Required for universal deployment (proxied scenarios)
    plugins: [react()],
    server: {
      port: parseInt(env.VITE_PORT),
      host: true, // Allow external connections
      proxy: {
        '/api': {
          target: `http://localhost:${env.API_PORT}`,
          changeOrigin: true,
        },
        '/ws': {
          target: `ws://localhost:${env.API_PORT}`,
          ws: true,
          changeOrigin: true,
        },
      },
    },
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
        '@components': path.resolve(__dirname, './src/components'),
        '@hooks': path.resolve(__dirname, './src/hooks'),
        '@services': path.resolve(__dirname, './src/services'),
        '@types': path.resolve(__dirname, './src/types'),
        '@utils': path.resolve(__dirname, './src/utils'),
      },
    },
    build: {
      outDir: 'dist',
      sourcemap: true,
    },
  }
})