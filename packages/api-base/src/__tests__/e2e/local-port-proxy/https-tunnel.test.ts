import { afterEach, describe, expect, it } from 'vitest'
import express from 'express'
import * as fs from 'node:fs'
import * as http from 'node:http'
import * as https from 'node:https'
import * as path from 'node:path'
import { fileURLToPath } from 'node:url'
import { chromium, webkit, type Browser, type BrowserType } from 'playwright'
import { createLocalPortProxy } from '../../../server/local-port-proxy.js'

const here = path.dirname(fileURLToPath(import.meta.url))
const tunnelKey = fs.readFileSync(path.join(here, 'fixtures/tunnel-test-key.pem'))
const tunnelCert = fs.readFileSync(path.join(here, 'fixtures/tunnel-test-cert.pem'))

type Fixture = {
  appPort: number
  chatPort: number
  tunnelPort: number
  observedRpc: { method?: string; path?: string; cookie?: string; body?: unknown }
  close: () => Promise<void>
}

async function listen(server: http.Server | https.Server, hostname = '127.0.0.1'): Promise<number> {
  await new Promise<void>((resolve, reject) => {
    server.once('error', reject)
    server.listen(0, hostname, resolve)
  })
  return (server.address() as { port: number }).port
}

async function closeServer(server: http.Server | https.Server): Promise<void> {
  if (!server.listening) return
  await new Promise<void>((resolve) => server.close(() => resolve()))
}

