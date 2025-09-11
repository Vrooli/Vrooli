import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Validate required environment variables
const requiredEnvVars = ['VITE_API_URL', 'VITE_WS_URL']
const missingVars = requiredEnvVars.filter(varName => !process.env[varName])

if (missingVars.length > 0) {
  throw new Error(`Missing required environment variables: ${missingVars.join(', ')}`)
}

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: parseInt(process.env.UI_PORT || '9101'),
    host: true,
    proxy: {
      '/api': {
        target: process.env.VITE_API_URL,
        changeOrigin: true,
      },
      '/ws': {
        target: process.env.VITE_WS_URL,
        ws: true,
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
})