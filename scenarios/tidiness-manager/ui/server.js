import { createScenarioServer } from '@vrooli/api-base/server'

const app = createScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'tidiness-manager',
  corsOrigins: '*',
  // Inject config into HTML to avoid fetch issues with SPA routing
  scenarioConfig: {
    apiUrl: `http://127.0.0.1:${process.env.API_PORT}/api/v1`,
    wsUrl: `ws://127.0.0.1:${process.env.API_PORT}/ws`,
    apiPort: process.env.API_PORT,
    wsPort: process.env.API_PORT,
    uiPort: process.env.UI_PORT,
    service: 'tidiness-manager'
  }
})

app.listen(process.env.UI_PORT)
