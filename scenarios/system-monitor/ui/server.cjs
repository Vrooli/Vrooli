#!/usr/bin/env node

// System Monitor UI server powered by @vrooli/api-base. Serves the production
// bundle and proxies API requests exactly as documented in packages/api-base.
const path = require('node:path')

const UI_PORT = process.env.UI_PORT || process.env.PORT || '3000'
const UI_HOST = (process.env.UI_HOST || '0.0.0.0').trim() || '0.0.0.0'
const API_PORT = process.env.API_PORT || '8080'
const API_HOST = (process.env.API_HOST || '127.0.0.1').trim() || '127.0.0.1'

async function start() {
  const { createScenarioServer, createProxyMiddleware } = await import('@vrooli/api-base/server')

  const app = createScenarioServer({
    uiPort: UI_PORT,
    apiPort: API_PORT,
    apiHost: API_HOST,
    distDir: path.resolve('dist'),
    serviceName: 'system-monitor-ui',
    version: process.env.npm_package_version,
    corsOrigins: '*',
  })

  // Maintain backward compatibility: allow /api/v1/* to reach the same Go API
  // endpoints so clients calling resolveApiBase({ appendSuffix: true }) work.
  app.use('/api/v1', createProxyMiddleware({
    apiPort: API_PORT,
    apiHost: API_HOST,
  }))

  const server = app.listen(Number(UI_PORT), UI_HOST, () => {
    const displayHost = UI_HOST === '0.0.0.0' ? '127.0.0.1' : UI_HOST
    console.log(`System Monitor UI listening on http://${displayHost}:${UI_PORT}`)
    console.log(`Proxying API through http://${displayHost}:${UI_PORT}/api -> ${API_HOST}:${API_PORT}`)
  })

  const shutdown = (signal) => {
    console.log(`\nReceived ${signal}. Shutting down System Monitor UI...`)
    server.close(() => {
      console.log('UI server stopped.')
      process.exit(0)
    })
  }

  process.on('SIGINT', shutdown)
  process.on('SIGTERM', shutdown)
}

start().catch(error => {
  console.error('Failed to start System Monitor UI server:', error)
  process.exit(1)
})
