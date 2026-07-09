import { pathToFileURL } from 'node:url'
import { proxyToApi, startScenarioServer } from '@vrooli/api-base/server'

const connectRpcPath = /^\/vrooli\.react_component_library\.v1\./
const previewPath = /^\/preview(?:\/|$)/

export function shouldProxyToApi(path) {
  return connectRpcPath.test(path) || previewPath.test(path)
}

export function startReactComponentLibraryServer() {
  return startScenarioServer({
    uiPort: process.env.UI_PORT,
    apiPort: process.env.API_PORT,
    distDir: './dist',
    serviceName: 'react-component-library',
    corsOrigins: '*',
    setupRoutes(app) {
      app.use((req, res, next) => {
        if (!shouldProxyToApi(req.path)) {
          next()
          return
        }

        proxyToApi(req, res, req.originalUrl || req.url, {
          apiPort: process.env.API_PORT,
        }).catch(next)
      })
    },
  })
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  startReactComponentLibraryServer()
}
