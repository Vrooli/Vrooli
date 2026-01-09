const path = require('path')

async function start() {
  const uiPort = Number(process.env.UI_PORT || process.env.PORT || 3000)
  const apiPort = process.env.API_PORT

  if (!apiPort) {
    console.error('❌ API_PORT environment variable is required to start math-tools UI server')
    process.exit(1)
  }

  const { createScenarioServer } = await import('@vrooli/api-base/server')

  const app = createScenarioServer({
    uiPort,
    apiPort,
    distDir: path.join(__dirname, 'dist'),
    serviceName: 'math-tools',
    version: process.env.npm_package_version || '1.0.0',
    corsOrigins: '*'
  })

  app.listen(uiPort, () => {
    console.log(`Math Tools UI running on http://localhost:${uiPort}`)
    console.log(`Proxying API requests to http://localhost:${apiPort}`)
  })
}

start().catch((error) => {
  console.error('❌ Failed to start math-tools UI server:', error)
  process.exit(1)
})
