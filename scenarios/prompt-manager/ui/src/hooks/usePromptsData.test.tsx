/**
 * Tests for usePromptsData hook.
 *
 * Tests cover:
 * - Data fetching with React Query
 * - CRUD mutations
 * - Loading and error states
 * - Cache invalidation
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { usePromptsData } from './usePromptsData'
import * as promptService from '@/services/promptService'
import type { Prompt, CreatePromptRequest, UpdatePromptRequest } from '@/types'

// Mock the prompt service
vi.mock('@/services/promptService', () => ({
  getPrompts: vi.fn(),
  createPrompt: vi.fn(),
  updatePrompt: vi.fn(),
  updatePrompts: vi.fn(),
  deletePrompt: vi.fn(),
}))

// Helper to create a test prompt
function createTestPrompt(overrides: Partial<Prompt> = {}): Prompt {
  return {
    id: 'test-1',
    name: 'Test Prompt',
    description: 'A test description',
    content: '# Test content',
    modes: ['development', 'testing'],
    tags: ['tag1', 'tag2'],
    icon: 'file',
    targetToolId: 'tool-123',
    draft: false,
    folder: 'local',
    createdAt: '2025-01-01T00:00:00Z',
    updatedAt: '2025-01-01T00:00:00Z',
    usageCount: 5,
    lastUsed: '2025-01-01T12:00:00Z',
    effectivenessRating: 4.5,
    ...overrides,
  }
}

// Create a wrapper with QueryClient for each test
function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  })

  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    )
  }
}

describe('usePromptsData', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.resetAllMocks()
  })

  describe('data fetching', () => {
    it('should fetch prompts on mount', async () => {
      const mockPrompts = [createTestPrompt()]
      vi.mocked(promptService.getPrompts).mockResolvedValue(mockPrompts)

      const { result } = renderHook(() => usePromptsData(), {
        wrapper: createWrapper(),
      })

      // Initially loading
      expect(result.current.isLoading).toBe(true)

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.prompts).toEqual(mockPrompts)
      expect(promptService.getPrompts).toHaveBeenCalledWith(true)
    })

    it('should return empty array when no prompts', async () => {
      vi.mocked(promptService.getPrompts).mockResolvedValue([])

      const { result } = renderHook(() => usePromptsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.prompts).toEqual([])
    })

    it('should set error state on fetch failure', async () => {
      const error = new Error('Failed to fetch')
      vi.mocked(promptService.getPrompts).mockRejectedValue(error)

      const { result } = renderHook(() => usePromptsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isError).toBe(true)
      })

      expect(result.current.error).toBe(error)
    })

    it('should refetch data when refetch is called', async () => {
      const mockPrompts = [createTestPrompt()]
      vi.mocked(promptService.getPrompts).mockResolvedValue(mockPrompts)

      const { result } = renderHook(() => usePromptsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      // Call refetch
      result.current.refetch()

      await waitFor(() => {
        expect(promptService.getPrompts).toHaveBeenCalledTimes(2)
      })
    })
  })

  describe('createPrompt mutation', () => {
    it('should create a new prompt', async () => {
      vi.mocked(promptService.getPrompts).mockResolvedValue([])
      const newPrompt = createTestPrompt({ id: 'new-1', name: 'New Prompt' })
      vi.mocked(promptService.createPrompt).mockResolvedValue(newPrompt)

      const { result } = renderHook(() => usePromptsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      const request: CreatePromptRequest = {
        name: 'New Prompt',
        description: 'New description',
        content: 'New content',
        folder: 'local',
      }

      const created = await result.current.createPrompt(request)

      expect(created).toEqual(newPrompt)
      expect(promptService.createPrompt).toHaveBeenCalledWith(request)
    })

    it('should set isCreating during creation', async () => {
      vi.mocked(promptService.getPrompts).mockResolvedValue([])
      let resolveCreate: (value: Prompt) => void = () => {}
      vi.mocked(promptService.createPrompt).mockImplementation(
        () => new Promise((resolve) => { resolveCreate = resolve })
      )

      const { result } = renderHook(() => usePromptsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      const createPromise = result.current.createPrompt({
        name: 'New',
        description: '',
        content: 'Content',
        folder: 'local',
      })

      await waitFor(() => {
        expect(result.current.isCreating).toBe(true)
      })

      resolveCreate(createTestPrompt())
      await createPromise

      await waitFor(() => {
        expect(result.current.isCreating).toBe(false)
      })
    })
  })

  describe('updatePrompt mutation', () => {
    it('should update a single prompt', async () => {
      const original = createTestPrompt()
      vi.mocked(promptService.getPrompts).mockResolvedValue([original])
      const updated = { ...original, name: 'Updated Name' }
      vi.mocked(promptService.updatePrompt).mockResolvedValue(updated)

      const { result } = renderHook(() => usePromptsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      const updates: UpdatePromptRequest = { name: 'Updated Name' }
      const updatedPrompt = await result.current.updatePrompt(original.id, updates)

      expect(updatedPrompt).toEqual(updated)
      expect(promptService.updatePrompt).toHaveBeenCalledWith(original.id, updates)
    })

    it('should set isUpdating during update', async () => {
      vi.mocked(promptService.getPrompts).mockResolvedValue([createTestPrompt()])
      let resolveUpdate: (value: Prompt) => void = () => {}
      vi.mocked(promptService.updatePrompt).mockImplementation(
        () => new Promise((resolve) => { resolveUpdate = resolve })
      )

      const { result } = renderHook(() => usePromptsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      const updatePromise = result.current.updatePrompt('test-1', { name: 'New' })

      await waitFor(() => {
        expect(result.current.isUpdating).toBe(true)
      })

      resolveUpdate(createTestPrompt({ name: 'New' }))
      await updatePromise

      await waitFor(() => {
        expect(result.current.isUpdating).toBe(false)
      })
    })
  })

  describe('updatePrompts batch mutation', () => {
    it('should update multiple prompts', async () => {
      const prompt1 = createTestPrompt({ id: 'p1', name: 'Prompt 1' })
      const prompt2 = createTestPrompt({ id: 'p2', name: 'Prompt 2' })
      vi.mocked(promptService.getPrompts).mockResolvedValue([prompt1, prompt2])

      const updatedResults = new Map<string, Prompt | Error>([
        ['p1', { ...prompt1, name: 'Updated 1' }],
        ['p2', { ...prompt2, name: 'Updated 2' }],
      ])
      vi.mocked(promptService.updatePrompts).mockResolvedValue(updatedResults)

      const { result } = renderHook(() => usePromptsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      const updates = new Map<string, UpdatePromptRequest>([
        ['p1', { name: 'Updated 1' }],
        ['p2', { name: 'Updated 2' }],
      ])

      const results = await result.current.updatePrompts(updates)

      expect(results).toEqual(updatedResults)
      expect(promptService.updatePrompts).toHaveBeenCalledWith(updates)
    })

    it('should set isUpdating during batch update', async () => {
      vi.mocked(promptService.getPrompts).mockResolvedValue([createTestPrompt()])
      let resolveUpdate: (value: Map<string, Prompt | Error>) => void = () => {}
      vi.mocked(promptService.updatePrompts).mockImplementation(
        () => new Promise((resolve) => { resolveUpdate = resolve })
      )

      const { result } = renderHook(() => usePromptsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      const updatePromise = result.current.updatePrompts(
        new Map([['test-1', { name: 'New' }]])
      )

      await waitFor(() => {
        expect(result.current.isUpdating).toBe(true)
      })

      resolveUpdate(new Map())
      await updatePromise

      await waitFor(() => {
        expect(result.current.isUpdating).toBe(false)
      })
    })
  })

  describe('deletePrompt mutation', () => {
    it('should delete a prompt', async () => {
      const prompt = createTestPrompt()
      vi.mocked(promptService.getPrompts).mockResolvedValue([prompt])
      vi.mocked(promptService.deletePrompt).mockResolvedValue(undefined)

      const { result } = renderHook(() => usePromptsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      await result.current.deletePrompt(prompt.id)

      expect(promptService.deletePrompt).toHaveBeenCalledWith(prompt.id)
    })

    it('should set isDeleting during deletion', async () => {
      vi.mocked(promptService.getPrompts).mockResolvedValue([createTestPrompt()])
      let resolveDelete: () => void = () => {}
      vi.mocked(promptService.deletePrompt).mockImplementation(
        () => new Promise((resolve) => { resolveDelete = resolve })
      )

      const { result } = renderHook(() => usePromptsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      const deletePromise = result.current.deletePrompt('test-1')

      await waitFor(() => {
        expect(result.current.isDeleting).toBe(true)
      })

      resolveDelete()
      await deletePromise

      await waitFor(() => {
        expect(result.current.isDeleting).toBe(false)
      })
    })
  })

  describe('error handling', () => {
    it('should handle create mutation error', async () => {
      vi.mocked(promptService.getPrompts).mockResolvedValue([])
      const error = new Error('Create failed')
      vi.mocked(promptService.createPrompt).mockRejectedValue(error)

      const { result } = renderHook(() => usePromptsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      await expect(
        result.current.createPrompt({
          name: 'Test',
          description: '',
          content: 'Content',
          folder: 'local',
        })
      ).rejects.toThrow('Create failed')
    })

    it('should handle update mutation error', async () => {
      vi.mocked(promptService.getPrompts).mockResolvedValue([createTestPrompt()])
      const error = new Error('Update failed')
      vi.mocked(promptService.updatePrompt).mockRejectedValue(error)

      const { result } = renderHook(() => usePromptsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      await expect(
        result.current.updatePrompt('test-1', { name: 'New' })
      ).rejects.toThrow('Update failed')
    })

    it('should handle delete mutation error', async () => {
      vi.mocked(promptService.getPrompts).mockResolvedValue([createTestPrompt()])
      const error = new Error('Delete failed')
      vi.mocked(promptService.deletePrompt).mockRejectedValue(error)

      const { result } = renderHook(() => usePromptsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      await expect(result.current.deletePrompt('test-1')).rejects.toThrow(
        'Delete failed'
      )
    })
  })
})
