// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#ui-test-utilities
// Test data factories for creating mock data
//
// IMPORTANT: Types are imported from lib/api.ts to maintain a single source of truth.
// This prevents type drift between production code and test utilities.

import type {
  HealthResponse,
  Task,
  Project,
  Note,
  PaginationMeta,
  ListResponse,
} from '../lib/api';

// Re-export types for convenient test imports
export type { HealthResponse, Task, Project, Note, PaginationMeta, ListResponse };

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
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    ...overrides,
  };
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

/**
 * Creates a mock list response with pagination.
 */
export function createMockListResponse<T>(
  data: T[],
  pagination: Partial<PaginationMeta> = {}
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
