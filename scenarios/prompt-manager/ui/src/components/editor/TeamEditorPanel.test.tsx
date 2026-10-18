import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import * as React from 'react'
import { render, screen, waitFor } from '@/test-utils/renderWithProviders'
import { TeamEditorPanel } from './TeamEditorPanel'
import type { TeamDetails } from '@/types/team'
import { buildBoundedParallelExecution, buildDefaultCreateTeamRequest, buildIndependentCoordination } from '@/lib/schemas'

const state = vi.hoisted(() => ({ health: 'running' as 'running' | 'yellow' }))

vi.mock('@/hooks/useTeamEditorStore', () => ({
  useTeamEditorStore: (selector: (value: Record<string, unknown>) => unknown) => selector({
    selectedMemberId: null,
    setSelectedMemberId: vi.fn(),
    edges: [],
    setEdges: vi.fn(),
    updateEdge: vi.fn(),
    reset: vi.fn(),
  }),
}))
vi.mock('@/hooks/useResizableSplitPanel', () => ({
  useResizableSplitPanel: () => ({
    width: 400, isResizing: false, isCollapsed: false, containerRef: { current: null },
    handleResizeStart: vi.fn(), expand: vi.fn(), collapse: vi.fn(),
  }),
}))
vi.mock('@/hooks/useMediaQuery', () => ({ useIsCompactHeader: () => false, useIsMobile: () => false }))
vi.mock('@/hooks/useGlobalKeydown', () => ({ useGlobalKeydown: vi.fn() }))
vi.mock('@/hooks/useTopicsGraph', () => ({ useTopicsGraph: () => ({ graph: null }) }))
vi.mock('@/services/effortService', () => ({ useTeamContractors: () => ({ contractors: [] }) }))
vi.mock('@/services/orgChartService', () => ({ getEdges: vi.fn().mockResolvedValue([]) }))
vi.mock('./teamTabs', () => ({
  TeamDashboardTab: ({ onHealthChange }: { onHealthChange?: (health: 'running' | 'yellow') => void }) => {
    React.useEffect(() => onHealthChange?.(state.health), [onHealthChange])
    return <div data-testid="dashboard-callback" />
  },
  TeamFilesTab: () => null,
  TeamPromptMatrixTab: () => null,
}))
vi.mock('./TeamActivityTab', () => ({ TeamActivityTab: () => null }))
vi.mock('./OrgChartPanel', () => ({ OrgChartPanel: () => null }))
vi.mock('./ManagedTeamsPanel', () => ({ ManagedTeamsPanel: () => null }))
vi.mock('./TopicsGraphPanel', () => ({ TopicsGraphPanel: () => null }))
vi.mock('./MemberDetailPanel', () => ({ MemberDetailPanel: () => null }))
vi.mock('./TeamCodeView', () => ({ TeamCodeView: () => null }))
vi.mock('./MembersTab', () => ({ MemberPickerModal: () => null }))

const team: TeamDetails = {
  id: 'scenario-qa',
  displayName: 'Scenario QA',
  mission: 'Validate important scenarios.',
  enabled: true,
  runtime: { mode: 'multi-process' },
  coordination: buildIndependentCoordination(),
  execution: buildBoundedParallelExecution(2),
  operatingContract: buildDefaultCreateTeamRequest('Scenario QA').operatingContract,
  memberCount: 1,
  roles: [],
  members: [{ agentId: 'worker', displayName: 'Worker', roles: [], status: 'active' }],
  createdAt: '2026-10-16T00:00:00Z',
  updatedAt: '2026-10-16T00:00:00Z',
}

describe('TeamEditorPanel health summary labels', () => {
  beforeEach(() => { state.health = 'running' })
  afterEach(() => { vi.clearAllMocks() })

  function renderPanel() {
    return render(<TeamEditorPanel
      team={team}
      onUpdate={vi.fn().mockResolvedValue(undefined)}
      onAddMember={vi.fn()}
      onUpdateMember={vi.fn()}
      onRemoveMember={vi.fn()}
      onSetRoles={vi.fn()}
      onClose={vi.fn()}
      onDelete={vi.fn()}
    />)
  }

  it('labels a running callback as current execution, not unavailable evidence', async () => {
    state.health = 'running'
    renderPanel()

    await waitFor(() => expect(screen.getByTestId('team-editor-header')).toBeInTheDocument())
    expect(screen.getByTitle('A heartbeat execution is currently running')).toHaveClass('bg-sky-500')
  })

  it('labels yellow callback as owner attention or failure, not only failure', async () => {
    state.health = 'yellow'
    renderPanel()

    await waitFor(() => expect(screen.getByTitle('Heartbeat attention or recorded failures need review')).toBeInTheDocument())
  })
})
