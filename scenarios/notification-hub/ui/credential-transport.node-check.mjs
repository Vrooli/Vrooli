import { test } from 'node:test'
import { strict as assert } from 'node:assert'
import { allowCredentialTransport } from './credential-transport.js'
const path = '/vrooli.notification_hub.v1.conversations.ConversationsService/AnswerRootWithPassword'
const enrollment = '/api/v1/identity/root-enrollment'
const rootRead = '/api/v1/identity/root-decision-read'
test('credential ingress rejects remote cleartext and permits TLS or local sockets', () => {
  assert.equal(allowCredentialTransport(path, { remoteAddress: '192.0.2.1' }), false)
  assert.equal(allowCredentialTransport(path, { remoteAddress: '127.0.0.1' }), true)
  assert.equal(allowCredentialTransport(path, { remoteAddress: '::ffff:127.0.0.1' }), true)
  assert.equal(allowCredentialTransport(path, { remoteAddress: '192.0.2.1', encrypted: true }), true)
  assert.equal(allowCredentialTransport(path.replace('Answer', '%41nswer'), { remoteAddress: '127.0.0.1' }), false)
  assert.equal(allowCredentialTransport(rootRead, { remoteAddress: '192.0.2.1' }), false)
  assert.equal(allowCredentialTransport(enrollment, { remoteAddress: '192.0.2.1' }), false)
  assert.equal(allowCredentialTransport(enrollment, { remoteAddress: '127.0.0.1' }), true)
  assert.equal(allowCredentialTransport(enrollment.replace('root', '%72oot'), { remoteAddress: '127.0.0.1' }), false)
})

test('actual HTTP ingress ignores forwarded TLS claims', async (t) => {
  const { createServer, request } = await import('node:http')
  const { networkInterfaces } = await import('node:os')
  const address = Object.values(networkInterfaces()).flat().find((item) => item?.family === 'IPv4' && !item.internal)?.address
  if (!address) return t.skip('no non-loopback interface')
  let forwarded = 0
  const server = createServer((req, res) => {
    if (!allowCredentialTransport(req.url, req.socket)) { res.writeHead(403).end(); return }
    forwarded++; res.writeHead(204).end()
  })
  await new Promise((resolve) => server.listen(0, '0.0.0.0', resolve))
  t.after(() => server.close())
  const status = await new Promise((resolve, reject) => {
    const req = request({ host: address, port: server.address().port, path, method: 'POST', headers: { 'X-Forwarded-Proto': 'https', Origin: 'https://example.test' } }, (res) => { res.resume(); res.on('end', () => resolve(res.statusCode)) })
    req.on('error', reject); req.end()
  })
  assert.equal(status, 403)
  assert.equal(forwarded, 0)
})

test('Express proxy blocks encoded credential paths from a remote socket', async (t) => {
  const { default: express } = await import('express')
  const { proxyToApi } = await import('@vrooli/api-base/server')
  const { createServer, request } = await import('node:http')
  const { networkInterfaces } = await import('node:os')
  const address = Object.values(networkInterfaces()).flat().find((item) => item?.family === 'IPv4' && !item.internal)?.address
  if (!address) return t.skip('no non-loopback interface')
  let reached = 0
  const upstream = createServer((req, res) => { reached++; res.writeHead(204).end() })
  await new Promise((resolve) => upstream.listen(0, '127.0.0.1', resolve))
  t.after(() => upstream.close())
  const app = express()
  app.use((req, res, next) => {
    if (!allowCredentialTransport(req.path, req.socket)) return res.status(403).end()
    if (!/^\/vrooli\.notification_hub\.v1\./.test(req.path)) return next()
    proxyToApi(req, res, req.originalUrl || req.url, { apiPort: upstream.address().port }).catch(next)
  })
  const server = app.listen(0, '0.0.0.0')
  await new Promise((resolve) => server.once('listening', resolve))
  t.after(() => server.close())
  for (const url of [path, path.replace('Answer', '%41nswer'), path.replace('Answer', '%2541nswer')]) {
    const status = await new Promise((resolve, reject) => {
      const req = request({ host: address, port: server.address().port, path: url, method: 'POST', headers: { 'X-Forwarded-Proto': 'https', Origin: 'https://example.test' } }, (res) => { res.resume(); res.on('end', () => resolve(res.statusCode)) })
      req.on('error', reject); req.end()
    })
    assert.equal(status, 403, url)
  }
  assert.equal(reached, 0)
})

