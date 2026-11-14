import { proxyToApi, startScenarioServer } from '@vrooli/api-base/server'

// Both ports come from the scenario lifecycle. createScenarioServer already
// rejects either one when it is missing or unparseable, so these guards change
// no behaviour: the process still refuses to start. What they change is the
// message — the library reports "Invalid UI_PORT configuration", which names
// the symptom, and these name the cause and the fix.
const uiPort = process.env.UI_PORT
if (!uiPort) {
  throw new Error(
    'UI_PORT is not set. The scenario lifecycle supplies it — start this scenario with `vrooli scenario start scenario-to-android` rather than running server.js directly.',
  )
}

const apiPort = process.env.API_PORT
if (!apiPort) {
  throw new Error(
    'API_PORT is not set. The scenario lifecycle supplies it — start this scenario with `vrooli scenario start scenario-to-android` rather than running server.js directly.',
  )
}

const connectRpcPath = /^\/vrooli\.scenario_to_android\.v1\./

startScenarioServer({
  uiPort,
  apiPort,
  distDir: './dist',
  serviceName: 'scenario-to-android',
  corsOrigins: '*',
  // The API is loopback-bound while this UI server is network reachable.
  // Do not expose retained APK bytes through the general /api proxy.
  proxyHeaders: { 'X-Vrooli-Scenario-UI-Proxy': 'scenario-to-android' },
  setupRoutes(app) {
    app.use((req, res, next) => {
      if (!connectRpcPath.test(req.path)) {
        next()
        return
      }

      proxyToApi(req, res, req.originalUrl || req.url, {
        apiPort,
      }).catch(next)
    })
  },
})
