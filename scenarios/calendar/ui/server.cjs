const path = require('path')

async function start() {
  const uiPort = Number(process.env.UI_PORT || process.env.PORT || 3000)
  const apiPort = process.env.API_PORT

  if (!apiPort) {
    console.error('❌ API_PORT environment variable is required to start calendar UI server')
    process.exit(1)
  }

  const { createScenarioServer } = await import('@vrooli/api-base/server')

  const app = createScenarioServer({
    uiPort,
    apiPort,
    distDir: path.join(__dirname, 'dist'),
    serviceName: 'calendar',
    version: process.env.npm_package_version || '1.0.0',
    corsOrigins: '*'
  })

  app.listen(uiPort, () => {
    console.log(`Calendar UI running on http://localhost:${uiPort}`)
    console.log(`Proxying API requests to http://localhost:${apiPort}`)
  })
}

start().catch((error) => {
  console.error('❌ Failed to start calendar UI server:', error)
  process.exit(1)
})
