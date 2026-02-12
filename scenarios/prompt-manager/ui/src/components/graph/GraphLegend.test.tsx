/**
 * Tests for GraphLegend component.
 * Pure presentation component — no store or API mocks needed.
 */

import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { GraphLegend } from './GraphLegend'

describe('GraphLegend', () => {
  it('should render all four node type labels', () => {
    render(<GraphLegend />)

    expect(screen.getByText('Team')).toBeInTheDocument()
    expect(screen.getByText('Agent')).toBeInTheDocument()
    expect(screen.getByText('Skill')).toBeInTheDocument()
    expect(screen.getByText('CLI')).toBeInTheDocument()
  })

  it('should render all three health level labels', () => {
    render(<GraphLegend />)

    expect(screen.getByText('Critical (<30%)')).toBeInTheDocument()
    expect(screen.getByText('Warning (30-60%)')).toBeInTheDocument()
    expect(screen.getByText('Healthy (>60%)')).toBeInTheDocument()
  })

  it('should render section headers', () => {
    render(<GraphLegend />)

    expect(screen.getByText('Node Types')).toBeInTheDocument()
    expect(screen.getByText('Health')).toBeInTheDocument()
  })

  it('should apply custom className', () => {
    const { container } = render(<GraphLegend className="custom-class" />)

    expect(container.firstChild).toHaveClass('custom-class')
  })
})
