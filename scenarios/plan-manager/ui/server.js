import { proxyToApi, startScenarioServer } from '@vrooli/api-base/server'
import helmet from 'helmet'

const connectRpcPath = /^\/vrooli\.plan_manager\.v1\./

const requiredEnv = (name) => {
  const value = process.env[name]
  if (!value) {
    throw new Error(`${name} must be set`)
  }
  return value
}

const uiPort = requiredEnv('UI_PORT')
const apiPort = requiredEnv('API_PORT')

// Standard security headers for the static UI. Helmet owns the CSP and common
// hardening headers for the static SPA + the Connect proxy hop.
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
const frameAncestors = ["'self'", 'http://localhost:*', 'http://127.0.0.1:*', 'http://[::1]:*', ...extraFrameAncestors]

const securityMiddleware = helmet({
  frameguard: false,
  contentSecurityPolicy: {
    useDefaults: true,
    directives: {
      defaultSrc: ["'self'"],
      baseUri: ["'self'"],
      frameAncestors,
      objectSrc: ["'none'"],
      imgSrc: ["'self'", 'data:'],
      styleSrc: ["'self'", "'unsafe-inline'"],
      scriptSrc: ["'self'"],
      connectSrc: ["'self'"],
      fontSrc: ["'self'"],
      formAction: ["'self'"],
    },
  },
  referrerPolicy: { policy: 'strict-origin-when-cross-origin' },
  permissionsPolicy: {
    features: {
      camera: [],
      microphone: [],
      geolocation: [],
    },
  },
})

startScenarioServer({
  uiPort,
  apiPort,
  distDir: './dist',
  serviceName: 'plan-manager',
  corsOrigins: '*',
  setupRoutes(app) {
    app.use(securityMiddleware)

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
