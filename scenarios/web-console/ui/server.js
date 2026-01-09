import { startScenarioServer } from '@vrooli/api-base/server'

const { UI_PORT, API_PORT } = process.env

if (!UI_PORT) {
  console.error('Web Console UI requires UI_PORT environment variable')
  process.exit(1)
}

if (!API_PORT) {
  console.error('Web Console UI requires API_PORT environment variable')
  process.exit(1)
}

startScenarioServer({
  uiPort: UI_PORT,
  apiPort: API_PORT,
  distDir: './dist',
  serviceName: 'web-console',
  version: '0.2.0',
  corsOrigins: '*',
  verbose: true,

  // Enable WebSocket proxying (/ws/* -> /api/v1/*)
  wsPathPrefix: '/ws',

  // Add required forwarding headers for web-console API
  proxyHeaders: (req) => {
    const existingForwardedFor = req.headers['x-forwarded-for']
    const remoteAddress = req.socket.remoteAddress || '127.0.0.1'
    const forwardedFor = existingForwardedFor
      ? `${existingForwardedFor}, ${remoteAddress}`
      : remoteAddress

    return {
      'x-forwarded-for': forwardedFor,
      'x-forwarded-proto': req.headers['x-forwarded-proto'] || (req.socket.encrypted ? 'https' : 'http'),
      'x-forwarded-host': req.headers['x-forwarded-host'] || req.headers['host'] || ''
    }
  }
})
