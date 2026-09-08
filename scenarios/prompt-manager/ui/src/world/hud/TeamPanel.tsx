import { selectors } from '@/constants/selectors'
import type { TeamView, Place, Actor, WorldStore } from '../sim'
import { useCallback, useEffect, useRef, useSyncExternalStore } from 'react'
import { insideSpace, spaceActors } from '../sim/layout/spaces'

type SpaceMenuActions = { walking: boolean; onClose: () => void; onManage: () => void; onAgent: (id: string) => void }

/** Subscribe only while the menu is mounted; actor motion does not redraw the world HUD. */
export function ConnectedSpaceMenu({ store, spaceId, ...actions }: SpaceMenuActions & { store: WorldStore; spaceId: string }) {
  const subscribe = useCallback((listener: () => void) => store.subscribeState(listener), [store])
  const snapshot = useCallback(() => store.getState(), [store])
  const state = useSyncExternalStore(subscribe, snapshot, snapshot)
  const space = state.places[spaceId]
  return space?.space ? <SpaceMenu space={space} actors={spaceActors(state, space)} {...actions} /> : null
}

/** Enclosures expose occupants even when their bodies are hidden by a roof. */
export function SpaceMenu({ space, actors, walking, onClose, onManage, onAgent }: {
  space: Place; actors: Actor[]; walking: boolean;
  onClose: () => void; onManage: () => void; onAgent: (id: string) => void;
}) {
  const close = useRef<HTMLButtonElement>(null)
  useEffect(() => { close.current?.focus() }, [space.id])
  return <section role="dialog" aria-label={`${space.label} space`} onKeyDown={event => { if (event.key === 'Escape') { event.preventDefault(); event.stopPropagation(); onClose() } }} className="absolute right-4 top-28 z-30 w-80 max-w-[calc(100%-2rem)] rounded-xl border bg-background/95 p-4 shadow-xl backdrop-blur">
    <div className="flex items-center justify-between gap-3"><h2 className="font-semibold">{space.label}</h2>
      <button ref={close} type="button" onClick={onClose} className="rounded px-2 py-1 text-sm hover:bg-muted focus-visible:ring-2" aria-label="Close space menu">×</button></div>
    <p className="mt-1 text-xs text-muted-foreground">{space.space?.kind === 'campsite' ? `${space.space.variant.toUpperCase()} campsite` : space.teamId ? 'Team office' : 'Shared space'} · {actors.filter(actor => insideSpace(space, actor.position)).length} here</p>
    <button type="button" onClick={onManage} className="my-3 w-full rounded-md border px-3 py-2 text-sm hover:bg-muted focus-visible:ring-2">Manage space</button>
    <ul aria-label="Space occupants" className="max-h-72 space-y-2 overflow-y-auto">
      {actors.map(actor => <li key={actor.id} className="flex items-center justify-between gap-2 rounded-md bg-muted/50 p-2 text-sm">
        <span className="min-w-0 truncate">{actor.name}<span className="block text-xs text-muted-foreground">{insideSpace(space, actor.position) ? 'Here' : 'Assigned · away'}</span></span>
        <button type="button" onClick={() => onAgent(actor.id)} className="shrink-0 rounded-md border px-2 py-1 hover:bg-background focus-visible:ring-2">{walking ? 'Come here' : 'View agent'}</button>
      </li>)}
    </ul>
    {actors.length === 0 && <p className="text-sm text-muted-foreground">No agents here or assigned to this space.</p>}
  </section>
}

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
