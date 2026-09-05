/**
 * Tests for TaskKanbanCard.
 *
 * Covers:
 * - Renders priority badge, title, assignee, timestamp
 * - Inline title editing (Enter commits, Escape cancels)
 * - Delete button fires callback
 * - Notes section expands/collapses
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@/test-utils/renderWithProviders'
import { TaskKanbanCard } from './TaskKanbanCard'
import type { TeamTask } from '@/services/heartbeatService'
import type { TeamMember } from '@/types/team'

// Mock @dnd-kit/sortable so we don't need a DndContext
vi.mock('@dnd-kit/sortable', () => ({
  useSortable: () => ({
    attributes: {},
    listeners: {},
    setNodeRef: () => {},
    transform: null,
    transition: null,
    isDragging: false,
  }),
}))

vi.mock('@dnd-kit/utilities', () => ({
  CSS: {
    Transform: { toString: () => '' },
  },
}))

const baseTask: TeamTask = {
  id: 'task-1',
  title: 'Write unit tests',
  status: 'todo',
  assignee: 'agent-a',
  priority: 'P2',
  createdBy: 'ui-user',
  createdAt: '2026-03-20T10:00:00Z',
  updatedAt: '2026-03-22T14:30:00Z',
  notes: [
    { at: '2026-03-21T12:00:00Z', by: 'agent-a', text: 'Started research' },
    { at: '2026-03-22T14:30:00Z', by: 'agent-b', text: 'Draft complete' },
  ],
}

const members: TeamMember[] = [
  { agentId: 'agent-a', displayName: 'Alice', roles: [], status: 'active' },
  { agentId: 'agent-b', displayName: 'Bob', roles: [], status: 'active' },
]

const defaultProps = {
  task: baseTask,
  members,
  allAgents: [],
  onUpdateTask: vi.fn(),
  onDeleteTask: vi.fn(),
}

describe('TaskKanbanCard', () => {
  it('renders priority badge and title', () => {
    render(<TaskKanbanCard {...defaultProps} />)

    expect(screen.getByText('P2')).toBeInTheDocument()
    expect(screen.getByText('Write unit tests')).toBeInTheDocument()
  })

  it('renders assignee name', () => {
    render(<TaskKanbanCard {...defaultProps} />)

    expect(screen.getByText('Alice')).toBeInTheDocument()
  })

  it('shows note count badge', () => {
    render(<TaskKanbanCard {...defaultProps} />)

    expect(screen.getByText('2')).toBeInTheDocument()
  })

  it('expands and collapses notes when clicking the note count', () => {
    render(<TaskKanbanCard {...defaultProps} />)

    // Notes should not be visible initially
    expect(screen.queryByText(/Started research/)).not.toBeInTheDocument()

    // Click note count to expand
    fireEvent.click(screen.getByText('2'))
    expect(screen.getByText(/Started research/)).toBeInTheDocument()
    expect(screen.getByText(/Draft complete/)).toBeInTheDocument()

    // Click again to collapse
    fireEvent.click(screen.getByText('2'))
    expect(screen.queryByText(/Started research/)).not.toBeInTheDocument()
  })

  it('fires onDeleteTask when delete button is clicked', () => {
    const onDeleteTask = vi.fn()
    render(<TaskKanbanCard {...defaultProps} onDeleteTask={onDeleteTask} />)

    const deleteButton = screen.getByTitle('Delete task')
    fireEvent.click(deleteButton)

    expect(onDeleteTask).toHaveBeenCalledWith(baseTask)
  })

  it('enters edit mode on title click and commits on Enter', () => {
    const onUpdateTask = vi.fn()
    render(<TaskKanbanCard {...defaultProps} onUpdateTask={onUpdateTask} />)

    // Click title to enter edit mode
    fireEvent.click(screen.getByText('Write unit tests'))

    const input = screen.getByDisplayValue('Write unit tests')
    fireEvent.change(input, { target: { value: 'Updated title' } })
    fireEvent.keyDown(input, { key: 'Enter' })

    expect(onUpdateTask).toHaveBeenCalledWith('task-1', { title: 'Updated title' })
  })

  it('cancels edit on Escape', () => {
    const onUpdateTask = vi.fn()
    render(<TaskKanbanCard {...defaultProps} onUpdateTask={onUpdateTask} />)

    fireEvent.click(screen.getByText('Write unit tests'))

    const input = screen.getByDisplayValue('Write unit tests')
    fireEvent.change(input, { target: { value: 'Changed' } })
    fireEvent.keyDown(input, { key: 'Escape' })

    // Should NOT call update
    expect(onUpdateTask).not.toHaveBeenCalled()
    // Original title should be back
    expect(screen.getByText('Write unit tests')).toBeInTheDocument()
  })

  it('does not show assignee section when unassigned', () => {
    const taskNoAssignee = { ...baseTask, assignee: '' }
    render(<TaskKanbanCard {...defaultProps} task={taskNoAssignee} />)

    expect(screen.queryByText('Alice')).not.toBeInTheDocument()
  })

  it('does not show notes button when there are no notes', () => {
    const taskNoNotes = { ...baseTask, notes: [] }
    render(<TaskKanbanCard {...defaultProps} task={taskNoNotes} />)

    // No note count should be visible
    expect(screen.queryByRole('button', { name: /0/ })).not.toBeInTheDocument()
  })

  it('renders drag overlay variant with rotation class', () => {
    const { container } = render(<TaskKanbanCard {...defaultProps} dragOverlay />)

    const card = container.firstChild as HTMLElement
    expect(card.className).toContain('rotate-3')
    expect(card.className).toContain('shadow-xl')
  })
})
