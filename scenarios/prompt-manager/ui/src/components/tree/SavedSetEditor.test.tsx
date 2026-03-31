/**
 * Tests for SavedSetEditor component.
 *
 * Covers: name input, removing items, adding items via search,
 * save/cancel actions, and validation.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { SavedSetEditor } from './SavedSetEditor'
import * as storage from '@/lib/copySetStorage'
import type { CopySetEntry } from '@/lib/copySetStorage'

vi.mock('@/lib/copySetStorage', () => ({
  updateCopySetIds: vi.fn().mockReturnValue(true),
  renameCopySet: vi.fn(),
}))

function makeEntry(overrides: Partial<CopySetEntry> = {}): CopySetEntry {
  return {
    id: 'entry-1',
    ids: ['a', 'b'],
    copyCount: 1,
    lastCopiedAt: '2026-01-01T00:00:00.000Z',
    createdAt: '2026-01-01T00:00:00.000Z',
    name: null,
    lastFormat: 'xml',
    ...overrides,
  }
}

const allEntities = [
  { id: 'a', name: 'Skill A' },
  { id: 'b', name: 'Skill B' },
  { id: 'c', name: 'Skill C' },
  { id: 'd', name: 'Skill D' },
]

const noopSave = vi.fn()
const noopCancel = vi.fn()

beforeEach(() => {
  vi.clearAllMocks()
})

describe('SavedSetEditor', () => {
  it('shows current items as removable chips', () => {
    render(
      <SavedSetEditor
        entry={makeEntry({ ids: ['a', 'b'] })}
        entityType="skills"
        allEntities={allEntities}
        onSave={noopSave}
        onCancel={noopCancel}
      />
    )
    expect(screen.getByText('Skill A')).toBeDefined()
    expect(screen.getByText('Skill B')).toBeDefined()
  })

  it('removes item when X is clicked on chip', () => {
    render(
      <SavedSetEditor
        entry={makeEntry({ ids: ['a', 'b'] })}
        entityType="skills"
        allEntities={allEntities}
        onSave={noopSave}
        onCancel={noopCancel}
      />
    )
    // Items count should start at 2
    expect(screen.getByText('Items (2)')).toBeDefined()
    // There are multiple X buttons (Remove titles)
    const removeButtons = screen.getAllByTitle('Remove')
    fireEvent.click(removeButtons[0] as HTMLElement)
    // Items count should decrease to 1
    expect(screen.getByText('Items (1)')).toBeDefined()
    expect(screen.getByText('Skill B')).toBeDefined()
  })

  it('adds entity via search', () => {
    render(
      <SavedSetEditor
        entry={makeEntry({ ids: ['a'] })}
        entityType="skills"
        allEntities={allEntities}
        onSave={noopSave}
        onCancel={noopCancel}
      />
    )
    const searchInput = screen.getByPlaceholderText('Search to add...')
    fireEvent.change(searchInput, { target: { value: 'Skill C' } })
    // Should show Skill C in the dropdown
    const addButtons = screen.getAllByText('Skill C')
    fireEvent.click(addButtons[addButtons.length - 1] as HTMLElement)
    // Now Skill C should appear as a chip
    expect(screen.getAllByText('Skill C').length).toBeGreaterThanOrEqual(1)
  })

  it('shows name input with correct placeholder', () => {
    render(
      <SavedSetEditor
        entry={makeEntry({ name: null })}
        entityType="skills"
        allEntities={allEntities}
        onSave={noopSave}
        onCancel={noopCancel}
      />
    )
    expect(screen.getByPlaceholderText('Name this set (optional)')).toBeDefined()
  })

  it('populates name input when entry has a name', () => {
    render(
      <SavedSetEditor
        entry={makeEntry({ name: 'My Set' })}
        entityType="skills"
        allEntities={allEntities}
        onSave={noopSave}
        onCancel={noopCancel}
      />
    )
    expect(screen.getByPlaceholderText('Name this set (optional)')).toHaveProperty('value', 'My Set')
  })

  it('calls storage functions and onSave on save', () => {
    render(
      <SavedSetEditor
        entry={makeEntry({ ids: ['a', 'b'], name: null })}
        entityType="skills"
        allEntities={allEntities}
        onSave={noopSave}
        onCancel={noopCancel}
      />
    )
    // Type a name
    const nameInput = screen.getByPlaceholderText('Name this set (optional)')
    fireEvent.change(nameInput, { target: { value: 'Test Name' } })

    fireEvent.click(screen.getByText('Save'))

    expect(storage.renameCopySet).toHaveBeenCalledWith('skills', 'entry-1', 'Test Name')
    expect(storage.updateCopySetIds).toHaveBeenCalledWith('skills', 'entry-1', ['a', 'b'])
    expect(noopSave).toHaveBeenCalled()
  })

  it('calls onCancel without saving', () => {
    render(
      <SavedSetEditor
        entry={makeEntry()}
        entityType="skills"
        allEntities={allEntities}
        onSave={noopSave}
        onCancel={noopCancel}
      />
    )
    fireEvent.click(screen.getByText('Cancel'))
    expect(noopCancel).toHaveBeenCalled()
    expect(storage.updateCopySetIds).not.toHaveBeenCalled()
    expect(storage.renameCopySet).not.toHaveBeenCalled()
  })

  it('shows error when set is empty on save', () => {
    render(
      <SavedSetEditor
        entry={makeEntry({ ids: ['a'] })}
        entityType="skills"
        allEntities={allEntities}
        onSave={noopSave}
        onCancel={noopCancel}
      />
    )
    // Remove the only item
    fireEvent.click(screen.getByTitle('Remove'))
    // Try to save
    fireEvent.click(screen.getByText('Save'))
    expect(screen.getByText('Set must contain at least one item')).toBeDefined()
    expect(noopSave).not.toHaveBeenCalled()
  })

  it('shows error on collision', () => {
    vi.mocked(storage.updateCopySetIds).mockReturnValue(false)
    render(
      <SavedSetEditor
        entry={makeEntry({ ids: ['a', 'b'] })}
        entityType="skills"
        allEntities={allEntities}
        onSave={noopSave}
        onCancel={noopCancel}
      />
    )
    fireEvent.click(screen.getByText('Save'))
    expect(screen.getByText('Another set with these exact items already exists')).toBeDefined()
    expect(noopSave).not.toHaveBeenCalled()
  })
})
