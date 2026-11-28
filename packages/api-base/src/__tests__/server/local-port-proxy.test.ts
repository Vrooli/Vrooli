import { afterEach, describe, expect, it } from 'vitest'
import express from 'express'
import * as http from 'node:http'
import { networkInterfaces } from 'node:os'
import { gzipSync } from 'node:zlib'
import WebSocket from 'ws'
import { createLocalPortProxy, type LocalPortProxyController } from '../../server/local-port-proxy.js'
import { createScenarioServer } from '../../server/template.js'

type RunningServer = { server: http.Server; port: number }

async function listen(server: http.Server): Promise<number> {
  await new Promise<void>((resolve, reject) => {
    server.once('error', reject)
    server.listen(0, '127.0.0.1', resolve)
  })
  return (server.address() as { port: number }).port
}

async function listenOnAnyAddress(server: http.Server): Promise<number> {
  await new Promise<void>((resolve, reject) => {
    server.once('error', reject)
    server.listen(0, '0.0.0.0', resolve)
  })
  return (server.address() as { port: number }).port
}

async function close(server: http.Server): Promise<void> {
  await new Promise<void>((resolve) => server.close(() => resolve()))
}

async function requestStatus(port: number, path: string, method: string, headers: http.OutgoingHttpHeaders = {}, hostname = '127.0.0.1'): Promise<{ status: number; headers: http.IncomingHttpHeaders }> {
  return await new Promise((resolve, reject) => {
    const req = http.request({ hostname, port, path, method, headers }, (res) => {
      res.resume()
      res.on('end', () => resolve({ status: res.statusCode || 0, headers: res.headers }))
    })
    req.on('error', reject)
    req.end()
  })
}

