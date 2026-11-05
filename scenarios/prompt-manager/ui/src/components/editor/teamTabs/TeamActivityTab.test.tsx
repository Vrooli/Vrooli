/**
 * Tests for TeamActivityTab sub-tab control.
 *
 * Covers:
 * - Default sub-tab is 'handoffs'
 * - External initialSubTab prop switches the sub-tab
 * - onSubTabChange fires when user changes sub-tabs
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@/test-utils/renderWithProviders'
import { TeamActivityTab } from './TeamActivityTab'

// Mock the sub-tab content components to avoid their dependencies
vi.mock('./HandoffTimeline', () => ({
  HandoffTimeline: () => <div data-testid="handoff-timeline">Handoffs Content</div>,
}))
vi.mock('./kanban', () => ({
  TaskKanbanBoard: () => <div data-testid="task-board">Tasks Content</div>,
}))

const defaultProps = {
  teamId: 'team-1',
  members: [],
}

describe('TeamActivityTab', () => {
  it('should default to handoffs sub-tab', () => {
    render(<TeamActivityTab {...defaultProps} />)

    // Handoffs tab trigger should be active
    const handoffsTab = screen.getByRole('tab', { name: /handoffs/i })
    expect(handoffsTab).toHaveAttribute('data-state', 'active')

    // Handoffs content should be visible
    expect(screen.getByTestId('handoff-timeline')).toBeInTheDocument()
  })

  it('should switch to tasks sub-tab when initialSubTab is set', () => {
    render(<TeamActivityTab {...defaultProps} initialSubTab="tasks" />)

    const tasksTab = screen.getByRole('tab', { name: /tasks/i })
    expect(tasksTab).toHaveAttribute('data-state', 'active')

    expect(screen.getByTestId('task-board')).toBeInTheDocument()
  })

  it('should respond to initialSubTab changes', () => {
    const { rerender } = render(<TeamActivityTab {...defaultProps} />)

    // Default: handoffs
    expect(screen.getByTestId('handoff-timeline')).toBeInTheDocument()

    rerender(<TeamActivityTab {...defaultProps} initialSubTab="tasks" />)

    expect(screen.getByTestId('task-board')).toBeInTheDocument()
  })

  it('should call onSubTabChange when user clicks a sub-tab', () => {
    const onSubTabChange = vi.fn()
    render(<TeamActivityTab {...defaultProps} onSubTabChange={onSubTabChange} />)

    const tasksTab = screen.getByRole('tab', { name: /tasks/i })
    fireEvent.mouseDown(tasksTab)
    fireEvent.mouseUp(tasksTab)
    fireEvent.click(tasksTab)
    fireEvent.focus(tasksTab)

    expect(onSubTabChange).toHaveBeenCalledWith('tasks')
  })

  it('should not call onSubTabChange on initial render', () => {
    const onSubTabChange = vi.fn()
    render(<TeamActivityTab {...defaultProps} onSubTabChange={onSubTabChange} />)

    expect(onSubTabChange).not.toHaveBeenCalled()
  })
})
