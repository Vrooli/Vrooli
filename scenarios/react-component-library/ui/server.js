import { createScenarioServer } from '@vrooli/api-base/server'

const app = createScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'react-component-library',
  version: '1.0.0',
  corsOrigins: '*'
})

app.listen(process.env.UI_PORT)
