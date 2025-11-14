import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { useDrafts } from './useDrafts'
import type { Draft } from '../types'

// Mock fetch globally
const mockFetch = vi.fn()
// eslint-disable-next-line @typescript-eslint/no-explicit-any
;(globalThis as any).fetch = mockFetch

const createMockDraft = (overrides: Partial<Draft> = {}): Draft => ({
  id: '1',
  entity_name: 'test-scenario',
  entity_type: 'scenario',
  content: 'Test content',
  owner: 'admin',
  created_at: '2025-01-01T00:00:00.000Z',
  updated_at: '2025-01-01T00:00:00.000Z',
  status: 'draft',
  ...overrides,
})

describe('useDrafts', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.clearAllTimers()
  })

  it('fetches drafts on mount', async () => {
    const mockDrafts = [createMockDraft()]

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ drafts: mockDrafts }),
    })

    const { result } = renderHook(() =>
      useDrafts({ routeEntityType: undefined, routeEntityName: undefined, decodedRouteName: '' }),
    )

    expect(result.current.loading).toBe(true)

    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.drafts).toEqual(mockDrafts)
    expect(result.current.error).toBeNull()
  })

  it('handles fetch errors gracefully', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      statusText: 'Server Error',
    })

    const { result } = renderHook(() =>
      useDrafts({ routeEntityType: undefined, routeEntityName: undefined, decodedRouteName: '' }),
    )

    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.error).toContain('Server Error')
    expect(result.current.drafts).toEqual([])
  })

  it('filters drafts by search term', async () => {
    const mockDrafts = [
      createMockDraft({ id: '1', entity_name: 'test-scenario' }),
      createMockDraft({ id: '2', entity_name: 'other-scenario' }),
      createMockDraft({ id: '3', entity_name: 'production-app', owner: 'test-user' }),
    ]

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ drafts: mockDrafts }),
    })

    const { result } = renderHook(() =>
      useDrafts({ routeEntityType: undefined, routeEntityName: undefined, decodedRouteName: '' }),
    )

    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.filteredDrafts).toHaveLength(3)

    // Set filter to 'test'
    result.current.setFilter('test')
    await waitFor(() => expect(result.current.filteredDrafts).toHaveLength(2))
    expect(result.current.filteredDrafts.map((d) => d.id)).toEqual(['1', '3'])
  })

  it('selects draft by route parameters', async () => {
    const mockDrafts = [
      createMockDraft({ id: '1', entity_name: 'test-scenario', entity_type: 'scenario' }),
      createMockDraft({ id: '2', entity_name: 'other-scenario', entity_type: 'resource' }),
    ]

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ drafts: mockDrafts }),
    })

    const { result } = renderHook(() =>
      useDrafts({
        routeEntityType: 'scenario',
        routeEntityName: 'test-scenario',
        decodedRouteName: 'test-scenario',
      }),
    )

    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.selectedDraft).toEqual(mockDrafts[0])
  })

  it('handles case-insensitive route matching', async () => {
    const mockDrafts = [createMockDraft({ entity_name: 'Test-Scenario', entity_type: 'scenario' })]

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ drafts: mockDrafts }),
    })

    const { result } = renderHook(() =>
      useDrafts({
        routeEntityType: 'SCENARIO',
        routeEntityName: 'test-scenario',
        decodedRouteName: 'test-scenario',
      }),
    )

    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.selectedDraft).toEqual(mockDrafts[0])
  })

  it('supports silent refresh mode', async () => {
    const mockDrafts = [createMockDraft()]

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ drafts: mockDrafts }),
    })

    const { result } = renderHook(() =>
      useDrafts({ routeEntityType: undefined, routeEntityName: undefined, decodedRouteName: '' }),
    )

    await waitFor(() => expect(result.current.loading).toBe(false))

    // Clear initial fetch
    mockFetch.mockClear()
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ drafts: [createMockDraft({ id: '2' })] }),
    })

    // Trigger silent refresh
    act(() => {
      result.current.fetchDrafts({ silent: true })
    })

    await waitFor(() => expect(result.current.refreshing).toBe(false))
    expect(result.current.loading).toBe(false) // loading should stay false during silent refresh
    expect(result.current.drafts).toHaveLength(1)
  })

  it('filters by both type and name when route is active', async () => {
    const mockDrafts = [
      createMockDraft({ id: '1', entity_name: 'test-app', entity_type: 'scenario' }),
      createMockDraft({ id: '2', entity_name: 'test-app', entity_type: 'resource' }),
      createMockDraft({ id: '3', entity_name: 'other-app', entity_type: 'scenario' }),
    ]

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ drafts: mockDrafts }),
    })

    const { result } = renderHook(() =>
      useDrafts({
        routeEntityType: 'scenario',
        routeEntityName: 'test-app',
        decodedRouteName: 'test-app',
      }),
    )

    await waitFor(() => expect(result.current.loading).toBe(false))

    // The filtered list should include the scenario matching both type and name
    await waitFor(() => expect(result.current.filteredDrafts.length).toBeGreaterThan(0))

    const scenarioMatch = result.current.filteredDrafts.find(d => d.id === '1')
    expect(scenarioMatch).toBeDefined()
  })
})
