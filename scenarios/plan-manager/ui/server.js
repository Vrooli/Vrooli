import { proxyToApi, startScenarioServer } from '@vrooli/api-base/server'

const connectRpcPath = /^\/vrooli\.plan_manager\.v1\./

startScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'plan-manager',
  corsOrigins: '*',
  setupRoutes(app) {
    app.use((req, res, next) => {
      if (!connectRpcPath.test(req.path)) {
        next()
        return
      }

      proxyToApi(req, res, req.originalUrl || req.url, {
        apiPort: process.env.API_PORT,
      }).catch(next)
    })
  },
})
