/**
 * Tests for resolveApiBase on REMOTE hosts
 *
 * CRITICAL: These tests caught the bug where /apps/{slug}/preview was incorrectly
 * being treated as a proxy context on remote hosts.
 */

import { describe, it, expect } from 'vitest'
import { resolveApiBase } from '../../client/resolve.js'

describe('resolveApiBase - Remote Host Scenarios', () => {
  /**
   * BUG FIX TEST: This test reproduces the exact bug reported
   *
   * On remote host at /apps/scenario-auditor/preview:
   * - OLD: Incorrectly returned https://app-monitor.itsagitime.com/apps/scenario-auditor/proxy/api/v1
   * - NEW: Correctly returns https://app-monitor.itsagitime.com/api/v1
   */
  it('should NOT treat /apps/:id/preview as proxy context on remote host', () => {
    const mockWindow = {
      location: {
        origin: 'https://app-monitor.itsagitime.com',
        hostname: 'app-monitor.itsagitime.com',
        pathname: '/apps/scenario-auditor/preview',
        protocol: 'https:',
        href: 'https://app-monitor.itsagitime.com/apps/scenario-auditor/preview',
      },
    }

    const resolved = resolveApiBase({
      appendSuffix: true,
      windowObject: mockWindow as any,
    })

    // Should resolve to origin, NOT a proxy path
    expect(resolved).toBe('https://app-monitor.itsagitime.com/api/v1')
    expect(resolved).not.toContain('/proxy')
    expect(resolved).not.toContain('/apps/scenario-auditor/')
  })

  it('should treat /apps/:id/proxy/* as proxy context on remote host', () => {
    const mockWindow = {
      location: {
        origin: 'https://app-monitor.itsagitime.com',
        hostname: 'app-monitor.itsagitime.com',
        pathname: '/apps/scenario-auditor/proxy/index.html',
        protocol: 'https:',
        href: 'https://app-monitor.itsagitime.com/apps/scenario-auditor/proxy/index.html',
      },
    }

    const resolved = resolveApiBase({
      appendSuffix: true,
      windowObject: mockWindow as any,
    })

    // Should resolve to proxy path
    expect(resolved).toBe('https://app-monitor.itsagitime.com/apps/scenario-auditor/proxy/api/v1')
    expect(resolved).toContain('/proxy')
  })

  it('should handle /apps/:id/settings/profile on remote host', () => {
    const mockWindow = {
      location: {
        origin: 'https://app-monitor.itsagitime.com',
        hostname: 'app-monitor.itsagitime.com',
        pathname: '/apps/scenario-auditor/settings/profile',
        protocol: 'https:',
        href: 'https://app-monitor.itsagitime.com/apps/scenario-auditor/settings/profile',
      },
    }

    const resolved = resolveApiBase({
      appendSuffix: true,
      windowObject: mockWindow as any,
    })

    // Should resolve to origin, not proxy
    expect(resolved).toBe('https://app-monitor.itsagitime.com/api/v1')
    expect(resolved).not.toContain('/proxy')
  })

  it('should handle root /apps path on remote host', () => {
    const mockWindow = {
      location: {
        origin: 'https://app-monitor.itsagitime.com',
        hostname: 'app-monitor.itsagitime.com',
        pathname: '/apps',
        protocol: 'https:',
        href: 'https://app-monitor.itsagitime.com/apps',
      },
    }

    const resolved = resolveApiBase({
      appendSuffix: true,
      windowObject: mockWindow as any,
    })

    // Should resolve to origin
    expect(resolved).toBe('https://app-monitor.itsagitime.com/api/v1')
    expect(resolved).not.toContain('/proxy')
  })

  it('should handle /apps/:id/proxy without trailing content', () => {
    const mockWindow = {
      location: {
        origin: 'https://app-monitor.itsagitime.com',
        hostname: 'app-monitor.itsagitime.com',
        pathname: '/apps/scenario-auditor/proxy',
        protocol: 'https:',
        href: 'https://app-monitor.itsagitime.com/apps/scenario-auditor/proxy',
      },
    }

    const resolved = resolveApiBase({
      appendSuffix: true,
      windowObject: mockWindow as any,
    })

    // Should resolve to proxy path
    expect(resolved).toBe('https://app-monitor.itsagitime.com/apps/scenario-auditor/proxy/api/v1')
  })

  it('should differentiate between host and proxy routes on remote', () => {
    // Host route
    const hostWindow = {
      location: {
        origin: 'https://app-monitor.itsagitime.com',
        hostname: 'app-monitor.itsagitime.com',
        pathname: '/apps/scenario-auditor/preview',
        protocol: 'https:',
        href: 'https://app-monitor.itsagitime.com/apps/scenario-auditor/preview',
      },
    }

    const hostResolved = resolveApiBase({
      appendSuffix: true,
      windowObject: hostWindow as any,
    })

    // Proxy route
    const proxyWindow = {
      location: {
        origin: 'https://app-monitor.itsagitime.com',
        hostname: 'app-monitor.itsagitime.com',
        pathname: '/apps/scenario-auditor/proxy/index.html',
        protocol: 'https:',
        href: 'https://app-monitor.itsagitime.com/apps/scenario-auditor/proxy/index.html',
      },
    }

    const proxyResolved = resolveApiBase({
      appendSuffix: true,
      windowObject: proxyWindow as any,
    })

    // They should be different
    expect(hostResolved).toBe('https://app-monitor.itsagitime.com/api/v1')
    expect(proxyResolved).toBe('https://app-monitor.itsagitime.com/apps/scenario-auditor/proxy/api/v1')
    expect(hostResolved).not.toBe(proxyResolved)
  })

  it('should work on localhost (remote=false) without triggering app shell pattern', () => {
    const mockWindow = {
      location: {
        origin: 'http://localhost:3000',
        hostname: 'localhost',
        pathname: '/apps/scenario-auditor/preview',
        protocol: 'http:',
        href: 'http://localhost:3000/apps/scenario-auditor/preview',
      },
    }

    const resolved = resolveApiBase({
      appendSuffix: true,
      windowObject: mockWindow as any,
    })

    // On localhost, app shell pattern shouldn't apply at all
    expect(resolved).toBe('http://localhost:3000/api/v1')
    expect(resolved).not.toContain('/proxy')
  })
})
