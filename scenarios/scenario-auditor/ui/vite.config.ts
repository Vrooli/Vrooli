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
        handle: async (req: any, res: any, next: any) => {
          if (req.url === '/health') {
            // Test API connectivity
            const apiPort = process.env.API_PORT || '18507'
            let apiConnected = false
            let apiError = null
            let latencyMs = null
            const checkTimestamp = new Date().toISOString()

            try {
              const apiUrl = `http://localhost:${apiPort}/api/v1/health`
              const startTime = Date.now()
              const response = await fetch(apiUrl, {
                signal: AbortSignal.timeout(3000)
              })

              if (response.ok) {
                apiConnected = true
                latencyMs = Date.now() - startTime
              } else {
                apiError = {
                  code: `HTTP_${response.status}`,
                  message: `API returned status ${response.status}`,
                  category: 'network',
                  retryable: true
                }
              }
            } catch (error) {
              const errorMessage = error instanceof Error ? error.message : 'unknown error'
              apiError = {
                code: errorMessage.includes('timeout') ? 'TIMEOUT' : 'CONNECTION_REFUSED',
                message: `Failed to connect to API: ${errorMessage}`,
                category: 'network',
                retryable: true
              }
            }

            const overallStatus = apiConnected ? 'healthy' : 'degraded'
            res.statusCode = apiConnected ? 200 : 503
            res.setHeader('Content-Type', 'application/json')
            res.end(JSON.stringify({
              status: overallStatus,
              service: 'scenario-auditor-ui',
              timestamp: new Date().toISOString(),
              readiness: true,
              api_connectivity: {
                connected: apiConnected,
                api_url: `http://localhost:${apiPort}/api/v1`,
                last_check: checkTimestamp,
                error: apiError,
                latency_ms: latencyMs
              }
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
        target: `http://localhost:${process.env.API_PORT || '18507'}`,
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
})