#!/usr/bin/env node
import { createScenarioServer } from '@vrooli/api-base/server'

const app = createScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'deployment-manager',
  version: '1.0.0',
  corsOrigins: '*'
})

app.listen(process.env.UI_PORT, () => {
  console.log(`✅ Deployment Manager UI serving on http://localhost:${process.env.UI_PORT}`)
})
