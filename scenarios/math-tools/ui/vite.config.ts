import { defineConfig, type Plugin } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'node:path'
import type { IncomingMessage, ServerResponse } from 'node:http'

function resolvePort(): number {
  const candidates = [process.env.UI_PORT, process.env.PORT, '5173']
  for (const candidate of candidates) {
    if (!candidate) continue
    const numeric = Number(candidate)
    if (Number.isFinite(numeric) && numeric > 0) {
      return numeric
    }
  }
  return 5173
}

type HealthError = {
  code: string
  message: string
  category: 'network' | 'configuration' | 'authentication' | 'resource' | 'internal'
  retryable: boolean
  details?: Record<string, unknown>
}

type HealthPayload = {
  status: 'healthy' | 'degraded' | 'unhealthy'
  service: string
  timestamp: string
  readiness: boolean
  api_connectivity: {
    connected: boolean
    api_url: string
    last_check: string
    latency_ms: number | null
    error: HealthError | null
  }
}

function determineApiBase(): string {
  const explicit = process.env.VITE_API_BASE_URL ?? process.env.API_BASE_URL
  if (explicit && explicit.trim().length > 0) {
    return explicit.trim().replace(/\/+$/u, '')
  }

  const port = process.env.API_PORT ?? process.env.VITE_API_PORT ?? '15000'
  return `http://127.0.0.1:${port}`
}

async function buildHealthPayload(): Promise<HealthPayload> {
  const timestamp = new Date().toISOString()
  const apiBase = determineApiBase()
  const healthUrl = `${apiBase.replace(/\/+$/u, '')}/health`

  let connected = false
  let latencyMs: number | null = null
  let error: HealthError | null = null

  try {
    const started = Date.now()
    const response = await fetch(healthUrl, {
      method: 'GET',
      headers: { Accept: 'application/json' },
    })
    latencyMs = Date.now() - started
    connected = response.ok

    if (!connected) {
      error = {
        code: `HTTP_${response.status}`,
        message: response.statusText || `Unexpected response (${response.status})`,
        category: 'network',
        retryable: response.status >= 500,
        details: { status: response.status, statusText: response.statusText },
      }
    }
  } catch (caught) {
    const message = caught instanceof Error ? caught.message : 'Unable to reach API'
    const details = caught && typeof caught === 'object' && 'name' in (caught as Record<string, unknown>)
      ? { name: String((caught as Record<string, unknown>).name) }
      : undefined

    error = {
      code: 'CONNECTION_ERROR',
      message,
      category: 'network',
      retryable: true,
      ...(details ? { details } : {}),
    }
    connected = false
    latencyMs = null
  }

  const status: HealthPayload['status'] = connected ? 'healthy' : 'degraded'

  return {
    status,
    service: 'Math Tools UI',
    timestamp,
    readiness: connected,
    api_connectivity: {
      connected,
      api_url: apiBase,
      last_check: timestamp,
      latency_ms: connected ? latencyMs : null,
      error: connected ? null : error,
    },
  }
}

function sendJson(req: IncomingMessage, res: ServerResponse, payload: string): void {
  res.statusCode = 200
  res.setHeader('Content-Type', 'application/json')
  if (req.method === 'HEAD') {
    res.end()
    return
  }
  res.end(payload)
}

function sendError(res: ServerResponse, error: Error): void {
  res.statusCode = 500
  res.setHeader('Content-Type', 'application/json')
  const message = error.message || 'Health probe failed'
  res.end(
    JSON.stringify({
      status: 'unhealthy',
      service: 'Math Tools UI',
      timestamp: new Date().toISOString(),
      readiness: false,
      api_connectivity: {
        connected: false,
        api_url: determineApiBase(),
        last_check: new Date().toISOString(),
        latency_ms: null,
        error: {
          code: 'HEALTH_HANDLER_ERROR',
          message,
          category: 'internal',
          retryable: true,
        },
      },
    }),
  )
}

function healthCheckPlugin(): Plugin {
  const HEALTH_PATH = '/health'

  const handleHealthRequest = (req: IncomingMessage, res: ServerResponse) => {
    buildHealthPayload()
      .then(payload => sendJson(req, res, JSON.stringify(payload)))
      .catch(error => {
        sendError(res, error instanceof Error ? error : new Error('Unknown health handler error'))
      })
  }

  return {
    name: 'math-tools-health-check',
    configureServer(server) {
      server.middlewares.use((req, res, next) => {
        if (!req || !req.url) {
          next()
          return
        }

        const requestPath = req.url.split('?')[0]
        if (requestPath === HEALTH_PATH) {
          handleHealthRequest(req, res)
          return
        }

        next()
      })
    },
    configurePreviewServer(server) {
      server.middlewares.use((req, res, next) => {
        if (!req || !req.url) {
          next()
          return
        }

        const requestPath = req.url.split('?')[0]
        if (requestPath === HEALTH_PATH) {
          handleHealthRequest(req, res)
          return
        }

        next()
      })
    },
  }
}

export default defineConfig(() => {
  const port = resolvePort()

  return {
    plugins: [react(), healthCheckPlugin()],
    base: '/',
    resolve: {
      alias: [{ find: '@', replacement: path.resolve(__dirname, 'src') }],
    },
    build: {
      sourcemap: true,
    },
    server: {
      host: true,
      port,
      strictPort: true,
      fs: {
        allow: [path.resolve(__dirname)],
      },
    },
    preview: {
      host: true,
      port,
      strictPort: true,
    },
  }
})
