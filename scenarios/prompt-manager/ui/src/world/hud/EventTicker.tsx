import { selectors } from '@/constants/selectors'
import type { WorldEvent } from '../sim'
import { formatClock } from './format'

interface EventTickerProps {
  events: WorldEvent[]
  names: Record<string, string>
  teamNames: Record<string, string>
  limit: number
  onFocusActor: (agentId: string) => void
}

const KIND_LABEL: Record<WorldEvent['kind'], string> = {
  'run.started': 'started a run',
  'run.finished': 'finished a run',
  'run.failed': 'run failed',
  'heartbeat.upcoming': 'heartbeat scheduled',
  'heartbeat.cancelled': 'heartbeat cancelled',
  'agent.message': 'said',
  'failed.acknowledged': 'failure acknowledged',
  'actor.arrived': 'arrived',
  'actor.state': 'changed state',
}

/** Signal events only (state transitions and arrivals are noise for the operator). */
const SHOWN_KINDS = new Set<WorldEvent['kind']>(['run.started', 'run.finished', 'run.failed', 'heartbeat.upcoming', 'heartbeat.cancelled', 'agent.message', 'failed.acknowledged'])

export function EventTicker({ events, names, teamNames, limit, onFocusActor }: EventTickerProps) {
  const shown = events.filter((e) => SHOWN_KINDS.has(e.kind)).slice(0, limit)
  return (
    <ol className="space-y-0.5 text-xs" data-testid={selectors.world.hud.ticker} aria-label="Recent events" aria-live="polite">
      {shown.length === 0 && <li className="text-muted-foreground">No events yet</li>}
      {shown.map((event) => {
        const who = event.agentId ? (names[event.agentId] ?? event.agentId) : event.teamId ? (teamNames[event.teamId] ?? event.teamId) : 'World'
        const tone = event.kind === 'run.failed' ? 'text-red-600 dark:text-red-400' : event.kind === 'run.started' ? 'text-sky-600 dark:text-sky-400' : 'text-foreground'
        return (
          <li key={event.seq} className="flex gap-2">
            <span className="tabular-nums text-muted-foreground">{formatClock(event.at)}</span>
            {event.agentId ? (
              <button type="button" className={`truncate text-left hover:underline ${tone}`} onClick={() => onFocusActor(event.agentId ?? '')} data-testid={`${selectors.world.hud.ticker}-event-${event.seq}`}>
                {who} {KIND_LABEL[event.kind]}
                {event.kind === 'agent.message' && event.message ? ` “${event.message}”` : ''}
                {event.kind === 'run.failed' && event.message ? `: ${event.message}` : ''}
              </button>
            ) : (
              <span className={`truncate ${tone}`}>
                {who} {KIND_LABEL[event.kind]}
              </span>
            )}
          </li>
        )
      })}
    </ol>
  )
}
