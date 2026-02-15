/**
 * Tests for GraphJsonView component.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import type { GraphResponse } from '@/lib/schemas'

// ============================================================================
// Mocks
// ============================================================================

let lastEditorValue = ''

vi.mock('@monaco-editor/react', async () => {
  const React = await import('react')
  return {
    default: React.forwardRef(function MockEditor(
      props: { value?: string },
      _ref: unknown,
    ) {
      lastEditorValue = props.value ?? ''
      return React.createElement('div', { 'data-testid': 'monaco-editor' }, props.value?.slice(0, 200) ?? '')
    }),
  }
})

vi.mock('@/hooks/use-toast', () => ({
  toast: vi.fn(),
}))

const MOCK_GRAPH_RESPONSE: GraphResponse = {
  generatedAt: '2026-01-01T00:00:00Z',
  graph: {
    nodes: [
      { id: 'team-1', type: 'team', label: 'Test Team', description: 'A team', status: '', tags: [] },
      { id: 'agent-1', type: 'agent', label: 'Test Agent', description: 'An agent', status: 'active', tags: ['test'] },
      { id: 'skill-1', type: 'skill', label: 'Test Skill', description: 'A skill', status: '', tags: [] },
      { id: 'cli-1', type: 'cli', label: 'Test CLI', description: 'A CLI', status: '', tags: [] },
    ],
    edges: [
      { from: 'team-1', to: 'agent-1', kind: 'membership', category: '', sourceFile: '', lineNumber: 0 },
      { from: 'agent-1', to: 'skill-1', kind: 'bold-listed', category: '', sourceFile: 'TOOLS.md', lineNumber: 1 },
      { from: 'skill-1', to: 'cli-1', kind: 'code-usage', category: 'CodeScenarioCLI', sourceFile: 'skill.md', lineNumber: 5 },
    ],
    healthScores: [
      { nodeId: 'team-1', score: 0.5, factors: { 'outgoing-edges': 0.8 }, messages: [] },
      { nodeId: 'agent-1', score: 0.7, factors: { 'incoming-edges': 0.6, 'outgoing-edges': 0.8 }, messages: [] },
      { nodeId: 'skill-1', score: 0.3, factors: { 'incoming-edges': 0.4 }, messages: [] },
      { nodeId: 'cli-1', score: 0.1, factors: {}, messages: [] },
    ],
  },
}

// Must import after mocks
import { useGraphStore } from '@/stores/graphStore'
import { ThemeProvider } from '@/hooks/use-theme'
import { GraphJsonView } from './GraphJsonView'

function renderWithTheme(ui: React.ReactElement) {
  return render(<ThemeProvider>{ui}</ThemeProvider>)
}

// ============================================================================
// Tests
// ============================================================================

describe('GraphJsonView', () => {
  beforeEach(() => {
    lastEditorValue = ''
    useGraphStore.setState({
      graph: MOCK_GRAPH_RESPONSE,
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
    })
    vi.clearAllMocks()
  })

  it('should render filtered graph data as JSON', () => {
    renderWithTheme(<GraphJsonView />)

    expect(screen.getByTestId('monaco-editor')).toBeInTheDocument()
    expect(screen.getByText('graph.json')).toBeInTheDocument()
    expect(screen.getByText('Read-only')).toBeInTheDocument()

    // Verify all 4 nodes are in the JSON
    const parsed = JSON.parse(lastEditorValue) as GraphResponse
    expect(parsed.graph.nodes).toHaveLength(4)
    expect(parsed.graph.edges).toHaveLength(3)
    expect(parsed.graph.healthScores).toHaveLength(4)
  })

  it('should filter out disabled node types from JSON', () => {
    // Disable agents
    useGraphStore.setState({
      filters: {
        showTeams: true,
        showAgents: false,
        showSkills: true,
        showCLIs: true,
        collapseCLIs: false,
        showLowSignalEdges: true,
        autoFitOnChange: true,
        healthThreshold: 0,
      },
    })

    renderWithTheme(<GraphJsonView />)

    const parsed = JSON.parse(lastEditorValue) as GraphResponse
    // Agent node should be gone
    expect(parsed.graph.nodes.map((n) => n.id)).not.toContain('agent-1')
    expect(parsed.graph.nodes).toHaveLength(3)
    // Edges involving agent-1 should be gone
    expect(parsed.graph.edges.every((e) => e.from !== 'agent-1' && e.to !== 'agent-1')).toBe(true)
    // Agent health score should be gone
    expect(parsed.graph.healthScores.every((hs) => hs.nodeId !== 'agent-1')).toBe(true)
  })

  it('should copy JSON to clipboard when copy button is clicked', async () => {
    const writeTextMock = vi.fn().mockResolvedValue(undefined)
    Object.assign(navigator, {
      clipboard: { writeText: writeTextMock },
    })

    renderWithTheme(<GraphJsonView />)

    fireEvent.click(screen.getByTestId('graph-json-copy-button'))

    await waitFor(() => {
      expect(writeTextMock).toHaveBeenCalledTimes(1)
    })

    // Should have been called with the full JSON string
    const copiedJson = writeTextMock.mock.calls[0][0] as string
    const parsed = JSON.parse(copiedJson) as GraphResponse
    expect(parsed.graph.nodes).toHaveLength(4)
  })
})
