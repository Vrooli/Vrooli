/**
 * React Three Fiber Testing Utilities
 *
 * Core testing utilities for R3F components that don't require WebGL context.
 * Provides lightweight Three.js mocks, frame loop simulation, render tracking,
 * and pointer event simulation.
 *
 * @example
 * import {
 *   FrameLoopSimulator,
 *   createMockMesh,
 *   createMockPointerEvent,
 * } from '@/test/r3f-test-utils'
 *
 * const simulator = new FrameLoopSimulator()
 * simulator.registerCallback((state, delta) => { ... })
 * simulator.tickFrames(60) // Simulate 1 second at 60fps
 */

import { vi } from 'vitest'

// =============================================================================
// TYPES
// =============================================================================

/** Lightweight Vector3-like interface for testing */
export interface MockVector3 {
  x: number
  y: number
  z: number
  set: (x: number, y: number, z: number) => MockVector3
  copy: (v: MockVector3) => MockVector3
  clone: () => MockVector3
  add: (v: MockVector3) => MockVector3
  sub: (v: MockVector3) => MockVector3
  multiplyScalar: (s: number) => MockVector3
  normalize: () => MockVector3
  lerp: (v: MockVector3, alpha: number) => MockVector3
  length: () => number
  distanceTo: (v: MockVector3) => number
  toArray: () => [number, number, number]
}

/** Lightweight Euler-like interface for testing */
export interface MockEuler {
  x: number
  y: number
  z: number
  order: string
  set: (x: number, y: number, z: number, order?: string) => MockEuler
  copy: (e: MockEuler) => MockEuler
  clone: () => MockEuler
}

/** Mock Object3D structure for testing */
export interface MockObject3D {
  position: MockVector3
  rotation: MockEuler
  scale: MockVector3
  visible: boolean
  userData: Record<string, unknown>
  children: MockObject3D[]
  parent: MockObject3D | null
  add: (child: MockObject3D) => void
  remove: (child: MockObject3D) => void
  traverse: (callback: (obj: MockObject3D) => void) => void
}

/** Mock Mesh extends Object3D with material/geometry */
export interface MockMesh extends MockObject3D {
  material: MockMaterial | MockMaterial[]
  geometry: MockGeometry
  isMesh: true
}

/** Mock Group */
export interface MockGroup extends MockObject3D {
  isGroup: true
}

/** Mock Material */
export interface MockMaterial {
  color: { r: number; g: number; b: number; setHex: (hex: number) => void }
  opacity: number
  transparent: boolean
  needsUpdate: boolean
  dispose: () => void
  uniforms?: Record<string, { value: unknown }>
}

/** Mock Geometry */
export interface MockGeometry {
  dispose: () => void
  boundingBox: { min: MockVector3; max: MockVector3 } | null
  computeBoundingBox: () => void
}

/** useFrame callback signature */
export type FrameCallback = (
  state: MockThreeState,
  delta: number,
  frame?: XRFrame
) => void

/** Simplified useThree state for testing */
export interface MockThreeState {
  clock: { elapsedTime: number; getDelta: () => number }
  camera: MockObject3D & { fov?: number; aspect?: number; near?: number; far?: number }
  scene: MockObject3D
  gl: { domElement: HTMLCanvasElement }
  size: { width: number; height: number }
  viewport: { width: number; height: number; factor: number }
  pointer: { x: number; y: number }
  raycaster: {
    ray: { origin: MockVector3; direction: MockVector3 }
    setFromCamera: (coords: { x: number; y: number }, camera: unknown) => void
    intersectObjects: (objects: unknown[], recursive?: boolean) => unknown[]
  }
  invalidate: () => void
}

/** Pointer event types for R3F */
export type R3FPointerEventType =
  | 'click'
  | 'contextmenu'
  | 'dblclick'
  | 'pointerup'
  | 'pointerdown'
  | 'pointerover'
  | 'pointerout'
  | 'pointerenter'
  | 'pointerleave'
  | 'pointermove'
  | 'pointermissed'
  | 'wheel'

/** R3F-style pointer event */
export interface MockR3FPointerEvent {
  /** Stop event propagation through scene graph */
  stopPropagation: () => void
  /** Pointer coordinates in normalized device coordinates (-1 to 1) */
  pointer: { x: number; y: number }
  /** Intersection point in world coordinates */
  point: MockVector3
  /** Intersected object */
  object: MockObject3D
  /** Distance from camera to intersection */
  distance: number
  /** Face normal at intersection */
  face?: { normal: MockVector3 }
  /** Original DOM event */
  nativeEvent: PointerEvent | MouseEvent | WheelEvent
  /** Event delta for wheel events */
  delta?: number
  /** Ray used for intersection */
  ray: { origin: MockVector3; direction: MockVector3 }
  /** Camera used for event */
  camera: MockObject3D
  /** Event type */
  eventObject: MockObject3D
  /** UV coordinates at intersection (if available) */
  uv?: { x: number; y: number }
}

