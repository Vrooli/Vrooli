import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import express from 'express'
import * as http from 'node:http'
import type { AddressInfo } from 'node:net'
import { createProxyMiddleware } from '../../server/proxy.js'
import { resetProxyAgentsForTesting } from '../../server/agent.js'

async function findAvailablePort(startPort: number): Promise<number> {
  const server = http.createServer()
  return new Promise((resolve, reject) => {
    server.once('error', error => {
      server.close()
      if ((error as NodeJS.ErrnoException).code === 'EADDRINUSE') {
        resolve(findAvailablePort(startPort + 1))
      } else {
        reject(error)
      }
    })

    server.listen(startPort, '127.0.0.1', () => {
      const { port } = server.address() as AddressInfo
      server.close(() => resolve(port))
    })
  })
}

describe('HTTP proxy keep-alive integration', () => {
  let apiServer: http.Server
  let apiPort: number
  let socketCounter = 0

  beforeAll(async () => {
    apiPort = await findAvailablePort(48000)
    apiServer = http.createServer((req, res) => {
      const socketId = (req.socket as any).__proxyId
      res.statusCode = 200
      res.setHeader('Content-Type', 'text/plain')
      res.setHeader('Connection', 'keep-alive')
      res.end(String(socketId))
    })

    apiServer.on('connection', socket => {
      socketCounter += 1
      ;(socket as any).__proxyId = socketCounter
    })

    await new Promise<void>(resolve => {
      apiServer.listen(apiPort, '127.0.0.1', () => resolve())
    })
  })

  afterAll(async () => {
    await new Promise<void>(resolve => {
      apiServer.close(() => resolve())
    })
  })

  afterEach(() => {
    resetProxyAgentsForTesting()
  })

  it('reuses the same upstream socket by default', async () => {
    const { server, port } = await startUiProxy()

    const first = await fetchSocketId(port)
    const second = await fetchSocketId(port)

    expect(first).toBeGreaterThan(0)
    expect(second).toBe(first)

    await stopServer(server)
  })

  async function startUiProxy(keepAlive?: boolean) {
    const app = express()
    app.use('/api', createProxyMiddleware({
      apiPort,
      apiHost: '127.0.0.1',
      keepAlive,
    }))

    const server: http.Server = await new Promise(resolve => {
      const created = app.listen(0, '127.0.0.1', () => resolve(created))
    })

    const address = server.address() as AddressInfo
    return { server, port: address.port }
  }

  async function stopServer(server: http.Server): Promise<void> {
    await new Promise<void>(resolve => server.close(() => resolve()))
  }

  async function fetchSocketId(port: number): Promise<number> {
    return new Promise<number>((resolve, reject) => {
      const req = http.request({
        hostname: '127.0.0.1',
        port,
        path: '/api/test',
        method: 'GET',
      }, res => {
        const chunks: Buffer[] = []
        res.on('data', chunk => chunks.push(Buffer.from(chunk)))
        res.on('end', () => {
          const text = Buffer.concat(chunks).toString('utf8')
          resolve(Number.parseInt(text, 10))
        })
      })

      req.on('error', reject)
      req.end()
    })
  }
})
