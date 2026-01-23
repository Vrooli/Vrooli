/**
 * Tests for useSkillsData hook.
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
import { useSkillsData } from './useSkillsData'
import * as skillService from '@/services/skillService'
import type { Skill, CreateSkillRequest, UpdateSkillRequest } from '@/types'

// Mock the skill service
vi.mock('@/services/skillService', () => ({
  getSkills: vi.fn(),
  createSkill: vi.fn(),
  updateSkill: vi.fn(),
  updateSkills: vi.fn(),
  deleteSkill: vi.fn(),
}))

// Helper to create a test skill
function createTestSkill(overrides: Partial<Skill> = {}): Skill {
  return {
    id: 'test-1',
    name: 'Test Skill',
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

describe('useSkillsData', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.resetAllMocks()
  })

  describe('data fetching', () => {
    it('should fetch skills on mount', async () => {
      const mockSkills = [createTestSkill()]
      vi.mocked(skillService.getSkills).mockResolvedValue(mockSkills)

      const { result } = renderHook(() => useSkillsData(), {
        wrapper: createWrapper(),
      })

      // Initially loading
      expect(result.current.isLoading).toBe(true)

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.skills).toEqual(mockSkills)
      expect(skillService.getSkills).toHaveBeenCalledWith(true)
    })

    it('should return empty array when no skills', async () => {
      vi.mocked(skillService.getSkills).mockResolvedValue([])

      const { result } = renderHook(() => useSkillsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.skills).toEqual([])
    })

    it('should set error state on fetch failure', async () => {
      const error = new Error('Failed to fetch')
      vi.mocked(skillService.getSkills).mockRejectedValue(error)

      const { result } = renderHook(() => useSkillsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isError).toBe(true)
      })

      expect(result.current.error).toBe(error)
    })

    it('should refetch data when refetch is called', async () => {
      const mockSkills = [createTestSkill()]
      vi.mocked(skillService.getSkills).mockResolvedValue(mockSkills)

      const { result } = renderHook(() => useSkillsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      // Call refetch
      result.current.refetch()

      await waitFor(() => {
        expect(skillService.getSkills).toHaveBeenCalledTimes(2)
      })
    })
  })

  describe('createSkill mutation', () => {
    it('should create a new skill', async () => {
      vi.mocked(skillService.getSkills).mockResolvedValue([])
      const newSkill = createTestSkill({ id: 'new-1', name: 'New Skill' })
      vi.mocked(skillService.createSkill).mockResolvedValue(newSkill)

      const { result } = renderHook(() => useSkillsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      const request: CreateSkillRequest = {
        name: 'New Skill',
        description: 'New description',
        content: 'New content',
        folder: 'local',
      }

      const created = await result.current.createSkill(request)

      expect(created).toEqual(newSkill)
      expect(skillService.createSkill).toHaveBeenCalledWith(request)
    })

    it('should set isCreating during creation', async () => {
      vi.mocked(skillService.getSkills).mockResolvedValue([])
      let resolveCreate: (value: Skill) => void = () => {}
      vi.mocked(skillService.createSkill).mockImplementation(
        () => new Promise((resolve) => { resolveCreate = resolve })
      )

      const { result } = renderHook(() => useSkillsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      const createPromise = result.current.createSkill({
        name: 'New',
        description: '',
        content: 'Content',
        folder: 'local',
      })

      await waitFor(() => {
        expect(result.current.isCreating).toBe(true)
      })

      resolveCreate(createTestSkill())
      await createPromise

      await waitFor(() => {
        expect(result.current.isCreating).toBe(false)
      })
    })
  })

  describe('updateSkill mutation', () => {
    it('should update a single skill', async () => {
      const original = createTestSkill()
      vi.mocked(skillService.getSkills).mockResolvedValue([original])
      const updated = { ...original, name: 'Updated Name' }
      vi.mocked(skillService.updateSkill).mockResolvedValue(updated)

      const { result } = renderHook(() => useSkillsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      const updates: UpdateSkillRequest = { name: 'Updated Name' }
      const updatedSkill = await result.current.updateSkill(original.id, updates)

      expect(updatedSkill).toEqual(updated)
      expect(skillService.updateSkill).toHaveBeenCalledWith(original.id, updates)
    })

    it('should set isUpdating during update', async () => {
      vi.mocked(skillService.getSkills).mockResolvedValue([createTestSkill()])
      let resolveUpdate: (value: Skill) => void = () => {}
      vi.mocked(skillService.updateSkill).mockImplementation(
        () => new Promise((resolve) => { resolveUpdate = resolve })
      )

      const { result } = renderHook(() => useSkillsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      const updatePromise = result.current.updateSkill('test-1', { name: 'New' })

      await waitFor(() => {
        expect(result.current.isUpdating).toBe(true)
      })

      resolveUpdate(createTestSkill({ name: 'New' }))
      await updatePromise

      await waitFor(() => {
        expect(result.current.isUpdating).toBe(false)
      })
    })
  })

  describe('updateSkills batch mutation', () => {
    it('should update multiple skills', async () => {
      const skill1 = createTestSkill({ id: 'p1', name: 'Skill 1' })
      const skill2 = createTestSkill({ id: 'p2', name: 'Skill 2' })
      vi.mocked(skillService.getSkills).mockResolvedValue([skill1, skill2])

      const updatedResults = new Map<string, Skill | Error>([
        ['p1', { ...skill1, name: 'Updated 1' }],
        ['p2', { ...skill2, name: 'Updated 2' }],
      ])
      vi.mocked(skillService.updateSkills).mockResolvedValue(updatedResults)

      const { result } = renderHook(() => useSkillsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      const updates = new Map<string, UpdateSkillRequest>([
        ['p1', { name: 'Updated 1' }],
        ['p2', { name: 'Updated 2' }],
      ])

      const results = await result.current.updateSkills(updates)

      expect(results).toEqual(updatedResults)
      expect(skillService.updateSkills).toHaveBeenCalledWith(updates)
    })

    it('should set isUpdating during batch update', async () => {
      vi.mocked(skillService.getSkills).mockResolvedValue([createTestSkill()])
      let resolveUpdate: (value: Map<string, Skill | Error>) => void = () => {}
      vi.mocked(skillService.updateSkills).mockImplementation(
        () => new Promise((resolve) => { resolveUpdate = resolve })
      )

      const { result } = renderHook(() => useSkillsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      const updatePromise = result.current.updateSkills(
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

  describe('deleteSkill mutation', () => {
    it('should delete a skill', async () => {
      const skill = createTestSkill()
      vi.mocked(skillService.getSkills).mockResolvedValue([skill])
      vi.mocked(skillService.deleteSkill).mockResolvedValue(undefined)

      const { result } = renderHook(() => useSkillsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      await result.current.deleteSkill(skill.id)

      expect(skillService.deleteSkill).toHaveBeenCalledWith(skill.id)
    })

    it('should set isDeleting during deletion', async () => {
      vi.mocked(skillService.getSkills).mockResolvedValue([createTestSkill()])
      let resolveDelete: () => void = () => {}
      vi.mocked(skillService.deleteSkill).mockImplementation(
        () => new Promise((resolve) => { resolveDelete = resolve })
      )

      const { result } = renderHook(() => useSkillsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      const deletePromise = result.current.deleteSkill('test-1')

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
      vi.mocked(skillService.getSkills).mockResolvedValue([])
      const error = new Error('Create failed')
      vi.mocked(skillService.createSkill).mockRejectedValue(error)

      const { result } = renderHook(() => useSkillsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      await expect(
        result.current.createSkill({
          name: 'Test',
          description: '',
          content: 'Content',
          folder: 'local',
        })
      ).rejects.toThrow('Create failed')
    })

    it('should handle update mutation error', async () => {
      vi.mocked(skillService.getSkills).mockResolvedValue([createTestSkill()])
      const error = new Error('Update failed')
      vi.mocked(skillService.updateSkill).mockRejectedValue(error)

      const { result } = renderHook(() => useSkillsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      await expect(
        result.current.updateSkill('test-1', { name: 'New' })
      ).rejects.toThrow('Update failed')
    })

    it('should handle delete mutation error', async () => {
      vi.mocked(skillService.getSkills).mockResolvedValue([createTestSkill()])
      const error = new Error('Delete failed')
      vi.mocked(skillService.deleteSkill).mockRejectedValue(error)

      const { result } = renderHook(() => useSkillsData(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      await expect(result.current.deleteSkill('test-1')).rejects.toThrow(
        'Delete failed'
      )
    })
  })
})
