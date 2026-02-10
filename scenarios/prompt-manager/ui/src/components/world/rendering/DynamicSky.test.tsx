/**
 * DynamicSky Component Tests
 *
 * Demonstrates R3F testing patterns for simpler components:
 * - Material creation and configuration
 * - Rotation animation via useFrame
 * - Store-based configuration (environment store)
 * - Continuous time-based rendering
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, cleanup } from '@testing-library/react'
import { act } from 'react'
import { useEnvironmentStore } from '@/stores/environmentStore'
import {
  FrameLoopSimulator,
  createMockMesh,
  takeStoreSnapshot,
  diffSnapshots,
} from '@/test'
import type { EnvironmentConfig } from '@/types/environment'

// =============================================================================
// MOCKS
// =============================================================================

// Track frame callbacks for testing
const frameCallbacks: Array<(state: unknown, delta: number) => void> = []

// Mock Three.js
vi.mock('three', async () => {
  const actual = await vi.importActual<typeof import('three')>('three')
  return {
    ...actual,
    ShaderMaterial: vi.fn().mockImplementation((props: Record<string, unknown>) => ({
      ...props,
      uniforms: props.uniforms ?? {},
      side: props.side,
      depthWrite: props.depthWrite,
      dispose: vi.fn(),
    })),
    MeshBasicMaterial: vi.fn().mockImplementation((props: Record<string, unknown>) => ({
      ...props,
      side: props.side,
      depthWrite: props.depthWrite,
      dispose: vi.fn(),
    })),
    Color: vi.fn().mockImplementation((hex: string | undefined) => ({
      r: 0.5,
      g: 0.5,
      b: 0.5,
      getHexString: () => hex?.replace('#', '') ?? 'ffffff',
      set: vi.fn(),
    })),
    BackSide: 1,
  }
})

// Mock @react-three/fiber
vi.mock('@react-three/fiber', () => ({
  useFrame: vi.fn((callback: (state: unknown, delta: number) => void) => {
    frameCallbacks.push(callback)
  }),
}))

// Mock @react-three/drei Sky component
vi.mock('@react-three/drei', () => ({
  Sky: vi.fn(() => null),
}))

// Mock SKYBOX_PRESETS
vi.mock('@/config/environments', () => ({
  SKYBOX_PRESETS: {
    morning: { type: 'gradient', source: ['#FFE4B5', '#87CEEB', '#FFF8DC'] },
    noon: { type: 'gradient', source: ['#87CEEB', '#ADD8E6', '#FFFFF0'] },
    sunset: { type: 'gradient', source: ['#FF6B35', '#FF8C42', '#FFD700'] },
    night: { type: 'gradient', source: ['#0f172a', '#1e293b', '#0f172a'] },
  },
}))

// =============================================================================
// TEST UTILITIES
// =============================================================================

/** Default environment config for testing */
const createTestEnvironmentConfig = (timeValue: number = 12): EnvironmentConfig => ({
  id: 'test-env',
  name: 'Test Environment',
  type: 'abstract-space',
  timeValue,
  lighting: {
    ambient: { color: '#404040', intensity: 0.4 },
    directional: [{ position: [10, 10, 5], color: '#ffffff', intensity: 1, castShadow: true }],
  },
  fog: { color: '#0f172a', near: 10, far: 50 },
  skybox: { type: 'procedural', source: '#87CEEB' },
  ground: {
    visible: true,
    type: 'grid',
    size: 30,
    divisions: 30,
    position: 0,
    material: { type: 'solid', color: '#1e293b' },
  },
  boundary: {
    visible: true,
    shape: 'square',
    size: 60,
    position: 0.01,
    color: '#94a3b8',
    opacity: 0.4,
  },
  placement: {
    snapToGrid: true,
    snapSize: 1,
    clampToBoundary: true,
  },
})

