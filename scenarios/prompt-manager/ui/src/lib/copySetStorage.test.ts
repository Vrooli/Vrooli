/**
 * Tests for copySetStorage - copy set CRUD, matching, and eviction.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  loadCopySets,
  saveCopySets,
  recordCopySet,
  deleteCopySet,
  renameCopySet,
  updateCopySetIds,
  arraysEqual,
  DISPLAY_LIMIT,
} from './copySetStorage'
import type { CopySetEntry } from './copySetStorage'

// Mock localStorage
const store: Record<string, string> = {}
const localStorageMock = {
  getItem: vi.fn((key: string) => store[key] ?? null),
  setItem: vi.fn((key: string, value: string) => { store[key] = value }),
  removeItem: vi.fn((key: string) => { Reflect.deleteProperty(store, key) }),
  clear: vi.fn(() => { for (const k of Object.keys(store)) Reflect.deleteProperty(store, k) }),
  get length() { return Object.keys(store).length },
  key: vi.fn((i: number) => Object.keys(store)[i] ?? null),
}
Object.defineProperty(globalThis, 'localStorage', { value: localStorageMock, writable: true })

function makeEntry(overrides: Partial<CopySetEntry> = {}): CopySetEntry {
  return {
    id: 'entry-1',
    ids: ['a', 'b', 'c'],
    copyCount: 1,
    lastCopiedAt: '2026-01-01T00:00:00.000Z',
    createdAt: '2026-01-01T00:00:00.000Z',
    name: null,
    lastFormat: 'xml',
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  for (const k of Object.keys(store)) Reflect.deleteProperty(store, k)
})

describe('arraysEqual', () => {
  it('returns true for identical sorted arrays', () => {
    expect(arraysEqual(['a', 'b', 'c'], ['a', 'b', 'c'])).toBe(true)
  })

  it('returns false for different arrays', () => {
    expect(arraysEqual(['a', 'b'], ['a', 'c'])).toBe(false)
  })

  it('returns false for different lengths', () => {
    expect(arraysEqual(['a', 'b'], ['a', 'b', 'c'])).toBe(false)
  })

  it('returns true for empty arrays', () => {
    expect(arraysEqual([], [])).toBe(true)
  })
})

describe('loadCopySets', () => {
  it('returns empty array when nothing stored', () => {
    expect(loadCopySets('skills')).toEqual([])
  })

  it('returns empty array for corrupted JSON', () => {
    store['pm.copySets.skills'] = 'not-json'
    expect(loadCopySets('skills')).toEqual([])
  })

  it('returns empty array for wrong schema version', () => {
    store['pm.copySets.skills'] = JSON.stringify({ version: 999, entries: [] })
    expect(loadCopySets('skills')).toEqual([])
  })

  it('returns valid entries', () => {
    const entry = makeEntry()
    store['pm.copySets.skills'] = JSON.stringify({ version: 1, entries: [entry] })
    const result = loadCopySets('skills')
    expect(result).toHaveLength(1)
    expect(result[0]?.id).toBe('entry-1')
  })

  it('filters out invalid entries', () => {
    store['pm.copySets.skills'] = JSON.stringify({
      version: 1,
      entries: [makeEntry(), { id: 123, ids: 'not-array' }],
    })
    expect(loadCopySets('skills')).toHaveLength(1)
  })
})

describe('saveCopySets', () => {
  it('writes to correct key', () => {
    saveCopySets('agents', [makeEntry()])
    const raw = store['pm.copySets.agents']
    expect(raw).toBeDefined()
    const parsed = JSON.parse(raw ?? '{}')
    expect(parsed.version).toBe(1)
    expect(parsed.entries).toHaveLength(1)
  })

  it('isolates Action copy sets under the actions namespace', () => {
    saveCopySets('actions', [makeEntry({ ids: ['team.swarm.work.list'] })])
    saveCopySets('skills', [makeEntry({ ids: ['implementation-plan-authoring'] })])

    expect(loadCopySets('actions')[0]?.ids).toEqual(['team.swarm.work.list'])
    expect(loadCopySets('skills')[0]?.ids).toEqual(['implementation-plan-authoring'])
    expect(store['pm.copySets.actions']).toBeDefined()
  })
})

describe('recordCopySet', () => {
  it('creates a new entry when no match exists', () => {
    const entry = recordCopySet('skills', ['b', 'a'], 'json')
    expect(entry.copyCount).toBe(1)
    expect(entry.ids).toEqual(['a', 'b']) // sorted
    expect(entry.lastFormat).toBe('json')
    expect(entry.name).toBeNull()
  })

  it('increments count when matching set exists (order-independent)', () => {
    // First record
    recordCopySet('skills', ['a', 'b', 'c'], 'xml')
    // Same IDs in different order
    const entry = recordCopySet('skills', ['c', 'a', 'b'], 'markdown')
    expect(entry.copyCount).toBe(2)
    expect(entry.lastFormat).toBe('markdown')
    expect(loadCopySets('skills')).toHaveLength(1)
  })

  it('creates separate entries for different ID sets', () => {
    recordCopySet('skills', ['a', 'b'], 'xml')
    recordCopySet('skills', ['a', 'c'], 'xml')
    expect(loadCopySets('skills')).toHaveLength(2)
  })

  it('stores per entity type separately', () => {
    recordCopySet('skills', ['a'], 'xml')
    recordCopySet('agents', ['a'], 'xml')
    expect(loadCopySets('skills')).toHaveLength(1)
    expect(loadCopySets('agents')).toHaveLength(1)
  })
})

describe('eviction', () => {
  it('evicts oldest unnamed entries when exceeding limit', () => {
    // Seed 301 unnamed entries with increasing timestamps
    const entries: CopySetEntry[] = []
    for (let i = 0; i < 301; i++) {
      entries.push(makeEntry({
        id: `entry-${i}`,
        ids: [`id-${i}`],
        lastCopiedAt: new Date(2026, 0, 1, 0, 0, i).toISOString(),
      }))
    }
    saveCopySets('skills', entries)

    // Recording one more triggers eviction
    recordCopySet('skills', ['new-id'], 'xml')
    const result = loadCopySets('skills')
    // Should have at most 301 (300 unnamed max + the new one, with oldest evicted)
    expect(result.length).toBeLessThanOrEqual(301)
    // The oldest entry (entry-0) should be evicted
    expect(result.find((e) => e.id === 'entry-0')).toBeUndefined()
  })

  it('never evicts named entries', () => {
    const entries: CopySetEntry[] = []
    // 300 unnamed + 5 named
    for (let i = 0; i < 300; i++) {
      entries.push(makeEntry({
        id: `unnamed-${i}`,
        ids: [`uid-${i}`],
        lastCopiedAt: new Date(2026, 0, 1, 0, 0, i).toISOString(),
      }))
    }
    for (let i = 0; i < 5; i++) {
      entries.push(makeEntry({
        id: `named-${i}`,
        ids: [`nid-${i}`],
        name: `Set ${i}`,
        lastCopiedAt: '2020-01-01T00:00:00.000Z', // Very old
      }))
    }
    saveCopySets('skills', entries)

    recordCopySet('skills', ['trigger-eviction'], 'xml')
    const result = loadCopySets('skills')
    // All named entries should survive
    for (let i = 0; i < 5; i++) {
      expect(result.find((e) => e.id === `named-${i}`)).toBeDefined()
    }
  })
})

describe('deleteCopySet', () => {
  it('removes an entry by ID', () => {
    recordCopySet('skills', ['a', 'b'], 'xml')
    const entries = loadCopySets('skills')
    expect(entries).toHaveLength(1)
    deleteCopySet('skills', entries[0]?.id ?? '')
    expect(loadCopySets('skills')).toHaveLength(0)
  })

  it('does nothing for non-existent ID', () => {
    recordCopySet('skills', ['a'], 'xml')
    deleteCopySet('skills', 'non-existent')
    expect(loadCopySets('skills')).toHaveLength(1)
  })
})

describe('renameCopySet', () => {
  it('sets a name on an entry', () => {
    const entry = recordCopySet('skills', ['a'], 'xml')
    renameCopySet('skills', entry.id, 'My Set')
    const updated = loadCopySets('skills')
    expect(updated[0]?.name).toBe('My Set')
  })

  it('clears a name (unpins)', () => {
    const entry = recordCopySet('skills', ['a'], 'xml')
    renameCopySet('skills', entry.id, 'My Set')
    renameCopySet('skills', entry.id, null)
    const updated = loadCopySets('skills')
    expect(updated[0]?.name).toBeNull()
  })

  it('does nothing for non-existent ID', () => {
    recordCopySet('skills', ['a'], 'xml')
    renameCopySet('skills', 'non-existent', 'Name')
    expect(loadCopySets('skills')[0]?.name).toBeNull()
  })
})

describe('updateCopySetIds', () => {
  it('updates IDs and re-sorts', () => {
    const entry = recordCopySet('skills', ['a', 'b'], 'xml')
    const ok = updateCopySetIds('skills', entry.id, ['c', 'a'])
    expect(ok).toBe(true)
    const updated = loadCopySets('skills')
    expect(updated[0]?.ids).toEqual(['a', 'c'])
  })

  it('returns false on collision with different entry', () => {
    recordCopySet('skills', ['a', 'b'], 'xml')
    const entry2 = recordCopySet('skills', ['c', 'd'], 'xml')
    // Try to change entry2 IDs to match entry1
    const ok = updateCopySetIds('skills', entry2.id, ['a', 'b'])
    expect(ok).toBe(false)
  })

  it('allows updating to same IDs (no-op)', () => {
    const entry = recordCopySet('skills', ['a', 'b'], 'xml')
    const ok = updateCopySetIds('skills', entry.id, ['b', 'a'])
    expect(ok).toBe(true)
  })

  it('returns false for non-existent entry', () => {
    const ok = updateCopySetIds('skills', 'non-existent', ['a'])
    expect(ok).toBe(false)
  })
})

describe('DISPLAY_LIMIT', () => {
  it('is 20', () => {
    expect(DISPLAY_LIMIT).toBe(20)
  })
})
