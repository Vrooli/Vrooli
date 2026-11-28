import http from 'node:http'

import { proxyToApi, startScenarioServer } from '@vrooli/api-base/server'

// Both ports come from the scenario lifecycle. createScenarioServer already
// rejects either one when it is missing or unparseable, so these guards change
// no behaviour: the process still refuses to start. What they change is the
// message — the library reports "Invalid UI_PORT configuration", which names
// the symptom, and these name the cause and the fix.
const uiPort = process.env.UI_PORT
if (!uiPort) {
  throw new Error(
    'UI_PORT is not set. The scenario lifecycle supplies it — start this scenario with `vrooli scenario start backdrop-studio` rather than running server.js directly.',
  )
}

const apiPort = process.env.API_PORT
if (!apiPort) {
  throw new Error(
    'API_PORT is not set. The scenario lifecycle supplies it — start this scenario with `vrooli scenario start backdrop-studio` rather than running server.js directly.',
  )
}

const connectRpcPath = /^\/vrooli\.backdrop_studio\.v1\./

// A render answers when its candidates exist, and a model-backed candidate on a
// CPU host is tens of minutes of diffusion. The proxy's default 15 s socket
// timeout cut every such request mid-render: the API saw its caller vanish,
// gave up its wait, and the image-tools job ran to completion for nobody. The
// render route gets the API's own operation budget (two hours) plus a minute
// of slack, so the API's deadline — which cancels the job — always fires first.
const renderSubmitPath = '/vrooli.backdrop_studio.v1.render.RenderService/Submit'
const renderSubmitTimeoutMs = (2 * 60 + 1) * 60 * 1000

/**
 * Proxy one render, and close the upstream request when the browser leaves.
 *
 * proxyToApi does not watch the downstream connection, so a closed tab would
 * otherwise hold a render open until it finished. Destroying the request's own
 * agent closes its upstream socket; the API reads that as an abandoned render
 * and cancels the image-tools job it submitted.
 */
function proxyRender(req, res, next) {
  const agent = new http.Agent({ keepAlive: false })
  res.on('close', () => {
    if (!res.writableFinished) {
      agent.destroy()
    }
  })
  proxyToApi(req, res, req.originalUrl || req.url, {
    apiPort,
    agent,
    timeout: renderSubmitTimeoutMs,
  }).catch(next)
}

startScenarioServer({
  uiPort,
  apiPort,
  distDir: './dist',
  serviceName: 'backdrop-studio',
  setupRoutes(app) {
    app.use((req, res, next) => {
      if (!connectRpcPath.test(req.path)) {
        next()
        return
      }
      if (req.path === renderSubmitPath) {
        proxyRender(req, res, next)
        return
      }

      proxyToApi(req, res, req.originalUrl || req.url, {
        apiPort,
      }).catch(next)
    })
  },
})
