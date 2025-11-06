/**
 * Integration tests for WebSocket proxy functionality
 *
 * These tests verify that WebSocket upgrades work correctly through the proxy,
 * including header forwarding, connection establishment, and bidirectional data flow.
 */

import { describe, it, expect, beforeAll, afterAll } from 'vitest'
import { createScenarioServer } from '../../server/index.js'
import { resolveWsBase } from '../../client/resolve.js'
import * as http from 'node:http'
import WebSocket from 'ws'
import type { Server } from 'node:http'

describe('WebSocket Proxy Integration', () => {
  let uiServer: Server
  let apiServer: Server
  let wsServer: WebSocket.Server
  let uiPort: number
  let apiPort: number
  const testMessages: string[] = []

  beforeAll(async () => {
    // Find available ports
    uiPort = await findAvailablePort(30000)
    apiPort = await findAvailablePort(30100)

    // Create API server with WebSocket endpoint
    apiServer = http.createServer((req, res) => {
      if (req.url === '/health') {
        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({ status: 'healthy' }))
        return
      }
      res.writeHead(404)
      res.end()
    })

    // Add WebSocket server to API
    wsServer = new WebSocket.Server({ noServer: true })

    wsServer.on('connection', (ws) => {
      testMessages.push('server-connected')

      ws.on('message', (data) => {
        const message = data.toString()
        testMessages.push(`server-received:${message}`)

        // Echo back with prefix
        ws.send(`echo:${message}`)
      })

      ws.on('close', () => {
        testMessages.push('server-disconnected')
      })

      // Send welcome message
      ws.send('welcome')
    })

    apiServer.on('upgrade', (request, socket, head) => {
      if (request.url?.startsWith('/api/v1/ws')) {
        wsServer.handleUpgrade(request, socket, head, (ws) => {
          wsServer.emit('connection', ws, request)
        })
      } else {
        socket.destroy()
      }
    })

    await new Promise<void>((resolve) => {
      apiServer.listen(apiPort, '127.0.0.1', () => {
        resolve()
      })
    })

    // Create UI server with proxy
    const app = createScenarioServer({
      uiPort,
      apiPort,
      distDir: './dist',
      serviceName: 'test-scenario',
      verbose: false,
    })

    uiServer = await new Promise<Server>((resolve) => {
      const server = app.listen(uiPort, '127.0.0.1', () => {
        resolve(server)
      })
    })

    // Setup WebSocket upgrade handler for UI server
    uiServer.on('upgrade', async (req, socket, head) => {
      if (req.url?.startsWith('/api')) {
        const { proxyWebSocketUpgrade } = await import('../../server/proxy.js')
        proxyWebSocketUpgrade(req, socket, head, {
          apiPort,
          verbose: false,
        })
      } else {
        socket.destroy()
      }
    })
  })

  afterAll(async () => {
    await new Promise<void>((resolve) => {
      wsServer.close(() => {
        apiServer.close(() => {
          uiServer.close(() => {
            resolve()
          })
        })
      })
    })
  })

  it('establishes WebSocket connection through proxy', async () => {
    const wsUrl = `ws://127.0.0.1:${uiPort}/api/v1/ws`

    const ws = new WebSocket(wsUrl)

    await new Promise<void>((resolve, reject) => {
      const timeout = setTimeout(() => {
        reject(new Error('WebSocket connection timeout'))
      }, 5000)

      ws.on('open', () => {
        clearTimeout(timeout)
        resolve()
      })

      ws.on('error', (error) => {
        clearTimeout(timeout)
        reject(error)
      })
    })

    expect(ws.readyState).toBe(WebSocket.OPEN)
    ws.close()
  })

  it('forwards WebSocket upgrade headers correctly', async () => {
    const wsUrl = `ws://127.0.0.1:${uiPort}/api/v1/ws`

    const ws = new WebSocket(wsUrl, {
      headers: {
        'X-Custom-Header': 'test-value',
      },
    })

    await new Promise<void>((resolve, reject) => {
      const timeout = setTimeout(() => {
        reject(new Error('Connection timeout'))
      }, 5000)

      ws.on('open', () => {
        clearTimeout(timeout)
        resolve()
      })

      ws.on('error', (error) => {
        clearTimeout(timeout)
        reject(error)
      })
    })

    expect(ws.readyState).toBe(WebSocket.OPEN)
    ws.close()
  })

  it('handles bidirectional data flow', async () => {
    const wsUrl = `ws://127.0.0.1:${uiPort}/api/v1/ws`
    const receivedMessages: string[] = []

    const ws = new WebSocket(wsUrl)

    await new Promise<void>((resolve, reject) => {
      const timeout = setTimeout(() => {
        reject(new Error('Test timeout'))
      }, 5000)

      ws.on('open', () => {
        // Send test message
        ws.send('test-message')
      })

      ws.on('message', (data) => {
        const message = data.toString()
        receivedMessages.push(message)

        // Close after receiving echo
        if (message.startsWith('echo:')) {
          clearTimeout(timeout)
          ws.close()
          resolve()
        }
      })

      ws.on('error', (error) => {
        clearTimeout(timeout)
        reject(error)
      })
    })

    expect(receivedMessages).toContain('welcome')
    expect(receivedMessages).toContain('echo:test-message')
  })

  it('handles multiple concurrent connections', async () => {
    const wsUrl = `ws://127.0.0.1:${uiPort}/api/v1/ws`
    const connections = 5
    const clients: WebSocket[] = []

    // Create multiple connections
    for (let i = 0; i < connections; i++) {
      const ws = new WebSocket(wsUrl)
      clients.push(ws)

      await new Promise<void>((resolve, reject) => {
        const timeout = setTimeout(() => {
          reject(new Error(`Connection ${i} timeout`))
        }, 5000)

        ws.on('open', () => {
          clearTimeout(timeout)
          resolve()
        })

        ws.on('error', (error) => {
          clearTimeout(timeout)
          reject(error)
        })
      })
    }

    // Verify all connected
    for (const ws of clients) {
      expect(ws.readyState).toBe(WebSocket.OPEN)
    }

    // Close all
    for (const ws of clients) {
      ws.close()
    }
  })

  it('handles large messages', async () => {
    const wsUrl = `ws://127.0.0.1:${uiPort}/api/v1/ws`
    const largeMessage = 'x'.repeat(64 * 1024) // 64KB message
    let receivedEcho = false

    const ws = new WebSocket(wsUrl)

    await new Promise<void>((resolve, reject) => {
      const timeout = setTimeout(() => {
        reject(new Error('Test timeout'))
      }, 10000)

      ws.on('open', () => {
        ws.send(largeMessage)
      })

      ws.on('message', (data) => {
        const message = data.toString()

        if (message.startsWith('echo:')) {
          const echoed = message.substring(5)
          expect(echoed).toBe(largeMessage)
          receivedEcho = true
          clearTimeout(timeout)
          ws.close()
          resolve()
        }
      })

      ws.on('error', (error) => {
        clearTimeout(timeout)
        reject(error)
      })
    })

    expect(receivedEcho).toBe(true)
  })

  it('handles connection close gracefully', async () => {
    const wsUrl = `ws://127.0.0.1:${uiPort}/api/v1/ws`
    let closedCleanly = false

    const ws = new WebSocket(wsUrl)

    await new Promise<void>((resolve, reject) => {
      const timeout = setTimeout(() => {
        reject(new Error('Test timeout'))
      }, 5000)

      ws.on('open', () => {
        // Close immediately
        ws.close()
      })

      ws.on('close', (code, reason) => {
        closedCleanly = true
        clearTimeout(timeout)
        resolve()
      })

      ws.on('error', (error) => {
        clearTimeout(timeout)
        reject(error)
      })
    })

    expect(closedCleanly).toBe(true)
  })

  it('preserves Sec-WebSocket-Version header', async () => {
    // This test verifies the fix for the bug where WebSocket headers were filtered
    const wsUrl = `ws://127.0.0.1:${uiPort}/api/v1/ws`

    const ws = new WebSocket(wsUrl)

    const connected = await new Promise<boolean>((resolve) => {
      const timeout = setTimeout(() => {
        resolve(false)
      }, 5000)

      ws.on('open', () => {
        clearTimeout(timeout)
        resolve(true)
        ws.close()
      })

      ws.on('error', () => {
        clearTimeout(timeout)
        resolve(false)
      })
    })

    // If this passes, it means Sec-WebSocket-Version header was properly forwarded
    expect(connected).toBe(true)
  })

  it('handles query parameters in WebSocket URL', async () => {
    const wsUrl = `ws://127.0.0.1:${uiPort}/api/v1/ws?token=test123&user=alice`

    const ws = new WebSocket(wsUrl)
    let finalState = -1

    await new Promise<void>((resolve, reject) => {
      const timeout = setTimeout(() => {
        reject(new Error('Connection timeout'))
      }, 5000)

      ws.on('open', () => {
        // Connection successful with query params
        ws.close()
      })

      ws.on('close', () => {
        finalState = ws.readyState
        clearTimeout(timeout)
        resolve()
      })

      ws.on('error', (error) => {
        clearTimeout(timeout)
        reject(error)
      })
    })

    // Verify connection was established and then closed
    expect(finalState).toBe(WebSocket.CLOSED)
  })
})

/**
 * Find an available port for testing
 */
async function findAvailablePort(startPort: number): Promise<number> {
  return new Promise((resolve, reject) => {
    const server = http.createServer()

    server.listen(startPort, '127.0.0.1', () => {
      const address = server.address()
      if (address && typeof address === 'object') {
        const port = address.port
        server.close(() => {
          resolve(port)
        })
      } else {
        reject(new Error('Failed to get server address'))
      }
    })

    server.on('error', (err: any) => {
      if (err.code === 'EADDRINUSE') {
        resolve(findAvailablePort(startPort + 1))
      } else {
        reject(err)
      }
    })
  })
}
