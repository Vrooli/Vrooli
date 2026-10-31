import { proxyToApi, startScenarioServer } from '@vrooli/api-base/server'

const connectRpcPath = /^\/vrooli\.data_backup_manager\.v1\./

function requiredEnv(name) {
  const value = process.env[name]
  if (!value) {
    throw new Error(`${name} is required`)
  }
  return value
}

const uiPort = requiredEnv('UI_PORT')
const apiPort = requiredEnv('API_PORT')

startScenarioServer({
  uiPort,
  apiPort,
  distDir: './dist',
  serviceName: 'data-backup-manager',
  corsOrigins: '*',
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
