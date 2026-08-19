import { startScenarioServer } from '@vrooli/api-base/server'

startScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'swarm-manager',
  corsOrigins: '*',
  embeddedProxy: true,
  wsPathPrefix: '/ws',
  wsPathTransform: (incomingPath) => {
    const path = incomingPath || '/ws'
    return path.startsWith('/ws/voice/stream')
      ? path.replace(/^\/ws/, '/api/v1')
      : path
  },
})
