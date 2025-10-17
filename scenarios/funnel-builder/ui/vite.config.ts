import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [
    react(),
    {
      name: 'health-check',
      configureServer(server) {
        server.middlewares.use((req, res, next) => {
          if (req.url === '/health') {
            res.writeHead(200, { 'Content-Type': 'text/plain' })
            res.end('OK')
            return
          }
          next()
        })
      }
    }
  ],
  server: {
    host: '0.0.0.0',
    port: parseInt(process.env.FUNNEL_UI_PORT || '20000'),
    proxy: {
      '/api': {
        target: `http://localhost:${process.env.FUNNEL_API_PORT || '15000'}`,
        changeOrigin: true
      }
    }
  },
  preview: {
    port: parseInt(process.env.FUNNEL_UI_PORT || '20000'),
    host: '0.0.0.0'
  }
})