import { createScenarioServer, proxyToApi } from '@vrooli/api-base/server'
import { createApplicationProxy } from './server/applicationProxy.js'
import { createHomeListener } from './server/homeListener.js'

const uiPort = process.env.UI_PORT
if (!uiPort) throw new Error('UI_PORT is not set. Start Portal through the Vrooli scenario lifecycle.')
const apiPort = process.env.API_PORT
if (!apiPort) throw new Error('API_PORT is not set. Start Portal through the Vrooli scenario lifecycle.')

const applicationProxy = createApplicationProxy({ apiPort })
const connectRpcPath = /^\/vrooli\.portal\.v1\./
const app = createScenarioServer({
  uiPort,
  apiPort,
  distDir: './dist',
  serviceName: 'portal',
  corsOrigins: '*',
  setupRoutes(expressApp) {
    expressApp.use(applicationProxy.router)
    expressApp.use((req, res, next) => {
      if (!connectRpcPath.test(req.path)) return next()
      // Lifecycle operations can legitimately take longer than ordinary reads.
      // A transport timeout leaves their outcome unknown; callers must refresh.
      const timeout = req.path === '/vrooli.portal.v1.surfaces.ApplicationLifecycleService/Act' ? 190000 : undefined
      proxyToApi(req, res, req.originalUrl || req.url, { apiPort, timeout }).catch(next)
    })
  },
})
const listener = createHomeListener(app)
const server = listener.listen(Number(uiPort), '0.0.0.0', () => {
  console.log(`Portal UI listening on ${listener.secure ? 'HTTPS' : 'HTTP'} port ${uiPort}`)
})
server.on('upgrade', (req, socket, head) => {
  applicationProxy.handleUpgrade(req, socket, head).then(handled => {
    if (!handled) socket.destroy()
  }).catch(() => socket.destroy())
})
const shutdown = () => {
  applicationProxy.destroy()
  server.close(() => process.exit(0))
}
process.once('SIGTERM', shutdown)
process.once('SIGINT', shutdown)
