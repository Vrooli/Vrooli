import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: process.env.UI_PORT || 20101,
    host: true
  },
  define: {
    'process.env.API_PORT': JSON.stringify(process.env.API_PORT || '20100')
  }
})