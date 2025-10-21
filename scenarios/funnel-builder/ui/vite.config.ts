import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [
    react(),
    {
      name: 'health-check',
      configureServer(server) {
        server.middlewares.use(async (req, res, next) => {
          if (req.url === '/health') {
            const apiPort = process.env.API_PORT || process.env.FUNNEL_API_PORT || '15000'
            const apiUrl = `http://localhost:${apiPort}/health`

            // Check API connectivity
            let apiConnected = false
            let apiLatency = null
            let apiError = null
            const checkStart = Date.now()

            try {
              const response = await fetch(apiUrl, { signal: AbortSignal.timeout(5000) })
              if (response.ok) {
                apiConnected = true
                apiLatency = Date.now() - checkStart
              } else {
                throw new Error(`API returned ${response.status}`)
              }
            } catch (error: any) {
              apiError = {
                code: error.name === 'TimeoutError' ? 'TIMEOUT' : 'CONNECTION_FAILED',
                message: error.message,
                category: 'network',
                retryable: true
              }
            }

            const healthResponse = {
              status: apiConnected ? 'healthy' : 'degraded',
              service: 'funnel-builder-ui',
              timestamp: new Date().toISOString(),
              readiness: true,
              api_connectivity: {
                connected: apiConnected,
                api_url: apiUrl,
                last_check: new Date().toISOString(),
                latency_ms: apiLatency,
                error: apiError
              }
            }

            res.writeHead(200, { 'Content-Type': 'application/json' })
            res.end(JSON.stringify(healthResponse))
            return
          }
          next()
        })
      }
    }
  ],
  server: {
    host: process.env.VITE_HOST || process.env.UI_HOST || '0.0.0.0',
    port: parseInt(process.env.UI_PORT || process.env.FUNNEL_UI_PORT || '20000'),
    proxy: {
      '/api': {
        target: `http://localhost:${process.env.API_PORT || process.env.FUNNEL_API_PORT || '15000'}`,
        changeOrigin: true
      }
    }
  },
  preview: {
    port: parseInt(process.env.UI_PORT || process.env.FUNNEL_UI_PORT || '20000'),
    host: process.env.VITE_HOST || process.env.UI_HOST || '0.0.0.0'
  }
})