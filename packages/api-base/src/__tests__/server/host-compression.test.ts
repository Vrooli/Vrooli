import { afterEach, describe, expect, it } from 'vitest'
import express from 'express'
import * as http from 'node:http'
import { gzipSync, deflateSync, brotliCompressSync } from 'node:zlib'
import { createScenarioProxyHost } from '../../server/host.js'

const cleanup: Array<() => Promise<void> | void> = []
afterEach(async () => { for (const close of cleanup.splice(0).reverse()) await close() })
async function listen(server: http.Server): Promise<number> {
  await new Promise<void>(resolve => server.listen(0, '127.0.0.1', resolve))
  cleanup.push(() => new Promise<void>((resolve, reject) => server.close(error => error ? reject(error) : resolve())))
  const address = server.address()
  if (!address || typeof address === 'string') throw new Error('Test listener unavailable')
  return address.port
}
async function request(port: number) {
  return await new Promise<{ status: number; headers: http.IncomingHttpHeaders; body: string }>((resolve, reject) => {
    http.get({ hostname: '127.0.0.1', port, path: '/apps/child/proxy/', headers: { 'accept-encoding': 'gzip, br', accept: 'text/html' } }, response => {
      const chunks: Buffer[] = []
      response.on('data', chunk => chunks.push(Buffer.from(chunk)))
      response.on('end', () => resolve({ status: response.statusCode ?? 0, headers: response.headers, body: Buffer.concat(chunks).toString('utf8') }))
      response.on('error', reject)
    }).on('error', reject)
  })
}

describe('hosted HTML compression contract', () => {
  for (const encoding of ['gzip', 'deflate', 'br']) {
    it(`decodes ${encoding} before injection and keeps streamed/cached headers truthful`, async () => {
      const html = '<!doctype html><html><head><title>Native TV child</title></head><body><h1>Your day</h1></body></html>'
      let calls = 0
      let upstreamAcceptEncoding: string | undefined
      const upstream = http.createServer((req, res) => {
        calls++
        upstreamAcceptEncoding = req.headers['accept-encoding']
        // Exercise an upstream that compresses even when identity was requested.
        const bytes = encoding === 'gzip' ? gzipSync(html) : encoding === 'deflate' ? deflateSync(html) : brotliCompressSync(html)
        res.writeHead(200, { 'content-type': 'text/html; charset=utf-8', 'content-encoding': encoding, 'content-length': bytes.length, etag: '"upstream-bytes"', 'cache-control': 'public, max-age=60' })
        res.end(bytes)
      })
      const upstreamPort = await listen(upstream)
      const proxy = createScenarioProxyHost({ hostScenario: 'portal', fetchAppMetadata: async () => ({ id: 'child', scenario_name: 'child', port_mappings: { UI_PORT: upstreamPort } }), enableServerTiming: false })
      cleanup.push(() => proxy.destroy())
      const app = express(); app.use(proxy.router)
      const hostPort = await listen(http.createServer(app))
      for (let attempt = 0; attempt < 2; attempt++) {
        const response = await request(hostPort)
        expect(response.status).toBe(200)
        expect(response.headers['content-encoding']).toBeUndefined()
        expect(response.headers.etag).not.toBe('"upstream-bytes"')
        expect(response.body).toContain('<h1>Your day</h1>')
        expect(response.body).toContain('<base ')
        expect(response.body).not.toContain('\uFFFD')
      }
      expect(upstreamAcceptEncoding).toBe('identity')
      expect(calls).toBe(1)
    })
  }
})
