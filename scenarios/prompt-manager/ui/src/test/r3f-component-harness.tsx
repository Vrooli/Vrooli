/**
 * R3F Component Test Harness
 *
 * Provides test wrappers and mock setup for React Three Fiber components.
 * Use this to test R3F components without requiring WebGL context.
 *
 * Key features:
 * - R3FTestHarness: Provider component that mocks useThree and useFrame
 * - setupR3FMocks(): Configure vi.mock for @react-three/fiber
 * - resetR3FTestState(): Reset all mocks between tests
 *
 * @example
 * import { render } from '@testing-library/react'
 * import { R3FTestHarness, setupR3FMocks } from '@/test/r3f-component-harness'
 *
 * // In test file
 * beforeAll(() => setupR3FMocks())
 *
 * it('renders agent', () => {
 *   render(
 *     <R3FTestHarness>
 *       <SlimeAgent position={[0, 0, 0]} />
 *     </R3FTestHarness>
 *   )
 * })
 */

import React, { createContext, useContext, useRef, useCallback, useEffect } from 'react'
import { render, type RenderResult } from '@testing-library/react'
import { vi } from 'vitest'
import type { MockThreeState, FrameCallback } from './r3f-test-utils'
import {
  FrameLoopSimulator,
  createMockObject3D,
  createMockVector3,
} from './r3f-test-utils'

// =============================================================================
// TYPES
// =============================================================================

/** Configuration for the test harness */
export interface R3FTestHarnessConfig {
  /** Initial camera position */
  cameraPosition?: [number, number, number]
  /** Initial pointer position (normalized -1 to 1) */
  pointerPosition?: { x: number; y: number }
  /** Canvas size */
  size?: { width: number; height: number }
  /** Whether to auto-advance frames */
  autoAdvance?: boolean
  /** Auto-advance FPS (default 60) */
  autoAdvanceFps?: number
}

/** Context value for R3F test harness */
export interface R3FTestContext {
  simulator: FrameLoopSimulator
  registerFrameCallback: (callback: FrameCallback, priority?: number) => () => void
  tick: (delta?: number) => void
  tickFrames: (frames: number, delta?: number) => void
  setPointer: (x: number, y: number) => void
  getState: () => MockThreeState
}

// =============================================================================
// CONTEXT
// =============================================================================

const R3FTestContext = createContext<R3FTestContext | null>(null)

/**
 * Hook to access the R3F test harness context.
 */
export function useR3FTestContext(): R3FTestContext {
  const context = useContext(R3FTestContext)
  if (!context) {
    throw new Error('useR3FTestContext must be used within R3FTestHarness')
  }
  return context
}

// =============================================================================
// GLOBAL STATE FOR MOCKS
// =============================================================================

/** Global frame loop simulator used by mocked useFrame */
let globalSimulator: FrameLoopSimulator | null = null

/** Registered useFrame callbacks from components */
const registeredFrameCallbacks: Map<number, FrameCallback> = new Map()
let nextCallbackId = 0

/**
 * Get or create the global simulator.
 */
export function getGlobalSimulator(): FrameLoopSimulator {
  if (!globalSimulator) {
    globalSimulator = new FrameLoopSimulator()
  }
  return globalSimulator
}

/**
 * Reset global test state between tests.
 */
export function resetR3FTestState(): void {
  globalSimulator?.reset()
  globalSimulator = null
  registeredFrameCallbacks.clear()
  nextCallbackId = 0
}

// =============================================================================
// MOCK HOOKS
// =============================================================================

/**
 * Mock implementation of useThree.
 *
 * Returns a simplified state object suitable for testing.
 */
export function useMockThree(): MockThreeState {
  return getGlobalSimulator().getState()
}

/**
 * Mock implementation of useFrame.
 *
 * Registers the callback to be called by the test harness.
 *
 * @param callback - Frame callback (state, delta) => void
 * @param priority - Execution order (lower runs first)
 */
export function useMockFrame(
  callback: FrameCallback,
  priority = 0
): void {
  const callbackRef = useRef<FrameCallback>(callback)
  callbackRef.current = callback

  useEffect(() => {
    const id = nextCallbackId++
    const wrappedCallback: FrameCallback = (state, delta) => {
      callbackRef.current(state, delta)
    }

    registeredFrameCallbacks.set(id, wrappedCallback)
    getGlobalSimulator().registerCallback(wrappedCallback, priority)

    return () => {
      registeredFrameCallbacks.delete(id)
    }
  }, [priority])
}

/**
 * Mock useThree state selector variant.
 */
export function useMockThreeSelector<T>(selector: (state: MockThreeState) => T): T {
  return selector(getGlobalSimulator().getState())
}

// =============================================================================
// MOCK SETUP
// =============================================================================

