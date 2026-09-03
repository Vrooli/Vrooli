import { useMemo, useState } from 'react'
import { selectors } from '@/constants/selectors'
import type { WorldActions } from '../data'
import type { FeedStatus } from '../data'
import type { WorldStore } from '../sim'
import { AgentCard } from './AgentCard'
import { EventTicker } from './EventTicker'
import { Filters } from './Filters'
import { matchesFilters, type FilterState } from './filterState'
import { SummaryStrip, type SummaryFilter } from './SummaryStrip'
import { TeamPanel } from './TeamPanel'
import { TwoDMode } from './TwoDMode'
import { useWorldView } from './useWorldView'

export interface HudProps {
  store: WorldStore
  actions: WorldActions
  feed: FeedStatus
  focusedId: string | null
  onFocus: (agentId: string | null) => void
  onFocusTeam: (teamId: string) => void
  onHome: () => void
  following: boolean
  onFollowChange: (follow: boolean) => void
  filters: FilterState
  onFiltersChange: (filters: FilterState) => void
  summaryFilter: SummaryFilter | null
  onSummaryFilterChange: (filter: SummaryFilter | null) => void
  highlightedTeamId: string | null
  onHighlightTeam: (teamId: string | null) => void
  twoD: boolean
  onTwoDChange: (twoD: boolean) => void
  tickerLimit: number
  onCustomize?: (agentId: string) => void
}

/**
 * The HUD reads only the sim view and calls only data actions. In 2D mode
 * the canvas is unmounted and the actor list takes its place; every action
 * stays available.
 */
export function WorldHud(props: HudProps) {
  const view = useWorldView(props.store)
  const [panelOpen, setPanelOpen] = useState(true)
  const names = useMemo(() => Object.fromEntries(view.actors.map((a) => [a.id, a.name])), [view.actors])
  const teamNames = useMemo(() => Object.fromEntries(view.teams.map((t) => [t.id, t.label])), [view.teams])
  const focused = props.focusedId ? view.actors.find((a) => a.id === props.focusedId) ?? null : null
  const filtered = useMemo(
    () => view.actors.filter((a) => matchesFilters(a, props.filters, props.summaryFilter)),
    [view.actors, props.filters, props.summaryFilter],
  )

  return (
    <div className="pointer-events-none absolute inset-0 z-20 flex flex-col" data-testid={selectors.world.hud.root}>
      <div className="flex items-start justify-center px-4 pt-3">
        <SummaryStrip
          summary={view.summary}
          now={view.time}
          activeFilter={props.summaryFilter}
          onToggleFilter={(f) => props.onSummaryFilterChange(props.summaryFilter === f ? null : f)}
          teamNames={teamNames}
        />
      </div>
      {props.twoD && (
        <div className="pointer-events-auto flex-1 overflow-hidden">
          <TwoDMode actors={filtered} teams={view.teams} now={view.time} focusedId={props.focusedId} onFocus={props.onFocus} />
        </div>
      )}
      <div className="mt-auto flex items-end justify-between gap-3 p-3">
        <aside className={`pointer-events-auto w-72 rounded-lg border border-border bg-background/85 shadow-sm backdrop-blur ${panelOpen ? '' : 'w-auto'}`}>
          <div className="flex items-center justify-between px-3 py-1.5">
            <button type="button" className="text-xs font-semibold uppercase tracking-wide text-muted-foreground" onClick={() => setPanelOpen((open) => !open)} aria-expanded={panelOpen}>
              Swarm {panelOpen ? '▾' : '▸'}
            </button>
            <div className="flex items-center gap-2">
              <span className="text-[10px] uppercase tracking-wide text-muted-foreground" data-testid={selectors.world.hud.feedStatus} title={props.feed.lastError ?? undefined}>
                feed: {props.feed.mode}
              </span>
              <button type="button" onClick={props.onHome} className="rounded border border-border px-1.5 py-0.5 text-[10px] hover:bg-muted" data-testid={selectors.world.hud.home} title="Home view (Esc)">
                Home
              </button>
              <label className="flex items-center gap-1 text-[10px] text-muted-foreground">
                <input type="checkbox" checked={props.twoD} onChange={(e) => props.onTwoDChange(e.target.checked)} data-testid={selectors.world.hud.twoDToggle} />
                2D
              </label>
            </div>
          </div>
          {panelOpen && (
            <div className="space-y-3 border-t border-border px-3 py-2">
              <Filters value={props.filters} teams={view.teams} onChange={props.onFiltersChange} />
              <TeamPanel teams={view.teams} highlightedTeamId={props.highlightedTeamId} onFocusTeam={props.onFocusTeam} onHighlightTeam={props.onHighlightTeam} />
              <div className="max-h-40 overflow-auto">
                <EventTicker events={view.events} names={names} teamNames={teamNames} limit={props.tickerLimit} onFocusActor={props.onFocus} />
              </div>
            </div>
          )}
        </aside>
        {focused && (
          <AgentCard
            actor={focused}
            teamName={focused.teamId ? teamNames[focused.teamId] : undefined}
            now={view.time}
            actions={props.actions}
            following={props.following}
            onFollowChange={props.onFollowChange}
            onClose={() => props.onFocus(null)}
            onCustomize={props.onCustomize ? () => props.onCustomize?.(focused.id) : undefined}
            docked={props.twoD}
          />
        )}
      </div>
    </div>
  )
}
