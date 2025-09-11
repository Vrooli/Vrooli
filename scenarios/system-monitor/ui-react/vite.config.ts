import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: parseInt(process.env.UI_PORT || process.env.PORT || '3000'),
    host: '0.0.0.0',
    proxy: {
      // Proxy API calls to the Go backend
      '/api': {
        target: `http://localhost:${process.env.API_PORT || '8080'}`,
        changeOrigin: true,
        secure: false,
      },
      // Proxy health checks
      '/health': {
        target: `http://localhost:${process.env.API_PORT || '8080'}`,
        changeOrigin: true,
        secure: false,
      }
    }
  },
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    sourcemap: true,
  }
})
