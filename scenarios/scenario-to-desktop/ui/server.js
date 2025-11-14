import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { startScenarioServer } from '@vrooli/api-base/server'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

const SERVICE_NAME = 'scenario-to-desktop'
const VERSION = process.env.npm_package_version || '1.0.0'
const API_HOST = process.env.API_HOST || '127.0.0.1'
const UI_PORT = process.env.UI_PORT || process.env.PORT
const API_PORT = process.env.API_PORT

function requireEnv(value, name) {
  if (!value) {
    throw new Error(`[scenario-to-desktop/ui] ${name} environment variable is required`)
  }
  return value
}

export function startServer() {
  const resolvedUiPort = requireEnv(UI_PORT, 'UI_PORT')
  const resolvedApiPort = requireEnv(API_PORT, 'API_PORT')

  return startScenarioServer({
    uiPort: resolvedUiPort,
    apiPort: resolvedApiPort,
    apiHost: API_HOST,
    distDir: path.join(__dirname, 'dist'),
    serviceName: SERVICE_NAME,
    version: VERSION,
    corsOrigins: '*',
  })
}

if (import.meta.url === `file://${process.argv[1]}`) {
  startServer()
}
