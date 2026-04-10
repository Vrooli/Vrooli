import { afterEach, vi } from 'vitest'

afterEach(() => {
  vi.clearAllMocks()
  vi.restoreAllMocks()
})

if (typeof globalThis.ResizeObserver === 'undefined') {
  globalThis.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
}
