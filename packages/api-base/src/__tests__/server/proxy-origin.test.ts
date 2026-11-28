import { createServer, get, type Server } from 'node:http'
import type { AddressInfo } from 'node:net'
import express from 'express'
import { expect, it } from 'vitest'
import { proxyToApi } from '../../server/proxy.js'

async function listen(server: Server): Promise<number> {
  await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve))
  return (server.address() as AddressInfo).port
}

it('preserves the routed HTTP authority while replacing untrusted forwarding hints', async () => {
  const upstream = createServer((request, response) => {
    response.setHeader('Content-Type', 'application/json')
    response.end(JSON.stringify({ host: request.headers.host, forwarded: request.headers['x-forwarded-host'] }))
  })
  const apiPort = await listen(upstream)
  const app = express()
  app.use((request, response, next) => { void proxyToApi(request, response, '/', { apiPort, keepAlive: false }).catch(next) })
  const frontend = createServer(app)
  try {
    const port = await listen(frontend)
    const payload = await new Promise((resolve, reject) => {
      get({ hostname: '127.0.0.1', port, path: '/', headers: { host: '192.168.1.173:24965', 'x-forwarded-host': 'untrusted.example' } }, (response) => {
        let body = ''
        response.on('data', (chunk) => { body += String(chunk) })
        response.on('end', () => { try { resolve(JSON.parse(body)) } catch (error) { reject(error) } })
        response.on('error', reject)
      }).on('error', reject)
    })
    expect(payload).toEqual({ host: `127.0.0.1:${apiPort}`, forwarded: '192.168.1.173:24965' })
  } finally {
    for (const server of [frontend, upstream]) {
      server.closeAllConnections()
      await new Promise<void>((resolve, reject) => server.close((error) => error ? reject(error) : resolve()))
    }
  }
})