// =============================================================================
// MOCK CONSTRUCTORS
// =============================================================================

/**
 * Creates a lightweight mock Vector3 with all essential methods.
 * Avoids Three.js import for faster tests.
 */
export function createMockVector3(x = 0, y = 0, z = 0): MockVector3 {
  const vec: MockVector3 = {
    x,
    y,
    z,
    set(nx, ny, nz) {
      vec.x = nx
      vec.y = ny
      vec.z = nz
      return vec
    },
    copy(v) {
      vec.x = v.x
      vec.y = v.y
      vec.z = v.z
      return vec
    },
    clone() {
      return createMockVector3(vec.x, vec.y, vec.z)
    },
    add(v) {
      vec.x += v.x
      vec.y += v.y
      vec.z += v.z
      return vec
    },
    sub(v) {
      vec.x -= v.x
      vec.y -= v.y
      vec.z -= v.z
      return vec
    },
    multiplyScalar(s) {
      vec.x *= s
      vec.y *= s
      vec.z *= s
      return vec
    },
    normalize() {
      const len = vec.length()
      if (len > 0) {
        vec.x /= len
        vec.y /= len
        vec.z /= len
      }
      return vec
    },
    lerp(v, alpha) {
      vec.x += (v.x - vec.x) * alpha
      vec.y += (v.y - vec.y) * alpha
      vec.z += (v.z - vec.z) * alpha
      return vec
    },
    length() {
      return Math.sqrt(vec.x * vec.x + vec.y * vec.y + vec.z * vec.z)
    },
    distanceTo(v) {
      const dx = vec.x - v.x
      const dy = vec.y - v.y
      const dz = vec.z - v.z
      return Math.sqrt(dx * dx + dy * dy + dz * dz)
    },
    toArray() {
      return [vec.x, vec.y, vec.z]
    },
  }
  return vec
}

/**
 * Creates a lightweight mock Euler with essential methods.
 */
export function createMockEuler(x = 0, y = 0, z = 0, order = 'XYZ'): MockEuler {
  const euler: MockEuler = {
    x,
    y,
    z,
    order,
    set(nx, ny, nz, newOrder) {
      euler.x = nx
      euler.y = ny
      euler.z = nz
      if (newOrder) euler.order = newOrder
      return euler
    },
    copy(e) {
      euler.x = e.x
      euler.y = e.y
      euler.z = e.z
      euler.order = e.order
      return euler
    },
    clone() {
      return createMockEuler(euler.x, euler.y, euler.z, euler.order)
    },
  }
  return euler
}

/**
 * Creates a mock Object3D with position, rotation, scale, and hierarchy.
 */
export function createMockObject3D(): MockObject3D {
  const obj: MockObject3D = {
    position: createMockVector3(),
    rotation: createMockEuler(),
    scale: createMockVector3(1, 1, 1),
    visible: true,
    userData: {},
    children: [],
    parent: null,
    add(child) {
      if (child.parent) {
        child.parent.remove(child)
      }
      child.parent = obj
      obj.children.push(child)
    },
    remove(child) {
      const idx = obj.children.indexOf(child)
      if (idx !== -1) {
        obj.children.splice(idx, 1)
        child.parent = null
      }
    },
    traverse(callback) {
      callback(obj)
      for (const child of obj.children) {
        child.traverse(callback)
      }
    },
  }
  return obj
}

/**
 * Creates a mock Mesh with material and geometry.
 */
export function createMockMesh(): MockMesh {
  return {
    ...createMockObject3D(),
    isMesh: true,
    material: createMockMaterial(),
    geometry: createMockGeometry(),
  }
}

/**
 * Creates a mock Group (Object3D with isGroup flag).
 */
export function createMockGroup(): MockGroup {
  return {
    ...createMockObject3D(),
    isGroup: true,
  }
}

/**
 * Creates a mock Material.
 */
export function createMockMaterial(): MockMaterial {
  return {
    color: {
      r: 1,
      g: 1,
      b: 1,
      setHex: vi.fn(),
    },
    opacity: 1,
    transparent: false,
    needsUpdate: false,
    dispose: vi.fn(),
  }
}

/**
 * Creates a mock Geometry.
 */
export function createMockGeometry(): MockGeometry {
  return {
    dispose: vi.fn(),
    boundingBox: null,
    computeBoundingBox: vi.fn(),
  }
}

