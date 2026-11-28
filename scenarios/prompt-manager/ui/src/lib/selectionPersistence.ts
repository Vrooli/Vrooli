/**
 * Selection persistence - saves and restores combine mode selection
 * across page refreshes via localStorage.
 */

import type { CombineMode, CombineEntityType } from '@/stores/combineStore'

const STORAGE_KEY = 'pm.selection'

const VALID_MODES = new Set<string>(['skill-combine', 'ai-select'])
const VALID_ENTITY_TYPES = new Set<string>(['skills', 'agents', 'teams', 'topics', 'actions'])

export interface PersistedSelection {
  isActive: boolean
  mode: CombineMode
  entityType: CombineEntityType
  selectedIds: string[]
}

/** Load persisted selection from localStorage. Returns null if not found or invalid. */
export function loadPersistedSelection(): PersistedSelection | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw) as unknown
    if (!parsed || typeof parsed !== 'object') return null
    const obj = parsed as Record<string, unknown>

    if (typeof obj.isActive !== 'boolean') return null
    if (!obj.isActive) return null
    if (!VALID_MODES.has(obj.mode as string)) return null
    if (!VALID_ENTITY_TYPES.has(obj.entityType as string)) return null
    if (!Array.isArray(obj.selectedIds)) return null
    if (!obj.selectedIds.every((id: unknown) => typeof id === 'string')) return null
    if (obj.selectedIds.length === 0) return null

    return {
      isActive: true,
      mode: obj.mode as CombineMode,
      entityType: obj.entityType as CombineEntityType,
      selectedIds: obj.selectedIds,
    }
  } catch {
    return null
  }
}

/** Save selection state to localStorage. */
export function savePersistedSelection(state: PersistedSelection): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
  } catch {
    // localStorage unavailable or quota exceeded
  }
}

/** Clear persisted selection from localStorage. */
export function clearPersistedSelection(): void {
  try {
    localStorage.removeItem(STORAGE_KEY)
  } catch {
    // localStorage unavailable
  }
}
