import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

// Get environment variables with fallbacks for build time
const UI_PORT = process.env.UI_PORT || '3000';
const API_PORT = process.env.API_PORT || '8080';
const WS_PORT = process.env.WS_PORT || '8081';

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
    port: parseInt(UI_PORT),
    host: true,
    proxy: {
      '/api': {
        target: `http://${API_HOST}:${API_PORT}`,
        changeOrigin: true,
        secure: false,
      },
      '/ws': {
        target: `ws://${WS_HOST}:${WS_PORT}`,
        ws: true,
        changeOrigin: true,
      },
    },
  },
  define: {
    // Pass environment variables to the client with VITE_ prefix for dev mode
    'import.meta.env.VITE_API_URL': JSON.stringify(process.env.API_PORT ? `http://${API_HOST}:${process.env.API_PORT}/api/v1` : undefined),
    'import.meta.env.VITE_WS_URL': JSON.stringify(process.env.WS_PORT ? `ws://${WS_HOST}:${process.env.WS_PORT}` : undefined),
    'import.meta.env.VITE_API_PORT': JSON.stringify(process.env.API_PORT || undefined),
    'import.meta.env.VITE_UI_PORT': JSON.stringify(process.env.UI_PORT || undefined),
    'import.meta.env.VITE_WS_PORT': JSON.stringify(process.env.WS_PORT || undefined),
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