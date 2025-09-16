import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

const buildHealthPayload = (apiPort: string | undefined) => ({
  status: 'healthy',
  service: 'api-manager-ui',
  timestamp: new Date().toISOString(),
  readiness: true,
  api_connectivity: {
    connected: false,
    api_url: apiPort ? `http://localhost:${apiPort}` : null,
    last_check: new Date().toISOString(),
    error: null as any,
    latency_ms: null as number | null
  }
})

export default defineConfig({
  plugins: [
    react(),
    {
      name: 'health-endpoint',
      configureServer(server) {
        server.middlewares.use('/health', async (req, res, next) => {
          if (req.method !== 'GET') {
            return next()
          }

          const apiPort = process.env.API_PORT
          const healthResponse = buildHealthPayload(apiPort)

          if (!apiPort) {
            healthResponse.status = 'degraded'
            healthResponse.api_connectivity.error = {
              code: 'MISSING_CONFIG',
              message: 'API_PORT environment variable not configured',
              category: 'configuration',
              retryable: false
            }
            res.setHeader('Content-Type', 'application/json')
            res.end(JSON.stringify(healthResponse))
            return
          }

          const startTime = Date.now()

          try {
            // Test API connectivity
            const apiResponse = await fetch(`http://localhost:${apiPort}/health`, {
              method: 'GET',
              headers: { 'Accept': 'application/json' },
              signal: AbortSignal.timeout(5000)
            })

            healthResponse.api_connectivity.latency_ms = Date.now() - startTime

            if (apiResponse.ok) {
              healthResponse.api_connectivity.connected = true
              healthResponse.api_connectivity.error = null
            } else {
              healthResponse.api_connectivity.connected = false
              healthResponse.api_connectivity.error = {
                code: `HTTP_${apiResponse.status}`,
                message: `API returned status ${apiResponse.status}: ${apiResponse.statusText}`,
                category: 'network',
                retryable: apiResponse.status >= 500 && apiResponse.status < 600
              }
              healthResponse.status = 'degraded'
            }
          } catch (error: any) {
            healthResponse.api_connectivity.latency_ms = Date.now() - startTime
            healthResponse.api_connectivity.connected = false

            const isTimeout = error.name === 'AbortError' || error.message?.includes('timeout')
            const isConnRefused = error.message?.includes('ECONNREFUSED') || error.code === 'ECONNREFUSED'
            
            if (isTimeout) {
              healthResponse.api_connectivity.error = {
                code: 'TIMEOUT',
                message: 'API health check timed out after 5 seconds',
                category: 'network',
                retryable: true
              }
              healthResponse.status = 'unhealthy'
            } else if (isConnRefused) {
              healthResponse.api_connectivity.error = {
                code: 'CONNECTION_REFUSED',
                message: 'Failed to connect to API: Connection refused',
                category: 'network',
                retryable: true
              }
              healthResponse.status = 'unhealthy'
            } else {
              healthResponse.api_connectivity.error = {
                code: 'CONNECTION_FAILED',
                message: `Failed to connect to API: ${error.message}`,
                category: 'network',
                retryable: true
              }
              healthResponse.status = 'unhealthy'
            }
          }

          res.setHeader('Content-Type', 'application/json')
          res.end(JSON.stringify(healthResponse))
        })
      }
    }
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: parseInt(process.env.UI_PORT || process.env.PORT || '3000'),
    host: '0.0.0.0',
    proxy: {
      '/api': {
        target: process.env.API_MANAGER_URL || `http://localhost:${process.env.API_PORT || '8100'}`,
        changeOrigin: true,
        secure: false,
      }
    }
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['react', 'react-dom', 'react-router-dom'],
          query: ['@tanstack/react-query'],
          charts: ['recharts'],
        },
      },
    },
  },
})