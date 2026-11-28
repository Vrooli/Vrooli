/**
 * Streams through proxyToApi, against a real upstream and a real Express server.
 *
 * A Connect job watch (or an event stream) is silent between frames for as long as
 * its work runs. The proxy's idle timeout must not end it, an upstream that dies
 * mid-stream must close the client's response rather than leave it hanging, and a
 * client that leaves must stop the upstream work. Unary calls keep their timeout.
 */

import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import express from 'express'
import * as http from 'node:http'
import type { AddressInfo } from 'node:net'
import { isStreamingRequest, isStreamingResponse, proxyToApi } from '../../server/proxy.js'
import { resetProxyAgentsForTesting } from '../../server/agent.js'

// Shorter than every pause below, so a proxy that still applied it would end each case.
const IDLE_LIMIT_MS = 150
const PAUSE_MS = 400

type Upstream = (req: http.IncomingMessage, res: http.ServerResponse) => void

let upstream: http.Server
let front: http.Server
let frontPort: number
let upstreamClosedConnections: number

const listen = (server: http.Server) =>
  new Promise<number>((resolve) => server.listen(0, '127.0.0.1', () => resolve((server.address() as AddressInfo).port)))

const closeServer = (server: http.Server | undefined) =>
  new Promise<void>((resolve) => {
    if (!server?.listening) return resolve()
    server.closeAllConnections?.()
    server.close(() => resolve())
  })

async function startProxy(handler: Upstream) {
  upstream = http.createServer((req, res) => {
    res.on('close', () => {
      if (!res.writableFinished) upstreamClosedConnections++
    })
    handler(req, res)
  })
  const apiPort = await listen(upstream)
  const app = express()
  app.use((req, res) => {
    void proxyToApi(req, res, req.originalUrl, { apiPort, timeout: IDLE_LIMIT_MS, keepAlive: false })
  })
  front = http.createServer(app)
  frontPort = await listen(front)
}

interface Outcome {
  status: number
  body: string
  /** The response ended cleanly (it was not cut off). */
  complete: boolean
  elapsedMs: number
}

function call(headers: Record<string, string>, body = ''): Promise<Outcome> {
  const started = Date.now()
  return new Promise((resolve, reject) => {
    const req = http.request(
      { host: '127.0.0.1', port: frontPort, method: 'POST', path: '/watch', headers, agent: false },
      (res) => {
        let text = ''
        res.on('data', (chunk) => (text += chunk))
        res.on('end', () => resolve({ status: res.statusCode ?? 0, body: text, complete: res.complete, elapsedMs: Date.now() - started }))
        res.on('error', () => resolve({ status: res.statusCode ?? 0, body: text, complete: false, elapsedMs: Date.now() - started }))
        res.on('close', () => resolve({ status: res.statusCode ?? 0, body: text, complete: res.complete, elapsedMs: Date.now() - started }))
      },
    )
    req.on('error', reject)
    req.end(body)
  })
}

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

beforeEach(() => {
  upstreamClosedConnections = 0
})

afterEach(async () => {
  await closeServer(front)
  await closeServer(upstream)
  resetProxyAgentsForTesting()
})

describe('proxyToApi with streams', () => {
  it('keeps a Connect stream open through a silence longer than the idle timeout', async () => {
    await startProxy(async (_req, res) => {
      res.writeHead(200, { 'content-type': 'application/connect+json' })
      res.write('frame-1;')
      await sleep(PAUSE_MS)
      res.write('frame-2;')
      res.end()
    })
    const outcome = await call({ 'content-type': 'application/connect+json' })
    expect(outcome).toMatchObject({ status: 200, body: 'frame-1;frame-2;', complete: true })
  })

  it('keeps an event stream open when the upstream marks the response as a stream', async () => {
    await startProxy(async (_req, res) => {
      res.writeHead(200, { 'content-type': 'text/event-stream' })
      res.write('data: one\n\n')
      await sleep(PAUSE_MS)
      res.write('data: two\n\n')
      res.end()
    })
    // The request carries no streaming hint: only the response says it is a stream.
    const outcome = await call({ 'content-type': 'application/json' }, '{}')
    expect(outcome).toMatchObject({ status: 200, body: 'data: one\n\ndata: two\n\n', complete: true })
  })

  it('still times a unary call out when the upstream stays silent', async () => {
    await startProxy(async (_req, res) => {
      await sleep(PAUSE_MS * 3)
      res.end('late')
    })
    const outcome = await call({ 'content-type': 'application/json' }, '{}')
    expect(outcome.status).toBe(502)
    expect(outcome.body).toContain('timed out')
    expect(outcome.elapsedMs).toBeLessThan(PAUSE_MS * 3)
  })

  it('closes the client response when the upstream dies mid-stream, instead of hanging', async () => {
    await startProxy((_req, res) => {
      res.writeHead(200, { 'content-type': 'application/connect+json' })
      res.write('frame-1;')
      setTimeout(() => res.destroy(), 50)
    })
    const outcome = await call({ 'content-type': 'application/connect+json' })
    expect(outcome.body).toBe('frame-1;')
    expect(outcome.complete).toBe(false)
    expect(outcome.elapsedMs).toBeLessThan(2000)
  })

  it('closes a unary response cut off mid-body, too', async () => {
    await startProxy((_req, res) => {
      res.writeHead(200, { 'content-type': 'application/json', 'content-length': '100' })
      res.write('{"part"')
      setTimeout(() => res.destroy(), 50)
    })
    const outcome = await call({ 'content-type': 'application/json' }, '{}')
    expect(outcome.complete).toBe(false)
    expect(outcome.elapsedMs).toBeLessThan(2000)
  })

  it('stops the upstream work when the client leaves a stream', async () => {
    await startProxy((_req, res) => {
      res.writeHead(200, { 'content-type': 'application/connect+json' })
      res.write('frame-1;')
    })
    await new Promise<void>((resolve, reject) => {
      const req = http.request(
        { host: '127.0.0.1', port: frontPort, method: 'POST', path: '/watch', headers: { 'content-type': 'application/connect+json' }, agent: false },
        (res) => {
          res.once('data', () => {
            req.destroy()
            resolve()
          })
        },
      )
      req.on('error', () => undefined)
      req.end('')
      setTimeout(() => reject(new Error('no frame arrived')), 2000)
    })
    const deadline = Date.now() + 2000
    while (upstreamClosedConnections === 0 && Date.now() < deadline) await sleep(20)
    expect(upstreamClosedConnections).toBe(1)
  })
})

describe('streaming detection', () => {
  it.each([
    [{ 'content-type': 'application/connect+json' }, true],
    [{ 'content-type': 'Application/Connect+Proto; charset=utf-8' }, true],
    [{ 'content-type': 'application/grpc-web+proto' }, true],
    [{ accept: 'text/event-stream' }, true],
    [{ 'content-type': 'application/json' }, false],
    [{ 'content-type': 'application/proto', accept: 'application/json' }, false],
    [{}, false],
  ])('request %j streams: %s', (headers, expected) => {
    expect(isStreamingRequest(headers)).toBe(expected)
  })

  it.each([
    [{ 'content-type': 'text/event-stream; charset=utf-8' }, true],
    [{ 'content-type': 'application/connect+json' }, true],
    [{ 'content-type': 'application/x-ndjson' }, true],
    [{ 'content-type': 'application/json' }, false],
    [{}, false],
  ])('response %j streams: %s', (headers, expected) => {
    expect(isStreamingResponse(headers)).toBe(expected)
  })
})
