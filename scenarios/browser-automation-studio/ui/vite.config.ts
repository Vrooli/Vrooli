import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

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
    port: parseInt(process.env.BROWSER_AUTOMATION_UI_PORT || '3090'),
    host: true,
    proxy: {
      '/api': {
        target: `http://localhost:${process.env.BROWSER_AUTOMATION_API_PORT || '8090'}`,
        changeOrigin: true,
      },
      '/ws': {
        target: `ws://localhost:${process.env.BROWSER_AUTOMATION_WS_PORT || '8091'}`,
        ws: true,
        changeOrigin: true,
      },
    },
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