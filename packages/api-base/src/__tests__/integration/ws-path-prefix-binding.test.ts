/**
 * Regression tests for wsPathPrefix binding across both server start paths.
 *
 * The existing websocket-proxy integration test attaches an upgrade handler by
 * hand, so it proves proxyWebSocketUpgrade works but never exercises the
 * wsPathPrefix option itself. That gap allowed a configured wsPathPrefix to be
 * silently dropped whenever a scenario started its app with app.listen instead
 * of startScenarioServer: the upgrade fell through to the ordinary HTTP proxy,
 * arrived at the API as a plain GET, and was rejected there with a 400 that
 * named nothing about the missing binding.
 *
 * These tests drive the option end to end on both start paths.
 */

import { describe, it, expect, beforeAll, afterAll } from 'vitest'
import { createScenarioServer, startScenarioServer } from '../../server/index.js'
import * as http from 'node:http'
import WebSocket from 'ws'
import type { Server } from 'node:http'

const UPGRADE_TIMEOUT_MS = 5000

describe('wsPathPrefix binding', () => {
  let apiServer: Server
  let wsServer: WebSocket.Server
  let apiPort: number
  const started: Server[] = []

  beforeAll(async () => {
    apiPort = await findAvailablePort(31200)

    apiServer = http.createServer((_req, res) => {
      res.writeHead(404)
      res.end()
    })

    wsServer = new WebSocket.Server({ noServer: true })
    wsServer.on('connection', (ws, request) => {
      // Echo the path back so the test can assert the upstream URL was
      // preserved, not just that some socket opened.
      ws.send(`connected:${request.url}`)
    })

    apiServer.on('upgrade', (request, socket, head) => {
      if (request.url?.startsWith('/api/v1/voice/stream')) {
        wsServer.handleUpgrade(request, socket, head, (ws) => {
          wsServer.emit('connection', ws, request)
        })
        return
      }
      socket.destroy()
    })

    await new Promise<void>((resolve) => apiServer.listen(apiPort, '127.0.0.1', resolve))
  })

  afterAll(async () => {
    for (const server of started) {
      if (typeof server?.close !== 'function') continue
      await new Promise<void>((resolve) => server.close(() => resolve()))
    }
    await new Promise<void>((resolve) => wsServer.close(() => resolve()))
    await new Promise<void>((resolve) => apiServer.close(() => resolve()))
  })

  /** Open a socket through the UI port and return the server's first message. */
  async function connectThrough(uiPort: number): Promise<string> {
    const ws = new WebSocket(`ws://127.0.0.1:${uiPort}/api/v1/voice/stream?format=pcm_s16le`)
    try {
      return await new Promise<string>((resolve, reject) => {
        const timer = setTimeout(() => reject(new Error('upgrade timeout')), UPGRADE_TIMEOUT_MS)
        ws.on('message', (data) => {
          clearTimeout(timer)
          resolve(data.toString())
        })
        ws.on('error', (error) => {
          clearTimeout(timer)
          reject(error)
        })
      })
    } finally {
      ws.close()
    }
  }

  it('proxies upgrades when the caller starts the app with app.listen', async () => {
    const uiPort = await findAvailablePort(31300)
    const app = createScenarioServer({
      uiPort,
      apiPort,
      distDir: './dist',
      serviceName: 'ws-prefix-listen',
      verbose: false,
      wsPathPrefix: '/api/v1/voice/stream',
      wsPathTransform: (path) => path,
    })

    const server = await new Promise<Server>((resolve) => {
      const listening = app.listen(uiPort, '127.0.0.1', () => resolve(listening))
    })
    started.push(server)

    await expect(connectThrough(uiPort)).resolves.toBe(
      'connected:/api/v1/voice/stream?format=pcm_s16le'
    )
  })

  it('proxies upgrades when startScenarioServer owns the http server', async () => {
    const uiPort = await findAvailablePort(31400)
    const app = startScenarioServer({
      uiPort,
      apiPort,
      distDir: './dist',
      serviceName: 'ws-prefix-start',
      verbose: false,
      wsPathPrefix: '/api/v1/voice/stream',
      wsPathTransform: (path) => path,
    })
    // startScenarioServer owns its http.Server and does not expose it, so this
    // listener is closed by process exit rather than by afterAll.
    void app

    await expect(connectThrough(uiPort)).resolves.toBe(
      'connected:/api/v1/voice/stream?format=pcm_s16le'
    )
  })

  it('preserves the upstream path for a deep prefix with no transform', async () => {
    // The former default rewrote the matched prefix to '/api/v1', so this
    // configuration reached the API as '/api/v1?format=...' and was refused.
    // wsPathPrefix selects; it must not rewrite.
    const uiPort = await findAvailablePort(31600)
    const app = createScenarioServer({
      uiPort,
      apiPort,
      distDir: './dist',
      serviceName: 'ws-prefix-default',
      verbose: false,
      wsPathPrefix: '/api/v1/voice/stream',
    })

    const server = await new Promise<Server>((resolve) => {
      const listening = app.listen(uiPort, '127.0.0.1', () => resolve(listening))
    })
    started.push(server)

    await expect(connectThrough(uiPort)).resolves.toBe(
      'connected:/api/v1/voice/stream?format=pcm_s16le'
    )
  })

  it('is unchanged for an /api/v1 prefix with no transform', async () => {
    // scenario-to-cloud's shape: the only in-repo consumer that relied on the
    // former default. Because the match guard admits only URLs already starting
    // with the prefix, the old rewrite was the identity function here, so the
    // new preserve-by-default behaviour is indistinguishable. This locks that
    // equivalence rather than leaving it as an argument in a commit message.
    const uiPort = await findAvailablePort(31800)
    const app = createScenarioServer({
      uiPort,
      apiPort,
      distDir: './dist',
      serviceName: 'ws-prefix-apiv1',
      verbose: false,
      wsPathPrefix: '/api/v1',
    })

    const server = await new Promise<Server>((resolve) => {
      const listening = app.listen(uiPort, '127.0.0.1', () => resolve(listening))
    })
    started.push(server)

    await expect(connectThrough(uiPort)).resolves.toBe(
      'connected:/api/v1/voice/stream?format=pcm_s16le'
    )
  })

  it('still honours an explicit remap transform', async () => {
    const uiPort = await findAvailablePort(31700)
    const app = createScenarioServer({
      uiPort,
      apiPort,
      distDir: './dist',
      serviceName: 'ws-prefix-remap',
      verbose: false,
      wsPathPrefix: '/ws',
      wsPathTransform: (url) => url.replace(/^\/ws/, '/api/v1/voice/stream'),
    })

    const server = await new Promise<Server>((resolve) => {
      const listening = app.listen(uiPort, '127.0.0.1', () => resolve(listening))
    })
    started.push(server)

    const ws = new WebSocket(`ws://127.0.0.1:${uiPort}/ws?format=pcm_s16le`)
    const first = await new Promise<string>((resolve, reject) => {
      const timer = setTimeout(() => reject(new Error('upgrade timeout')), UPGRADE_TIMEOUT_MS)
      ws.on('message', (data) => {
        clearTimeout(timer)
        resolve(data.toString())
      })
      ws.on('error', (error) => {
        clearTimeout(timer)
        reject(error)
      })
    })
    ws.close()

    expect(first).toBe('connected:/api/v1/voice/stream?format=pcm_s16le')
  })

  it('delivers each upgrade exactly once when listen is wrapped', async () => {
    const uiPort = await findAvailablePort(31500)
    const app = createScenarioServer({
      uiPort,
      apiPort,
      distDir: './dist',
      serviceName: 'ws-prefix-once',
      verbose: false,
      wsPathPrefix: '/api/v1/voice/stream',
      wsPathTransform: (path) => path,
    })

    const server = await new Promise<Server>((resolve) => {
      const listening = app.listen(uiPort, '127.0.0.1', () => resolve(listening))
    })
    started.push(server)

    // A double-bound handler proxies the same upgrade twice, which surfaces as
    // two upstream connections for one client socket.
    let upstreamConnections = 0
    wsServer.on('connection', () => {
      upstreamConnections += 1
    })

    await connectThrough(uiPort)
    await new Promise((resolve) => setTimeout(resolve, 250))

    expect(upstreamConnections).toBe(1)
  })
})

async function findAvailablePort(startPort: number): Promise<number> {
  return new Promise((resolve, reject) => {
    const server = http.createServer()
    server.listen(startPort, '127.0.0.1', () => {
      const address = server.address()
      if (address && typeof address === 'object') {
        const port = address.port
        server.close(() => resolve(port))
        return
      }
      reject(new Error('Failed to get server address'))
    })
    server.on('error', () => {
      findAvailablePort(startPort + 1).then(resolve, reject)
    })
  })
}
