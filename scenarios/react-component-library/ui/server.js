import { pathToFileURL } from 'node:url'
import path from 'node:path'
import { proxyToApi, startScenarioServer } from '@vrooli/api-base/server'

const connectRpcPath = /^\/vrooli\.react_component_library\.v1\./
const previewPath = /^\/preview(?:\/|$)/

export function shouldProxyToApi(path) {
  return connectRpcPath.test(path) || previewPath.test(path)
}

export function isAssetDetailRoute(routePath) {
  const match = /^\/assets\/([^/]+)$/.exec(routePath)
  return Boolean(match && !path.extname(match[1]))
}

export function startReactComponentLibraryServer() {
  return startScenarioServer({
    uiPort: process.env.UI_PORT,
    apiPort: process.env.API_PORT,
    distDir: './dist',
    serviceName: 'react-component-library',
    corsOrigins: '*',
    setupRoutes(app) {
      // api-base correctly treats /assets/* as static asset requests to avoid
      // returning HTML for a missing JS/CSS file. The catalog deliberately
      // uses the same prefix for its SPA detail route, so claim only a
      // extensionless single segment before that generic safeguard runs.
      app.get('/assets/:id', (req, res, next) => {
        if (!isAssetDetailRoute(req.path)) {
          next()
          return
        }
        res.sendFile(path.resolve(process.cwd(), 'dist', 'index.html'))
      })
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
