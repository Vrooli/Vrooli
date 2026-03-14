import { useState, useCallback, useMemo } from 'react'
import { copyToClipboard } from '@/lib/clipboard'
import { Search, Filter, FileOutput } from 'lucide-react'
import { cn } from '@/lib/utils'
import { EventsDisplay } from '@/components/shared/EventsDisplay'
import { Dialog } from '@/components/shared/Dialog'
import { toast } from '@/hooks/use-toast'
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

function escapeXml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&apos;')
}

function toXmlNode(name: string, value: unknown, indent = 0): string {
  const pad = '  '.repeat(indent)

  if (value === null || value === undefined) {
    return `${pad}<${name} />`
  }

  if (Array.isArray(value)) {
    const items = value.map((item) => toXmlNode('item', item, indent + 1)).join('\n')
    return `${pad}<${name}>\n${items}\n${pad}</${name}>`
  }

  if (typeof value === 'object') {
    const entries = Object.entries(value as Record<string, unknown>)
    if (entries.length === 0) {
      return `${pad}<${name} />`
    }
    const children = entries.map(([key, child]) => toXmlNode(key, child, indent + 1)).join('\n')
    return `${pad}<${name}>\n${children}\n${pad}</${name}>`
  }

  return `${pad}<${name}>${escapeXml(String(value as string | number | boolean))}</${name}>`
}

function runEventsToXml(events: RunEvent[]): string {
  const body = events
    .map((event) => toXmlNode('event', {
      id: event.id,
      runId: event.runId,
      sequence: event.sequence,
      eventType: event.eventType,
      timestamp: event.timestamp,
      data: event.data,
    }, 1))
    .join('\n')

  return `<?xml version="1.0" encoding="UTF-8"?>\n<runEvents>\n${body}\n</runEvents>`
}