// =============================================================================
// FRAME LOOP SIMULATOR
// =============================================================================

/**
 * Simulates the R3F useFrame loop for testing animations.
 *
 * @example
 * const simulator = new FrameLoopSimulator()
 * const meshRef = { current: createMockMesh() }
 *
 * // Register a useFrame-style callback
 * simulator.registerCallback((state, delta) => {
 *   if (meshRef.current) {
 *     meshRef.current.rotation.y += delta
 *   }
 * })
 *
 * // Simulate 60 frames (1 second at 60fps)
 * simulator.tickFrames(60)
 *
 * // Check animation result
 * expect(meshRef.current.rotation.y).toBeCloseTo(1.0, 1)
 */
export class FrameLoopSimulator {
  private callbacks: Map<number, { callback: FrameCallback; priority: number }> = new Map()
  private nextId = 0
  private elapsedTime = 0
  private targetFps = 60
  private state: MockThreeState

  constructor(initialState?: Partial<MockThreeState>) {
    this.state = {
      clock: {
        elapsedTime: 0,
        getDelta: () => 1 / this.targetFps,
      },
      camera: {
        ...createMockObject3D(),
        fov: 75,
        aspect: 16 / 9,
        near: 0.1,
        far: 1000,
      },
      scene: createMockObject3D(),
      gl: { domElement: document.createElement('canvas') },
      size: { width: 1920, height: 1080 },
      viewport: { width: 1920, height: 1080, factor: 1 },
      pointer: { x: 0, y: 0 },
      raycaster: {
        ray: { origin: createMockVector3(), direction: createMockVector3(0, 0, -1) },
        setFromCamera: vi.fn(),
        intersectObjects: vi.fn(() => []),
      },
      invalidate: vi.fn(),
      ...initialState,
    }
  }

  /**
   * Register a useFrame-style callback.
   *
   * @param callback - Function called each frame with (state, delta)
   * @param priority - Execution order (lower runs first, default 0)
   * @returns Unsubscribe function
   */
  registerCallback(callback: FrameCallback, priority = 0): () => void {
    const id = this.nextId++
    this.callbacks.set(id, { callback, priority })
    return () => {
      this.callbacks.delete(id)
    }
  }

  /**
   * Simulate a single frame tick.
   *
   * @param delta - Frame delta time in seconds (default: 1/60)
   */
  tick(delta = 1 / this.targetFps): void {
    this.elapsedTime += delta
    this.state.clock.elapsedTime = this.elapsedTime

    // Sort by priority and execute
    const sorted = [...this.callbacks.values()].sort((a, b) => a.priority - b.priority)
    for (const { callback } of sorted) {
      callback(this.state, delta)
    }
  }

  /**
   * Simulate multiple frame ticks.
   *
   * @param frames - Number of frames to simulate
   * @param delta - Delta per frame (default: 1/60)
   */
  tickFrames(frames: number, delta = 1 / this.targetFps): void {
    for (let i = 0; i < frames; i++) {
      this.tick(delta)
    }
  }

  /**
   * Simulate time passing (calculates frames based on targetFps).
   *
   * @param seconds - Duration in seconds
   */
  tickTime(seconds: number): void {
    const frames = Math.round(seconds * this.targetFps)
    this.tickFrames(frames)
  }

  /**
   * Get the current mock three state.
   */
  getState(): MockThreeState {
    return this.state
  }

  /**
   * Get the total elapsed time.
   */
  getElapsedTime(): number {
    return this.elapsedTime
  }

  /**
   * Set the pointer position (for interaction testing).
   */
  setPointer(x: number, y: number): void {
    this.state.pointer.x = x
    this.state.pointer.y = y
  }

  /**
   * Set the camera position.
   */
  setCameraPosition(x: number, y: number, z: number): void {
    this.state.camera.position.set(x, y, z)
  }

  /**
   * Reset the simulator to initial state.
   */
  reset(): void {
    this.elapsedTime = 0
    this.state.clock.elapsedTime = 0
    this.callbacks.clear()
  }

  /**
   * Set target FPS for delta calculations.
   */
  setTargetFps(fps: number): void {
    this.targetFps = fps
  }
}

// =============================================================================
// RENDER TRACKER
// =============================================================================

/**
 * Tracks component re-renders for performance assertions.
 *
 * @example
 * const tracker = new RenderTracker()
 *
 * function TestComponent() {
 *   tracker.recordRender()
 *   return null
 * }
 *
 * // Render component multiple times
 * render(<TestComponent />)
 * rerender(<TestComponent />)
 *
 * expect(tracker.getRenderCount()).toBe(2)
 */
