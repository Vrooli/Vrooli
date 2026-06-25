import { proxyToApi, startScenarioServer } from '@vrooli/api-base/server'

const connectRpcPath = /^\/vrooli\.plan_manager\.v1\./

// Standard security headers for the static UI. The shared api-base server only
// disables X-Powered-By, so we apply the rest here on every response (the static
// SPA + the Connect proxy hop).
//
// IMPORTANT: this scenario renders INSIDE the Vrooli host shell via an iframe, so
// we must NOT set X-Frame-Options: DENY or frame-ancestors 'none' — that would
// blank the embedded view. Instead, per the `vrooli-ui-security-v1` standard
// (ui-health interop_helmet_frame_ancestors), frame-ancestors pins 'self' plus
// the loopback origins the host uses, with optional env-driven extras via
// FRAME_ANCESTORS. connect-src is same-origin because Connect-RPC is proxied
// through this server.
const extraFrameAncestors = (process.env.FRAME_ANCESTORS || '')
  .split(/[,\s]+/)
  .filter(Boolean)
const frameAncestors = [
  "'self'",
  'http://localhost:*',
  'http://127.0.0.1:*',
  'http://[::1]:*',
  ...extraFrameAncestors,
]

const SECURITY_HEADERS = {
  'Content-Security-Policy': [
    "default-src 'self'",
    "base-uri 'self'",
    `frame-ancestors ${frameAncestors.join(' ')}`,
    "object-src 'none'",
    "img-src 'self' data:",
    "style-src 'self' 'unsafe-inline'",
    "script-src 'self'",
    "connect-src 'self'",
    "font-src 'self'",
    "form-action 'self'",
  ].join('; '),
  'X-Content-Type-Options': 'nosniff',
  'Referrer-Policy': 'strict-origin-when-cross-origin',
  'Permissions-Policy': 'camera=(), microphone=(), geolocation=()',
}

startScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'plan-manager',
  corsOrigins: '*',
  setupRoutes(app) {
    app.use((_req, res, next) => {
      for (const [name, value] of Object.entries(SECURITY_HEADERS)) {
        res.setHeader(name, value)
      }
      next()
    })

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
