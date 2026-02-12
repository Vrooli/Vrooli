/**
 * Tests for GraphToolbar component.
 * Verifies filter toggles, layout switching, regenerate, and health threshold.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { useGraphStore } from '@/stores/graphStore'
import { GraphToolbar } from './GraphToolbar'

// Mock lucide-react icons to simple spans
vi.mock('lucide-react', () => ({
  Users: (props: Record<string, unknown>) => <span data-testid="icon-users" {...props} />,
  Bot: (props: Record<string, unknown>) => <span data-testid="icon-bot" {...props} />,
  Sparkles: (props: Record<string, unknown>) => <span data-testid="icon-sparkles" {...props} />,
  Terminal: (props: Record<string, unknown>) => <span data-testid="icon-terminal" {...props} />,
  LayoutGrid: (props: Record<string, unknown>) => <span data-testid="icon-layout" {...props} />,
  RefreshCw: (props: Record<string, unknown>) => <span data-testid="icon-refresh" {...props} />,
  Maximize2: (props: Record<string, unknown>) => <span data-testid="icon-maximize" {...props} />,
}))

describe('GraphToolbar', () => {
  const defaultProps = {
    layoutDirection: 'TB' as const,
    onToggleLayout: vi.fn(),
    onFitView: vi.fn(),
  }

  beforeEach(() => {
    useGraphStore.setState({
      loading: false,
      filters: {
        showTeams: true,
        showAgents: true,
        showSkills: true,
        showCLIs: true,
        healthThreshold: 0,
      },
    })
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('should render all four type filter buttons', () => {
    render(<GraphToolbar {...defaultProps} />)

    expect(screen.getByText('Teams')).toBeInTheDocument()
    expect(screen.getByText('Agents')).toBeInTheDocument()
    expect(screen.getByText('Skills')).toBeInTheDocument()
    expect(screen.getByText('CLIs')).toBeInTheDocument()
  })

  it('should call setFilter when a type toggle is clicked', () => {
    const spy = vi.fn()
    useGraphStore.setState({ setFilter: spy } as never)

    render(<GraphToolbar {...defaultProps} />)
    fireEvent.click(screen.getByText('Teams'))

    expect(spy).toHaveBeenCalledWith('showTeams', false)
  })

  it('should toggle filter value correctly (off -> on)', () => {
    const spy = vi.fn()
    useGraphStore.setState({
      setFilter: spy,
      filters: {
        showTeams: false,
        showAgents: true,
        showSkills: true,
        showCLIs: true,
        healthThreshold: 0,
      },
    } as never)

    render(<GraphToolbar {...defaultProps} />)
    fireEvent.click(screen.getByText('Teams'))

    expect(spy).toHaveBeenCalledWith('showTeams', true)
  })

  it('should show "Vertical" label for TB layout', () => {
    render(<GraphToolbar {...defaultProps} layoutDirection="TB" />)
    expect(screen.getByText('Vertical')).toBeInTheDocument()
  })

  it('should show "Horizontal" label for LR layout', () => {
    render(<GraphToolbar {...defaultProps} layoutDirection="LR" />)
    expect(screen.getByText('Horizontal')).toBeInTheDocument()
  })

  it('should call onToggleLayout when layout button is clicked', () => {
    render(<GraphToolbar {...defaultProps} />)
    fireEvent.click(screen.getByText('Vertical'))

    expect(defaultProps.onToggleLayout).toHaveBeenCalledOnce()
  })

  it('should call onFitView when fit view button is clicked', () => {
    render(<GraphToolbar {...defaultProps} />)
    const fitBtn = screen.getByTitle('Fit to view')
    fireEvent.click(fitBtn)

    expect(defaultProps.onFitView).toHaveBeenCalledOnce()
  })

  it('should call regenerateGraph when regenerate button is clicked', () => {
    const regen = vi.fn().mockResolvedValue(undefined)
    useGraphStore.setState({ regenerateGraph: regen } as never)

    render(<GraphToolbar {...defaultProps} />)
    fireEvent.click(screen.getByTitle('Regenerate graph'))

    expect(regen).toHaveBeenCalledOnce()
  })

  it('should disable regenerate button when loading', () => {
    useGraphStore.setState({ loading: true })

    render(<GraphToolbar {...defaultProps} />)
    const btn = screen.getByTitle('Regenerate graph')

    expect(btn).toBeDisabled()
  })

  it('should display health threshold percentage', () => {
    useGraphStore.setState({
      filters: {
        showTeams: true,
        showAgents: true,
        showSkills: true,
        showCLIs: true,
        healthThreshold: 0.35,
      },
    })

    render(<GraphToolbar {...defaultProps} />)
    expect(screen.getByText(/35%/)).toBeInTheDocument()
  })

  it('should update health threshold on slider change', () => {
    const spy = vi.fn()
    useGraphStore.setState({ setFilter: spy } as never)

    render(<GraphToolbar {...defaultProps} />)
    const slider = screen.getByRole('slider')
    fireEvent.change(slider, { target: { value: '0.5' } })

    expect(spy).toHaveBeenCalledWith('healthThreshold', 0.5)
  })
})