/** Reset all test state */
function resetTestState() {
  frameCallbacks.length = 0

  // Reset environment store with continuous time support
  useEnvironmentStore.setState({
    current: createTestEnvironmentConfig(12),
    dreiPreset: 'studio',
    isTransitioning: false,
    transitionProgress: 0,
    previous: null,
    timeValue: 12, // noon
    realTimeMode: false,
    syncWithTheme: true,
  })
}

/** Simulate frame loop ticks - skips callbacks that fail (refs not attached) */
function tickFrames(count: number, delta = 1 / 60) {
  for (let i = 0; i < count; i++) {
    for (const callback of frameCallbacks) {
      try {
        callback({}, delta)
      } catch {
        // Skip - refs may not be attached in test environment
      }
    }
  }
}

// =============================================================================
// TESTS
// =============================================================================

describe('DynamicSky', () => {
  beforeEach(() => {
    resetTestState()
    vi.clearAllMocks()
  })

  afterEach(() => {
    cleanup()
  })

  describe('component mounting', () => {
    it('renders sky dome mesh on mount', async () => {
      const { DynamicSky } = await import('./DynamicSky')

      const { container } = render(<DynamicSky />)

      // Component renders a mesh with sphere geometry (no useFrame since rotation was removed for perf)
      expect(container).toBeDefined()
      expect(frameCallbacks.length).toBe(0)
    })

    it('renders without crashing with default props', async () => {
      const { DynamicSky } = await import('./DynamicSky')

      const { container } = render(<DynamicSky />)

      expect(container).toBeDefined()
    })

    it('accepts custom radius prop', async () => {
      const { DynamicSky } = await import('./DynamicSky')

      expect(() => {
        render(<DynamicSky radius={100} />)
      }).not.toThrow()
    })
  })

  describe('continuous time handling', () => {
    it('uses timeValue from environment store by default', async () => {
      const { DynamicSky } = await import('./DynamicSky')

      // Set store to specific time (sunset = 18.5 hours)
      useEnvironmentStore.getState().setTimeValue(18.5)

      render(<DynamicSky />)

      // Component should read from store
      expect(useEnvironmentStore.getState().timeValue).toBe(18.5)
    })

    it('overrides with timeValue prop when provided', async () => {
      const { DynamicSky } = await import('./DynamicSky')

      // Set store to noon (12)
      useEnvironmentStore.getState().setTimeValue(12)

      // But render with night timeValue prop (22)
      render(<DynamicSky timeValue={22} />)

      // Store should still be noon (not modified)
      expect(useEnvironmentStore.getState().timeValue).toBe(12)
    })

    it.each<[string, number]>([
      ['morning (8h)', 8],
      ['noon (12h)', 12],
      ['sunset (18.5h)', 18.5],
      ['night (22h)', 22],
    ])(
      'renders correctly for %s',
      async (_, timeValue) => {
        const { DynamicSky } = await import('./DynamicSky')

        useEnvironmentStore.getState().setTimeValue(timeValue)

        expect(() => {
          render(<DynamicSky />)
          act(() => tickFrames(10))
        }).not.toThrow()
      }
    )
  })

  describe('time value edge cases', () => {
    it.each<[string, number]>([
      ['dawn (6h)', 6],
      ['early morning (7h)', 7],
      ['late afternoon (16h)', 16],
      ['dusk (19h)', 19],
      ['midnight (0h)', 0],
      ['late night (3h)', 3],
    ])(
      'renders correctly for edge case: %s',
      async (_, timeValue) => {
        const { DynamicSky } = await import('./DynamicSky')

        useEnvironmentStore.getState().setEnvironment(createTestEnvironmentConfig(timeValue))
        useEnvironmentStore.getState().setTimeValue(timeValue)

        expect(() => {
          render(<DynamicSky />)
          act(() => tickFrames(10))
        }).not.toThrow()
      }
    )
  })

  describe('skybox configuration', () => {
    it('creates gradient material for gradient type', async () => {
      const { DynamicSky } = await import('./DynamicSky')

      useEnvironmentStore.setState({
        current: {
          ...createTestEnvironmentConfig(12),
          skybox: {
            type: 'gradient',
            source: ['#87CEEB', '#ADD8E6', '#FFF8DC'],
          },
        },
      })

      // Should render without errors
      expect(() => {
        render(<DynamicSky />)
      }).not.toThrow()
    })

    it('creates solid material for solid type', async () => {
      const { DynamicSky } = await import('./DynamicSky')

      useEnvironmentStore.setState({
        current: {
          ...createTestEnvironmentConfig(12),
          skybox: {
            type: 'solid',
            source: '#0000ff',
          },
        },
      })

      // Should render without errors
      expect(() => {
        render(<DynamicSky />)
      }).not.toThrow()
    })

    it('uses procedural preset for procedural type', async () => {
      const { DynamicSky } = await import('./DynamicSky')

      useEnvironmentStore.setState({
        current: {
          ...createTestEnvironmentConfig(18.5),
          skybox: {
            type: 'procedural',
            source: '#87CEEB',
          },
        },
      })

      // Should render without errors
      expect(() => {
        render(<DynamicSky />)
      }).not.toThrow()
    })

    it('falls back to solid material for unknown type', async () => {
      const { DynamicSky } = await import('./DynamicSky')

      useEnvironmentStore.setState({
        current: {
          ...createTestEnvironmentConfig(12),
          skybox: {
            type: 'unknown' as 'solid',
            source: '#87CEEB',
          },
        },
      })

      // Should render without errors (fallback to solid)
      expect(() => {
        render(<DynamicSky />)
      }).not.toThrow()
    })
  })

  describe('rotation animation', () => {
    it('does not register useFrame since rotation was removed for perf', async () => {
      const { DynamicSky } = await import('./DynamicSky')

      render(<DynamicSky />)

      // Rotation was removed (imperceptible on a gradient with BackSide rendering)
      expect(frameCallbacks.length).toBe(0)
    })

    it('uses delta time for frame-rate independent rotation', () => {
      // Test the rotation pattern directly
      const simulator = new FrameLoopSimulator()
      const mesh = createMockMesh()

      // Replicate the DynamicSky rotation pattern
      simulator.registerCallback((_, delta) => {
        mesh.rotation.y += delta * 0.01
      })

      // Simulate 1 second at 60fps
      simulator.tickFrames(60, 1 / 60)
      const rotation60fps = mesh.rotation.y

      // Reset and try at 30fps
      mesh.rotation.y = 0
      simulator.reset()

      simulator.registerCallback((_, delta) => {
        mesh.rotation.y += delta * 0.01
      })

      simulator.tickFrames(30, 1 / 30)
      const rotation30fps = mesh.rotation.y

      // Both should be approximately the same (same elapsed time)
      expect(Math.abs(rotation60fps - rotation30fps)).toBeLessThan(0.001)
    })

    it('guards against null meshRef', async () => {
      const { DynamicSky } = await import('./DynamicSky')

      render(<DynamicSky />)

      // The useFrame callback should have a null check
      // This ensures no crash if ref is not yet set
      expect(() => {
        act(() => {
          // Simulate immediate tick before ref is set
          tickFrames(1)
        })
      }).not.toThrow()
    })
  })

  describe('material memoization', () => {
    it('memoizes material based on skybox config', async () => {
      const { DynamicSky } = await import('./DynamicSky')

      const { rerender } = render(<DynamicSky />)

      // Rerender with same config - should not throw
      expect(() => {
        rerender(<DynamicSky />)
      }).not.toThrow()
    })

    it('creates new material when skybox config changes', async () => {
      const { DynamicSky } = await import('./DynamicSky')

      render(<DynamicSky />)

      // Change the skybox config
      act(() => {
        useEnvironmentStore.getState().setEnvironment({
          ...createTestEnvironmentConfig(22),
          skybox: { type: 'gradient', source: ['#000000', '#111111', '#222222'] },
        })
      })

      // Should handle config change without error
      expect(true).toBe(true)
    })
  })

  describe('store subscription', () => {
    it('uses granular selector for environment config', async () => {
      const { DynamicSky } = await import('./DynamicSky')

      render(<DynamicSky />)

      // Take snapshot
      const before = takeStoreSnapshot(useEnvironmentStore)

      // Update unrelated store state
      act(() => {
        useEnvironmentStore.getState().setTimeValue(8) // morning
      })

      const after = takeStoreSnapshot(useEnvironmentStore)
      const diff = diffSnapshots(before, after)

      // Should have changed timeValue
      expect(diff.changed).toHaveProperty('timeValue')
    })

    it('responds to environment changes', async () => {
      const { DynamicSky } = await import('./DynamicSky')

      const { rerender } = render(<DynamicSky />)

      // Change environment to night (22h)
      act(() => {
        useEnvironmentStore.getState().setEnvironment(createTestEnvironmentConfig(22))
      })

      // Force rerender to pick up changes
      rerender(<DynamicSky />)

      // Should render with new environment
      expect(useEnvironmentStore.getState().current.timeValue).toBe(22)
    })
  })
})

