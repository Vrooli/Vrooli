import { startScenarioServer } from '@vrooli/api-base/server'

const CONNECT_RPC_PROXY_TIMEOUT_MS = 15 * 60 * 1000

startScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'audio-tools',
  corsOrigins: '*',
  proxyTimeoutMs: CONNECT_RPC_PROXY_TIMEOUT_MS,
  // Dictation uses a same-origin WebSocket rather than Connect RPC. Preserve
  // its endpoint exactly when the lifecycle UI server receives an upgrade.
  wsPathPrefix: '/api/v1/voice/stream',
  wsPathTransform: (path) => path,
})
