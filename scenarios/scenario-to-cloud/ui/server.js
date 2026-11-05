import { startScenarioServer } from '@vrooli/api-base/server'

// Both ports come from the scenario lifecycle. createScenarioServer already
// rejects either one when it is missing or unparseable, so these guards change
// no behaviour: the process still refuses to start. What they change is the
// message — the library reports "Invalid UI_PORT configuration", which names
// the symptom, and these name the cause and the fix.
const uiPort = process.env.UI_PORT
if (!uiPort) {
  throw new Error(
    'UI_PORT is not set. The scenario lifecycle supplies it — start this scenario with `vrooli scenario start scenario-to-cloud` rather than running server.js directly.',
  )
}

const apiPort = process.env.API_PORT
if (!apiPort) {
  throw new Error(
    'API_PORT is not set. The scenario lifecycle supplies it — start this scenario with `vrooli scenario start scenario-to-cloud` rather than running server.js directly.',
  )
}

startScenarioServer({
  uiPort,
  apiPort,
  distDir: './dist',
  serviceName: 'scenario-to-cloud',
  corsOrigins: '*',
  // Long timeout for bundle builds and VPS operations (setup/deploy can take 30+ minutes)
  proxyTimeoutMs: 35 * 60 * 1000, // 35 minutes - matches API server WriteTimeout
  // Enable WebSocket proxying for terminal connections (/api/v1/deployments/{id}/terminal)
  wsPathPrefix: '/api/v1',
  // Enable verbose logging for WebSocket debugging (can disable in production)
  verbose: true,
})
