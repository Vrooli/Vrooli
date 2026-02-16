/**
 * AISearchStatusPanel - Displays AI search service status and reindex controls.
 *
 * Self-contained component that manages its own state/polling lifecycle.
 * Used in both AISearchModal and SettingsDialog.
 */

import { useState, useEffect, useCallback, useRef } from 'react'
import { RefreshCw, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { AISearchStatus, AIReindexStatus } from '@/lib/schemas'
import { getAISearchStatus, reindexAISearch, getAISearchReindexStatus, cancelAISearchReindex } from '@/services/skillService'

interface AISearchStatusPanelProps {
  /** Controls fetch/polling lifecycle (e.g. modal isOpen or dialog isOpen) */
  active: boolean
  /** Vertical layout with larger text for settings dialog */
  compact?: boolean
}

export function AISearchStatusPanel({ active, compact }: AISearchStatusPanelProps) {
  const [status, setStatus] = useState<AISearchStatus | null>(null)
  const [statusError, setStatusError] = useState<string | null>(null)
  const [reindexStatus, setReindexStatus] = useState<AIReindexStatus | null>(null)
  const [reindexLoading, setReindexLoading] = useState(false)
  const wasReindexRunning = useRef(false)

  // Load AI status when active
  useEffect(() => {
    if (!active) return

    let alive = true

    const loadStatus = async () => {
      setStatusError(null)
      try {
        const nextStatus = await getAISearchStatus()
        if (alive) setStatus(nextStatus)
      } catch (err) {
        if (alive) {
          setStatus(null)
          setStatusError(err instanceof Error ? err.message : 'Failed to load AI search status')
        }
      }
    }

    const loadReindexStatus = async () => {
      try {
        const nextReindexStatus = await getAISearchReindexStatus()
        if (alive) setReindexStatus(nextReindexStatus)
      } catch {
        if (alive) setReindexStatus(null)
      }
    }

    void loadStatus()
    void loadReindexStatus()

    return () => { alive = false }
  }, [active])

  // Poll reindex status while running
  useEffect(() => {
    if (!active || !reindexStatus?.running) return

    const interval = setInterval(() => {
      getAISearchReindexStatus()
        .then((next) => setReindexStatus(next))
        .catch(() => {})
    }, 1500)

    return () => clearInterval(interval)
  }, [active, reindexStatus?.running])

  // Refresh AI status when reindex completes
  useEffect(() => {
    const running = Boolean(reindexStatus?.running)
    if (active && wasReindexRunning.current && !running) {
      getAISearchStatus()
        .then((nextStatus) => setStatus(nextStatus))
        .catch(() => {})
    }
    wasReindexRunning.current = running
  }, [active, reindexStatus?.running])

  const handleReindex = useCallback(async () => {
    if (reindexLoading) return
    setReindexLoading(true)
    setStatusError(null)
    try {
      const nextStatus = await reindexAISearch()
      setReindexStatus(nextStatus)
    } catch (err) {
      setStatusError(err instanceof Error ? err.message : 'Failed to start reindex')
    } finally {
      setReindexLoading(false)
    }
  }, [reindexLoading])

  const handleCancelReindex = useCallback(async () => {
    if (reindexLoading) return
    setReindexLoading(true)
    setStatusError(null)
    try {
      const nextStatus = await cancelAISearchReindex()
      setReindexStatus(nextStatus)
    } catch (err) {
      setStatusError(err instanceof Error ? err.message : 'Failed to cancel reindex')
    } finally {
      setReindexLoading(false)
    }
  }, [reindexLoading])

  const reindexCompleted = reindexStatus
    ? reindexStatus.indexed + reindexStatus.skipped + reindexStatus.errors
    : 0
  const reindexPercent =
    reindexStatus && reindexStatus.total > 0
      ? Math.min(100, Math.round((reindexCompleted / reindexStatus.total) * 100))
      : null

  if (compact) {
    return (
      <div className="rounded-lg border border-border bg-muted/40 p-3 space-y-2">
        <div className="flex flex-col gap-1.5 text-sm text-muted-foreground">
          <span
            className={cn(
              'inline-flex items-center gap-1 rounded-full px-2 py-0.5 w-fit',
              status
                ? status.available
                  ? 'bg-green-500/15 text-green-400'
                  : 'bg-amber-500/15 text-amber-400'
                : 'bg-muted text-muted-foreground'
            )}
          >
            {status ? (status.available ? 'AI ready' : 'AI unavailable') : 'AI status pending'}
          </span>
          <span>Ollama: {status ? (status.ollama ? 'online' : 'offline') : '—'}</span>
          <span>Qdrant: {status ? (status.qdrant ? 'online' : 'offline') : '—'}</span>
          <span>Indexed: {status ? status.indexedCount : '—'}</span>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => { void handleReindex() }}
            disabled={reindexLoading || reindexStatus?.running || !status?.available}
            className={cn(
              'inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium transition-colors',
              reindexLoading || reindexStatus?.running || !status?.available
                ? 'bg-muted text-muted-foreground cursor-not-allowed'
                : 'bg-primary/15 text-primary hover:bg-primary/25'
            )}
            title={status?.available ? 'Rebuild AI search index' : 'AI resources unavailable'}
          >
            <RefreshCw className={cn('h-3 w-3', reindexLoading ? 'animate-spin' : '')} />
            Reindex
          </button>
          {reindexStatus?.running && (
            <button
              type="button"
              onClick={() => { void handleCancelReindex() }}
              disabled={reindexLoading}
              className={cn(
                'inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium transition-colors',
                reindexLoading
                  ? 'bg-muted text-muted-foreground cursor-not-allowed'
                  : 'bg-destructive/15 text-destructive hover:bg-destructive/25'
              )}
            >
              <X className="h-3 w-3" />
              Cancel
            </button>
          )}
        </div>
        {statusError && <div className="text-xs text-destructive">{statusError}</div>}
        {status?.message && !status.available && (
          <div className="text-xs text-amber-500">{status.message}</div>
        )}
        {status?.available && status.indexedCount === 0 && (
          <div className="text-xs text-amber-500">
            No embeddings indexed yet. Run reindex to enable AI results.
          </div>
        )}
        {reindexStatus?.running && (
          <div className="text-xs text-muted-foreground">
            Reindexing {reindexCompleted}/{reindexStatus.total || '—'}
            {reindexPercent !== null ? ` (${reindexPercent}%)` : ''}...
          </div>
        )}
        {reindexStatus?.error && !reindexStatus.running && (
          <div className="text-xs text-destructive">{reindexStatus.error}</div>
        )}
        {reindexStatus?.message && !reindexStatus.running && (
          <div className="text-xs text-muted-foreground">{reindexStatus.message}</div>
        )}
      </div>
    )
  }

  // Default layout — horizontal, compact for modal
  return (
    <>
      <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
        <span
          className={cn(
            'inline-flex items-center gap-1 rounded-full px-2 py-0.5',
            status
              ? status.available
                ? 'bg-green-500/15 text-green-400'
                : 'bg-amber-500/15 text-amber-400'
              : 'bg-muted text-muted-foreground'
          )}
        >
          {status ? (status.available ? 'AI ready' : 'AI unavailable') : 'AI status pending'}
        </span>
        <span>Ollama: {status ? (status.ollama ? 'online' : 'offline') : '—'}</span>
        <span>Qdrant: {status ? (status.qdrant ? 'online' : 'offline') : '—'}</span>
        <span>Indexed: {status ? status.indexedCount : '—'}</span>
        <div className="ml-auto flex items-center gap-2">
          <button
            type="button"
            onClick={() => { void handleReindex() }}
            disabled={reindexLoading || reindexStatus?.running || !status?.available}
            className={cn(
              'inline-flex items-center gap-1 rounded-md px-2 py-1 text-[11px] font-medium transition-colors',
              reindexLoading || reindexStatus?.running || !status?.available
                ? 'bg-muted text-muted-foreground cursor-not-allowed'
                : 'bg-primary/15 text-primary hover:bg-primary/25'
            )}
            title={status?.available ? 'Rebuild AI search index' : 'AI resources unavailable'}
          >
            <RefreshCw className={cn('h-3 w-3', reindexLoading ? 'animate-spin' : '')} />
            Reindex
          </button>
          {reindexStatus?.running && (
            <button
              type="button"
              onClick={() => { void handleCancelReindex() }}
              disabled={reindexLoading}
              className={cn(
                'inline-flex items-center gap-1 rounded-md px-2 py-1 text-[11px] font-medium transition-colors',
                reindexLoading
                  ? 'bg-muted text-muted-foreground cursor-not-allowed'
                  : 'bg-destructive/15 text-destructive hover:bg-destructive/25'
              )}
            >
              <X className="h-3 w-3" />
              Cancel
            </button>
          )}
        </div>
      </div>
      {statusError && <div className="mt-1 text-xs text-destructive">{statusError}</div>}
      {status?.message && !status.available && (
        <div className="mt-1 text-xs text-amber-500">{status.message}</div>
      )}
      {status?.available && status.indexedCount === 0 && (
        <div className="mt-1 text-xs text-amber-500">
          No embeddings indexed yet. Run reindex to enable AI results.
        </div>
      )}
      {reindexStatus?.running && (
        <div className="mt-1 text-xs text-muted-foreground">
          Reindexing {reindexCompleted}/{reindexStatus.total || '—'}
          {reindexPercent !== null ? ` (${reindexPercent}%)` : ''}...
        </div>
      )}
      {reindexStatus?.error && !reindexStatus.running && (
        <div className="mt-1 text-xs text-destructive">{reindexStatus.error}</div>
      )}
      {reindexStatus?.message && !reindexStatus.running && (
        <div className="mt-1 text-xs text-muted-foreground">{reindexStatus.message}</div>
      )}
    </>
  )
}
