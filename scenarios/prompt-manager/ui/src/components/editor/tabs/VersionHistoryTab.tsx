/**
 * VersionHistoryTab - Timeline of skill versions with revert capability.
 *
 * Displays version history for a skill in newest-first order.
 * Each entry shows version number, timestamp, and name.
 * The current version is highlighted. Revert requires confirmation.
 */

import { useState } from 'react'
import { History, RotateCcw } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useVersionHistory, useRevertVersion } from '@/hooks/useVersionHistory'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { LoadingSpinner } from '@/components/ui/loading-spinner'

interface VersionHistoryTabProps {
  skillId: string
  className?: string
}

/**
 * Format a timestamp for display.
 */
function formatTimestamp(ts: string): string {
  try {
    const date = new Date(ts)
    return date.toLocaleDateString(undefined, {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  } catch {
    return ts
  }
}

export function VersionHistoryTab({ skillId, className }: VersionHistoryTabProps) {
  const { data, isLoading, isError, error } = useVersionHistory(skillId)
  const revertMutation = useRevertVersion()
  const [revertTarget, setRevertTarget] = useState<number | null>(null)

  const versions = data?.versions ?? []
  const currentVersion = data?.current ?? 0

  // Show newest first
  const sortedVersions = [...versions].reverse()

  const handleRevert = () => {
    if (revertTarget === null) return
    revertMutation.mutate(
      { skillId, version: revertTarget },
      { onSettled: () => setRevertTarget(null) }
    )
  }

  if (isLoading) {
    return (
      <div className={cn('flex items-center justify-center py-12', className)}>
        <LoadingSpinner size="md" />
      </div>
    )
  }

  if (isError) {
    return (
      <div className={cn('p-4 text-sm text-destructive', className)}>
        Failed to load version history: {error.message}
      </div>
    )
  }

  if (sortedVersions.length === 0) {
    return (
      <div className={cn('flex flex-col items-center justify-center py-12 text-muted-foreground', className)}>
        <History className="h-8 w-8 mb-2 opacity-50" />
        <p className="text-sm">No version history yet</p>
        <p className="text-xs mt-1">Versions are recorded when you save changes</p>
      </div>
    )
  }

  return (
    <div className={cn('flex flex-col gap-1 p-2', className)}>
      <h3 className="text-sm font-medium text-muted-foreground px-2 py-1">
        Version History ({sortedVersions.length})
      </h3>

      <div className="flex flex-col gap-0.5">
        {sortedVersions.map((v) => {
          const isCurrent = v.version === currentVersion
          return (
            <div
              key={v.version}
              className={cn(
                'flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors',
                isCurrent
                  ? 'bg-primary/10 border border-primary/20'
                  : 'hover:bg-muted/50'
              )}
            >
              {/* Version indicator dot */}
              <div className="flex flex-col items-center flex-shrink-0">
                <div
                  className={cn(
                    'w-2.5 h-2.5 rounded-full',
                    isCurrent ? 'bg-primary' : 'bg-muted-foreground/30'
                  )}
                />
              </div>

              {/* Version info */}
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="font-medium">
                    v{v.version}
                  </span>
                  {isCurrent && (
                    <span className="text-xs px-1.5 py-0.5 rounded bg-primary/20 text-primary font-medium">
                      Current
                    </span>
                  )}
                </div>
                <div className="text-xs text-muted-foreground mt-0.5 truncate">
                  {v.name && <span className="mr-2">{v.name}</span>}
                  {formatTimestamp(v.updatedAt)}
                </div>
                {v.createdBy && (
                  <div className="text-xs text-muted-foreground/70 truncate">
                    by {v.createdBy}
                  </div>
                )}
              </div>

              {/* Revert button - not shown for current */}
              {!isCurrent && (
                <button
                  type="button"
                  onClick={() => setRevertTarget(v.version)}
                  disabled={revertMutation.isPending}
                  className={cn(
                    'flex items-center gap-1 px-2 py-1 text-xs rounded-md',
                    'text-muted-foreground hover:text-foreground hover:bg-muted',
                    'transition-colors flex-shrink-0',
                    revertMutation.isPending && 'opacity-50 cursor-not-allowed'
                  )}
                  title={`Revert to version ${v.version}`}
                >
                  <RotateCcw className="h-3 w-3" />
                  Revert
                </button>
              )}
            </div>
          )
        })}
      </div>

      {/* Revert confirmation dialog */}
      <ConfirmDialog
        isOpen={revertTarget !== null}
        onClose={() => setRevertTarget(null)}
        onConfirm={handleRevert}
        title="Revert to previous version?"
        message={`This will restore the skill content to version ${revertTarget}. The current content will be saved as a new version before reverting.`}
        confirmLabel="Revert"
        variant="warning"
        isLoading={revertMutation.isPending}
      />
    </div>
  )
}
