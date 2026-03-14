import { startScenarioServer } from '@vrooli/api-base/server'

startScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'git-control-tower',
  corsOrigins: '*',
  embeddedProxy: true,
  // Visual capture and workflow capture requests can take minutes when
  // capturing multiple presets/pages via BAS. The default 15s proxy timeout
  // kills the connection before subsequent presets finish.
  proxyTimeoutMs: 10 * 60 * 1000, // 10 minutes
})
