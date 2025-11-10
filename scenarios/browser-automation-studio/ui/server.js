import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { startScenarioServer } from '@vrooli/api-base/server'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

const SERVICE_NAME = 'browser-automation-studio'
const VERSION = process.env.npm_package_version || '1.0.0'
const API_HOST = process.env.API_HOST || 'localhost'
const WS_HOST = process.env.WS_HOST
const UI_PORT = process.env.UI_PORT || process.env.PORT || '3000'
const API_PORT = process.env.API_PORT

function ensureApiPort(port) {
  if (!port) {
    throw new Error('[server] API_PORT environment variable is required')
  }
  return port
}

export function startServer() {
  return startScenarioServer({
    uiPort: UI_PORT,
    apiPort: ensureApiPort(API_PORT),
    apiHost: API_HOST,
    wsHost: WS_HOST || API_HOST,
    distDir: path.join(__dirname, 'dist'),
    serviceName: SERVICE_NAME,
    version: VERSION,
    corsOrigins: '*',
    wsPathPrefix: '/ws',
    wsPathTransform: (pathValue) => pathValue,
    proxyTimeoutMs: 60000,
  })
}

if (import.meta.url === `file://${process.argv[1]}`) {
  startServer()
}
