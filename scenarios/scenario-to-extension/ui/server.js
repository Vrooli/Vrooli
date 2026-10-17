import { startScenarioServer } from '@vrooli/api-base/server'

// api-base owns the production UI server, same-origin API proxy, tunnel-safe
// routing, health/config endpoints, and graceful lifecycle behavior. The UI
// must not maintain a second proxy implementation beside that owner.
startScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  apiHost: process.env.API_HOST || '127.0.0.1',
  distDir: './dist',
  serviceName: 'scenario-to-extension-ui',
  version: '2.0.0',
  corsOrigins: '*',
})
