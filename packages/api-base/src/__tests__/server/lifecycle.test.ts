import { describe, expect, it, vi } from 'vitest'

describe('assertLifecycleManagedUI', () => {
  it('throws outside lifecycle management', async () => {
    vi.resetModules()
    const originalLifecycle = process.env.VROOLI_LIFECYCLE_MANAGED
    const originalNodeEnv = process.env.NODE_ENV
    const originalVitest = process.env.VITEST
    delete process.env.VROOLI_LIFECYCLE_MANAGED
    delete process.env.VITEST
    process.env.NODE_ENV = 'development'

    try {
      const { assertLifecycleManagedUI } = await import('../../server/lifecycle.js')
      expect(() => assertLifecycleManagedUI({ serviceName: 'demo-ui' })).toThrow(/vrooli scenario start demo-ui/)
    } finally {
      if (originalLifecycle === undefined) {
        delete process.env.VROOLI_LIFECYCLE_MANAGED
      } else {
        process.env.VROOLI_LIFECYCLE_MANAGED = originalLifecycle
      }
      if (originalNodeEnv === undefined) {
        delete process.env.NODE_ENV
      } else {
        process.env.NODE_ENV = originalNodeEnv
      }
      if (originalVitest === undefined) {
        delete process.env.VITEST
      } else {
        process.env.VITEST = originalVitest
      }
    }
  })

  it('allows lifecycle-managed execution', async () => {
    vi.resetModules()
    const originalLifecycle = process.env.VROOLI_LIFECYCLE_MANAGED
    process.env.VROOLI_LIFECYCLE_MANAGED = 'true'

    try {
      const { assertLifecycleManagedUI } = await import('../../server/lifecycle.js')
      expect(() => assertLifecycleManagedUI({ serviceName: 'demo-ui' })).not.toThrow()
    } finally {
      if (originalLifecycle === undefined) {
        delete process.env.VROOLI_LIFECYCLE_MANAGED
      } else {
        process.env.VROOLI_LIFECYCLE_MANAGED = originalLifecycle
      }
    }
  })
})
