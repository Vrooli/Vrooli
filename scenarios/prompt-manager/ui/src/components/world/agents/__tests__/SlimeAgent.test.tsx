/**
 * SlimeAgent Component Tests
 *
 * Tests mounting, prop acceptance, and registry integration.
 * Uses the R3F test harness for mocked Three.js context.
 */

import { describe, it, expect, vi } from 'vitest'
import { render } from '@testing-library/react'
import {
  R3FTestHarness,
  setupR3FMocks,
  setupDreiMocks,
} from '@/test/r3f-component-harness'

// Mock R3F and drei before importing components
vi.mock('@react-three/fiber', () => setupR3FMocks())
vi.mock('@react-three/drei', () => setupDreiMocks())

// Mock lodStore to avoid Zustand issues in tests
vi.mock('@/stores/lodStore', () => ({
  useLODStore: {
    getState: () => ({
      calculateLODLevel: () => 'high' as const,
      updateObjectLOD: vi.fn(),
      removeObject: vi.fn(),
    }),
  },
}))

// Mock useHoverHighlight
vi.mock('@/hooks/useHoverHighlight', () => ({
  useHoverHighlight: () => ({
    isHovered: false,
    hoverProps: {},
  }),
}))

// Mock the slime shader binding (runs in non-WebGL env)
vi.mock('@/lib/shaders/slimeShader', () => ({
  bindSlimeShader: vi.fn(),
  syncSlimeShader: vi.fn(),
}))

// Now import the component under test
import { SlimeAgent } from '../SlimeAgent'
import { AGENT_REGISTRY } from '../../AgentProvider'

describe('SlimeAgent', () => {
  const defaultProps = {
    position: [0, 0, 0] as [number, number, number],
    cursorPosition: null,
    selectedNodes: [],
    isAnimating: false,
  }

  it('mounts without error', () => {
    const { container } = render(
      <R3FTestHarness>
        <SlimeAgent {...defaultProps} />
      </R3FTestHarness>
    )

    expect(container).toBeDefined()
  })

  it('accepts all AgentProps fields', () => {
    const fullProps = {
      position: [1, 2, 3] as [number, number, number],
      cursorPosition: { x: 0.5, y: -0.3 },
      selectedNodes: ['node-1', 'node-2'],
      isAnimating: true,
      onAnimationComplete: vi.fn(),
      onAgentClick: vi.fn(),
      agentId: 'test-agent-123',
      colors: {
        body: '#ff0000',
        head: '#00ff00',
        accent: '#0000ff',
      },
      isSeated: false,
      seatRotation: 0,
    }

    const { container } = render(
      <R3FTestHarness>
        <SlimeAgent {...fullProps} />
      </R3FTestHarness>
    )

    expect(container).toBeDefined()
  })

  it('renders with seated props', () => {
    const { container } = render(
      <R3FTestHarness>
        <SlimeAgent
          {...defaultProps}
          isSeated={true}
          seatRotation={Math.PI / 2}
        />
      </R3FTestHarness>
    )

    expect(container).toBeDefined()
  })

  it('renders floating orbs when nodes are selected', () => {
    const { container } = render(
      <R3FTestHarness>
        <SlimeAgent
          {...defaultProps}
          selectedNodes={['node-1', 'node-2', 'node-3']}
        />
      </R3FTestHarness>
    )

    expect(container).toBeDefined()
  })
})

describe('Agent registry', () => {
  it('includes slime agent', () => {
    expect(AGENT_REGISTRY.slime).toBeDefined()
  })

  it('slime is the default/only agent', () => {
    const agentIds = Object.keys(AGENT_REGISTRY)
    expect(agentIds).toContain('slime')
    expect(agentIds).toHaveLength(1)
  })

  it('slime agent has correct metadata', () => {
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion -- test already asserts key exists
    const slimeConfig = AGENT_REGISTRY.slime!
    expect(slimeConfig.displayName).toBe('Slime Agent')
    expect(slimeConfig.description).toContain('blob')
    expect(slimeConfig.Component).toBeDefined()
  })
})
