import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { createScenarioServer } from '@vrooli/api-base/server'

const CONNECT_RPC_PROXY_TIMEOUT_MS = 15 * 60 * 1000
const UI_DIR = path.dirname(fileURLToPath(import.meta.url))
const DIST_RELATIVE_DIR = process.env.AUDIO_TOOLS_UI_DIST_DIR?.trim() || 'dist'
const DIST_DIR = path.resolve(UI_DIR, DIST_RELATIVE_DIR)
const DIST_RELATIVE_TO_PACKAGE = path.relative(UI_DIR, DIST_DIR)
if (
  DIST_RELATIVE_TO_PACKAGE === '..' ||
  DIST_RELATIVE_TO_PACKAGE.startsWith(`..${path.sep}`) ||
  path.isAbsolute(DIST_RELATIVE_TO_PACKAGE)
) {
  throw new Error('AUDIO_TOOLS_UI_DIST_DIR must stay within the Audio Tools UI package')
}

function requirePort(name) {
  const raw = process.env[name]
  if (!raw) throw new Error(`${name} is required`)
  const port = Number(raw)
  if (!Number.isInteger(port) || port < 1 || port > 65_535) {
    throw new Error(`${name} must be a valid TCP port`)
  }
  return port
}

const UI_PORT = requirePort('UI_PORT')
const API_PORT = requirePort('API_PORT')

const app = createScenarioServer({
  uiPort: UI_PORT,
  apiPort: API_PORT,
  distDir: DIST_DIR,
  serviceName: 'audio-tools',
  proxyTimeoutMs: CONNECT_RPC_PROXY_TIMEOUT_MS,
  wsPathPrefix: '/ws',
})

// The shared server serves static assets and API proxies but does not install
// a history fallback. React Router routes must still resolve on a direct
// browser/BAS navigation rather than returning its static-server 404.
app.get('*', (req, res, next) => {
  // `req.accepts('html')` treats the generic `Accept: */*` used for module
  // imports as HTML-compatible. That caused lazy route chunks to receive the
  // SPA document instead of JavaScript, leaving the app permanently on its
  // Suspense fallback. Only document navigations explicitly ask for HTML.
  if (!req.headers.accept?.includes('text/html')) return next()
  res.sendFile(path.join(DIST_DIR, 'index.html'))
})

app.listen(UI_PORT, () => {
  console.log(`audio-tools UI listening on ${UI_PORT}`)
})
