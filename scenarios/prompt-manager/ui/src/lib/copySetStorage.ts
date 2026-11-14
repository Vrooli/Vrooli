/**
 * Copy set storage - CRUD operations for saved selection sets.
 *
 * Each entity type (skills, agents, teams, topics, actions) stores its copy sets
 * separately in localStorage. Sets are deduplicated by strict ID equality
 * (order-independent). Named sets are pinned and never evicted.
 */

import type { CombineFormat, CombineEntityType } from '@/stores/combineStore'

/** Maximum unnamed entries per entity type before eviction. */
const MAX_UNNAMED = 300

/** Maximum entries to display in the UI. */
export const DISPLAY_LIMIT = 20

/** Schema version for future migrations. */
const SCHEMA_VERSION = 1

export interface CopySetEntry {
  /** Unique identifier for this entry. */
  id: string
  /** Sorted array of entity IDs (sorted for deterministic equality). */
  ids: string[]
  /** Number of times this exact set was copied. */
  copyCount: number
  /** ISO timestamp of the most recent copy. */
  lastCopiedAt: string
  /** ISO timestamp of first creation. */
  createdAt: string
  /** User-assigned name. Null = auto-captured, unnamed. Named entries are pinned. */
  name: string | null
  /** Format used on the most recent copy. */
  lastFormat: CombineFormat
}

interface CopySetStore {
  version: number
  entries: CopySetEntry[]
}

function storageKey(entityType: CombineEntityType): string {
  return `pm.copySets.${entityType}`
}

function generateId(): string {
  return crypto.randomUUID()
}

/** Strict sorted-array equality check. Both arrays must already be sorted. */
export function arraysEqual(a: string[], b: string[]): boolean {
  if (a.length !== b.length) return false
  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) return false
  }
  return true
}

function sortIds(ids: string[]): string[] {
  return [...ids].sort()
}

function isValidEntry(raw: unknown): raw is CopySetEntry {
  if (!raw || typeof raw !== 'object') return false
  const obj = raw as Record<string, unknown>
  return (
    typeof obj.id === 'string' &&
    Array.isArray(obj.ids) &&
    obj.ids.every((id: unknown) => typeof id === 'string') &&
    typeof obj.copyCount === 'number' &&
    typeof obj.lastCopiedAt === 'string' &&
    typeof obj.createdAt === 'string' &&
    (obj.name === null || typeof obj.name === 'string') &&
    typeof obj.lastFormat === 'string'
  )
}

/** Load all copy set entries for an entity type. Returns [] on error. */
export function loadCopySets(entityType: CombineEntityType): CopySetEntry[] {
  try {
    const raw = localStorage.getItem(storageKey(entityType))
    if (!raw) return []
    const parsed = JSON.parse(raw) as unknown
    if (!parsed || typeof parsed !== 'object') return []
    const store = parsed as Record<string, unknown>
    if (store.version !== SCHEMA_VERSION) return []
    if (!Array.isArray(store.entries)) return []
    return (store.entries as unknown[]).filter(isValidEntry)
  } catch {
    return []
  }
}

/** Save all copy set entries for an entity type. */
export function saveCopySets(entityType: CombineEntityType, entries: CopySetEntry[]): void {
  try {
    const store: CopySetStore = { version: SCHEMA_VERSION, entries }
    localStorage.setItem(storageKey(entityType), JSON.stringify(store))
  } catch {
    // localStorage unavailable or quota exceeded
  }
}

/**
 * Evict oldest unnamed entries when limit is exceeded.
 * Named entries are never evicted.
 */
function evictIfNeeded(entries: CopySetEntry[]): CopySetEntry[] {
  const named = entries.filter((e) => e.name !== null)
  const unnamed = entries.filter((e) => e.name === null)
  if (unnamed.length <= MAX_UNNAMED) return entries
  // Sort unnamed by lastCopiedAt ascending (oldest first), then slice
  unnamed.sort((a, b) => a.lastCopiedAt.localeCompare(b.lastCopiedAt))
  return [...named, ...unnamed.slice(unnamed.length - MAX_UNNAMED)]
}

/**
 * Record a copy event. Finds a matching set by ID equality and increments
 * its count, or creates a new entry. Returns the created/updated entry.
 */
export function recordCopySet(
  entityType: CombineEntityType,
  ids: string[],
  format: CombineFormat,
): CopySetEntry {
  const sorted = sortIds(ids)
  const entries = loadCopySets(entityType)
  const now = new Date().toISOString()

  const existingIndex = entries.findIndex((e) => arraysEqual(e.ids, sorted))
  let entry: CopySetEntry

  const existing = existingIndex >= 0 ? entries[existingIndex] : undefined
  if (existing) {
    entry = {
      ...existing,
      copyCount: existing.copyCount + 1,
      lastCopiedAt: now,
      lastFormat: format,
    }
    entries[existingIndex] = entry
  } else {
    entry = {
      id: generateId(),
      ids: sorted,
      copyCount: 1,
      lastCopiedAt: now,
      createdAt: now,
      name: null,
      lastFormat: format,
    }
    entries.push(entry)
  }

  saveCopySets(entityType, evictIfNeeded(entries))
  return entry
}

/** Delete a copy set entry by ID. */
export function deleteCopySet(entityType: CombineEntityType, entryId: string): void {
  const entries = loadCopySets(entityType)
  saveCopySets(entityType, entries.filter((e) => e.id !== entryId))
}

/** Rename a copy set entry. Pass null to unname (unpin). */
export function renameCopySet(entityType: CombineEntityType, entryId: string, name: string | null): void {
  const entries = loadCopySets(entityType)
  const index = entries.findIndex((e) => e.id === entryId)
  const found = index >= 0 ? entries[index] : undefined
  if (!found) return
  entries[index] = { ...found, name }
  saveCopySets(entityType, entries)
}

/**
 * Update the IDs in an existing entry. Re-sorts IDs.
 * Returns false if the new ID set collides with a different existing entry.
 */
export function updateCopySetIds(
  entityType: CombineEntityType,
  entryId: string,
  newIds: string[],
): boolean {
  const sorted = sortIds(newIds)
  const entries = loadCopySets(entityType)

  // Check for collision with a different entry
  const collision = entries.find((e) => e.id !== entryId && arraysEqual(e.ids, sorted))
  if (collision) return false

  const index = entries.findIndex((e) => e.id === entryId)
  const found = index >= 0 ? entries[index] : undefined
  if (!found) return false

  entries[index] = { ...found, ids: sorted }
  saveCopySets(entityType, entries)
  return true
}