export class RenderTracker {
  private renderCount = 0
  private renderTimestamps: number[] = []
  private renderReasons: string[] = []

  /**
   * Record a render event.
   *
   * @param reason - Optional reason for the render (for debugging)
   */
  recordRender(reason?: string): void {
    this.renderCount++
    this.renderTimestamps.push(performance.now())
    if (reason) {
      this.renderReasons.push(reason)
    }
  }

  /**
   * Get total render count.
   */
  getRenderCount(): number {
    return this.renderCount
  }

  /**
   * Get all render timestamps.
   */
  getRenderTimestamps(): number[] {
    return [...this.renderTimestamps]
  }

  /**
   * Get all render reasons.
   */
  getRenderReasons(): string[] {
    return [...this.renderReasons]
  }

  /**
   * Assert that render count is within expected range.
   *
   * @param maxRenders - Maximum acceptable render count
   * @param message - Custom assertion message
   */
  assertMaxRenders(maxRenders: number, message?: string): void {
    if (this.renderCount > maxRenders) {
      throw new Error(
        message ?? `Expected at most ${maxRenders} renders, but got ${this.renderCount}. ` +
          `Render reasons: ${this.renderReasons.join(', ') || 'none recorded'}`
      )
    }
  }

  /**
   * Assert exact render count.
   *
   * @param expected - Expected render count
   * @param message - Custom assertion message
   */
  assertRenderCount(expected: number, message?: string): void {
    if (this.renderCount !== expected) {
      throw new Error(
        message ?? `Expected ${expected} renders, but got ${this.renderCount}. ` +
          `Render reasons: ${this.renderReasons.join(', ') || 'none recorded'}`
      )
    }
  }

  /**
   * Reset the tracker.
   */
  reset(): void {
    this.renderCount = 0
    this.renderTimestamps = []
    this.renderReasons = []
  }
}

// =============================================================================
// POINTER EVENT SIMULATION
// =============================================================================

/**
 * Creates a mock R3F pointer event for testing interactions.
 *
 * @param type - Event type
 * @param options - Event options
 */
export function createMockPointerEvent(
  type: R3FPointerEventType,
  options: {
    pointer?: { x: number; y: number }
    point?: MockVector3
    object?: MockObject3D
    distance?: number
    delta?: number
    uv?: { x: number; y: number }
  } = {}
): MockR3FPointerEvent {
  const stopPropagation = vi.fn()

  const event: MockR3FPointerEvent = {
    stopPropagation,
    pointer: options.pointer ?? { x: 0, y: 0 },
    point: options.point ?? createMockVector3(),
    object: options.object ?? createMockMesh(),
    distance: options.distance ?? 10,
    face: { normal: createMockVector3(0, 1, 0) },
    nativeEvent: new PointerEvent(type, {
      clientX: ((options.pointer?.x ?? 0) + 1) * 500,
      clientY: ((options.pointer?.y ?? 0) + 1) * 500,
      bubbles: true,
    }),
    ray: {
      origin: createMockVector3(0, 0, 10),
      direction: createMockVector3(0, 0, -1),
    },
    camera: createMockObject3D(),
    eventObject: options.object ?? createMockMesh(),
    delta: options.delta,
    uv: options.uv,
  }

  return event
}

/**
 * Simulates a complete drag sequence (pointerdown -> pointermove -> pointerup).
 *
 * @param callbacks - Event handlers to call
 * @param options - Drag sequence options
 *
 * @example
 * const onDragStart = vi.fn()
 * const onDrag = vi.fn()
 * const onDragEnd = vi.fn()
 *
 * simulateDragSequence(
 *   { onPointerDown: onDragStart, onPointerMove: onDrag, onPointerUp: onDragEnd },
 *   { startPoint: [0, 0, 0], endPoint: [5, 0, 0], steps: 10 }
 * )
 *
 * expect(onDragStart).toHaveBeenCalledTimes(1)
 * expect(onDrag).toHaveBeenCalledTimes(10)
 * expect(onDragEnd).toHaveBeenCalledTimes(1)
 */
