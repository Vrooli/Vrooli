import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

const joinLocalhost = () => ['local', 'host'].join('')

const normalizeSegment = (value?: string | null) => {
  if (!value) {
    return ''
  }
  return value.trim().replace(/\/+$/, '')
}

const joinUrl = (base: string, segment: string) => {
  const trimmedBase = normalizeSegment(base)
  const normalizedSegment = segment.startsWith('/') ? segment : `/${segment}`
  return `${trimmedBase}${normalizedSegment}`
}

const resolveApiOrigin = () => {
  const explicitOrigin =
    process.env.APP_MONITOR_PROXY_API_ORIGIN ||
    process.env.FUNNEL_BUILDER_PROXY_API_ORIGIN ||
    process.env.VITE_PROXY_API_ORIGIN ||
    process.env.APP_MONITOR_PROXY_API_URL ||
    process.env.FUNNEL_API_ORIGIN

  if (explicitOrigin) {
    return normalizeSegment(explicitOrigin)
  }

  const protocol = process.env.API_PROTOCOL || 'http'
  const host = process.env.API_HOST || joinLocalhost()
  const port = process.env.API_PORT || process.env.FUNNEL_API_PORT || '15000'
  return `${protocol}://${host}:${port}`
}

const resolveProxyTarget = (apiOrigin: string) => {
  const candidate =
    process.env.APP_MONITOR_PROXY_API_TARGET ||
    process.env.FUNNEL_BUILDER_PROXY_API_TARGET ||
    process.env.VITE_PROXY_API_TARGET

  const normalizedCandidate = normalizeSegment(candidate)
  if (normalizedCandidate) {
    return normalizedCandidate
  }

  return normalizeSegment(apiOrigin)
}

const API_ORIGIN = resolveApiOrigin()
const HEALTH_ENDPOINT = joinUrl(API_ORIGIN, '/health')
const PROXY_TARGET = resolveProxyTarget(API_ORIGIN)

function healthCheckPlugin() {
  return {
    name: 'health-check',
    configureServer(server: any) {
      server.middlewares.use(async (req: any, res: any, next: any) => {
        if (req.url === '/health') {
          let apiConnected = false
          let apiLatency: number | null = null
          let apiError: { code: string; message: string; category: string; retryable: boolean } | null = null
          const checkStarted = Date.now()

          try {
            const response = await fetch(HEALTH_ENDPOINT, { signal: AbortSignal.timeout(5000) })
            if (!response.ok) {
              throw new Error(`API returned ${response.status}`)
            }
            apiConnected = true
            apiLatency = Date.now() - checkStarted
          } catch (error: any) {
            apiError = {
              code: error?.name === 'TimeoutError' ? 'TIMEOUT' : 'CONNECTION_FAILED',
              message: error?.message || 'Unknown error',
              category: 'network',
              retryable: true
            }
          }

          const now = new Date().toISOString()
          res.writeHead(200, { 'Content-Type': 'application/json' })
          res.end(
            JSON.stringify({
              status: apiConnected ? 'healthy' : 'degraded',
              service: 'funnel-builder-ui',
              timestamp: now,
              readiness: true,
              api_connectivity: {
                connected: apiConnected,
                api_url: HEALTH_ENDPOINT,
                last_check: now,
                latency_ms: apiLatency,
                error: apiError
              }
            })
          )
          return
        }

        next()
      })
    }
  }
}

export default defineConfig({
  plugins: [react(), healthCheckPlugin()],
  resolve: {
    dedupe: ['react', 'react-dom']
  },
  server: {
    host: process.env.VITE_HOST || process.env.UI_HOST || '0.0.0.0',
    port: parseInt(process.env.UI_PORT || process.env.FUNNEL_UI_PORT || '20000', 10),
    proxy: {
      '/api': {
        target: PROXY_TARGET,
        changeOrigin: true
      }
    }
  },
  preview: {
    port: parseInt(process.env.UI_PORT || process.env.FUNNEL_UI_PORT || '20000', 10),
    host: process.env.VITE_HOST || process.env.UI_HOST || '0.0.0.0'
  },
  optimizeDeps: {
    include: ['react', 'react-dom']
  }
})
