import { proxyToApi, startScenarioServer } from '@vrooli/api-base/server'

const connectRpcPath = /^\/vrooli\.extraction_range\.v1\./

// Both ports come from the scenario lifecycle. createScenarioServer already
// rejects either one when it is missing or unparseable, so these guards change
// no behaviour: the process still refuses to start. What they change is the
// message — the library reports "Invalid UI_PORT configuration", which names
// the symptom, and these name the cause and the fix.
const uiPort = process.env.UI_PORT
if (!uiPort) {
  throw new Error(
    'UI_PORT is not set. The scenario lifecycle supplies it — start this scenario with `vrooli scenario start extraction-range` rather than running server.js directly.',
  )
}

const apiPort = process.env.API_PORT
if (!apiPort) {
  throw new Error(
    'API_PORT is not set. The scenario lifecycle supplies it — start this scenario with `vrooli scenario start extraction-range` rather than running server.js directly.',
  )
}

startScenarioServer({
  uiPort,
  apiPort,
  distDir: './dist',
  serviceName: 'extraction-range',
  corsOrigins: '*',
  setupRoutes(app) {
    // The range's REST/practice endpoints intentionally live at the API root
    // rather than under `/api`; keep those paths on the same origin for the
    // browser workbench without routing arbitrary SPA paths to the API.
    app.use((req, res, next) => {
      if (!/^\/(?:levels(?:\/|$)|score(?:$)|scoreboard(?:$)|practice\/)/.test(req.path)) {
        next()
        return
      }
      proxyToApi(req, res, req.originalUrl || req.url, { apiPort }).catch(next)
    })

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