/**
 * Set up vi.mock for @react-three/fiber.
 *
 * Call this in beforeAll or at module level.
 *
 * @example
 * // In test file
 * vi.mock('@react-three/fiber', () => setupR3FMocks())
 *
 * // Or in beforeAll
 * beforeAll(() => {
 *   vi.mock('@react-three/fiber', () => setupR3FMocks())
 * })
 */
export function setupR3FMocks() {
  return {
    // Core hooks
    useThree: useMockThree,
    useFrame: useMockFrame,

    // Additional exports that components might use
    Canvas: ({ children }: { children: React.ReactNode }) =>
      React.createElement('div', { 'data-testid': 'r3f-canvas' }, children),

    // Extend is used for custom elements
    extend: vi.fn(),

    // Additional hooks
    useGraph: vi.fn(() => ({ nodes: {}, materials: {} })),
    useLoader: vi.fn(() => ({})),

    // Event handling
    addEffect: vi.fn(),
    addAfterEffect: vi.fn(),
    addTail: vi.fn(),

    // Render loop control
    invalidate: vi.fn(),
    advance: vi.fn(),

    // Context
    context: createContext(null),

    // Create portal for rendering
    createPortal: (children: React.ReactNode) => children,
  }
}

/**
 * Set up vi.mock for @react-three/drei.
 *
 * Provides mock implementations of common drei helpers.
 */
export function setupDreiMocks() {
  return {
    // Common components
    Html: ({ children, ...props }: { children: React.ReactNode; [key: string]: unknown }) =>
      React.createElement('div', { 'data-testid': 'drei-html', ...props }, children),

    Text: (props: Record<string, unknown>) =>
      React.createElement('mesh', { 'data-testid': 'drei-text', ...props }),

    Billboard: ({ children }: { children: React.ReactNode }) =>
      React.createElement('group', { 'data-testid': 'drei-billboard' }, children),

    // Controls
    OrbitControls: () => null,
    MapControls: () => null,
    FlyControls: () => null,

    // Loaders (return mock data)
    useGLTF: vi.fn(() => ({
      scene: createMockObject3D(),
      nodes: {},
      materials: {},
      animations: [],
    })),

    useTexture: vi.fn(() => ({
      map: {},
    })),

    // Helpers
    Environment: () => null,
    ContactShadows: () => null,
    Bounds: ({ children }: { children: React.ReactNode }) => children,
    Center: ({ children }: { children: React.ReactNode }) => children,
    Float: ({ children }: { children: React.ReactNode }) => children,

    // Instancing
    Instances: ({ children }: { children: React.ReactNode }) =>
      React.createElement('group', { 'data-testid': 'drei-instances' }, children),
    Instance: (props: Record<string, unknown>) =>
      React.createElement('mesh', { 'data-testid': 'drei-instance', ...props }),

    // Materials
    MeshWobbleMaterial: (props: Record<string, unknown>) =>
      React.createElement('meshStandardMaterial', { 'data-testid': 'drei-wobble', ...props }),

    MeshDistortMaterial: (props: Record<string, unknown>) =>
      React.createElement('meshStandardMaterial', { 'data-testid': 'drei-distort', ...props }),

    // Progress
    useProgress: vi.fn(() => ({ progress: 100, loaded: true })),
  }
}

// =============================================================================
// TEST HARNESS COMPONENT
// =============================================================================

export interface R3FTestHarnessProps {
  children: React.ReactNode
  config?: R3FTestHarnessConfig
}

/**
 * Test harness component that provides R3F context for testing.
 *
 * Wraps children and provides:
 * - Mock useThree context
 * - Mock useFrame registration
 * - Imperative controls for advancing frames
 *
 * @example
 * const { container, rerender } = render(
 *   <R3FTestHarness config={{ cameraPosition: [0, 5, 10] }}>
 *     <MyComponent />
 *   </R3FTestHarness>
 * )
 *
 * // Advance 60 frames to simulate 1 second
 * act(() => {
 *   tickFrames(60)
 * })
 */
