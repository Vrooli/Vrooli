import { proxyToApi, startScenarioServer } from '@vrooli/api-base/server'

// DBM must never run as root: its API rebuilds shared resource CLIs, and under
// root those builds fail on git's "dubious ownership" check. Refuse the UI too
// so a mistaken privileged start is loud on both components.
if (typeof process.getuid === 'function' && process.getuid() === 0) {
  console.error('data-backup-manager must not run as root: start it as the owning user')
  process.exit(1)
}

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
