import { startScenarioServer } from '@vrooli/api-base/server'

function requiredPort(name) {
  const raw = process.env[name]
  const value = Number.parseInt(raw ?? '', 10)
  if (!Number.isInteger(value) || value < 1 || value > 65535) {
    throw new Error(`${name} must be a valid TCP port`)
  }
  return value
}

startScenarioServer({
  uiPort: requiredPort('UI_PORT'),
  apiPort: requiredPort('API_PORT'),
  distDir: './dist',
  serviceName: 'vrooli-onboarding',
  corsOrigins: '*',
})
