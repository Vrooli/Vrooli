import { startScenarioServer } from '@vrooli/api-base/server'

startScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'scenario-to-cloud',
  corsOrigins: '*',
  // Long timeout for bundle builds and VPS operations (setup/deploy can take 30+ minutes)
  proxyTimeoutMs: 35 * 60 * 1000, // 35 minutes - matches API server WriteTimeout
})
