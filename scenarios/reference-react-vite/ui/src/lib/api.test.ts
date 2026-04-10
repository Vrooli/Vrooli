// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#api-client-tests
// [REQ:P0-002a] API Error Handling - Tests ApiError class and handleResponse error parsing
// [REQ:P1-003a] Error Recovery Hints - Tests error recovery message propagation
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { ApiError } from './api';

// Type declaration for globalThis in test environment
declare const globalThis: {
  fetch: typeof fetch;
};

// =============================================================================
// ApiError Class Tests
// [REQ:P0-002a] Tests the ApiError class structure and properties
// =============================================================================

describe('ApiError', () => {
  describe('construction', () => {
    it('creates an error with all properties', () => {
      const error = new ApiError(
        'Task not found',
        'NOT_FOUND',
        404,
        'Verify the resource ID is correct',
        false
      );

      expect(error.message).toBe('Task not found');
      expect(error.code).toBe('NOT_FOUND');
      expect(error.status).toBe(404);
      expect(error.recovery).toBe('Verify the resource ID is correct');
      expect(error.retryable).toBe(false);
      expect(error.name).toBe('ApiError');
    });

    it('handles optional properties', () => {
      const error = new ApiError('Server error', 'INTERNAL_ERROR', 500);

      expect(error.message).toBe('Server error');
      expect(error.code).toBe('INTERNAL_ERROR');
      expect(error.status).toBe(500);
      expect(error.recovery).toBeUndefined();
      expect(error.retryable).toBeUndefined();
    });

    it('extends Error class', () => {
      const error = new ApiError('Test error', 'TEST', 400);
      expect(error).toBeInstanceOf(Error);
      expect(error).toBeInstanceOf(ApiError);
    });

    it('preserves stack trace', () => {
      const error = new ApiError('Test error', 'TEST', 400);
      expect(error.stack).toBeDefined();
      expect(error.stack).toContain('ApiError');
    });
  });

  describe('error codes', () => {
    const errorCases = [
      { code: 'BAD_REQUEST', status: 400, retryable: false },
      { code: 'UNAUTHORIZED', status: 401, retryable: false },
      { code: 'NOT_FOUND', status: 404, retryable: false },
      { code: 'VALIDATION_ERROR', status: 422, retryable: false },
      { code: 'INTERNAL_ERROR', status: 500, retryable: true },
      { code: 'SERVICE_UNAVAILABLE', status: 503, retryable: true },
    ];

    errorCases.forEach(({ code, status, retryable }) => {
      it(`handles ${code} error`, () => {
        const error = new ApiError('Test', code, status, undefined, retryable);
        expect(error.code).toBe(code);
        expect(error.status).toBe(status);
        expect(error.retryable).toBe(retryable);
      });
    });
  });

  describe('recovery hints', () => {
    it('preserves recovery hint for user guidance', () => {
      const error = new ApiError(
        'Validation failed',
        'VALIDATION_ERROR',
        422,
        'Check the field values and try again',
        false
      );

      expect(error.recovery).toBe('Check the field values and try again');
    });

    it('can determine if error is retryable', () => {
      const retryableError = new ApiError('Timeout', 'TIMEOUT', 504, undefined, true);
      const nonRetryableError = new ApiError('Not found', 'NOT_FOUND', 404, undefined, false);

      expect(retryableError.retryable).toBe(true);
      expect(nonRetryableError.retryable).toBe(false);
    });
  });
});

// =============================================================================
// Fetch Response Handling Tests
// [REQ:P0-002a] Tests error response parsing from API
// =============================================================================

