import { defineConfig, Plugin } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// Health check middleware for interoperability with Vrooli lifecycle orchestration
// The /health endpoint is standardized across all scenarios for monitoring and orchestration
function healthCheckPlugin(): Plugin {
  return {
    name: 'health-check',
    configureServer(server) {
      server.middlewares.stack.unshift({
        route: '',
        handle: (req: any, res: any, next: any) => {
          if (req.url === '/health') {
            res.statusCode = 200
            res.setHeader('Content-Type', 'application/json')
            res.end(JSON.stringify({
              status: 'healthy',
              service: 'scenario-auditor-ui',
              timestamp: new Date().toISOString(),
              uptime: process.uptime()
            }))
            return
          }
          next()
        }
      })
    }
  }
}

export default defineConfig({
  plugins: [react(), healthCheckPlugin()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    host: '0.0.0.0',
    port: Number(process.env.UI_PORT) || 36224,
    allowedHosts: ['scenario-auditor.itsagitime.com'],
    proxy: {
      '/api': {
        target: `http://localhost:${process.env.API_PORT || '15001'}`,
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
})