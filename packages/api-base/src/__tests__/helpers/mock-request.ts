/**
 * Mock HTTP request/response utilities for testing
 */

import type { Request, Response } from 'express'
import type { IncomingHttpHeaders } from 'node:http'

export interface MockRequestOptions {
  method?: string
  url?: string
  path?: string
  headers?: IncomingHttpHeaders
  body?: any
  query?: Record<string, any>
  params?: Record<string, any>
}

/**
 * Create a mock Express request
 */
export function mockRequest(options: MockRequestOptions = {}): any {
  const {
    method = 'GET',
    url = '/',
    path: urlPath = url,
    headers = {},
    body = {},
    query = {},
    params = {},
  } = options

  return {
    method,
    url,
    path: urlPath,
    headers,
    body,
    query,
    params,
    get: (name: string) => headers[name.toLowerCase()],
    pipe: () => {
      // Mock pipe for streaming requests
      return {
        on: () => {},
        end: () => {},
      }
    },
  }
}

/**
 * Create a mock Express response with tracking
 */
export function mockResponse(): {
  res: any
  getStatus: () => number
  getHeaders: () => Record<string, any>
  getBody: () => string
  getJson: () => any
  getResult: () => { status: number; headers: Record<string, any>; body: string; json: any }
} {
  let status = 200
  let body = ''
  const headers: Record<string, any> = {}
  let jsonData: any = null

  const res: any = {
    status(code: number) {
      status = code
      return this as Response
    },
    setHeader(name: string, value: any) {
      headers[name.toLowerCase()] = value
      return this as Response
    },
    header(name: string, value: any) {
      headers[name.toLowerCase()] = value
      return this as Response
    },
    getHeader(name: string) {
      return headers[name.toLowerCase()]
    },
    removeHeader(name: string) {
      delete headers[name.toLowerCase()]
      return this as Response
    },
    write(chunk: any) {
      body += String(chunk)
      return true
    },
    end(chunk?: any) {
      if (chunk !== undefined) {
        body += String(chunk)
      }
      return this as Response
    },
    json(data: any) {
      jsonData = data
      headers['content-type'] = 'application/json'
      body = JSON.stringify(data)
      return this as Response
    },
    send(data: any) {
      body = typeof data === 'string' ? data : JSON.stringify(data)
      return this as Response
    },
    sendStatus(code: number) {
      status = code
      return this as Response
    },
  }

  return {
    res,
    getStatus: () => status,
    getHeaders: () => ({ ...headers }),
    getBody: () => body,
    getJson: () => jsonData,
    getResult: () => ({ status, headers: { ...headers }, body, json: jsonData }),
  }
}
