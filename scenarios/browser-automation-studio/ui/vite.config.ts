import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

// Validate required environment variables
const requiredEnvVars = ['UI_PORT', 'API_PORT', 'WS_PORT'];
for (const envVar of requiredEnvVars) {
  if (!process.env[envVar]) {
    throw new Error(`Required environment variable ${envVar} is not set. Please run through Vrooli lifecycle system.`);
  }
}

const API_HOST = process.env.API_HOST || 'localhost';
const WS_HOST = process.env.WS_HOST || 'localhost';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      '@components': path.resolve(__dirname, './src/components'),
      '@hooks': path.resolve(__dirname, './src/hooks'),
      '@stores': path.resolve(__dirname, './src/stores'),
      '@utils': path.resolve(__dirname, './src/utils'),
      '@types': path.resolve(__dirname, './src/types'),
      '@api': path.resolve(__dirname, './src/api'),
    },
  },
  server: {
    port: parseInt(process.env.UI_PORT),
    host: true,
    proxy: {
      '/api': {
        target: `http://${API_HOST}:${process.env.API_PORT}`,
        changeOrigin: true,
        secure: false,
      },
      '/ws': {
        target: `ws://${WS_HOST}:${process.env.WS_PORT}`,
        ws: true,
        changeOrigin: true,
      },
    },
  },
  define: {
    // Pass environment variables to the client with VITE_ prefix
    'import.meta.env.VITE_API_URL': JSON.stringify(`http://${API_HOST}:${process.env.API_PORT}/api/v1`),
    'import.meta.env.VITE_WS_URL': JSON.stringify(`ws://${WS_HOST}:${process.env.WS_PORT}`),
    'import.meta.env.VITE_API_PORT': JSON.stringify(process.env.API_PORT),
    'import.meta.env.VITE_UI_PORT': JSON.stringify(process.env.UI_PORT),
    'import.meta.env.VITE_WS_PORT': JSON.stringify(process.env.WS_PORT),
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
    minify: 'esbuild',
    target: 'esnext',
  },
  optimizeDeps: {
    exclude: ['lucide-react'],
  },
});