async function createFixture(): Promise<Fixture> {
  const observedRpc: Fixture['observedRpc'] = {}
  const app = express()
  app.use(express.json())
  app.get('/', (_req, res) => {
    res.cookie('service_session', 'tunnel-preview', {
      httpOnly: true,
      sameSite: 'none',
      secure: true,
      path: '/',
    })
    res.type('html').send(`<!doctype html>
      <html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width">
      <title>Mizar tunnel regression</title><link rel="stylesheet" href="/assets/app.css">
      <script defer src="/assets/app.js"></script></head>
      <body><main><h1 id="heading">Loading app…</h1><p id="rpc-state">RPC pending</p><output id="rpc-result"></output></main></body></html>`)
  })
  app.get('/assets/app.css', (_req, res) => {
    res.type('text/css').send('html,body{min-height:100%;background:#101827;color:#f8fafc}body{font:18px system-ui;margin:0}main{padding:32px}.ok{color:#22c55e}')
  })
  app.get('/assets/app.js', (_req, res) => {
    res.type('application/javascript').send(`
      document.querySelector('#heading').textContent='Mizar dashboard rendered';
      const logo=document.createElement('img');logo.alt='Dynamic app logo';logo.src=location.origin+'/assets/mark.svg';document.querySelector('main').prepend(logo);
      fetch('/rpc/UsageRecommendations',{method:'POST',headers:{'content-type':'application/json','connect-protocol-version':'1'},body:JSON.stringify({probe:'https-tunnel'})})
        .then(async response=>{if(!response.ok)throw new Error('RPC HTTP '+response.status);const data=await response.json();document.querySelector('#rpc-state').textContent='RPC succeeded';document.querySelector('#rpc-state').className='ok';document.querySelector('#rpc-result').textContent=data.message;document.body.dataset.rpc='ok'})
        .catch(error=>{document.querySelector('#rpc-state').textContent='RPC failed: '+error.message;document.body.dataset.rpc='failed'});
    `)
  })
  app.get('/assets/mark.svg', (_req, res) => {
    res.type('image/svg+xml').send('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 8 8"><rect width="8" height="8" fill="#22c55e"/></svg>')
  })
  app.post('/rpc/UsageRecommendations', (req, res) => {
    observedRpc.method = req.method
    observedRpc.path = req.path
    observedRpc.cookie = req.headers.cookie
    observedRpc.body = req.body
    res.json({ message: 'Usage recommendations loaded' })
  })

  const appServer = http.createServer(app)
  const appPort = await listen(appServer)

  const proxy = createLocalPortProxy()
  const chat = express()
  chat.get('/', (_req, res) => res.type('html').send(`<!doctype html><a href="https://127.0.0.1:${tunnelPort}/_local/${appPort}/" id="open-mizar">Open Mizar through tunnel</a>`))
  chat.get('/chat', (_req, res) => res.type('html').send(`<!doctype html><a href="/_local/${appPort}/" id="open-mizar-rewritten">Open rewritten Mizar link</a>`))
  chat.use('/_local', proxy.router)
  const chatServer = http.createServer(chat)
  chatServer.on('upgrade', (req, socket, head) => {
    if (!proxy.handleUpgrade(req, socket, head)) socket.destroy()
  })
  const chatPort = await listen(chatServer)

  // Model a secure tunnel edge: browser HTTPS terminates here, then the edge
  // forwards plain HTTP to the loopback Web Console server with the public
  // authority and scheme metadata preserved.
  let tunnelPort = 0
  const tunnelServer = https.createServer({ key: tunnelKey, cert: tunnelCert }, (req, res) => {
    const headers = { ...req.headers, host: req.headers.host, 'x-forwarded-proto': 'https' }
    const upstream = http.request({ hostname: '127.0.0.1', port: chatPort, method: req.method, path: req.url, headers }, (upstreamRes) => {
      res.writeHead(upstreamRes.statusCode || 502, upstreamRes.headers)
      upstreamRes.pipe(res)
    })
    upstream.on('error', () => {
      if (!res.headersSent) res.writeHead(502)
      res.end('local tunnel origin unavailable')
    })
    req.pipe(upstream)
  })
  tunnelPort = await listen(tunnelServer)

  // The source chat and tunnel use different sites, so the browser exercises
  // the same user-activated cross-site navigation Web Console users encounter
  // when opening links from an external chat surface.
  const sourceChatServer = http.createServer((_req, res) => {
    res.writeHead(200, { 'content-type': 'text/html; charset=utf-8' })
    res.end(`<a href="https://127.0.0.1:${tunnelPort}/_local/${appPort}/" id="open-mizar">Open Mizar through tunnel</a>`)
  })
  const sourceChatPort = await listen(sourceChatServer, 'localhost')

  return {
    appPort,
    chatPort: sourceChatPort,
    tunnelPort,
    observedRpc,
    close: async () => {
      proxy.destroy()
      await Promise.all([closeServer(sourceChatServer), closeServer(tunnelServer), closeServer(chatServer), closeServer(appServer)])
    },
  }
}

const browserTypes: Array<[string, BrowserType]> = [['Chromium', chromium]]
if (process.env.VROOLI_LOCAL_PROXY_WEBKIT === '1') browserTypes.push(['WebKit', webkit])

