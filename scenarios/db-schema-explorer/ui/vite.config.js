import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: parseInt(process.env.UI_PORT || '35000'),
    host: true,
    proxy: {
      '/api': {
        target: `http://localhost:${process.env.API_PORT || '15000'}`,
        changeOrigin: true,
      },
      '/health': {
        target: `http://localhost:${process.env.API_PORT || '15000'}`,
        changeOrigin: true,
      }
    }
  },
  build: {
    outDir: 'dist',
    sourcemap: true
  }
})