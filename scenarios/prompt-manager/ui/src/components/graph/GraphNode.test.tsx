/**
 * Tests for GraphNode (GraphFlowNode) component.
 * Verifies shape variants, health tinting, query styles, and label rendering.
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { GraphFlowNode, type GraphNodeData } from './GraphNode'

// Mock @xyflow/react Handle and Position
vi.mock('@xyflow/react', () => ({
  Handle: ({ type }: { type: string }) => <div data-testid={`handle-${type}`} />,
  Position: { Top: 'top', Bottom: 'bottom', Left: 'left', Right: 'right' },
  memo: (fn: unknown) => fn,
}))

function makeProps(data: Partial<GraphNodeData> = {}): { data: GraphNodeData } {
  return {
    data: {
      label: 'Test Node',
      nodeType: 'agent',
      healthScore: null,
      queryState: 'normal',
      ...data,
    },
  }
}

// Helper to render with necessary props (we only use `data`)
function renderNode(data: Partial<GraphNodeData> = {}) {
  const Component = GraphFlowNode as unknown as React.ComponentType<{ data: GraphNodeData }>
  return render(<Component {...makeProps(data)} />)
}

describe('GraphFlowNode', () => {
  it('should render the label', () => {
    renderNode({ label: 'My Agent' })
    expect(screen.getByText('My Agent')).toBeInTheDocument()
  })

  it('should render target and source handles', () => {
    renderNode()
    expect(screen.getByTestId('handle-target')).toBeInTheDocument()
    expect(screen.getByTestId('handle-source')).toBeInTheDocument()
  })

  it('should apply team shape (rounded-lg)', () => {
    const { container } = renderNode({ nodeType: 'team' })
    const shape = container.querySelector('.rounded-lg')
    expect(shape).toBeInTheDocument()
  })

  it('should apply agent shape (rounded-full)', () => {
    const { container } = renderNode({ nodeType: 'agent' })
    const shape = container.querySelector('.rounded-full')
    expect(shape).toBeInTheDocument()
  })

  it('should apply skill shape (rotate-45 diamond)', () => {
    const { container } = renderNode({ nodeType: 'skill' })
    const shape = container.querySelector('.rotate-45')
    expect(shape).toBeInTheDocument()
  })

  it('should apply CLI shape (clip-hexagon)', () => {
    const { container } = renderNode({ nodeType: 'cli' })
    const shape = container.querySelector('.clip-hexagon')
    expect(shape).toBeInTheDocument()
  })

  it('should show health percentage when healthScore is provided', () => {
    renderNode({ healthScore: 0.73 })
    expect(screen.getByText('73%')).toBeInTheDocument()
  })

  it('should not show health percentage when healthScore is null', () => {
    renderNode({ healthScore: null })
    expect(screen.queryByText('%')).not.toBeInTheDocument()
  })

  it('should apply red fill/border for critical health (<0.3)', () => {
    const { container } = renderNode({ healthScore: 0.1 })
    const shape = container.querySelector('.bg-red-500\\/20.border-red-400\\/90')
    expect(shape).toBeInTheDocument()
  })

  it('should apply yellow fill/border for warning health (0.3-0.6)', () => {
    const { container } = renderNode({ healthScore: 0.45 })
    const shape = container.querySelector('.bg-yellow-500\\/20.border-yellow-300\\/90')
    expect(shape).toBeInTheDocument()
  })

  it('should apply green fill/border for healthy score (>=0.6)', () => {
    const { container } = renderNode({ healthScore: 0.8 })
    expect(container.querySelector('.bg-emerald-500\\/20.border-emerald-300\\/80')).toBeInTheDocument()
  })

  it('should preserve health colors and add ring when queryState is selected', () => {
    const { container } = renderNode({ queryState: 'selected', healthScore: 0.8 })
    // Health colors preserved (green)
    expect(container.querySelector('.bg-emerald-500\\/20.border-emerald-300\\/80')).toBeInTheDocument()
    // Selection ring indicator
    expect(container.querySelector('.ring-2')).toBeInTheDocument()
  })

  it('should use dimmed query styling when queryState is dimmed', () => {
    const { container } = renderNode({ queryState: 'dimmed' })
    expect(container.querySelector('.bg-muted\\/30.border-border\\/40')).toBeInTheDocument()
    expect(container.querySelector('.opacity-45')).toBeInTheDocument()
  })

  it('should counter-rotate label inside diamond (skill)', () => {
    const { container } = renderNode({ nodeType: 'skill', label: 'Diamond Label' })
    const counterRotated = container.querySelector('.-rotate-45')
    expect(counterRotated).toBeInTheDocument()
  })
})
