/**
 * Tests for GraphNodeTooltip component.
 * Pure presentation — verifies positioning, health color coding, and factor display.
 */

import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { GraphNodeTooltip } from './GraphNodeTooltip'
import type { GraphNode, HealthScore } from '@/lib/schemas'

const makeNode = (overrides?: Partial<GraphNode>): GraphNode => ({
  id: 'skill-1',
  type: 'skill',
  label: 'Test Skill',
  description: 'A skill',
  status: '',
  tags: [],
  ...overrides,
})

const makeHealth = (overrides: Partial<HealthScore> = {}): HealthScore => {
  const { messages, ...rest } = overrides
  return {
    nodeId: 'skill-1',
    score: 0.7,
    factors: { 'incoming-edges': 0.6, 'outgoing-edges': 0.8 },
    ...rest,
    messages: messages ?? [],
  }
}

describe('GraphNodeTooltip', () => {
  it('should display node label and type', () => {
    render(<GraphNodeTooltip node={makeNode()} x={100} y={50} />)

    expect(screen.getByText('Test Skill')).toBeInTheDocument()
    expect(screen.getByText('skill')).toBeInTheDocument()
  })

  it('should position using x and y props with offset', () => {
    const { container } = render(<GraphNodeTooltip node={makeNode()} x={200} y={100} />)
    const el = container.firstChild as HTMLElement

    expect(el.style.left).toBe('212px')  // x + 12
    expect(el.style.top).toBe('92px')    // y - 8
  })

  it('should show "No health data" when healthScore is null', () => {
    render(<GraphNodeTooltip node={makeNode()} healthScore={null} x={0} y={0} />)

    expect(screen.getByText('No health data')).toBeInTheDocument()
  })

  it('should show "No health data" when healthScore is omitted', () => {
    render(<GraphNodeTooltip node={makeNode()} x={0} y={0} />)

    expect(screen.getByText('No health data')).toBeInTheDocument()
  })

  it('should display health percentage when score provided', () => {
    render(<GraphNodeTooltip node={makeNode()} healthScore={makeHealth({ score: 0.7 })} x={0} y={0} />)

    expect(screen.getByText('70%')).toBeInTheDocument()
    expect(screen.queryByText('No health data')).not.toBeInTheDocument()
  })

  it('should show red color for critical health (<0.3)', () => {
    render(<GraphNodeTooltip node={makeNode()} healthScore={makeHealth({ score: 0.1 })} x={0} y={0} />)

    const percentEl = screen.getByText('10%')
    expect(percentEl).toHaveClass('text-red-400')
  })

  it('should show yellow color for warning health (0.3-0.6)', () => {
    render(<GraphNodeTooltip node={makeNode()} healthScore={makeHealth({ score: 0.45 })} x={0} y={0} />)

    const percentEl = screen.getByText('45%')
    expect(percentEl).toHaveClass('text-yellow-400')
  })

  it('should show green color for healthy health (>=0.6)', () => {
    render(<GraphNodeTooltip node={makeNode()} healthScore={makeHealth({ score: 0.85 })} x={0} y={0} />)

    const percentEl = screen.getByText('85%')
    expect(percentEl).toHaveClass('text-green-400')
  })

  it('should display health factor breakdown', () => {
    render(
      <GraphNodeTooltip
        node={makeNode()}
        healthScore={makeHealth({ factors: { 'incoming-edges': 0.6, 'outgoing-edges': 0.8 } })}
        x={0}
        y={0}
      />,
    )

    expect(screen.getByText('incoming-edges')).toBeInTheDocument()
    expect(screen.getByText('60%')).toBeInTheDocument()
    expect(screen.getByText('outgoing-edges')).toBeInTheDocument()
    expect(screen.getByText('80%')).toBeInTheDocument()
  })

  it('should not render factors section when factors is empty', () => {
    render(
      <GraphNodeTooltip
        node={makeNode()}
        healthScore={makeHealth({ score: 0.5, factors: {} })}
        x={0}
        y={0}
      />,
    )

    // Health score visible but no factor rows
    expect(screen.getByText('50%')).toBeInTheDocument()
    expect(screen.queryByText('incoming-edges')).not.toBeInTheDocument()
  })

  it('should apply custom className', () => {
    const { container } = render(
      <GraphNodeTooltip node={makeNode()} x={0} y={0} className="test-tooltip" />,
    )

    expect(container.firstChild).toHaveClass('test-tooltip')
  })
})