describe('local loopback port proxy', () => {
  const servers: http.Server[] = []
  const proxies: LocalPortProxyController[] = []

  afterEach(async () => {
    for (const proxy of proxies.splice(0)) proxy.destroy()
    for (const server of servers.splice(0)) await close(server)
  })

  async function fixture(handler: http.RequestListener, options: Parameters<typeof createLocalPortProxy>[0] = {}) {
    const upstream = http.createServer(handler)
    const upstreamPort = await listen(upstream)
    servers.push(upstream)

    const proxy = createLocalPortProxy(options)
    proxies.push(proxy)
    const app = express()
    app.use(options.pathPrefix || '/_local', proxy.router)
    const ui = http.createServer(app)
    ui.on('upgrade', (req, socket, head) => {
      if (!proxy.handleUpgrade(req, socket, head)) socket.destroy()
    })
    const uiPort = await listen(ui)
    servers.push(ui)
    return { targetPort: upstreamPort, uiPort }
  }

  it('streams requests to loopback, preserves the path and query, and strips operator credentials and browser context', async () => {
    let received: http.IncomingMessage | undefined
    let cookie: string | undefined
    const { targetPort, uiPort } = await fixture((req, res) => {
      received = req
      cookie = req.headers.cookie
      res.setHeader('content-type', 'application/json')
      const chunks: Buffer[] = []
      req.on('data', (chunk) => chunks.push(Buffer.from(chunk)))
      req.on('end', () => res.end(JSON.stringify({ method: req.method, url: req.url, body: Buffer.concat(chunks).toString() })))
    })
    const tokenResponse = await requestStatus(uiPort, `/_local/${targetPort}/`, 'GET', { accept: 'text/html', 'sec-fetch-site': 'same-origin' })
    const capability = new URL(tokenResponse.headers.location!, `http://127.0.0.1:${uiPort}`).pathname.split('/')[3]
    const result = await fetch(`http://127.0.0.1:${uiPort}/_local/${targetPort}/${capability}/task/run?mode=fast`, {
      method: 'POST',
      headers: {
        authorization: 'Bearer web-console-secret',
        cookie: `web-console-session=secret; vroot_p${targetPort}_service=ok; vroot_p9999_other=no`,
        origin: `http://127.0.0.1:${uiPort}`,
      },
      body: 'payload',
    })
    expect(result.status).toBe(200)
    expect(received?.url).toBe('/task/run?mode=fast')
    expect(received?.method).toBe('POST')
    expect(received?.headers.host).toBe(`127.0.0.1:${targetPort}`)
    expect(received?.headers.authorization).toBeUndefined()
    // The proxy's origin is not the service's; forwarding it would make the
    // service's own origin policy judge Web Console's page.
    expect(received?.headers.origin).toBeUndefined()
    expect(cookie).toBe('service=ok')
    expect(await result.text()).toContain('payload')
  })

  it('signals retry after a loopback service disconnect and restores the same port-scoped draft when it returns', async () => {
    let serviceAvailable = true
    const drafts = new Map<string, string>()
    const { targetPort, uiPort } = await fixture((req, res) => {
      if (!serviceAvailable) {
        req.socket.destroy()
        return
      }
      const session = String(req.headers.cookie || '').match(/(?:^|;\s*)sid=([^;]+)/)?.[1] || 'anonymous'
      if (req.method === 'POST' && req.url === '/draft?view=setup') {
        const chunks: Buffer[] = []
        req.on('data', (chunk) => chunks.push(Buffer.from(chunk)))
        req.on('end', () => {
          drafts.set(session, Buffer.concat(chunks).toString())
          res.statusCode = 204
          res.end()
        })
        return
      }
      if (req.method === 'GET' && req.url === '/draft?view=setup') {
        res.setHeader('content-type', 'application/json')
        res.end(JSON.stringify({ text: drafts.get(session) || '' }))
        return
      }
      res.statusCode = 404
      res.end()
    })
    const bootstrap = await requestStatus(uiPort, `/_local/${targetPort}/`, 'GET', { accept: 'text/html', 'sec-fetch-site': 'same-origin' })
    const capability = new URL(bootstrap.headers.location!, `http://127.0.0.1:${uiPort}`).pathname.split('/')[3]
    const draftPath = `http://127.0.0.1:${uiPort}/_local/${targetPort}/${capability}/draft?view=setup`
    const cookie = `vroot_p${targetPort}_sid=operator-1; vroot_p9999_sid=other-session`

    const saved = await fetch(draftPath, { method: 'POST', headers: { cookie }, body: 'unfinished setup draft' })
    expect(saved.status).toBe(204)

    serviceAvailable = false
    const lost = await fetch(draftPath, { headers: { cookie } })
    expect(lost.status).toBe(502)
    expect(lost.headers.get('retry-after')).toBe('1')
    expect(await lost.json()).toEqual({ error: 'Local service unavailable' })

    serviceAvailable = true
    const recovered = await fetch(draftPath, { headers: { cookie } })
    expect(recovered.status).toBe(200)
    expect(await recovered.json()).toEqual({ text: 'unfinished setup draft' })
    expect([...drafts.entries()]).toEqual([['operator-1', 'unfinished setup draft']])
  })

  it('injects a relative-path base tag, sandboxes HTML, and rewrites same-port redirects', async () => {
    const { targetPort, uiPort } = await fixture((req, res) => {
      if (req.url === '/redirect') {
        res.writeHead(302, { location: '/dashboard?tab=jobs' }).end()
        return
      }
      if (req.url === '/nested/login') {
        res.writeHead(302, { location: '../dashboard', 'cache-control': 'public, max-age=3600' }).end()
        return
      }
      const body = '<!doctype html><html><head></head><body>ok</body></html>'
      res.writeHead(200, { 'content-type': 'text/html; charset=utf-8', 'content-length': String(Buffer.byteLength(body)), 'content-security-policy': "default-src 'self'; object-src 'none'" })
      res.end(body)
    })
    const html = await fetch(`http://127.0.0.1:${uiPort}/_local/${targetPort}/`, { headers: { accept: 'text/html', 'sec-fetch-site': 'same-origin' } })
    const body = await html.text()
    expect(html.status).toBe(200)
    expect(body).toMatch(new RegExp(`<base href="/_local/${targetPort}/\\d{13}\\.[A-Za-z0-9_-]{43}/">`))
    expect(body).toContain('window.WebSocket')
    expect(body).toContain('Element.prototype.setAttribute=function')
    expect(body).toContain('HTMLImageElement,"src"')
    expect(body).toContain('credentials:"include"')
    expect(body).toContain('void navigator.serviceWorker')
    expect(body).toContain('Service workers are unavailable in the isolated local preview')
    expect(body).toContain('getRegistrations:()=>Promise.resolve([])')
    expect(body).toContain('addEventListener:()=>{}')
    expect(body).toContain('u.host===location.host&&!proxyRequest(u.href)')
    const bootstrap = body.match(/<script(?:\s[^>]*)?>([\s\S]*?)<\/script>/i)?.[1]
    expect(bootstrap).toBeTruthy()
    expect(() => new Function(bootstrap!)).not.toThrow()
    expect(html.headers.get('content-security-policy')).toContain('sandbox')
    expect(html.headers.get('content-security-policy')).not.toContain('allow-same-origin')
    expect(html.headers.get('content-length')).toBeNull()
    const injectedNonce = body.match(/<script nonce="([^"]+)">/)?.[1]
    expect(injectedNonce).toBeTruthy()
    expect(html.headers.get('content-security-policy')).toContain(`'nonce-${injectedNonce}'`)
    const tokenResponse = await requestStatus(uiPort, `/_local/${targetPort}/`, 'GET', { accept: 'text/html', 'sec-fetch-site': 'same-origin' })
    const capability = new URL(tokenResponse.headers.location!, `http://127.0.0.1:${uiPort}`).pathname.split('/')[3]
    const redirect = await fetch(`http://127.0.0.1:${uiPort}/_local/${targetPort}/${capability}/redirect`, { redirect: 'manual' })
    expect(redirect.headers.get('location')).toMatch(new RegExp(`^/_local/${targetPort}/\\d{13}\\.[A-Za-z0-9_-]{43}/dashboard\\?tab=jobs$`))
    const relativeRedirect = await fetch(`http://127.0.0.1:${uiPort}/_local/${targetPort}/${capability}/nested/login`, { redirect: 'manual' })
    expect(relativeRedirect.headers.get('location')).toBe(`/_local/${targetPort}/${capability}/dashboard`)
    expect(relativeRedirect.headers.get('cache-control')).toBe('no-store')
  })

  it('rewrites root-relative HTML and CSS resources in compressed streaming responses', async () => {
    const html = '<!doctype html><html><head></head><body><img src="/images/logo.svg"><form action="/save"><link href="/assets/app.css"></form></body></html>'
    const compressed = gzipSync(Buffer.from(html))
    const { targetPort, uiPort } = await fixture((_req, res) => {
      res.writeHead(200, { 'content-type': 'text/html', 'content-encoding': 'gzip' })
      res.write(compressed.subarray(0, 17))
      res.end(compressed.subarray(17))
    })
    const response = await fetch(`http://127.0.0.1:${uiPort}/_local/${targetPort}/`, { headers: { accept: 'text/html', 'sec-fetch-site': 'same-origin' } })
    const body = await response.text()
    const prefix = `/_local/${targetPort}/\\d{13}\\.[A-Za-z0-9_-]{43}`
    expect(body).toMatch(new RegExp(`<img src="${prefix}/images/logo\\.svg">`))
    expect(body).toMatch(new RegExp(`<form action="${prefix}/save">`))
    expect(body).toMatch(new RegExp(`<link href="${prefix}/assets/app\\.css">`))
    expect(body).toContain('credentials:"include"')
  })

  it('rewrites root-relative CSS URLs and safely fails malformed compressed CSS', async () => {
    const css = Buffer.from('body{background:url("/images/bg.png")}')
    const { targetPort, uiPort } = await fixture((req, res) => {
      if (req.url === '/broken.css') {
        res.writeHead(200, { 'content-type': 'text/css', 'content-encoding': 'gzip' })
        res.end('not-gzip')
        return
      }
      res.writeHead(200, { 'content-type': 'text/css' })
      res.end(css)
    })
    const tokenResponse = await requestStatus(uiPort, `/_local/${targetPort}/`, 'GET', { accept: 'text/html', 'sec-fetch-site': 'same-origin' })
    const capability = new URL(tokenResponse.headers.location!, `http://127.0.0.1:${uiPort}`).pathname.split('/')[3]
    const result = await fetch(`http://127.0.0.1:${uiPort}/_local/${targetPort}/${capability}/styles.css`)
    expect(await result.text()).toBe(`body{background:url("/_local/${targetPort}/${capability}/images/bg.png")}`)
    await expect(fetch(`http://127.0.0.1:${uiPort}/_local/${targetPort}/${capability}/broken.css`)).rejects.toThrow()
  })

  it('isolates response cookies by port and scopes their browser path', async () => {
    const { targetPort, uiPort } = await fixture((_req, res) => {
      res.setHeader('set-cookie', ['sid=abc; Path=/; HttpOnly', 'theme=dark; Domain=localhost; SameSite=Lax'])
      res.end('ok')
    })
    const tokenResponse = await requestStatus(uiPort, `/_local/${targetPort}/`, 'GET', { accept: 'text/html', 'sec-fetch-site': 'same-origin' })
    const capability = new URL(tokenResponse.headers.location!, `http://127.0.0.1:${uiPort}`).pathname.split('/')[3]
    const result = await fetch(`http://127.0.0.1:${uiPort}/_local/${targetPort}/${capability}/`)
    const cookies = result.headers.getSetCookie()
    expect(cookies).toHaveLength(2)
    expect(cookies[0]).toContain(`vroot_p${targetPort}_sid=abc`)
    expect(cookies[0]).toContain(`Path=/_local/${targetPort}/`)
    expect(cookies[1]).toContain(`vroot_p${targetPort}_theme=dark`)
    expect(cookies[1]).not.toContain('Domain=')
  })

  it('rejects malformed, excluded, traversing, and non-HTTP methods without contacting upstream', async () => {
    let hits = 0
    const { targetPort, uiPort } = await fixture((_req, res) => { hits++; res.end('unexpected') }, { excludedPorts: ["19999"] })
    for (const path of [`000${targetPort}`, '0', '65536', 'abc', '19999']) {
      const result = await fetch(`http://127.0.0.1:${uiPort}/_local/${path}/`)
      expect(result.status).toBe(400)
    }
    const traversal = await fetch(`http://127.0.0.1:${uiPort}/_local/${targetPort}/%252e%252e/admin`)
    expect(traversal.status).toBe(400)
    let deeplyEncodedTraversal = '%2e%2e'
    for (let i = 0; i < 5; i++) deeplyEncodedTraversal = deeplyEncodedTraversal.replace(/%/g, '%25')
    const deepTraversal = await fetch(`http://127.0.0.1:${uiPort}/_local/${targetPort}/${deeplyEncodedTraversal}/admin`)
    expect(deepTraversal.status).toBe(400)
    const method = await new Promise<number>((resolve, reject) => {
      const request = http.request(`http://127.0.0.1:${uiPort}/_local/${targetPort}/`, { method: 'TRACE' }, (res) => {
        res.resume()
        res.on('end', () => resolve(res.statusCode || 0))
      })
      request.on('error', reject)
      request.end()
    })
    expect(method).toBe(405)
    expect(hits).toBe(0)
  })

  it('requires a per-port capability for writes and allows only opaque-origin credentialed CORS with that capability', async () => {
    let hits = 0
    const { targetPort, uiPort } = await fixture((_req, res) => { hits++; res.end('accepted') })
    const path = `/_local/${targetPort}/write`
    const crossSite = await requestStatus(uiPort, path, 'POST', {
      origin: 'https://attacker.example',
      'sec-fetch-site': 'cross-site',
      'content-type': 'application/json',
    })
    expect(crossSite.status).toBe(403)

    const crossSiteNavigation = await requestStatus(uiPort, `/_local/${targetPort}/`, 'GET', {
      accept: 'text/html',
      'sec-fetch-site': 'cross-site',
    })
    expect(crossSiteNavigation.status).toBe(403)

    const safariCrossSiteDocumentNavigation = await requestStatus(uiPort, `/_local/${targetPort}/`, 'GET', {
      accept: 'text/html',
      'sec-fetch-site': 'cross-site',
      'sec-fetch-mode': 'navigate',
      'sec-fetch-dest': 'document',
    })
    expect(safariCrossSiteDocumentNavigation.status).toBe(200)
    expect(safariCrossSiteDocumentNavigation.headers['cache-control']).toBe('no-store')
    expect(safariCrossSiteDocumentNavigation.headers['x-frame-options']).toBe('DENY')
    expect(safariCrossSiteDocumentNavigation.headers['content-security-policy']).toContain("frame-ancestors 'none'")

    const sameSiteDocumentNavigation = await requestStatus(uiPort, `/_local/${targetPort}/`, 'GET', {
      accept: 'text/html',
      'sec-fetch-site': 'same-site',
      'sec-fetch-mode': 'navigate',
      'sec-fetch-dest': 'document',
    })
    expect(sameSiteDocumentNavigation.status).toBe(302)
    expect(sameSiteDocumentNavigation.headers.location).toMatch(new RegExp(`^/_local/${targetPort}/\\d{13}\\.[A-Za-z0-9_-]{43}/$`))

    const clickedCrossSiteDocumentNavigation = await requestStatus(uiPort, `/_local/${targetPort}/`, 'GET', {
      accept: 'text/html',
      'sec-fetch-site': 'cross-site',
      'sec-fetch-mode': 'navigate',
      'sec-fetch-dest': 'document',
      'sec-fetch-user': '?1',
    })
    expect(clickedCrossSiteDocumentNavigation.status).toBe(302)

    const scriptedCrossSiteNavigation = await requestStatus(uiPort, `/_local/${targetPort}/`, 'GET', {
      accept: 'text/html',
      'sec-fetch-site': 'cross-site',
      'sec-fetch-mode': 'navigate',
      'sec-fetch-dest': 'document',
    })
    expect(scriptedCrossSiteNavigation.status).toBe(200)
    expect(scriptedCrossSiteNavigation.headers.location).toBeUndefined()
    expect(scriptedCrossSiteNavigation.headers['x-frame-options']).toBe('DENY')

    const crossSiteFrame = await requestStatus(uiPort, `/_local/${targetPort}/`, 'GET', {
      accept: 'text/html',
      'sec-fetch-site': 'cross-site',
      'sec-fetch-mode': 'navigate',
      'sec-fetch-dest': 'iframe',
    })
    expect(crossSiteFrame.status).toBe(403)

    const originBearingCrossSiteNavigation = await requestStatus(uiPort, `/_local/${targetPort}/`, 'GET', {
      accept: 'text/html',
      origin: 'https://attacker.example',
      'sec-fetch-site': 'cross-site',
      'sec-fetch-mode': 'navigate',
      'sec-fetch-dest': 'document',
      'sec-fetch-user': '?1',
    })
    expect(originBearingCrossSiteNavigation.status).toBe(403)

    const preflight = await requestStatus(uiPort, path, 'OPTIONS', {
      origin: 'https://attacker.example',
      'sec-fetch-site': 'cross-site',
      'access-control-request-method': 'POST',
    })
    expect(preflight.status).toBe(403)

    const sandboxWrite = await requestStatus(uiPort, path, 'POST', {
      origin: 'null',
      'sec-fetch-site': 'cross-site',
    })
    expect(sandboxWrite.status).toBe(403)
    const unauthenticatedRead = await requestStatus(uiPort, path, 'GET')
    expect(unauthenticatedRead.status).toBe(403)

    const tokenResponse = await requestStatus(uiPort, `/_local/${targetPort}/`, 'GET', { accept: 'text/html', 'sec-fetch-site': 'same-origin' })
    const capability = new URL(tokenResponse.headers.location!, `http://127.0.0.1:${uiPort}`).pathname.split('/')[3]
    const capablePath = `/_local/${targetPort}/${capability}/write`
    const capablePreflight = await requestStatus(uiPort, capablePath, 'OPTIONS', {
      origin: 'null',
      'sec-fetch-site': 'cross-site',
      'access-control-request-method': 'POST',
    })
    expect(capablePreflight.status).toBe(204)
    expect(capablePreflight.headers['access-control-allow-origin']).toBe('null')
    expect(capablePreflight.headers['access-control-allow-credentials']).toBe('true')
    const capableWrite = await requestStatus(uiPort, capablePath, 'POST', {
      origin: 'null',
      'sec-fetch-site': 'cross-site',
    })
    expect(capableWrite.status).toBe(200)
    expect(capableWrite.headers['access-control-allow-origin']).toBe('null')
    expect(capableWrite.headers['access-control-allow-credentials']).toBe('true')
    const opaqueSandboxPostWithoutOrigin = await requestStatus(uiPort, capablePath, 'POST', {
      'sec-fetch-mode': 'cors',
      'content-type': 'application/json',
    })
    expect(opaqueSandboxPostWithoutOrigin.status).toBe(200)
    expect(opaqueSandboxPostWithoutOrigin.headers['access-control-allow-origin']).toBe('null')
    expect(opaqueSandboxPostWithoutOrigin.headers['access-control-allow-credentials']).toBe('true')
    const wrongScheme = await requestStatus(uiPort, capablePath, 'POST', {
      origin: `https://127.0.0.1:${uiPort}`,
      'sec-fetch-site': 'cross-site',
    })
    expect(wrongScheme.status).toBe(403)
    expect(hits).toBe(2)
  })

  it('keeps its capability policy independent of the scenario origin policy', async () => {
    const target = http.createServer((_req, res) => {
      res.setHeader('content-type', 'text/html')
      res.end('<html><head></head><body>local app</body></html>')
    })
    const targetPort = await listen(target)
    servers.push(target)
    const app = createScenarioServer({
      uiPort: '31000',
      apiPort: '32000',
      distDir: '/missing-test-dist',
      localPortProxy: true,
    })
    const ui = app.listen(0, '127.0.0.1') as http.Server
    const uiPort = await new Promise<number>((resolve, reject) => {
      ui.once('error', reject)
      ui.once('listening', () => resolve((ui.address() as { port: number }).port))
    })
    servers.push(ui)
    const response = await requestStatus(uiPort, `/_local/${targetPort}/`, 'GET', {
      accept: 'text/html',
      origin: 'https://attacker.example',
      'sec-fetch-site': 'cross-site',
    })
    expect(response.status).toBe(403)
    expect(response.headers['access-control-allow-origin']).toBeUndefined()
    expect((await requestStatus(uiPort, '/_local/31000/', 'GET')).status).toBe(400)
    expect((await requestStatus(uiPort, '/_local/32000/', 'GET')).status).toBe(400)
  })

  it('rejects direct non-loopback clients even when they forge Fetch Metadata', async (context) => {
    const externalAddress = Object.values(networkInterfaces()).flatMap((entries) => entries || [])
      .find((entry) => entry.family === 'IPv4' && !entry.internal && !entry.address.startsWith('127.'))?.address
    if (!externalAddress) {
      context.skip('No non-loopback IPv4 interface is available for the direct-access regression')
      return
    }
    const upstream = http.createServer((_req, res) => { res.end('local service') })
    const targetPort = await listen(upstream)
    servers.push(upstream)
    const proxy = createLocalPortProxy()
    proxies.push(proxy)
    const ui = http.createServer(express().use('/_local', proxy.router))
    ui.on('upgrade', (req, socket, head) => { if (!proxy.handleUpgrade(req, socket, head)) socket.destroy() })
    const uiPort = await listenOnAnyAddress(ui)
    servers.push(ui)

    const forgedBootstrap = await requestStatus(uiPort, `/_local/${targetPort}/`, 'GET', {
      accept: 'text/html',
      'sec-fetch-site': 'none',
    }, externalAddress)
    expect(forgedBootstrap.status).toBe(403)
    expect(forgedBootstrap.headers.location).toBeUndefined()
  })

  it('relays WebSocket upgrades and bidirectional frames', async () => {
    const target = http.createServer()
    let receivedUrl: string | undefined
    const wsServer = new WebSocket.Server({ noServer: true })
    wsServer.on('connection', (socket) => socket.on('message', (message) => socket.send(message)))
    target.on('upgrade', (req, socket, head) => {
      receivedUrl = req.url
      wsServer.handleUpgrade(req, socket, head, (client) => wsServer.emit('connection', client, req))
    })
    const targetPort = await listen(target)
    servers.push(target)
    const proxy = createLocalPortProxy()
    proxies.push(proxy)
    const ui = http.createServer(express().use('/_local', proxy.router))
    ui.on('upgrade', (req, socket, head) => { if (!proxy.handleUpgrade(req, socket, head)) socket.destroy() })
    const uiPort = await listen(ui)
    servers.push(ui)

    const tokenResponse = await requestStatus(uiPort, `/_local/${targetPort}/`, 'GET', { accept: 'text/html', 'sec-fetch-site': 'same-origin' })
    const capability = new URL(tokenResponse.headers.location!, `http://127.0.0.1:${uiPort}`).pathname.split('/')[3]
    const client = new WebSocket(`ws://127.0.0.1:${uiPort}/_local/${targetPort}/${capability}/events?sig=a%20b&escaped=%2f&plus=a+b`)
    const received = new Promise<string>((resolve, reject) => {
      client.once('error', reject)
      client.once('open', () => client.send('round-trip'))
      client.once('message', (value) => resolve(value.toString()))
    })
    expect(await received).toBe('round-trip')
    expect(receivedUrl).toBe('/events?sig=a%20b&escaped=%2f&plus=a+b')
    client.close()
    wsServer.close()
  })

  it('proxies an explicitly addressed IPv6 loopback service', async (context) => {
    const upstream = http.createServer((_req, res) => {
      res.setHeader('content-type', 'text/plain')
      res.end('ipv6-ok')
    })
    let targetPort: number
    try {
      await new Promise<void>((resolve, reject) => {
        upstream.once('error', reject)
        upstream.listen(0, '::1', resolve)
      })
      targetPort = (upstream.address() as { port: number }).port
    } catch {
      context.skip('IPv6 loopback is unavailable in this environment')
      return
    }
    servers.push(upstream)
    const proxy = createLocalPortProxy()
    proxies.push(proxy)
    const ui = http.createServer(express().use('/_local', proxy.router))
    ui.on('upgrade', (req, socket, head) => { if (!proxy.handleUpgrade(req, socket, head)) socket.destroy() })
    const uiPort = await listen(ui)
    servers.push(ui)
    const bootstrap = await requestStatus(uiPort, `/_local/ipv6/${targetPort}/health`, 'GET', { accept: 'text/html', 'sec-fetch-site': 'same-origin' })
    const capability = new URL(bootstrap.headers.location!, `http://127.0.0.1:${uiPort}`).pathname.split('/')[4]
    const result = await fetch(`http://127.0.0.1:${uiPort}/_local/ipv6/${targetPort}/${capability}/health`)
    expect(result.status).toBe(200)
    expect(await result.text()).toBe('ipv6-ok')
  })

  it('binds capabilities to the selected loopback address family', async (context) => {
    const ipv4 = http.createServer((_req, res) => { res.end('ipv4') })
    const ipv4Port = await listen(ipv4)
    servers.push(ipv4)
    const ipv6 = http.createServer((_req, res) => { res.end('ipv6') })
    try {
      await new Promise<void>((resolve, reject) => {
        ipv6.once('error', reject)
        ipv6.listen(ipv4Port, '::1', resolve)
      })
    } catch {
      context.skip('IPv4 and IPv6 loopback cannot bind the same numeric port in this environment')
      return
    }
    servers.push(ipv6)

    const proxy = createLocalPortProxy()
    proxies.push(proxy)
    const ui = http.createServer(express().use('/_local', proxy.router))
    ui.on('upgrade', (req, socket, head) => { if (!proxy.handleUpgrade(req, socket, head)) socket.destroy() })
    const uiPort = await listen(ui)
    servers.push(ui)

    const ipv4Bootstrap = await requestStatus(uiPort, `/_local/${ipv4Port}/`, 'GET', { accept: 'text/html', 'sec-fetch-site': 'same-origin' })
    const ipv4Capability = new URL(ipv4Bootstrap.headers.location!, `http://127.0.0.1:${uiPort}`).pathname.split('/')[3]
    const ipv6Bootstrap = await requestStatus(uiPort, `/_local/ipv6/${ipv4Port}/`, 'GET', { accept: 'text/html', 'sec-fetch-site': 'same-origin' })
    const ipv6Capability = new URL(ipv6Bootstrap.headers.location!, `http://127.0.0.1:${uiPort}`).pathname.split('/')[4]

    expect((await requestStatus(uiPort, `/_local/ipv6/${ipv4Port}/${ipv4Capability}/`, 'GET')).status).toBe(403)
    expect((await requestStatus(uiPort, `/_local/${ipv4Port}/${ipv6Capability}/`, 'GET')).status).toBe(403)
    const ipv4Result = await fetch(`http://127.0.0.1:${uiPort}/_local/${ipv4Port}/${ipv4Capability}/`)
    const ipv6Result = await fetch(`http://127.0.0.1:${uiPort}/_local/ipv6/${ipv4Port}/${ipv6Capability}/`)
    expect(await ipv4Result.text()).toBe('ipv4')
    expect(await ipv6Result.text()).toBe('ipv6')
  })
})
