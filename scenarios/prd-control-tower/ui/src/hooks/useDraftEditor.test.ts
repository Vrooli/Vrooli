import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { useDraftEditor } from './useDraftEditor'
import type { Draft } from '../types'
import { ViewModes } from '../types'

// Mock fetch globally
const mockFetch = vi.fn()
// eslint-disable-next-line @typescript-eslint/no-explicit-any
;(globalThis as any).fetch = mockFetch

const createMockDraft = (overrides: Partial<Draft> = {}): Draft => ({
  id: 'draft-1',
  entity_name: 'test-scenario',
  entity_type: 'scenario',
  content: 'Initial content',
  owner: 'admin',
  created_at: '2025-01-01T00:00:00.000Z',
  updated_at: '2025-01-01T00:00:00.000Z',
  status: 'draft',
  ...overrides,
})

const mockConfirm = vi.fn()
const mockFetchDrafts = vi.fn()

describe('useDraftEditor', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.clearAllTimers()
    vi.useRealTimers()
  })

  it('initializes with empty state when no draft selected', () => {
    const { result } = renderHook(() =>
      useDraftEditor({
        selectedDraft: null,
        fetchDrafts: mockFetchDrafts,
        confirm: mockConfirm,
      }),
    )

    expect(result.current.editorContent).toBe('')
    expect(result.current.hasUnsavedChanges).toBe(false)
    expect(result.current.viewMode).toBe(ViewModes.SPLIT)
  })

  it('loads draft content when draft is selected', () => {
    const draft = createMockDraft({ content: 'Test content' })

    const { result } = renderHook(() =>
      useDraftEditor({
        selectedDraft: draft,
        fetchDrafts: mockFetchDrafts,
        confirm: mockConfirm,
      }),
    )

    expect(result.current.editorContent).toBe('Test content')
    expect(result.current.hasUnsavedChanges).toBe(false)
  })

  it('tracks unsaved changes when content is modified', () => {
    const draft = createMockDraft({ content: 'Original content' })

    const { result } = renderHook(() =>
      useDraftEditor({
        selectedDraft: draft,
        fetchDrafts: mockFetchDrafts,
        confirm: mockConfirm,
      }),
    )

    expect(result.current.hasUnsavedChanges).toBe(false)

    act(() => {
      result.current.handleEditorChange('Modified content')
    })

    expect(result.current.editorContent).toBe('Modified content')
    expect(result.current.hasUnsavedChanges).toBe(true)
  })

  it('saves draft successfully', async () => {
    const draft = createMockDraft()

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ success: true }),
    })

    const { result } = renderHook(() =>
      useDraftEditor({
        selectedDraft: draft,
        fetchDrafts: mockFetchDrafts,
        confirm: mockConfirm,
      }),
    )

    act(() => {
      result.current.handleEditorChange('New content')
    })

    expect(result.current.hasUnsavedChanges).toBe(true)

    await act(async () => {
      await result.current.handleSaveDraft()
    })

    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining(`/drafts/${draft.id}`),
      expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify({ content: 'New content' }),
      }),
    )

    expect(result.current.hasUnsavedChanges).toBe(false)
    expect(result.current.saveStatus).toEqual({ type: 'success', message: 'Draft saved successfully.' })
    expect(mockFetchDrafts).toHaveBeenCalledWith({ silent: true })
  })

  it('handles save errors', async () => {
    const draft = createMockDraft()

    mockFetch.mockResolvedValueOnce({
      ok: false,
      statusText: 'Internal Server Error',
    })

    const { result } = renderHook(() =>
      useDraftEditor({
        selectedDraft: draft,
        fetchDrafts: mockFetchDrafts,
        confirm: mockConfirm,
      }),
    )

    await act(async () => {
      await result.current.handleSaveDraft()
    })

    expect(result.current.saveStatus?.type).toBe('error')
    expect(result.current.saveStatus?.message).toContain('Internal Server Error')
  })

  it('discards changes with confirmation', async () => {
    const draft = createMockDraft({ content: 'Original' })

    mockConfirm.mockResolvedValueOnce(true)

    const { result } = renderHook(() =>
      useDraftEditor({
        selectedDraft: draft,
        fetchDrafts: mockFetchDrafts,
        confirm: mockConfirm,
      }),
    )

    act(() => {
      result.current.handleEditorChange('Modified')
    })

    expect(result.current.hasUnsavedChanges).toBe(true)

    await act(async () => {
      await result.current.handleDiscardChanges()
    })

    expect(mockConfirm).toHaveBeenCalledWith(
      expect.objectContaining({
        title: 'Discard Changes?',
        variant: 'warning',
      }),
    )

    expect(result.current.editorContent).toBe('Original')
    expect(result.current.hasUnsavedChanges).toBe(false)
  })

  it('does not discard changes when user cancels', async () => {
    const draft = createMockDraft({ content: 'Original' })

    mockConfirm.mockResolvedValueOnce(false)

    const { result } = renderHook(() =>
      useDraftEditor({
        selectedDraft: draft,
        fetchDrafts: mockFetchDrafts,
        confirm: mockConfirm,
      }),
    )

    act(() => {
      result.current.handleEditorChange('Modified')
    })

    await act(async () => {
      await result.current.handleDiscardChanges()
    })

    expect(result.current.editorContent).toBe('Modified')
    expect(result.current.hasUnsavedChanges).toBe(true)
  })

  it('refreshes draft with confirmation if unsaved changes exist', async () => {
    const draft = createMockDraft()

    mockConfirm.mockResolvedValueOnce(true)

    const { result } = renderHook(() =>
      useDraftEditor({
        selectedDraft: draft,
        fetchDrafts: mockFetchDrafts,
        confirm: mockConfirm,
      }),
    )

    act(() => {
      result.current.handleEditorChange('Unsaved work')
    })

    await act(async () => {
      await result.current.handleRefreshDraft()
    })

    expect(mockConfirm).toHaveBeenCalledWith(
      expect.objectContaining({
        title: 'Refresh Draft?',
        message: expect.stringContaining('discard unsaved changes'),
      }),
    )

    expect(mockFetchDrafts).toHaveBeenCalledWith({ silent: true })
  })

  it.skip('auto-clears save status after timeout', async () => {
    // TODO: This test needs work with fake timers and React's async rendering
    // The saveStatus clear happens via setTimeout but doesn't trigger a re-render properly in tests
    const draft = createMockDraft()

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ success: true }),
    })

    const { result } = renderHook(() =>
      useDraftEditor({
        selectedDraft: draft,
        fetchDrafts: mockFetchDrafts,
        confirm: mockConfirm,
      }),
    )

    await act(async () => {
      await result.current.handleSaveDraft()
    })

    expect(result.current.saveStatus?.type).toBe('success')

    // Fast-forward time by 4 seconds (success timeout)
    act(() => {
      vi.advanceTimersByTime(4000)
    })

    // Wait for state update
    await waitFor(() => expect(result.current.saveStatus).toBeNull())
  })

  it('resets state when draft becomes null', () => {
    const draft = createMockDraft({ content: 'Test' })

    const { result, rerender } = renderHook(
      ({ selectedDraft }: { selectedDraft: Draft | null }) =>
        useDraftEditor({
          selectedDraft,
          fetchDrafts: mockFetchDrafts,
          confirm: mockConfirm,
        }),
      { initialProps: { selectedDraft: draft as Draft | null } },
    )

    expect(result.current.editorContent).toBe('Test')

    rerender({ selectedDraft: null })

    expect(result.current.editorContent).toBe('')
    expect(result.current.hasUnsavedChanges).toBe(false)
    expect(result.current.viewMode).toBe(ViewModes.SPLIT)
  })

  it('calculates draft metrics', () => {
    const draft = createMockDraft({
      content: '# Title\n\nSome content with multiple words and lines.\n\n## Section\n\nMore text.',
    })

    const { result } = renderHook(() =>
      useDraftEditor({
        selectedDraft: draft,
        fetchDrafts: mockFetchDrafts,
        confirm: mockConfirm,
      }),
    )

    expect(result.current.draftMetrics).toBeDefined()
    expect(result.current.draftMetrics.wordCount).toBeGreaterThan(0)
  })
})
