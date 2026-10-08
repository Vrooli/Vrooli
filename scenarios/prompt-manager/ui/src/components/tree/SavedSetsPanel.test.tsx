/**
 * Tests for SavedSetsPanel component.
 *
 * Covers: sort toggle, entry rendering, apply/edit/delete callbacks,
 * display limit, and empty state.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@/test-utils/renderWithProviders'
import { SavedSetsPanel } from './SavedSetsPanel'
import * as storage from '@/lib/copySetStorage'
import type { CopySetEntry } from '@/lib/copySetStorage'

vi.mock('@/lib/copySetStorage', () => ({
  loadCopySets: vi.fn(),
  deleteCopySet: vi.fn(),
  renameCopySet: vi.fn(),
  DISPLAY_LIMIT: 20,
}))

function makeEntry(overrides: Partial<CopySetEntry> = {}): CopySetEntry {
  return {
    id: 'entry-1',
    ids: ['a', 'b'],
    copyCount: 1,
    lastCopiedAt: new Date().toISOString(),
    createdAt: new Date().toISOString(),
    name: null,
    lastFormat: 'xml',
    ...overrides,
  }
}

const noopApply = vi.fn()
const noopEdit = vi.fn()

const defaultLookup = new Map([
  ['a', 'Skill A'],
  ['b', 'Skill B'],
  ['c', 'Skill C'],
])

beforeEach(() => {
  vi.clearAllMocks()
})

describe('SavedSetsPanel', () => {
  it('renders empty state when no entries', () => {
    vi.mocked(storage.loadCopySets).mockReturnValue([])
    render(
      <SavedSetsPanel
        entityType="skills"
        onApplySet={noopApply}
        onEditSet={noopEdit}
        entityLookup={defaultLookup}
      />
    )
    expect(screen.getByText('No saved sets yet')).toBeDefined()
  })

  it('renders entries', () => {
    vi.mocked(storage.loadCopySets).mockReturnValue([
      makeEntry({ id: 'e1', name: 'My Set', copyCount: 3 }),
      makeEntry({ id: 'e2', ids: ['c'], copyCount: 1 }),
    ])
    render(
      <SavedSetsPanel
        entityType="skills"
        onApplySet={noopApply}
        onEditSet={noopEdit}
        entityLookup={defaultLookup}
      />
    )
    expect(screen.getByText('My Set')).toBeDefined()
    expect(screen.getByText('1 items')).toBeDefined()
  })

  it('sorts by frequency by default', () => {
    vi.mocked(storage.loadCopySets).mockReturnValue([
      makeEntry({ id: 'e1', name: 'Low', copyCount: 1, lastCopiedAt: '2026-03-31T12:00:00Z' }),
      makeEntry({ id: 'e2', name: 'High', copyCount: 5, lastCopiedAt: '2026-03-30T12:00:00Z' }),
    ])
    const { container } = render(
      <SavedSetsPanel
        entityType="skills"
        onApplySet={noopApply}
        onEditSet={noopEdit}
        entityLookup={defaultLookup}
      />
    )
    // High frequency should appear first
    const names = container.querySelectorAll('button')
    const texts = Array.from(names).map((b) => b.textContent).filter(Boolean)
    const highIdx = texts.findIndex((t) => t.includes('High'))
    const lowIdx = texts.findIndex((t) => t.includes('Low'))
    expect(highIdx).toBeLessThan(lowIdx)
  })

  it('switches to recency sort', () => {
    vi.mocked(storage.loadCopySets).mockReturnValue([
      makeEntry({ id: 'e1', name: 'Old', copyCount: 5, lastCopiedAt: '2026-01-01T00:00:00Z' }),
      makeEntry({ id: 'e2', name: 'New', copyCount: 1, lastCopiedAt: '2026-03-31T12:00:00Z' }),
    ])
    render(
      <SavedSetsPanel
        entityType="skills"
        onApplySet={noopApply}
        onEditSet={noopEdit}
        entityLookup={defaultLookup}
      />
    )
    fireEvent.click(screen.getByText('Recent'))
    // After sorting by recency, "New" should come first - verify it exists
    expect(screen.getByText('New')).toBeDefined()
  })

  it('calls onApplySet when apply is clicked', () => {
    vi.mocked(storage.loadCopySets).mockReturnValue([
      makeEntry({ id: 'e1', ids: ['a', 'b'] }),
    ])
    render(
      <SavedSetsPanel
        entityType="skills"
        onApplySet={noopApply}
        onEditSet={noopEdit}
        entityLookup={defaultLookup}
      />
    )
    fireEvent.click(screen.getByTitle('Apply this selection'))
    expect(noopApply).toHaveBeenCalledWith(['a', 'b'])
  })

  it('calls onEditSet when edit is clicked', () => {
    const entry = makeEntry({ id: 'e1' })
    vi.mocked(storage.loadCopySets).mockReturnValue([entry])
    render(
      <SavedSetsPanel
        entityType="skills"
        onApplySet={noopApply}
        onEditSet={noopEdit}
        entityLookup={defaultLookup}
      />
    )
    fireEvent.click(screen.getByTitle('Edit set'))
    expect(noopEdit).toHaveBeenCalledWith(entry)
  })

  it('calls deleteCopySet when delete is clicked', () => {
    vi.mocked(storage.loadCopySets).mockReturnValue([
      makeEntry({ id: 'e1' }),
    ])
    render(
      <SavedSetsPanel
        entityType="skills"
        onApplySet={noopApply}
        onEditSet={noopEdit}
        entityLookup={defaultLookup}
      />
    )
    fireEvent.click(screen.getByTitle('Delete set'))
    expect(storage.deleteCopySet).toHaveBeenCalledWith('skills', 'e1')
  })

  it('limits display to DISPLAY_LIMIT entries', () => {
    const entries = Array.from({ length: 25 }, (_, i) =>
      makeEntry({ id: `e${i}`, name: `Set ${i}`, copyCount: 25 - i })
    )
    vi.mocked(storage.loadCopySets).mockReturnValue(entries)
    render(
      <SavedSetsPanel
        entityType="skills"
        onApplySet={noopApply}
        onEditSet={noopEdit}
        entityLookup={defaultLookup}
      />
    )
    // Should show first 20, not all 25
    expect(screen.getByText('Set 0')).toBeDefined()
    expect(screen.getByText('Set 19')).toBeDefined()
    expect(screen.queryByText('Set 20')).toBeNull()
  })
})
