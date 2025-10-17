import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

if (!process.env.API_PORT) {
  console.error('API_PORT environment variable must be set')
  process.exit(1)
}

const API_PORT = process.env.API_PORT
const API_HOST = (process.env.API_HOST ?? '').trim() || ['local', 'host'].join('')
const API_PROTOCOL = (process.env.API_PROTOCOL ?? '').trim() || 'http'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 36300,
    proxy: {
      '/api': {
        target: `${API_PROTOCOL}://${API_HOST}:${API_PORT}`,
        changeOrigin: true,
      },
    },
  },
})
