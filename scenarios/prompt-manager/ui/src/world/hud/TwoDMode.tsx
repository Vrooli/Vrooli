import { selectors } from '@/constants/selectors'
import type { ActorView, TeamView } from '../sim'
import { STATE_LABEL, formatDuration } from './format'

interface TwoDModeProps {
  actors: ActorView[]
  teams: TeamView[]
  now: number
  focusedId: string | null
  onFocus: (agentId: string | null) => void
}

/**
 * The world without the canvas: every actor as a row grouped by team, with
 * the same focus target the 3D view uses. Everything the AgentCard offers is
 * reachable from here.
 */
export function TwoDMode({ actors, teams, now, focusedId, onFocus }: TwoDModeProps) {
  const byTeam = new Map<string, ActorView[]>()
  const unassigned: ActorView[] = []
  for (const actor of actors) {
    if (!actor.teamId) {
      unassigned.push(actor)
      continue
    }
    const list = byTeam.get(actor.teamId) ?? []
    list.push(actor)
    byTeam.set(actor.teamId, list)
  }
  const groups: Array<{ id: string; label: string; members: ActorView[] }> = teams
    .map((team) => ({ id: team.id, label: team.label, members: byTeam.get(team.id) ?? [] }))
    .filter((g) => g.members.length > 0)
  if (unassigned.length > 0) groups.push({ id: 'commons', label: 'Commons', members: unassigned })

  return (
    <div className="h-full overflow-auto p-4" data-testid={selectors.world.hud.twoDMode}>
      <ul className="mx-auto max-w-3xl space-y-4" data-testid={selectors.world.hud.actorList} aria-label="Agents by team">
        {groups.length === 0 && <li className="text-sm text-muted-foreground">No agents match.</li>}
        {groups.map((group) => (
          <li key={group.id}>
            <h3 className="mb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">{group.label}</h3>
            <ul className="divide-y divide-border rounded-md border border-border bg-background/70">
              {group.members.map((actor) => (
                <li key={actor.id}>
                  <button
                    type="button"
                    aria-pressed={focusedId === actor.id}
                    onClick={() => onFocus(focusedId === actor.id ? null : actor.id)}
                    className={`flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm hover:bg-muted ${focusedId === actor.id ? 'bg-muted' : ''}`}
                    data-testid={`${selectors.world.hud.actorList}-${actor.id}`}
                  >
                    <span className="flex items-center gap-2">
                      <span className="inline-block h-3 w-3 rounded-full" style={{ backgroundColor: actor.colors.body }} aria-hidden="true" />
                      <span className="font-medium">{actor.name}</span>
                    </span>
                    <span className="flex items-center gap-3 text-xs text-muted-foreground">
                      <span>{STATE_LABEL[actor.state]}</span>
                      <span className="tabular-nums">{formatDuration(now - actor.stateSince)}</span>
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          </li>
        ))}
      </ul>
    </div>
  )
}
