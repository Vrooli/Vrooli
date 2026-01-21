import http from 'node:http'
import https from 'node:https'
import { execSync } from 'node:child_process'
import { createScenarioServer } from '@vrooli/api-base/server'

const UI_PORT = process.env.UI_PORT
const API_PORT = process.env.API_PORT

/**
 * Resolve the UI port for a scenario.
 * Tries environment variable first, then vrooli CLI, then returns null.
 */
const resolveScenarioPort = (scenarioName, portType = 'UI_PORT') => {
  const envVar = `${scenarioName.replace(/-/g, '_').toUpperCase()}_${portType}`
  if (process.env[envVar]) {
    return { port: process.env[envVar], source: 'env' }
  }
  try {
    const output = execSync(`vrooli scenario port ${scenarioName} ${portType}`, {
      encoding: 'utf-8',
      stdio: ['ignore', 'pipe', 'ignore'],
    }).trim()
    if (output) return { port: output, source: 'vrooli' }
  } catch {
    // Scenario not running or vrooli not available
  }
  return { port: null, source: 'not_found' }
}

const app = createScenarioServer({
  uiPort: UI_PORT,
  apiPort: API_PORT,
  distDir: './dist',
  serviceName: 'agent-inbox',
  corsOrigins: '*',
  verbose: process.env.DEBUG === 'true',
  // Extended timeout for AI completions with web search (can take 60+ seconds)
  proxyTimeoutMs: 180000, // 3 minutes
  setupRoutes: (expressApp) => {
    // Embedded scenario proxy routes
    // Pattern: /embedded/{scenario}/* proxies to the scenario's UI
    // This allows embedding scenario UIs in iframes that work in any context
    // (localhost, tunnel, remote)

    // Target endpoint - returns actual URL for discovery
    expressApp.get('/embedded/:scenario/target', (req, res) => {
      const scenarioName = req.params.scenario
      const { port, source } = resolveScenarioPort(scenarioName, 'UI_PORT')

      if (!port) {
        return res.status(503).json({
          error: 'Scenario not available',
          scenario: scenarioName,
          detail: 'Could not resolve UI port for this scenario'
        })
      }

      const host = process.env[`${scenarioName.replace(/-/g, '_').toUpperCase()}_UI_HOST`] || '127.0.0.1'
      const scheme = (process.env[`${scenarioName.replace(/-/g, '_').toUpperCase()}_SCHEME`] || 'http').toLowerCase()

      res.json({
        url: `${scheme}://${host}:${port}`,
        scenario: scenarioName,
        source
      })
    })

    // Proxy endpoint - forwards all requests to scenario UI
    expressApp.use('/embedded/:scenario', (req, res) => {
      const scenarioName = req.params.scenario
      const { port } = resolveScenarioPort(scenarioName, 'UI_PORT')

      if (!port) {
        return res.status(503).json({
          error: 'Scenario not available',
          scenario: scenarioName,
          detail: 'Could not resolve UI port for this scenario'
        })
      }

      const host = process.env[`${scenarioName.replace(/-/g, '_').toUpperCase()}_UI_HOST`] || '127.0.0.1'
      const scheme = (process.env[`${scenarioName.replace(/-/g, '_').toUpperCase()}_SCHEME`] || 'http').toLowerCase()
      const agent = scheme === 'https' ? https : http

      const target = `${scheme}://${host}:${port}`
      // Remove /embedded/{scenario} prefix to get the actual path
      const rewrittenPath = req.originalUrl.replace(new RegExp(`^/embedded/${scenarioName}`), '') || '/'

      let url
      try {
        url = new URL(rewrittenPath, target)
      } catch (e) {
        return res.status(400).json({
          error: 'Invalid URL',
          detail: e.message
        })
      }

      if (process.env.DEBUG === 'true') {
        console.log(`[agent-inbox] Proxying to ${scenarioName}: ${req.method} ${url.href}`)
      }

      const proxyReq = agent.request(
        url,
        {
          method: req.method,
          headers: {
            ...req.headers,
            host: `${host}:${port}`,
          },
        },
        (proxyRes) => {
          res.writeHead(proxyRes.statusCode || 502, proxyRes.headers)
          proxyRes.pipe(res, { end: true })
        },
      )

      proxyReq.on('error', (error) => {
        console.error(`[agent-inbox] Scenario proxy error (${scenarioName}):`, error.message)
        if (!res.headersSent) {
          res.status(502).json({
            error: 'Failed to proxy to scenario UI',
            scenario: scenarioName,
            detail: error.message,
            target
          })
        } else {
          res.end()
        }
      })

      // Set timeout for proxy requests
      proxyReq.setTimeout(30000, () => {
        proxyReq.destroy()
        if (!res.headersSent) {
          res.status(504).json({
            error: 'Proxy timeout',
            scenario: scenarioName,
            detail: 'Request to scenario UI timed out'
          })
        }
      })

      if (req.readable) {
        req.pipe(proxyReq, { end: true })
      } else {
        proxyReq.end()
      }
    })

    // Also proxy API requests from embedded scenarios
    // This handles cases where the embedded UI makes API calls relative to its path
    expressApp.use('/embedded/:scenario/api', (req, res) => {
      const scenarioName = req.params.scenario
      const { port } = resolveScenarioPort(scenarioName, 'API_PORT')

      if (!port) {
        return res.status(503).json({
          error: 'Scenario API not available',
          scenario: scenarioName,
          detail: 'Could not resolve API port for this scenario'
        })
      }

      const host = process.env[`${scenarioName.replace(/-/g, '_').toUpperCase()}_API_HOST`] || '127.0.0.1'
      const scheme = (process.env[`${scenarioName.replace(/-/g, '_').toUpperCase()}_API_SCHEME`] || 'http').toLowerCase()
      const agent = scheme === 'https' ? https : http

      const target = `${scheme}://${host}:${port}`
      // Rewrite path: /embedded/{scenario}/api/* -> /api/*
      const rewrittenPath = req.originalUrl.replace(new RegExp(`^/embedded/${scenarioName}`), '') || '/'

      let url
      try {
        url = new URL(rewrittenPath, target)
      } catch (e) {
        return res.status(400).json({
          error: 'Invalid URL',
          detail: e.message
        })
      }

      if (process.env.DEBUG === 'true') {
        console.log(`[agent-inbox] Proxying API to ${scenarioName}: ${req.method} ${url.href}`)
      }

      const proxyReq = agent.request(
        url,
        {
          method: req.method,
          headers: {
            ...req.headers,
            host: `${host}:${port}`,
          },
        },
        (proxyRes) => {
          res.writeHead(proxyRes.statusCode || 502, proxyRes.headers)
          proxyRes.pipe(res, { end: true })
        },
      )

      proxyReq.on('error', (error) => {
        console.error(`[agent-inbox] Scenario API proxy error (${scenarioName}):`, error.message)
        if (!res.headersSent) {
          res.status(502).json({
            error: 'Failed to proxy to scenario API',
            scenario: scenarioName,
            detail: error.message,
            target
          })
        } else {
          res.end()
        }
      })

      proxyReq.setTimeout(60000, () => {
        proxyReq.destroy()
        if (!res.headersSent) {
          res.status(504).json({
            error: 'API proxy timeout',
            scenario: scenarioName,
            detail: 'Request to scenario API timed out'
          })
        }
      })

      if (req.readable) {
        req.pipe(proxyReq, { end: true })
      } else {
        proxyReq.end()
      }
    })
  },
})

app.listen(UI_PORT, () => {
  console.log(`✅ Agent Inbox UI serving on http://localhost:${UI_PORT}`)
})