describe('API Response Handling', () => {
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    vi.clearAllMocks();
  });

  describe('error parsing', () => {
    it('parses structured error response', async () => {
      const errorResponse = {
        code: 'NOT_FOUND',
        message: 'Task not found',
        recovery: 'Verify the task ID',
        retryable: false,
      };

      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: false,
        status: 404,
        json: () => Promise.resolve(errorResponse),
      } as Response);

      // Import dynamically to use mocked fetch
      const { fetchTask } = await import('./api');

      await expect(fetchTask('nonexistent')).rejects.toMatchObject({
        message: 'Task not found',
        code: 'NOT_FOUND',
        status: 404,
        recovery: 'Verify the task ID',
        retryable: false,
      });
    });

    it('handles non-JSON error response', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: () => Promise.reject(new Error('Invalid JSON')),
      } as Response);

      const { fetchTask } = await import('./api');

      await expect(fetchTask('test')).rejects.toMatchObject({
        code: 'UNKNOWN_ERROR',
        status: 500,
      });
    });

    it('handles 204 No Content response', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
        status: 204,
        json: () => Promise.reject(new Error('No content')),
      } as Response);

      const { deleteTask } = await import('./api');
      const result = await deleteTask('test-id');

      expect(result).toBeUndefined();
    });
  });

  describe('request construction', () => {
    it('includes Content-Type header', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ status: 'healthy', service: 'test', timestamp: '' }),
      } as Response);

      const { fetchHealth } = await import('./api');
      await fetchHealth();

      expect(vi.mocked(globalThis.fetch)).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
          }),
        })
      );
    });

    it('disables cache for health checks', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ status: 'healthy', service: 'test', timestamp: '' }),
      } as Response);

      const { fetchHealth } = await import('./api');
      await fetchHealth();

      expect(vi.mocked(globalThis.fetch)).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          cache: 'no-store',
        })
      );
    });

    it('sends POST with JSON body', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
        status: 201,
        json: () => Promise.resolve({
          id: '1',
          title: 'Test Task',
          status: 'pending',
          priority: 2,
          created_at: '',
          updated_at: '',
        }),
      } as Response);

      const { createTask } = await import('./api');
      await createTask({ title: 'Test Task' });

      expect(vi.mocked(globalThis.fetch)).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ title: 'Test Task' }),
        })
      );
    });

    it('sends PATCH with partial update', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({
          id: '1',
          title: 'Updated',
          status: 'in_progress',
          priority: 2,
          created_at: '',
          updated_at: '',
        }),
      } as Response);

      const { updateTask } = await import('./api');
      await updateTask('1', { status: 'in_progress' });

      expect(vi.mocked(globalThis.fetch)).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          method: 'PATCH',
          body: JSON.stringify({ status: 'in_progress' }),
        })
      );
    });

    it('sends DELETE request', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
        status: 204,
      } as Response);

      const { deleteTask } = await import('./api');
      await deleteTask('1');

      expect(vi.mocked(globalThis.fetch)).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          method: 'DELETE',
        })
      );
    });
  });

  describe('query parameter handling', () => {
    it('builds query params for list endpoints', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ data: [], pagination: { total: 0, limit: 10, offset: 0 } }),
      } as Response);

      const { fetchTasks } = await import('./api');
      await fetchTasks({ limit: 10, offset: 20, status: 'pending' });

      const callUrl = vi.mocked(globalThis.fetch).mock.calls[0]?.[0];
      expect(callUrl).toContain('limit=10');
      expect(callUrl).toContain('offset=20');
      expect(callUrl).toContain('status=pending');
    });

    it('omits undefined query params', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ data: [], pagination: { total: 0, limit: 20, offset: 0 } }),
      } as Response);

      const { fetchTasks } = await import('./api');
      await fetchTasks({ status: 'completed' });

      const callUrl = vi.mocked(globalThis.fetch).mock.calls[0]?.[0];
      expect(callUrl).toContain('status=completed');
      expect(callUrl).not.toContain('limit=');
      expect(callUrl).not.toContain('offset=');
    });

    it('handles projects query params', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ data: [], pagination: { total: 0, limit: 5, offset: 0 } }),
      } as Response);

      const { fetchProjects } = await import('./api');
      await fetchProjects({ limit: 5, status: 'active' });

      const callUrl = vi.mocked(globalThis.fetch).mock.calls[0]?.[0];
      expect(callUrl).toContain('limit=5');
      expect(callUrl).toContain('status=active');
    });
  });

  describe('CRUD operations', () => {
    it('creates project with all fields', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
        status: 201,
        json: () => Promise.resolve({
          id: '1',
          name: 'New Project',
          description: 'Description',
          color: '#ff0000',
          status: 'active',
          created_at: '',
          updated_at: '',
        }),
      } as Response);

      const { createProject } = await import('./api');
      const result = await createProject({
        name: 'New Project',
        description: 'Description',
        color: '#ff0000',
      });

      expect(result.name).toBe('New Project');
      expect(vi.mocked(globalThis.fetch)).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({
            name: 'New Project',
            description: 'Description',
            color: '#ff0000',
          }),
        })
      );
    });

    it('updates project status', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({
          id: '1',
          name: 'Project',
          status: 'complete',
          created_at: '',
          updated_at: '',
        }),
      } as Response);

      const { updateProject } = await import('./api');
      await updateProject('1', { status: 'complete' });

      expect(vi.mocked(globalThis.fetch)).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          method: 'PATCH',
        })
      );
    });

    it('deletes project', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
        status: 204,
      } as Response);

      const { deleteProject } = await import('./api');
      await deleteProject('1');

      expect(vi.mocked(globalThis.fetch)).toHaveBeenCalledWith(
        expect.stringContaining('/projects/1'),
        expect.objectContaining({ method: 'DELETE' })
      );
    });
  });

  describe('notes operations', () => {
    it('fetches notes for task', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({
          data: [{ id: '1', content: 'Note', task_id: 'task-1', created_at: '', updated_at: '' }],
          pagination: { total: 1, limit: 20, offset: 0 },
        }),
      } as Response);

      const { fetchNotes } = await import('./api');
      const result = await fetchNotes('task-1');

      expect(result.data).toHaveLength(1);
      expect(vi.mocked(globalThis.fetch)).toHaveBeenCalledWith(
        expect.stringContaining('/tasks/task-1/notes'),
        expect.any(Object)
      );
    });

    it('creates note on task', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
        status: 201,
        json: () => Promise.resolve({
          id: '1',
          content: 'New note',
          task_id: 'task-1',
          created_at: '',
          updated_at: '',
        }),
      } as Response);

      const { createNote } = await import('./api');
      await createNote('task-1', { content: 'New note' });

      expect(vi.mocked(globalThis.fetch)).toHaveBeenCalledWith(
        expect.stringContaining('/tasks/task-1/notes'),
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ content: 'New note' }),
        })
      );
    });

    it('updates note content', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({
          id: '1',
          content: 'Updated note',
          task_id: 'task-1',
          created_at: '',
          updated_at: '',
        }),
      } as Response);

      const { updateNote } = await import('./api');
      await updateNote('1', { content: 'Updated note' });

      expect(vi.mocked(globalThis.fetch)).toHaveBeenCalledWith(
        expect.stringContaining('/notes/1'),
        expect.objectContaining({
          method: 'PATCH',
        })
      );
    });

    it('deletes note', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok: true,
        status: 204,
      } as Response);

      const { deleteNote } = await import('./api');
      await deleteNote('1');

      expect(vi.mocked(globalThis.fetch)).toHaveBeenCalledWith(
        expect.stringContaining('/notes/1'),
        expect.objectContaining({ method: 'DELETE' })
      );
    });
  });
});
