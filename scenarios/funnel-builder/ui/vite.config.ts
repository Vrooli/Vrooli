import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: parseInt(process.env.FUNNEL_UI_PORT || '20000'),
    proxy: {
      '/api': {
        target: `http://localhost:${process.env.FUNNEL_API_PORT || '15000'}`,
        changeOrigin: true
      }
    }
  }
})