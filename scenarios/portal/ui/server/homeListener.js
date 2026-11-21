import { readFileSync } from 'node:fs'
import { createServer } from 'node:https'

// One UI gateway owns pages, catalogue RPCs and child application sockets.
// A partial TLS configuration fails closed; no certificate bypass or downgrade.
export function createHomeListener(app, {
  env = process.env, readFile = readFileSync, httpsServer = createServer,
} = {}) {
  const cert = env.PORTAL_TLS_CERT_FILE
  const key = env.PORTAL_TLS_KEY_FILE
  if (Boolean(cert) !== Boolean(key)) throw new Error('Portal TLS requires both certificate and key files')
  if (!cert) return { secure: false, listen: (port, host, ready) => app.listen(port, host, ready) }
  const server = httpsServer({ cert: readFile(cert), key: readFile(key), minVersion: 'TLSv1.2' }, app)
  return { secure: true, listen: (port, host, ready) => server.listen(port, host, ready) }
}
