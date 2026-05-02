import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import type { TopicsGraphResponse } from '@/types/topicsGraph'

const getTopicsGraph = vi.fn<[string | undefined], Promise<TopicsGraphResponse>>()
const getDrainStatus = vi.fn<[], Promise<unknown>>()
vi.mock('@/services/memberFlowService', () => ({
  getTopicsGraph: (teamId?: string) => getTopicsGraph(teamId),
  getDrainStatus: () => getDrainStatus(),
}))

vi.mock('@xyflow/react', async () => {
  const actual = await vi.importActual<typeof import('@xyflow/react')>('@xyflow/react')
  return {
    ...actual,
    ReactFlow: (props: { children?: React.ReactNode; nodes: { id: string }[]; edges: unknown[] }) => (
      <div data-testid="mock-reactflow" data-node-count={props.nodes.length} data-edge-count={props.edges.length}>
        {props.children}
      </div>
    ),
    Background: () => null,
    Controls: () => null,
    MiniMap: () => null,
    Panel: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  }
})

import { TopicsGraphPanel } from './TopicsGraphPanel'

function fixture(): TopicsGraphResponse {
  return {
    nodes: [
      {
        kind: 'member',
        id: 'member:marketing-crew/researcher',
        label: 'researcher',
        ref: { team: 'marketing-crew', member: 'researcher' },
      },
      {
        kind: 'knowledge_sink',
        id: 'prefix:research-inbox/audience/*',
        label: 'research-inbox/audience/*',
      },
    ],
    edges: [
      {
        from: 'prefix:research-inbox/audience/*',
        to: 'member:marketing-crew/researcher',
        prefix: 'research-inbox/audience/*',
        kind: 'intake',
      },
    ],
    validation: {
      findings: [
        {
          rule: 'orphan_input',
          severity: 'error',
          member: { team: 'marketing-crew', member: 'researcher' },
          prefix: 'research-inbox/audience/*',
          detail: 'No producer',
        },
      ],
      errors: 1,
      warnings: 0,
    },
  }
}

describe('TopicsGraphPanel', () => {
  beforeEach(() => {
    getTopicsGraph.mockReset()
    getDrainStatus.mockReset()
  })

  it('fetches graph for team and renders nodes/edges', async () => {
    getTopicsGraph.mockResolvedValue(fixture())
    render(<TopicsGraphPanel teamId="marketing-crew" />)
    await waitFor(() => expect(screen.getByTestId('mock-reactflow')).toBeInTheDocument())
    expect(getTopicsGraph).toHaveBeenCalledWith('marketing-crew')
    const flow = screen.getByTestId('mock-reactflow')
    expect(flow.getAttribute('data-node-count')).toBe('2')
    expect(flow.getAttribute('data-edge-count')).toBe('1')
  })

  it('shows the validation panel with findings', async () => {
    getTopicsGraph.mockResolvedValue(fixture())
    render(<TopicsGraphPanel teamId="marketing-crew" />)
    await waitFor(() => expect(screen.getByTestId('topics-validation-panel')).toBeInTheDocument())
    expect(screen.getByTestId('topics-finding-error-orphan_input')).toBeInTheDocument()
  })

  it('renders an empty state when the graph has no nodes', async () => {
    getTopicsGraph.mockResolvedValue({
      nodes: [],
      edges: [],
      validation: { findings: [], errors: 0, warnings: 0 },
    })
    render(<TopicsGraphPanel teamId="empty-team" />)
    await waitFor(() => expect(screen.getByText(/No topic flow declared/)).toBeInTheDocument())
  })

  it('shows an error state and retries on demand', async () => {
    getTopicsGraph.mockRejectedValueOnce(new Error('boom'))
    render(<TopicsGraphPanel teamId="marketing-crew" />)
    await waitFor(() => expect(screen.getByText(/Failed to load topics graph: boom/)).toBeInTheDocument())

    getTopicsGraph.mockResolvedValueOnce(fixture())
    fireEvent.click(screen.getByText('Retry'))
    await waitFor(() => expect(screen.getByTestId('mock-reactflow')).toBeInTheDocument())
  })

  it('toggles the validation panel via the toggle button', async () => {
    getTopicsGraph.mockResolvedValue(fixture())
    render(<TopicsGraphPanel teamId="marketing-crew" />)
    await waitFor(() => expect(screen.getByTestId('topics-validation-panel')).toBeInTheDocument())
    fireEvent.click(screen.getByTestId('topics-validation-toggle'))
    expect(screen.queryByTestId('topics-validation-panel')).not.toBeInTheDocument()
  })
})
