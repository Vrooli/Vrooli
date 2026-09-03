import { describe, expect, it, vi } from 'vitest'
import { render, waitFor } from '@/test-utils/renderWithProviders'
import { OperatingMapFlow } from './OperatingMapFlow'

const { navigate, getOperatingMap, state } = vi.hoisted(() => ({
  navigate: vi.fn(),
  getOperatingMap: vi.fn(),
  state: { latestProps: undefined as Record<string, unknown> | undefined },
}))

vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useNavigate: () => navigate,
}))
vi.mock('@/lib/api', () => ({ api: { getOperatingMap } }))
vi.mock('./FlowShell', () => ({
  FlowShell: (props: Record<string, unknown>) => { state.latestProps = props; return <div data-testid="flow-shell" /> },
  layoutFlowDagre: <NodeType, EdgeType>(nodes: NodeType[], edges: EdgeType[]) => ({ nodes, edges }),
}))

describe('OperatingMapFlow', () => {
  it('renders the composed map and drills a team into its Topics view', async () => {
    getOperatingMap.mockResolvedValue({
      teams: [{ id: 'marketing-crew', label: 'marketing-crew', goal_linkage: 'primary: Broadcast', valid: true }],
      topics: [{ id: 'campaign/brief', label: 'campaign/brief' }],
      edges: [{ from: 'marketing-crew', to: 'campaign/brief' }],
    })
    render(<OperatingMapFlow />)
    await waitFor(() => expect(state.latestProps?.nodes).toHaveLength(2))
    const nodes = state.latestProps?.nodes as Array<{ id: string; data: { label: string }; style: Record<string, unknown> }>
    expect(nodes[0]?.data.label).toContain('primary: Broadcast')
    expect(nodes[0]?.data.label).toContain('contract: valid')
    expect(nodes[0]?.style).toMatchObject({
      background: 'hsl(var(--card))',
      color: 'hsl(var(--card-foreground))',
    })
    expect(nodes[1]?.style).toMatchObject({
      background: 'hsl(var(--secondary))',
      color: 'hsl(var(--secondary-foreground))',
    })
    const edges = state.latestProps?.edges as Array<{ style: Record<string, unknown> }>
    expect(edges[0]?.style).toMatchObject({ stroke: 'hsl(var(--muted-foreground))' })
    const onNodeClick = state.latestProps?.onNodeClick as (_: unknown, node: { id: string }) => void
    onNodeClick({}, { id: 'marketing-crew' })
    expect(navigate).toHaveBeenCalledWith('/teams/marketing-crew?tab=members&subTab=topics')
  })
})
