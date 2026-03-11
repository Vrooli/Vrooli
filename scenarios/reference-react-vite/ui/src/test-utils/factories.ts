// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#ui-test-utilities
// Test data factories for creating mock data

/**
 * Creates a mock health response matching the API's actual response shape.
 */
export function createMockHealthResponse(overrides: Partial<HealthResponse> = {}): HealthResponse {
  return {
    status: 'healthy',
    service: 'reference-react-vite',
    timestamp: new Date().toISOString(),
    ...overrides,
  };
}

export interface HealthResponse {
  status: string;
  service: string;
  timestamp: string;
}

/**
 * Creates a mock task.
 */
export function createMockTask(overrides: Partial<Task> = {}): Task {
  return {
    id: 'task-001',
    title: 'Test Task',
    description: 'A test task description',
    status: 'pending',
    priority: 2,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    ...overrides,
  };
}

export interface Task {
  id: string;
  project_id?: string;
  title: string;
  description?: string;
  status: 'pending' | 'in_progress' | 'completed' | 'archived';
  priority: 1 | 2 | 3;
  due_date?: string;
  created_at: string;
  updated_at: string;
}

/**
 * Creates a mock project.
 */
export function createMockProject(overrides: Partial<Project> = {}): Project {
  return {
    id: 'proj-001',
    name: 'Test Project',
    description: 'A test project description',
    status: 'active',
    color: '#3498db',
    task_count: 0,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    ...overrides,
  };
}

export interface Project {
  id: string;
  name: string;
  description?: string;
  status: 'active' | 'paused' | 'complete' | 'archived';
  color?: string;
  task_count?: number;
  created_at: string;
  updated_at: string;
}

/**
 * Creates a mock note.
 */
export function createMockNote(overrides: Partial<Note> = {}): Note {
  return {
    id: 'note-001',
    task_id: 'task-001',
    content: 'Test note content',
    author: 'Test Author',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    ...overrides,
  };
}

export interface Note {
  id: string;
  task_id: string;
  content: string;
  author?: string;
  created_at: string;
  updated_at: string;
}

/**
 * Creates a mock list response with pagination.
 */
export function createMockListResponse<T>(
  data: T[],
  pagination: Partial<Pagination> = {}
): ListResponse<T> {
  return {
    data,
    pagination: {
      total: data.length,
      limit: 20,
      offset: 0,
      ...pagination,
    },
  };
}

export interface Pagination {
  total: number;
  limit: number;
  offset: number;
}

export interface ListResponse<T> {
  data: T[];
  pagination: Pagination;
}
