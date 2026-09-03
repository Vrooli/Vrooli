import { selectors } from '@/constants/selectors'
import type { SummaryView } from '../sim'
import { formatCountdown } from './format'

export type SummaryFilter = 'running' | 'gathered' | 'idle' | 'failed'

interface SummaryStripProps {
  summary: SummaryView
  now: number
  activeFilter: SummaryFilter | null
  onToggleFilter: (filter: SummaryFilter) => void
  teamNames: Record<string, string>
}

const COUNTS: Array<{ id: SummaryFilter; label: string; tone: string }> = [
  { id: 'running', label: 'Running', tone: 'text-sky-600 dark:text-sky-400' },
  { id: 'gathered', label: 'Gathering', tone: 'text-amber-600 dark:text-amber-400' },
  { id: 'idle', label: 'Idle', tone: 'text-muted-foreground' },
  { id: 'failed', label: 'Failed', tone: 'text-red-600 dark:text-red-400' },
]

/** "What is my swarm doing right now": four counts that double as filters, and the next heartbeat. */
export function SummaryStrip({ summary, now, activeFilter, onToggleFilter, teamNames }: SummaryStripProps) {
  const next = summary.nextHeartbeat
  return (
    <div
      className="pointer-events-auto flex items-center gap-1 rounded-lg border border-border bg-background/85 px-2 py-1 shadow-sm backdrop-blur"
      data-testid={selectors.world.hud.summary}
      role="group"
      aria-label="Swarm summary"
    >
      {COUNTS.map((count) => {
        const value = summary[count.id]
        const active = activeFilter === count.id
        return (
          <button
            key={count.id}
            type="button"
            aria-pressed={active}
            data-testid={`${selectors.world.hud.summary}-${count.id}`}
            onClick={() => onToggleFilter(count.id)}
            className={`flex items-baseline gap-1 rounded-md px-2 py-1 text-xs transition-colors hover:bg-muted ${active ? 'bg-muted ring-1 ring-primary/40' : ''}`}
            title={`Show only ${count.label.toLowerCase()} agents`}
          >
            <span className={`text-base font-semibold tabular-nums ${count.tone}`}>{value}</span>
            <span className="text-muted-foreground">{count.label}</span>
          </button>
        )
      })}
      <span className="mx-1 h-5 w-px bg-border" aria-hidden="true" />
      <span className="px-2 text-xs text-muted-foreground" data-testid={selectors.world.hud.nextHeartbeat}>
        {next ? (
          <>
            Next heartbeat <span className="font-medium text-foreground">{teamNames[next.teamId] ?? next.teamId}</span>{' '}
            <span className="tabular-nums">{formatCountdown(next.scheduledAt, now)}</span>
          </>
        ) : (
          'No heartbeat scheduled'
        )}
      </span>
    </div>
  )
}
