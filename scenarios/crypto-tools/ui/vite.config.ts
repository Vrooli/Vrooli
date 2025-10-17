import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// Validate required environment variables at build time
if (!process.env.UI_PORT) {
  console.error('❌ UI_PORT environment variable is required');
  process.exit(1);
}

if (!process.env.API_PORT) {
  console.error('❌ API_PORT environment variable is required');
  process.exit(1);
}

const uiPort = parseInt(process.env.UI_PORT);
const apiPort = parseInt(process.env.API_PORT);

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src')
    }
  },
  server: {
    port: uiPort,
    host: true,
    proxy: {
      '/api': {
        target: `http://localhost:${apiPort}`,
        changeOrigin: true
      },
      '/health': {
        target: `http://localhost:${apiPort}`,
        changeOrigin: true
      }
    }
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
    rollupOptions: {
      output: {
        manualChunks: {
          'react-vendor': ['react', 'react-dom'],
          'crypto-vendor': ['crypto-js'],
          'query-vendor': ['@tanstack/react-query', 'axios']
        }
      }
    }
  }
})