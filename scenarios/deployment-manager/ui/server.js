#!/usr/bin/env node
import http from 'node:http'
import https from 'node:https'
import { execSync } from 'node:child_process'
import { createScenarioServer } from '@vrooli/api-base/server'

const UI_PORT = process.env.UI_PORT
const API_PORT = process.env.API_PORT

const resolveAnalyzerApiPort = () => {
  if (process.env.SCENARIO_DEPENDENCY_ANALYZER_API_PORT) {
    return { port: process.env.SCENARIO_DEPENDENCY_ANALYZER_API_PORT, source: 'env' }
  }
  try {
    const output = execSync('vrooli scenario port scenario-dependency-analyzer API_PORT', {
      encoding: 'utf-8',
      stdio: ['ignore', 'pipe', 'ignore'],
    }).trim()
    if (output) return { port: output, source: 'vrooli' }
  } catch {
    // ignore and fallback
  }
  return { port: null, source: 'missing' }
}

const app = createScenarioServer({
  uiPort: UI_PORT,
  apiPort: API_PORT,
  distDir: './dist',
  serviceName: 'deployment-manager',
  version: '1.0.0',
  corsOrigins: '*',
  embeddedProxy: true,
  setupRoutes: (expressApp) => {
    // Referer-based API proxy for embedded analyzer requests.
    // The built-in embedded proxy handles the UI proxy, but API calls from the
    // embedded iframe carry a Referer pointing at /embedded/scenario-dependency-analyzer
    // and need to be forwarded to the analyzer's own API server.
    const { port: analyzerApiPort, source: apiPortSource } = resolveAnalyzerApiPort()
    const analyzerApiHost = process.env.SCENARIO_DEPENDENCY_ANALYZER_API_HOST || '127.0.0.1'
    const analyzerApiScheme = (process.env.SCENARIO_DEPENDENCY_ANALYZER_API_SCHEME || 'http').toLowerCase()

    if (analyzerApiPort) {
      const apiAgent = analyzerApiScheme === 'https' ? https : http
      console.log(`[deployment-manager] Analyzer API proxy → ${analyzerApiScheme}://${analyzerApiHost}:${analyzerApiPort} (source: ${apiPortSource})`)

      expressApp.use('/api', (req, res, next) => {
        const referer = typeof req.headers.referer === 'string' ? req.headers.referer : ''
        if (!referer.includes('/embedded/scenario-dependency-analyzer')) {
          return next()
        }

        const target = `${analyzerApiScheme}://${analyzerApiHost}:${analyzerApiPort}`
        const url = new URL(req.originalUrl, target)

        const proxyReq = apiAgent.request(
          url,
          {
            method: req.method,
            headers: {
              ...req.headers,
              host: `${analyzerApiHost}:${analyzerApiPort}`,
            },
          },
          (proxyRes) => {
            res.writeHead(proxyRes.statusCode || 502, proxyRes.headers)
            proxyRes.pipe(res, { end: true })
          },
        )

        proxyReq.on('error', (error) => {
          console.error('[deployment-manager] Analyzer API proxy error:', error.message)
          if (!res.headersSent) {
            res.status(502).json({ error: 'Failed to proxy analyzer API', detail: error.message, target })
          } else {
            res.end()
          }
        })

        if (req.readable) {
          req.pipe(proxyReq, { end: true })
        } else {
          proxyReq.end()
        }
      })
    } else {
      console.warn('[deployment-manager] Analyzer API port missing; API proxy disabled for embedded analyzer requests')
    }
  },
})

app.listen(UI_PORT, () => {
  console.log(`✅ Deployment Manager UI serving on http://localhost:${UI_PORT}`)
})
