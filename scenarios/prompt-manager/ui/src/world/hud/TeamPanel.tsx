import { selectors } from '@/constants/selectors'
import type { TeamView } from '../sim'

interface TeamPanelProps {
  teams: TeamView[]
  highlightedTeamId: string | null
  onFocusTeam: (teamId: string) => void
  onHighlightTeam: (teamId: string | null) => void
}

/** Rooms with their member state counts; click focuses the room, hover highlights it. */
export function TeamPanel({ teams, highlightedTeamId, onFocusTeam, onHighlightTeam }: TeamPanelProps) {
  if (teams.length === 0) {
    return (
      <p className="text-xs text-muted-foreground" data-testid={selectors.world.hud.teamPanel}>
        No teams yet. Agents gather in the commons.
      </p>
    )
  }
  return (
    <ul className="space-y-1" data-testid={selectors.world.hud.teamPanel} aria-label="Teams">
      {teams.map((team) => {
        const running = team.states.working + team.states.walkingToDesk
        const gathered = team.states.gathered + team.states.walkingToTable
        const highlighted = highlightedTeamId === team.id
        return (
          <li key={team.id}>
            <button
              type="button"
              onClick={() => onFocusTeam(team.id)}
              onMouseEnter={() => onHighlightTeam(team.id)}
              onMouseLeave={() => onHighlightTeam(null)}
              onFocus={() => onHighlightTeam(team.id)}
              onBlur={() => onHighlightTeam(null)}
              className={`flex w-full items-center justify-between gap-2 rounded-md px-2 py-1 text-left text-xs hover:bg-muted ${highlighted ? 'bg-muted' : ''}`}
              data-testid={`${selectors.world.hud.teamPanel}-${team.id}`}
            >
              <span className="truncate font-medium">{team.label}</span>
              <span className="flex shrink-0 gap-2 tabular-nums text-muted-foreground">
                <span title="members">{team.memberIds.length}</span>
                {running > 0 && <span className="text-sky-600 dark:text-sky-400" title="running">{running}▶</span>}
                {gathered > 0 && <span className="text-amber-600 dark:text-amber-400" title="gathering">{gathered}◆</span>}
                {team.states.failed > 0 && <span className="text-red-600 dark:text-red-400" title="failed">{team.states.failed}!</span>}
              </span>
            </button>
          </li>
        )
      })}
    </ul>
  )
}
