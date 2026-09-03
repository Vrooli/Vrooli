export interface FilterState {
  search: string
  teamId: string | null
  onlyFailed: boolean
}

export const EMPTY_FILTERS: FilterState = { search: '', teamId: null, onlyFailed: false }

/** Which actors pass the filters (used by the scene to dim the rest and by 2D mode to list). */
export function matchesFilters(actor: { name: string; teamId?: string; state: string }, filters: FilterState, summaryFilter: string | null): boolean {
  if (filters.search && !actor.name.toLowerCase().includes(filters.search.toLowerCase())) return false
  if (filters.teamId && actor.teamId !== filters.teamId) return false
  if (filters.onlyFailed && actor.state !== 'failed') return false
  if (summaryFilter === 'running' && actor.state !== 'working' && actor.state !== 'walkingToDesk') return false
  if (summaryFilter === 'gathered' && actor.state !== 'gathered' && actor.state !== 'walkingToTable') return false
  if (summaryFilter === 'idle' && actor.state !== 'idle' && actor.state !== 'socializing') return false
  if (summaryFilter === 'failed' && actor.state !== 'failed') return false
  return true
}
