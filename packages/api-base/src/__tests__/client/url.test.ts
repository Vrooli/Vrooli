/**
 * Tests for URL building utilities
 */

import { describe, it, expect } from 'vitest'
import { buildApiUrl, buildWsUrl } from '../../client/url.js'
import { mockLocalhost, mockRemoteHost } from '../helpers/mock-window.js'

describe('buildApiUrl', () => {
  it('builds URL with absolute path', () => {
    const result = buildApiUrl('/health', {
      baseUrl: 'http://localhost:8080/api/v1',
    })

    expect(result).toBe('http://localhost:8080/api/v1/health')
  })

  it('builds URL with relative path', () => {
    const result = buildApiUrl('users', {
      baseUrl: 'http://localhost:8080/api/v1',
    })

    expect(result).toBe('http://localhost:8080/api/v1/users')
  })

  it('normalizes double slashes', () => {
    const result = buildApiUrl('/health', {
      baseUrl: 'http://localhost:8080/api/v1/',
    })

    // Should not have double slash
    expect(result).toBe('http://localhost:8080/api/v1/health')
  })

  it('resolves base URL when not provided', () => {
    const result = buildApiUrl('/health', {
      windowObject: mockLocalhost(),
      defaultPort: '8080',
    })

    expect(result).toContain('127.0.0.1:8080')
    expect(result).toContain('/health')
  })

  it('handles query parameters', () => {
    const result = buildApiUrl('/users?limit=10', {
      baseUrl: 'http://localhost:8080/api/v1',
    })

    expect(result).toBe('http://localhost:8080/api/v1/users?limit=10')
  })

  it('handles hash fragments', () => {
    const result = buildApiUrl('/section#top', {
      baseUrl: 'http://localhost:8080/api/v1',
    })

    expect(result).toBe('http://localhost:8080/api/v1/section#top')
  })

  it('works with empty path', () => {
    const result = buildApiUrl('', {
      baseUrl: 'http://localhost:8080/api/v1',
    })

    expect(result).toBe('http://localhost:8080/api/v1')
  })

  it('works with root path', () => {
    const result = buildApiUrl('/', {
      baseUrl: 'http://localhost:8080/api/v1',
    })

    expect(result).toBe('http://localhost:8080/api/v1/')
  })
})

describe('buildWsUrl', () => {
  it('builds WebSocket URL with ws protocol', () => {
    const result = buildWsUrl('/stream', {
      baseUrl: 'ws://localhost:8080/ws',
    })

    expect(result).toBe('ws://localhost:8080/ws/stream')
  })

  it('builds WebSocket URL with wss protocol', () => {
    const result = buildWsUrl('/stream', {
      baseUrl: 'wss://example.com/ws',
    })

    expect(result).toBe('wss://example.com/ws/stream')
  })

  it('resolves base URL when not provided', () => {
    const result = buildWsUrl('/stream', {
      windowObject: mockLocalhost(),
      defaultPort: '8080',
    })

    expect(result).toContain('ws://127.0.0.1:8080')
    expect(result).toContain('/stream')
  })

  it('handles wss for https origins', () => {
    const result = buildWsUrl('/stream', {
      windowObject: mockRemoteHost('example.com'),
    })

    expect(result).toContain('wss://')
  })
})
