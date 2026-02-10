/**
 * DeleteTeamDialog - Custom dialog for deleting a team with optional agent cleanup.
 *
 * Shows which agents are exclusive to this team (not in any other team)
 * and lets the user optionally select them for deletion too.
 */

import { useState, useEffect, useRef, useCallback } from 'react'
import { AlertTriangle, X, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { getTeamExclusiveMembers } from '@/services/teamService'
import type { ExclusiveMember } from '@/lib/schemas'

interface DeleteTeamDialogProps {
  isOpen: boolean
  teamId: string | null
  teamName: string
  onClose: () => void
  onConfirm: (agentIdsToDelete: string[]) => void
  isLoading?: boolean
}

export function DeleteTeamDialog({
  isOpen,
  teamId,
  teamName,
  onClose,
  onConfirm,
  isLoading = false,
}: DeleteTeamDialogProps) {
  const dialogRef = useRef<HTMLDivElement>(null)
  const confirmButtonRef = useRef<HTMLButtonElement>(null)

  const [exclusiveMembers, setExclusiveMembers] = useState<ExclusiveMember[]>([])
  const [selectedAgentIds, setSelectedAgentIds] = useState<Set<string>>(new Set())
  const [isFetching, setIsFetching] = useState(false)

  // Fetch exclusive members when dialog opens
  useEffect(() => {
    if (!isOpen || !teamId) {
      setExclusiveMembers([])
      setSelectedAgentIds(new Set())
      return
    }

    let cancelled = false
    setIsFetching(true)

    void getTeamExclusiveMembers(teamId).then((members) => {
      if (!cancelled) {
        setExclusiveMembers(members)
        setSelectedAgentIds(new Set(members.map((m) => m.agentId)))
        setIsFetching(false)
      }
    })

    return () => {
      cancelled = true
    }
  }, [isOpen, teamId])

  // Handle escape key
  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !isLoading) {
        onClose()
      }
    },
    [onClose, isLoading]
  )

  // Handle click outside
  const handleClickOutside = useCallback(
    (event: MouseEvent) => {
      if (
        dialogRef.current &&
        !dialogRef.current.contains(event.target as Node) &&
        !isLoading
      ) {
        onClose()
      }
    },
    [onClose, isLoading]
  )

  // Set up event listeners
  useEffect(() => {
    if (isOpen) {
      document.addEventListener('keydown', handleKeyDown)
      document.addEventListener('mousedown', handleClickOutside)
      setTimeout(() => confirmButtonRef.current?.focus(), 0)
      document.body.style.overflow = 'hidden'
    }

    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      document.removeEventListener('mousedown', handleClickOutside)
      document.body.style.overflow = ''
    }
  }, [isOpen, handleKeyDown, handleClickOutside])

  const toggleAgent = useCallback((agentId: string) => {
    setSelectedAgentIds((prev) => {
      const next = new Set(prev)
      if (next.has(agentId)) {
        next.delete(agentId)
      } else {
        next.add(agentId)
      }
      return next
    })
  }, [])

  const toggleAll = useCallback(() => {
    if (selectedAgentIds.size === exclusiveMembers.length) {
      setSelectedAgentIds(new Set())
    } else {
      setSelectedAgentIds(new Set(exclusiveMembers.map((m) => m.agentId)))
    }
  }, [selectedAgentIds.size, exclusiveMembers])

  const handleConfirm = useCallback(() => {
    onConfirm(Array.from(selectedAgentIds))
  }, [onConfirm, selectedAgentIds])

  if (!isOpen) return null

  const agentCount = selectedAgentIds.size
  const confirmLabel = isLoading
    ? 'Deleting...'
    : agentCount > 0
      ? `Delete team & ${agentCount} agent${agentCount !== 1 ? 's' : ''}`
      : 'Delete team'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" />

      {/* Dialog */}
      <div
        ref={dialogRef}
        className={cn(
          'relative w-full max-w-md mx-4 p-6',
          'bg-slate-900 border border-white/10 rounded-xl shadow-2xl',
          'animate-in fade-in-0 zoom-in-95 duration-150'
        )}
        role="dialog"
        aria-modal="true"
        aria-labelledby="delete-team-title"
        aria-describedby="delete-team-description"
      >
        {/* Close button */}
        <button
          type="button"
          onClick={onClose}
          disabled={isLoading}
          className={cn(
            'absolute top-4 right-4 p-1 rounded',
            'text-slate-400 hover:text-white hover:bg-white/10 transition-colors',
            isLoading && 'opacity-50 cursor-not-allowed'
          )}
          aria-label="Close dialog"
        >
          <X className="h-5 w-5" />
        </button>

        {/* Icon */}
        <div className="w-12 h-12 mx-auto mb-4 rounded-full flex items-center justify-center bg-red-500/20">
          <AlertTriangle className="h-6 w-6 text-red-400" />
        </div>

        {/* Title */}
        <h2
          id="delete-team-title"
          className="text-lg font-semibold text-white text-center mb-2"
        >
          Delete Team
        </h2>

        {/* Message */}
        <p
          id="delete-team-description"
          className="text-sm text-slate-400 text-center mb-4"
        >
          Are you sure you want to delete &ldquo;{teamName}&rdquo;? This cannot be undone.
        </p>

        {/* Exclusive members section */}
        {isFetching ? (
          <div className="flex items-center justify-center py-3 mb-4">
            <Loader2 className="h-4 w-4 animate-spin text-slate-400 mr-2" />
            <span className="text-sm text-slate-400">Checking for exclusive agents...</span>
          </div>
        ) : exclusiveMembers.length > 0 ? (
          <div className="mb-4">
            <div className="flex items-center justify-between mb-2">
              <p className="text-sm text-slate-300">
                These agents only belong to this team:
              </p>
              <button
                type="button"
                onClick={toggleAll}
                disabled={isLoading}
                className="text-xs text-primary hover:text-primary/80 transition-colors"
              >
                {selectedAgentIds.size === exclusiveMembers.length ? 'Deselect all' : 'Select all'}
              </button>
            </div>
            <div className="max-h-40 overflow-y-auto rounded-lg border border-white/10 bg-slate-800/50">
              {exclusiveMembers.map((member) => (
                <label
                  key={member.agentId}
                  className={cn(
                    'flex items-center gap-3 px-3 py-2 cursor-pointer',
                    'hover:bg-white/5 transition-colors',
                    isLoading && 'opacity-50 cursor-not-allowed'
                  )}
                >
                  <input
                    type="checkbox"
                    checked={selectedAgentIds.has(member.agentId)}
                    onChange={() => toggleAgent(member.agentId)}
                    disabled={isLoading}
                    className="rounded border-slate-600 bg-slate-700 text-primary focus:ring-primary/50"
                  />
                  <div className="min-w-0 flex-1">
                    <span className="text-sm text-white truncate block">
                      {member.displayName}
                    </span>
                    <span className="text-xs text-slate-500 truncate block">
                      {member.agentId}
                    </span>
                  </div>
                </label>
              ))}
            </div>
          </div>
        ) : null}

        {/* Actions */}
        <div className="flex gap-3">
          <button
            type="button"
            onClick={onClose}
            disabled={isLoading}
            className={cn(
              'flex-1 px-4 py-2 text-sm font-medium rounded-lg',
              'bg-slate-800 text-slate-300 hover:bg-slate-700 hover:text-white',
              'border border-white/10 transition-colors',
              isLoading && 'opacity-50 cursor-not-allowed'
            )}
          >
            Cancel
          </button>
          <button
            ref={confirmButtonRef}
            type="button"
            onClick={handleConfirm}
            disabled={isLoading || isFetching}
            className={cn(
              'flex-1 px-4 py-2 text-sm font-medium rounded-lg transition-colors',
              'bg-red-600 text-white hover:bg-red-500',
              (isLoading || isFetching) && 'opacity-50 cursor-not-allowed'
            )}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}
