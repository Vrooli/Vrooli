/**
 * Tests for GraphView component.
 *
 * Regression test for React error #185 (Maximum update depth exceeded)
 * caused by selectFilteredNodes returning unstable references.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, act, fireEvent } from '@/test-utils/renderWithProviders'
import { MemoryRouter } from 'react-router-dom'
import type { GraphResponse } from '@/lib/schemas'

// ============================================================================
// Mocks
// ============================================================================

// Track render count to detect infinite loops
let renderCount = 0
const MAX_RENDERS = 100
const mockFitView = vi.fn().mockResolvedValue(true)
const mockSetViewport = vi.fn().mockResolvedValue(true)
const mockGetViewport = vi.fn().mockReturnValue({ x: 0, y: 0, zoom: 1 })
const mockFlowToScreenPosition = vi.fn().mockReturnValue({ x: 100, y: 100 })
let latestReactFlowProps: Record<string, unknown> | null = null

// Mock graph data matching the real API shape
const MOCK_GRAPH_RESPONSE: GraphResponse = {
  generatedAt: '2026-01-01T00:00:00Z',
  graph: {
    nodes: [
      { id: 'team-1', type: 'team', label: 'Test Team', description: 'A team', status: '', tags: [] },
      { id: 'agent-1', type: 'agent', label: 'Test Agent', description: 'An agent', status: 'active', tags: ['test'] },
      { id: 'skill-1', type: 'skill', label: 'Test Skill', description: 'A skill', status: '', tags: [] },
      { id: 'cli:test-tool', type: 'cli', label: 'Test CLI', description: 'A CLI', status: '', tags: [] },
    ],
    edges: [
      { from: 'team-1', to: 'agent-1', kind: 'membership', category: '', sourceFile: '', lineNumber: 0 },
      { from: 'agent-1', to: 'skill-1', kind: 'bold-listed', category: '', sourceFile: 'TOOLS.md', lineNumber: 1 },
      { from: 'skill-1', to: 'cli:test-tool', kind: 'code-usage', category: 'CodeScenarioCLI', sourceFile: 'skill.md', lineNumber: 5 },
    ],
    healthScores: [
      { nodeId: 'team-1', score: 0.5, factors: { 'outgoing-edges': 0.8 }, messages: [] },
      { nodeId: 'agent-1', score: 0.7, factors: { 'incoming-edges': 0.6, 'outgoing-edges': 0.8 }, messages: [] },
      { nodeId: 'skill-1', score: 0.3, factors: { 'incoming-edges': 0.4 }, messages: [] },
      { nodeId: 'cli:test-tool', score: 0.1, factors: {}, messages: [] },
    ],
  },
}

// Mock the graphService module
vi.mock('@/services/graphService', () => ({
  getGraph: vi.fn(),
  regenerateGraph: vi.fn(),
  getOrphanedSkills: vi.fn().mockResolvedValue([]),
  getSkilllessAgents: vi.fn().mockResolvedValue([]),
  getEmptyTeams: vi.fn().mockResolvedValue([]),
  getUnaffiliatedAgents: vi.fn().mockResolvedValue([]),
  getCLIlessSkills: vi.fn().mockResolvedValue([]),
  getCircularRefs: vi.fn().mockResolvedValue([]),
  invalidateGraphCache: vi.fn(),
}))

vi.mock('@/hooks/useMediaQuery', () => ({
  useIsMobile: vi.fn(() => false),
}))

vi.mock('@/hooks/use-theme', () => ({
  useResolvedTheme: vi.fn(() => 'dark'),
}))

// Mock Monaco editor (used by GraphJsonView)
vi.mock('@monaco-editor/react', async () => {
  const React = await import('react')
  return {
    default: React.forwardRef(function MockEditor(
      props: { value?: string; 'data-testid'?: string },
      _ref: unknown,
    ) {
      return React.createElement('div', { 'data-testid': 'monaco-editor' }, props.value?.slice(0, 100) ?? '')
    }),
  }
})

// Mock @dagrejs/dagre
vi.mock('@dagrejs/dagre', () => {
  class MockGraph {
    private nodes = new Map<string, { width: number; height: number; x: number; y: number }>()
    setDefaultEdgeLabel(_fn: () => object) {}
    setGraph(_opts: object) {}
    setNode(id: string, dims: { width: number; height: number }) {
      this.nodes.set(id, { ...dims, x: 0, y: 0 })
    }
    setEdge(_from: string, _to: string) {}
    node(id: string) {
      return this.nodes.get(id) ?? { width: 160, height: 80, x: 100, y: 100 }
    }
    getNodes() { return this.nodes }
  }

  return {
    default: {
      graphlib: {
        Graph: MockGraph,
      },
      layout: (g: MockGraph) => {
        let x = 0, y = 0
        for (const [, node] of (g as unknown as { getNodes: () => Map<string, { x: number; y: number }> }).getNodes()) {
          node.x = x
          node.y = y
          x += 200
          if (x > 800) { x = 0; y += 200 }
        }
      },
    },
  }
})

// Mock @xyflow/react with render counting
vi.mock('@xyflow/react', async () => {
  const React = await import('react')

  const MockReactFlow = React.forwardRef(function MockReactFlow(
    props: Record<string, unknown>,
    _ref: unknown,
  ) {
    renderCount++
    latestReactFlowProps = props
    if (renderCount > MAX_RENDERS) {
      throw new Error(`INFINITE LOOP DETECTED: ReactFlow rendered ${renderCount} times`)
    }
    const nodes = (props.nodes as Array<{ id: string; data?: { label?: string } }> | undefined) ?? []
    const edges = (props.edges as Array<{ id: string }> | undefined) ?? []
    return React.createElement('div', { 'data-testid': 'react-flow' },
      React.createElement('span', { 'data-testid': 'node-count' }, `${nodes.length} nodes`),
      React.createElement('span', { 'data-testid': 'edge-count' }, `${edges.length} edges`),
      nodes.map((n) =>
        React.createElement('div', { key: n.id, 'data-testid': `node-${n.id}` }, n.data?.label ?? n.id),
      ),
    )
  })

  return {
    ReactFlow: MockReactFlow,
    ReactFlowProvider: ({ children }: { children: React.ReactNode }) =>
      React.createElement('div', { 'data-testid': 'react-flow-provider' }, children),
    Background: () => null,
    Controls: () => null,
    MiniMap: () => null,
    Panel: ({ children }: { children: React.ReactNode }) =>
      React.createElement('div', null, children),
    MarkerType: { ArrowClosed: 'arrowclosed' },
    useNodesState: (initial: unknown[]) => {
      const [nodes, setNodes] = React.useState(initial)
      const onNodesChange = React.useCallback(() => {}, [])
      return [nodes, setNodes, onNodesChange]
    },
    useEdgesState: (initial: unknown[]) => {
      const [edges, setEdges] = React.useState(initial)
      const onEdgesChange = React.useCallback(() => {}, [])
      return [edges, setEdges, onEdgesChange]
    },
    Position: { Top: 'top', Bottom: 'bottom', Left: 'left', Right: 'right' },
    Handle: () => null,
    useReactFlow: () => ({
      fitView: mockFitView,
      setViewport: mockSetViewport,
      getViewport: mockGetViewport,
      flowToScreenPosition: mockFlowToScreenPosition,
    }),
  }
})

// Must import after mocks
import { getGraph } from '@/services/graphService'
import { useGraphStore } from '@/stores/graphStore'
import { GraphView } from './GraphView'
import { useIsMobile } from '@/hooks/useMediaQuery'

function renderGraphView() {
  return render(
    <MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
      <GraphView />
    </MemoryRouter>
  )
}

// ============================================================================
// Tests
// ============================================================================

describe('GraphView', () => {
  beforeEach(() => {
    renderCount = 0
    latestReactFlowProps = null
    mockFlowToScreenPosition.mockReturnValue({ x: 100, y: 100 })
    // Reset Zustand store state
    useGraphStore.setState({
      graph: null,
      loading: false,
      error: null,
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
      highlightedNodeIds: new Set(),
      highlightSource: null,
      queryDisplayMode: 'dim-others',
      layoutDirection: 'TB',
      layoutMode: 'compact',
      fitViewRequested: 0,
      viewport: null,
    })
    localStorage.removeItem('pm.graphViewport')
    localStorage.removeItem('pm.graphViewSettings.v1')
    vi.clearAllMocks()
    vi.mocked(useIsMobile).mockReturnValue(false)
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('should render loading state then graph without infinite loop (regression: React #185)', async () => {
    // This test catches the infinite re-render loop caused by selectFilteredNodes
    // returning unstable array references. Before the fix, this would crash with
    // "Maximum update depth exceeded" (React error #185).
    const mockGetGraph = vi.mocked(getGraph)
    mockGetGraph.mockResolvedValue(MOCK_GRAPH_RESPONSE)

    renderGraphView()

    // Should show loading initially
    expect(screen.getByText('Loading graph...')).toBeInTheDocument()

    // Wait for graph to load
    await waitFor(() => {
      expect(screen.getByTestId('edge-count')).toHaveTextContent('3 edges')
    })

    // Should show nodes
    expect(screen.getByTestId('node-count')).toHaveTextContent('4 nodes')
    expect(screen.getByTestId('edge-count')).toHaveTextContent('3 edges')

    // CRITICAL: Verify no infinite loop — healthy render completes well under limit
    expect(renderCount).toBeLessThan(20)
    expect(mockFitView).toHaveBeenCalled()
  })

  it('should render empty state when graph has no nodes', async () => {
    const mockGetGraph = vi.mocked(getGraph)
    mockGetGraph.mockResolvedValue({
      generatedAt: '2026-01-01T00:00:00Z',
      graph: { nodes: [], edges: [], healthScores: [] },
    })

    renderGraphView()

    await waitFor(() => {
      expect(screen.getByText('No graph data available')).toBeInTheDocument()
    })
  })

  it('should render error state on fetch failure', async () => {
    const mockGetGraph = vi.mocked(getGraph)
    mockGetGraph.mockRejectedValue(new Error('Network error'))

    renderGraphView()

    await waitFor(() => {
      expect(screen.getByText('Failed to load graph')).toBeInTheDocument()
    })

    expect(screen.getByText('Network error')).toBeInTheDocument()
  })

  it('should handle null graph response (validation error)', async () => {
    const mockGetGraph = vi.mocked(getGraph)
    mockGetGraph.mockResolvedValue(null as unknown as GraphResponse)

    renderGraphView()

    await waitFor(() => {
      expect(screen.getByText('No graph data available')).toBeInTheDocument()
    })
  })

  it('should not re-render excessively after data loads', async () => {
    const mockGetGraph = vi.mocked(getGraph)
    mockGetGraph.mockResolvedValue(MOCK_GRAPH_RESPONSE)

    renderGraphView()

    await waitFor(() => {
      expect(screen.getByTestId('react-flow')).toBeInTheDocument()
    })

    // Record render count after initial data load
    const postLoadRenders = renderCount

    // Wait an additional tick to ensure no further renders
    await act(async () => {
      await new Promise((r) => setTimeout(r, 100))
    })

    // Should not have many additional renders after data settled
    const additionalRenders = renderCount - postLoadRenders
    expect(additionalRenders).toBeLessThan(5)
  })

  it('should display all node types', async () => {
    const mockGetGraph = vi.mocked(getGraph)
    mockGetGraph.mockResolvedValue(MOCK_GRAPH_RESPONSE)

    renderGraphView()

    await waitFor(() => {
      expect(screen.getByTestId('node-team-1')).toHaveTextContent('Test Team')
    })
    expect(screen.getByTestId('node-agent-1')).toHaveTextContent('Test Agent')
    expect(screen.getByTestId('node-skill-1')).toHaveTextContent('Test Skill')
    expect(screen.getByTestId('node-cli:test-tool')).toHaveTextContent('Test CLI')
  })

  it('should mark graph edges as non-interactive', async () => {
    const mockGetGraph = vi.mocked(getGraph)
    mockGetGraph.mockResolvedValue(MOCK_GRAPH_RESPONSE)

    renderGraphView()

    await waitFor(() => {
      expect(screen.getByTestId('react-flow')).toBeInTheDocument()
    })

    const flowProps = latestReactFlowProps as Record<string, unknown>
    const flowEdges = (flowProps.edges as unknown[]) as Array<{
      selectable?: boolean
      focusable?: boolean
      style?: { pointerEvents?: string }
    }>
    expect(flowEdges.length).toBeGreaterThan(0)
    for (const edge of flowEdges) {
      expect(edge.selectable).toBe(false)
      expect(edge.focusable).toBe(false)
      expect(edge.style?.pointerEvents).toBe('none')
    }
  })

  it('should restore saved viewport when available', async () => {
    const mockGetGraph = vi.mocked(getGraph)
    mockGetGraph.mockResolvedValue(MOCK_GRAPH_RESPONSE)
    useGraphStore.setState({ viewport: { x: 120, y: 80, zoom: 0.75 } })

    renderGraphView()

    await waitFor(() => {
      expect(mockSetViewport).toHaveBeenCalledWith({ x: 120, y: 80, zoom: 0.75 }, { duration: 0 })
    })
  })

  it('should clamp desktop popover position to viewport bounds', async () => {
    const mockGetGraph = vi.mocked(getGraph)
    mockGetGraph.mockResolvedValue(MOCK_GRAPH_RESPONSE)
    mockFlowToScreenPosition.mockReturnValue({ x: 2000, y: -500 })

    renderGraphView()

    await waitFor(() => {
      expect(screen.getByTestId('react-flow')).toBeInTheDocument()
    })

    const flowProps = latestReactFlowProps as Record<string, unknown>
    const onNodeClick = flowProps.onNodeClick as (event: unknown, node: { id: string; position: { x: number; y: number } }) => void

    act(() => {
      onNodeClick({}, { id: 'agent-1', position: { x: 0, y: 0 } })
    })

    const popover = await screen.findByTestId('graph-node-popover-desktop')
    expect(popover).toHaveStyle({ left: '736px', top: '8px' })
  })

  it('should hide non-selected query results when hide-others mode is active', async () => {
    const mockGetGraph = vi.mocked(getGraph)
    mockGetGraph.mockResolvedValue(MOCK_GRAPH_RESPONSE)

    useGraphStore.setState({
      highlightedNodeIds: new Set(['agent-1']),
      queryDisplayMode: 'hide-others',
    })

    renderGraphView()

    await waitFor(() => {
      expect(screen.getByTestId('react-flow')).toBeInTheDocument()
    })

    expect(screen.getByTestId('node-count')).toHaveTextContent('1 nodes')
    const flowProps = latestReactFlowProps as Record<string, unknown>
    const flowNodes = (flowProps.nodes as unknown[]) as Array<{ data?: { queryState?: string } }>
    expect(flowNodes).toHaveLength(1)
    expect(flowNodes[0]?.data?.queryState).toBe('selected')
  })

  it('should dim non-selected nodes when dim-others mode is active', async () => {
    const mockGetGraph = vi.mocked(getGraph)
    mockGetGraph.mockResolvedValue(MOCK_GRAPH_RESPONSE)

    useGraphStore.setState({
      highlightedNodeIds: new Set(['agent-1']),
      queryDisplayMode: 'dim-others',
    })

    renderGraphView()

    await waitFor(() => {
      expect(screen.getByTestId('react-flow')).toBeInTheDocument()
    })

    const flowProps = latestReactFlowProps as Record<string, unknown>
    const flowNodes = (flowProps.nodes as unknown[]) as Array<{ id: string; data?: { queryState?: string } }>
    expect(flowNodes).toHaveLength(4)
    expect(flowNodes.find((node) => node.id === 'agent-1')?.data?.queryState).toBe('selected')
    expect(flowNodes.find((node) => node.id === 'team-1')?.data?.queryState).toBe('dimmed')
  })

  it('should keep the mode toggle visible on mobile', async () => {
    const mockGetGraph = vi.mocked(getGraph)
    mockGetGraph.mockResolvedValue(MOCK_GRAPH_RESPONSE)
    vi.mocked(useIsMobile).mockReturnValue(true)

    renderGraphView()

    await waitFor(() => {
      expect(screen.getByTestId('react-flow')).toBeInTheDocument()
    })

    expect(screen.getByTestId('graph-mode-toggle')).toBeInTheDocument()
  })

  it('should toggle between visual and JSON modes', async () => {
    const mockGetGraph = vi.mocked(getGraph)
    mockGetGraph.mockResolvedValue(MOCK_GRAPH_RESPONSE)

    renderGraphView()

    await waitFor(() => {
      expect(screen.getByTestId('react-flow')).toBeInTheDocument()
    })

    // Toggle should be visible with Visual active
    expect(screen.getByTestId('graph-mode-toggle')).toBeInTheDocument()

    // Click JSON mode
    fireEvent.click(screen.getByTestId('graph-mode-json'))

    // ReactFlow should be gone, Monaco should appear
    await waitFor(() => {
      expect(screen.queryByTestId('react-flow')).not.toBeInTheDocument()
      expect(screen.getByTestId('monaco-editor')).toBeInTheDocument()
    })

    // Click Visual mode to switch back
    fireEvent.click(screen.getByTestId('graph-mode-visual'))

    await waitFor(() => {
      expect(screen.getByTestId('react-flow')).toBeInTheDocument()
      expect(screen.queryByTestId('monaco-editor')).not.toBeInTheDocument()
    })
  })

  // ==========================================================================
  // Click-to-focus neighborhood tests
  // ==========================================================================

  describe('click-to-focus neighborhood', () => {
    async function setupGraph() {
      const mockGetGraph = vi.mocked(getGraph)
      mockGetGraph.mockResolvedValue(MOCK_GRAPH_RESPONSE)
      renderGraphView()
      await waitFor(() => {
        expect(screen.getByTestId('react-flow')).toBeInTheDocument()
      })
    }

    /** Get fresh props from the latest render (callbacks update on re-render). */
    function getProps(): Record<string, unknown> {
      expect(latestReactFlowProps).not.toBeNull()
      return latestReactFlowProps as Record<string, unknown>
    }

    async function clickNode(id: string) {
      await act(async () => {
        getOnNodeClick(getProps())({}, { id, position: { x: 0, y: 0 } })
        await Promise.resolve()
      })
    }

    async function clickPane() {
      await act(async () => {
        getOnPaneClick(getProps())()
        await Promise.resolve()
      })
    }

    function getOnNodeClick(props: Record<string, unknown>) {
      return props.onNodeClick as (event: unknown, node: { id: string; position: { x: number; y: number } }) => void
    }

    function getOnPaneClick(props: Record<string, unknown>) {
      return props.onPaneClick as () => void
    }

    it('should set focus highlight when clicking a node with no active query', async () => {
      await setupGraph()

      await clickNode('team-1')

      const state = useGraphStore.getState()
      expect(state.highlightSource).toBe('focus')
      expect(state.highlightedNodeIds.size).toBeGreaterThan(0)
      expect(state.highlightedNodeIds.has('team-1')).toBe(true)
      // Should include neighbor agent-1
      expect(state.highlightedNodeIds.has('agent-1')).toBe(true)
    })

    it('should clear focus when clicking the same node again', async () => {
      await setupGraph()

      // First click: focus
      await clickNode('team-1')
      expect(useGraphStore.getState().highlightSource).toBe('focus')

      // Second click on same node: toggle off (re-read fresh props)
      await clickNode('team-1')

      const state = useGraphStore.getState()
      expect(state.highlightSource).toBe(null)
      expect(state.highlightedNodeIds.size).toBe(0)
    })

    it('should switch focus when clicking a different node', async () => {
      await setupGraph()

      // Click team-1
      await clickNode('team-1')
      expect(useGraphStore.getState().highlightedNodeIds.has('team-1')).toBe(true)

      // Click skill-1 (different node, re-read fresh props)
      await clickNode('skill-1')

      const state = useGraphStore.getState()
      expect(state.highlightSource).toBe('focus')
      expect(state.highlightedNodeIds.has('skill-1')).toBe(true)
    })

    it('should clear focus on pane click', async () => {
      await setupGraph()

      // Click a node to focus
      await clickNode('agent-1')
      expect(useGraphStore.getState().highlightSource).toBe('focus')

      // Click empty pane (re-read fresh props)
      await clickPane()

      const state = useGraphStore.getState()
      expect(state.highlightSource).toBe(null)
      expect(state.highlightedNodeIds.size).toBe(0)
    })

    it('should not replace query highlights when clicking a node during active query', async () => {
      // Set up an active query highlight
      useGraphStore.setState({
        highlightedNodeIds: new Set(['skill-1']),
        highlightSource: 'query',
      })

      await setupGraph()

      // Click a node while query is active
      await clickNode('team-1')

      const state = useGraphStore.getState()
      // Query should still be active (not replaced by focus)
      expect(state.highlightSource).toBe('query')
      expect(state.highlightedNodeIds).toEqual(new Set(['skill-1']))
    })

    it('should include cli: prefixed nodes in focus neighborhood', async () => {
      await setupGraph()

      // Click skill-1 which connects to cli:test-tool
      await act(async () => {
        getOnNodeClick(latestReactFlowProps ?? {})({}, { id: 'skill-1', position: { x: 0, y: 0 } })
        await Promise.resolve()
      })

      const state = useGraphStore.getState()
      expect(state.highlightSource).toBe('focus')
      // CLI node should be IN the highlight set (was previously dropped by cli: prefix filter)
      expect(state.highlightedNodeIds.has('cli:test-tool')).toBe(true)
      expect(state.highlightedNodeIds.has('skill-1')).toBe(true)
    })

    it('should dim cli: nodes in focus mode with dim-others display', async () => {
      useGraphStore.setState({ queryDisplayMode: 'dim-others' })
      await setupGraph()

      // Click team-1 — full neighborhood includes cli:test-tool at depth 3
      await act(async () => {
        getOnNodeClick(latestReactFlowProps ?? {})({}, { id: 'team-1', position: { x: 0, y: 0 } })
        await Promise.resolve()
      })

      // The cli:test-tool node should be selected (not dimmed)
      const flowProps = latestReactFlowProps as Record<string, unknown>
      const flowNodes = (flowProps.nodes as unknown[]) as Array<{ id: string; data?: { queryState?: string } }>
      const cliNode = flowNodes.find((n) => n.id === 'cli:test-tool')
      expect(cliNode?.data?.queryState).toBe('selected')
    })

    it('should not clear query highlights on pane click', async () => {
      useGraphStore.setState({
        highlightedNodeIds: new Set(['skill-1']),
        highlightSource: 'query',
      })

      await setupGraph()

      await clickPane()

      const state = useGraphStore.getState()
      // Query highlights should remain
      expect(state.highlightSource).toBe('query')
      expect(state.highlightedNodeIds.has('skill-1')).toBe(true)
    })
  })
})
