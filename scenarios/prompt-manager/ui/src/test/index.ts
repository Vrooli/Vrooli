/**
 * R3F Testing Utilities - Unified Export
 *
 * This barrel export provides a single import point for all R3F testing utilities.
 * Prefer importing from '@/test' instead of individual modules.
 *
 * Quick Start:
 * ```typescript
 * import { setupR3FMocks, tickFrames, createMockMesh } from '@/test'
 *
 * vi.mock('@react-three/fiber', () => setupR3FMocks())
 *
 * it('renders and animates', () => {
 *   render(<R3FTestHarness><MyComponent /></R3FTestHarness>)
 *   act(() => tickFrames(60))
 * })
 * ```
 *
 * Modules:
 * - deep-utils: Deep clone/equality for snapshots
 * - r3f-test-utils: Mock Three.js objects, frame simulation, pointer events
 * - r3f-component-harness: React wrappers, R3F hook mocks, imperative controls
 * - r3f-store-test-utils: Zustand snapshot/diff, subscription tracking
 */

// =============================================================================
// DEEP UTILITIES
// =============================================================================

export { deepClone, deepEqual } from './deep-utils'

// =============================================================================
// ROUTINE REACT TESTING
// =============================================================================

export {
  createTestQueryClient,
} from './query'

export {
  createTestWrapper,
  renderWithProviders,
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

// =============================================================================
// THREE.JS MOCK PRIMITIVES (from r3f-test-utils)
// =============================================================================

export {
  // Mock constructors
  createMockVector3,
  createMockEuler,
  createMockObject3D,
  createMockMesh,
  createMockGroup,
  createMockMaterial,
  createMockGeometry,
  // Pointer event simulation
  createMockPointerEvent,
  simulateDragSequence,
  simulateHoverSequence,
  // Ref tracking
  createTrackedRef,
  RefMutationTracker,
  // Frame simulation
  FrameLoopSimulator,
  // Render tracking
  RenderTracker,
} from './r3f-test-utils'

export type {
  MockVector3,
  MockEuler,
  MockObject3D,
  MockMesh,
  MockGroup,
  MockMaterial,
  MockGeometry,
  MockThreeState,
  MockR3FPointerEvent,
  FrameCallback,
  R3FPointerEventType,
} from './r3f-test-utils'

// =============================================================================
// R3F COMPONENT TESTING (from r3f-component-harness)
// =============================================================================

export {
  // Test harness
  R3FTestHarness,
  useR3FTestContext,
  createR3FTestEnv,
  // Mock setup factories
  setupR3FMocks,
  setupDreiMocks,
  // Mock hooks
  useMockThree,
  useMockFrame,
  useMockThreeSelector,
  // Imperative frame control
  tick,
  tickFrames,
  tickTime,
  setPointer,
  getState,
  // State management
  resetR3FTestState,
  getGlobalSimulator,
  installR3FDOMWarningFilter,
  // Assertions
  assertRenderedSuccessfully,
  assertFrameCallbackRegistered,
  getRegisteredCallbackCount,
} from './r3f-component-harness'

export type {
  R3FTestHarnessConfig,
  R3FTestContext,
  R3FTestHarnessProps,
} from './r3f-component-harness'

// =============================================================================
// ZUSTAND STORE TESTING (from r3f-store-test-utils)
// =============================================================================

export {
  // Snapshots
  takeStoreSnapshot,
  diffSnapshots,
  assertOnlyFieldsChanged,
  // Subscription tracking
  StoreSubscriptionTracker,
  // Mutation tracking
  MutationTracker,
  // Code validation
  validateGetStateUsage,
  createStoreUsageSpy,
  // Store helpers
  createTestStore,
  waitForStoreState,
  mockStoreActions,
} from './r3f-store-test-utils'

export type {
  AnyStore,
  StoreSnapshot,
  SnapshotDiff,
  SubscriptionEvent,
} from './r3f-store-test-utils'
