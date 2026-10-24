import { createScenarioServer } from '@vrooli/api-base/server'

// Both ports come from the scenario lifecycle. createScenarioServer already
// rejects either one when it is missing or unparseable, so these guards change
// no behaviour: the process still refuses to start. What they change is the
// message — the library reports "Invalid UI_PORT configuration", which names
// the symptom, and these name the cause and the fix.
const uiPort = process.env.UI_PORT
if (!uiPort) {
  throw new Error(
    'UI_PORT is not set. The scenario lifecycle supplies it — start this scenario with `vrooli scenario start tidiness-manager` rather than running server.js directly.',
  )
}

const apiPort = process.env.API_PORT
if (!apiPort) {
  throw new Error(
    'API_PORT is not set. The scenario lifecycle supplies it — start this scenario with `vrooli scenario start tidiness-manager` rather than running server.js directly.',
  )
}

const app = createScenarioServer({
  uiPort,
  apiPort,
  distDir: './dist',
  serviceName: 'tidiness-manager',
  corsOrigins: '*',
  // Inject config into HTML to avoid fetch issues with SPA routing
  scenarioConfig: {
    apiUrl: `http://127.0.0.1:${apiPort}/api/v1`,
    wsUrl: `ws://127.0.0.1:${apiPort}/ws`,
    apiPort,
    wsPort: apiPort,
    uiPort,
    service: 'tidiness-manager'
  }
})

app.listen(uiPort)
