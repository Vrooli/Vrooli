import { startScenarioServer } from '@vrooli/api-base/server'

startScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'knowledge-observatory',
  corsOrigins: '*',
})
