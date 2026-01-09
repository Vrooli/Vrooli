import { startScenarioServer } from '@vrooli/api-base/server'

startScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'scenario-to-cloud',
  corsOrigins: '*',
  // Long timeout for bundle builds and VPS operations (setup/deploy can take 30+ minutes)
  proxyTimeoutMs: 35 * 60 * 1000, // 35 minutes - matches API server WriteTimeout
  // Enable WebSocket proxying for terminal connections (/api/v1/deployments/{id}/terminal)
  wsPathPrefix: '/api/v1',
  // Enable verbose logging for WebSocket debugging (can disable in production)
  verbose: true,
})