export function simulateDragSequence(
  callbacks: {
    onPointerDown?: (event: MockR3FPointerEvent) => void
    onPointerMove?: (event: MockR3FPointerEvent) => void
    onPointerUp?: (event: MockR3FPointerEvent) => void
  },
  options: {
    startPoint?: [number, number, number]
    endPoint?: [number, number, number]
    steps?: number
    object?: MockObject3D
  } = {}
): MockR3FPointerEvent[] {
  const {
    startPoint = [0, 0, 0],
    endPoint = [1, 0, 0],
    steps = 5,
    object = createMockMesh(),
  } = options

  const events: MockR3FPointerEvent[] = []

  // Pointer down
  const downEvent = createMockPointerEvent('pointerdown', {
    point: createMockVector3(...startPoint),
    object,
  })
  events.push(downEvent)
  callbacks.onPointerDown?.(downEvent)

  // Pointer move (interpolate between start and end)
  for (let i = 1; i <= steps; i++) {
    const t = i / steps
    const point = createMockVector3(
      startPoint[0] + (endPoint[0] - startPoint[0]) * t,
      startPoint[1] + (endPoint[1] - startPoint[1]) * t,
      startPoint[2] + (endPoint[2] - startPoint[2]) * t
    )
    const moveEvent = createMockPointerEvent('pointermove', {
      point,
      object,
    })
    events.push(moveEvent)
    callbacks.onPointerMove?.(moveEvent)
  }

  // Pointer up
  const upEvent = createMockPointerEvent('pointerup', {
    point: createMockVector3(...endPoint),
    object,
  })
  events.push(upEvent)
  callbacks.onPointerUp?.(upEvent)

  return events
}

/**
 * Simulates a hover sequence (pointerenter -> pointerleave).
 */
export function simulateHoverSequence(
  callbacks: {
    onPointerEnter?: (event: MockR3FPointerEvent) => void
    onPointerLeave?: (event: MockR3FPointerEvent) => void
    onPointerOver?: (event: MockR3FPointerEvent) => void
    onPointerOut?: (event: MockR3FPointerEvent) => void
  },
  object?: MockObject3D
): void {
  const target = object ?? createMockMesh()

  const enterEvent = createMockPointerEvent('pointerenter', { object: target })
  callbacks.onPointerEnter?.(enterEvent)
  callbacks.onPointerOver?.(createMockPointerEvent('pointerover', { object: target }))

  const leaveEvent = createMockPointerEvent('pointerleave', { object: target })
  callbacks.onPointerLeave?.(leaveEvent)
  callbacks.onPointerOut?.(createMockPointerEvent('pointerout', { object: target }))
}

// =============================================================================
// REF MUTATION TRACKING
// =============================================================================

/**
 * Creates a tracked ref that records all mutations for assertion.
 *
 * @example
 * const { ref, tracker } = createTrackedRef(createMockMesh())
 *
 * // Use ref in component/test
 * ref.current.position.x = 5
 * ref.current.rotation.y += 0.1
 *
 * // Assert mutations occurred
 * expect(tracker.getPropertyChanges('position.x')).toContain(5)
 */
export function createTrackedRef<T extends MockObject3D>(
  initial: T
): { ref: { current: T | null }; tracker: RefMutationTracker<T> } {
  const tracker = new RefMutationTracker<T>()
  const proxy = tracker.wrap(initial)

  return {
    ref: { current: proxy },
    tracker,
  }
}

/**
 * Tracks mutations made to a ref object.
 */
export class RefMutationTracker<T extends object> {
  private changes: Map<string, unknown[]> = new Map()

  /**
   * Wrap an object to track property changes.
   */
  wrap(target: T): T {
    return this.createProxy(target, '')
  }

  private createProxy<U extends object>(target: U, path: string): U {
    const createNestedProxy = this.createProxy.bind(this)
    const changes = this.changes
    return new Proxy(target, {
      get(obj, prop) {
        const value = (obj as Record<string | symbol, unknown>)[prop]
        if (typeof value === 'object' && value !== null && typeof prop === 'string') {
          return createNestedProxy(value as U, path ? `${path}.${prop}` : prop)
        }
        return value
      },
      set(obj, prop, value) {
        (obj as Record<string | symbol, unknown>)[prop] = value
        if (typeof prop === 'string') {
          const fullPath = path ? `${path}.${prop}` : prop
          const existing = changes.get(fullPath) ?? []
          existing.push(value)
          changes.set(fullPath, existing)
        }
        return true
      },
    })
  }

  /**
   * Get all recorded changes for a property path.
   */
  getPropertyChanges(path: string): unknown[] {
    return this.changes.get(path) ?? []
  }

  /**
   * Get all property paths that were changed.
   */
  getChangedPaths(): string[] {
    return [...this.changes.keys()]
  }

  /**
   * Check if a property was ever changed.
   */
  wasChanged(path: string): boolean {
    return this.changes.has(path)
  }

  /**
   * Get the last value set for a property.
   */
  getLastValue(path: string): unknown {
    const changes = this.changes.get(path)
    return changes?.[changes.length - 1]
  }

  /**
   * Reset all tracked changes.
   */
  reset(): void {
    this.changes.clear()
  }
}
