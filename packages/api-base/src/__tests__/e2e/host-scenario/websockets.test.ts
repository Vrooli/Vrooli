/**
 * E2E Tests: WebSocket Support for Host and Child
 *
 * Tests that both host and child scenarios can have working WebSocket connections
 * that don't interfere with each other. Tests simultaneous connections, message
 * delivery, and proper routing.
 */

import { describe, it, expect, beforeAll, afterAll } from 'vitest'
import WebSocket from 'ws'
import type { TestContext } from './setup.js'
import { setupTestEnvironment, cleanupTestEnvironment } from './setup.js'

describe('WebSockets: Host and Child', () => {
  let ctx: TestContext

  // WebSocket servers
  let hostWsServer: WebSocket.Server
  let childWsServer: WebSocket.Server

  // Track messages for verification
  const testMessages: string[] = []

  beforeAll(async () => {
    ctx = await setupTestEnvironment(2000) // Use port offset to avoid conflicts

    // Setup Host WebSocket server
    hostWsServer = new WebSocket.Server({ noServer: true })
    hostWsServer.on('connection', (ws) => {
      testMessages.push('host-ws-connected')

      ws.on('message', (data) => {
        const message = data.toString()
        testMessages.push(`host-ws-received:${message}`)
        ws.send(`host-echo:${message}`)
      })

      ws.on('close', () => {
        testMessages.push('host-ws-disconnected')
      })

      ws.send('host-welcome')
    })

    ctx.hostApiServer.on('upgrade', (request, socket, head) => {
      if (request.url?.startsWith('/api/v1/ws')) {
        hostWsServer.handleUpgrade(request, socket, head, (ws) => {
          hostWsServer.emit('connection', ws, request)
        })
      } else {
        socket.destroy()
      }
    })

    // Setup Child WebSocket server
    childWsServer = new WebSocket.Server({ noServer: true })
    childWsServer.on('connection', (ws) => {
      testMessages.push('child-ws-connected')

      ws.on('message', (data) => {
        const message = data.toString()
        testMessages.push(`child-ws-received:${message}`)
        ws.send(`child-echo:${message}`)
      })

      ws.on('close', () => {
        testMessages.push('child-ws-disconnected')
      })

      ws.send('child-welcome')
    })

    ctx.childApiServer.on('upgrade', (request, socket, head) => {
      if (request.url?.startsWith('/api/v1/ws')) {
        childWsServer.handleUpgrade(request, socket, head, (ws) => {
          childWsServer.emit('connection', ws, request)
        })
      } else {
        socket.destroy()
      }
    })

    // Setup WebSocket proxy upgrade handlers
    ctx.hostUiServer.on('upgrade', async (req, socket, head) => {
      if (req.url?.startsWith('/api')) {
        const { proxyWebSocketUpgrade } = await import('../../../server/proxy.js')
        proxyWebSocketUpgrade(req, socket, head, {
          apiPort: ctx.hostApiPort,
          verbose: false,
        })
      } else {
        socket.destroy()
      }
    })

    ctx.childUiServer.on('upgrade', async (req, socket, head) => {
      if (req.url?.startsWith('/api')) {
        const { proxyWebSocketUpgrade } = await import('../../../server/proxy.js')
        proxyWebSocketUpgrade(req, socket, head, {
          apiPort: ctx.childApiPort,
          verbose: false,
        })
      } else {
        socket.destroy()
      }
    })
  }, 60000)

  afterAll(async () => {
    await new Promise<void>((resolve) => {
      hostWsServer.close(() => {
        childWsServer.close(() => {
          resolve()
        })
      })
    })

    await cleanupTestEnvironment(ctx)
  })

  it('should establish host WebSocket connection', async () => {
    const wsUrl = `ws://127.0.0.1:${ctx.hostUiPort}/api/v1/ws`
    const ws = new WebSocket(wsUrl)
    const messages: string[] = []

    await new Promise<void>((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('Timeout')), 5000)

      ws.on('open', () => {
        ws.send('host-test')
      })

      ws.on('message', (data) => {
        messages.push(data.toString())
        if (messages.length >= 2) {
          clearTimeout(timeout)
          ws.close()
        }
      })

      ws.on('close', () => {
        resolve()
      })

      ws.on('error', (err) => {
        clearTimeout(timeout)
        reject(err)
      })
    })

    expect(messages).toContain('host-welcome')
    expect(messages).toContain('host-echo:host-test')
  }, 30000)

  it('should establish child WebSocket connection', async () => {
    const wsUrl = `ws://127.0.0.1:${ctx.childUiPort}/api/v1/ws`
    const ws = new WebSocket(wsUrl)
    const messages: string[] = []

    await new Promise<void>((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('Timeout')), 5000)

      ws.on('open', () => {
        ws.send('child-test')
      })

      ws.on('message', (data) => {
        messages.push(data.toString())
        if (messages.length >= 2) {
          clearTimeout(timeout)
          ws.close()
        }
      })

      ws.on('close', () => {
        resolve()
      })

      ws.on('error', (err) => {
        clearTimeout(timeout)
        reject(err)
      })
    })

    expect(messages).toContain('child-welcome')
    expect(messages).toContain('child-echo:child-test')
  }, 30000)

  it('should handle simultaneous WebSocket connections without interference', async () => {
    const hostWsUrl = `ws://127.0.0.1:${ctx.hostUiPort}/api/v1/ws`
    const childWsUrl = `ws://127.0.0.1:${ctx.childUiPort}/api/v1/ws`

    const hostWs = new WebSocket(hostWsUrl)
    const childWs = new WebSocket(childWsUrl)

    const hostMessages: string[] = []
    const childMessages: string[] = []

    await new Promise<void>((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('Timeout')), 5000)
      let hostReady = false
      let childReady = false

      const checkDone = () => {
        if (hostReady && childReady) {
          clearTimeout(timeout)
          hostWs.close()
          childWs.close()
          resolve()
        }
      }

      hostWs.on('open', () => hostWs.send('host-concurrent'))
      hostWs.on('message', (data) => {
        hostMessages.push(data.toString())
        if (hostMessages.length >= 2) {
          hostReady = true
          checkDone()
        }
      })

      childWs.on('open', () => childWs.send('child-concurrent'))
      childWs.on('message', (data) => {
        childMessages.push(data.toString())
        if (childMessages.length >= 2) {
          childReady = true
          checkDone()
        }
      })

      hostWs.on('error', (err) => {
        clearTimeout(timeout)
        reject(err)
      })
      childWs.on('error', (err) => {
        clearTimeout(timeout)
        reject(err)
      })
    })

    // Host received its own messages
    expect(hostMessages).toContain('host-welcome')
    expect(hostMessages).toContain('host-echo:host-concurrent')

    // Child received its own messages
    expect(childMessages).toContain('child-welcome')
    expect(childMessages).toContain('child-echo:child-concurrent')

    // No cross-contamination
    expect(hostMessages).not.toContain('child-welcome')
    expect(hostMessages).not.toContain('child-echo:child-concurrent')
    expect(childMessages).not.toContain('host-welcome')
    expect(childMessages).not.toContain('host-echo:host-concurrent')
  }, 30000)

  it('should handle multiple messages on same connection', async () => {
    const wsUrl = `ws://127.0.0.1:${ctx.hostUiPort}/api/v1/ws`
    const ws = new WebSocket(wsUrl)
    const messages: string[] = []

    await new Promise<void>((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('Timeout')), 5000)

      ws.on('open', () => {
        ws.send('message1')
        ws.send('message2')
        ws.send('message3')
      })

      ws.on('message', (data) => {
        messages.push(data.toString())
        // Welcome + 3 echoes = 4 messages
        if (messages.length >= 4) {
          clearTimeout(timeout)
          ws.close()
        }
      })

      ws.on('close', () => {
        resolve()
      })

      ws.on('error', (err) => {
        clearTimeout(timeout)
        reject(err)
      })
    })

    expect(messages).toContain('host-welcome')
    expect(messages).toContain('host-echo:message1')
    expect(messages).toContain('host-echo:message2')
    expect(messages).toContain('host-echo:message3')
  }, 30000)

  it('should handle WebSocket close gracefully', async () => {
    const wsUrl = `ws://127.0.0.1:${ctx.hostUiPort}/api/v1/ws`
    const ws = new WebSocket(wsUrl)
    let closedCleanly = false

    await new Promise<void>((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('Timeout')), 5000)

      ws.on('open', () => {
        // Close immediately after opening
        ws.close()
      })

      ws.on('close', (code, reason) => {
        closedCleanly = true
        clearTimeout(timeout)
        resolve()
      })

      ws.on('error', (err) => {
        clearTimeout(timeout)
        reject(err)
      })
    })

    expect(closedCleanly).toBe(true)
  }, 30000)

  it('should handle large WebSocket messages', async () => {
    const wsUrl = `ws://127.0.0.1:${ctx.hostUiPort}/api/v1/ws`
    const largeMessage = 'x'.repeat(64 * 1024) // 64KB
    const ws = new WebSocket(wsUrl)
    let receivedEcho = false

    await new Promise<void>((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('Timeout')), 10000)

      ws.on('open', () => {
        ws.send(largeMessage)
      })

      ws.on('message', (data) => {
        const message = data.toString()

        if (message.startsWith('host-echo:')) {
          const echoed = message.substring(10)
          expect(echoed).toBe(largeMessage)
          receivedEcho = true
          clearTimeout(timeout)
          ws.close()
        }
      })

      ws.on('close', () => {
        resolve()
      })

      ws.on('error', (err) => {
        clearTimeout(timeout)
        reject(err)
      })
    })

    expect(receivedEcho).toBe(true)
  }, 30000)

  it('should handle multiple concurrent connections to same server', async () => {
    const wsUrl = `ws://127.0.0.1:${ctx.hostUiPort}/api/v1/ws`
    const connections = 5
    const clients: WebSocket[] = []

    // Create multiple connections
    for (let i = 0; i < connections; i++) {
      const ws = new WebSocket(wsUrl)
      clients.push(ws)

      await new Promise<void>((resolve, reject) => {
        const timeout = setTimeout(() => reject(new Error(`Connection ${i} timeout`)), 5000)

        ws.on('open', () => {
          clearTimeout(timeout)
          resolve()
        })

        ws.on('error', (err) => {
          clearTimeout(timeout)
          reject(err)
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

    // Wait for all to close
    await new Promise((resolve) => setTimeout(resolve, 500))
  }, 30000)

  it('should preserve message order', async () => {
    const wsUrl = `ws://127.0.0.1:${ctx.hostUiPort}/api/v1/ws`
    const ws = new WebSocket(wsUrl)
    const messages: string[] = []
    const sentMessages = ['first', 'second', 'third', 'fourth', 'fifth']

    await new Promise<void>((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('Timeout')), 5000)

      ws.on('open', () => {
        // Send all messages
        sentMessages.forEach((msg) => ws.send(msg))
      })

      ws.on('message', (data) => {
        const message = data.toString()
        if (message !== 'host-welcome') {
          messages.push(message)
        }

        // Wait for all echoes
        if (messages.length >= sentMessages.length) {
          clearTimeout(timeout)
          ws.close()
        }
      })

      ws.on('close', () => {
        resolve()
      })

      ws.on('error', (err) => {
        clearTimeout(timeout)
        reject(err)
      })
    })

    // Verify order is preserved
    expect(messages).toEqual([
      'host-echo:first',
      'host-echo:second',
      'host-echo:third',
      'host-echo:fourth',
      'host-echo:fifth',
    ])
  }, 30000)
})
