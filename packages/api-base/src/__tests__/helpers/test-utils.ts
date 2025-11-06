/**
 * General test utilities
 */

import type { PortEntry, ProxyInfo } from '../../shared/types.js'

/**
 * Create a mock PortEntry for testing
 */
export function mockPortEntry(overrides: Partial<PortEntry> = {}): PortEntry {
  return {
    port: 8080,
    label: 'API',
    normalizedLabel: 'api',
    slug: 'api',
    source: 'test',
    priority: 10,
    kind: null,
    isPrimary: false,
    path: '/apps/test/proxy',
    aliases: ['api', '8080'],
    ...overrides,
  }
}

/**
 * Create a mock ProxyInfo for testing
 */
export function mockProxyInfo(overrides: Partial<ProxyInfo> = {}): ProxyInfo {
  const primaryPort = mockPortEntry({ isPrimary: true, ...overrides.primary })

  return {
    appId: 'test-scenario',
    hostScenario: 'test-host',
    targetScenario: 'test-target',
    generatedAt: Date.now(),
    hosts: ['localhost', '127.0.0.1'],
    primary: primaryPort,
    ports: [primaryPort],
    basePath: '/apps/test-scenario/proxy',
    ...overrides,
  }
}

/**
 * Wait for a promise to resolve or reject
 */
export function waitFor<T>(
  fn: () => T | Promise<T>,
  timeout = 1000,
  interval = 50
): Promise<T> {
  return new Promise((resolve, reject) => {
    const start = Date.now()

    const check = async () => {
      try {
        const result = await fn()
        if (result) {
          resolve(result)
          return
        }
      } catch (error) {
        // Continue waiting
      }

      if (Date.now() - start > timeout) {
        reject(new Error(`Timeout after ${timeout}ms`))
        return
      }

      setTimeout(check, interval)
    }

    check()
  })
}

/**
 * Sleep for a specified duration
 */
export function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms))
}

/**
 * Spy on console methods
 */
export function spyOnConsole() {
  const logs: string[] = []
  const warnings: string[] = []
  const errors: string[] = []

  const originalLog = console.log
  const originalWarn = console.warn
  const originalError = console.error

  console.log = (...args: any[]) => {
    logs.push(args.map(String).join(' '))
  }

  console.warn = (...args: any[]) => {
    warnings.push(args.map(String).join(' '))
  }

  console.error = (...args: any[]) => {
    errors.push(args.map(String).join(' '))
  }

  return {
    logs,
    warnings,
    errors,
    restore: () => {
      console.log = originalLog
      console.warn = originalWarn
      console.error = originalError
    },
  }
}
