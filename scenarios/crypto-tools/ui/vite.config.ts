import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src')
    }
  },
  server: {
    port: parseInt(process.env.UI_PORT || (() => {
      console.error('❌ UI_PORT environment variable is required');
      process.exit(1);
    })()),
    host: true,
    proxy: {
      '/api': {
        target: `http://localhost:${process.env.API_PORT || (() => {
          console.error('❌ API_PORT environment variable is required');
          process.exit(1);
        })()}`,
        changeOrigin: true
      },
      '/health': {
        target: `http://localhost:${process.env.API_PORT || '8090'}`,
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