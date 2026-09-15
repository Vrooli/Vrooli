import { startScenarioServer } from '@vrooli/api-base/server'

// Both ports come from the scenario lifecycle. createScenarioServer already
// rejects either one when it is missing or unparseable, so these guards change
// no behaviour: the process still refuses to start. What they change is the
// message — the library reports "Invalid UI_PORT configuration", which names
// the symptom, and these name the cause and the fix.
const uiPort = process.env.UI_PORT
if (!uiPort) {
  throw new Error(
    'UI_PORT is not set. The scenario lifecycle supplies it — start this scenario with `vrooli scenario start test-genie` rather than running server.js directly.',
  )
}

const apiPort = process.env.API_PORT
if (!apiPort) {
  throw new Error(
    'API_PORT is not set. The scenario lifecycle supplies it — start this scenario with `vrooli scenario start test-genie` rather than running server.js directly.',
  )
}

startScenarioServer({
  uiPort,
  apiPort,
  distDir: './dist',
  serviceName: 'test-genie',
  corsOrigins: '*',
  verbose: process.env.NODE_ENV !== 'production',
  // WebSocket configuration for real-time agent updates
  wsPathPrefix: '/ws',
  wsPathTransform: (incomingPath) => incomingPath || '/ws',
  configBuilder: (env) => ({
    apiUrl: `http://127.0.0.1:${env.API_PORT}/api/v1`,
    wsUrl: `ws://127.0.0.1:${env.API_PORT}/ws`,
    apiPort: env.API_PORT,
    wsPort: env.API_PORT,
    uiPort: env.UI_PORT,
    version: '1.0.0',
    service: 'test-genie-ui',
  }),
})
