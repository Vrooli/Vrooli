import { startScenarioServer } from '@vrooli/api-base/server'

startScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'prd-control-tower',
  corsOrigins: '*',
  // Diagnostics scans can run scenario-auditor for a minute+, so extend proxy timeout.
  proxyTimeoutMs: 180000,
})
