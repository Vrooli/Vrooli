import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: process.env.UI_PORT || 3000,
    proxy: {
      '/api': {
        target: `http://localhost:${process.env.API_PORT || 15000}`,
        changeOrigin: true,
      }
    }
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['react', 'react-dom'],
          virtual: ['react-window', '@tanstack/react-virtual']
        }
      }
    }
  },
  optimizeDeps: {
    include: ['react', 'react-dom', 'react-window']
  }
})