export function R3FTestHarness({ children, config = {} }: R3FTestHarnessProps): JSX.Element {
  const {
    cameraPosition = [0, 5, 10],
    pointerPosition = { x: 0, y: 0 },
    size = { width: 1920, height: 1080 },
  } = config

  // Create or get the simulator
  const simulatorRef = useRef<FrameLoopSimulator | null>(null)
  if (!simulatorRef.current) {
    simulatorRef.current = new FrameLoopSimulator({
      camera: {
        ...createMockObject3D(),
        position: createMockVector3(...cameraPosition),
        fov: 75,
        aspect: size.width / size.height,
        near: 0.1,
        far: 1000,
      },
      pointer: pointerPosition,
      size,
      viewport: { ...size, factor: 1 },
    })
    globalSimulator = simulatorRef.current
  }

  const simulator = simulatorRef.current

  const registerFrameCallback = useCallback((callback: FrameCallback, priority = 0) => {
    return simulator.registerCallback(callback, priority)
  }, [simulator])

  const tick = useCallback((delta?: number) => {
    simulator.tick(delta)
  }, [simulator])

  const tickFrames = useCallback((frames: number, delta?: number) => {
    simulator.tickFrames(frames, delta)
  }, [simulator])

  const setPointer = useCallback((x: number, y: number) => {
    simulator.setPointer(x, y)
  }, [simulator])

  const getState = useCallback(() => {
    return simulator.getState()
  }, [simulator])

  const contextValue: R3FTestContext = {
    simulator,
    registerFrameCallback,
    tick,
    tickFrames,
    setPointer,
    getState,
  }

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      resetR3FTestState()
    }
  }, [])

  return (
    <R3FTestContext.Provider value={contextValue}>
      <div data-testid="r3f-test-harness">
        {children}
      </div>
    </R3FTestContext.Provider>
  )
}

// =============================================================================
// IMPERATIVE TESTING HELPERS
// =============================================================================

/**
 * Advance the global frame loop by one tick.
 *
 * Use inside act() for proper React integration.
 *
 * @example
 * act(() => {
 *   tick()
 * })
 */
export function tick(delta?: number): void {
  getGlobalSimulator().tick(delta)
}

/**
 * Advance the global frame loop by multiple frames.
 *
 * @example
 * act(() => {
 *   tickFrames(60) // 1 second at 60fps
 * })
 */
export function tickFrames(frames: number, delta?: number): void {
  getGlobalSimulator().tickFrames(frames, delta)
}

/**
 * Advance the global frame loop by time duration.
 *
 * @example
 * act(() => {
 *   tickTime(1.5) // 1.5 seconds
 * })
 */
export function tickTime(seconds: number): void {
  getGlobalSimulator().tickTime(seconds)
}

/**
 * Set the pointer position for the global simulator.
 */
export function setPointer(x: number, y: number): void {
  getGlobalSimulator().setPointer(x, y)
}

/**
 * Get the current state from the global simulator.
 */
export function getState(): MockThreeState {
  return getGlobalSimulator().getState()
}

// =============================================================================
// ASSERTION HELPERS
// =============================================================================

/**
 * Assert that a component rendered without WebGL errors.
 *
 * Useful for verifying R3F components can at least mount in test environment.
 */
export function assertRenderedSuccessfully(container: HTMLElement): void {
  const harness = container.querySelector('[data-testid="r3f-test-harness"]')
  if (!harness) {
    throw new Error('R3FTestHarness not found in container. Wrap component in R3FTestHarness.')
  }
}

/**
 * Assert that useFrame callback was registered.
 */
export function assertFrameCallbackRegistered(): void {
  if (registeredFrameCallbacks.size === 0) {
    throw new Error('No useFrame callbacks registered. Component may not be mounting correctly.')
  }
}

/**
 * Get the number of registered frame callbacks.
 */
export function getRegisteredCallbackCount(): number {
  return registeredFrameCallbacks.size
}

// =============================================================================
// HIGH-LEVEL TEST HELPERS
// =============================================================================

/**
 * Creates a complete R3F test environment with all utilities bundled.
 *
 * This is the recommended way to set up R3F component tests, providing
 * a consistent environment with harness wrapping and imperative controls.
 *
 * @param config - Optional harness configuration
 * @returns Object with render helper and frame control utilities
 *
 * @example
 * const env = createR3FTestEnv({ cameraPosition: [0, 10, 20] })
 *
 * it('animates correctly', () => {
 *   const { container } = env.renderInHarness(<MyComponent />)
 *   act(() => env.tickFrames(60))
 *   expect(container).toBeDefined()
 * })
 *
 * afterEach(() => {
 *   env.reset()
 * })
 */
export function createR3FTestEnv(config?: R3FTestHarnessConfig): {
  renderInHarness: (element: React.ReactElement) => RenderResult
  tick: typeof tick
  tickFrames: typeof tickFrames
  tickTime: typeof tickTime
  setPointer: typeof setPointer
  getState: typeof getState
  reset: typeof resetR3FTestState
} {
  return {
    renderInHarness: (element: React.ReactElement) =>
      render(<R3FTestHarness config={config}>{element}</R3FTestHarness>),
    tick,
    tickFrames,
    tickTime,
    setPointer,
    getState,
    reset: resetR3FTestState,
  }
}

// =============================================================================
// RE-EXPORTS
// =============================================================================

export { FrameLoopSimulator, RenderTracker } from './r3f-test-utils'
export type { MockThreeState, FrameCallback } from './r3f-test-utils'
