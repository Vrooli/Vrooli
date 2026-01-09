import { createScenarioServer } from '@vrooli/api-base/server'

const app = createScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'scenario-auditor',
  version: '2.0.0',
  corsOrigins: '*',
  verbose: true
})

const PORT = process.env.UI_PORT
app.listen(PORT, () => {
  console.log(`Scenario Auditor UI available at http://localhost:${PORT}`)
})
