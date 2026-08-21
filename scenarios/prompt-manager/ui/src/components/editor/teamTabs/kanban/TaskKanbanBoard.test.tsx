/**
 * Tests for TaskKanbanBoard.
 *
 * Covers:
 * - Renders 4 columns with correct titles
 * - Shows loading spinner while fetching
 * - Shows error state with retry
 * - Shows empty state when all columns are empty
 * - Add task form opens/closes and creates task
 * - Delete triggers confirmation dialog
 */

import { afterEach, describe, it, expect, vi, beforeEach, type MockInstance } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@/test-utils/renderWithProviders'
import { TaskKanbanBoard } from './TaskKanbanBoard'
import type { TeamTask, TaskBoardResponse } from '@/services/heartbeatService'
import type { TeamMember } from '@/types/team'

// Mock heartbeatService
vi.mock('@/services/heartbeatService', () => ({
  getTaskBoard: vi.fn(),
  addTask: vi.fn(),
  updateTask: vi.fn(),
  deleteTask: vi.fn(),
}))

// Mock @dnd-kit
vi.mock('@dnd-kit/core', () => ({
  DndContext: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  DragOverlay: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  closestCenter: vi.fn(),
  pointerWithin: vi.fn(() => []),
  PointerSensor: vi.fn(),
  useSensor: vi.fn(() => ({})),
  useSensors: vi.fn(() => []),
  useDroppable: () => ({ setNodeRef: () => {}, isOver: false }),
}))

vi.mock('@dnd-kit/sortable', () => ({
  SortableContext: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  verticalListSortingStrategy: {},
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

// Mock ConfirmDialog to inspect its props
vi.mock('@/components/shared/ConfirmDialog', () => ({
  ConfirmDialog: ({ isOpen, title, onConfirm }: { isOpen: boolean; title: string; onConfirm: () => void }) =>
    isOpen ? (
      <div data-testid="confirm-dialog">
        <span>{title}</span>
        <button onClick={onConfirm}>Confirm</button>
      </div>
    ) : null,
}))

import * as heartbeatService from '@/services/heartbeatService'

const mockTasks: TeamTask[] = [
  {
    id: 'task-1',
    title: 'First task',
    status: 'todo',
    assignee: 'agent-a',
    priority: 'P1',
    createdBy: 'ui-user',
    createdAt: '2026-03-20T10:00:00Z',
    updatedAt: '2026-03-22T14:30:00Z',
  },
  {
    id: 'task-2',
    title: 'Second task',
    status: 'in-progress',
    assignee: 'agent-b',
    priority: 'P2',
    createdBy: 'ui-user',
    createdAt: '2026-03-20T11:00:00Z',
    updatedAt: '2026-03-22T15:00:00Z',
  },
  {
    id: 'task-3',
    title: 'Blocked task',
    status: 'blocked',
    assignee: '',
    priority: 'P3',
    createdBy: 'agent-a',
    createdAt: '2026-03-21T09:00:00Z',
    updatedAt: '2026-03-22T16:00:00Z',
  },
]

const members: TeamMember[] = [
  { agentId: 'agent-a', displayName: 'Alice', roles: [], status: 'active' },
  { agentId: 'agent-b', displayName: 'Bob', roles: [], status: 'active' },
]

const defaultProps = {
  teamId: 'team-1',
  members,
  allAgents: [],
}

let consoleErrorSpy: MockInstance

describe('TaskKanbanBoard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    vi.mocked(heartbeatService.getTaskBoard).mockResolvedValue({ teamId: 'team-1', tasks: mockTasks } satisfies TaskBoardResponse)
    vi.mocked(heartbeatService.addTask).mockResolvedValue({} as TeamTask)
    vi.mocked(heartbeatService.deleteTask).mockResolvedValue()
  })

  afterEach(() => {
    consoleErrorSpy.mockRestore()
  })

  it('renders 4 column headers', async () => {
    render(<TaskKanbanBoard {...defaultProps} />)

    await waitFor(() => {
      expect(screen.getByText('To Do')).toBeInTheDocument()
    })
    expect(screen.getByText('In Progress')).toBeInTheDocument()
    expect(screen.getByText('Blocked')).toBeInTheDocument()
    expect(screen.getByText('Done')).toBeInTheDocument()
  })

  it('distributes tasks into correct columns', async () => {
    render(<TaskKanbanBoard {...defaultProps} />)

    await waitFor(() => {
      expect(screen.getByText('First task')).toBeInTheDocument()
    })
    expect(screen.getByText('Second task')).toBeInTheDocument()
    expect(screen.getByText('Blocked task')).toBeInTheDocument()
  })

  it('shows loading spinner initially', () => {
    vi.mocked(heartbeatService.getTaskBoard).mockReturnValue(new Promise(() => {}))
    const { container } = render(<TaskKanbanBoard {...defaultProps} />)

    expect(container.querySelector('.animate-spin')).toBeTruthy()
  })

  it('shows error state with retry', async () => {
    vi.mocked(heartbeatService.getTaskBoard).mockRejectedValueOnce(new Error('Network error'))

    render(<TaskKanbanBoard {...defaultProps} />)

    await waitFor(() => {
      expect(screen.getByText(/Network error/)).toBeInTheDocument()
    })
    expect(screen.getByText('Retry')).toBeInTheDocument()
    expect(consoleErrorSpy).toHaveBeenCalledWith(
      '[TaskKanbanBoard] Failed to load tasks:',
      expect.any(Error)
    )
  })

  it('shows empty state when no tasks exist', async () => {
    vi.mocked(heartbeatService.getTaskBoard).mockResolvedValue({ teamId: 'team-1', tasks: [] })

    render(<TaskKanbanBoard {...defaultProps} />)

    await waitFor(() => {
      expect(screen.getByText(/No tasks yet/)).toBeInTheDocument()
    })
  })

  it('opens and closes the add task form', async () => {
    render(<TaskKanbanBoard {...defaultProps} />)

    await waitFor(() => {
      expect(screen.getByText('Add Task')).toBeInTheDocument()
    })

    // Open form
    fireEvent.click(screen.getByText('Add Task'))
    expect(screen.getByPlaceholderText('What needs to be done?')).toBeInTheDocument()

    // Close form
    fireEvent.click(screen.getByText('Close'))
    expect(screen.queryByPlaceholderText('What needs to be done?')).not.toBeInTheDocument()
  })

  it('creates a task via the add form', async () => {
    render(<TaskKanbanBoard {...defaultProps} />)

    await waitFor(() => {
      expect(screen.getByText('Add Task')).toBeInTheDocument()
    })

    fireEvent.click(screen.getByText('Add Task'))

    const input = screen.getByPlaceholderText('What needs to be done?')
    fireEvent.change(input, { target: { value: 'New task title' } })
    fireEvent.keyDown(input, { key: 'Enter' })

    await waitFor(() => {
      expect(heartbeatService.addTask).toHaveBeenCalledWith('team-1', expect.objectContaining({
        title: 'New task title',
        from: 'ui-user',
      }))
    })
  })

  it('shows delete confirmation when delete button is clicked', async () => {
    render(<TaskKanbanBoard {...defaultProps} />)

    await waitFor(() => {
      expect(screen.getByText('First task')).toBeInTheDocument()
    })

    // Click the first delete button
    const deleteButtons = screen.getAllByTitle('Delete task')
    expect(deleteButtons.length).toBeGreaterThan(0)
    fireEvent.click(deleteButtons[0] as HTMLElement)

    await waitFor(() => {
      expect(screen.getByTestId('confirm-dialog')).toBeInTheDocument()
    })
  })
})
