/**
 * Testing utilities - unified export.
 *
 * Prefer importing from '@/test' instead of individual modules.
 * Rendering of the 3D world is never unit-tested through mocks; the world
 * smoke tool (scripts/world-smoke) owns that surface.
 */

// =============================================================================
// DEEP UTILITIES
// =============================================================================

export { deepClone, deepEqual } from './deep-utils'

// =============================================================================
// ROUTINE REACT TESTING
// =============================================================================

export { createTestQueryClient } from './query'
export { renderWithProviders } from '@/test-utils/renderWithProviders'

export {
  createTestWrapper,
  renderHookWithProviders,
} from './render'

export {
  createStorageMock,
  installStorageMock,
  resetStorageMock,
} from './storage'

export type {
  StorageMock,
} from './storage'

export {
  setViewport,
  restoreViewport,
} from './viewport'

export {
  installFetchGuard,
  jsonResponse,
} from './network'

export type {
  FetchCall,
  FetchGuard,
} from './network'