export function RunEventsTab({ runId, live, className }: RunEventsTabProps) {
  const [filterTypes, setFilterTypes] = useState<RunEvent['eventType'][]>([])
  const [searchQuery, setSearchQuery] = useState('')
  const [filteredEvents, setFilteredEvents] = useState<RunEvent[]>([])
  const [isFilterDialogOpen, setIsFilterDialogOpen] = useState(false)
  const [isExportDialogOpen, setIsExportDialogOpen] = useState(false)
  const [exportFormat, setExportFormat] = useState<'json' | 'xml'>('json')

  const toggleType = useCallback((type: RunEvent['eventType']) => {
    setFilterTypes((prev) =>
      prev.includes(type)
        ? prev.filter((t) => t !== type)
        : [...prev, type]
    )
  }, [])

  const exportText = useMemo(() => {
    if (exportFormat === 'xml') {
      return runEventsToXml(filteredEvents)
    }
    return JSON.stringify(filteredEvents, null, 2)
  }, [exportFormat, filteredEvents])

  const exportFilename = useMemo(() => (
    `run-${runId}-events.${exportFormat === 'xml' ? 'xml' : 'json'}`
  ), [runId, exportFormat])

  const handleCopyExport = useCallback(async () => {
    try {
      await copyToClipboard(exportText)
      toast({ title: `Copied ${exportFormat.toUpperCase()}`, variant: 'success' })
      setIsExportDialogOpen(false)
    } catch (err) {
      console.error('Failed to copy exported events:', err)
      toast({ title: 'Failed to copy export', variant: 'destructive' })
    }
  }, [exportFormat, exportText])

  const handleDownloadExport = useCallback(() => {
    const mimeType = exportFormat === 'xml' ? 'application/xml' : 'application/json'
    const blob = new Blob([exportText], { type: mimeType })
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = exportFilename
    document.body.appendChild(anchor)
    anchor.click()
    document.body.removeChild(anchor)
    URL.revokeObjectURL(url)
    setIsExportDialogOpen(false)
  }, [exportFilename, exportFormat, exportText])

  return (
    <div className={cn('flex flex-col gap-3', className)}>
      <div className="flex items-center gap-2">
        <div className="relative flex-1">
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

        <button
          type="button"
          onClick={() => setIsFilterDialogOpen(true)}
          className={cn(
            'relative p-2 rounded-md border border-border transition-colors',
            'text-muted-foreground hover:text-foreground hover:bg-muted/50'
          )}
          aria-label="Filter events"
          title="Filter events"
        >
          <Filter className="h-4 w-4" />
          {filterTypes.length > 0 && (
            <span className="absolute -top-1 -right-1 min-w-[16px] h-4 px-1 rounded-full bg-primary text-primary-foreground text-[10px] leading-4 text-center">
              {filterTypes.length}
            </span>
          )}
        </button>

        <button
          type="button"
          onClick={() => setIsExportDialogOpen(true)}
          className={cn(
            'p-2 rounded-md border border-border transition-colors',
            'text-muted-foreground hover:text-foreground hover:bg-muted/50'
          )}
          aria-label="Export events"
          title="Export events"
        >
          <FileOutput className="h-4 w-4" />
        </button>
      </div>

      <EventsDisplay
        runId={runId}
        live={live}
        filterTypes={filterTypes.length > 0 ? filterTypes : undefined}
        searchQuery={searchQuery || undefined}
        showHeader={false}
        onFilteredEventsChange={setFilteredEvents}
      />

      <Dialog
        isOpen={isFilterDialogOpen}
        onClose={() => setIsFilterDialogOpen(false)}
        title="Filter Events"
        maxWidth="max-w-md"
      >
        <div className="space-y-4">
          <p className="text-sm text-slate-300">Choose event types to include.</p>
          <div className="flex gap-1.5 flex-wrap">
            {EVENT_TYPES.map((type) => (
              <button
                key={type}
                type="button"
                onClick={() => toggleType(type)}
                className={cn(
                  'px-2.5 py-1.5 text-xs rounded border transition-colors',
                  filterTypes.includes(type)
                    ? 'bg-primary/10 text-primary border-primary/40'
                    : 'text-muted-foreground border-border hover:text-foreground hover:bg-muted/50'
                )}
              >
                {type.replace('_', ' ')}
              </button>
            ))}
          </div>
          <div className="flex items-center justify-between pt-2">
            <button
              type="button"
              onClick={() => setFilterTypes([])}
              className="px-3 py-1.5 text-xs rounded border border-border text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors"
            >
              Clear filters
            </button>
            <button
              type="button"
              onClick={() => setIsFilterDialogOpen(false)}
              className="px-3 py-1.5 text-xs rounded bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
            >
              Done
            </button>
          </div>
        </div>
      </Dialog>

      <Dialog
        isOpen={isExportDialogOpen}
        onClose={() => setIsExportDialogOpen(false)}
        title="Export Events"
        maxWidth="max-w-md"
      >
        <div className="space-y-4">
          <div>
            <p className="text-sm font-medium text-white mb-2">Format</p>
            <div className="inline-flex rounded-lg border border-border bg-background p-0.5">
              {(['json', 'xml'] as const).map((format) => (
                <button
                  key={format}
                  type="button"
                  onClick={() => setExportFormat(format)}
                  className={cn(
                    'px-3 py-1.5 text-xs font-medium rounded-md transition-colors uppercase',
                    exportFormat === format
                      ? 'bg-primary text-primary-foreground'
                      : 'text-muted-foreground hover:text-foreground'
                  )}
                >
                  {format}
                </button>
              ))}
            </div>
            <p className="text-xs text-muted-foreground mt-2">
              Exporting {filteredEvents.length} event{filteredEvents.length !== 1 ? 's' : ''}.
            </p>
          </div>

          <div className="flex items-center justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={() => void handleCopyExport()}
              className="px-3 py-1.5 text-xs rounded border border-border text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors"
            >
              Copy
            </button>
            <button
              type="button"
              onClick={handleDownloadExport}
              className="px-3 py-1.5 text-xs rounded bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
            >
              Download
            </button>
          </div>
        </div>
      </Dialog>
    </div>
  )
}
