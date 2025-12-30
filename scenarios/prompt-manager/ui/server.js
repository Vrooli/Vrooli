import { createScenarioServer } from '@vrooli/api-base/server'

const app = createScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'prompt-manager',
  version: '1.0.0',
  corsOrigins: '*'
})

const PORT = process.env.UI_PORT
if (!PORT) {
  console.error('UI_PORT environment variable is required')
  process.exit(1)
}

const server = app.listen(PORT, () => {
  console.log(`Prompt Manager UI server running on http://127.0.0.1:${PORT}`)
  console.log(`API proxy target: http://127.0.0.1:${process.env.API_PORT}`)
})

// Graceful shutdown
process.on('SIGTERM', () => {
  server.close(() => {
    console.log('UI server stopped')
  })
})

process.on('SIGINT', () => {
  server.close(() => {
    console.log('UI server stopped')
    process.exit(0)
  })
})
