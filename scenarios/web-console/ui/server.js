import { startScenarioServer } from '@vrooli/api-base/server'

// Both ports come from the scenario lifecycle. createScenarioServer already
// rejects either one when it is missing or unparseable, so these guards change
// no behaviour: the process still refuses to start. What they change is the
// message — the library reports "Invalid UI_PORT configuration", which names
// the symptom, and these name the cause and the fix.
const uiPort = process.env.UI_PORT
if (!uiPort) {
  throw new Error(
    'UI_PORT is not set. The scenario lifecycle supplies it — start this scenario with `vrooli scenario start web-console` rather than running server.js directly.',
  )
}

const apiPort = process.env.API_PORT
if (!apiPort) {
  throw new Error(
    'API_PORT is not set. The scenario lifecycle supplies it — start this scenario with `vrooli scenario start web-console` rather than running server.js directly.',
  )
}

startScenarioServer({
  uiPort,
  apiPort,
  distDir: './dist',
  serviceName: 'web-console',
  // Session file previews render in opaque-origin sandbox frames. Their
  // module scripts, styles and media are capability URLs (the preview id is
  // the authorization), so any origin may read them without credentials.
  publicPaths: [/^\/api\/v1\/sessions\/[^/]+\/file-previews\/[^/]+\/runtime\//],
  wsPathPrefix: '/api/v1',
  wsPathTransform: (path) => path,
  // Terminal localhost links use this same-origin path so secure tunnel
  // browsers can reach host services without forwarding arbitrary ports.
  // API Base pins each request to 127.0.0.1 and refuses these control ports.
  localPortProxy: { pathPrefix: '/_local', excludedPorts: [uiPort, apiPort] },
})
