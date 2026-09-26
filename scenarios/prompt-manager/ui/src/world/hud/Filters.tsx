import { selectors } from '@/constants/selectors'
import type { TeamView } from '../sim'

import { type FilterState } from './filterState'

interface FiltersProps {
  value: FilterState
  teams: TeamView[]
  onChange: (next: FilterState) => void
}

export function Filters({ value, teams, onChange }: FiltersProps) {
  return (
    <div className="flex flex-wrap items-center gap-2 text-xs" data-testid={selectors.world.hud.filters}>
      <input
        type="search"
        value={value.search}
        onChange={(e) => onChange({ ...value, search: e.target.value })}
        placeholder="Find agent"
        aria-label="Find agent by name"
        className="h-7 w-36 rounded-md border border-border bg-background px-2"
        data-testid={selectors.world.hud.search}
      />
      <select
        value={value.teamId ?? ''}
        onChange={(e) => onChange({ ...value, teamId: e.target.value || null })}
        aria-label="Highlight team"
        className="h-7 rounded-md border border-border bg-background px-2"
        data-testid={selectors.world.hud.teamFilter}
      >
        <option value="">All teams</option>
        {teams.map((team) => (
          <option key={team.id} value={team.id}>
            {team.label}
          </option>
        ))}
      </select>
      <label className="flex items-center gap-1">
        <input type="checkbox" checked={value.onlyFailed} onChange={(e) => onChange({ ...value, onlyFailed: e.target.checked })} data-testid={selectors.world.hud.onlyFailed} />
        Only failed
      </label>
    </div>
  )
}

