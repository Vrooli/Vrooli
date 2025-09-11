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
    port: parseInt(process.env.UI_PORT || '35001'),
    host: true,
    proxy: {
      '/api': {
        target: `http://localhost:${process.env.API_PORT || '15001'}`,
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
          'date-vendor': ['date-fns'],
          'query-vendor': ['@tanstack/react-query', 'axios']
        }
      }
    }
  }
})