import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@/test-utils/renderWithProviders'
import { StatsBar } from './StatsBar'
import { useSelectionStore } from '@/stores/selectionStore'

vi.mock('@/hooks/useTeamData', () => ({
  useTeamData: () => ({ teams: [{ id: 't1' }, { id: 't2' }] }),
}))

vi.mock('@/hooks/useAgentData', () => ({
  useAgentData: () => ({ agents: [{ id: 'a1' }, { id: 'a2' }, { id: 'a3' }] }),
}))

vi.mock('@/hooks/useSkillsData', () => ({
  useSkillsData: () => ({ skills: [{ id: 's1' }, { id: 's2' }, { id: 's3' }, { id: 's4' }] }),
}))

describe('StatsBar', () => {
  beforeEach(() => {
    useSelectionStore.setState({ selectedSkillIds: [] })
  })

  it('renders team/agent/skill counts', () => {
    render(<StatsBar />)

    const stats = screen.getByTestId('view-overlay-stats')
    expect(stats).toHaveTextContent('2')
    expect(stats).toHaveTextContent('3')
    expect(stats).toHaveTextContent('4')
  })

  it('renders selected badge in compact mode when selections exist', () => {
    useSelectionStore.setState({ selectedSkillIds: ['one', 'two'] })

    render(<StatsBar compact />)

    expect(screen.getByText('2 selected')).toBeInTheDocument()
  })
})
