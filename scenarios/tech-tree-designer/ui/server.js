import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { startScenarioServer } from '@vrooli/api-base/server'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

const SERVICE_NAME = 'tech-tree-designer'
const VERSION = process.env.npm_package_version || '1.0.0'
const DEFAULT_UI_PORT = process.env.UI_PORT || process.env.PORT || '3000'
const API_PORT = process.env.API_PORT

function requireEnv(value, name) {
  if (!value) {
    throw new Error(`[${SERVICE_NAME}] Missing required environment variable: ${name}`)
  }
  return value
}

startScenarioServer({
  uiPort: requireEnv(DEFAULT_UI_PORT, 'UI_PORT'),
  apiPort: requireEnv(API_PORT, 'API_PORT'),
  distDir: path.join(__dirname, 'dist'),
  serviceName: SERVICE_NAME,
  version: VERSION,
  corsOrigins: '*',
})