test('same-origin Connect proxy preserves the Cloudflare Access assertion for the API', async (t) => {
  const { default: express } = await import('express')
  const { proxyToApi } = await import('@vrooli/api-base/server')
  const { createServer, request } = await import('node:http')
  const assertion = 'signed-cloudflare-access-assertion'
  const path = '/vrooli.notification_hub.v1.identity.IdentityService/GetSession'
  const received = []
  const upstream = createServer((req, res) => {
    received.push(req.headers['cf-access-jwt-assertion'] ?? '')
    res.writeHead(received.at(-1) ? 200 : 401, { 'Content-Type': 'application/json' }).end(JSON.stringify({ signedIn: Boolean(received.at(-1)) }))
  })
  await new Promise((resolve) => upstream.listen(0, '127.0.0.1', resolve))
  t.after(() => upstream.close())

  const app = express()
  app.use((req, res, next) => {
    if (!/^\/vrooli\.notification_hub\.v1\./.test(req.path)) return next()
    proxyToApi(req, res, req.originalUrl || req.url, { apiPort: upstream.address().port }).catch(next)
  })
  const server = app.listen(0, '127.0.0.1')
  await new Promise((resolve) => server.once('listening', resolve))
  t.after(() => server.close())

  const call = (headers = {}) => new Promise((resolve, reject) => {
    const req = request({ host: '127.0.0.1', port: server.address().port, path, method: 'POST', headers: { 'Content-Type': 'application/json', ...headers } }, (res) => {
      let body = ''
      res.setEncoding('utf8')
      res.on('data', (chunk) => { body += chunk })
      res.on('end', () => resolve({ status: res.statusCode, body: JSON.parse(body) }))
    })
    req.on('error', reject)
    req.end('{}')
  })

  assert.deepEqual(await call({ 'Cf-Access-Jwt-Assertion': assertion }), { status: 200, body: { signedIn: true } })
  assert.deepEqual(await call(), { status: 401, body: { signedIn: false } })
  assert.deepEqual(received, [assertion, ''])
})

test('createScenarioServer gates credentials before its generic Connect proxy', async (t) => {
  const { createScenarioServer } = await import('@vrooli/api-base/server')
  const { createServer, request } = await import('node:http')
  const { networkInterfaces } = await import('node:os')
  const address = Object.values(networkInterfaces()).flat().find((item) => item?.family === 'IPv4' && !item.internal)?.address
  if (!address) return t.skip('no non-loopback interface')
  let reached = 0
  const upstream = createServer((req, res) => { reached++; res.writeHead(204).end() })
  await new Promise((resolve) => upstream.listen(0, '127.0.0.1', resolve))
  t.after(() => upstream.close())
  const app = createScenarioServer({ uiPort: '31337', apiPort: String(upstream.address().port), disableLifecycleGuard: true, setupBeforeProxy(app) { app.use((req, res, next) => allowCredentialTransport(req.path, req.socket) ? next() : res.status(403).end()) } })
  const server = app.listen(0, '0.0.0.0')
  await new Promise((resolve) => server.once('listening', resolve))
  t.after(() => server.close())
  const call = (host, url) => new Promise((resolve, reject) => {
    const req = request({ host, port: server.address().port, path: url, method: 'POST', headers: { 'X-Forwarded-Proto': 'https' } }, (res) => { res.resume(); res.on('end', () => resolve(res.statusCode)) })
    req.on('error', reject); req.end()
  })
  for (const url of [path, path.replace('Answer', '%41nswer'), path.replace('Answer', '%2541nswer'), enrollment, enrollment.replace('root', '%72oot'), enrollment.replace('root', '%2572oot'), rootRead, rootRead.replace('root', '%72oot'), rootRead.replace('root', '%2572oot')]) assert.equal(await call(address, url), 403, url)
  assert.equal(reached, 0)
  assert.equal(await call('127.0.0.1', path), 204)
  assert.equal(reached, 1)
})
