/**
 * Tests for proxy context detection
 */

import { describe, it, expect } from 'vitest'
import { isProxyContext, getProxyInfo, getProxyIndex } from '../../client/detect.js'
import { mockWindow, mockLocalhost, mockProxyContext } from '../helpers/mock-window.js'
import { mockProxyInfo } from '../helpers/test-utils.js'

describe('isProxyContext', () => {
  it('returns true when __VROOLI_PROXY_INFO__ is set', () => {
    const win = mockWindow({
      globals: { __VROOLI_PROXY_INFO__: mockProxyInfo() },
    })

    expect(isProxyContext(win)).toBe(true)
  })

  it('returns true when __APP_MONITOR_PROXY_INFO__ is set (legacy)', () => {
    const win = mockWindow({
      globals: { __APP_MONITOR_PROXY_INFO__: mockProxyInfo() },
    })

    expect(isProxyContext(win)).toBe(true)
  })

  it('returns true when pathname contains /apps/ and /proxy/', () => {
    const win = mockWindow({
      pathname: '/apps/scenario-name/proxy/',
    })

    expect(isProxyContext(win)).toBe(true)
  })

  it('returns true when pathname contains /proxy/ (generic)', () => {
    const win = mockWindow({
      pathname: '/embed/custom/proxy/',
    })

    expect(isProxyContext(win)).toBe(true)
  })

  it('returns false for localhost without proxy indicators', () => {
    const win = mockLocalhost()

    expect(isProxyContext(win)).toBe(false)
  })

  it('returns false when window is undefined', () => {
    expect(isProxyContext(undefined)).toBe(false)
  })

  it('supports custom proxy global names', () => {
    const win = mockWindow({
      globals: { __CUSTOM_PROXY__: mockProxyInfo() },
    })

    expect(isProxyContext(win, ['__CUSTOM_PROXY__'])).toBe(true)
  })
})

describe('getProxyInfo', () => {
  it('retrieves proxy info from __VROOLI_PROXY_INFO__', () => {
    const expectedInfo = mockProxyInfo({ appId: 'test-app' })
    const win = mockWindow({
      globals: { __VROOLI_PROXY_INFO__: expectedInfo },
    })

    const result = getProxyInfo(win)

    expect(result).toEqual(expectedInfo)
  })

  it('retrieves proxy info from __APP_MONITOR_PROXY_INFO__ (legacy)', () => {
    const expectedInfo = mockProxyInfo({ appId: 'legacy-app' })
    const win = mockWindow({
      globals: { __APP_MONITOR_PROXY_INFO__: expectedInfo },
    })

    const result = getProxyInfo(win)

    expect(result).toEqual(expectedInfo)
  })

  it('prefers __VROOLI_PROXY_INFO__ over legacy', () => {
    const newInfo = mockProxyInfo({ appId: 'new-app' })
    const legacyInfo = mockProxyInfo({ appId: 'legacy-app' })

    const win = mockWindow({
      globals: {
        __VROOLI_PROXY_INFO__: newInfo,
        __APP_MONITOR_PROXY_INFO__: legacyInfo,
      },
    })

    const result = getProxyInfo(win)

    expect(result?.appId).toBe('new-app')
  })

  it('returns null when no proxy info found', () => {
    const win = mockLocalhost()

    const result = getProxyInfo(win)

    expect(result).toBeNull()
  })

  it('returns null when window is undefined', () => {
    const result = getProxyInfo(undefined)

    expect(result).toBeNull()
  })

  it('supports custom proxy global names', () => {
    const expectedInfo = mockProxyInfo({ appId: 'custom-app' })
    const win = mockWindow({
      globals: { __CUSTOM_PROXY__: expectedInfo },
    })

    const result = getProxyInfo(win, ['__CUSTOM_PROXY__'])

    expect(result).toEqual(expectedInfo)
  })

  it('validates proxy info structure', () => {
    const invalidInfo = { invalid: 'data' }
    const win = mockWindow({
      globals: { __VROOLI_PROXY_INFO__: invalidInfo },
    })

    const result = getProxyInfo(win)

    // Should return null for invalid structure
    expect(result).toBeNull()
  })
})

describe('getProxyIndex', () => {
  it('builds index from proxy info', () => {
    const proxyInfo = mockProxyInfo({
      appId: 'test-app',
      ports: [
        {
          port: 8080,
          label: 'UI',
          normalizedLabel: 'ui',
          slug: 'ui',
          source: 'test',
          isPrimary: true,
          path: '/apps/test/proxy',
          aliases: ['ui', '8080', 'primary'],
        },
        {
          port: 8081,
          label: 'API',
          normalizedLabel: 'api',
          slug: 'api',
          source: 'test',
          isPrimary: false,
          path: '/apps/test/proxy',
          aliases: ['api', '8081'],
        },
      ],
    })

    const win = mockWindow({
      globals: { __VROOLI_PROXY_INFO__: proxyInfo },
    })

    const result = getProxyIndex(win)

    expect(result).not.toBeNull()
    expect(result?.appId).toBe('test-app')
    expect(result?.aliasMap).toBeInstanceOf(Map)
    expect(result?.hosts).toBeInstanceOf(Set)
    expect(result?.aliasMap.has('ui')).toBe(true)
    expect(result?.aliasMap.has('api')).toBe(true)
    expect(result?.aliasMap.has('8080')).toBe(true)
  })

  it('returns null when no proxy info available', () => {
    const win = mockLocalhost()

    const result = getProxyIndex(win)

    expect(result).toBeNull()
  })
})