describe.each(browserTypes)('shared local-port proxy through HTTPS tunnel (%s)', (_name, browserType) => {
  let fixture: Fixture | undefined
  let browser: Browser | undefined

  afterEach(async () => {
    await browser?.close()
    browser = undefined
    await fixture?.close()
    fixture = undefined
  })

  it('renders the proxied app, its CSS and its opaque-origin RPC after a real cross-site click', async () => {
    fixture = await createFixture()
    browser = await browserType.launch({
      headless: true,
      executablePath: _name === 'WebKit' ? process.env.VROOLI_LOCAL_PROXY_WEBKIT_EXECUTABLE : undefined,
    })
    const context = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true })
    const page = await context.newPage()
    const pageErrors: string[] = []
    const failedRequests: Array<{ url: string; error: string | null }> = []
    const badResponses: Array<{ url: string; status: number }> = []
    const rpcResponses: Array<{ method: string; status: number; allowOrigin: string | undefined }> = []
    page.on('pageerror', (error) => pageErrors.push(error.message))
    page.on('requestfailed', (request) => failedRequests.push({ url: request.url(), error: request.failure()?.errorText || null }))
    page.on('response', (response) => {
      if (response.status() >= 400) badResponses.push({ url: response.url(), status: response.status() })
      if (response.url().includes('/rpc/UsageRecommendations')) {
        rpcResponses.push({
          method: response.request().method(),
          status: response.status(),
          allowOrigin: response.headers()['access-control-allow-origin'],
        })
      }
    })

    await page.goto(`http://localhost:${fixture.chatPort}/`)
    await page.getByRole('link', { name: 'Open Mizar through tunnel' }).click()
    await page.waitForFunction(
      () => Boolean(document.querySelector('#heading') || document.querySelector('#local-proxy-continue')),
      null,
      { timeout: 10_000 },
    )
    if (await page.locator('#local-proxy-continue').count()) {
      await page.getByRole('link', { name: 'Continue' }).click()
    }
    try {
      await page.getByText('RPC succeeded').waitFor({ timeout: 15_000 })
    } catch {
      throw new Error(JSON.stringify({
        url: page.url(),
        body: await page.locator('body').innerText().catch(() => ''),
        pageErrors,
        failedRequests,
        badResponses,
      }, null, 2))
    }

    expect(page.url()).toMatch(new RegExp(`^https://127\\.0\\.0\\.1:${fixture.tunnelPort}/_local/${fixture.appPort}/\\d{13}\\.[A-Za-z0-9_-]{43}/$`))
    await expectVisibleApp(page)
    expect(await page.locator('body').evaluate((element) => getComputedStyle(element).backgroundColor)).toBe('rgb(16, 24, 39)')
    expect(fixture.observedRpc).toMatchObject({
      method: 'POST',
      path: '/rpc/UsageRecommendations',
      cookie: 'service_session=tunnel-preview',
      body: { probe: 'https-tunnel' },
    })
    expect(rpcResponses).toContainEqual({ method: 'POST', status: 200, allowOrigin: 'null' })
    expect(pageErrors).toEqual([])
    await context.close()
  }, 30_000)

  it('opens a localhost link rewritten to the same-origin tunnel route without a confirmation page', async () => {
    fixture = await createFixture()
    browser = await browserType.launch({
      headless: true,
      executablePath: _name === 'WebKit' ? process.env.VROOLI_LOCAL_PROXY_WEBKIT_EXECUTABLE : undefined,
    })
    const context = await browser.newContext({ ignoreHTTPSErrors: true, viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true })
    const page = await context.newPage()
    const pageErrors: string[] = []
    page.on('pageerror', (error) => pageErrors.push(error.message))

    await page.goto(`https://127.0.0.1:${fixture.tunnelPort}/chat`)
    await page.getByRole('link', { name: 'Open rewritten Mizar link' }).click()
    await page.getByText('RPC succeeded').waitFor({ timeout: 15_000 })

    expect(page.url()).toMatch(new RegExp(`^https://127\\.0\\.0\\.1:${fixture.tunnelPort}/_local/${fixture.appPort}/\\d{13}\\.[A-Za-z0-9_-]{43}/$`))
    expect(await page.locator('#local-proxy-continue').count()).toBe(0)
    await expectVisibleApp(page)
    expect(await page.locator('body').evaluate((element) => getComputedStyle(element).backgroundColor)).toBe('rgb(16, 24, 39)')
    expect(fixture.observedRpc).toMatchObject({ method: 'POST', path: '/rpc/UsageRecommendations', cookie: 'service_session=tunnel-preview' })
    expect(pageErrors).toEqual([])
    await context.close()
  }, 30_000)
})

async function expectVisibleApp(page: import('playwright').Page): Promise<void> {
  await page.getByRole('heading', { name: 'Mizar dashboard rendered' }).waitFor()
  await page.getByText('Usage recommendations loaded').waitFor()
    await page.getByRole('img', { name: 'Dynamic app logo' }).waitFor()
  expect(await page.title()).toBe('Mizar tunnel regression')
}