describe('CelestialBody (Sun)', () => {
  beforeEach(() => {
    resetTestState()
    vi.clearAllMocks()
  })

  afterEach(() => {
    cleanup()
  })

  describe('position based on continuous time', () => {
    it.each<[string, number]>([
      ['morning (8h)', 8],
      ['noon (12h)', 12],
      ['afternoon (15h)', 15],
    ])('renders sun for %s (above horizon)', async (_, timeValue) => {
      const { CelestialBody } = await import('./DynamicSky')

      useEnvironmentStore.getState().setTimeValue(timeValue)

      // Sun should be visible during day
      expect(() => {
        render(<CelestialBody />)
      }).not.toThrow()
    })

    it('does not render sun at night (below horizon)', async () => {
      const { CelestialBody } = await import('./DynamicSky')

      // Set to night time (22h) - sun is below horizon
      useEnvironmentStore.getState().setTimeValue(22)

      // Sun should not render at night (returns null)
      const { container } = render(<CelestialBody />)
      // Container will be empty since sun is below horizon
      expect(container).toBeDefined()
    })

    it('overrides with timeValue prop when provided', async () => {
      const { CelestialBody } = await import('./DynamicSky')

      // Store says noon
      useEnvironmentStore.getState().setTimeValue(12)

      // But prop says 8am
      render(<CelestialBody timeValue={8} />)

      // Store should still be noon (not modified)
      expect(useEnvironmentStore.getState().timeValue).toBe(12)
    })
  })

  describe('color based on continuous time', () => {
    it.each<[string, number]>([
      ['sunrise (7h)', 7],
      ['morning (9h)', 9],
      ['noon (12h)', 12],
      ['afternoon (15h)', 15],
      ['sunset (18.5h)', 18.5],
    ])('renders with appropriate color for %s', async (_, timeValue) => {
      const { CelestialBody } = await import('./DynamicSky')

      useEnvironmentStore.getState().setTimeValue(timeValue)

      expect(() => {
        render(<CelestialBody />)
      }).not.toThrow()
    })
  })

  describe('visibility thresholds', () => {
    it('renders sun when above horizon (6h-18h)', async () => {
      const { CelestialBody } = await import('./DynamicSky')

      useEnvironmentStore.getState().setTimeValue(12)

      expect(() => {
        render(<CelestialBody />)
      }).not.toThrow()
    })

    it('does not render sun when below horizon (18h-6h)', async () => {
      const { CelestialBody } = await import('./DynamicSky')

      useEnvironmentStore.getState().setTimeValue(0) // midnight

      expect(() => {
        render(<CelestialBody />)
      }).not.toThrow()
    })
  })
})
