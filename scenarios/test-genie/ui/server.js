import { startScenarioServer } from '@vrooli/api-base/server'

startScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'test-genie',
  corsOrigins: '*',
  verbose: process.env.NODE_ENV !== 'production',
  // WebSocket configuration for real-time agent updates
  wsPathPrefix: '/ws',
  wsPathTransform: (incomingPath) => incomingPath || '/ws',
  configBuilder: (env) => ({
    apiUrl: `http://127.0.0.1:${env.API_PORT}/api/v1`,
    wsUrl: `ws://127.0.0.1:${env.API_PORT}/ws`,
    apiPort: env.API_PORT,
    wsPort: env.API_PORT,
    uiPort: env.UI_PORT,
    version: '1.0.0',
    service: 'test-genie-ui',
  }),
})
