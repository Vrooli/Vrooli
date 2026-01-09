const path = require('path')
const { version } = require('./package.json')

const uiPortEnv = process.env.UI_PORT || process.env.PORT || '3000'
const apiPortEnv = process.env.API_PORT
const uiPort = Number(uiPortEnv) || 3000

async function start() {
  const { createScenarioServer } = await import('@vrooli/api-base/server')

  const app = createScenarioServer({
    uiPort: uiPortEnv,
    apiPort: apiPortEnv,
    distDir: path.join(__dirname, 'dist'),
    serviceName: 'scalable-app-cookbook',
    version,
    corsOrigins: '*',
  })

  app.listen(uiPort, () => {
    console.log(`Scalable App Cookbook UI running on http://localhost:${uiPort}`)
    if (apiPortEnv) {
      console.log(`Proxying API requests to port ${apiPortEnv}`)
    }
  })
}

start().catch((error) => {
  console.error('Failed to start Scalable App Cookbook UI server:', error)
  process.exit(1)
})
