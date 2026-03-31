/**
 * Tests for selectionPersistence - persist/restore selection across refresh.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  loadPersistedSelection,
  savePersistedSelection,
  clearPersistedSelection,
} from './selectionPersistence'

// Mock localStorage
const store: Record<string, string> = {}
const localStorageMock = {
  getItem: vi.fn((key: string) => store[key] ?? null),
  setItem: vi.fn((key: string, value: string) => { store[key] = value }),
  removeItem: vi.fn((key: string) => { Reflect.deleteProperty(store, key) }),
  clear: vi.fn(),
  get length() { return Object.keys(store).length },
  key: vi.fn((i: number) => Object.keys(store)[i] ?? null),
}
Object.defineProperty(globalThis, 'localStorage', { value: localStorageMock, writable: true })

beforeEach(() => {
  vi.clearAllMocks()
  for (const k of Object.keys(store)) Reflect.deleteProperty(store, k)
})

describe('loadPersistedSelection', () => {
  it('returns null when nothing stored', () => {
    expect(loadPersistedSelection()).toBeNull()
  })

  it('returns null for corrupted JSON', () => {
    store['pm.selection'] = 'not-json'
    expect(loadPersistedSelection()).toBeNull()
  })

  it('returns null when isActive is false', () => {
    store['pm.selection'] = JSON.stringify({
      isActive: false,
      mode: 'skill-combine',
      entityType: 'skills',
      selectedIds: ['a'],
    })
    expect(loadPersistedSelection()).toBeNull()
  })

  it('returns null for invalid mode', () => {
    store['pm.selection'] = JSON.stringify({
      isActive: true,
      mode: 'invalid-mode',
      entityType: 'skills',
      selectedIds: ['a'],
    })
    expect(loadPersistedSelection()).toBeNull()
  })

  it('returns null for invalid entityType', () => {
    store['pm.selection'] = JSON.stringify({
      isActive: true,
      mode: 'skill-combine',
      entityType: 'invalid-type',
      selectedIds: ['a'],
    })
    expect(loadPersistedSelection()).toBeNull()
  })

  it('returns null for empty selectedIds', () => {
    store['pm.selection'] = JSON.stringify({
      isActive: true,
      mode: 'skill-combine',
      entityType: 'skills',
      selectedIds: [],
    })
    expect(loadPersistedSelection()).toBeNull()
  })

  it('returns null for non-string selectedIds', () => {
    store['pm.selection'] = JSON.stringify({
      isActive: true,
      mode: 'skill-combine',
      entityType: 'skills',
      selectedIds: [1, 2, 3],
    })
    expect(loadPersistedSelection()).toBeNull()
  })

  it('returns valid selection', () => {
    store['pm.selection'] = JSON.stringify({
      isActive: true,
      mode: 'ai-select',
      entityType: 'agents',
      selectedIds: ['agent-1', 'agent-2'],
    })
    const result = loadPersistedSelection()
    expect(result).toEqual({
      isActive: true,
      mode: 'ai-select',
      entityType: 'agents',
      selectedIds: ['agent-1', 'agent-2'],
    })
  })
})

describe('savePersistedSelection', () => {
  it('writes to pm.selection key', () => {
    savePersistedSelection({
      isActive: true,
      mode: 'skill-combine',
      entityType: 'skills',
      selectedIds: ['a', 'b'],
    })
    const raw = store['pm.selection']
    expect(raw).toBeDefined()
    const parsed = JSON.parse(raw ?? '{}')
    expect(parsed.selectedIds).toEqual(['a', 'b'])
  })
})

describe('clearPersistedSelection', () => {
  it('removes the key', () => {
    store['pm.selection'] = '{"isActive":true}'
    clearPersistedSelection()
    expect(store['pm.selection']).toBeUndefined()
  })
})
