import { startScenarioServer } from '@vrooli/api-base/server'

startScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'agent-inbox',
  corsOrigins: '*',
  verbose: process.env.DEBUG === 'true',
  // Extended timeout for AI completions with web search (can take 60+ seconds)
  proxyTimeoutMs: 180000, // 3 minutes
})
