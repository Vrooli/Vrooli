/**
 * RunEventsTab - Event stream tab with filtering controls.
 *
 * Uses EventsDisplay component with:
 * - Type filter: toggleable chips per event type
 * - Text search input that filters events
 */

import { useState, useCallback } from 'react'
import { Search } from 'lucide-react'
import { cn } from '@/lib/utils'
import { EventsDisplay } from '@/components/shared/EventsDisplay'
import type { RunEvent } from '@/services/heartbeatService'

interface RunEventsTabProps {
  runId: string
  live?: boolean
  className?: string
}

const EVENT_TYPES: RunEvent['eventType'][] = [
  'message',
  'tool_call',
  'tool_result',
  'status',
  'metric',
  'log',
  'error',
]

export function RunEventsTab({ runId, live, className }: RunEventsTabProps) {
  const [filterTypes, setFilterTypes] = useState<RunEvent['eventType'][]>([])
  const [searchQuery, setSearchQuery] = useState('')

  const toggleType = useCallback((type: RunEvent['eventType']) => {
    setFilterTypes((prev) =>
      prev.includes(type)
        ? prev.filter((t) => t !== type)
        : [...prev, type]
    )
  }, [])

  return (
    <div className={cn('flex flex-col gap-3', className)}>
      {/* Filter controls */}
      <div className="flex flex-col gap-2">
        {/* Type filter chips */}
        <div className="flex gap-1 flex-wrap">
          {EVENT_TYPES.map((type) => (
            <button
              key={type}
              type="button"
              onClick={() => toggleType(type)}
              className={cn(
                'px-2 py-1 text-[10px] rounded border transition-colors',
                filterTypes.includes(type)
                  ? 'bg-primary/10 text-primary border-primary/40'
                  : 'text-muted-foreground border-border hover:text-foreground hover:bg-muted/50'
              )}
            >
              {type.replace('_', ' ')}
            </button>
          ))}
          {filterTypes.length > 0 && (
            <button
              type="button"
              onClick={() => setFilterTypes([])}
              className="px-2 py-1 text-[10px] rounded border border-border text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors"
            >
              Clear
            </button>
          )}
        </div>

        {/* Text search */}
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search events..."
            className={cn(
              'w-full pl-8 pr-3 py-1.5 text-xs',
              'bg-muted border border-border rounded-md',
              'text-foreground placeholder:text-muted-foreground',
              'focus:outline-none focus:ring-2 focus:ring-primary'
            )}
          />
        </div>
      </div>

      {/* Event stream */}
      <EventsDisplay
        runId={runId}
        live={live}
        filterTypes={filterTypes.length > 0 ? filterTypes : undefined}
        searchQuery={searchQuery || undefined}
      />
    </div>
  )
}
