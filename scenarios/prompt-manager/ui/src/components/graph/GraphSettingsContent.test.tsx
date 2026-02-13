/**
 * Tests for GraphSettingsContent component.
 * Verifies filter toggles, layout switching, regenerate, and health threshold.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { useGraphStore } from '@/stores/graphStore'
import { useGraphHealthConfigStore } from '@/stores/graphHealthConfigStore'
import { GraphSettingsContent } from './GraphSettingsContent'

// Mock lucide-react icons to simple spans
vi.mock('lucide-react', () => ({
  Users: (props: Record<string, unknown>) => <span data-testid="icon-users" {...props} />,
  Bot: (props: Record<string, unknown>) => <span data-testid="icon-bot" {...props} />,
  Sparkles: (props: Record<string, unknown>) => <span data-testid="icon-sparkles" {...props} />,
  Terminal: (props: Record<string, unknown>) => <span data-testid="icon-terminal" {...props} />,
  LayoutGrid: (props: Record<string, unknown>) => <span data-testid="icon-layout" {...props} />,
  RefreshCw: (props: Record<string, unknown>) => <span data-testid="icon-refresh" {...props} />,
  Maximize2: (props: Record<string, unknown>) => <span data-testid="icon-maximize" {...props} />,
  Link2Off: (props: Record<string, unknown>) => <span data-testid="icon-link2off" {...props} />,
  FoldVertical: (props: Record<string, unknown>) => <span data-testid="icon-fold-vertical" {...props} />,
  SlidersHorizontal: (props: Record<string, unknown>) => <span data-testid="icon-sliders" {...props} />,
}))

describe('GraphSettingsContent', () => {
  beforeEach(() => {
    useGraphStore.setState({
      loading: false,
      layoutDirection: 'TB',
      layoutMode: 'compact',
      fitViewRequested: 0,
      filters: {
        showTeams: true,
        showAgents: true,
        showSkills: true,
        showCLIs: true,
        collapseCLIs: false,
        showLowSignalEdges: true,
        autoFitOnChange: true,
        healthThreshold: 0,
      },
    })
    useGraphHealthConfigStore.setState({
      loaded: true,
      loading: false,
      saving: false,
      dirty: false,
      error: null,
      savedConfig: {
        team: { outgoingEdges: 1, incomingEdges: 1, codeUsage: 0.5, recentActivity: 0.5 },
        agent: { outgoingEdges: 1, incomingEdges: 1, codeUsage: 0.5, recentActivity: 0.5 },
        skill: { outgoingEdges: 1, incomingEdges: 1, codeUsage: 0.5, recentActivity: 0.5 },
        cli: { neutralCommands: ['vrooli'], externalToolScore: 0, scenarioFallbackScore: 0 },
      },
      config: {
        team: { outgoingEdges: 1, incomingEdges: 1, codeUsage: 0.5, recentActivity: 0.5 },
        agent: { outgoingEdges: 1, incomingEdges: 1, codeUsage: 0.5, recentActivity: 0.5 },
        skill: { outgoingEdges: 1, incomingEdges: 1, codeUsage: 0.5, recentActivity: 0.5 },
        cli: { neutralCommands: ['vrooli'], externalToolScore: 0, scenarioFallbackScore: 0 },
      },
      loadConfig: vi.fn().mockResolvedValue(undefined),
      saveConfig: vi.fn().mockResolvedValue(true),
      resetToDefault: vi.fn(),
      setEntityWeight: vi.fn(),
      setCLIField: vi.fn(),
      setNeutralCommandsText: vi.fn(),
    })
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('should render all four type filter buttons', () => {
    render(<GraphSettingsContent />)

    expect(screen.getByText('Display')).toBeInTheDocument()
    expect(screen.getByText('Health')).toBeInTheDocument()
    expect(screen.getByText('Teams')).toBeInTheDocument()
    expect(screen.getByText('Agents')).toBeInTheDocument()
    expect(screen.getByText('Skills')).toBeInTheDocument()
    expect(screen.getByText('CLIs')).toBeInTheDocument()
  })

  it('should switch to health tab', () => {
    render(<GraphSettingsContent />)
    fireEvent.click(screen.getByText('Health'))

    expect(screen.getByText('Entity Scoring Weights')).toBeInTheDocument()
    expect(screen.getByText('Save + Recompute')).toBeInTheDocument()
    expect(screen.getByText('No unsaved health changes')).toBeInTheDocument()
  })

  it('should show unsaved preview banner when health config is dirty', () => {
    useGraphHealthConfigStore.setState({ dirty: true })

    render(<GraphSettingsContent />)
    fireEvent.click(screen.getByText('Health'))

    expect(screen.getByText(/Unsaved preview active/)).toBeInTheDocument()
  })

  it('should call setFilter when a type toggle is clicked', () => {
    const spy = vi.fn()
    useGraphStore.setState({ setFilter: spy } as never)

    render(<GraphSettingsContent />)
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
        collapseCLIs: false,
        showLowSignalEdges: true,
        autoFitOnChange: true,
        healthThreshold: 0,
      },
    } as never)

    render(<GraphSettingsContent />)
    fireEvent.click(screen.getByText('Teams'))

    expect(spy).toHaveBeenCalledWith('showTeams', true)
  })

  it('should show layout direction buttons', () => {
    render(<GraphSettingsContent />)
    expect(screen.getByText('Hier')).toBeInTheDocument()
    expect(screen.getByText('Compact')).toBeInTheDocument()
    expect(screen.getByText('Grouped')).toBeInTheDocument()
    expect(screen.getByText('Vertical')).toBeInTheDocument()
    expect(screen.getByText('Horizontal')).toBeInTheDocument()
  })

  it('should call setLayoutMode when layout mode button is clicked', () => {
    const spy = vi.fn()
    useGraphStore.setState({ setLayoutMode: spy } as never)

    render(<GraphSettingsContent />)
    fireEvent.click(screen.getByText('Grouped'))

    expect(spy).toHaveBeenCalledWith('grouped')
  })

  it('should call setLayoutDirection when layout button is clicked', () => {
    const spy = vi.fn()
    useGraphStore.setState({ setLayoutDirection: spy } as never)

    render(<GraphSettingsContent />)
    fireEvent.click(screen.getByText('Horizontal'))

    expect(spy).toHaveBeenCalledWith('TB')
  })

  it('should call requestFitView when fit view button is clicked', () => {
    const spy = vi.fn()
    useGraphStore.setState({ requestFitView: spy } as never)

    render(<GraphSettingsContent />)
    fireEvent.click(screen.getByTitle('Fit to view'))

    expect(spy).toHaveBeenCalledOnce()
  })

  it('should call regenerateGraph when regenerate button is clicked', () => {
    const regen = vi.fn().mockResolvedValue(undefined)
    useGraphStore.setState({ regenerateGraph: regen } as never)

    render(<GraphSettingsContent />)
    fireEvent.click(screen.getByTitle('Regenerate graph'))

    expect(regen).toHaveBeenCalledOnce()
  })

  it('should disable regenerate button when loading', () => {
    useGraphStore.setState({ loading: true })

    render(<GraphSettingsContent />)
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
        collapseCLIs: false,
        showLowSignalEdges: true,
        autoFitOnChange: true,
        healthThreshold: 0.35,
      },
    })

    render(<GraphSettingsContent />)
    expect(screen.getByText(/35%/)).toBeInTheDocument()
  })

  it('should update health threshold on slider change', () => {
    const spy = vi.fn()
    useGraphStore.setState({ setFilter: spy } as never)

    render(<GraphSettingsContent />)
    const slider = screen.getByRole('slider')
    fireEvent.change(slider, { target: { value: '0.5' } })

    expect(spy).toHaveBeenCalledWith('healthThreshold', 0.5)
  })

  it('should toggle collapse CLI filter', () => {
    const spy = vi.fn()
    useGraphStore.setState({ setFilter: spy } as never)

    render(<GraphSettingsContent />)
    fireEvent.click(screen.getByText('Collapse CLI Nodes'))

    expect(spy).toHaveBeenCalledWith('collapseCLIs', true)
  })
})
